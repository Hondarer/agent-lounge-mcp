package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
	"github.com/hondarer/agent-lounge-mcp/internal/version"
)

// newServer creates a server backed by a database in a temporary directory.
func newServer(t *testing.T) (*mcp.Server, Session) {
	t.Helper()
	cfg := config.Default()
	cfg.DBPath = filepath.Join(t.TempDir(), "lounge.db")
	st, err := store.Open(context.Background(), cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	sess := Session{Store: st}
	return New(cfg, sess), sess
}

// connect connects a server and a client over in-memory transports.
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()
	s, _ := newServer(t)
	return testutil.Connect(t, s, nil)
}

// serveHTTP serves a server on a loopback port and returns its base URL.
func serveHTTP(t *testing.T) string {
	t.Helper()
	s, sess := newServer(t)
	hs := httptest.NewServer(Handler(s, sess))
	t.Cleanup(hs.Close)
	return hs.URL
}

type info struct {
	Version       string `json:"version"`
	SchemaVersion int    `json:"schema_version"`
}

func TestListTools(t *testing.T) {
	cs := connect(t)
	names := testutil.ToolSet(t, cs)
	for _, want := range []string{"lounge_info"} {
		if !names[want] {
			t.Errorf("tool %q is not registered", want)
		}
	}
}

func TestInstructions(t *testing.T) {
	cs := connect(t)
	if got := cs.InitializeResult().Instructions; got == "" {
		t.Error("server sent no instructions")
	}
}

func TestLoungeInfo(t *testing.T) {
	cs := connect(t)
	out := testutil.Call[info](t, cs, "lounge_info", nil)
	if out.Version != version.Version || out.SchemaVersion < 1 {
		t.Fatalf("got %+v", out)
	}
}

func TestStreamableHTTP(t *testing.T) {
	base := serveHTTP(t)

	// Two sessions share the one server and database.
	for range 2 {
		cs := testutil.ConnectHTTP(t, base+MCPPath)
		out := testutil.Call[info](t, cs, "lounge_info", nil)
		if out.Version != version.Version || out.SchemaVersion < 1 {
			t.Fatalf("got %+v", out)
		}
	}
}

func TestHealth(t *testing.T) {
	base := serveHTTP(t)
	res, err := http.Get(base + HealthPath)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var body struct {
		State string `json:"state"`
		info
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.State != "ok" || body.SchemaVersion < 1 {
		t.Fatalf("got %+v", body)
	}
}

func TestCrossOriginRequestIsRejected(t *testing.T) {
	base := serveHTTP(t)
	req, err := http.NewRequest(http.MethodPost, base+MCPPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusForbidden)
	}
}
