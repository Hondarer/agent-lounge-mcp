package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

// lines collects the connection diagnostics of a server.
type lines struct {
	mu  sync.Mutex
	all []string
}

func (l *lines) logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.all = append(l.all, fmt.Sprintf(format, args...))
}

func (l *lines) text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.all, "\n")
}

func logServer(t *testing.T) (*Server, *lines) {
	t.Helper()
	cfg := config.Default()
	cfg.LogConnections = true
	s, _ := newServerWith(t, cfg)
	l := &lines{}
	s.conns.logf = l.logf
	return s, l
}

// connectAs connects a client with the given name and version.
func connectAs(t *testing.T, s *Server, name, version string) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	if _, err := s.MCP.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: name, Version: version}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// Without --log-connections, the server writes no connection diagnostics.
func TestConnectionLogOff(t *testing.T) {
	s, _ := newServer(t)
	if s.conns != nil {
		t.Fatal("connection log without LogConnections")
	}
}

// Each connection is numbered; its client, requests, tool names, log level and the outcome of
// the shutdown notice are written, and nothing of the tool arguments.
func TestConnectionLog(t *testing.T) {
	s, l := logServer(t)
	a := connectAs(t, s, "agent-a", "1.2.3")
	b := connectAs(t, s, "agent-b", "4.5")
	c := connectAs(t, s, "agent-c", "6")
	if err := c.SetLoggingLevel(context.Background(), &mcp.SetLoggingLevelParams{Level: "error"}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetLoggingLevel(context.Background(), &mcp.SetLoggingLevelParams{Level: "info"}); err != nil {
		t.Fatal(err)
	}
	testutil.Call[joined](t, b, "lounge_join", map[string]any{"name": "secret-session-name"})
	testutil.CallErr(t, b, "chat_send", map[string]any{"to": "nobody", "body": "secret body"})
	if _, err := b.ListTools(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	s.BeginShutdown(context.Background())

	got := l.text()
	for _, want := range []string{
		`connection 1: initialized client="agent-a" version="1.2.3" protocol=`,
		`connection 2: initialized client="agent-b" version="4.5" protocol=`,
		"connection 1: log level info",
		"connection 2: request tools/call lounge_join",
		"connection 2: request tools/call chat_send",
		"connection 2: request tools/list",
		"connection 1: shutdown notice sent",
		"connection 2: shutdown notice not sent (the client set no log level)",
		"connection 3: shutdown notice not sent (the log level error is above warning)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, secret := range []string{"secret", "notifications/"} {
		if strings.Contains(got, secret) {
			t.Errorf("%q written:\n%s", secret, got)
		}
	}
}

// Client names and versions lose control characters and quotes and are cut to 64 characters.
func TestConnectionLogCleansClientInfo(t *testing.T) {
	s, l := logServer(t)
	connectAs(t, s, "evil\"\n"+strings.Repeat("x", 100), "1\r\x1b[31m'`")
	got := l.text()
	want := `connection 1: initialized client="evil` + strings.Repeat("x", 60) + `" version="1[31m" protocol=`
	if !strings.HasPrefix(got, want) {
		t.Errorf("got %q, want prefix %q", got, want)
	}
	if strings.Count(got, "\n") != 0 {
		t.Errorf("more than one line: %q", got)
	}
}

func TestClean(t *testing.T) {
	for in, want := range map[string]string{
		"":                      "",
		"日本語のクライアント":            "日本語のクライアント",
		"a\tb\u0085c":           "abc",
		strings.Repeat("あ", 70): strings.Repeat("あ", 64),
		"\"q\"\u2028line\u2029": "qline",
		"a\u202eb\u2066c\ufeff": "abc",
	} {
		if got := clean(in); got != want {
			t.Errorf("clean(%q) = %q, want %q", in, got, want)
		}
	}
}

// A client of a protocol version with the initialize handshake, over HTTP, is written once,
// with the version the server agreed to.
func TestConnectionLogInitialize(t *testing.T) {
	s, l := logServer(t)
	hs := httptest.NewServer(Handler(s, Session{Store: s.d.Store, LocalUser: s.d.LocalUser}))
	t.Cleanup(hs.Close)
	post := func(id, body string) *http.Response {
		req, _ := http.NewRequest("POST", hs.URL+MCPPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if id != "" {
			req.Header.Set("Mcp-Session-Id", id)
			req.Header.Set("Mcp-Protocol-Version", "2025-06-18")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		return res
	}
	res := post("", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"old-cli","version":"9"}}}`)
	id := res.Header.Get("Mcp-Session-Id")
	if id == "" {
		t.Fatalf("no session: %s", res.Status)
	}
	post(id, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	post(id, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	got := l.text()
	want := "connection 1: initialized client=\"old-cli\" version=\"9\" protocol=2025-06-18\nconnection 1: request tools/list"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, id) {
		t.Errorf("session ID written: %s", got)
	}
}
