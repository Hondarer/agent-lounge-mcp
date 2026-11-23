package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const day = 24 * time.Hour

var defaultPolicy = CleanupPolicy{ExpireAfter: 14 * day, RetainFor: 30 * day, SessionIdle: 60 * day}

func cleanup(t *testing.T, s *Store, p CleanupPolicy) CleanupResult {
	t.Helper()
	r, err := s.Cleanup(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func messageExists(t *testing.T, s *Store, id int64) bool {
	t.Helper()
	var n int
	if err := s.db.QueryRowContext(context.Background(), "SELECT count(*) FROM messages WHERE id = ?", id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func deliveryState(t *testing.T, s *Store, id int64, recipient Session) (string, time.Time) {
	t.Helper()
	var state string
	var expired *int64
	err := s.db.QueryRowContext(context.Background(),
		"SELECT state, expired_at FROM deliveries WHERE message_id = ? AND recipient_session_id = ?", id, recipient.ID).Scan(&state, &expired)
	if err != nil {
		t.Fatal(err)
	}
	if expired == nil {
		return state, time.Time{}
	}
	return state, fromMillis(*expired)
}

// keepAlive marks sessions as used, so that the session step does not delete them.
func keepAlive(t *testing.T, s *Store, sessions ...Session) {
	t.Helper()
	for _, sess := range sessions {
		unread(t, s, sess)
	}
}

// A queued delivery expires after the period; a delivered one never expires. Expired ones are
// not unread, not handed out by FetchInbox, and readable with History.
func TestCleanupExpires(t *testing.T) {
	s, local := openLocal(t)
	clock := useClock(s)
	alice, bob := join(t, s, local, "alice"), join(t, s, local, "bob")
	fetched, _ := sendTo(s, alice, ScopeUser, LocalUser, "bob", "fetched")
	fetch(t, s, bob)
	queued, _ := sendTo(s, alice, ScopeUser, LocalUser, "bob", "queued")

	clock.advance(14 * day)
	if r := cleanup(t, s, defaultPolicy); !r.Empty() {
		t.Errorf("at the period: %+v", r)
	}
	clock.advance(time.Millisecond)
	if r := cleanup(t, s, defaultPolicy); r != (CleanupResult{Expired: 1}) {
		t.Errorf("after the period: %+v", r)
	}
	if state, at := deliveryState(t, s, queued.MessageID, bob); state != StateExpired || !at.Equal(clock.now()) {
		t.Errorf("queued delivery: %s %v", state, at)
	}
	if state, _ := deliveryState(t, s, fetched.MessageID, bob); state != StateDelivered {
		t.Errorf("delivered delivery: %s", state)
	}
	if n := unread(t, s, bob); n != 0 {
		t.Errorf("unread = %d, want 0", n)
	}
	if msgs := fetch(t, s, bob); len(msgs) != 0 {
		t.Errorf("inbox = %+v", msgs)
	}
	msgs, _, err := s.History(context.Background(), bob.ID, HistoryQuery{Limit: 10})
	if err != nil || len(msgs) != 2 || msgs[0].Body != "queued" || msgs[0].State != StateExpired {
		t.Errorf("history = %+v %v", msgs, err)
	}
	if st := status(t, s, alice, queued.MessageID); st.Counts[StateExpired] != 1 || !st.Recipients[0].ExpiredAt.Equal(clock.now()) {
		t.Errorf("status = %+v", st)
	}
}

// A message is deleted RetainFor after its last change of state, once nothing is queued.
func TestCleanupDeletesMessages(t *testing.T) {
	s, local := openLocal(t)
	clock := useClock(s)
	alice, bob := join(t, s, local, "alice"), join(t, s, local, "bob")
	join(t, s, local, "carol")
	p := CleanupPolicy{RetainFor: 30 * day}

	// A room message: bob fetches and acks it after 10 days, carol never fetches it.
	room, _ := sendRoom(s, alice, ScopeUser, LoungeRoom, "room")
	// A message without recipients (alice alone in the scope): its sent time counts.
	other := addUser(t, s, "other")
	lonely := join(t, s, other.ID, "lonely")
	none, err := sendRoom(s, lonely, ScopeUser, LoungeRoom, "nobody hears")
	if err != nil || len(none.Recipients) != 0 {
		t.Fatalf("send without recipients: %+v %v", none, err)
	}
	clock.advance(10 * day)
	fetch(t, s, bob)
	ack(t, s, bob, room.MessageID)

	// 31 days after sending: the message without deliveries goes; the room message has a
	// queued delivery (carol) and stays.
	clock.advance(21*day + time.Millisecond)
	if r := cleanup(t, s, p); r != (CleanupResult{Messages: 1}) {
		t.Errorf("after 31 days: %+v", r)
	}
	if messageExists(t, s, none.MessageID) || !messageExists(t, s, room.MessageID) {
		t.Error("wrong messages deleted")
	}
	// carol's delivery expires at day 31: the last change is now.
	cleanup(t, s, CleanupPolicy{ExpireAfter: 14 * day})
	clock.advance(30 * day)
	if r := cleanup(t, s, p); !r.Empty() {
		t.Errorf("30 days after the expiry: %+v", r)
	}
	clock.advance(time.Millisecond)
	if r := cleanup(t, s, p); r != (CleanupResult{Messages: 1, MessageDeliveries: 2}) {
		t.Errorf("after 30 days after the expiry: %+v", r)
	}
	if messageExists(t, s, room.MessageID) {
		t.Error("room message not deleted")
	}

	// A delivered message that is never acked is deleted too.
	m, _ := sendTo(s, alice, ScopeUser, LocalUser, "bob", "never acked")
	clock.advance(day)
	fetch(t, s, bob)
	clock.advance(30*day + time.Millisecond)
	if r := cleanup(t, s, p); r != (CleanupResult{Messages: 1, MessageDeliveries: 1}) || messageExists(t, s, m.MessageID) {
		t.Errorf("delivered message: %+v", r)
	}

	// A queued message is never deleted by this step, however old.
	q, _ := sendTo(s, alice, ScopeUser, LocalUser, "carol", "queued")
	clock.advance(365 * day)
	if r := cleanup(t, s, p); !r.Empty() || !messageExists(t, s, q.MessageID) {
		t.Errorf("queued message: %+v", r)
	}
}

// Sessions unused for longer than SessionIdle are deleted with their messages and deliveries.
func TestCleanupDeletesSessions(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	clock := useClock(s)
	alice, bob, carol := join(t, s, local, "alice"), join(t, s, local, "bob"), join(t, s, local, "carol")
	fromAlice, _ := sendRoom(s, alice, ScopeUser, LoungeRoom, "from alice") // to bob and carol
	toAlice, _ := sendTo(s, bob, ScopeUser, LocalUser, "alice", "to alice")
	betweenOthers, _ := sendTo(s, bob, ScopeUser, LocalUser, "carol", "bob to carol")
	roomByBob, _ := sendRoom(s, bob, ScopeUser, LoungeRoom, "from bob") // to alice and carol
	p := CleanupPolicy{SessionIdle: 60 * day}

	clock.advance(60 * day)
	keepAlive(t, s, bob, carol)
	if r := cleanup(t, s, p); !r.Empty() {
		t.Errorf("at the period: %+v", r)
	}
	clock.advance(time.Millisecond)
	// alice: her room message (2 deliveries), the message to her (1), and her delivery of
	// bob's room message.
	if r := cleanup(t, s, p); r != (CleanupResult{Sessions: 1, SessionMessages: 2, SessionDeliveries: 4}) {
		t.Errorf("after the period: %+v", r)
	}
	if exists, err := sessionExists(ctx, s, alice); err != nil || exists {
		t.Errorf("alice exists = %v, %v", exists, err)
	}
	if exists, _ := sessionExists(ctx, s, bob); !exists {
		t.Error("bob was deleted")
	}
	if messageExists(t, s, fromAlice.MessageID) || messageExists(t, s, toAlice.MessageID) {
		t.Error("alice's messages remain")
	}
	if !messageExists(t, s, betweenOthers.MessageID) || !messageExists(t, s, roomByBob.MessageID) {
		t.Error("other messages were deleted")
	}
	if st := status(t, s, bob, roomByBob.MessageID); len(st.Recipients) != 1 || st.Recipients[0].Name != "carol" {
		t.Errorf("bob's room message: %+v", st)
	}
	// Joining again with the name creates a new session.
	again, resumed, err := s.JoinSession(ctx, JoinParams{UserID: local, Name: "alice"})
	if err != nil || resumed || unread(t, s, again) != 0 {
		t.Errorf("join again: %+v resumed=%v %v", again, resumed, err)
	}
}

// A zero period disables its step.
func TestCleanupZeroDisables(t *testing.T) {
	s, local := openLocal(t)
	clock := useClock(s)
	alice, bob := join(t, s, local, "alice"), join(t, s, local, "bob")
	q, _ := sendTo(s, alice, ScopeUser, LocalUser, "bob", "queued")
	d, _ := sendTo(s, bob, ScopeUser, LocalUser, "alice", "delivered")
	fetch(t, s, alice)
	clock.advance(1000 * day)
	if r := cleanup(t, s, CleanupPolicy{}); !r.Empty() {
		t.Errorf("all disabled: %+v", r)
	}
	if r := cleanup(t, s, CleanupPolicy{RetainFor: day}); r != (CleanupResult{Messages: 1, MessageDeliveries: 1}) {
		t.Errorf("retain only: %+v", r)
	}
	if !messageExists(t, s, q.MessageID) || messageExists(t, s, d.MessageID) {
		t.Error("wrong message deleted")
	}
	if state, _ := deliveryState(t, s, q.MessageID, bob); state != StateQueued {
		t.Errorf("state = %s", state)
	}
	if list, _, _ := s.ListSessions(context.Background(), local, 10); len(list) != 2 {
		t.Errorf("sessions = %d", len(list))
	}
	// The default policy runs the three steps in order: expire, then the expired message is
	// still kept (its last change is now), then the idle sessions go with it.
	if r := cleanup(t, s, defaultPolicy); r != (CleanupResult{Expired: 1, Sessions: 2, SessionMessages: 1, SessionDeliveries: 1}) {
		t.Errorf("default policy: %+v", r)
	}
}

// The cleanup queries use the indexes of schema version 4 instead of scanning tables.
func TestCleanupQueryPlans(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "lounge.db"))
	for query, index := range map[string]string{
		"UPDATE deliveries SET state = 'expired', expired_at = 1 WHERE state = 'queued' AND queued_at < 1": "deliveries_queued",
		"SELECT m.id FROM messages m WHERE m.sent_at < 1":                                                  "messages_sent_at",
		"SELECT id FROM sessions WHERE last_seen_at < 1":                                                   "sessions_last_seen",
	} {
		rows, err := s.db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query)
		if err != nil {
			t.Fatal(err)
		}
		var plan []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}
			plan = append(plan, detail)
		}
		rows.Close()
		if !slices.ContainsFunc(plan, func(d string) bool { return strings.Contains(d, index) }) {
			t.Errorf("%s: plan %v does not use %s", query, plan, index)
		}
	}
}

