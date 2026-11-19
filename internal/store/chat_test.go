package store

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

// openLocal opens a store with the local user created and returns the user's ID.
func openLocal(t *testing.T) (*Store, int64) {
	t.Helper()
	s := open(t, filepath.Join(t.TempDir(), "lounge.db"))
	u, err := s.EnsureLocalUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return s, u.ID
}

// sendTo sends to a session of the given user in the given scope.
func sendTo(s *Store, sender Session, scope, user, name, body string) (Sent, error) {
	return s.Send(context.Background(), sender.ID, scope, Target{User: user, Name: name}, body)
}

// sendRoom sends to a room of the given scope.
func sendRoom(s *Store, sender Session, scope, room, body string) (Sent, error) {
	return s.Send(context.Background(), sender.ID, scope, Target{Room: room}, body)
}

// names returns the session names of members.
func names(members []Member) []string {
	out := []string{}
	for _, m := range members {
		out = append(out, m.Name)
	}
	return out
}

// ids returns the session IDs of members.
func ids(members []Member) []int64 {
	out := []int64{}
	for _, m := range members {
		out = append(out, m.ID)
	}
	return out
}

func join(t *testing.T, s *Store, userID int64, name string) Session {
	t.Helper()
	sess, _, err := s.JoinSession(context.Background(), JoinParams{UserID: userID, Name: name})
	if err != nil {
		t.Fatal(err)
	}
	return sess
}

// sessionExists reports whether sess is still recorded, without changing it.
func sessionExists(ctx context.Context, s *Store, sess Session) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM sessions WHERE id = ? AND user_id = ? AND name = ?",
		sess.ID, sess.UserID, sess.Name).Scan(&n)
	return n > 0, err
}

// Touch updates the time of last use only of the session with the same ID, user and name.
func TestTouch(t *testing.T) {
	s, user := openLocal(t)
	ctx := context.Background()
	alice := join(t, s, user, "alice")
	lastSeen := func() int64 {
		var v int64
		if err := s.db.QueryRowContext(ctx, "SELECT last_seen_at FROM sessions WHERE id = ?", alice.ID).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE sessions SET last_seen_at = 1 WHERE id = ?", alice.ID); err != nil {
		t.Fatal(err)
	}
	for name, other := range map[string]Session{
		"another name": {ID: alice.ID, UserID: alice.UserID, Name: "bob"},
		"another user": {ID: alice.ID, UserID: alice.UserID + 1, Name: alice.Name},
		"another ID":   {ID: alice.ID + 100, UserID: alice.UserID, Name: alice.Name},
	} {
		if exists, err := s.Touch(ctx, other); err != nil || exists {
			t.Errorf("%s: exists = %v, %v", name, exists, err)
		}
	}
	if lastSeen() != 1 {
		t.Errorf("a touch of another session changed last_seen_at to %d", lastSeen())
	}
	if exists, err := s.Touch(ctx, alice); err != nil || !exists {
		t.Fatalf("exists = %v, %v", exists, err)
	}
	if lastSeen() == 1 {
		t.Error("last_seen_at was not updated")
	}
}

