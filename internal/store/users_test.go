package store

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// addUser adds a user with a fake token hash and returns it.
func addUser(t *testing.T, s *Store, name string) User {
	t.Helper()
	u, err := s.AddUser(context.Background(), name, "hash-"+name, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestUserLifecycle(t *testing.T) {
	ctx := context.Background()
	s, _ := openLocal(t)

	alice, err := s.AddUser(ctx, "alice", "hash-1", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if alice.Name != "alice" || alice.ID == 0 {
		t.Fatalf("added %+v", alice)
	}
	// Only the hash is stored, and it finds the user.
	var stored string
	if err := s.db.QueryRowContext(ctx, "SELECT token_hash FROM users WHERE name = 'alice'").Scan(&stored); err != nil || stored != "hash-1" {
		t.Fatalf("stored hash = %q, %v", stored, err)
	}
	if u, err := s.UserByTokenHash(ctx, "hash-1"); err != nil || u.ID != alice.ID || u.Name != "alice" || !u.CreatedAt.Equal(alice.CreatedAt) {
		t.Errorf("lookup = %+v, %v", u, err)
	}
	if _, err := s.UserByTokenHash(ctx, "none:local"); err == nil {
		t.Error("the local user's marker was taken as a token hash")
	}
	// The user's space has its lounge room.
	if rooms, err := s.ListRooms(ctx, alice.ID); err != nil || !slices.Equal(rooms, []string{LoungeRoom}) {
		t.Errorf("rooms = %v, %v", rooms, err)
	}

	// Names: taken, reserved, invalid.
	if _, err := s.AddUser(ctx, "alice", "hash-2", time.Time{}); !errors.Is(err, ErrUserExists) {
		t.Errorf("duplicate name: %v", err)
	}
	if _, err := s.AddUser(ctx, LocalUser, "hash-3", time.Time{}); !errors.Is(err, ErrReservedUser) {
		t.Errorf("local: %v", err)
	}
	for _, name := range []string{"", "Alice", "-a", ".a", "a/b", "a b", strings.Repeat("a", 33), "日本"} {
		if _, err := s.AddUser(ctx, name, "h", time.Time{}); err == nil || !strings.Contains(err.Error(), "invalid user name") {
			t.Errorf("name %q: %v", name, err)
		}
	}
	for _, name := range []string{"a", "0", "a.b_c-d", strings.Repeat("z", 32)} {
		if err := ValidateUserName(name); err != nil {
			t.Errorf("name %q: %v", name, err)
		}
	}
	if n, err := s.TokenUserCount(ctx); err != nil || n != 1 {
		t.Errorf("token users = %d, %v", n, err)
	}

	// Rotate: the old hash no longer finds the user; the data stays.
	work := join(t, s, alice.ID, "work")
	if err := s.RotateToken(ctx, "alice", "hash-new", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserByTokenHash(ctx, "hash-1"); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("old hash after rotate: %v", err)
	}
	if u, err := s.UserByTokenHash(ctx, "hash-new"); err != nil || u.ID != alice.ID {
		t.Errorf("new hash: %+v, %v", u, err)
	}
	if sessions, _, err := s.ListSessions(ctx, alice.ID, 10); err != nil || len(sessions) != 1 || sessions[0].ID != work.ID {
		t.Errorf("sessions after rotate: %+v, %v", sessions, err)
	}
	for _, name := range []string{"nobody", LocalUser} {
		if err := s.RotateToken(ctx, name, "x", time.Time{}); err == nil {
			t.Errorf("rotate %s was accepted", name)
		}
	}

	// List: name order, sessions and last use.
	users, err := s.ListUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Name != "alice" || users[0].Sessions != 1 || users[0].LastUsedAt.IsZero() ||
		users[1].Name != LocalUser || users[1].Sessions != 0 || !users[1].LastUsedAt.IsZero() {
		t.Errorf("users = %+v", users)
	}
}

func TestRemoveUser(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	bob := addUser(t, s, "bob")
	repo := &Location{Origin: "github.com/org/a"}
	b1 := joinWith(t, s, JoinParams{UserID: bob.ID, Name: "b1", Location: repo})
	b2 := join(t, s, bob.ID, "b2")
	l1 := joinWith(t, s, JoinParams{UserID: local, Name: "l1", Location: repo})

	// bob sends 2 messages (3 deliveries), and receives 1 direct message from l1.
	if _, err := sendRoom(s, b1, ScopeRepository, LoungeRoom, "to the repository"); err != nil { // to l1
		t.Fatal(err)
	}
	if _, err := sendRoom(s, b1, ScopeUser, LoungeRoom, "to bob's sessions"); err != nil { // to b2
		t.Fatal(err)
	}
	if _, err := sendTo(s, l1, ScopeRepository, "bob", "b1", "to b1"); err != nil {
		t.Fatal(err)
	}
	if _, err := sendTo(s, l1, ScopeRepository, "bob", "b2", "x"); !errors.Is(err, ErrNoSuchSession) {
		t.Fatalf("b2 is outside the repository: %v", err)
	}
	want := UserData{Sessions: 2, Messages: 3, Deliveries: 3}
	if got, err := s.CountUserData(ctx, "bob"); err != nil || got != want {
		t.Errorf("count = %+v, %v; want %+v", got, err, want)
	}
	got, err := s.RemoveUser(ctx, "bob")
	if err != nil || got != want {
		t.Errorf("removed = %+v, %v; want %+v", got, err, want)
	}
	var rows int
	if err := s.db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM users WHERE name = 'bob') + (SELECT count(*) FROM sessions WHERE user_id = ?)
		+ (SELECT count(*) FROM messages) + (SELECT count(*) FROM deliveries) + (SELECT count(*) FROM rooms WHERE user_id = ?)`,
		bob.ID, bob.ID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Errorf("%d rows of bob remain", rows)
	}
	if _, err := s.UserByTokenHash(ctx, "hash-bob"); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("token of a removed user: %v", err)
	}
	_ = b2
	for _, name := range []string{"bob", LocalUser} {
		if _, err := s.RemoveUser(ctx, name); err == nil {
			t.Errorf("remove %s was accepted", name)
		}
	}
	// local and its session are untouched.
	if sessions, _, err := s.ListSessions(ctx, local, 10); err != nil || len(sessions) != 1 {
		t.Errorf("local sessions = %+v, %v", sessions, err)
	}
}

// Session names are unique per user: two users may have sessions of the same name.
func TestSameSessionNameForTwoUsers(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	alice := addUser(t, s, "alice")
	a, resumed, err := s.JoinSession(ctx, JoinParams{UserID: alice.ID, Name: "claude"})
	if err != nil || resumed || a.UserName != "alice" {
		t.Fatalf("alice: %+v %v %v", a, resumed, err)
	}
	l, resumed, err := s.JoinSession(ctx, JoinParams{UserID: local, Name: "claude"})
	if err != nil || resumed || l.ID == a.ID || l.UserName != LocalUser {
		t.Fatalf("local: %+v %v %v", l, resumed, err)
	}
	// Joining again continues the session of the same user only.
	again, resumed, err := s.JoinSession(ctx, JoinParams{UserID: alice.ID, Name: "claude"})
	if err != nil || !resumed || again.ID != a.ID {
		t.Errorf("alice again: %+v %v %v", again, resumed, err)
	}
}

// The four combinations of scope and target, with two users.
func TestScopesAndTargets(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	bob := addUser(t, s, "bob")
	repoA, repoB := &Location{Origin: "github.com/org/a"}, &Location{Origin: "github.com/org/b"}
	l1 := joinWith(t, s, JoinParams{UserID: local, Name: "l1", Location: repoA})
	l2 := joinWith(t, s, JoinParams{UserID: local, Name: "l2", Location: repoB})
	b1 := joinWith(t, s, JoinParams{UserID: bob.ID, Name: "l1", Location: repoA}) // same name as local's l1
	b2 := joinWith(t, s, JoinParams{UserID: bob.ID, Name: "b2", Location: repoB})
	b3 := join(t, s, bob.ID, "b3") // no repository

	// user + room: the other sessions of the sender's user.
	sent, err := sendRoom(s, l1, ScopeUser, LoungeRoom, "user room")
	if err != nil || !slices.Equal(ids(sent.Recipients), []int64{l2.ID}) || sent.Scope != (Scope{ScopeUser, LocalUser}) {
		t.Errorf("user room: %+v %v", sent, err)
	}
	// user + to: a session of the sender's user only.
	if sent, err = sendTo(s, l1, ScopeUser, LocalUser, "l2", "user direct"); err != nil || sent.Recipients[0].ID != l2.ID {
		t.Errorf("user direct: %+v %v", sent, err)
	}
	if _, err := sendTo(s, l1, ScopeUser, "bob", "b2", "x"); !errors.Is(err, ErrNoSuchSession) {
		t.Errorf("user scope to another user: %v", err)
	}
	// repository + room: the sessions of the unit, of any user.
	sent, err = sendRoom(s, l1, ScopeRepository, LoungeRoom, "repo room")
	if err != nil || !slices.Equal(ids(sent.Recipients), []int64{b1.ID}) || sent.Recipients[0].User != "bob" ||
		sent.Scope != (Scope{ScopeRepository, "github.com/org/a"}) {
		t.Errorf("repository room: %+v %v", sent, err)
	}
	// repository + to: a session of the unit, of any user.
	sent, err = sendTo(s, l1, ScopeRepository, "bob", "l1", "repo direct")
	if err != nil || sent.Recipients[0].ID != b1.ID || sent.Scope.ID != "github.com/org/a" {
		t.Errorf("repository direct: %+v %v", sent, err)
	}
	// Outside the unit and missing: the same error.
	for _, target := range [][2]string{{"bob", "b2"}, {"bob", "b3"}, {"bob", "nobody"}, {"nobody", "l1"}, {LocalUser, "l2"}} {
		if _, err := sendTo(s, l1, ScopeRepository, target[0], target[1], "x"); !errors.Is(err, ErrNoSuchSession) {
			t.Errorf("repository to %v: %v", target, err)
		}
	}
	if _, err := sendTo(s, b3, ScopeRepository, "bob", "b2", "x"); !errors.Is(err, ErrNoRepository) {
		t.Errorf("sender without repository: %v", err)
	}
	if _, err := sendTo(s, l1, ScopeRepository, LocalUser, "l1", "x"); !errors.Is(err, ErrSendToSelf) {
		t.Errorf("to self: %v", err)
	}
	if _, err := sendRoom(s, l1, "shared", LoungeRoom, "x"); !errors.Is(err, ErrInvalidScope) {
		t.Errorf("unknown scope: %v", err)
	}

	// The inbox of bob's l1 has the room and the direct message, with scope, id and sender's user.
	msgs, err := s.FetchInbox(ctx, b1.ID, 10)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("b1 inbox: %+v %v", msgs, err)
	}
	for i, room := range []string{LoungeRoom, ""} {
		m := msgs[i]
		if m.FromUser != LocalUser || m.From != "l1" || m.Room != room || m.Scope != (Scope{ScopeRepository, "github.com/org/a"}) {
			t.Errorf("message %d = %+v", i, m)
		}
	}
	// Every message records its scope id.
	var empty int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM messages WHERE scope_id = ''").Scan(&empty); err != nil || empty != 0 {
		t.Errorf("%d messages without scope id (%v)", empty, err)
	}

	// VisibleSessions: the viewer's user, and other users' sessions of the viewer's unit.
	viewer, sessions, _, err := s.VisibleSessions(ctx, l1.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64][]Scope{}
	for _, sess := range sessions {
		got[sess.ID] = SharedScopes(viewer, sess)
	}
	want := map[int64][]Scope{
		l1.ID: {{ScopeUser, LocalUser}, {ScopeRepository, "github.com/org/a"}},
		l2.ID: {{ScopeUser, LocalUser}},
		b1.ID: {{ScopeRepository, "github.com/org/a"}},
	}
	if len(got) != len(want) {
		t.Errorf("visible = %v, want %v", got, want)
	}
	for id, w := range want {
		if !slices.Equal(got[id], w) {
			t.Errorf("session %d: scopes %v, want %v", id, got[id], w)
		}
	}
	if !slices.Equal(ScopesOf(viewer), []Scope{{ScopeUser, LocalUser}, {ScopeRepository, "github.com/org/a"}}) {
		t.Errorf("viewer scopes = %v", ScopesOf(viewer))
	}
	// A session without a repository sees its own user only.
	_, sessions, _, err = s.VisibleSessions(ctx, b3.ID, 100)
	if err != nil || len(sessions) != 3 {
		t.Errorf("b3 sees %+v, %v", sessions, err)
	}
	_ = b2
}

func TestLimits(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	alice, bob, carol := join(t, s, local, "alice"), join(t, s, local, "bob"), join(t, s, local, "carol")

	// Body size.
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", strings.Repeat("a", DefaultMaxBodyBytes)); err != nil {
		t.Errorf("body of the limit: %v", err)
	}
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", strings.Repeat("a", DefaultMaxBodyBytes+1)); !errors.Is(err, ErrBodyTooLarge) {
		t.Errorf("body over the limit: %v", err)
	}

	// Unread: fill bob's inbox to the limit.
	if _, err := s.db.ExecContext(ctx, `
		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
		INSERT INTO messages (scope, scope_id, sender_session_id, to_session_id, body, sent_at)
		SELECT 'user', 'local', ?, ?, 'fill', 1 FROM n`, DefaultMaxUnread-1, alice.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO deliveries (message_id, recipient_session_id, queued_at)
		SELECT id, ?, 1 FROM messages WHERE to_session_id = ? AND body = 'fill'`, bob.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if n := unread(t, s, bob); n != DefaultMaxUnread {
		t.Fatalf("bob unread = %d, want %d", n, DefaultMaxUnread)
	}
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "x"); !errors.Is(err, ErrInboxFull) {
		t.Errorf("direct to a full inbox: %v", err)
	}
	sent, err := sendRoom(s, alice, ScopeUser, LoungeRoom, "room")
	if err != nil || !slices.Equal(names(sent.Recipients), []string{"carol"}) || !slices.Equal(names(sent.Skipped), []string{"bob"}) {
		t.Errorf("room with a full inbox: %+v %v", sent, err)
	}
	if n := unread(t, s, bob); n != DefaultMaxUnread {
		t.Errorf("bob unread after the room message = %d", n)
	}
	// Reading makes room again.
	if _, err := s.FetchInbox(ctx, bob.ID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := sendTo(s, alice, ScopeUser, LocalUser, "bob", "x"); err != nil {
		t.Errorf("after reading one: %v", err)
	}
	_ = carol

	// Sessions per user: 3 exist; fill up to the limit.
	if _, err := s.db.ExecContext(ctx, `
		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
		INSERT INTO sessions (user_id, name, created_at, last_seen_at) SELECT ?, 'fill-' || i, 1, 1 FROM n`,
		DefaultMaxSessionsPerUser-3, local); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.JoinSession(ctx, JoinParams{UserID: local, Name: "one-more"}); !errors.Is(err, ErrTooManySessions) {
		t.Errorf("new name over the limit: %v", err)
	}
	if _, _, err := s.JoinSession(ctx, JoinParams{UserID: local}); !errors.Is(err, ErrTooManySessions) {
		t.Errorf("generated name over the limit: %v", err)
	}
	if _, resumed, err := s.JoinSession(ctx, JoinParams{UserID: local, Name: "alice"}); err != nil || !resumed {
		t.Errorf("existing name at the limit: %v", err)
	}
	// The limit is per user.
	other := addUser(t, s, "other")
	if _, _, err := s.JoinSession(ctx, JoinParams{UserID: other.ID, Name: "one-more"}); err != nil {
		t.Errorf("another user: %v", err)
	}
}

// A database of schema version 2 with data migrates to the latest version: the data stays, with
// the user local, and every message gets its scope id.
func TestMigrateVersion2KeepsData(t *testing.T) {
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
	if len(ms) < 3 {
		t.Fatal("no version 3 script")
	}
	if err := s.applyMigrations(ctx, ms[:2]); err != nil {
		t.Fatal(err)
	}
	// Data as version 2 writes it: the local user with sessions in one repository, a direct
	// message, a message to the user's room and one to the repository's room.
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, name, token_hash, created_at) VALUES (1, 'local', 'none:local', 10);
		INSERT INTO rooms (id, scope, user_id, repository, name, created_at) VALUES
			(2, 'user', 1, '', 'lounge', 11), (3, 'repository', NULL, 'github.com/org/a', 'lounge', 12);
		INSERT INTO sessions (id, user_id, name, agent, os_user, machine, origin, folder, repository, created_at, last_seen_at) VALUES
			(1, 1, 'alice', 'claude', 'honda', 'host-a', 'github.com/org/a', 'c:\work\a', 'github.com/org/a', 12, 20),
			(2, 1, 'bob', 'codex', '', '', 'github.com/org/a', '', 'github.com/org/a', 13, 21);
		INSERT INTO messages (id, scope, sender_session_id, to_session_id, room_id, body, sent_at) VALUES
			(1, 'user', 1, 2, NULL, 'direct', 30), (2, 'user', 2, NULL, 2, 'room', 31), (3, 'repository', 1, NULL, 3, 'repo', 32);
		INSERT INTO deliveries (message_id, recipient_session_id, state, queued_at, delivered_at) VALUES
			(1, 2, 'queued', 30, NULL), (2, 1, 'delivered', 31, 33), (3, 2, 'queued', 32, NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	dump := func() string {
		t.Helper()
		var b strings.Builder
		for _, q := range []string{
			"SELECT id, name, token_hash, created_at FROM users ORDER BY id",
			"SELECT * FROM sessions ORDER BY id",
			"SELECT * FROM rooms ORDER BY id",
			"SELECT id, scope, sender_session_id, coalesce(to_session_id, 0), coalesce(room_id, 0), body, sent_at FROM messages ORDER BY id",
			"SELECT * FROM deliveries ORDER BY message_id",
		} {
			rows, err := s.db.QueryContext(ctx, q)
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
					b.WriteString(sqlString(v) + "|")
				}
				b.WriteString("\n")
			}
			rows.Close()
		}
		return b.String()
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
	var ids []string
	rows, err := s.db.QueryContext(ctx, "SELECT scope_id FROM messages ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if !slices.Equal(ids, []string{"local", "local", "github.com/org/a"}) {
		t.Errorf("scope ids = %v", ids)
	}

	// The data stays the local user's and works.
	local, err := s.EnsureLocalUser(ctx)
	if err != nil || local.ID != 1 {
		t.Fatalf("local = %+v, %v", local, err)
	}
	msgs, err := s.FetchInbox(ctx, 2, 10)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("bob's inbox = %+v, %v", msgs, err)
	}
	if msgs[0].Scope != (Scope{ScopeUser, LocalUser}) || msgs[0].FromUser != LocalUser || msgs[0].Room != "" ||
		msgs[1].Scope != (Scope{ScopeRepository, "github.com/org/a"}) || msgs[1].Room != LoungeRoom {
		t.Errorf("bob's inbox = %+v", msgs)
	}
	// A new user shares the migrated repository room.
	carol := addUser(t, s, "carol")
	c := joinWith(t, s, JoinParams{UserID: carol.ID, Name: "c", Location: &Location{Origin: "github.com/org/a"}})
	sent, err := sendRoom(s, c, ScopeRepository, LoungeRoom, "hello")
	if err != nil || !slices.Equal(names(sent.Recipients), []string{"alice", "bob"}) {
		t.Errorf("send after migration: %+v %v", sent, err)
	}
	var rooms int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM rooms WHERE scope = 'repository'").Scan(&rooms); err != nil || rooms != 1 {
		t.Errorf("repository rooms = %d, %v", rooms, err)
	}
	if rows, err := s.db.QueryContext(ctx, "PRAGMA foreign_key_check"); err != nil {
		t.Fatal(err)
	} else {
		if rows.Next() {
			t.Error("foreign key violations after the migration")
		}
		rows.Close()
	}
}
