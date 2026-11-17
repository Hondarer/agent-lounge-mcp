package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

// joinWith joins with profile values; the values are taken as already normalized.
func joinWith(t *testing.T, s *Store, p JoinParams) Session {
	t.Helper()
	sess, _, err := s.JoinSession(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestJoinSessionProfile(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)

	first := joinWith(t, s, JoinParams{UserID: user, Name: "work", OSUser: ptr("honda"), Machine: ptr("host-a"),
		Location: &Location{Origin: "github.com/owner/repo", Folder: `c:\work\repo`}})
	want := Profile{OSUser: "honda", Machine: "host-a", Origin: "github.com/owner/repo", Folder: `c:\work\repo`, Repository: "github.com/owner/repo"}
	if first.Profile != want {
		t.Fatalf("first join profile = %+v, want %+v", first.Profile, want)
	}

	// Omitted values are kept.
	again := joinWith(t, s, JoinParams{UserID: user, Name: "work"})
	if again.Profile != want {
		t.Errorf("join without values: %+v, want %+v", again.Profile, want)
	}

	// The location is replaced as one set: a folder without an origin replaces both, so the old
	// origin does not remain, and the unit becomes machine:folder.
	moved := joinWith(t, s, JoinParams{UserID: user, Name: "work", Location: &Location{Folder: `c:\scratch`}})
	if moved.Origin != "" || moved.Folder != `c:\scratch` || moved.Repository != `host-a:c:\scratch` || moved.OSUser != "honda" {
		t.Errorf("after moving: %+v", moved.Profile)
	}

	// Without an origin, a folder needs a machine: clearing the machine fails and changes nothing.
	if _, _, err := s.JoinSession(ctx, JoinParams{UserID: user, Name: "work", Machine: ptr("")}); !errors.Is(err, ErrMachineRequired) {
		t.Fatalf("clearing the machine of a folder without origin: err = %v", err)
	}
	if _, _, err := s.JoinSession(ctx, JoinParams{UserID: user, Name: "new", Location: &Location{Folder: "/x"}}); !errors.Is(err, ErrMachineRequired) {
		t.Fatalf("new session with a folder and no machine: err = %v", err)
	}
	sessions, _, err := s.ListSessions(ctx, user, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Machine != "host-a" || sessions[0].Repository != `host-a:c:\scratch` {
		t.Errorf("a failed join changed the data: %+v", sessions)
	}

	// Empty values clear.
	cleared := joinWith(t, s, JoinParams{UserID: user, Name: "work", OSUser: ptr(""), Machine: ptr(""), Location: &Location{}})
	if cleared.Profile != (Profile{}) {
		t.Errorf("after clearing: %+v", cleared.Profile)
	}
}

func TestSendToRepositoryRoom(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	// A second user: the repository space crosses users.
	if _, err := s.db.ExecContext(ctx, "INSERT INTO users (name, token_hash, created_at) VALUES ('other', 'h', 1)"); err != nil {
		t.Fatal(err)
	}
	var other int64
	if err := s.db.QueryRowContext(ctx, "SELECT id FROM users WHERE name = 'other'").Scan(&other); err != nil {
		t.Fatal(err)
	}
	repoA := &Location{Origin: "github.com/org/a"}
	alice := joinWith(t, s, JoinParams{UserID: user, Name: "alice", OSUser: ptr("honda"), Machine: ptr("host-a"), Location: repoA})
	bob := joinWith(t, s, JoinParams{UserID: user, Name: "bob", Location: repoA})
	carol := joinWith(t, s, JoinParams{UserID: other, Name: "carol", Location: repoA})
	dave := joinWith(t, s, JoinParams{UserID: user, Name: "dave", Location: &Location{Origin: "github.com/org/b"}})
	erin := join(t, s, user, "erin") // no repository

	// The room does not exist before the first message.
	var rooms int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM rooms WHERE scope = 'repository'").Scan(&rooms); err != nil {
		t.Fatal(err)
	}
	if rooms != 0 {
		t.Fatalf("repository rooms before sending = %d", rooms)
	}

	sent, err := s.SendToRepositoryRoom(ctx, alice, LoungeRoom, "repo hello")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sent.Recipients, []string{"bob", "carol"}) || sent.Scope != ScopeRepository || sent.Repository != "github.com/org/a" {
		t.Errorf("sent = %+v", sent)
	}
	if !slices.Equal(sent.RecipientIDs, []int64{bob.ID, carol.ID}) {
		t.Errorf("recipient IDs = %v", sent.RecipientIDs)
	}
	for _, sess := range []Session{dave, erin, alice} {
		if n := unread(t, s, sess); n != 0 {
			t.Errorf("%s unread = %d, want 0", sess.Name, n)
		}
	}
	msgs, err := s.FetchInbox(ctx, carol.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("carol got %+v", msgs)
	}
	m := msgs[0]
	if m.Scope != ScopeRepository || m.Repository != "github.com/org/a" || m.Room != LoungeRoom || m.From != "alice" ||
		m.FromOSUser != "honda" || m.FromMachine != "host-a" || m.Body != "repo hello" {
		t.Errorf("carol's message = %+v", m)
	}

	// The room is created once per unit.
	if _, err := s.SendToRepositoryRoom(ctx, bob, LoungeRoom, "again"); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM rooms WHERE scope = 'repository'").Scan(&rooms); err != nil {
		t.Fatal(err)
	}
	if rooms != 1 {
		t.Errorf("repository rooms = %d, want 1", rooms)
	}

	// The sender's unit is read at sending: after bob moves to b, his message reaches dave only.
	joinWith(t, s, JoinParams{UserID: user, Name: "bob", Location: &Location{Origin: "github.com/org/b"}})
	sent, err = s.SendToRepositoryRoom(ctx, bob, LoungeRoom, "now in b")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sent.Recipients, []string{"dave"}) || sent.Repository != "github.com/org/b" {
		t.Errorf("after moving: %+v", sent)
	}
	// What bob received before moving stays in his inbox.
	if n := unread(t, s, bob); n != 1 {
		t.Errorf("bob unread = %d, want 1 (received before moving)", n)
	}

	if _, err := s.SendToRepositoryRoom(ctx, erin, LoungeRoom, "x"); !errors.Is(err, ErrNoRepository) {
		t.Errorf("sender without repository: err = %v", err)
	}
	if _, err := s.SendToRepositoryRoom(ctx, alice, "other", "x"); !errors.Is(err, ErrNoSuchRoom) {
		t.Errorf("unknown room: err = %v", err)
	}

	// The user space is unchanged: the lounge of the user reaches the user's sessions only.
	sent, err = s.SendToRoom(ctx, alice, LoungeRoom, "user hello")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sent.Recipients, []string{"bob", "dave", "erin"}) || sent.Scope != ScopeUser || sent.Repository != "" {
		t.Errorf("user room: %+v", sent)
	}
}

