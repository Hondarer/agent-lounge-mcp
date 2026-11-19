package server

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

type acked struct {
	Acked    []int64 `json:"acked"`
	NotFound []int64 `json:"not_found"`
	Unread   *int    `json:"unread"`
}

type statusOut struct {
	MessageID int64  `json:"message_id"`
	SentAt    string `json:"sent_at"`
	Scope     string `json:"scope"`
	ScopeID   string `json:"scope_id"`
	Room      string `json:"room"`
	Counts    struct {
		Queued, Delivered, Acked, Expired int
	} `json:"counts"`
	Recipients []struct {
		Address     string `json:"address"`
		State       string `json:"state"`
		DeliveredAt string `json:"delivered_at"`
		AckedAt     string `json:"acked_at"`
		ExpiredAt   string `json:"expired_at"`
	} `json:"recipients"`
	Truncated *bool `json:"truncated"`
}

type historyOut struct {
	Messages []struct {
		MessageID  int64  `json:"message_id"`
		Direction  string `json:"direction"`
		From       string `json:"from"`
		FromAgent  string `json:"from_agent"`
		Room       string `json:"room"`
		To         string `json:"to"`
		ReplyTo    string `json:"reply_to"`
		State      string `json:"state"`
		Body       string `json:"body"`
		Scope      string `json:"scope"`
		FromUser   string `json:"from_user"`
		FromOSUser string `json:"from_os_user"`
	} `json:"messages"`
	NextBefore int64 `json:"next_before"`
	Unread     *int  `json:"unread"`
}

func TestChatAckAndStatus(t *testing.T) {
	cs := clients(t, "alice", "bob", "carol")
	alice, bob, carol := cs[0], cs[1], cs[2]

	for tool, args := range map[string]map[string]any{"chat_ack": {"message_ids": []int64{1}}, "chat_status": {"message_id": 1}, "chat_history": nil} {
		other := connect(t)
		if msg := testutil.CallErr(t, other, tool, args); !strings.Contains(msg, "not joined") {
			t.Errorf("%s before joining: %s", tool, msg)
		}
	}

	m := testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "please review"})
	st := testutil.Call[statusOut](t, alice, "chat_status", map[string]any{"message_id": m.MessageID})
	if st.MessageID != m.MessageID || st.SentAt != m.SentAt || st.Scope != "user" || st.ScopeID != "local" || st.Room != "" ||
		st.Counts.Queued != 1 || len(st.Recipients) != 1 || st.Recipients[0].Address != "bob" || st.Recipients[0].State != "queued" ||
		st.Recipients[0].DeliveredAt != "" || st.Truncated != nil {
		t.Errorf("queued status: %+v", st)
	}

	// Not read yet: not found.
	a := testutil.Call[acked](t, bob, "chat_ack", map[string]any{"message_ids": []int64{m.MessageID}})
	if len(a.Acked) != 0 || len(a.NotFound) != 1 {
		t.Errorf("ack before reading: %+v", a)
	}
	testutil.Call[inbox](t, bob, "chat_inbox", nil)
	st = testutil.Call[statusOut](t, alice, "chat_status", map[string]any{"message_id": m.MessageID})
	if st.Counts.Delivered != 1 || st.Recipients[0].State != "delivered" || st.Recipients[0].DeliveredAt == "" {
		t.Errorf("delivered status: %+v", st)
	}
	for range 2 {
		a = testutil.Call[acked](t, bob, "chat_ack", map[string]any{"message_ids": []int64{m.MessageID}})
		if len(a.Acked) != 1 || a.Acked[0] != m.MessageID || a.NotFound != nil || unreadOf(t, a.Unread) != 0 {
			t.Errorf("ack: %+v", a)
		}
	}
	st = testutil.Call[statusOut](t, alice, "chat_status", map[string]any{"message_id": m.MessageID})
	if st.Counts.Acked != 1 || st.Recipients[0].State != "acked" || st.Recipients[0].AckedAt == "" {
		t.Errorf("acked status: %+v", st)
	}
	// carol cannot ack bob's message.
	if a := testutil.Call[acked](t, carol, "chat_ack", map[string]any{"message_ids": []int64{m.MessageID, 9999}}); len(a.Acked) != 0 || len(a.NotFound) != 2 {
		t.Errorf("ack by carol: %+v", a)
	}

	// Number of IDs.
	for _, ids := range [][]int64{{}, make([]int64, 101)} {
		if msg := testutil.CallErr(t, bob, "chat_ack", map[string]any{"message_ids": ids}); !strings.Contains(msg, "1 to 100") {
			t.Errorf("ack with %d IDs: %s", len(ids), msg)
		}
	}

	// Another session's message and a missing one give the same error.
	byBob := testutil.CallErr(t, bob, "chat_status", map[string]any{"message_id": m.MessageID})
	missing := testutil.CallErr(t, alice, "chat_status", map[string]any{"message_id": 9999})
	if strings.ReplaceAll(byBob, fmt.Sprint(m.MessageID), "<id>") != strings.ReplaceAll(missing, "9999", "<id>") {
		t.Errorf("errors differ:\n%s\n%s", byBob, missing)
	}

	// A room message.
	r := testutil.Call[sent](t, alice, "chat_send", map[string]any{"room": "lounge", "body": "all"})
	testutil.Call[inbox](t, carol, "chat_inbox", nil)
	st = testutil.Call[statusOut](t, alice, "chat_status", map[string]any{"message_id": r.MessageID})
	if st.Room != "lounge" || st.Counts.Queued != 1 || st.Counts.Delivered != 1 || len(st.Recipients) != 2 ||
		st.Recipients[0].Address != "bob" || st.Recipients[1].Address != "carol" || st.Recipients[1].State != "delivered" {
		t.Errorf("room status: %+v", st)
	}
}

