package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// RecordFilter selects messages and memories for the administration commands.
// The zero value selects everything. Several fields together keep the records
// that meet all of them.
//
// User is compared with the names stored on the records, not with the rows of
// users, so a name can be given after the user was removed. On a message those
// names are the sender, the direct recipient and each delivery recipient. On a
// memory they are the identifier of a user scope and the creator (created_user),
// which stays when a memory outside that user's own scope outlives the user.
// Messages the removed user sent do not: they are deleted with the user.
type RecordFilter struct {
	User  string
	Scope string
	// ScopeID is compared only when HasScopeID is set, so an empty identifier
	// can be selected on its own.
	ScopeID    string
	HasScopeID bool
	// Since is inclusive and Until is exclusive. The zero time sets no bound.
	// A message is compared by the time it was sent, a memory by the time it
	// was last updated.
	Since time.Time
	Until time.Time
}

// IsZero reports whether f selects everything.
func (f RecordFilter) IsZero() bool {
	return f.User == "" && f.Scope == "" && !f.HasScopeID && f.Since.IsZero() && f.Until.IsZero()
}

func (f RecordFilter) validate() error {
	if f.Scope != "" && !ValidScope(f.Scope) {
		return ErrInvalidScope
	}
	return nil
}

// messageWhere returns the conditions for f, and for the list-only state and
// beforeID ("" and 0 add nothing), on a query that joins messages m, the sender
// session s and user u, the room r, and the direct recipient session ts and user tu.
func (f RecordFilter) messageWhere(state string, beforeID int64) (string, []any) {
	var conds []string
	var args []any
	if f.User != "" {
		// The direct recipient is also a delivery recipient when the message was
		// queued for them. Both are checked so a direct recipient still matches
		// when the delivery row is gone.
		conds = append(conds, `(u.name = ? OR tu.name = ? OR EXISTS (
			SELECT 1 FROM deliveries fd
			JOIN sessions fs ON fs.id = fd.recipient_session_id
			JOIN users fu ON fu.id = fs.user_id
			WHERE fd.message_id = m.id AND fu.name = ?))`)
		args = append(args, f.User, f.User, f.User)
	}
	if f.Scope != "" {
		conds = append(conds, "m.scope = ?")
		args = append(args, f.Scope)
	}
	if f.HasScopeID {
		conds = append(conds, "m.scope_id = ?")
		args = append(args, f.ScopeID)
	}
	if !f.Since.IsZero() {
		conds = append(conds, "m.sent_at >= ?")
		args = append(args, millis(f.Since))
	}
	if !f.Until.IsZero() {
		conds = append(conds, "m.sent_at < ?")
		args = append(args, millis(f.Until))
	}
	if state != "" {
		conds = append(conds, "EXISTS (SELECT 1 FROM deliveries sd WHERE sd.message_id = m.id AND sd.state = ?)")
		args = append(args, state)
	}
	if beforeID > 0 {
		conds = append(conds, "m.id < ?")
		args = append(args, beforeID)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// memoryWhere returns the conditions for f on memories. Columns are unqualified.
func (f RecordFilter) memoryWhere() (string, []any) {
	var conds []string
	var args []any
	if f.User != "" {
		conds = append(conds, "((scope = ? AND scope_id = ?) OR created_user = ?)")
		args = append(args, ScopeUser, f.User, f.User)
	}
	if f.Scope != "" {
		conds = append(conds, "scope = ?")
		args = append(args, f.Scope)
	}
	if f.HasScopeID {
		conds = append(conds, "scope_id = ?")
		args = append(args, f.ScopeID)
	}
	if !f.Since.IsZero() {
		conds = append(conds, "updated_at >= ?")
		args = append(args, millis(f.Since))
	}
	if !f.Until.IsZero() {
		conds = append(conds, "updated_at < ?")
		args = append(args, millis(f.Until))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ErrMessageNotFound is returned by MessageAdmin when the ID names no message.
var ErrMessageNotFound = errors.New("no such message")

// AdminRecipient is one recipient of a message as the administration commands
// show it. At is the time the delivery entered State.
type AdminRecipient struct {
	User, Name string
	State      string
	At         time.Time
}

// AdminMessage is a message as the administration commands show it. Room is set
// for a room message; ToUser and ToName are set for a message addressed to a
// session. Deliveries is nil unless the caller asked for them, and is empty
// (not nil) when it asked and the message has no recipient. Body is set by
// MessageAdmin.
type AdminMessage struct {
	ID                 int64
	SentAt             time.Time
	FromUser, FromName string
	Room               string
	ToUser, ToName     string
	Scope              Scope
	Queued             int
	Delivered          int
	Acked              int
	Expired            int
	Deliveries         []AdminRecipient
	Body               string
}

// MessageQuery is the arguments of ListMessages. Limit is at least 1. BeforeID
// of 0 does not filter. Details loads one row per recipient.
type MessageQuery struct {
	Filter   RecordFilter
	State    string
	BeforeID int64
	Limit    int
	Details  bool
}

const adminMessageFrom = `
FROM messages m
JOIN sessions s ON s.id = m.sender_session_id
JOIN users u ON u.id = s.user_id
LEFT JOIN rooms r ON r.id = m.room_id
LEFT JOIN sessions ts ON ts.id = m.to_session_id
LEFT JOIN users tu ON tu.id = ts.user_id`

const adminMessageCols = `m.id, m.sent_at, u.name, s.name, coalesce(r.name, ''), coalesce(tu.name, ''), coalesce(ts.name, ''),
	m.scope, m.scope_id,
	(SELECT count(*) FROM deliveries d WHERE d.message_id = m.id AND d.state = 'queued'),
	(SELECT count(*) FROM deliveries d WHERE d.message_id = m.id AND d.state = 'delivered'),
	(SELECT count(*) FROM deliveries d WHERE d.message_id = m.id AND d.state = 'acked'),
	(SELECT count(*) FROM deliveries d WHERE d.message_id = m.id AND d.state = 'expired')`

// ListMessages returns the messages that match q, newest first (ID descending),
// at most q.Limit of them, read in one transaction. It does not read the body.
func (s *Store) ListMessages(ctx context.Context, q MessageQuery) ([]AdminMessage, error) {
	if err := q.Filter.validate(); err != nil {
		return nil, err
	}
	if q.State != "" && !slices.Contains(DeliveryStates, q.State) {
		return nil, fmt.Errorf("unknown delivery state %q", q.State)
	}
	if q.Limit < 1 {
		return nil, fmt.Errorf("limit must be at least 1")
	}
	out := []AdminMessage{}
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		where, args := q.Filter.messageWhere(q.State, q.BeforeID)
		args = append(args, q.Limit)
		rows, err := tx.QueryContext(ctx, "SELECT "+adminMessageCols+adminMessageFrom+where+" ORDER BY m.id DESC LIMIT ?", args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanAdminMessage(rows, nil)
			if err != nil {
				return err
			}
			out = append(out, m)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if q.Details && len(out) > 0 {
			return attachRecipients(ctx, tx, out)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return out, nil
}

// MessageAdmin returns one message with its body and one row per recipient, read
// in one transaction. An unknown ID fails with ErrMessageNotFound.
func (s *Store) MessageAdmin(ctx context.Context, id int64) (AdminMessage, error) {
	var m AdminMessage
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		var body string
		row := tx.QueryRowContext(ctx, "SELECT "+adminMessageCols+", m.body"+adminMessageFrom+" WHERE m.id = ?", id)
		var err error
		m, err = scanAdminMessage(row, &body)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMessageNotFound
		}
		if err != nil {
			return err
		}
		m.Body = body
		got, err := recipientsByMessage(ctx, tx, []int64{id})
		if err != nil {
			return err
		}
		m.Deliveries = got[id]
		if m.Deliveries == nil {
			m.Deliveries = []AdminRecipient{}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			return AdminMessage{}, fmt.Errorf("message %d: %w", id, ErrMessageNotFound)
		}
		return AdminMessage{}, fmt.Errorf("message %d: %w", id, err)
	}
	return m, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAdminMessage(sc scanner, body *string) (AdminMessage, error) {
	var m AdminMessage
	var sent int64
	dest := []any{&m.ID, &sent, &m.FromUser, &m.FromName, &m.Room, &m.ToUser, &m.ToName, &m.Scope.Kind, &m.Scope.ID,
		&m.Queued, &m.Delivered, &m.Acked, &m.Expired}
	if body != nil {
		dest = append(dest, body)
	}
	if err := sc.Scan(dest...); err != nil {
		return AdminMessage{}, err
	}
	m.SentAt = fromMillis(sent)
	return m, nil
}

func attachRecipients(ctx context.Context, tx *sql.Tx, msgs []AdminMessage) error {
	ids := make([]int64, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	got, err := recipientsByMessage(ctx, tx, ids)
	if err != nil {
		return err
	}
	for i := range msgs {
		msgs[i].Deliveries = got[msgs[i].ID]
		if msgs[i].Deliveries == nil {
			msgs[i].Deliveries = []AdminRecipient{}
		}
	}
	return nil
}

// recipientsByMessage returns the recipients of the messages, in user/name order.
func recipientsByMessage(ctx context.Context, tx *sql.Tx, ids []int64) (map[int64][]AdminRecipient, error) {
	out := map[int64][]AdminRecipient{}
	if len(ids) == 0 {
		return out, nil
	}
	in, args := idList(ids)
	rows, err := tx.QueryContext(ctx, `
		SELECT d.message_id, du.name, ds.name, d.state,
			coalesce(d.queued_at, 0), coalesce(d.delivered_at, 0), coalesce(d.acked_at, 0), coalesce(d.expired_at, 0)
		FROM deliveries d
		JOIN sessions ds ON ds.id = d.recipient_session_id
		JOIN users du ON du.id = ds.user_id
		WHERE d.message_id IN (`+in+`)
		ORDER BY d.message_id, du.name || '/' || ds.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, queued, delivered, acked, expired int64
		var r AdminRecipient
		if err := rows.Scan(&id, &r.User, &r.Name, &r.State, &queued, &delivered, &acked, &expired); err != nil {
			return nil, err
		}
		r.At = recipientStateTime(r.State, queued, delivered, acked, expired)
		out[id] = append(out[id], r)
	}
	return out, rows.Err()
}

// recipientStateTime returns the time a delivery entered its current state.
func recipientStateTime(state string, queued, delivered, acked, expired int64) time.Time {
	ms := queued
	switch state {
	case StateDelivered:
		ms = delivered
	case StateAcked:
		ms = acked
	case StateExpired:
		ms = expired
	}
	return optionalTime(ms)
}
