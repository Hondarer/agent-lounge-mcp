package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

// waitInbox is the result of chat_inbox with the waiting field.
type waitInbox struct {
	inbox
	WaitedSeconds *int `json:"waited_seconds"`
}

// callResult is the outcome of a tool call made in the background.
type callResult struct {
	res *mcp.CallToolResult
	err error
}

// callAsync calls a tool in the background.
func callAsync(ctx context.Context, cs *mcp.ClientSession, name string, args map[string]any) <-chan callResult {
	ch := make(chan callResult, 1)
	go func() {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		ch <- callResult{res, err}
	}()
	return ch
}

// receive waits for a background call, failing the test after 5 seconds.
func receive(t *testing.T, ch <-chan callResult) *mcp.CallToolResult {
	t.Helper()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatal(r.err)
		}
		return r.res
	case <-time.After(5 * time.Second):
		t.Fatal("the call did not return")
		return nil
	}
}

// decode decodes the structured output of a successful result.
func decode[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	if res.IsError {
		t.Fatalf("call failed: %s", testutil.Text(res))
	}
	var out T
	b, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// waitServer creates a server whose chat_inbox waits at most waitMax and joins clients under
// the given names.
func waitServer(t *testing.T, waitMax time.Duration, names ...string) (*Server, []*mcp.ClientSession) {
	t.Helper()
	cfg := config.Default()
	cfg.InboxWaitMax = waitMax
	s, _ := newServerWith(t, cfg)
	var cs []*mcp.ClientSession
	for _, name := range names {
		c := testutil.Connect(t, s.MCP, nil)
		testutil.Call[joined](t, c, "lounge_join", map[string]any{"name": name})
		cs = append(cs, c)
	}
	return s, cs
}

func waitingCalls(t *testing.T, s *Server, n int) {
	t.Helper()
	testutil.WaitFor(t, func() bool { return s.d.WaitingCalls() == n }, "the number of waiting chat_inbox calls did not become as expected")
}

func TestInboxWaitReceivesMessage(t *testing.T) {
	s, cs := waitServer(t, 10*time.Second, "alice", "bob")
	alice, bob := cs[0], cs[1]

	for _, args := range []map[string]any{{"to": "bob", "body": "direct"}, {"room": "lounge", "body": "to the room"}} {
		ch := callAsync(context.Background(), bob, "chat_inbox", map[string]any{"wait_seconds": 10})
		waitingCalls(t, s, 1)
		testutil.Call[sent](t, alice, "chat_send", args)
		out := decode[waitInbox](t, receive(t, ch))
		if len(out.Messages) != 1 || out.Messages[0].Body != args["body"] || out.WaitedSeconds == nil || *out.WaitedSeconds >= 10 {
			t.Errorf("%v: inbox = %+v", args, out)
		}
		if unreadOf(t, out.Unread) != 0 {
			t.Errorf("unread = %d", *out.Unread)
		}
		waitingCalls(t, s, 0)
	}
}

func TestInboxWaitReturnsUnreadAtOnce(t *testing.T) {
	_, cs := waitServer(t, 10*time.Second, "alice", "bob")
	testutil.Call[sent](t, cs[0], "chat_send", map[string]any{"to": "bob", "body": "already here"})
	out := testutil.Call[waitInbox](t, cs[1], "chat_inbox", map[string]any{"wait_seconds": 10})
	if len(out.Messages) != 1 || out.WaitedSeconds != nil {
		t.Errorf("inbox = %+v, want the message without waited_seconds", out)
	}
}

// A wait longer than the limit is treated as the limit; when it runs out, the result is an
// empty list with waited_seconds equal to the wait.
func TestInboxWaitTimesOutAtLimit(t *testing.T) {
	_, cs := waitServer(t, 200*time.Millisecond, "bob")
	start := time.Now()
	out := testutil.Call[waitInbox](t, cs[0], "chat_inbox", map[string]any{"wait_seconds": 30})
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond || elapsed > 3*time.Second {
		t.Errorf("waited %s, want the 200ms limit", elapsed)
	}
	if len(out.Messages) != 0 || out.WaitedSeconds == nil || *out.WaitedSeconds != 1 {
		t.Errorf("inbox = %+v, want an empty list with waited_seconds 1", out)
	}
}

func TestInboxWaitDisabled(t *testing.T) {
	_, cs := waitServer(t, 0, "bob")
	out := testutil.Call[waitInbox](t, cs[0], "chat_inbox", map[string]any{"wait_seconds": 30})
	if len(out.Messages) != 0 || out.WaitedSeconds != nil {
		t.Errorf("inbox = %+v, want an empty list at once", out)
	}
	if msg := testutil.CallErr(t, cs[0], "chat_inbox", map[string]any{"wait_seconds": -1}); !strings.Contains(msg, "wait_seconds") {
		t.Errorf("negative wait: %s", msg)
	}
}

func TestInboxWaitWithoutWaitSecondsReturnsAtOnce(t *testing.T) {
	_, cs := waitServer(t, 10*time.Second, "bob")
	out := testutil.Call[waitInbox](t, cs[0], "chat_inbox", nil)
	if len(out.Messages) != 0 || out.WaitedSeconds != nil {
		t.Errorf("inbox = %+v", out)
	}
}

// A cancelled wait stops without changing any message.
func TestInboxWaitCancelled(t *testing.T) {
	s, cs := waitServer(t, 10*time.Second, "alice", "bob")
	alice, bob := cs[0], cs[1]
	ctx, cancel := context.WithCancel(context.Background())
	ch := callAsync(ctx, bob, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	cancel()
	select {
	case r := <-ch:
		if r.err == nil {
			t.Errorf("cancelled call returned %v", testutil.Text(r.res))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the cancelled call did not return")
	}
	waitingCalls(t, s, 0)

	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "after the cancel"})
	out := testutil.Call[inbox](t, bob, "chat_inbox", nil)
	if len(out.Messages) != 1 || out.Messages[0].Body != "after the cancel" {
		t.Errorf("inbox = %+v", out)
	}
}

// When another connection takes the name over, the waiting call returns the takeover error.
func TestInboxWaitTakenOver(t *testing.T) {
	s, cs := waitServer(t, 10*time.Second, "worker")
	ch := callAsync(context.Background(), cs[0], "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)

	second := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, second, "lounge_join", map[string]any{"name": "worker"})
	res := receive(t, ch)
	if !res.IsError || !strings.Contains(testutil.Text(res), "took the session over") {
		t.Errorf("waiting call after takeover: %s", testutil.Text(res))
	}
	waitingCalls(t, s, 0)
}

// Several calls of one session wait at once: all are woken, one receives the message, and the
// others wait on.
func TestInboxConcurrentWaiters(t *testing.T) {
	s, cs := waitServer(t, time.Second, "alice", "bob")
	alice, bob := cs[0], cs[1]
	const n = 3
	var chs []<-chan callResult
	for range n {
		chs = append(chs, callAsync(context.Background(), bob, "chat_inbox", map[string]any{"wait_seconds": 1}))
	}
	waitingCalls(t, s, n)
	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "once"})

	received, timedOut := 0, 0
	for _, ch := range chs {
		out := decode[waitInbox](t, receive(t, ch))
		received += len(out.Messages)
		if len(out.Messages) == 0 && out.WaitedSeconds != nil && *out.WaitedSeconds == 1 {
			timedOut++
		}
	}
	if received != 1 || timedOut != n-1 {
		t.Errorf("received %d messages, %d calls timed out; want 1 and %d", received, timedOut, n-1)
	}
}
