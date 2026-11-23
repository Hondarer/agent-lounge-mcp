package server

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

type joined struct {
	Name    string    `json:"name"`
	Agent   string    `json:"agent"`
	User    string    `json:"user"`
	Resumed bool      `json:"resumed"`
	Rooms   []roomRef `json:"rooms"`
	Unread  *int      `json:"unread"`
}

type sent struct {
	MessageID  int64    `json:"message_id"`
	SentAt     string   `json:"sent_at"`
	Recipients []string `json:"recipients"`
	Unread     *int     `json:"unread"`
}

type inbox struct {
	Messages []struct {
		MessageID int64  `json:"message_id"`
		From      string `json:"from"`
		FromAgent string `json:"from_agent"`
		Scope     string `json:"scope"`
		Room      string `json:"room"`
		Body      string `json:"body"`
		SentAt    string `json:"sent_at"`
	} `json:"messages"`
	Unread *int `json:"unread"`
}

type who struct {
	Sessions []struct {
		Name      string `json:"name"`
		Agent     string `json:"agent"`
		Connected bool   `json:"connected"`
		You       bool   `json:"you"`
	} `json:"sessions"`
	Rooms  []roomRef `json:"rooms"`
	Unread *int      `json:"unread"`
}

// clients connects n clients to one server and joins them under the given names.
func clients(t *testing.T, names ...string) []*mcp.ClientSession {
	t.Helper()
	s, _ := newServer(t)
	var out []*mcp.ClientSession
	for _, name := range names {
		cs := testutil.Connect(t, s.MCP, nil)
		testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": name, "agent": name + "-agent"})
		out = append(out, cs)
	}
	return out
}

func unreadOf(t *testing.T, p *int) int {
	t.Helper()
	if p == nil {
		t.Fatal("result has no unread field")
	}
	return *p
}

func TestJoinAndWho(t *testing.T) {
	s, _ := newServer(t)
	alice := testutil.Connect(t, s.MCP, nil)

	j := testutil.Call[joined](t, alice, "lounge_join", map[string]any{"name": "alice", "agent": "claude-code@host-a"})
	if j.Name != "alice" || j.Agent != "claude-code@host-a" || j.User != "local" || j.Resumed || !slices.Equal(roomNames(j.Rooms), []string{"user:lounge", "shared:lounge"}) || unreadOf(t, j.Unread) != 0 {
		t.Fatalf("join = %+v", j)
	}

	bob := testutil.Connect(t, s.MCP, nil)
	auto := testutil.Call[joined](t, bob, "lounge_join", nil)
	if !strings.HasPrefix(auto.Name, "session-") {
		t.Errorf("generated name = %q", auto.Name)
	}

	w := testutil.Call[who](t, alice, "lounge_who", nil)
	if len(w.Sessions) != 2 || !slices.Equal(roomNames(w.Rooms), []string{"user:lounge", "shared:lounge"}) {
		t.Fatalf("who = %+v", w)
	}
	for _, sess := range w.Sessions {
		if !sess.Connected || sess.You != (sess.Name == "alice") {
			t.Errorf("session %+v", sess)
		}
	}
}

func TestJoinRejectsInvalidArguments(t *testing.T) {
	s, _ := newServer(t)
	cs := testutil.Connect(t, s.MCP, nil)
	for _, name := range []string{"has space", "日本。", strings.Repeat("a", 65), "a/b"} {
		if msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": name}); !strings.Contains(msg, "invalid name") {
			t.Errorf("name %q: %s", name, msg)
		}
	}
	if msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": "ok", "agent": strings.Repeat("a", 129)}); !strings.Contains(msg, "too long") {
		t.Errorf("long agent: %s", msg)
	}
	// A failed join leaves the connection not joined.
	if msg := testutil.CallErr(t, cs, "lounge_who", nil); !strings.Contains(msg, "lounge_join") {
		t.Errorf("who after failed joins: %s", msg)
	}
}

func TestToolsRequireJoin(t *testing.T) {
	s, _ := newServer(t)
	cs := testutil.Connect(t, s.MCP, nil)
	for name, args := range map[string]map[string]any{
		"lounge_who": nil,
		"chat_send":  {"to": "x", "body": "hi"},
		"chat_inbox": nil,
	} {
		if msg := testutil.CallErr(t, cs, name, args); !strings.Contains(msg, "not joined") || !strings.Contains(msg, "lounge_join") {
			t.Errorf("%s before joining: %s", name, msg)
		}
	}
	// lounge_info works without joining and reports no unread count.
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "lounge_info"})
	if err != nil || res.IsError {
		t.Fatalf("lounge_info: %v %v", err, res)
	}
	if strings.Contains(testutil.Text(res), "nread") {
		t.Errorf("lounge_info before joining mentions unread: %s", testutil.Text(res))
	}
}

