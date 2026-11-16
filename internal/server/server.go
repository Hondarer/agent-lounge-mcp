// Package server assembles agent-lounge-mcp's MCP server and its HTTP endpoints.
package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/tools"
	"github.com/hondarer/agent-lounge-mcp/internal/version"
)

// HTTP paths served by Handler.
const (
	MCPPath    = "/mcp"     // MCP over Streamable HTTP
	HealthPath = "/healthz" // liveness check for scripts and reverse proxies
)

// Session holds what a server uses besides its configuration.
type Session struct {
	Store *store.Store
}

// New creates the MCP server with all tools registered.
func New(cfg config.Config, sess Session) *mcp.Server {
	d := &tools.Deps{Cfg: cfg, Store: sess.Store}
	impl := &mcp.Implementation{Name: "agent-lounge-mcp", Title: "agent-lounge-mcp", Version: version.Version}
	registry := tools.Prepare(d)
	s := mcp.NewServer(impl, &mcp.ServerOptions{Instructions: tools.Instructions(d)})
	registry.Register(s)
	return s
}

// Handler serves s over Streamable HTTP at MCPPath and a liveness check at HealthPath.
// All MCP sessions share the one server.
func Handler(s *mcp.Server, sess Session) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(MCPPath, mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, nil))
	mux.HandleFunc("GET "+HealthPath, func(w http.ResponseWriter, r *http.Request) {
		health(r.Context(), w, sess)
	})
	// Agents are not browsers: reject cross-origin browser requests, which could otherwise
	// reach a server on the loopback interface from any web page.
	return http.NewCrossOriginProtection().Handler(mux)
}

func health(ctx context.Context, w http.ResponseWriter, sess Session) {
	w.Header().Set("Content-Type", "application/json")
	v, err := sess.Store.SchemaVersion(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"state": "error", "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"state": "ok", "version": version.Version, "schema_version": v})
}