// A database of schema version 1 with data migrates to the latest version with its sessions,
// rooms, messages and delivery states kept, and with its constraints and foreign keys working.
func TestMigrateVersion1KeepsData(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lounge.db")

	// Build a version 1 database with the first script only.
	s, err := openUnmigrated(path)
	if err != nil {
		t.Fatal(err)
	}
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 2 {
		t.Fatal("no version 2 script")
	}
	if err := s.applyMigrations(ctx, ms[:1]); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.SchemaVersion(ctx); v != 1 {
		t.Fatalf("schema version = %d, want 1", v)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, name, token_hash, created_at) VALUES (1, 'local', 'none:local', 10);
		INSERT INTO rooms (id, scope, user_id, name, created_at) VALUES (2, 'user', 1, 'lounge', 11);
		INSERT INTO sessions (id, user_id, name, agent, created_at, last_seen_at) VALUES
			(1, 1, 'alice', 'claude', 12, 20), (2, 1, 'bob', 'codex', 13, 21);
		INSERT INTO messages (id, scope, sender_session_id, to_session_id, room_id, body, sent_at) VALUES
			(1, 'user', 1, 2, NULL, 'direct', 30), (2, 'user', 2, NULL, 2, 'room', 31);
		INSERT INTO deliveries (message_id, recipient_session_id, state, queued_at, delivered_at) VALUES
			(1, 2, 'delivered', 30, 32), (2, 1, 'queued', 31, NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	dump := func() string {
		t.Helper()
		var b strings.Builder
		for _, q := range []string{
			"SELECT id, name, token_hash, created_at FROM users ORDER BY id",
			"SELECT id, user_id, name, agent, created_at, last_seen_at FROM sessions ORDER BY id",
			"SELECT id, scope, coalesce(user_id, 0), name, created_at FROM rooms ORDER BY id",
			"SELECT id, scope, sender_session_id, coalesce(to_session_id, 0), coalesce(room_id, 0), body, sent_at FROM messages ORDER BY id",
			"SELECT message_id, recipient_session_id, state, queued_at, coalesce(delivered_at, 0) FROM deliveries ORDER BY message_id",
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

	// Opening migrates.
	s = open(t, path)
	if v, _ := s.SchemaVersion(ctx); v != len(ms) {
		t.Fatalf("schema version after migration = %d, want %d", v, len(ms))
	}
	if after := dump(); after != before {
		t.Errorf("data changed by the migration:\nbefore:\n%s\nafter:\n%s", before, after)
	}

	// The tables stay STRICT, the indexes exist, and no foreign key is broken.
	for _, table := range []string{"sessions", "rooms", "messages"} {
		var strict int
		if err := s.db.QueryRowContext(ctx, "SELECT strict FROM pragma_table_list WHERE name = ?", table).Scan(&strict); err != nil {
			t.Fatal(err)
		}
		if strict != 1 {
			t.Errorf("table %s is not STRICT", table)
		}
	}
	for _, index := range []string{"sessions_repository", "rooms_space_name", "messages_room", "messages_direct", "messages_sender"} {
		var n int
		if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE type = 'index' AND name = ?", index).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("index %s is missing", index)
		}
	}
	if rows, err := s.db.QueryContext(ctx, "PRAGMA foreign_key_check"); err != nil {
		t.Fatal(err)
	} else {
		if rows.Next() {
			t.Error("foreign key violations after the migration")
		}
		rows.Close()
	}
	// No table refers to a table name of the rebuild.
	var stale int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE sql LIKE '%_v2%'").Scan(&stale); err != nil {
		t.Fatal(err)
	}
	if stale != 0 {
		t.Error("the schema still refers to a temporary table name")
	}

	// The migrated data works: alice's unread room message is fetched, the new columns are
	// empty, and new checks hold.
	alice := Session{ID: 1, UserID: 1, Name: "alice"}
	msgs, err := s.FetchInbox(ctx, alice.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Body != "room" || msgs[0].Room != LoungeRoom || msgs[0].Scope != ScopeUser || msgs[0].Repository != "" {
		t.Errorf("alice's inbox after migration = %+v", msgs)
	}
	sessions, _, err := s.ListSessions(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, sess := range sessions {
		if sess.Profile != (Profile{}) {
			t.Errorf("session %s has profile %+v after migration", sess.Name, sess.Profile)
		}
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO rooms (scope, user_id, repository, name, created_at) VALUES ('repository', NULL, '', 'lounge', 1)"); err == nil {
		t.Error("a repository room without a repository was accepted")
	}
	// Foreign keys are enforced again after the migration, and cascades reach the rebuilt tables.
	var fk int
	if err := s.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys = %d, %v", fk, err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = 2"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM messages) + (SELECT count(*) FROM deliveries)").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("rows left after deleting bob = %d, want 0 (cascade)", n)
	}
}

func sqlString(v any) string {
	switch v := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}
