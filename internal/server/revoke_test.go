package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/auth"
	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

const revokedText = "your token is no longer valid"

// revokeServer serves a server whose waiting chat_inbox calls check their token every
// interval, with the user alice joined as session a over HTTP.
func revokeServer(t *testing.T, interval time.Duration) (s *Server, sess Session, base, token string, a *mcp.ClientSession) {
	t.Helper()
	cfg := config.Default()
	cfg.InboxWaitMax = 10 * time.Second
	s, sess = newServerWith(t, cfg)
	s.d.TokenCheckInterval = interval
	hs := httptest.NewServer(Handler(s, sess))
	t.Cleanup(hs.Close)
	token = newUser(t, sess, "alice")
	a, err := testutil.ConnectToken(t, hs.URL+MCPPath, token)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Call[joined](t, a, "lounge_join", map[string]any{"name": "a"})
	return s, sess, hs.URL, token, a
}

func rotate(t *testing.T, sess Session, name string) string {
	t.Helper()
	fresh, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Store.RotateToken(context.Background(), name, auth.Hash(fresh)); err != nil {
		t.Fatal(err)
	}
	return fresh
}

// A waiting chat_inbox ends with an error soon after its token is rotated.
func TestWaitingInboxEndsWhenTokenRotated(t *testing.T) {
	s, sess, _, _, a := revokeServer(t, 50*time.Millisecond)
	ch := callAsync(context.Background(), a, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	rotate(t, sess, "alice")
	res := receive(t, ch)
	if !res.IsError || !strings.Contains(testutil.Text(res), revokedText) {
		t.Errorf("waiting call: %s", testutil.Text(res))
	}
	waitingCalls(t, s, 0)
}

// A waiting chat_inbox ends in the same way when its user is removed.
func TestWaitingInboxEndsWhenUserRemoved(t *testing.T) {
	s, sess, _, _, a := revokeServer(t, 50*time.Millisecond)
	ch := callAsync(context.Background(), a, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	if _, err := sess.Store.RemoveUser(context.Background(), "alice"); err != nil {
		t.Fatal(err)
	}
	res := receive(t, ch)
	if !res.IsError || !strings.Contains(testutil.Text(res), revokedText) {
		t.Errorf("waiting call: %s", testutil.Text(res))
	}
}

// A message that arrives after the token was rotated is not handed to the waiting call: it
// stays unread for the session. The interval is long, so only the wake-up check can catch it.
func TestRotatedTokenReceivesNoMessage(t *testing.T) {
	s, sess, base, _, a := revokeServer(t, time.Hour)
	ch := callAsync(context.Background(), a, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	fresh := rotate(t, sess, "alice")
	b, err := testutil.ConnectToken(t, base+MCPPath, fresh)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Call[joined](t, b, "lounge_join", map[string]any{"name": "b"})
	testutil.Call[sent](t, b, "chat_send", map[string]any{"to": "a", "body": "after the rotation"})
	res := receive(t, ch)
	if !res.IsError || !strings.Contains(testutil.Text(res), revokedText) {
		t.Fatalf("waiting call: %s", testutil.Text(res))
	}
	// The message is still unread: the holder of the new token receives it.
	again, err := testutil.ConnectToken(t, base+MCPPath, fresh)
	if err != nil {
		t.Fatal(err)
	}
	if j := testutil.Call[joined](t, again, "lounge_join", map[string]any{"name": "a"}); !j.Resumed || unreadOf(t, j.Unread) != 1 {
		t.Errorf("joined again: %+v", j)
	}
	in := testutil.Call[inbox](t, again, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Body != "after the rotation" {
		t.Errorf("inbox: %+v", in)
	}
}

// With a valid token, and without a token (the user local), the checks do not disturb the wait.
func TestTokenChecksKeepValidWait(t *testing.T) {
	s, _, base, token, a := revokeServer(t, 20*time.Millisecond)
	b, err := testutil.ConnectToken(t, base+MCPPath, token)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Call[joined](t, b, "lounge_join", map[string]any{"name": "b"})
	ch := callAsync(context.Background(), a, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	time.Sleep(200 * time.Millisecond) // several checks pass
	testutil.Call[sent](t, b, "chat_send", map[string]any{"to": "a", "body": "hello"})
	if res := receive(t, ch); res.IsError || !strings.Contains(testutil.Text(res), "hello") {
		t.Errorf("valid token: %s", testutil.Text(res))
	}

	local := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, local, "lounge_join", map[string]any{"name": "l"})
	ch = callAsync(context.Background(), local, "chat_inbox", map[string]any{"wait_seconds": 1})
	if res := receive(t, ch); res.IsError {
		t.Errorf("local user: %s", testutil.Text(res))
	}
}
