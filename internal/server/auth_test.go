package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/auth"
	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

// tokenServer serves a server with cfg over HTTP and returns its base URL and the session.
func tokenServer(t *testing.T, cfg config.Config) (string, Session) {
	t.Helper()
	s, sess := newServerWith(t, cfg)
	hs := httptest.NewServer(Handler(s, sess))
	t.Cleanup(hs.Close)
	return hs.URL, sess
}

// newUser adds a user and returns its token.
func newUser(t *testing.T, sess Session, name string) string {
	t.Helper()
	token, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sess.Store.AddUser(context.Background(), name, auth.Hash(token), time.Time{}); err != nil {
		t.Fatal(err)
	}
	return token
}

// post sends a raw MCP initialize request and returns the response status, the
// WWW-Authenticate header and the body. header values are added to the request.
func post(t *testing.T, base string, header map[string]string) (int, string, string) {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"raw","version":"1"}}}`
	req, err := http.NewRequest(http.MethodPost, base+MCPPath, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range header {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header.Get("WWW-Authenticate"), string(b)
}

// Configurations that require a token, one per condition.
func tokenRequiredConfigs() map[string]config.Config {
	remote := config.Default()
	remote.Listen, remote.AllowRemote = "0.0.0.0:0", true
	proxied := config.Default()
	proxied.AllowedHosts = []string{"lounge.example.com"}
	required := config.Default()
	required.RequireToken = true
	return map[string]config.Config{"--listen non-loopback": remote, "--allowed-host": proxied, "--require-token": required}
}

// Every row of the per-request decision, with and without a required token.
func TestAuthenticationDecision(t *testing.T) {
	configs := tokenRequiredConfigs()
	configs["default (loopback, no token required)"] = config.Default()
	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			base, sess := tokenServer(t, cfg)
			token := newUser(t, sess, "alice")
			other, _ := auth.NewToken() // well formed, of no user

			// A valid token: its user.
			cs, err := testutil.ConnectToken(t, base+MCPPath, token)
			if err != nil {
				t.Fatalf("valid token: %v", err)
			}
			if j := testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "a"}); j.User != "alice" {
				t.Errorf("valid token: user %q", j.User)
			}

			// No token: local, or 401 with the challenge when a token is required.
			status, challenge, body := post(t, base, nil)
			if cfg.TokenRequired() {
				if status != http.StatusUnauthorized || challenge != "Bearer" || !strings.Contains(body, "requires a token") {
					t.Errorf("no token: %d %q %q", status, challenge, body)
				}
				if _, err := testutil.ConnectToken(t, base+MCPPath, ""); err == nil {
					t.Error("a client without a token connected")
				}
			} else {
				if status != http.StatusOK {
					t.Errorf("no token: %d %q", status, body)
				}
				cs := testutil.ConnectHTTP(t, base+MCPPath)
				if j := testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "a"}); j.User != "local" {
					t.Errorf("no token: user %q", j.User)
				}
			}

			// An invalid token, or another form: 401 always, never local.
			for _, h := range []string{"Bearer " + other, "Bearer not-a-token", "Basic YWxpY2U6eA==", "Bearer", token} {
				status, challenge, body := post(t, base, map[string]string{"Authorization": h})
				if status != http.StatusUnauthorized || !strings.HasPrefix(challenge, "Bearer") || !strings.Contains(body, "not valid") {
					t.Errorf("Authorization %.12q...: %d %q %q", h, status, challenge, body)
				}
				if strings.Contains(body, token) || strings.Contains(body, other) {
					t.Error("the response quotes the token")
				}
				// The body is one sentence.
				if strings.Count(strings.TrimSpace(body), ". ") != 0 {
					t.Errorf("body is more than one sentence: %q", body)
				}
			}
			if _, err := testutil.ConnectToken(t, base+MCPPath, other); err == nil {
				t.Error("a client with a wrong token connected")
			}

			// /healthz needs no token.
			res, err := http.Get(base + HealthPath)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != http.StatusOK {
				t.Errorf("/healthz: %d", res.StatusCode)
			}
		})
	}
}

// The "Bearer" literal header (no token) is malformed, and so is a token alone; both are in
// the loop above. This checks that the missing-token body is one sentence too.
func TestUnauthorizedBodies(t *testing.T) {
	for _, msg := range []string{msgTokenMissing, msgTokenInvalid} {
		if strings.Count(msg, ". ") != 0 || !strings.HasSuffix(msg, ".") {
			t.Errorf("not one sentence: %q", msg)
		}
	}
}

// A connection belongs to the user of its first request: a request of another user on it is
// rejected with 403, and nothing is done.
func TestConnectionIsBoundToItsUser(t *testing.T) {
	base, sess := tokenServer(t, config.Default())
	alice, bob := newUser(t, sess, "alice"), newUser(t, sess, "bob")
	cs, err := testutil.ConnectToken(t, base+MCPPath, alice)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "a"})
	for name, header := range map[string]map[string]string{
		"bob":      {"Authorization": "Bearer " + bob},
		"no token": {},
	} {
		header["Mcp-Session-Id"] = cs.ID()
		status, _, body := post(t, base, header)
		if status != http.StatusForbidden {
			t.Errorf("%s on alice's connection: %d %q", name, status, body)
		}
	}
	// The connection itself still works for alice.
	testutil.Call[who](t, cs, "lounge_who", nil)

	// And a connection without a token (local) does not accept alice.
	local := testutil.ConnectHTTP(t, base+MCPPath)
	testutil.Call[joined](t, local, "lounge_join", map[string]any{"name": "l"})
	if status, _, _ := post(t, base, map[string]string{"Authorization": "Bearer " + alice, "Mcp-Session-Id": local.ID()}); status != http.StatusForbidden {
		t.Errorf("alice on local's connection: %d", status)
	}
}

// Rotating the token or removing the user rejects the next request of a connection.
func TestRotateAndRemoveTakeEffect(t *testing.T) {
	base, sess := tokenServer(t, config.Default())
	ctx := context.Background()
	token := newUser(t, sess, "alice")
	cs, err := testutil.ConnectToken(t, base+MCPPath, token)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "a"})

	fresh, _ := auth.NewToken()
	if err := sess.Store.RotateToken(ctx, "alice", auth.Hash(fresh), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.ListTools(ctx, nil); err == nil || !strings.Contains(err.Error(), "Unauthorized") {
		t.Errorf("a call with the rotated token: %v", err)
	}
	if status, _, _ := post(t, base, map[string]string{"Authorization": "Bearer " + token}); status != http.StatusUnauthorized {
		t.Errorf("old token after rotate: %d", status)
	}
	cs2, err := testutil.ConnectToken(t, base+MCPPath, fresh)
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	// The user and its session stay: joining continues it.
	if j := testutil.Call[joined](t, cs2, "lounge_join", map[string]any{"name": "a"}); j.User != "alice" || !j.Resumed {
		t.Errorf("after rotate: %+v", j)
	}

	if _, err := sess.Store.RemoveUser(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := cs2.ListTools(ctx, nil); err == nil || !strings.Contains(err.Error(), "Unauthorized") {
		t.Errorf("a call of a removed user: %v", err)
	}
	if status, _, _ := post(t, base, map[string]string{"Authorization": "Bearer " + fresh}); status != http.StatusUnauthorized {
		t.Errorf("token of a removed user: %d", status)
	}
}

// The three Host check configurations; cross-origin browser requests are rejected in all.
func TestHostCheck(t *testing.T) {
	remote := config.Default()
	remote.Listen, remote.AllowRemote = "0.0.0.0:0", true
	proxied := config.Default()
	proxied.AllowedHosts = []string{"lounge.example.com", "[2001:db8::1]"}
	for _, tc := range []struct {
		name     string
		cfg      config.Config
		accepted []string
		rejected []string
	}{
		{"default", config.Default(), []string{"localhost:7788", "127.0.0.1:7788", "[::1]:7788", "LOCALHOST"},
			[]string{"lounge.example.com", "evil.example:7788", "192.0.2.1:7788"}},
		{"--allowed-host", proxied, []string{"localhost:7788", "lounge.example.com", "Lounge.Example.com:443", "[2001:db8::1]:7788"},
			[]string{"evil.example", "example.com"}},
		{"--allow-remote", remote, []string{"localhost:7788", "192.0.2.1:7788", "anything.example"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, sess := tokenServer(t, tc.cfg)
			token := newUser(t, sess, "alice")
			auth := map[string]string{"Authorization": "Bearer " + token}
			for _, host := range tc.accepted {
				auth["Host"] = host
				if status, _, body := post(t, base, auth); status != http.StatusOK {
					t.Errorf("Host %s: %d %q", host, status, body)
				}
			}
			for _, host := range tc.rejected {
				auth["Host"] = host
				if status, _, _ := post(t, base, auth); status != http.StatusForbidden {
					t.Errorf("Host %s: %d, want 403", host, status)
				}
			}
			// Cross-origin browser requests are rejected, with a valid token too.
			delete(auth, "Host")
			auth["Origin"] = "https://example.com"
			auth["Sec-Fetch-Site"] = "cross-site"
			if status, _, _ := post(t, base, auth); status != http.StatusForbidden {
				t.Errorf("cross-origin request: %d, want 403", status)
			}
		})
	}
}