func TestDirectConversation(t *testing.T) {
	cs := clients(t, "alice", "bob", "carol")
	alice, bob, carol := cs[0], cs[1], cs[2]

	s := testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "こんにちは、bob"})
	if s.MessageID == 0 || !slices.Equal(s.Recipients, []string{"bob"}) || s.SentAt == "" {
		t.Fatalf("send = %+v", s)
	}

	// Bob notices the message in the result of an unrelated tool.
	if w := testutil.Call[who](t, bob, "lounge_who", nil); unreadOf(t, w.Unread) != 1 {
		t.Errorf("bob unread = %d, want 1", *w.Unread)
	}
	if w := testutil.Call[who](t, carol, "lounge_who", nil); unreadOf(t, w.Unread) != 0 {
		t.Errorf("carol unread = %d, want 0", *w.Unread)
	}

	in := testutil.Call[inbox](t, bob, "chat_inbox", nil)
	if len(in.Messages) != 1 {
		t.Fatalf("bob inbox = %+v", in)
	}
	m := in.Messages[0]
	if m.MessageID != s.MessageID || m.From != "alice" || m.FromAgent != "alice-agent" || m.Scope != "user" || m.Room != "" || m.Body != "こんにちは、bob" || m.SentAt != s.SentAt {
		t.Errorf("message = %+v", m)
	}
	if unreadOf(t, in.Unread) != 0 {
		t.Errorf("unread after reading = %d", *in.Unread)
	}

	// Bob replies; Alice sees the reply.
	testutil.Call[sent](t, bob, "chat_send", map[string]any{"to": "alice", "body": "reply"})
	in = testutil.Call[inbox](t, alice, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].From != "bob" || in.Messages[0].Body != "reply" {
		t.Errorf("alice inbox = %+v", in)
	}
	// Each message is handed out once.
	if in = testutil.Call[inbox](t, alice, "chat_inbox", nil); len(in.Messages) != 0 {
		t.Errorf("second read = %+v", in)
	}
}

func TestRoomConversation(t *testing.T) {
	cs := clients(t, "alice", "bob", "carol")
	alice, bob, carol := cs[0], cs[1], cs[2]

	s := testutil.Call[sent](t, alice, "chat_send", map[string]any{"room": "lounge", "body": "hello all"})
	if !slices.Equal(s.Recipients, []string{"bob", "carol"}) {
		t.Fatalf("recipients = %v", s.Recipients)
	}
	for _, c := range []*mcp.ClientSession{bob, carol} {
		in := testutil.Call[inbox](t, c, "chat_inbox", nil)
		if len(in.Messages) != 1 || in.Messages[0].Room != "lounge" || in.Messages[0].From != "alice" || in.Messages[0].Body != "hello all" {
			t.Errorf("inbox = %+v", in)
		}
	}
	if in := testutil.Call[inbox](t, alice, "chat_inbox", nil); len(in.Messages) != 0 {
		t.Errorf("the sender received its own room message: %+v", in)
	}
}

func TestSendErrors(t *testing.T) {
	cs := clients(t, "alice", "bob")
	alice := cs[0]
	for want, args := range map[string]map[string]any{
		"not both":                              {"to": "bob", "room": "lounge", "body": "x"},
		"give a destination":                    {"body": "x"},
		"must not be empty":                     {"to": "bob", "body": ""},
		"lounge_who":                            {"to": "nobody", "body": "x"},
		"your own session":                      {"to": "alice", "body": "x"},
		"no room named \"other\" in scope user": {"room": "other", "body": "x"},
	} {
		if msg := testutil.CallErr(t, alice, "chat_send", args); !strings.Contains(msg, want) {
			t.Errorf("%v: got %q, want it to contain %q", args, msg, want)
		}
	}
	if in := testutil.Call[inbox](t, cs[1], "chat_inbox", nil); len(in.Messages) != 0 {
		t.Errorf("failed sends were delivered: %+v", in)
	}
}

