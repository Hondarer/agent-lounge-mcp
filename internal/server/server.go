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
	// UserID is the user every connection acts as, from store.EnsureLocalUser.
	UserID int64
}

// Server is the MCP server of agent-lounge-mcp with its tools.
type Server struct {
	MCP *mcp.Server
	d   *tools.Deps
}

// New creates the MCP server with all tools registered.
func New(cfg config.Config, sess Session) *Server {
	d := &tools.Deps{Cfg: cfg, Store: sess.Store, UserID: sess.UserID}
	impl := &mcp.Implementation{Name: "agent-lounge-mcp", Title: "agent-lounge-mcp", Version: version.Version}
	registry := tools.Prepare(d)
	// With no Capabilities option the SDK declares the logging capability, which clients need
	// to set a log level and so to receive the shutdown notification.
	s := mcp.NewServer(impl, &mcp.ServerOptions{Instructions: tools.Instructions(d)})
	registry.Register(s)
	return &Server{MCP: s, d: d}
}

// BeginShutdown tells agents that the server is stopping. The HTTP server must keep accepting
// requests afterwards (see WaitForSessions), so that agents get this message rather than a
// connection error and /healthz reports "stopping". From now on every tool call fails
// with tools.ShutdownMessage, chat_inbox calls that wait for messages return it at once, and
// every connected MCP session is sent the same text as a log notification of level warning.
//
// The SDK sends log notifications only to clients that set a log level (logging/setLevel);
// other clients learn of the stop only from the error of their next or waiting tool call.
// ctx bounds the sending.
func (s *Server) BeginShutdown(ctx context.Context) {
	s.d.BeginShutdown()
	for ss := range s.MCP.Sessions() {
		// An error means the client is already gone; the others are still notified.
		_ = ss.Log(ctx, &mcp.LoggingMessageParams{Level: "warning", Logger: "agent-lounge-mcp", Data: tools.ShutdownMessage})
	}
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
func Handler(s *Server, sess Session) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(MCPPath, mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.MCP }, nil))
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
