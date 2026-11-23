package server

import (
	"slices"
	"strings"
	"testing"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

type roomOut struct {
	Scope   string `json:"scope"`
	ScopeID string `json:"scope_id"`
	Room    string `json:"room"`
	Created *bool  `json:"created"`
	Joined  *bool  `json:"joined"`
	Rooms   []struct {
		roomRef
		Joined  bool `json:"joined"`
		Members int  `json:"members"`
	} `json:"rooms"`
}

// Rooms over MCP: two users of one repository create, join, post to and leave a room; a later
// session reads the room's past messages; names fold case.
func TestChatRoom(t *testing.T) {
	base, sess := tokenServer(t, config.Default())
	aliceToken, bobToken := newUser(t, sess, "alice"), newUser(t, sess, "bob")
	origin := "https://github.com/owner/app.git"
	a := joinAs(t, base, aliceToken, map[string]any{"name": "claude", "origin": origin})
	b := joinAs(t, base, bobToken, map[string]any{"name": "codex", "origin": origin})

	c := testutil.Call[roomOut](t, a, "chat_room", map[string]any{"action": "create", "scope": "repository", "room": "Release"})
	if c.Room != "release" || c.Scope != "repository" || c.ScopeID != "github.com/owner/app" || c.Created == nil || !*c.Created || c.Joined == nil || !*c.Joined {
		t.Fatalf("create = %+v", c)
	}
	// Before bob joins, a post reaches nobody.
	s := testutil.Call[scopedSent](t, a, "chat_send", map[string]any{"room": "RELEASE", "scope": "repository", "body": "before"})
	if len(s.Recipients) != 0 {
		t.Errorf("before join: %+v", s)
	}

	j := testutil.Call[roomOut](t, b, "chat_room", map[string]any{"action": "join", "scope": "repository", "room": "release"})
	if j.Joined == nil || !*j.Joined || j.Created != nil {
		t.Errorf("join = %+v", j)
	}
	s = testutil.Call[scopedSent](t, a, "chat_send", map[string]any{"room": "release", "scope": "repository", "body": "after"})
	if !slices.Equal(s.Recipients, []string{"bob/codex"}) {
		t.Errorf("after join: %+v", s)
	}
	l := testutil.Call[roomOut](t, b, "chat_room", map[string]any{"action": "list", "scope": "repository"})
	if len(l.Rooms) != 2 || l.Rooms[0].Room != "lounge" || l.Rooms[1].Room != "release" || !l.Rooms[1].Joined || l.Rooms[1].Members != 1 {
		t.Errorf("list = %+v", l)
	}
	w := testutil.Call[joined](t, b, "lounge_who", nil)
	if got := roomNames(w.Rooms); !slices.Equal(got, []string{"user:lounge", "repository:lounge", "repository:release", "shared:lounge"}) {
		t.Errorf("who rooms = %v", got)
	}
	if lv := testutil.Call[roomOut](t, b, "chat_room", map[string]any{"action": "leave", "scope": "repository", "room": "release"}); lv.Joined == nil || *lv.Joined {
		t.Errorf("leave = %+v", lv)
	}

	for want, args := range map[string]map[string]any{
		"invalid action":                        {"action": "delete", "room": "x"},
		"give the room name":                    {"action": "join"},
		`no room named "missing" in scope user`: {"action": "join", "room": "missing"},
		"lounge cannot be left":                 {"action": "leave", "room": "lounge"},
		"invalid room name":                     {"action": "create", "room": "a b"},
		"cannot use scope branch":               {"action": "list", "scope": "branch"},
	} {
		if msg := testutil.CallErr(t, b, "chat_room", args); !strings.Contains(msg, want) {
			t.Errorf("%v: got %q, want it to contain %q", args, msg, want)
		}
	}
	if msg := testutil.CallErr(t, a, "chat_send", map[string]any{"room": "missing", "scope": "repository", "body": "x"}); !strings.Contains(msg, `no room named "missing" in scope repository`) {
		t.Errorf("send to a missing room: %q", msg)
	}
}