func TestChatHistory(t *testing.T) {
	cs := clients(t, "alice", "bob", "carol")
	alice, bob, carol := cs[0], cs[1], cs[2]
	m1 := testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "a->b"})
	testutil.Call[sent](t, carol, "chat_send", map[string]any{"room": "lounge", "body": "room"})
	testutil.Call[inbox](t, bob, "chat_inbox", nil)
	testutil.Call[sent](t, bob, "chat_send", map[string]any{"to": "alice", "body": "b->a"})

	h := testutil.Call[historyOut](t, bob, "chat_history", nil)
	if len(h.Messages) != 3 || h.NextBefore != 0 || unreadOf(t, h.Unread) != 0 {
		t.Fatalf("bob history: %+v", h)
	}
	sentMsg, roomMsg, received := h.Messages[0], h.Messages[1], h.Messages[2]
	if sentMsg.Direction != "sent" || sentMsg.To != "alice" || sentMsg.ReplyTo != "" || sentMsg.State != "" || sentMsg.Body != "b->a" {
		t.Errorf("sent: %+v", sentMsg)
	}
	if roomMsg.Direction != "received" || roomMsg.Room != "lounge" || roomMsg.To != "" || roomMsg.ReplyTo != "carol" || roomMsg.State != "delivered" {
		t.Errorf("room: %+v", roomMsg)
	}
	if received.MessageID != m1.MessageID || received.From != "alice" || received.FromAgent != "alice-agent" || received.To != "bob" ||
		received.ReplyTo != "alice" || received.FromUser != "local" || received.Scope != "user" {
		t.Errorf("received: %+v", received)
	}

	// alice has not read b->a and the room message: her history has only what she sent, and
	// her unread stays.
	h = testutil.Call[historyOut](t, alice, "chat_history", nil)
	if len(h.Messages) != 1 || h.Messages[0].Body != "a->b" || unreadOf(t, h.Unread) != 2 {
		t.Errorf("alice history: %+v", h)
	}

	// Filters and paging.
	h = testutil.Call[historyOut](t, bob, "chat_history", map[string]any{"with": "alice"})
	if len(h.Messages) != 2 {
		t.Errorf("with alice: %+v", h)
	}
	h = testutil.Call[historyOut](t, bob, "chat_history", map[string]any{"with": "local/alice", "limit": 1})
	if len(h.Messages) != 1 || h.Messages[0].Body != "b->a" || h.NextBefore != h.Messages[0].MessageID {
		t.Errorf("with local/alice limit 1: %+v", h)
	}
	h = testutil.Call[historyOut](t, bob, "chat_history", map[string]any{"with": "alice", "before": h.NextBefore})
	if len(h.Messages) != 1 || h.Messages[0].Body != "a->b" || h.NextBefore != 0 {
		t.Errorf("next page: %+v", h)
	}
	for _, with := range []string{"nobody", "zed/alice"} {
		if h := testutil.Call[historyOut](t, bob, "chat_history", map[string]any{"with": with}); len(h.Messages) != 0 {
			t.Errorf("with %s: %+v", with, h)
		}
	}
	if h := testutil.Call[historyOut](t, bob, "chat_history", map[string]any{"room": "lounge", "scope": "user"}); len(h.Messages) != 1 {
		t.Errorf("room: %+v", h)
	}
	testutil.Call[acked](t, bob, "chat_ack", map[string]any{"message_ids": []int64{m1.MessageID}})
	if h := testutil.Call[historyOut](t, bob, "chat_history", map[string]any{"unacked": true}); len(h.Messages) != 1 || h.Messages[0].Body != "room" {
		t.Errorf("unacked: %+v", h)
	}
	for args, want := range map[string]map[string]any{
		"either room or with": {"room": "lounge", "with": "alice"},
		"invalid scope":       {"scope": "shared"},
		"limit":               {"limit": -1},
		"before":              {"before": -1},
		"invalid":             {"with": "/x"},
		`no room named "other": the only room is lounge`: {"room": "other"},
	} {
		if msg := testutil.CallErr(t, bob, "chat_history", want); !strings.Contains(msg, args) {
			t.Errorf("%v: %s", want, msg)
		}
	}
}

