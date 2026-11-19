// Package server assembles agent-lounge-mcp's MCP server and its HTTP endpoints.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/tools"
	"github.com/hondarer/agent-lounge-mcp/internal/version"
)

// sessionPollInterval is how often WaitForSessions checks whether MCP sessions remain.
const sessionPollInterval = 50 * time.Millisecond

// HTTP paths served by Handler.
const (
	MCPPath    = "/mcp"     // MCP over Streamable HTTP
	HealthPath = "/healthz" // liveness check for scripts and reverse proxies
)

// Session holds what a server uses besides its configuration.
type Session struct {
	Store *store.Store
	// LocalUser is the user of requests without a token on a server that does not require
	// one (see store.EnsureLocalUser).
	LocalUser store.User
}

// Server is the MCP server of agent-lounge-mcp with its tools.
type Server struct {
	MCP *mcp.Server
	d   *tools.Deps
	cfg config.Config
	// conns writes the connection diagnostics; nil unless cfg.LogConnections.
	conns *connLog
}

// New creates the MCP server with all tools registered.
func New(cfg config.Config, sess Session) *Server {
	// The store enforces the limits that the tools state.
	sess.Store.SetLimits(cfg.Limits())
	sess.Store.SetRetention(cfg.Retention())
	d := &tools.Deps{Cfg: cfg, Store: sess.Store, LocalUser: sess.LocalUser}
	impl := &mcp.Implementation{Name: "agent-lounge-mcp", Title: "agent-lounge-mcp", Version: version.Version}
	registry := tools.Prepare(d)
	// An empty Capabilities keeps the SDK from declaring the logging capability, which it does
	// when the option is nil: the server sends no log notifications. The stop reaches agents as
	// the error of their tool calls (see BeginShutdown), which is all the CLIs pass on; none of
	// the CLIs checked sets a log level, and MCP deprecates logging from protocol 2026-07-28.
	// The SDK adds the tools capability from the registered tools.
	s := mcp.NewServer(impl, &mcp.ServerOptions{
		Instructions: tools.Instructions(d),
		Capabilities: &mcp.ServerCapabilities{},
	})
	registry.Register(s)
	srv := &Server{MCP: s, d: d, cfg: cfg}
	if cfg.LogConnections {
		srv.conns = newConnLog(s, nil)
		s.AddReceivingMiddleware(srv.conns.middleware())
	}
	return srv
}

// BeginShutdown tells agents that the server is stopping. The HTTP server must keep accepting
// requests afterwards (see WaitForSessions), so that agents get this message rather than a
// connection error and /healthz reports "stopping". From now on every tool call fails
// with tools.ShutdownMessage, and chat_inbox calls that wait for messages return it at once.
//
// The error of a tool call is the only notice: MCP has no notification that a client must
// pass on to its model, so an agent that is waiting for an instruction learns of the stop from
// its next tool call.
func (s *Server) BeginShutdown() {
	s.d.BeginShutdown()
}

// WaitForSessions is the grace period after BeginShutdown: it returns when no MCP session
// remains connected or when ctx is done, whichever comes first. During it, tool calls answer
// with the shutdown error. A server without connected agents returns at once.
func (s *Server) WaitForSessions(ctx context.Context) {
	ticker := time.NewTicker(sessionPollInterval)
	defer ticker.Stop()
	for {
		empty := true
		for range s.MCP.Sessions() {
			empty = false
			break
		}
		if empty {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Handler serves s over Streamable HTTP at MCPPath and a liveness check at HealthPath.
// All MCP sessions share the one server.
//
// A request to MCPPath passes, in order: the cross-origin check, the Host check (checkHost),
// authentication (authenticate), and the binding of the MCP connection to its user (bindUser),
// before it reaches the SDK. HealthPath needs no token: it reports no user data.
func Handler(s *Server, sess Session) http.Handler {
	mux := http.NewServeMux()
	// The SDK's own Host check (DNS rebinding protection) accepts only loopback names on a
	// loopback connection, which would reject a reverse proxy's Host; checkHost replaces it.
	//
	// SessionTimeout closes connections that go without a request for cfg.ConnectionIdle: a
	// client that stops using a connection without closing it (one left by a CLI that was
	// killed, or by a CLI that opens a new connection) would otherwise stay, and keep its
	// session shown as connected in lounge_who. Closing the connection ends the join (see
	// tools/presence.go): the next tool call on a new connection asks the agent to call
	// lounge_join again, and unread messages wait in the session. A request to a closed
	// connection gets HTTP 404, on which clients open a new connection.
	//
	// The handler keeps a state for each connection, which the join is bound to. On such a
	// handler the SDK (v1.8.0) serves the protocol versions up to 2025-11-25: a request of
	// version 2026-07-28 gets the error -32022 with the supported versions, except
	// server/discover, which answers with them; clients then connect again with initialize.
	// Version 2026-07-28 is served only without a state for each connection, which the join
	// cannot be bound to.
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.MCP },
		&mcp.StreamableHTTPOptions{DisableLocalhostProtection: true, SessionTimeout: s.cfg.ConnectionIdle})
	mux.Handle(MCPPath, checkHost(s.cfg, authenticate(s.cfg, sess, bindUser(mcpHandler))))
	mux.HandleFunc("GET "+HealthPath, func(w http.ResponseWriter, r *http.Request) {
		health(r.Context(), w, sess, s.d.Stopping(), s.d.WaitingCalls())
	})
	// Agents are not browsers: reject cross-origin browser requests, which could otherwise
	// reach a server on the loopback interface from any web page.
	return http.NewCrossOriginProtection().Handler(mux)
}

// health answers the liveness check. While the server stops, the state is "stopping". waiting
// is the number of chat_inbox calls waiting for messages now; scripts use it to know that a
// call has started to wait before they send.
func health(ctx context.Context, w http.ResponseWriter, sess Session, stopping bool, waiting int) {
	w.Header().Set("Content-Type", "application/json")
	v, err := sess.Store.SchemaVersion(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"state": "error", "error": err.Error()})
		return
	}
	state := "ok"
	if stopping {
		state = "stopping"
	}
	json.NewEncoder(w).Encode(map[string]any{"state": state, "version": version.Version, "schema_version": v, "waiting": waiting})
}
