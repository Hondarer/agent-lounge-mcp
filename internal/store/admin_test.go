package store

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestAllSessionsAndRemoveSession(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	ab := addUser(t, s, "a.b")
	a := addUser(t, s, "a")
	alice, bob := join(t, s, local, "alice"), join(t, s, local, "bob")
	join(t, s, a.ID, "x")
	join(t, s, ab.ID, "x")
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "1"); err != nil {
		t.Fatal(err)
	}
	if _, err := sendTo(s, bob, ScopeUser, LocalUser, "alice", "2"); err != nil {
		t.Fatal(err)
	}
	if _, err := sendRoom(s, alice, ScopeUser, LoungeRoom, "3"); err != nil {
		t.Fatal(err)
	}

	list, err := s.AllSessions(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, sum := range list {
		got = append(got, sum.UserName+"/"+sum.Name)
	}
	// Address order: "a.b/x" sorts before "a/x" ('.' < '/').
	if !slices.Equal(got, []string{"a.b/x", "a/x", "local/alice", "local/bob"}) {
		t.Errorf("sessions = %v", got)
	}
	if list[2].Unread != 1 || list[3].Unread != 2 {
		t.Errorf("unread = %d, %d", list[2].Unread, list[3].Unread)
	}
	if list, err := s.AllSessions(ctx, LocalUser); err != nil || len(list) != 2 {
		t.Errorf("local sessions: %d %v", len(list), err)
	}
	if _, err := s.AllSessions(ctx, "nobody"); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("unknown user: %v", err)
	}

	// alice sent 2 messages and received 1, each with one delivery.
	data, err := s.CountSessionData(ctx, LocalUser, "alice")
	if err != nil || data != (SessionData{Messages: 3, Deliveries: 3}) {
		t.Errorf("count: %+v %v", data, err)
	}
	if _, err := s.CountSessionData(ctx, LocalUser, "nobody"); !errors.Is(err, ErrNoSessionToRemove) {
		t.Errorf("count of a missing session: %v", err)
	}
	if _, err := s.RemoveSession(ctx, "a", "alice"); !errors.Is(err, ErrNoSessionToRemove) {
		t.Errorf("remove of a missing session: %v", err)
	}
	data, err = s.RemoveSession(ctx, LocalUser, "alice")
	if err != nil || data != (SessionData{Messages: 3, Deliveries: 3}) {
		t.Errorf("remove: %+v %v", data, err)
	}
	if exists, _ := sessionExists(ctx, s, alice); exists {
		t.Error("alice still exists")
	}
	if n := unread(t, s, bob); n != 0 {
		t.Errorf("bob unread = %d", n)
	}
}

