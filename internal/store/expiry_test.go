package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// userSummary returns the ListUsers entry of the user.
func userSummary(t *testing.T, s *Store, name string) UserSummary {
	t.Helper()
	users, err := s.ListUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range users {
		if u.Name == name {
			return u
		}
	}
	t.Fatalf("no user %s", name)
	return UserSummary{}
}

// The expiry of a token: given when it is issued, judged by the store's clock from the expiry
// time on, changed without changing the token, and not carried over by a rotation.
func TestTokenExpiry(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	now := time.UnixMilli(1_800_000_000_000).UTC()
	s.SetClock(func() time.Time { return now })
	expiry := now.Add(time.Hour)

	alice, err := s.AddUser(ctx, "alice", "hash-a", expiry)
	if err != nil {
		t.Fatal(err)
	}
	bob := addUser(t, s, "bob") // no expiry
	if got := userSummary(t, s, "alice").TokenExpiresAt; !got.Equal(expiry) {
		t.Errorf("listed expiry = %v, want %v", got, expiry)
	}
	if got := userSummary(t, s, "bob").TokenExpiresAt; !got.IsZero() {
		t.Errorf("bob's expiry = %v", got)
	}
	if got := userSummary(t, s, LocalUser).TokenExpiresAt; !got.IsZero() {
		t.Errorf("local's expiry = %v", got)
	}

	check := func(when string, want TokenState) {
		t.Helper()
		_, err := s.UserByTokenHash(ctx, "hash-a")
		state, serr := s.TokenStatus(ctx, alice.ID, "hash-a")
		if serr != nil || state != want {
			t.Errorf("%s: status %v, %v, want %v", when, state, serr, want)
		}
		switch want {
		case TokenOK:
			if err != nil {
				t.Errorf("%s: lookup: %v", when, err)
			}
		case TokenExpired:
			if !errors.Is(err, ErrTokenExpired) {
				t.Errorf("%s: lookup: %v, want ErrTokenExpired", when, err)
			}
		}
	}
	now = expiry.Add(-time.Millisecond)
	check("just before the expiry", TokenOK)
	now = expiry
	check("at the expiry", TokenExpired)
	// The other users are not affected.
	if _, err := s.UserByTokenHash(ctx, "hash-bob"); err != nil {
		t.Errorf("bob: %v", err)
	}
	if state, err := s.TokenStatus(ctx, bob.ID, "hash-bob"); err != nil || state != TokenOK {
		t.Errorf("bob: %v, %v", state, err)
	}

	// Extended: the same token is valid again. Never: no expiry.
	if err := s.SetTokenExpiry(ctx, "alice", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	check("after the extension", TokenOK)
	if err := s.SetTokenExpiry(ctx, "alice", time.Time{}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(10000 * time.Hour)
	check("without an expiry", TokenOK)
	if got := userSummary(t, s, "alice").TokenExpiresAt; !got.IsZero() {
		t.Errorf("listed expiry after never = %v", got)
	}
	for _, name := range []string{"nobody", LocalUser, "Bad/Name"} {
		if err := s.SetTokenExpiry(ctx, name, now); err == nil {
			t.Errorf("expiry of %s was accepted", name)
		}
	}
	if err := s.SetTokenExpiry(ctx, "nobody", now); !errors.Is(err, ErrNoSuchUser) {
		t.Errorf("nobody: %v", err)
	}

	// Rotate with an expiry, then without: the second token does not keep the first's expiry.
	if err := s.RotateToken(ctx, "alice", "hash-a2", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got := userSummary(t, s, "alice").TokenExpiresAt; !got.Equal(now.Add(time.Minute)) {
		t.Errorf("expiry after rotate = %v", got)
	}
	if state, _ := s.TokenStatus(ctx, alice.ID, "hash-a"); state != TokenRevoked {
		t.Errorf("old token after rotate: %v", state)
	}
	if err := s.RotateToken(ctx, "alice", "hash-a3", time.Time{}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	if _, err := s.UserByTokenHash(ctx, "hash-a3"); err != nil {
		t.Errorf("rotated without an expiry: %v", err)
	}
	if got := userSummary(t, s, "alice").TokenExpiresAt; !got.IsZero() {
		t.Errorf("expiry after rotate without one = %v", got)
	}

	// Removed: revoked, not expired.
	if _, err := s.RemoveUser(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if state, _ := s.TokenStatus(ctx, alice.ID, "hash-a3"); state != TokenRevoked {
		t.Errorf("removed: %v", state)
	}
	// local never has a valid token.
	if state, _ := s.TokenStatus(ctx, local, "none:local"); state != TokenRevoked {
		t.Errorf("local: %v", state)
	}
}

// The last use of a user is the later of the last use of its token and the last activity of
// its sessions. local is never recorded: its last use stays that of its sessions.
func TestTokenLastUse(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	now := time.UnixMilli(1_800_000_000_000).UTC()
	s.SetClock(func() time.Time { return now })
	alice := addUser(t, s, "alice")
	if got := userSummary(t, s, "alice").LastUsedAt; !got.IsZero() {
		t.Errorf("new user: %v", got)
	}

	// Recorded without a session.
	if err := s.RecordTokenUse(ctx, alice.ID); err != nil {
		t.Fatal(err)
	}
	if got := userSummary(t, s, "alice").LastUsedAt; !got.Equal(now) {
		t.Errorf("token only: %v, want %v", got, now)
	}
	// A later session activity wins.
	now = now.Add(time.Hour)
	join(t, s, alice.ID, "work")
	if got := userSummary(t, s, "alice").LastUsedAt; !got.Equal(now) {
		t.Errorf("later session: %v, want %v", got, now)
	}
	// A later token use wins.
	now = now.Add(time.Hour)
	if err := s.RecordTokenUse(ctx, alice.ID); err != nil {
		t.Fatal(err)
	}
	if got := userSummary(t, s, "alice").LastUsedAt; !got.Equal(now) {
		t.Errorf("later token use: %v, want %v", got, now)
	}

	// local: nothing recorded, its sessions decide.
	if got := userSummary(t, s, LocalUser).LastUsedAt; !got.IsZero() {
		t.Errorf("local without sessions: %v", got)
	}
	joinedAt := now
	join(t, s, local, "l")
	now = now.Add(time.Hour)
	if err := s.RecordTokenUse(ctx, local); err != nil {
		t.Fatal(err)
	}
	if got := userSummary(t, s, LocalUser).LastUsedAt; !got.Equal(joinedAt) {
		t.Errorf("local: %v, want %v", got, joinedAt)
	}
	var stored sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT last_used_at FROM users WHERE id = ?", local).Scan(&stored); err != nil || stored.Valid {
		t.Errorf("local's last_used_at = %v, %v", stored, err)
	}
	// A removed user is ignored.
	if err := s.RecordTokenUse(ctx, 9999); err != nil {
		t.Errorf("missing user: %v", err)
	}
}

// A database of schema version 5 migrates to the latest version: the data stays, every user
// gets no expiry and no recorded use, and the expiry works.
func TestMigrateVersion5(t *testing.T) {
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
	if len(ms) < 6 {
		t.Fatal("no version 6 script")
	}
	if err := s.applyMigrations(ctx, ms[:5]); err != nil {
		t.Fatal(err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, name, token_hash, created_at) VALUES (1, 'local', 'none:local', 10), (2, 'alice', 'hash-a', 11);
		INSERT INTO rooms (id, scope, user_id, repository, name, created_at) VALUES (2, 'user', 1, '', 'lounge', 11), (3, 'user', 2, '', 'lounge', 11);
		INSERT INTO sessions (id, user_id, name, agent, created_at, last_seen_at) VALUES (1, 1, 'l', 'claude', 12, 20), (2, 2, 'a', '', 13, 21);
		INSERT INTO messages (id, scope, scope_id, sender_session_id, to_session_id, room_id, body, sent_at, sender_agent) VALUES
			(1, 'user', 'alice', 2, NULL, 3, 'room', 30, '');
		INSERT INTO memories (id, scope, scope_id, user_id, body, created_at, updated_at, created_user, created_session, updated_user, updated_session) VALUES
			(1, 'user', 'alice', 2, 'kept', 40, 40, 'alice', 'a', 'alice', 'a');`)
	if err != nil {
		t.Fatal(err)
	}
	dump := func() string {
		return dumpRows(t, s, []string{
			"SELECT id, name, token_hash, created_at FROM users ORDER BY id", "SELECT id, user_id, name, agent, created_at, last_seen_at, os_user, machine, origin, folder, repository FROM sessions ORDER BY id",
			"SELECT * FROM rooms ORDER BY id", "SELECT * FROM messages ORDER BY id", "SELECT * FROM memories ORDER BY id",
		})
	}
	before := dump()
	s.Close()

	s = open(t, path)
	if v, _ := s.SchemaVersion(ctx); v != len(ms) {
		t.Fatalf("schema version = %d, want %d", v, len(ms))
	}
	if after := dump(); after != before {
		t.Errorf("data changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE token_expires_at IS NULL AND last_used_at IS NULL").Scan(&n); err != nil || n != 2 {
		t.Errorf("users without expiry and last use = %d, %v", n, err)
	}
	if u, err := s.UserByTokenHash(ctx, "hash-a"); err != nil || u.Name != "alice" {
		t.Errorf("lookup after migration: %+v, %v", u, err)
	}
	if got := userSummary(t, s, "alice"); !got.TokenExpiresAt.IsZero() || !got.LastUsedAt.Equal(fromMillis(21)) {
		t.Errorf("listed after migration: %+v", got)
	}
	s.SetClock(func() time.Time { return fromMillis(100) })
	if err := s.SetTokenExpiry(ctx, "alice", fromMillis(100)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserByTokenHash(ctx, "hash-a"); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expired after migration: %v", err)
	}
}
