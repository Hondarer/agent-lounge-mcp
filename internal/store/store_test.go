package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func open(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func latestVersion(t *testing.T) int {
	t.Helper()
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("no migrations embedded")
	}
	return len(ms)
}

func TestOpenCreatesSchema(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "lounge.db"))

	v, err := s.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := latestVersion(t); v != want {
		t.Fatalf("schema version = %d, want %d", v, want)
	}

	rows, err := s.db.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	tables := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables[name] = true
	}
	for _, want := range []string{"users", "sessions", "rooms", "messages", "deliveries"} {
		if !tables[want] {
			t.Errorf("table %q is missing", want)
		}
	}

	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM rooms WHERE scope = 'shared' AND name = 'lounge'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("shared lounge rooms = %d, want 1", n)
	}
}

func TestReopenKeepsData(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lounge.db")

	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO users (name, token_hash, created_at) VALUES ('honda', 'h', 1)"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s = open(t, path)
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("users = %d, want 1", n)
	}
	// The seed row of the first migration is not inserted again.
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM rooms").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("rooms = %d, want 1", n)
	}
}

func TestConnectionPragmas(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "lounge.db"))

	var mode string
	if err := s.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
	var fk, busy int
	if err := s.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatal(err)
	}
	if fk != 1 || busy != 5000 {
		t.Errorf("foreign_keys = %d, busy_timeout = %d; want 1, 5000", fk, busy)
	}

	// Foreign keys are enforced: a session cannot refer to a missing user.
	_, err := s.db.ExecContext(ctx, "INSERT INTO sessions (user_id, name, created_at, last_seen_at) VALUES (999, 's', 1, 1)")
	if err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
		t.Errorf("insert with a missing user: err = %v, want a foreign key error", err)
	}
}

func TestNewerSchemaIsRejected(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lounge.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "PRAGMA user_version = 9999"); err != nil {
		t.Fatal(err)
	}
	s.Close()

	if _, err := Open(ctx, path); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("Open = %v, want a newer-schema error", err)
	}
}

func TestFailedMigrationLeavesDatabaseUnchanged(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "lounge.db"))
	before := latestVersion(t)

	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	ms = append(ms,
		migration{version: before + 1, name: "ok.sql", sql: "CREATE TABLE extra (id INTEGER PRIMARY KEY);"},
		migration{version: before + 2, name: "broken.sql", sql: "CREATE TABLE broken (;"},
	)
	if err := s.applyMigrations(ctx, ms); err == nil || !strings.Contains(err.Error(), "broken.sql") {
		t.Fatalf("applyMigrations = %v, want an error naming broken.sql", err)
	}

	v, err := s.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v != before {
		t.Errorf("schema version = %d, want %d", v, before)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE name = 'extra'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("table of the rolled back migration exists")
	}
}

func TestConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "lounge.db"))

	const writers = 16
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := range writers {
		wg.Go(func() {
			errs <- s.tx(ctx, func(tx *sql.Tx) error {
				var n int
				if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n); err != nil {
					return err
				}
				_, err := tx.ExecContext(ctx, "INSERT INTO users (name, token_hash, created_at) VALUES (?, ?, 1)",
					"user"+string(rune('a'+i)), "hash"+string(rune('a'+i)))
				return err
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != writers {
		t.Errorf("users = %d, want %d", n, writers)
	}
}

// The memory stage plans full-text search with the FTS5 trigram tokenizer, which finds Japanese
// text without word boundaries. This checks that the driver's SQLite build provides it.
func TestFTS5TrigramIsAvailable(t *testing.T) {
	ctx := context.Background()
	s := open(t, filepath.Join(t.TempDir(), "lounge.db"))

	err := s.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			CREATE VIRTUAL TABLE temp.probe USING fts5(body, tokenize = 'trigram');
			INSERT INTO temp.probe (body) VALUES ('このビルドエラーはこの手順で直る'), ('the API changes this week');`)
		if err != nil {
			return err
		}
		for query, want := range map[string]int{"ビルドエラー": 1, "changes": 1, "存在しない": 0} {
			var n int
			if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM temp.probe WHERE probe MATCH ?", query).Scan(&n); err != nil {
				return err
			}
			if n != want {
				t.Errorf("MATCH %q found %d rows, want %d", query, n, want)
			}
		}
		_, err = tx.ExecContext(ctx, "DROP TABLE temp.probe")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenRejectsInvalidPath(t *testing.T) {
	for _, path := range []string{"", "lounge.db?mode=memory"} {
		if _, err := Open(context.Background(), path); err == nil {
			t.Errorf("Open(%q) succeeded, want an error", path)
		}
	}
}
