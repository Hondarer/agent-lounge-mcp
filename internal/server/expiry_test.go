package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/auth"
	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

const expiredText = "your token has expired"

// testClock is a clock for the store that tests move forward.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// clockServer serves a server with cfg over HTTP whose store uses a test clock, and whose
// waiting chat_inbox calls check their token every interval.
func clockServer(t *testing.T, cfg config.Config, interval time.Duration) (*Server, Session, string, *testClock) {
	t.Helper()
	cfg.InboxWaitMax = 10 * time.Second
	s, sess := newServerWith(t, cfg)
	clock := &testClock{now: time.UnixMilli(1_800_000_000_000).UTC()}
	sess.Store.SetClock(clock.Now)
	s.d.TokenCheckInterval = interval
	hs := httptest.NewServer(Handler(s, sess))
	t.Cleanup(hs.Close)
	return s, sess, hs.URL, clock
}

// newExpiringUser adds a user whose token expires at expiresAt and returns the token.
func newExpiringUser(t *testing.T, sess Session, name string, expiresAt time.Time) string {
	t.Helper()
	token, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sess.Store.AddUser(context.Background(), name, auth.Hash(token), expiresAt); err != nil {
		t.Fatal(err)
	}
	return token
}

// An expired token gets 401 with its own body, in every configuration (never the user local);
// extending the expiry makes the same token work again from the next request.
func TestExpiredToken(t *testing.T) {
	configs := tokenRequiredConfigs()
	configs["default (loopback, no token required)"] = config.Default()
	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			_, sess, base, clock := clockServer(t, cfg, time.Hour)
			expiry := clock.Now().Add(time.Hour)
			token := newExpiringUser(t, sess, "alice", expiry)
			forever := newExpiringUser(t, sess, "bob", time.Time{})
			header := map[string]string{"Authorization": "Bearer " + token}

			// Before the expiry: accepted.
			if status, _, body := post(t, base, header); status != http.StatusOK {
				t.Fatalf("before the expiry: %d %q", status, body)
			}
			cs := joinAs(t, base, token, map[string]any{"name": "a"})

			// From the expiry time on: 401 with the expiry body, also on an open connection.
			clock.Add(time.Hour)
			status, challenge, body := post(t, base, header)
			if status != http.StatusUnauthorized || challenge != `Bearer error="invalid_token"` || body != msgTokenExpired+"\n" {
				t.Errorf("expired: %d %q %q", status, challenge, body)
			}
			if strings.Contains(body, token) {
				t.Error("the body quotes the token")
			}
			if _, err := cs.ListTools(context.Background(), nil); err == nil {
				t.Error("an open connection still works with the expired token")
			}
			if _, err := testutil.ConnectToken(t, base+MCPPath, token); err == nil {
				t.Error("a new connection with the expired token was accepted")
			}
			// A token without an expiry is not affected.
			if status, _, body := post(t, base, map[string]string{"Authorization": "Bearer " + forever}); status != http.StatusOK {
				t.Errorf("token without an expiry: %d %q", status, body)
			}

			// Extended: the same token works again from the next request.
			if err := sess.Store.SetTokenExpiry(context.Background(), "alice", clock.Now().Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
			if status, _, body := post(t, base, header); status != http.StatusOK {
				t.Errorf("after the extension: %d %q", status, body)
			}
			if j := testutil.Call[joined](t, joinAs(t, base, token, map[string]any{"name": "a2"}), "lounge_join", map[string]any{"name": "a"}); j.User != "alice" {
				t.Errorf("join after the extension: %+v", j)
			}
			// Without an expiry, it does not expire.
			if err := sess.Store.SetTokenExpiry(context.Background(), "alice", time.Time{}); err != nil {
				t.Fatal(err)
			}
			clock.Add(1000 * 24 * time.Hour)
			if status, _, body := post(t, base, header); status != http.StatusOK {
				t.Errorf("after --never: %d %q", status, body)
			}
		})
	}
}