func TestCleanupCancelled(t *testing.T) {
	s, _ := openLocal(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Cleanup(ctx, defaultPolicy); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled cleanup: %v", err)
	}
}

// scopeActivityRow reads one memory_scope_activity row.
func scopeActivityRow(t *testing.T, s *Store, kind, id string) (last time.Time, discount int64, ok bool) {
	t.Helper()
	var lastMs int64
	err := s.db.QueryRowContext(context.Background(),
		"SELECT last_access, discount FROM memory_scope_activity WHERE scope = ? AND scope_id = ?",
		kind, id).Scan(&lastMs, &discount)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, 0, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return fromMillis(lastMs), discount, true
}

func repoSession(t *testing.T, s *Store, user int64, name, origin string) Session {
	t.Helper()
	return joinWith(t, s, JoinParams{UserID: user, Name: name, Location: &Location{Origin: origin}})
}

// searchNone searches sess in scope and requires no match. The search still records activity.
func searchNone(t *testing.T, s *Store, sess Session, scope string) {
	t.Helper()
	if ms := search(t, s, sess, MemorySearch{Scope: scope, Query: "zzznomatch", Limit: 10}); len(ms) != 0 {
		t.Fatalf("search = %d memories, want 0", len(ms))
	}
}

// A memory_scope_activity row created by a search that found nothing is kept until 60 days
// have passed since its last access, then deleted. Access after that creates the row again
// with discount 0. Exactly 60 days is kept: the row goes once the access is older than 60 days.
func TestIdleScopeActivity(t *testing.T) {
	if scopeActivityIdle != 60*day {
		t.Fatalf("scopeActivityIdle = %v, want 60 days", scopeActivityIdle)
	}
	s, local := openLocal(t)
	clock := useClock(s)
	empty := repoSession(t, s, local, "empty", "github.com/org/empty")
	searchNone(t, s, empty, ScopeRepository)
	if _, discount, ok := scopeActivityRow(t, s, ScopeRepository, empty.Repository); !ok || discount != 0 {
		t.Fatalf("search with no results: activity ok=%v discount=%d", ok, discount)
	}

	clock.advance(60*day - time.Millisecond)
	cleanup(t, s, CleanupPolicy{})
	if _, _, ok := scopeActivityRow(t, s, ScopeRepository, empty.Repository); !ok {
		t.Fatal("activity deleted just before 60 days")
	}

	clock.advance(time.Millisecond) // exactly 60 days
	cleanup(t, s, CleanupPolicy{})
	if _, _, ok := scopeActivityRow(t, s, ScopeRepository, empty.Repository); !ok {
		t.Fatal("activity deleted at exactly 60 days")
	}

	clock.advance(time.Millisecond) // older than 60 days
	cleanup(t, s, CleanupPolicy{})
	if _, _, ok := scopeActivityRow(t, s, ScopeRepository, empty.Repository); ok {
		t.Fatal("activity kept after 60 days")
	}

	searchNone(t, s, empty, ScopeRepository)
	last, discount, ok := scopeActivityRow(t, s, ScopeRepository, empty.Repository)
	if !ok || discount != 0 || !last.Equal(clock.now()) {
		t.Fatalf("re-access = %v discount %d ok %v, want %v discount 0", last, discount, ok, clock.now())
	}
	if n := memoryCount(t, s, "memories"); n != 0 {
		t.Errorf("%d memories, want a scope with none", n)
	}
}

