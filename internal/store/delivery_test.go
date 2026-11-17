package store

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

// testClock is a clock that moves only when a test advances it.
type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *testClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// useClock makes s use a test clock that starts at a fixed time.
func useClock(s *Store) *testClock {
	c := &testClock{t: time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)}
	s.SetClock(c.now)
	return c
}

func fetch(t *testing.T, s *Store, sess Session) []InboxMessage {
	t.Helper()
	msgs, err := s.FetchInbox(context.Background(), sess.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	return msgs
}

func ack(t *testing.T, s *Store, sess Session, ids ...int64) (acked, notFound []int64) {
	t.Helper()
	acked, notFound, err := s.Ack(context.Background(), sess.ID, ids)
	if err != nil {
		t.Fatal(err)
	}
	return acked, notFound
}

func status(t *testing.T, s *Store, sess Session, id int64) MessageStatus {
	t.Helper()
	st, err := s.MessageStatus(context.Background(), sess.ID, id, 100)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// The states move queued -> delivered -> acked, each with its time.
func TestDeliveryStates(t *testing.T) {
	s, local := openLocal(t)
	clock := useClock(s)
	alice, bob := join(t, s, local, "alice"), join(t, s, local, "bob")
	sent, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "please review")
	if err != nil {
		t.Fatal(err)
	}
	sentAt := clock.now()
	st := status(t, s, alice, sent.MessageID)
	if st.Counts[StateQueued] != 1 || len(st.Recipients) != 1 || st.Recipients[0].State != StateQueued ||
		!st.Recipients[0].DeliveredAt.IsZero() || !st.SentAt.Equal(sentAt) || st.Room != "" || st.Scope != (Scope{ScopeUser, LocalUser}) {
		t.Fatalf("queued: %+v", st)
	}

	clock.advance(time.Minute)
	fetch(t, s, bob)
	st = status(t, s, alice, sent.MessageID)
	r := st.Recipients[0]
	if r.State != StateDelivered || !r.DeliveredAt.Equal(sentAt.Add(time.Minute)) || !r.AckedAt.IsZero() || st.Counts[StateDelivered] != 1 || st.Counts[StateQueued] != 0 {
		t.Fatalf("delivered: %+v", st)
	}

	clock.advance(time.Minute)
	if acked, notFound := ack(t, s, bob, sent.MessageID); !slices.Equal(acked, []int64{sent.MessageID}) || len(notFound) != 0 {
		t.Fatalf("ack: %v %v", acked, notFound)
	}
	st = status(t, s, alice, sent.MessageID)
	r = st.Recipients[0]
	if r.State != StateAcked || !r.AckedAt.Equal(sentAt.Add(2*time.Minute)) || !r.DeliveredAt.Equal(sentAt.Add(time.Minute)) || st.Counts[StateAcked] != 1 {
		t.Fatalf("acked: %+v", st)
	}

	// Acking again gives the same result and keeps the time.
	clock.advance(time.Minute)
	if acked, notFound := ack(t, s, bob, sent.MessageID, sent.MessageID); !slices.Equal(acked, []int64{sent.MessageID}) || len(notFound) != 0 {
		t.Fatalf("second ack: %v %v", acked, notFound)
	}
	if r := status(t, s, alice, sent.MessageID).Recipients[0]; !r.AckedAt.Equal(sentAt.Add(2 * time.Minute)) {
		t.Errorf("acked_at changed: %v", r.AckedAt)
	}
}

// IDs that cannot be acked go to notFound, without telling why.
func TestAckNotFound(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	clock := useClock(s)
	alice, bob, carol := join(t, s, local, "alice"), join(t, s, local, "bob"), join(t, s, local, "carol")

	toExpire, _ := sendTo(s, alice, ScopeUser, LocalUser, "bob", "will expire")
	clock.advance(15 * 24 * time.Hour)
	if _, err := s.Cleanup(ctx, CleanupPolicy{ExpireAfter: 14 * 24 * time.Hour}); err != nil {
		t.Fatal(err)
	}
	queued, _ := sendTo(s, alice, ScopeUser, LocalUser, "bob", "not fetched")
	toCarol, _ := sendTo(s, alice, ScopeUser, LocalUser, "carol", "for carol")
	fetch(t, s, carol)
	good, _ := sendTo(s, alice, ScopeUser, LocalUser, "carol", "fetched")
	fetch(t, s, carol)

	// bob: expired, queued, another session's, missing.
	acked, notFound := ack(t, s, bob, toExpire.MessageID, queued.MessageID, toCarol.MessageID, 99999)
	if len(acked) != 0 || !slices.Equal(notFound, []int64{toExpire.MessageID, queued.MessageID, toCarol.MessageID, 99999}) {
		t.Errorf("bob: acked %v, not found %v", acked, notFound)
	}
	// The sender cannot ack its own message either.
	if acked, notFound := ack(t, s, alice, good.MessageID); len(acked) != 0 || len(notFound) != 1 {
		t.Errorf("sender: acked %v, not found %v", acked, notFound)
	}
	if acked, notFound := ack(t, s, carol, good.MessageID, 99999); !slices.Equal(acked, []int64{good.MessageID}) || !slices.Equal(notFound, []int64{99999}) {
		t.Errorf("carol: acked %v, not found %v", acked, notFound)
	}
	// Nothing changed for the IDs not found.
	if st := status(t, s, alice, queued.MessageID); st.Recipients[0].State != StateQueued {
		t.Errorf("queued message: %+v", st)
	}
}

func TestMessageStatus(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	alice, bob, carol := join(t, s, local, "alice"), join(t, s, local, "bob"), join(t, s, local, "carol")
	other := addUser(t, s, "zed")
	repo := &Location{Origin: "github.com/org/a"}
	joinWith(t, s, JoinParams{UserID: local, Name: "alice", Location: repo})
	z := joinWith(t, s, JoinParams{UserID: other.ID, Name: "a-zed", Location: repo})

	room, err := sendRoom(s, alice, ScopeUser, LoungeRoom, "hello")
	if err != nil {
		t.Fatal(err)
	}
	fetch(t, s, bob)
	st := status(t, s, alice, room.MessageID)
	if st.Room != LoungeRoom || st.Counts[StateQueued] != 1 || st.Counts[StateDelivered] != 1 || st.Counts[StateAcked] != 0 || st.Counts[StateExpired] != 0 {
		t.Errorf("room status: %+v", st)
	}
	var got []string
	for _, r := range st.Recipients {
		got = append(got, r.Name+":"+r.State)
	}
	if !slices.Equal(got, []string{"bob:delivered", "carol:queued"}) {
		t.Errorf("recipients = %v", got)
	}

	// Addresses order: own user's sessions by name, others by user/name.
	repoRoom, err := sendRoom(s, alice, ScopeRepository, LoungeRoom, "repo")
	if err != nil {
		t.Fatal(err)
	}
	if st := status(t, s, alice, repoRoom.MessageID); len(st.Recipients) != 1 || st.Recipients[0].ID != z.ID || st.Scope.Kind != ScopeRepository {
		t.Errorf("repository status: %+v", st)
	}

	// Only the sender reads the status; another session's message and a missing one give the
	// same error.
	for _, tc := range []struct {
		who Session
		id  int64
	}{{bob, room.MessageID}, {carol, room.MessageID}, {alice, 99999}} {
		if _, err := s.MessageStatus(ctx, tc.who.ID, tc.id, 100); !errors.Is(err, ErrNoSuchMessage) {
			t.Errorf("status of %d by %s: %v", tc.id, tc.who.Name, err)
		}
	}

	// A deleted recipient leaves the counts and the list.
	if _, err := s.RemoveSession(ctx, LocalUser, "carol"); err != nil {
		t.Fatal(err)
	}
	if st := status(t, s, alice, room.MessageID); len(st.Recipients) != 1 || st.Counts[StateQueued] != 0 {
		t.Errorf("after removing carol: %+v", st)
	}
}

func TestMessageStatusTruncated(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	alice := join(t, s, local, "alice")
	for i := range 102 {
		join(t, s, local, "s"+string(rune('a'+i/26))+string(rune('a'+i%26)))
	}
	sent, err := sendRoom(s, alice, ScopeUser, LoungeRoom, "all")
	if err != nil || len(sent.Recipients) != 102 {
		t.Fatalf("send: %d %v", len(sent.Recipients), err)
	}
	st, err := s.MessageStatus(ctx, alice.ID, sent.MessageID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Truncated || len(st.Recipients) != 100 || st.Counts[StateQueued] != 102 || st.Recipients[0].Name != "saa" || st.Recipients[99].Name != "sdv" {
		t.Errorf("truncated=%v len=%d counts=%v first=%s last=%s", st.Truncated, len(st.Recipients), st.Counts, st.Recipients[0].Name, st.Recipients[99].Name)
	}
	st, _ = s.MessageStatus(ctx, alice.ID, sent.MessageID, 102)
	if st.Truncated || len(st.Recipients) != 102 {
		t.Errorf("limit 102: truncated=%v len=%d", st.Truncated, len(st.Recipients))
	}
}

func history(t *testing.T, s *Store, sess Session, q HistoryQuery) ([]string, bool) {
	t.Helper()
	if q.Limit == 0 {
		q.Limit = 100
	}
	msgs, more, err := s.History(context.Background(), sess.ID, q)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, m := range msgs {
		bodies = append(bodies, m.Body)
	}
	return bodies, more
}

func TestHistory(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	alice, bob, carol := join(t, s, local, "alice"), join(t, s, local, "bob"), join(t, s, local, "carol")
	must := func(_ Sent, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(sendTo(s, alice, ScopeUser, LocalUser, "bob", "a->b 1"))
	must(sendTo(s, bob, ScopeUser, LocalUser, "alice", "b->a 1"))
	must(sendRoom(s, carol, ScopeUser, LoungeRoom, "c room"))
	must(sendTo(s, carol, ScopeUser, LocalUser, "bob", "c->b"))
	fetch(t, s, alice)
	fetch(t, s, bob)
	must(sendTo(s, bob, ScopeUser, LocalUser, "alice", "b->a queued"))
	// dave joins after the room message: it does not see it.
	dave := join(t, s, local, "dave")

	// alice: what she sent and what was delivered to her, newest first, not the queued one.
	if got, more := history(t, s, alice, HistoryQuery{}); !slices.Equal(got, []string{"c room", "b->a 1", "a->b 1"}) || more {
		t.Errorf("alice: %v %v", got, more)
	}
	// bob: what he sent, including what alice has not fetched yet, and what he fetched.
	if got, _ := history(t, s, bob, HistoryQuery{}); !slices.Equal(got, []string{"b->a queued", "c->b", "c room", "b->a 1", "a->b 1"}) {
		t.Errorf("bob: %v", got)
	}
	// carol: what she sent; messages between other sessions are not hers.
	if got, _ := history(t, s, carol, HistoryQuery{}); !slices.Equal(got, []string{"c->b", "c room"}) {
		t.Errorf("carol: %v", got)
	}
	if got, _ := history(t, s, dave, HistoryQuery{}); len(got) != 0 {
		t.Errorf("dave: %v", got)
	}

	// Filters.
	if got, _ := history(t, s, bob, HistoryQuery{WithUser: LocalUser, WithName: "alice"}); !slices.Equal(got, []string{"b->a queued", "b->a 1", "a->b 1"}) {
		t.Errorf("bob with alice: %v", got)
	}
	if got, _ := history(t, s, bob, HistoryQuery{Room: LoungeRoom}); !slices.Equal(got, []string{"c room"}) {
		t.Errorf("bob room: %v", got)
	}
	if got, _ := history(t, s, bob, HistoryQuery{WithUser: LocalUser, WithName: "nobody"}); len(got) != 0 {
		t.Errorf("bob with nobody: %v", got)
	}
	if got, _ := history(t, s, bob, HistoryQuery{Scope: ScopeRepository}); len(got) != 0 {
		t.Errorf("bob repository: %v", got)
	}
	if got, _ := history(t, s, bob, HistoryQuery{Scope: ScopeUser, Unacked: true}); !slices.Equal(got, []string{"c->b", "c room", "a->b 1"}) {
		t.Errorf("bob unacked: %v", got)
	}

	// Values of the messages.
	msgs, _, err := s.History(ctx, bob.ID, HistoryQuery{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	byBody := map[string]HistoryMessage{}
	for _, m := range msgs {
		byBody[m.Body] = m
	}
	if m := byBody["b->a queued"]; !m.Sent || m.ToName != "alice" || m.ToUser != LocalUser || m.State != "" || m.From != "bob" {
		t.Errorf("sent message: %+v", m)
	}
	if m := byBody["c room"]; m.Sent || m.Room != LoungeRoom || m.ToName != "" || m.State != StateDelivered || m.From != "carol" {
		t.Errorf("room message: %+v", m)
	}

	// Ack changes the state; unacked drops it. History itself changes nothing.
	c2b := byBody["c->b"].MessageID
	ack(t, s, bob, c2b)
	if got, _ := history(t, s, bob, HistoryQuery{Unacked: true}); !slices.Equal(got, []string{"c room", "a->b 1"}) {
		t.Errorf("bob unacked after ack: %v", got)
	}
	if n := unread(t, s, alice); n != 1 {
		t.Errorf("alice unread after history = %d, want 1", n)
	}
	if got := fetch(t, s, alice); len(got) != 1 || got[0].Body != "b->a queued" {
		t.Errorf("alice inbox after history: %+v", got)
	}
}

func TestHistoryPaging(t *testing.T) {
	s, local := openLocal(t)
	alice := join(t, s, local, "alice")
	join(t, s, local, "bob")
	var want []string
	for i := range 7 {
		body := string(rune('0' + i))
		if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", body); err != nil {
			t.Fatal(err)
		}
		want = append([]string{body}, want...)
	}
	var got []string
	var before int64
	for pages := 0; ; pages++ {
		msgs, more, err := s.History(context.Background(), alice.ID, HistoryQuery{Limit: 3, Before: before})
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range msgs {
			got = append(got, m.Body)
		}
		if !more {
			if pages != 2 || len(msgs) != 1 {
				t.Errorf("last page %d has %d messages", pages, len(msgs))
			}
			break
		}
		before = msgs[len(msgs)-1].MessageID
	}
	if !slices.Equal(got, want) {
		t.Errorf("pages = %v, want %v", got, want)
	}
}

// A received message keeps the values its sender declared when it sent it.
func TestSenderSnapshot(t *testing.T) {
	s, local := openLocal(t)
	alice := joinWith(t, s, JoinParams{UserID: local, Name: "alice", Agent: ptr("claude"), OSUser: ptr("honda"), Machine: ptr("host-a")})
	bob := join(t, s, local, "bob")
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "first"); err != nil {
		t.Fatal(err)
	}
	joinWith(t, s, JoinParams{UserID: local, Name: "alice", Agent: ptr("codex"), OSUser: ptr(""), Machine: ptr("host-b")})
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "second"); err != nil {
		t.Fatal(err)
	}
	msgs := fetch(t, s, bob)
	if len(msgs) != 2 || msgs[0].FromAgent != "claude" || msgs[0].FromOSUser != "honda" || msgs[0].FromMachine != "host-a" ||
		msgs[1].FromAgent != "codex" || msgs[1].FromOSUser != "" || msgs[1].FromMachine != "host-b" {
		t.Errorf("inbox = %+v", msgs)
	}
	h, _, err := s.History(context.Background(), bob.ID, HistoryQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 2 || h[1].FromAgent != "claude" || h[1].FromMachine != "host-a" || h[0].FromAgent != "codex" {
		t.Errorf("history = %+v", h)
	}
}

// Limits set on the store apply to new messages and sessions; lowering them deletes nothing.
func TestSetLimits(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	alice, bob := join(t, s, local, "alice"), join(t, s, local, "bob")
	for range 3 {
		if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "x"); err != nil {
			t.Fatal(err)
		}
	}
	s.SetLimits(Limits{MaxBodyBytes: 4, MaxUnread: 2, MaxSessionsPerUser: 2})
	if s.Limits().MaxUnread != 2 {
		t.Errorf("limits = %+v", s.Limits())
	}
	if n := unread(t, s, bob); n != 3 {
		t.Errorf("unread after lowering = %d, want 3", n)
	}
	if _, err := sendTo(s, bob, ScopeUser, LocalUser, "alice", "12345"); !errors.Is(err, ErrBodyTooLarge) {
		t.Errorf("body over the lowered limit: %v", err)
	}
	if _, err := sendTo(s, bob, ScopeUser, LocalUser, "alice", "1234"); err != nil {
		t.Errorf("body at the lowered limit: %v", err)
	}
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "x"); !errors.Is(err, ErrInboxFull) {
		t.Errorf("inbox over the lowered limit: %v", err)
	}
	if _, _, err := s.JoinSession(ctx, JoinParams{UserID: local, Name: "carol"}); !errors.Is(err, ErrTooManySessions) {
		t.Errorf("third session: %v", err)
	}
	if _, resumed, err := s.JoinSession(ctx, JoinParams{UserID: local, Name: "alice"}); err != nil || !resumed {
		t.Errorf("existing session: %v", err)
	}
	if list, _, _ := s.ListSessions(ctx, local, 10); len(list) != 2 {
		t.Errorf("sessions = %d", len(list))
	}
}
