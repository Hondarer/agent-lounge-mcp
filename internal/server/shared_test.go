package server

import (
	"slices"
	"strings"
	"testing"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

// Two users whose sessions declare different repositories reach each other in the shared scope.
func TestSharedScope(t *testing.T) {
	base, sess := tokenServer(t, config.Default())
	aliceToken, bobToken := newUser(t, sess, "alice"), newUser(t, sess, "bob")
	a := joinAs(t, base, aliceToken, map[string]any{"name": "claude", "origin": "https://github.com/owner/a.git"})
	b := joinAs(t, base, bobToken, map[string]any{"name": "codex", "origin": "https://github.com/owner/b.git"})

	w := testutil.Call[scopedWho](t, a, "lounge_who", nil)
	var found bool
	for _, s := range w.Sessions {
		if s.Address == "bob/codex" {
			found = slices.Contains(s.Scopes, scopeOut{"shared", ""}) && len(s.Scopes) == 1
		}
	}
	if !found {
		t.Fatalf("who = %+v", w)
	}
	if !slices.Contains(w.Scopes, scopeOut{"shared", ""}) {
		t.Errorf("top-level scopes = %+v", w.Scopes)
	}

	// The error for another user's address with the default scope names shared.
	if msg := testutil.CallErr(t, a, "chat_send", map[string]any{"to": "bob/codex", "body": "x"}); !strings.Contains(msg, "shared") {
		t.Errorf("default scope to bob/codex: %q", msg)
	}

	s := testutil.Call[scopedSent](t, a, "chat_send", map[string]any{"to": "bob/codex", "scope": "shared", "body": "direct"})
	if !slices.Equal(s.Recipients, []string{"bob/codex"}) || s.Scope != "shared" || s.ScopeID != "" {
		t.Errorf("direct: %+v", s)
	}
	s = testutil.Call[scopedSent](t, b, "chat_send", map[string]any{"room": "lounge", "scope": "shared", "body": "room"})
	if !slices.Equal(s.Recipients, []string{"alice/claude"}) || s.Scope != "shared" {
		t.Errorf("room: %+v", s)
	}
	in := testutil.Call[scopedInbox](t, a, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Scope != "shared" || in.Messages[0].ScopeID != "" {
		t.Errorf("inbox = %+v", in)
	}

	// Shared memories: created by alice, found, updated and deleted by bob.
	put := testutil.Call[memoryPutOut](t, a, "memory_put", map[string]any{"body": "shared knowledge", "scope": "shared"})
	if put.Memory.Scope != "shared" || put.Memory.ScopeID != "" {
		t.Fatalf("put = %+v", put)
	}
	got := testutil.Call[memorySearchOut](t, b, "memory_search", map[string]any{"query": "knowledge"})
	if len(got.Memories) != 1 || got.Memories[0].ID != put.Memory.ID {
		t.Fatalf("search = %+v", got)
	}
	up := testutil.Call[memoryPutOut](t, b, "memory_put", map[string]any{"id": put.Memory.ID, "body": "changed"})
	if up.Memory.Body != "changed" {
		t.Errorf("update = %+v", up)
	}
	testutil.Call[struct {
		Deleted bool `json:"deleted"`
	}](t, b, "memory_forget", map[string]any{"id": put.Memory.ID})
	if got := testutil.Call[memorySearchOut](t, a, "memory_search", map[string]any{"scope": "shared"}); len(got.Memories) != 0 {
		t.Errorf("after forget = %+v", got)
	}
}