// A waiting chat_inbox ends with the expiry error within its check interval after its token
// expires.
func TestWaitingInboxEndsWhenTokenExpires(t *testing.T) {
	s, sess, base, clock := clockServer(t, config.Default(), 50*time.Millisecond)
	token := newExpiringUser(t, sess, "alice", clock.Now().Add(time.Hour))
	a := joinAs(t, base, token, map[string]any{"name": "a"})
	ch := callAsync(context.Background(), a, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	time.Sleep(150 * time.Millisecond) // checks before the expiry keep the wait
	if s.d.WaitingCalls() != 1 {
		t.Fatal("the wait ended before the expiry")
	}
	clock.Add(time.Hour)
	res := receive(t, ch)
	if text := testutil.Text(res); !res.IsError || !strings.Contains(text, expiredText) || !strings.Contains(text, "user expiry") ||
		!strings.Contains(text, "user rotate") || strings.Contains(text, revokedText) {
		t.Errorf("waiting call: %s", text)
	}
	waitingCalls(t, s, 0)
}

// A message that arrives after the token expired is not handed to the waiting call: it stays
// unread for the session, and the holder of the token receives it once the expiry is extended.
// The interval is long, so only the wake-up check can catch it.
func TestExpiredTokenReceivesNoMessage(t *testing.T) {
	s, sess, base, clock := clockServer(t, config.Default(), time.Hour)
	token := newExpiringUser(t, sess, "alice", clock.Now().Add(time.Hour))
	origin := "https://github.com/owner/repo.git"
	a := joinAs(t, base, token, map[string]any{"name": "a", "origin": origin})
	l := testutil.ConnectHTTP(t, base+MCPPath) // local, in the same repository
	testutil.Call[joined](t, l, "lounge_join", map[string]any{"name": "l", "origin": origin})

	ch := callAsync(context.Background(), a, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	clock.Add(time.Hour)
	testutil.Call[sent](t, l, "chat_send", map[string]any{"to": "alice/a", "scope": "repository", "body": "after the expiry"})
	res := receive(t, ch)
	if !res.IsError || !strings.Contains(testutil.Text(res), expiredText) {
		t.Fatalf("waiting call: %s", testutil.Text(res))
	}

	if err := sess.Store.SetTokenExpiry(context.Background(), "alice", clock.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	again := joinAs(t, base, token, map[string]any{"name": "a", "origin": origin})
	in := testutil.Call[inbox](t, again, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Body != "after the expiry" {
		t.Errorf("inbox: %+v", in)
	}
}

// lastUsed returns the last use of the user as user list shows it (zero for none).
func lastUsed(t *testing.T, st *store.Store, name string) time.Time {
	t.Helper()
	users, err := st.ListUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range users {
		if u.Name == name {
			return u.LastUsedAt
		}
	}
	t.Fatalf("no user %s", name)
	return time.Time{}
}

// A request with a valid token records the last use of the token, also without joining, at
// most once a minute per user. Requests with an expired or invalid token, and requests without
// a token (the user local), record nothing.
func TestTokenLastUse(t *testing.T) {
	_, sess, base, clock := clockServer(t, config.Default(), time.Hour)
	start := clock.Now()
	token := newExpiringUser(t, sess, "alice", start.Add(10*time.Minute))
	other := newUser(t, sess, "bob")
	if !lastUsed(t, sess.Store, "alice").IsZero() {
		t.Fatal("a new user has a last use")
	}
	header := map[string]string{"Authorization": "Bearer " + token}

	// The first request records the time, without a session.
	if status, _, _ := post(t, base, header); status != http.StatusOK {
		t.Fatal(status)
	}
	if got := lastUsed(t, sess.Store, "alice"); !got.Equal(start) {
		t.Errorf("after the first request: %v, want %v", got, start)
	}
	// Within the minute: not written again.
	clock.Add(59 * time.Second)
	post(t, base, header)
	if got := lastUsed(t, sess.Store, "alice"); !got.Equal(start) {
		t.Errorf("within the minute: %v, want %v", got, start)
	}
	// A minute after the previous write: written.
	clock.Add(time.Second)
	post(t, base, header)
	if got, want := lastUsed(t, sess.Store, "alice"), start.Add(time.Minute); !got.Equal(want) {
		t.Errorf("after a minute: %v, want %v", got, want)
	}
	// Each user has its own interval.
	post(t, base, map[string]string{"Authorization": "Bearer " + other})
	if got, want := lastUsed(t, sess.Store, "bob"), start.Add(time.Minute); !got.Equal(want) {
		t.Errorf("bob: %v, want %v", got, want)
	}

	// Expired: not recorded.
	clock.Add(10 * time.Minute)
	if status, _, _ := post(t, base, header); status != http.StatusUnauthorized {
		t.Fatal(status)
	}
	if got, want := lastUsed(t, sess.Store, "alice"), start.Add(time.Minute); !got.Equal(want) {
		t.Errorf("after an expired request: %v, want %v", got, want)
	}
	// Invalid, and without a token: local keeps no last use without sessions.
	invalid, _ := auth.NewToken()
	post(t, base, map[string]string{"Authorization": "Bearer " + invalid})
	post(t, base, nil)
	if got := lastUsed(t, sess.Store, store.LocalUser); !got.IsZero() {
		t.Errorf("local: %v", got)
	}
}
