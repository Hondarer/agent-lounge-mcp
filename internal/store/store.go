// Package store is the SQLite access layer of agent-lounge-mcp. All SQL lives in this package;
// other packages use the methods of Store.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // pure Go driver registered as "sqlite"; no C compiler needed
)

// Store is a handle to the database file. It is safe for concurrent use.
type Store struct {
	db *sql.DB
}

// pragmas are applied to every connection of the pool.
//   - busy_timeout: wait for a competing writer instead of failing with SQLITE_BUSY.
//   - journal_mode(WAL): readers do not block the writer and the writer does not block readers.
//   - synchronous(NORMAL): the durability level recommended with WAL.
//   - foreign_keys: SQLite does not enforce them unless enabled per connection.
var pragmas = []string{
	"busy_timeout(5000)",
	"journal_mode(WAL)",
	"synchronous(NORMAL)",
	"foreign_keys(1)",
}

// Open opens the database file at path, creating it when missing, and migrates its schema to
// the version this binary expects.
func Open(ctx context.Context, path string) (*Store, error) {
	// The driver splits the data source name at the first '?'.
	if path == "" || strings.ContainsRune(path, '?') {
		return nil, fmt.Errorf("invalid database path %q", path)
	}
	// _txlock=immediate takes the write lock when a transaction begins, so a transaction that
	// reads and then writes cannot fail halfway with SQLITE_BUSY.
	dsn := path + "?_txlock=immediate"
	for _, p := range pragmas {
		dsn += "&_pragma=" + p
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return s, nil
}

// Close releases the database. It checkpoints the write-ahead log when it is the last connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// SchemaVersion returns the schema version recorded in the database file.
func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	return schemaVersion(ctx, s.db)
}

// tx runs fn in a write transaction: it commits when fn returns nil and rolls back otherwise.
func (s *Store) tx(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, ignoreTxDone(tx.Rollback()))
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func ignoreTxDone(err error) error {
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}
	return err
}

// queryer is implemented by *sql.DB and *sql.Tx.
type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func schemaVersion(ctx context.Context, q queryer) (int, error) {
	var v int
	if err := q.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return v, nil
}