// unread uses the session as a tool call does (Touch), and returns its unread messages.
func unread(t *testing.T, s *Store, sess Session) int {
	t.Helper()
	if _, err := s.Touch(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	n, err := s.Unread(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestEnsureLocalUserIsIdempotent(t *testing.T) {
	ctx := context.Background()
	s, id := openLocal(t)
	again, err := s.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != id || again.Name != LocalUser {
		t.Errorf("second call returned user %+v, want %d", again, id)
	}
	rooms, err := s.ListRooms(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rooms, []string{LoungeRoom}) {
		t.Errorf("rooms = %v, want [lounge]", rooms)
	}
}

func TestJoinSession(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)

	agent := "claude-code@host-a"
	first, resumed, err := s.JoinSession(ctx, JoinParams{UserID: user, Name: "work", Agent: &agent})
	if err != nil {
		t.Fatal(err)
	}
	if resumed || first.Name != "work" || first.Agent != agent {
		t.Fatalf("first join: %+v resumed=%v", first, resumed)
	}

	// Joining again continues the session and keeps the agent name when none is given.
	second, resumed, err := s.JoinSession(ctx, JoinParams{UserID: user, Name: "work"})
	if err != nil {
		t.Fatal(err)
	}
	if !resumed || second.ID != first.ID || second.Agent != agent {
		t.Fatalf("second join: %+v resumed=%v", second, resumed)
	}

	// An explicit agent name replaces the recorded one, including with an empty name.
	empty := ""
	third, _, err := s.JoinSession(ctx, JoinParams{UserID: user, Name: "work", Agent: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if third.Agent != "" {
		t.Errorf("agent = %q, want empty", third.Agent)
	}

	// Without a name the server chooses one.
	auto, resumed, err := s.JoinSession(ctx, JoinParams{UserID: user})
	if err != nil {
		t.Fatal(err)
	}
	if resumed || len(auto.Name) != len("session-0000") || auto.Name[:8] != "session-" || auto.ID == first.ID {
		t.Errorf("generated session: %+v resumed=%v", auto, resumed)
	}
}

func TestListSessionsOrderAndLimit(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	clock := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { clock = clock.Add(time.Second); return clock }

	a, b, c := join(t, s, user, "a"), join(t, s, user, "b"), join(t, s, user, "c")
	_ = b
	unread(t, s, a) // a becomes the most recently used session

	list, truncated, err := s.ListSessions(ctx, user, 10)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, sess := range list {
		names = append(names, sess.Name)
	}
	if truncated || !slices.Equal(names, []string{"a", "c", "b"}) {
		t.Errorf("sessions = %v truncated=%v, want [a c b]", names, truncated)
	}
	if !list[0].LastSeenAt.Equal(clock) || list[0].ID != a.ID || list[1].ID != c.ID {
		t.Errorf("unexpected first entries: %+v", list[:2])
	}

	list, truncated, err = s.ListSessions(ctx, user, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(list) != 2 {
		t.Errorf("limit 2: %d sessions truncated=%v", len(list), truncated)
	}
}

func TestSendToSession(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	alice, bob, carol := join(t, s, user, "alice"), join(t, s, user, "bob"), join(t, s, user, "carol")

	sent, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "こんにちは")
	if err != nil {
		t.Fatal(err)
	}
	if sent.MessageID == 0 || !slices.Equal(names(sent.Recipients), []string{"bob"}) || !slices.Equal(ids(sent.Recipients), []int64{bob.ID}) {
		t.Fatalf("sent = %+v", sent)
	}
	if n := unread(t, s, bob); n != 1 {
		t.Errorf("bob unread = %d, want 1", n)
	}
	if n := unread(t, s, carol) + unread(t, s, alice); n != 0 {
		t.Errorf("others unread = %d, want 0", n)
	}

	msgs, err := s.FetchInbox(ctx, bob.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("bob got %d messages, want 1", len(msgs))
	}
	m := msgs[0]
	if m.MessageID != sent.MessageID || m.From != "alice" || m.Body != "こんにちは" || m.Room != "" || m.Scope != (Scope{ScopeUser, LocalUser}) || m.FromUser != LocalUser || !m.SentAt.Equal(sent.SentAt) {
		t.Errorf("message = %+v", m)
	}

	// Fetched messages are delivered: they are not unread and not handed out again.
	if n := unread(t, s, bob); n != 0 {
		t.Errorf("bob unread after fetch = %d, want 0", n)
	}
	if msgs, err = s.FetchInbox(ctx, bob.ID, 10); err != nil || len(msgs) != 0 {
		t.Errorf("second fetch: %d messages, err %v", len(msgs), err)
	}
	var state string
	var deliveredAt *int64
	if err := s.db.QueryRowContext(ctx, "SELECT state, delivered_at FROM deliveries WHERE message_id = ?", sent.MessageID).Scan(&state, &deliveredAt); err != nil {
		t.Fatal(err)
	}
	if state != "delivered" || deliveredAt == nil {
		t.Errorf("delivery state = %q, delivered_at = %v", state, deliveredAt)
	}
}

func TestSendToSessionErrors(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	alice := join(t, s, user, "alice")

	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "nobody", "x"); !errors.Is(err, ErrNoSuchSession) {
		t.Errorf("unknown destination: err = %v", err)
	}
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "alice", "x"); !errors.Is(err, ErrSendToSelf) {
		t.Errorf("own session: err = %v", err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM messages").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d messages were stored by failed sends", n)
	}
}

func TestSendToRoom(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	alice := join(t, s, user, "alice")

	// Alone in the room: accepted with no recipients.
	sent, err := sendRoom(s, alice, ScopeUser, LoungeRoom, "anyone?")
	if err != nil {
		t.Fatal(err)
	}
	if sent.Recipients == nil || len(sent.Recipients) != 0 {
		t.Errorf("recipients = %#v, want an empty list", sent.Recipients)
	}

	bob, carol := join(t, s, user, "bob"), join(t, s, user, "carol")
	sent, err = sendRoom(s, alice, ScopeUser, LoungeRoom, "hello all")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names(sent.Recipients), []string{"bob", "carol"}) {
		t.Errorf("recipients = %v, want [bob carol]", sent.Recipients)
	}
	if !slices.Equal(ids(sent.Recipients), []int64{bob.ID, carol.ID}) {
		t.Errorf("recipient IDs = %v, want [%d %d]", ids(sent.Recipients), bob.ID, carol.ID)
	}
	// A session that joins later does not receive earlier room messages.
	dave := join(t, s, user, "dave")
	if n := unread(t, s, dave); n != 0 {
		t.Errorf("dave unread = %d, want 0", n)
	}
	if n := unread(t, s, alice); n != 0 {
		t.Errorf("sender unread = %d, want 0", n)
	}
	for _, sess := range []Session{bob, carol} {
		msgs, err := s.FetchInbox(ctx, sess.ID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 || msgs[0].Body != "hello all" || msgs[0].Room != LoungeRoom || msgs[0].From != "alice" {
			t.Errorf("%s got %+v", sess.Name, msgs)
		}
	}

	if _, err := sendRoom(s, alice, ScopeUser, "other", "x"); !errors.Is(err, ErrNoSuchRoom) {
		t.Errorf("unknown room: err = %v", err)
	}
}

func TestFetchInboxLimitAndOrder(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	alice, bob := join(t, s, user, "alice"), join(t, s, user, "bob")
	for _, body := range []string{"1", "2", "3"} {
		if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", body); err != nil {
			t.Fatal(err)
		}
	}
	msgs, err := s.FetchInbox(ctx, bob.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Body != "1" || msgs[1].Body != "2" {
		t.Fatalf("first fetch = %+v", msgs)
	}
	if n := unread(t, s, bob); n != 1 {
		t.Errorf("unread = %d, want 1", n)
	}
	msgs, err = s.FetchInbox(ctx, bob.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Body != "3" {
		t.Fatalf("second fetch = %+v", msgs)
	}
}

// Two connections may read one session's inbox at the same time; each message goes to one of them.
func TestConcurrentFetchDeliversOnce(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	alice, bob := join(t, s, user, "alice"), join(t, s, user, "bob")
	const total = 40
	for range total {
		if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "x"); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	seen := map[int64]int{}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for {
				msgs, err := s.FetchInbox(ctx, bob.ID, 3)
				if err != nil {
					t.Error(err)
					return
				}
				if len(msgs) == 0 {
					return
				}
				mu.Lock()
				for _, m := range msgs {
					seen[m.MessageID]++
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if len(seen) != total {
		t.Errorf("%d distinct messages fetched, want %d", len(seen), total)
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("message %d was handed out %d times", id, n)
		}
	}
}