// An activity row older than 60 days stays while its scope still has a memory. A pinned memory
// and a faded memory count.
func TestScopeActivityKeptWhileMemoryExists(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	clock := useClock(s)
	pinnedSess := repoSession(t, s, local, "pinned", "github.com/org/pinned")
	fadedSess := repoSession(t, s, local, "faded", "github.com/org/faded")
	pinned := createMemory(t, s, pinnedSess, ScopeRepository, "pinned note")
	if _, err := s.PinMemory(ctx, pinnedSess.ID, pinned.ID, true); err != nil {
		t.Fatal(err)
	}
	faded := createMemory(t, s, fadedSess, ScopeRepository, "fades later")

	clock.advance(60*day + time.Millisecond)
	cleanup(t, s, CleanupPolicy{})

	got, err := s.MemoryByID(ctx, pinned.ID)
	if err != nil || !got.Pinned || got.Faded {
		t.Fatalf("pinned memory = %+v, %v", got, err)
	}
	if _, _, ok := scopeActivityRow(t, s, pinned.Scope.Kind, pinned.Scope.ID); !ok {
		t.Error("activity of a scope with a pinned memory was deleted")
	}
	got, err = s.MemoryByID(ctx, faded.ID)
	if err != nil || got.Pinned || !got.Faded {
		t.Fatalf("faded memory = %+v, %v", got, err)
	}
	if _, _, ok := scopeActivityRow(t, s, faded.Scope.Kind, faded.Scope.ID); !ok {
		t.Error("activity of a scope with a faded memory was deleted")
	}
}