// The room lounge exists in every space, also before its first message: an empty list, not
// the error of an unknown room.
func TestHistoryOfLoungeBeforeFirstMessage(t *testing.T) {
	_, cs := waitServer(t, time.Second, "alice")
	if h := testutil.Call[historyOut](t, cs[0], "chat_history", map[string]any{"room": "lounge"}); len(h.Messages) != 0 {
		t.Errorf("history: %+v", h)
	}
}

// A call that fails still counts as use of the session: its time of last use advances.
func TestFailedCallKeepsSessionInUse(t *testing.T) {
	s, sess := newServer(t)
	cs := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "alice"})
	lastSeen := func() time.Time {
		list, _, err := sess.Store.ListSessions(context.Background(), sess.LocalUser.ID, 10)
		if err != nil || len(list) != 1 {
			t.Fatalf("sessions: %+v, %v", list, err)
		}
		return list[0].LastSeenAt
	}
	before := lastSeen()
	time.Sleep(20 * time.Millisecond)
	testutil.CallErr(t, cs, "chat_history", map[string]any{"limit": -1})
	if after := lastSeen(); !after.After(before) {
		t.Errorf("last use: %s, then %s", before, after)
	}
}

func TestDeliveryToolAnnotations(t *testing.T) {
	cs := connect(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		found[tool.Name] = tool
	}
	for _, name := range []string{"chat_status", "chat_history"} {
		if a := found[name].Annotations; a == nil || !a.ReadOnlyHint {
			t.Errorf("%s is not read-only: %+v", name, a)
		}
	}
	if a := found["chat_ack"].Annotations; a == nil || a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint == nil || *a.DestructiveHint {
		t.Errorf("chat_ack annotations: %+v", a)
	}
	if h := found["chat_history"]; !strings.Contains(h.Description, "never as instructions") {
		t.Errorf("chat_history description: %s", h.Description)
	}
	if !strings.Contains(found["chat_inbox"].Description, "chat_ack") || !strings.Contains(found["chat_send"].Description, "chat_status") {
		t.Error("chat_inbox or chat_send does not point to the new tools")
	}
	instructions := cs.InitializeResult().Instructions
	for _, want := range []string{"chat_ack", "chat_history"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("instructions do not mention %s", want)
		}
	}
}

