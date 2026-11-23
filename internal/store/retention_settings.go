package store

import (
	"context"
	"database/sql"
	"fmt"
)

// RecordRetention replaces all recorded policies in a single transaction.
func (s *Store) RecordRetention(ctx context.Context, policies RetentionPolicies) error {
	at := millis(s.now())
	err := s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM memory_retention"); err != nil {
			return err
		}
		for _, kind := range scopeKinds {
			p := policies.For(kind)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO memory_retention (scope, half_life_ns, threshold, grace_ns, recorded_at)
				VALUES (?, ?, ?, ?, ?)`, kind, p.HalfLife, p.Threshold, p.Grace, at); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("record memory retention: %w", err)
	}
	return nil
}

// RecordedRetention returns the policies last recorded by serve, with built-in defaults for
// any kind with no record.
func (s *Store) RecordedRetention(ctx context.Context) (RetentionPolicies, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT scope, half_life_ns, threshold, grace_ns FROM memory_retention")
	if err != nil {
		return RetentionPolicies{}, fmt.Errorf("read recorded memory retention: %w", err)
	}
	defer rows.Close()
	policies := RetentionPolicies{Common: DefaultRetention(), ByKind: make(map[string]RetentionPolicy)}
	for rows.Next() {
		var kind string
		var p RetentionPolicy
		if err := rows.Scan(&kind, &p.HalfLife, &p.Threshold, &p.Grace); err != nil {
			return RetentionPolicies{}, fmt.Errorf("scan recorded memory retention: %w", err)
		}
		policies.ByKind[kind] = p
	}
	if err := rows.Err(); err != nil {
		return RetentionPolicies{}, fmt.Errorf("read recorded memory retention: %w", err)
	}
	return policies, nil
}
