package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
	"github.com/hondarer/agent-lounge-mcp/internal/tools"
)

// After the stop begins, every tool fails with the shutdown message, joined or not.
func TestShutdownFailsToolCalls(t *testing.T) {
	s, cs := waitServer(t, 10*time.Second, "alice")
	fresh := testutil.Connect(t, s.MCP, nil)
	s.BeginShutdown()
	args := map[string]map[string]any{
		"lounge_join":   {"name": "alice"},
		"chat_send":     {"to": "bob", "body": "x"},
		"chat_inbox":    {"wait_seconds": 10},
		"chat_ack":      {"message_ids": []int64{1}},
		"chat_status":   {"message_id": 1},
		"chat_room":     {"action": "list"},
		"memory_get":    {"id": 1},
		"memory_pin":    {"id": 1},
		"memory_forget": {"id": 1},
	}
	for _, c := range []*mcp.ClientSession{cs[0], fresh} {
		for _, name := range toolNames {
			if msg := testutil.CallErr(t, c, name, args[name]); msg != tools.ShutdownMessage {
				t.Errorf("%s while stopping: %q", name, msg)
			}
		}
	}
}

func TestShutdownEndsWaitingInbox(t *testing.T) {
	s, cs := waitServer(t, 10*time.Second, "bob")
	ch := callAsync(context.Background(), cs[0], "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	s.BeginShutdown()
	res := receive(t, ch)
	if !res.IsError || testutil.Text(res) != tools.ShutdownMessage {
		t.Errorf("waiting call: %s", testutil.Text(res))
	}
	waitingCalls(t, s, 0)
}

// The server declares no logging capability: the stop reaches agents only as the error of
// their tool calls.
func TestNoLoggingCapability(t *testing.T) {
	s, _ := newServer(t)
	cs := testutil.Connect(t, s.MCP, nil)
	caps := cs.InitializeResult().Capabilities
	if caps.Logging != nil {
		t.Errorf("logging capability declared: %+v", caps.Logging)
	}
	if caps.Tools == nil {
		t.Error("tools capability missing")
	}
}

// /healthz keeps answering while the server stops, with the state "stopping".
func TestHealthWhileStopping(t *testing.T) {
	s, sess := newServer(t)
	hs := httptest.NewServer(Handler(s, sess))
	t.Cleanup(hs.Close)
	s.BeginShutdown()
	res, err := http.Get(hs.URL + HealthPath)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || body.State != "stopping" {
		t.Errorf("status %d, state %q", res.StatusCode, body.State)
	}
}

// WaitForSessions returns at once without sessions, and when the last session ends.
func TestWaitForSessions(t *testing.T) {
	s, _ := newServer(t)
	start := time.Now()
	s.WaitForSessions(context.Background())
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("without sessions it waited %s", elapsed)
	}

	cs := testutil.Connect(t, s.MCP, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start = time.Now()
	s.WaitForSessions(ctx)
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Errorf("with a session it returned after %s, before the deadline", elapsed)
	}

	done := make(chan struct{})
	go func() {
		s.WaitForSessions(context.Background())
		close(done)
	}()
	cs.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForSessions did not return after the session ended")
	}
}