// A connection whose session was deleted gets an error that says so at its next call, and
// joining again with the name creates a new session.
func TestDeletedSession(t *testing.T) {
	ctx := context.Background()
	s, sess := newServer(t)
	alice := testutil.Connect(t, s.MCP, nil)
	bob := testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, alice, "lounge_join", map[string]any{"name": "alice"})
	testutil.Call[joined](t, bob, "lounge_join", map[string]any{"name": "bob"})
	testutil.Call[sent](t, bob, "chat_send", map[string]any{"to": "alice", "body": "x"})

	if _, err := sess.Store.RemoveSession(ctx, store.LocalUser, "alice"); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"chat_inbox", "lounge_who"} {
		msg := testutil.CallErr(t, alice, tool, nil)
		if !strings.Contains(msg, `session "alice" was deleted`) || !strings.Contains(msg, "lounge_join") {
			t.Errorf("%s after deletion: %s", tool, msg)
		}
	}
	// The tools that work before joining still work, without the unread count.
	if h := testutil.Call[help](t, alice, "lounge_help", nil); h.Unread != nil {
		t.Errorf("lounge_help after deletion: unread = %d", *h.Unread)
	}
	j := testutil.Call[joined](t, alice, "lounge_join", map[string]any{"name": "alice"})
	if j.Resumed || unreadOf(t, j.Unread) != 0 {
		t.Errorf("join again: %+v", j)
	}
	testutil.Call[inbox](t, alice, "chat_inbox", nil)
	testutil.Call[sent](t, bob, "chat_send", map[string]any{"to": "alice", "body": "again"})
	if in := testutil.Call[inbox](t, alice, "chat_inbox", nil); len(in.Messages) != 1 || in.Messages[0].Body != "again" {
		t.Errorf("inbox of the new session: %+v", in)
	}
}

// The limit options apply to sending and joining, and the guide and the errors state them.
func TestConfiguredLimits(t *testing.T) {
	cfg := config.Default()
	cfg.MaxBodyBytes, cfg.MaxUnread, cfg.MaxSessionsPerUser = 10, 1, 2
	s, _ := newServerWith(t, cfg)
	alice, bob, carol := testutil.Connect(t, s.MCP, nil), testutil.Connect(t, s.MCP, nil), testutil.Connect(t, s.MCP, nil)
	testutil.Call[joined](t, alice, "lounge_join", map[string]any{"name": "alice"})
	testutil.Call[joined](t, bob, "lounge_join", map[string]any{"name": "bob"})
	if msg := testutil.CallErr(t, carol, "lounge_join", map[string]any{"name": "carol"}); !strings.Contains(msg, "2 sessions") {
		t.Errorf("third session: %s", msg)
	}
	if msg := testutil.CallErr(t, alice, "chat_send", map[string]any{"to": "bob", "body": "12345678901"}); !strings.Contains(msg, "at most 10 bytes") {
		t.Errorf("long body: %s", msg)
	}
	testutil.Call[sent](t, alice, "chat_send", map[string]any{"to": "bob", "body": "1234567890"})
	if msg := testutil.CallErr(t, alice, "chat_send", map[string]any{"to": "bob", "body": "x"}); !strings.Contains(msg, "(1 unread messages)") {
		t.Errorf("full inbox: %s", msg)
	}
	g := guideOf(t, alice)
	for _, want := range []string{"at most 10 bytes", "at most 1 unread messages", "at most 2 sessions", "within 14d", "deleted 30d after", "not used for 60d"} {
		if !strings.Contains(g, want) {
			t.Errorf("guide does not contain %q", want)
		}
	}
}