func TestInboxLimit(t *testing.T) {
	cs := clients(t, "alice", "bob")
	alice, bob := cs[0], cs[1]
	for _, body := range []string{"1", "2", "3"} {
		testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": body})
	}
	in := testutil.Call[inbox](t, bob, "chat_inbox", map[string]any{"limit": 2})
	if len(in.Messages) != 2 || in.Messages[0].Body != "1" || in.Messages[1].Body != "2" || unreadOf(t, in.Unread) != 1 {
		t.Fatalf("first read = %+v", in)
	}
	if msg := testutil.CallErr(t, bob, "chat_inbox", map[string]any{"limit": -1}); !strings.Contains(msg, "limit") {
		t.Errorf("negative limit: %s", msg)
	}
	in = testutil.Call[inbox](t, bob, "chat_inbox", map[string]any{"limit": 1000})
	if len(in.Messages) != 1 || in.Messages[0].Body != "3" {
		t.Fatalf("second read = %+v", in)
	}
}

// The unread count is also the last line of the result text, for agents that read only the text.
func TestUnreadLineInText(t *testing.T) {
	cs := clients(t, "alice", "bob")
	alice, bob := cs[0], cs[1]
	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "x"})

	// In this order: chat_inbox reads the message that lounge_info reports.
	for _, tc := range []struct{ name, wantLine string }{
		{"lounge_info", "Unread messages: 1 (call chat_inbox to read them)"},
		{"chat_inbox", "Unread messages: 0"},
	} {
		name, wantLine := tc.name, tc.wantLine
		res, err := bob.CallTool(context.Background(), &mcp.CallToolParams{Name: name})
		if err != nil || res.IsError {
			t.Fatalf("%s: %v %v", name, err, res)
		}
		if len(res.Content) != 1 {
			t.Fatalf("%s returned %d content items, want 1", name, len(res.Content))
		}
		body, last, ok := strings.Cut(testutil.Text(res), "\n")
		if !ok || last != wantLine {
			t.Errorf("%s text ends with %q, want %q", name, last, wantLine)
		}
		// The first line is the same JSON as the structured output.
		var fromText, structured map[string]any
		if err := json.Unmarshal([]byte(body), &fromText); err != nil {
			t.Fatalf("%s: first line is not JSON: %v", name, err)
		}
		b, _ := json.Marshal(res.StructuredContent)
		json.Unmarshal(b, &structured)
		if len(fromText) == 0 || len(fromText) != len(structured) {
			t.Errorf("%s: text %v differs from structured output %v", name, fromText, structured)
		}
	}
}

func TestRejoinContinuesSession(t *testing.T) {
	s, _ := newServer(t)
	alice := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, alice, "lounge_join", map[string]any{"name": "alice"})
	bob := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, bob, "lounge_join", map[string]any{"name": "bob", "agent": "codex"})
	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "while you were away"})

	// Bob's connection ends. The session and its unread message stay.
	bob.Close()
	testutil.WaitFor(t, func() bool {
		for _, sess := range testutil.Call[who](t, alice, "lounge_who", nil).Sessions {
			if sess.Name == "bob" {
				return !sess.Connected
			}
		}
		return false
	}, "bob is still shown as connected after its connection ended")

	// A new connection is not joined until it joins again with the same name.
	bob2 := testutil.Connect(t, s.MCP, nil)
	if msg := testutil.CallErr(t, bob2, "chat_inbox", nil); !strings.Contains(msg, "not joined") {
		t.Errorf("inbox before rejoining: %s", msg)
	}
	j := testutil.Call[joined](t, bob2, "lounge_join", map[string]any{"name": "bob"})
	if !j.Resumed || j.Agent != "codex" || unreadOf(t, j.Unread) != 1 {
		t.Fatalf("rejoin = %+v", j)
	}
	in := testutil.Call[inbox](t, bob2, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Body != "while you were away" {
		t.Errorf("inbox after rejoining = %+v", in)
	}
}

