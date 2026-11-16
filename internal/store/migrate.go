package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

// schemaFS embeds the migration scripts. A script is named NNNN_description.sql, where NNNN is
// the schema version the script produces. Versions start at 1 and have no gaps. A released
// script is never edited; schema changes are added as a new script.
//
//go:embed schema/*.sql
var schemaFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

// loadMigrations reads the scripts of fsys in version order.
func loadMigrations(fsys fs.FS) ([]migration, error) {
	names, err := fs.Glob(fsys, "schema/*.sql")
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	var ms []migration
	for i, name := range names {
		base := strings.TrimPrefix(name, "schema/")
		prefix, _, _ := strings.Cut(base, "_")
		v, err := strconv.Atoi(prefix)
		if err != nil || v != i+1 {
			return nil, fmt.Errorf("migration %s: want version prefix %04d", base, i+1)
		}
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		ms = append(ms, migration{version: v, name: base, sql: string(b)})
	}
	return ms, nil
}

// migrate brings the database to the latest schema version.
func (s *Store) migrate(ctx context.Context) error {
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		return err
	}
	return s.applyMigrations(ctx, ms)
}

// applyMigrations applies the scripts newer than the database in one transaction, so a failed
// upgrade leaves the file unchanged. The schema version is kept in PRAGMA user_version.
func (s *Store) applyMigrations(ctx context.Context, ms []migration) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		// Read inside the write transaction: another process may have migrated meanwhile.
		current, err := schemaVersion(ctx, tx)
		if err != nil {
			return err
		}
		if current > len(ms) {
			return fmt.Errorf("database schema version %d is newer than this binary supports (%d); use a newer agent-lounge-mcp", current, len(ms))
		}
		for _, m := range ms[current:] {
			if _, err := tx.ExecContext(ctx, m.sql); err != nil {
				return fmt.Errorf("apply migration %s: %w", m.name, err)
			}
		}
		if current == len(ms) {
			return nil
		}
		// PRAGMA does not accept bound parameters; the value is an int.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", len(ms))); err != nil {
			return fmt.Errorf("record schema version: %w", err)
		}
		return nil
	})
}
