package server

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

type scopeOut struct {
	Scope string `json:"scope"`
	ID    string `json:"id"`
}

type scopedWho struct {
	Repository string     `json:"repository"`
	Scopes     []scopeOut `json:"scopes"`
	Sessions   []struct {
		Name       string     `json:"name"`
		User       string     `json:"user"`
		Address    string     `json:"address"`
		Repository string     `json:"repository"`
		Scopes     []scopeOut `json:"scopes"`
		You        bool       `json:"you"`
	} `json:"sessions"`
}

// joinAs connects with a token over HTTP and joins with args.
func joinAs(t *testing.T, base, token string, args map[string]any) *mcp.ClientSession {
	t.Helper()
	cs, err := testutil.ConnectToken(t, base+MCPPath, token)
	if err != nil {
		t.Fatal(err)
	}
	testutil.Call[joined](t, cs, "lounge_join", args)
	return cs
}

// Two users: who lists the union of the scopes, each scope combination reaches the right
// sessions, addresses carry the user when it differs, and replies use scope and reply_to.
func TestTwoUsers(t *testing.T) {
	base, sess := tokenServer(t, config.Default())
	aliceToken, bobToken := newUser(t, sess, "alice"), newUser(t, sess, "bob")
	repo := "github.com/owner/repo"
	a1 := joinAs(t, base, aliceToken, map[string]any{"name": "claude", "origin": "https://github.com/owner/repo.git"})
	a2 := joinAs(t, base, aliceToken, map[string]any{"name": "codex", "origin": "https://github.com/owner/other.git"})
	b1 := joinAs(t, base, bobToken, map[string]any{"name": "claude", "origin": "git@github.com:owner/repo.git"}) // same name as alice's
	b2 := joinAs(t, base, bobToken, map[string]any{"name": "grok", "origin": "https://github.com/owner/other2.git"})
	// local (no token) in the same repository: also another user.
	l1 := testutil.ConnectHTTP(t, base+MCPPath)
	testutil.Call[joined](t, l1, "lounge_join", map[string]any{"name": "agy", "origin": "https://github.com/owner/repo"})

	// lounge_who from alice's claude.
	w := testutil.Call[scopedWho](t, a1, "lounge_who", nil)
	if !slices.Equal(w.Scopes, []scopeOut{{"user", "alice"}, {"repository", repo}, {"shared", ""}}) {
		t.Errorf("top-level scopes = %+v", w.Scopes)
	}
	got := map[string][]scopeOut{}
	for _, s := range w.Sessions {
		got[s.Address] = s.Scopes
		if want := s.User + "/" + s.Name; s.User != "alice" && s.Address != want {
			t.Errorf("address of %+v, want %s", s, want)
		}
	}
	want := map[string][]scopeOut{
		"claude":     {{"user", "alice"}, {"repository", repo}, {"shared", ""}},
		"codex":      {{"user", "alice"}, {"shared", ""}},
		"bob/claude": {{"repository", repo}, {"shared", ""}},
		"bob/grok":   {{"shared", ""}},
		"local/agy":  {{"repository", repo}, {"shared", ""}},
	}
	if len(got) != len(want) {
		t.Errorf("who lists %v, want %v", got, want)
	}
	for addr, sc := range want {
		if !slices.Equal(got[addr], sc) {
			t.Errorf("%s: scopes %+v, want %+v", addr, got[addr], sc)
		}
	}
	// bob's session of another repository is visible through the shared scope only.
	if len(got["bob/grok"]) != 1 {
		t.Error("bob/grok is not listed with the shared scope only")
	}

	// user + room: alice's other session only.
	s := testutil.Call[scopedSent](t, a1, "chat_send", map[string]any{"room": "lounge", "body": "user room"})
	if !slices.Equal(s.Recipients, []string{"codex"}) || s.Scope != "user" || s.ScopeID != "alice" {
		t.Errorf("user room: %+v", s)
	}
	// user + to.
	s = testutil.Call[scopedSent](t, a1, "chat_send", map[string]any{"to": "codex", "body": "user direct"})
	if !slices.Equal(s.Recipients, []string{"codex"}) || s.ScopeID != "alice" {
		t.Errorf("user direct: %+v", s)
	}
	// own-user/name is the same as name.
	s = testutil.Call[scopedSent](t, a1, "chat_send", map[string]any{"to": "alice/codex", "body": "qualified"})
	if !slices.Equal(s.Recipients, []string{"codex"}) {
		t.Errorf("alice/codex: %+v", s)
	}
	// repository + room: across users.
	s = testutil.Call[scopedSent](t, a1, "chat_send", map[string]any{"room": "lounge", "scope": "repository", "body": "repo room"})
	if !slices.Equal(s.Recipients, []string{"local/agy", "bob/claude"}) && !slices.Equal(s.Recipients, []string{"bob/claude", "local/agy"}) {
		t.Errorf("repository room: %+v", s)
	}
	if s.Scope != "repository" || s.ScopeID != repo {
		t.Errorf("repository room scope: %+v", s)
	}
	// repository + to: a session of another user.
	s = testutil.Call[scopedSent](t, a1, "chat_send", map[string]any{"to": "bob/claude", "scope": "repository", "body": "repo direct"})
	if !slices.Equal(s.Recipients, []string{"bob/claude"}) || s.ScopeID != repo {
		t.Errorf("repository direct: %+v", s)
	}

	// Another user's address without scope repository: an error that says to add it.
	for _, scope := range []any{nil, "user"} {
		args := map[string]any{"to": "bob/claude", "body": "x"}
		if scope != nil {
			args["scope"] = scope
		}
		if msg := testutil.CallErr(t, a1, "chat_send", args); !strings.Contains(msg, "scope repository, branch or shared") {
			t.Errorf("scope %v to bob/claude: %q", scope, msg)
		}
	}
	// Outside the scope and missing: the same error.
	errFor := func(to string) string {
		return strings.ReplaceAll(testutil.CallErr(t, a1, "chat_send", map[string]any{"to": to, "scope": "repository", "body": "x"}), to, "<to>")
	}
	outside, missing, noUser := errFor("bob/grok"), errFor("bob/nobody"), errFor("nobody/claude")
	if outside != missing || outside != noUser || !strings.Contains(outside, "lounge_who") {
		t.Errorf("errors differ:\noutside: %s\nmissing: %s\nno user: %s", outside, missing, noUser)
	}
	// alice's codex is in another repository: not reachable in scope repository either.
	if msg := errFor("codex"); msg != outside {
		t.Errorf("own session outside the repository: %s", msg)
	}
	for _, to := range []string{"bob/", "/claude", "a/b/c"} {
		if msg := testutil.CallErr(t, a1, "chat_send", map[string]any{"to": to, "scope": "repository", "body": "x"}); !strings.Contains(msg, "invalid to") {
			t.Errorf("to %q: %s", to, msg)
		}
	}

	// bob's claude received the room and the direct message, with the sender's user and reply_to.
	in := testutil.Call[scopedInbox](t, b1, "chat_inbox", nil)
	if len(in.Messages) != 2 {
		t.Fatalf("bob's inbox = %+v", in)
	}
	for i, room := range []string{"lounge", ""} {
		m := in.Messages[i]
		if m.FromUser != "alice" || m.From != "claude" || m.ReplyTo != "alice/claude" || m.Scope != "repository" || m.ScopeID != repo || m.Room != room {
			t.Errorf("bob's message %d = %+v", i, m)
		}
	}
	// bob replies with the message's scope and reply_to.
	m := in.Messages[1]
	s = testutil.Call[scopedSent](t, b1, "chat_send", map[string]any{"to": m.ReplyTo, "scope": m.Scope, "body": "reply"})
	if !slices.Equal(s.Recipients, []string{"alice/claude"}) {
		t.Errorf("reply: %+v", s)
	}
	in = testutil.Call[scopedInbox](t, a1, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Body != "reply" || in.Messages[0].ReplyTo != "bob/claude" || in.Messages[0].FromUser != "bob" {
		t.Errorf("alice's inbox = %+v", in)
	}
	// alice's codex got the user messages, with reply_to in the own-user form.
	in = testutil.Call[scopedInbox](t, a2, "chat_inbox", nil)
	if len(in.Messages) != 3 || in.Messages[0].ReplyTo != "claude" || in.Messages[0].Scope != "user" || in.Messages[0].ScopeID != "alice" {
		t.Errorf("alice's codex inbox = %+v", in)
	}
	// local's agy got the repository room message; bob's grok nothing.
	if in = testutil.Call[scopedInbox](t, l1, "chat_inbox", nil); len(in.Messages) != 1 || in.Messages[0].ReplyTo != "alice/claude" {
		t.Errorf("local's inbox = %+v", in)
	}
	if in = testutil.Call[scopedInbox](t, b2, "chat_inbox", nil); len(in.Messages) != 0 {
		t.Errorf("bob/grok received %+v", in)
	}
	// bob/grok cannot reach alice in either scope.
	if msg := testutil.CallErr(t, b2, "chat_send", map[string]any{"to": "alice/claude", "scope": "repository", "body": "x"}); !strings.Contains(msg, "no session") {
		t.Errorf("bob/grok to alice/claude: %s", msg)
	}
}

// A name takeover happens only within one user: bob joining under alice's session name gets
// his own session, and alice's connection keeps hers.
func TestTakeoverStaysWithinUser(t *testing.T) {
	base, sess := tokenServer(t, config.Default())
	aliceToken, bobToken := newUser(t, sess, "alice"), newUser(t, sess, "bob")
	a := joinAs(t, base, aliceToken, map[string]any{"name": "claude"})
	cs, err := testutil.ConnectToken(t, base+MCPPath, bobToken)
	if err != nil {
		t.Fatal(err)
	}
	if j := testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "claude"}); j.Resumed || j.User != "bob" {
		t.Errorf("bob's join = %+v", j)
	}
	// The list is ordered by last use, which may tie at the same millisecond: check it by address.
	w := testutil.Call[scopedWho](t, a, "lounge_who", nil)
	you := map[string]bool{}
	for _, s := range w.Sessions {
		you[s.Address] = s.You
	}
	if len(w.Sessions) != 2 || !you["claude"] || you["bob/claude"] {
		t.Errorf("alice after bob's join: %+v", w)
	}
	// Within alice's user, the later join takes the name over.
	a2 := joinAs(t, base, aliceToken, map[string]any{"name": "claude"})
	if msg := testutil.CallErr(t, a, "lounge_who", nil); !strings.Contains(msg, "took the session over") {
		t.Errorf("alice's first connection: %s", msg)
	}
	testutil.Call[scopedWho](t, a2, "lounge_who", nil)
}

