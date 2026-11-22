package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestBranchScopeRulesAndJoinReplacement(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)

	if want := []string{ScopeUser, ScopeRepository, ScopeBranch}; !slices.Equal(ScopeKinds(), want) {
		t.Fatalf("ScopeKinds() = %v, want %v", ScopeKinds(), want)
	}

	// Location + branch sets both.
	a := joinWith(t, s, JoinParams{
		UserID: user, Name: "a",
		Location: &Location{Origin: "github.com/org/repo"},
		Branch:   ptr("feature/x"),
	})
	if a.Branch != "feature/x" || a.Repository != "github.com/org/repo" {
		t.Fatalf("join with location+branch: %+v", a.Profile)
	}
	wantScopes := []Scope{
		{Kind: ScopeUser, ID: LocalUser},
		{Kind: ScopeRepository, ID: "github.com/org/repo"},
		{Kind: ScopeBranch, ID: "github.com/org/repo:feature/x"},
	}
	if got := ScopesOf(a); !slices.Equal(got, wantScopes) {
		t.Fatalf("ScopesOf(a) = %+v, want %+v", got, wantScopes)
	}

	// Omitting everything keeps both location and branch.
	a = joinWith(t, s, JoinParams{UserID: user, Name: "a"})
	if a.Repository != "github.com/org/repo" || a.Branch != "feature/x" {
		t.Errorf("omit all: %+v", a.Profile)
	}

	// Branch alone replaces only branch and keeps location.
	a = joinWith(t, s, JoinParams{UserID: user, Name: "a", Branch: ptr("feature/y")})
	if a.Repository != "github.com/org/repo" || a.Branch != "feature/y" {
		t.Errorf("branch alone: %+v", a.Profile)
	}

	// Location without branch clears branch.
	a = joinWith(t, s, JoinParams{UserID: user, Name: "a", Location: &Location{Origin: "github.com/org/other"}})
	if a.Repository != "github.com/org/other" || a.Branch != "" {
		t.Errorf("location without branch: %+v", a.Profile)
	}

	// Empty branch clears branch; branch without repository is recorded but branch scope is unavailable.
	noRepo := joinWith(t, s, JoinParams{UserID: user, Name: "norepo", Branch: ptr("main")})
	if noRepo.Branch != "main" || noRepo.Repository != "" {
		t.Fatalf("branch without repo: %+v", noRepo.Profile)
	}
	if got := ScopesOf(noRepo); !slices.Equal(got, []Scope{{Kind: ScopeUser, ID: LocalUser}}) {
		t.Errorf("ScopesOf(noRepo) = %+v", got)
	}
	if _, err := sendRoom(s, noRepo, ScopeBranch, LoungeRoom, "hi"); !errors.Is(err, ErrNoBranch) {
		t.Errorf("send branch without repo: err = %v, want ErrNoBranch", err)
	}
	if _, err := sendRoom(s, a, ScopeBranch, LoungeRoom, "hi"); !errors.Is(err, ErrNoBranch) {
		t.Errorf("send branch without branch: err = %v, want ErrNoBranch", err)
	}
	cleared := joinWith(t, s, JoinParams{UserID: user, Name: "norepo", Branch: ptr("")})
	if cleared.Branch != "" {
		t.Errorf("clear branch: %+v", cleared.Profile)
	}
	_ = ctx
}