func TestLaterJoinTakesNameOver(t *testing.T) {
	s, _ := newServer(t)
	first := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, first, "lounge_join", map[string]any{"name": "worker"})
	second := testutil.Connect(t, s.MCP, nil)
	if j := testutil.Call[joined](t, second, "lounge_join", map[string]any{"name": "worker"}); !j.Resumed {
		t.Errorf("second join = %+v", j)
	}

	// The first connection is told what happened and can join under another name.
	msg := testutil.CallErr(t, first, "chat_inbox", nil)
	if !strings.Contains(msg, "took the session over") || !strings.Contains(msg, "worker") {
		t.Errorf("first connection after takeover: %s", msg)
	}
	testutil.Call[who](t, second, "lounge_who", nil)
	testutil.Call[joined](t, first, "lounge_join", map[string]any{"name": "worker-2"})
	w := testutil.Call[who](t, first, "lounge_who", nil)
	if len(w.Sessions) != 2 {
		t.Fatalf("who = %+v", w)
	}
	for _, sess := range w.Sessions {
		if !sess.Connected || sess.You != (sess.Name == "worker-2") {
			t.Errorf("session %+v", sess)
		}
	}
}

func TestJoinUnderAnotherNameSwitchesSession(t *testing.T) {
	cs := clients(t, "alice", "bob")
	alice, bob := cs[0], cs[1]
	testutil.Call[joined](t, bob, "lounge_join", map[string]any{"name": "bob-2"})

	w := testutil.Call[who](t, alice, "lounge_who", nil)
	connected := map[string]bool{}
	for _, sess := range w.Sessions {
		connected[sess.Name] = sess.Connected
	}
	if connected["bob"] || !connected["bob-2"] || !connected["alice"] {
		t.Errorf("connected = %v", connected)
	}
	// Messages to the old name wait there; the connection now reads bob-2's inbox.
	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "old"})
	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob-2", "body": "new"})
	in := testutil.Call[inbox](t, bob, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Body != "new" {
		t.Errorf("inbox = %+v", in)
	}
}

// The same conversation over Streamable HTTP, as agents connect.
func TestConversationOverHTTP(t *testing.T) {
	base := serveHTTP(t)
	alice := testutil.ConnectHTTP(t, base+MCPPath)
	bob := testutil.ConnectHTTP(t, base+MCPPath)
	testutil.Call[joined](t, alice, "lounge_join", map[string]any{"name": "alice"})
	testutil.Call[joined](t, bob, "lounge_join", map[string]any{"name": "bob"})

	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "over http"})
	// The join is remembered across the HTTP requests of one MCP session.
	in := testutil.Call[inbox](t, bob, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Body != "over http" || in.Messages[0].From != "alice" {
		t.Fatalf("inbox = %+v", in)
	}
	testutil.Call[sent](t, bob, "chat_send", map[string]any{"room": "lounge", "body": "to the room"})
	in = testutil.Call[inbox](t, alice, "chat_inbox", nil)
	if len(in.Messages) != 1 || in.Messages[0].Room != "lounge" {
		t.Fatalf("inbox = %+v", in)
	}
}

func TestJoinAndAddressNormalizedSessionName(t *testing.T) {
	s, _ := newServer(t)
	recipient := testutil.Connect(t, s.MCP, nil)
	j := testutil.Call[joined](t, recipient, "lounge_join", map[string]any{"name": "レビュー担当２"})
	if j.Name != "レビュー担当2" {
		t.Fatalf("join name = %q", j.Name)
	}

	sender := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, sender, "lounge_join", map[string]any{"name": "sender"})
	for _, address := range []string{"レビュー担当2", "レビュー担当２", "local/レビュー担当2", "ＬＯＣＡＬ/レビュー担当２"} {
		out := testutil.Call[sent](t, sender, "chat_send", map[string]any{"to": address, "body": address})
		if !slices.Equal(out.Recipients, []string{j.Name}) {
			t.Errorf("send to %q: recipients = %v", address, out.Recipients)
		}
	}
	if got := testutil.Call[inbox](t, recipient, "chat_inbox", nil); len(got.Messages) != 4 {
		t.Errorf("received %d messages, want 4", len(got.Messages))
	}

	for _, name := range []string{"Reviewer", "reviewer"} {
		j := testutil.Call[joined](t, testutil.Connect(t, s.MCP, nil), "lounge_join", map[string]any{"name": name})
		if j.Name != name || j.Resumed {
			t.Errorf("join %q = %+v", name, j)
		}
	}
}

// roomRef is a room in the output of lounge_join, lounge_who and chat_room.
type roomRef struct {
	Scope   string `json:"scope"`
	ScopeID string `json:"scope_id"`
	Room    string `json:"room"`
}

// roomNames returns the rooms as scope:room.
func roomNames(rooms []roomRef) []string {
	out := make([]string, 0, len(rooms))
	for _, r := range rooms {
		out = append(out, r.Scope+":"+r.Room)
	}
	return out
}