// Forgetting disabled (half-life 0) and a grace of 0 still delete an activity row that has no
// memory and is older than 60 days. A scope that still has a memory keeps its row.
func TestIdleScopeActivityIgnoresRetention(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    RetentionPolicy
	}{
		{"half-life 0", RetentionPolicy{HalfLife: 0, Threshold: 0.1, Grace: 30 * day}},
		{"grace 0", RetentionPolicy{HalfLife: 7 * day, Threshold: 0.1, Grace: 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, local := openLocal(t)
			s.SetRetention(tc.p)
			clock := useClock(s)
			empty := repoSession(t, s, local, "empty", "github.com/org/empty")
			kept := repoSession(t, s, local, "kept", "github.com/org/kept")
			searchNone(t, s, empty, ScopeRepository)
			mem := createMemory(t, s, kept, ScopeRepository, "stays")

			clock.advance(60*day + time.Millisecond)
			cleanup(t, s, CleanupPolicy{})

			if _, _, ok := scopeActivityRow(t, s, ScopeRepository, empty.Repository); ok {
				t.Error("activity with no memory was kept")
			}
			if _, err := s.MemoryByID(ctx, mem.ID); err != nil {
				t.Fatalf("memory: %v", err)
			}
			if _, _, ok := scopeActivityRow(t, s, mem.Scope.Kind, mem.Scope.ID); !ok {
				t.Error("activity of a scope that still has a memory was deleted")
			}
		})
	}
}
