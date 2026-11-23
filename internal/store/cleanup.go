package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CleanupPolicy holds the periods of the periodic cleanup. A zero period disables its step.
type CleanupPolicy struct {
	// ExpireAfter is how long a delivery may stay queued before it expires.
	ExpireAfter time.Duration
	// RetainFor is how long a message is kept after its last change of state, once none of its
	// deliveries is queued.
	RetainFor time.Duration
	// SessionIdle is how long a session may stay unused before it is deleted.
	SessionIdle time.Duration
}

// CleanupResult counts what Cleanup did.
type CleanupResult struct {
	// Expired is the number of deliveries that expired.
	Expired int
	// Messages and MessageDeliveries are the messages deleted after the retention period and
	// their deliveries.
	Messages, MessageDeliveries int
	// Sessions are the unused sessions deleted; SessionMessages and SessionDeliveries are the
	// messages and deliveries deleted with them.
	Sessions, SessionMessages, SessionDeliveries int
	// Memories are the memories deleted by the forgetting model (the store's RetentionPolicy).
	Memories int
}

// Empty reports whether the cleanup did nothing.
func (r CleanupResult) Empty() bool {
	return r == CleanupResult{}
}

// Cleanup runs the four steps of the periodic cleanup, in this order, with the store's clock:
//
//  1. Expire: a delivery queued longer than ExpireAfter becomes expired. Only queued
//     deliveries expire: a delivered one stays delivered until it is acked (see delivery.go).
//  2. Delete messages: a message none of whose deliveries is queued, and whose last change of
//     state is older than RetainFor, is deleted with its deliveries. The last change is the
//     latest delivered_at, acked_at or expired_at of its deliveries, or its sent_at when it has
//     no delivery (it had no recipient, or their sessions were deleted). The rule covers every
//     state but queued, not only acked: messages that are never acked, and expired ones, would
//     otherwise stay forever. A queued message is kept until it expires.
//  3. Delete sessions: a session unused for longer than SessionIdle is deleted, with the
//     messages it sent, the messages addressed to it and every delivery to it. The memories
//     it wrote stay: a memory outlives its session.
//  4. Delete memories: a memory that faded more than the grace period ago is deleted, under
//     the store's RetentionPolicy (see retention.go), and the references older than
//     ReferenceWindow are dropped. The same transaction deletes a memory_scope_activity row
//     whose scope has no memory and whose last access is older than scopeActivityIdle.
//     Pinned and faded memories count as memories.
//
// Each step is its own transaction. When a step fails, Cleanup returns what the earlier steps
// did with the error; the caller tries again at its next run.
func (s *Store) Cleanup(ctx context.Context, p CleanupPolicy) (CleanupResult, error) {
	var r CleanupResult
	now := s.now()
	cutoff := func(d time.Duration) int64 { return millis(now.Add(-d)) }
	if p.ExpireAfter > 0 {
		err := s.tx(ctx, func(tx *sql.Tx) error {
			res, err := tx.ExecContext(ctx,
				"UPDATE deliveries SET state = 'expired', expired_at = ? WHERE state = 'queued' AND queued_at < ?",
				millis(now), cutoff(p.ExpireAfter))
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			r.Expired = int(n)
			return err
		})
		if err != nil {
			return r, fmt.Errorf("expire queued deliveries: %w", err)
		}
	}
	if p.RetainFor > 0 {
		err := s.tx(ctx, func(tx *sql.Tx) error {
			// Every state change of a delivery comes at or after the message was sent, so the
			// condition on sent_at holds for every message to delete and lets the index on
			// sent_at narrow the scan.
			const old = `
				SELECT m.id FROM messages m
				WHERE m.sent_at < ?1 AND NOT EXISTS (
					SELECT 1 FROM deliveries d WHERE d.message_id = m.id AND (d.state = 'queued'
						OR max(coalesce(d.delivered_at, 0), coalesce(d.acked_at, 0), coalesce(d.expired_at, 0)) >= ?1))`
			c := cutoff(p.RetainFor)
			if err := tx.QueryRowContext(ctx,
				"SELECT count(*) FROM deliveries WHERE message_id IN ("+old+")", c).Scan(&r.MessageDeliveries); err != nil {
				return err
			}
			// Deleting a message deletes its deliveries (ON DELETE CASCADE).
			res, err := tx.ExecContext(ctx, "DELETE FROM messages WHERE id IN ("+old+")", c)
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			r.Messages = int(n)
			return err
		})
		if err != nil {
			r.Messages, r.MessageDeliveries = 0, 0
			return r, fmt.Errorf("delete old messages: %w", err)
		}
	}
	if p.SessionIdle > 0 {
		err := s.tx(ctx, func(tx *sql.Tx) error {
			const idle = "SELECT id FROM sessions WHERE last_seen_at < ?"
			c := cutoff(p.SessionIdle)
			d, err := countSessionData(ctx, tx, idle, c)
			if err != nil {
				return err
			}
			if d.Sessions == 0 {
				return nil
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM sessions WHERE id IN ("+idle+")", c); err != nil {
				return err
			}
			r.Sessions, r.SessionMessages, r.SessionDeliveries = d.Sessions, d.Messages, d.Deliveries
			return nil
		})
		if err != nil {
			return r, fmt.Errorf("delete unused sessions: %w", err)
		}
	}
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var err error
		r.Memories, err = s.deleteForgottenMemories(ctx, tx, now)
		if err != nil {
			return err
		}
		return deleteIdleScopeActivity(ctx, tx, now)
	})
	if err != nil {
		r.Memories = 0
		return r, fmt.Errorf("delete forgotten memories: %w", err)
	}
	return r, nil
}

// countSessionData counts the sessions selected by the query sessions (which selects their
// IDs) and what deleting them deletes: the messages they sent or that were addressed to them,
// the deliveries of those messages and the deliveries to the sessions.
func countSessionData(ctx context.Context, tx *sql.Tx, sessions string, args ...any) (UserData, error) {
	var d UserData
	err := tx.QueryRowContext(ctx, `
		WITH own AS (`+sessions+`),
		msgs AS (SELECT id FROM messages WHERE sender_session_id IN own OR to_session_id IN own)
		SELECT (SELECT count(*) FROM own), (SELECT count(*) FROM msgs),
			(SELECT count(*) FROM deliveries WHERE message_id IN msgs OR recipient_session_id IN own)`, args...).
		Scan(&d.Sessions, &d.Messages, &d.Deliveries)
	return d, err
}