func TestSendToBranchRoomAndSwitch(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	if _, err := s.db.ExecContext(ctx, "INSERT INTO users (name, token_hash, created_at) VALUES ('other', 'h', 1)"); err != nil {
		t.Fatal(err)
	}
	var other int64
	if err := s.db.QueryRowContext(ctx, "SELECT id FROM users WHERE name = 'other'").Scan(&other); err != nil {
		t.Fatal(err)
	}

	repoA := &Location{Origin: "github.com/org/a"}
	repoB := &Location{Origin: "github.com/org/b"}
	alice := joinWith(t, s, JoinParams{UserID: user, Name: "alice", Location: repoA, Branch: ptr("feat")})
	bob := joinWith(t, s, JoinParams{UserID: other, Name: "bob", Location: repoA, Branch: ptr("feat")})
	carol := joinWith(t, s, JoinParams{UserID: user, Name: "carol", Location: repoA, Branch: ptr("main")})
	dave := joinWith(t, s, JoinParams{UserID: user, Name: "dave", Location: repoB, Branch: ptr("feat")})

	// SharedScopes between alice and bob includes repository and branch.
	if got := SharedScopes(alice, bob); !slices.Equal(got, []Scope{
		{Kind: ScopeRepository, ID: "github.com/org/a"},
		{Kind: ScopeBranch, ID: "github.com/org/a:feat"},
	}) {
		t.Errorf("SharedScopes(alice, bob) = %+v", got)
	}
	// SharedScopes between alice and carol (different branch) excludes branch.
	if got := SharedScopes(alice, carol); !slices.Equal(got, []Scope{
		{Kind: ScopeUser, ID: LocalUser},
		{Kind: ScopeRepository, ID: "github.com/org/a"},
	}) {
		t.Errorf("SharedScopes(alice, carol) = %+v", got)
	}

	sent, err := sendRoom(s, alice, ScopeBranch, LoungeRoom, "branch hello")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names(sent.Recipients), []string{"bob"}) || sent.Scope != (Scope{ScopeBranch, "github.com/org/a:feat"}) {
		t.Errorf("sent = %+v", sent)
	}
	if n := unread(t, s, bob); n != 1 {
		t.Errorf("bob unread = %d, want 1", n)
	}
	for _, sess := range []Session{alice, carol, dave} {
		if n := unread(t, s, sess); n != 0 {
			t.Errorf("%s unread = %d, want 0", sess.Name, n)
		}
	}

	// After carol switches to feat, carol gets new branch messages (not old ones).
	carol = joinWith(t, s, JoinParams{UserID: user, Name: "carol", Branch: ptr("feat")})
	sent, err = sendRoom(s, alice, ScopeBranch, LoungeRoom, "after switch")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names(sent.Recipients), []string{"bob", "carol"}) {
		t.Errorf("after switch recipients = %+v", sent.Recipients)
	}
	if n := unread(t, s, carol); n != 1 {
		t.Errorf("carol unread = %d, want 1", n)
	}
}

func TestBranchMemoriesAndLimit(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	lim := DefaultLimits()
	lim.MaxMemoriesPerScope = 2
	s.SetLimits(lim)

	repoA := &Location{Origin: "github.com/org/a"}
	a1 := joinWith(t, s, JoinParams{UserID: user, Name: "a1", Location: repoA, Branch: ptr("feat")})
	a2 := joinWith(t, s, JoinParams{UserID: user, Name: "a2", Location: repoA, Branch: ptr("feat")})
	otherBranch := joinWith(t, s, JoinParams{UserID: user, Name: "other", Location: repoA, Branch: ptr("main")})

	m1, err := s.CreateMemory(ctx, a1.ID, ScopeBranch, MemoryInput{Body: ptr("branch note 1")})
	if err != nil {
		t.Fatal(err)
	}
	if m1.Scope != (Scope{Kind: ScopeBranch, ID: "github.com/org/a:feat"}) {
		t.Fatalf("scope = %+v", m1.Scope)
	}
	if _, err := s.CreateMemory(ctx, a2.ID, ScopeBranch, MemoryInput{Body: ptr("branch note 2")}); err != nil {
		t.Fatal(err)
	}
	// Limit applies per branch scope.
	if _, err := s.CreateMemory(ctx, a1.ID, ScopeBranch, MemoryInput{Body: ptr("branch note 3")}); !errors.Is(err, ErrTooManyMemories) {
		t.Errorf("3rd branch memory: err = %v, want ErrTooManyMemories", err)
	}
	// Another branch has its own quota and cannot read feat's memory.
	if _, err := s.CreateMemory(ctx, otherBranch.ID, ScopeBranch, MemoryInput{Body: ptr("main note")}); err != nil {
		t.Fatalf("other branch memory: %v", err)
	}
	if _, err := s.GetMemory(ctx, otherBranch.ID, m1.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("other branch GetMemory: err = %v, want ErrNoSuchMemory", err)
	}
	found, _, err := s.SearchMemories(ctx, a2.ID, MemorySearch{Scope: ScopeBranch, Limit: 10})
	if err != nil || len(found) != 2 {
		t.Errorf("SearchMemories(a2, branch) = %d, %v", len(found), err)
	}
}

