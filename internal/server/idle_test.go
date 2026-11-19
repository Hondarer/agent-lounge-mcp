package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

// idleTime is the --connection-idle of the tests: short, yet long enough for a few requests.
const idleTime = 500 * time.Millisecond

func idleServer(t *testing.T, idle time.Duration) string {
	t.Helper()
	cfg := config.Default()
	cfg.ConnectionIdle = idle
	s, sess := newServerWith(t, cfg)
	hs := httptest.NewServer(Handler(s, sess))
	t.Cleanup(hs.Close)
	return hs.URL
}

// initialize opens a connection with raw requests and returns its MCP session ID.
func initialize(t *testing.T, base string) string {
	t.Helper()
	res, out := rpc(t, base, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25",`+
		`"capabilities":{},"clientInfo":{"name":"raw","version":"1"}}}`, nil)
	id := res.Header.Get("Mcp-Session-Id")
	if id == "" || out.Error != nil {
		t.Fatalf("initialize: status %d, error %+v", res.StatusCode, out.Error)
	}
	rpc(t, base, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, sessionHeader(id))
	return id
}

func sessionHeader(id string) map[string]string {
	return map[string]string{"Mcp-Session-Id": id, "Mcp-Protocol-Version": "2025-11-25"}
}

// A connection without requests for the idle time is closed: a request to it gets HTTP 404.
func TestIdleConnectionIsClosed(t *testing.T) {
	base := idleServer(t, idleTime)
	id := initialize(t, base)
	if res, _ := rpc(t, base, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, sessionHeader(id)); res.StatusCode != http.StatusOK {
		t.Fatalf("before the idle time: status %d", res.StatusCode)
	}
	time.Sleep(3 * idleTime)
	if res, _ := rpc(t, base, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`, sessionHeader(id)); res.StatusCode != http.StatusNotFound {
		t.Errorf("after the idle time: status %d, want 404", res.StatusCode)
	}
}

// Closing an idle connection ends its join: other sessions see it as not connected, and its
// messages wait in the session.
func TestIdleConnectionEndsJoin(t *testing.T) {
	base := idleServer(t, idleTime)
	id := initialize(t, base)
	res, out := rpc(t, base, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"lounge_join","arguments":{"name":"idle"}}}`,
		sessionHeader(id))
	if res.StatusCode != http.StatusOK || out.Error != nil {
		t.Fatalf("lounge_join: status %d, error %+v", res.StatusCode, out.Error)
	}
	bob := testutil.ConnectHTTP(t, base+MCPPath)
	testutil.Call[joined](t, bob, "lounge_join", map[string]any{"name": "bob"})
	testutil.Call[sent](t, bob, "chat_send", map[string]any{"to": "idle", "body": "while away"})
	connected := func() bool {
		for _, s := range testutil.Call[who](t, bob, "lounge_who", nil).Sessions {
			if s.Name == "idle" {
				return s.Connected
			}
		}
		t.Fatal("idle is not listed")
		return false
	}
	// bob's own requests keep bob's connection open.
	testutil.WaitFor(t, func() bool { return !connected() }, "the idle connection was not closed")

	again := testutil.ConnectHTTP(t, base+MCPPath)
	j := testutil.Call[joined](t, again, "lounge_join", map[string]any{"name": "idle"})
	if !j.Resumed || unreadOf(t, j.Unread) != 1 {
		t.Errorf("joined again: %+v", j)
	}
}

// A chat_inbox call that waits longer than the idle time keeps its connection open.
func TestWaitingInboxKeepsConnection(t *testing.T) {
	base := idleServer(t, idleTime)
	cs := testutil.ConnectHTTP(t, base+MCPPath)
	testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "waiter"})
	in := testutil.Call[inbox](t, cs, "chat_inbox", map[string]any{"wait_seconds": 2})
	if len(in.Messages) != 0 {
		t.Fatalf("inbox = %+v", in)
	}
	// Still joined on the same connection.
	testutil.Call[inbox](t, cs, "chat_inbox", nil)
}

// With --connection-idle 0, connections are not closed.
func TestIdleZeroKeepsConnection(t *testing.T) {
	base := idleServer(t, 0)
	cs := testutil.ConnectHTTP(t, base+MCPPath)
	testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "stays"})
	time.Sleep(3 * idleTime)
	testutil.Call[inbox](t, cs, "chat_inbox", nil)
	if err := cs.Ping(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}