func TestBodyLimit(t *testing.T) {
	cs := clients(t, "alice", "bob")
	if msg := testutil.CallErr(t, cs[0], "chat_send", map[string]any{"to": "bob", "body": strings.Repeat("あ", 65536/3+1)}); !strings.Contains(msg, "at most 65536 bytes") {
		t.Errorf("large body: %s", msg)
	}
	testutil.Call[scopedSent](t, cs[0], "chat_send", map[string]any{"to": "bob", "body": strings.Repeat("a", 65536)})
}

// A full inbox fails a direct message and is skipped by a room message.
func TestUnreadLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("sends 1000 messages")
	}
	cs := clients(t, "alice", "bob", "carol")
	ctx := context.Background()
	for range 1000 {
		res, err := cs[0].CallTool(ctx, &mcp.CallToolParams{Name: "chat_send", Arguments: map[string]any{"to": "bob", "body": "x"}})
		if err != nil || res.IsError {
			t.Fatalf("send: %v %v", err, res)
		}
	}
	if msg := testutil.CallErr(t, cs[0], "chat_send", map[string]any{"to": "bob", "body": "x"}); !strings.Contains(msg, "inbox of \"bob\" is full") {
		t.Errorf("direct to a full inbox: %s", msg)
	}
	s := testutil.Call[scopedSent](t, cs[0], "chat_send", map[string]any{"room": "lounge", "body": "room"})
	if !slices.Equal(s.Recipients, []string{"carol"}) || !slices.Equal(s.Skipped, []string{"bob"}) {
		t.Errorf("room with a full inbox: %+v", s)
	}
}

// The usage guide describes scopes, users, limits and the 401 case.
func TestHelpDescribesUsersAndScopes(t *testing.T) {
	cs := connect(t)
	guide := guideOf(t, cs)
	for _, want := range []string{
		"## Scopes and targets", "## Users", "user/name", "reply_to", "scope_id", "never inferred",
		"65536 bytes", "1000 unread", "500 sessions", "401", "decided by the token", "other users",
		"shared: every session", "chat_room action create", "100 rooms besides lounge",
	} {
		if !strings.Contains(guide, want) {
			t.Errorf("guide does not mention %q", want)
		}
	}
	for _, stale := range []string{"on this machine", "No authentication", "## Repository space", "same_repository", "Rooms cannot be created", "not the room messages sent before"} {
		if strings.Contains(guide, stale) || strings.Contains(cs.InitializeResult().Instructions, stale) {
			t.Errorf("guide or instructions still say %q", stale)
		}
	}
}