func TestExport(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	useClock(s)
	repo := &Location{Origin: "github.com/org/a"}
	alice := joinWith(t, s, JoinParams{UserID: local, Name: "alice", Agent: ptr("claude"), Location: repo})
	bob := join(t, s, local, "bob")
	zed := addUser(t, s, "zed")
	z := joinWith(t, s, JoinParams{UserID: zed.ID, Name: "z", Location: repo})
	must := func(_ Sent, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(sendTo(s, bob, ScopeUser, LocalUser, "alice", "b->a"))
	must(sendRoom(s, alice, ScopeUser, LoungeRoom, "user room"))
	must(sendTo(s, alice, ScopeUser, LocalUser, "bob", "a->b"))
	must(sendRoom(s, z, ScopeRepository, LoungeRoom, "repo room"))
	must(sendTo(s, alice, ScopeRepository, "zed", "z", "a->z"))
	fetch(t, s, alice)

	var got []string
	var first ExportMessage
	err := s.Export(ctx, func(m ExportMessage) error {
		if m.Body == "repo room" {
			first = m
		}
		got = append(got, m.Scope.Kind+"|"+m.Scope.ID+"|"+m.Room+"|"+m.ToUser+"/"+m.ToName+"|"+m.Body)
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"repository|github.com/org/a|lounge|/|repo room",
		"repository|github.com/org/a||zed/z|a->z",
		"user|local|lounge|/|user room",
		"user|local||local/alice|b->a",
		"user|local||local/bob|a->b",
	}
	if !slices.Equal(got, want) {
		t.Errorf("export order:\n%v\nwant\n%v", got, want)
	}
	if first.From != "z" || first.FromUser != "zed" || len(first.Deliveries) != 1 || first.Deliveries[0].Name != "alice" ||
		first.Deliveries[0].State != StateDelivered || first.Deliveries[0].DeliveredAt.IsZero() || first.Deliveries[0].QueuedAt.IsZero() ||
		!first.Deliveries[0].AckedAt.IsZero() {
		t.Errorf("repo room message: %+v", first)
	}

	stop := errors.New("stop")
	if err := s.Export(ctx, func(ExportMessage) error { return stop }, nil); !errors.Is(err, stop) {
		t.Errorf("export with a failing callback: %v", err)
	}
}

// A database of schema version 3 with data migrates to the latest version: the data stays, and
// every message gets its sender's current values.
func TestMigrateVersion3KeepsData(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lounge.db")
	s, err := openUnmigrated(path)
	if err != nil {
		t.Fatal(err)
	}
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 4 {
		t.Fatal("no version 4 script")
	}
	if err := s.applyMigrations(ctx, ms[:3]); err != nil {
		t.Fatal(err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, name, token_hash, created_at) VALUES (1, 'local', 'none:local', 10), (2, 'zed', 'h', 11);
		INSERT INTO rooms (id, scope, user_id, repository, name, created_at) VALUES (2, 'user', 1, '', 'lounge', 11);
		INSERT INTO sessions (id, user_id, name, agent, os_user, machine, origin, folder, repository, created_at, last_seen_at) VALUES
			(1, 1, 'alice', 'claude', 'honda', 'host-a', '', '', '', 12, 20),
			(2, 1, 'bob', 'codex', '', 'host-b', '', '', '', 13, 21);
		INSERT INTO messages (id, scope, scope_id, sender_session_id, to_session_id, room_id, body, sent_at) VALUES
			(1, 'user', 'local', 1, 2, NULL, 'direct', 30), (2, 'user', 'local', 2, NULL, 2, 'room', 31);
		INSERT INTO deliveries (message_id, recipient_session_id, state, queued_at, delivered_at, acked_at) VALUES
			(1, 2, 'acked', 30, 32, 33), (2, 1, 'queued', 31, NULL, NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	dump := func() string {
		t.Helper()
		return dumpRows(t, s, []string{
			"SELECT id, name, token_hash, created_at FROM users ORDER BY id",
			"SELECT id, user_id, name, agent, created_at, last_seen_at, os_user, machine, origin, folder, repository FROM sessions ORDER BY id",
			"SELECT * FROM rooms ORDER BY id",
			"SELECT id, scope, scope_id, sender_session_id, coalesce(to_session_id, 0), coalesce(room_id, 0), body, sent_at FROM messages ORDER BY id",
			"SELECT * FROM deliveries ORDER BY message_id",
		})
	}
	before := dump()
	s.Close()

	s = open(t, path)
	if v, _ := s.SchemaVersion(ctx); v != len(ms) {
		t.Fatalf("schema version after migration = %d, want %d", v, len(ms))
	}
	if after := dump(); after != before {
		t.Errorf("data changed by the migration:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if got := dumpRows(t, s, []string{"SELECT sender_agent, sender_os_user, sender_machine FROM messages ORDER BY id"}); got != "claude|honda|host-a|\ncodex||host-b|\n" {
		t.Errorf("sender values = %q", got)
	}
	// The migrated data works: alice reads the queued room message with bob's values, and
	// bob's history shows the acked message.
	s.SetClock(func() time.Time { return time.UnixMilli(40) })
	msgs, err := s.FetchInbox(ctx, 1, 10)
	if err != nil || len(msgs) != 1 || msgs[0].FromAgent != "codex" || msgs[0].FromMachine != "host-b" {
		t.Errorf("alice's inbox = %+v, %v", msgs, err)
	}
	h, _, err := s.History(ctx, 2, HistoryQuery{Limit: 10})
	if err != nil || len(h) != 2 || h[1].State != StateAcked || h[1].FromAgent != "claude" {
		t.Errorf("bob's history = %+v, %v", h, err)
	}
}

// dumpRows renders the rows of the queries, one line per row with values ended by '|'.
func dumpRows(t *testing.T, s *Store, queries []string) string {
	t.Helper()
	var b []byte
	for _, q := range queries {
		rows, err := s.db.QueryContext(context.Background(), q)
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			for _, v := range vals {
				b = append(b, sqlString(v)+"|"...)
			}
			b = append(b, '\n')
		}
		rows.Close()
	}
	return string(b)
}