func TestMigrationV6ToV7PreservesDataAndSequence(t *testing.T) {
	ctx := context.Background()
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 7 {
		t.Fatalf("migrations = %d, want at least 7", len(ms))
	}

	for _, deleteAll := range []bool{false, true} {
		name := "highest_deleted"
		if deleteAll {
			name = "all_deleted"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "v6.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			s := &Store{db: db, now: time.Now, limits: DefaultLimits(), retention: DefaultRetention()}
			if err := s.applyMigrations(ctx, ms[:6]); err != nil {
				db.Close()
				t.Fatal(err)
			}
			// Seed v6 data: user, session, repo room, message, and memories where the highest ID is deleted.
			if _, err := db.ExecContext(ctx, `
				INSERT INTO users (id, name, token_hash, created_at) VALUES (1, 'local', 'none:local', 1000);
				INSERT INTO sessions (id, user_id, name, agent, os_user, machine, origin, folder, repository, created_at, last_seen_at)
				VALUES (10, 1, 'alice', 'claude', 'u', 'm', 'github.com/o/r', '/f', 'github.com/o/r', 1000, 1000);
				INSERT INTO rooms (id, scope, user_id, repository, name, created_at)
				VALUES (20, 'repository', NULL, 'github.com/o/r', 'lounge', 1000);
				INSERT INTO messages (id, scope, scope_id, sender_session_id, room_id, body, sent_at, sender_agent, sender_os_user, sender_machine)
				VALUES (30, 'repository', 'github.com/o/r', 10, 20, 'v6 msg', 1000, 'claude', 'u', 'm');
				INSERT INTO memories (id, scope, scope_id, user_id, title, body, created_at, updated_at, created_user, created_session, updated_user, updated_session)
				VALUES (1, 'user', 'local', 1, 't1', 'keep', 1000, 1000, 'local', 'alice', 'local', 'alice'),
				       (5, 'user', 'local', 1, 't5', 'deleted', 1000, 1000, 'local', 'alice', 'local', 'alice');
				INSERT INTO memories_fts (rowid, title, body, tags) VALUES (1, 't1', 'keep', '');
			`); err != nil {
				db.Close()
				t.Fatal(err)
			}
			if deleteAll {
				if _, err := db.ExecContext(ctx, "DELETE FROM memories; DELETE FROM memories_fts;"); err != nil {
					db.Close()
					t.Fatal(err)
				}
			} else {
				if _, err := db.ExecContext(ctx, "DELETE FROM memories WHERE id = 5"); err != nil {
					db.Close()
					t.Fatal(err)
				}
			}
			db.Close()

			// Reopen with Open(), which applies migration 0007_branch.sql.
			upgraded := open(t, path)
			v, err := upgraded.SchemaVersion(ctx)
			if err != nil || v != len(ms) {
				t.Fatalf("version = %d, %v; want %d", v, err, len(ms))
			}

			// Verify existing room and message IDs survived and session.branch defaults to "".
			var body, branch string
			if err := upgraded.db.QueryRowContext(ctx, "SELECT body FROM messages WHERE id = 30").Scan(&body); err != nil || body != "v6 msg" {
				t.Errorf("message 30 = %q, %v", body, err)
			}
			if err := upgraded.db.QueryRowContext(ctx, "SELECT branch FROM sessions WHERE id = 10").Scan(&branch); err != nil || branch != "" {
				t.Errorf("session 10 branch = %q, %v", branch, err)
			}

			// Creating a new memory must get ID > 5 (never reusing deleted ID 5).
			m, err := upgraded.CreateMemory(ctx, 10, ScopeUser, MemoryInput{Body: ptr("after migration")})
			if err != nil {
				t.Fatal(err)
			}
			if m.ID <= 5 {
				t.Errorf("new memory ID = %d, want > 5 (sqlite_sequence not preserved)", m.ID)
			}
			if !deleteAll {
				got, err := upgraded.GetMemory(ctx, 10, 1)
				if err != nil || got.Body != "keep" {
					t.Errorf("existing memory 1 = %+v, %v", got, err)
				}
			}
		})
	}
}
