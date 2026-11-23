package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"
)

// The functions in this file serve the administration commands (session, memory, message, export), which open
// the database file directly, possibly while a server uses it. They read in one transaction, so the
// result is the database at one moment.

// ErrNoSessionToRemove is returned by CountSessionData and RemoveSession for a session that
// does not exist.
var ErrNoSessionToRemove = errors.New("no such session")

// SessionSummary is a session as listed by AllSessions.
type SessionSummary struct {
	Session
	// Unread is the number of its queued deliveries.
	Unread int
}

// AllSessions returns the sessions of every user, or of the user named user when it is not
// empty, in the order of their addresses (user/name). An unknown user fails with
// ErrNoSuchUser.
func (s *Store) AllSessions(ctx context.Context, user string) ([]SessionSummary, error) {
	out := []SessionSummary{}
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		cond, args := "1", []any{}
		if user != "" {
			var n int
			if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE name = ?", user).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				return ErrNoSuchUser
			}
			cond, args = "u.name = ?", []any{user}
		}
		rows, err := tx.QueryContext(ctx, "SELECT "+sessionColumns+`,
			(SELECT count(*) FROM deliveries d WHERE d.recipient_session_id = s.id AND d.state = 'queued')`+
			sessionFrom+"WHERE "+cond+" ORDER BY u.name || '/' || s.name", args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var sum SessionSummary
			var seen int64
			if err := rows.Scan(&sum.ID, &sum.UserID, &sum.UserName, &sum.Name, &sum.Agent, &sum.OSUser, &sum.Machine,
				&sum.Origin, &sum.Folder, &sum.Repository, &sum.Branch, &seen, &sum.Unread); err != nil {
				return err
			}
			sum.LastSeenAt = fromMillis(seen)
			out = append(out, sum)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return out, nil
}

// SessionData counts what deleting a session deletes: the messages it sent or that were
// addressed to it, and the deliveries of those messages and to the session.
type SessionData struct {
	Messages   int
	Deliveries int
}

const sessionByAddress = "SELECT s.id" + sessionFrom + "WHERE u.name = ? AND s.name = ?"

// CountSessionData counts what RemoveSession would delete.
func (s *Store) CountSessionData(ctx context.Context, user, name string) (SessionData, error) {
	var data SessionData
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		var err error
		data, err = sessionData(ctx, tx, user, name)
		return err
	})
	if err != nil {
		return SessionData{}, fmt.Errorf("session %s/%s: %w", user, name, err)
	}
	return data, nil
}

func sessionData(ctx context.Context, tx *sql.Tx, user, name string) (SessionData, error) {
	d, err := countSessionData(ctx, tx, sessionByAddress, user, name)
	if err != nil {
		return SessionData{}, err
	}
	if d.Sessions == 0 {
		return SessionData{}, ErrNoSessionToRemove
	}
	return SessionData{Messages: d.Messages, Deliveries: d.Deliveries}, nil
}

// RemoveSession deletes the session user/name with the messages it sent or that were addressed
// to it and their deliveries, and every delivery to it, and returns what it deleted. A
// connection joined as the session gets an error at its next tool call.
func (s *Store) RemoveSession(ctx context.Context, user, name string) (SessionData, error) {
	var data SessionData
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var err error
		if data, err = sessionData(ctx, tx, user, name); err != nil {
			return err
		}
		// Foreign keys cascade from the session to its messages and deliveries.
		_, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE id IN ("+sessionByAddress+")", user, name)
		return err
	})
	if err != nil {
		return SessionData{}, fmt.Errorf("remove session %s/%s: %w", user, name, err)
	}
	return data, nil
}

// ExportMessage is a message as Export reads it. Sessions are named by user and name.
type ExportMessage struct {
	ID    int64
	Scope Scope
	// Room is the room name of a room message; ToUser and ToName are the recipient of a
	// message addressed to a session. Exactly one of Room and ToName is set.
	Room           string
	ToUser, ToName string
	FromUser       string
	From           string
	// FromAgent is the agent name the sender declared when it sent the message.
	FromAgent  string
	SentAt     time.Time
	Body       string
	Deliveries []ExportDelivery
}

// ExportDelivery is the delivery of an exported message to one recipient. A time is zero when
// the delivery has not reached that state.
type ExportDelivery struct {
	User, Name  string
	State       string
	QueuedAt    time.Time
	DeliveredAt time.Time
	AckedAt     time.Time
	ExpiredAt   time.Time
}

// Export reads every message with its deliveries, then every memory, in one read transaction,
// so that the result is the content of the database at one moment, and passes them to
// onMessage and onMemory one at a time. onMemory may be nil, which skips the memories.
// Memories come in the order scope, scope ID, ID, with their tags; their retention is left
// unset, since it depends on the moment they are read. Messages
// come grouped by conversation (the room of one scope, or the pair of sessions of a message
// addressed to a session, in one scope) in the order scope, scope ID, rooms before pairs, room
// name or pair, and oldest first within a conversation. Deliveries are in the order of the
// recipients' user/name. When a callback fails, Export stops and returns its error.
func (s *Store) Export(ctx context.Context, onMessage func(ExportMessage) error, onMemory func(Memory) error) error {
	return s.ExportSelected(ctx, RecordFilter{}, onMessage, onMemory)
}

// ExportSelected reads like Export, keeping the messages and memories that match f.
// The zero filter reads every record, with the same queries and the same order as
// Export. A message is kept or dropped as a whole, with all of its deliveries.
func (s *Store) ExportSelected(ctx context.Context, f RecordFilter, onMessage func(ExportMessage) error, onMemory func(Memory) error) error {
	if err := f.validate(); err != nil {
		return err
	}
	return s.readTx(ctx, func(tx *sql.Tx) error {
		var err error
		if f.IsZero() {
			err = exportMessages(ctx, tx, onMessage)
		} else {
			err = exportMessagesFiltered(ctx, tx, f, onMessage)
		}
		if err != nil {
			return err
		}
		if onMemory == nil {
			return nil
		}
		if f.IsZero() {
			return exportMemories(ctx, tx, onMemory)
		}
		return exportMemoriesFiltered(ctx, tx, f, onMemory)
	})
}

// exportMemories passes every memory to fn, in the order scope, scope ID, ID.
func exportMemories(ctx context.Context, tx *sql.Tx, fn func(Memory) error) error {
	return exportMemoryQuery(ctx, tx, "SELECT id FROM memories ORDER BY scope, scope_id, id", nil, fn)
}

// exportMemoriesFiltered passes the memories that match f, in the same order.
func exportMemoriesFiltered(ctx context.Context, tx *sql.Tx, f RecordFilter, fn func(Memory) error) error {
	where, args := f.memoryWhere()
	return exportMemoryQuery(ctx, tx, "SELECT id FROM memories"+where+" ORDER BY scope, scope_id, id", args, fn)
}

func exportMemoryQuery(ctx context.Context, tx *sql.Tx, query string, args []any, fn func(Memory) error) error {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for chunk := range slices.Chunk(ids, 500) {
		ms, err := loadMemories(ctx, tx, chunk)
		if err != nil {
			return err
		}
		for _, id := range chunk {
			if err := fn(ms[id]); err != nil {
				return err
			}
		}
	}
	return nil
}

// exportMessageSQL is the unfiltered read. ExportSelected with no filter runs this
// text, so a filtered read cannot change what an unfiltered export returns.
const exportMessageSQL = `
			WITH msg AS (
				SELECT m.id, m.scope, m.scope_id, coalesce(r.name, '') AS room,
					coalesce(tu.name, '') AS to_user, coalesce(ts.name, '') AS to_name,
					u.name AS from_user, s.name AS from_name, m.sender_agent, m.sent_at, m.body,
					CASE WHEN m.room_id IS NULL
						THEN min(u.name || '/' || s.name, tu.name || '/' || ts.name) || char(10) ||
							max(u.name || '/' || s.name, tu.name || '/' || ts.name)
						ELSE '' END AS pair
				FROM messages m
				JOIN sessions s ON s.id = m.sender_session_id
				JOIN users u ON u.id = s.user_id
				LEFT JOIN rooms r ON r.id = m.room_id
				LEFT JOIN sessions ts ON ts.id = m.to_session_id
				LEFT JOIN users tu ON tu.id = ts.user_id
			)
			SELECT msg.id, msg.scope, msg.scope_id, msg.room, msg.to_user, msg.to_name, msg.from_user, msg.from_name,
				msg.sender_agent, msg.sent_at, msg.body,
				coalesce(du.name, ''), coalesce(ds.name, ''), coalesce(d.state, ''), coalesce(d.queued_at, 0),
				coalesce(d.delivered_at, 0), coalesce(d.acked_at, 0), coalesce(d.expired_at, 0)
			FROM msg
			LEFT JOIN deliveries d ON d.message_id = msg.id
			LEFT JOIN sessions ds ON ds.id = d.recipient_session_id
			LEFT JOIN users du ON du.id = ds.user_id
			ORDER BY msg.scope, msg.scope_id, msg.pair, msg.room, msg.id, du.name || '/' || ds.name`

// exportMessages passes every message to fn; see Export.
func exportMessages(ctx context.Context, tx *sql.Tx, fn func(ExportMessage) error) error {
	return queryExportMessages(ctx, tx, exportMessageSQL, nil, fn)
}

// exportMessagesFiltered passes the messages that match f, in the same order, each
// with all of its deliveries. The condition is applied inside the message set, so
// the grouping and the delivery order stay those of exportMessageSQL.
func exportMessagesFiltered(ctx context.Context, tx *sql.Tx, f RecordFilter, fn func(ExportMessage) error) error {
	where, args := f.messageWhere("", 0)
	q := `
			WITH msg AS (
				SELECT m.id, m.scope, m.scope_id, coalesce(r.name, '') AS room,
					coalesce(tu.name, '') AS to_user, coalesce(ts.name, '') AS to_name,
					u.name AS from_user, s.name AS from_name, m.sender_agent, m.sent_at, m.body,
					CASE WHEN m.room_id IS NULL
						THEN min(u.name || '/' || s.name, tu.name || '/' || ts.name) || char(10) ||
							max(u.name || '/' || s.name, tu.name || '/' || ts.name)
						ELSE '' END AS pair
				FROM messages m
				JOIN sessions s ON s.id = m.sender_session_id
				JOIN users u ON u.id = s.user_id
				LEFT JOIN rooms r ON r.id = m.room_id
				LEFT JOIN sessions ts ON ts.id = m.to_session_id
				LEFT JOIN users tu ON tu.id = ts.user_id` + where + `
			)
			SELECT msg.id, msg.scope, msg.scope_id, msg.room, msg.to_user, msg.to_name, msg.from_user, msg.from_name,
				msg.sender_agent, msg.sent_at, msg.body,
				coalesce(du.name, ''), coalesce(ds.name, ''), coalesce(d.state, ''), coalesce(d.queued_at, 0),
				coalesce(d.delivered_at, 0), coalesce(d.acked_at, 0), coalesce(d.expired_at, 0)
			FROM msg
			LEFT JOIN deliveries d ON d.message_id = msg.id
			LEFT JOIN sessions ds ON ds.id = d.recipient_session_id
			LEFT JOIN users du ON du.id = ds.user_id
			ORDER BY msg.scope, msg.scope_id, msg.pair, msg.room, msg.id, du.name || '/' || ds.name`
	return queryExportMessages(ctx, tx, q, args, fn)
}

func queryExportMessages(ctx context.Context, tx *sql.Tx, query string, args []any, fn func(ExportMessage) error) error {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	var cur *ExportMessage
	for rows.Next() {
		var m ExportMessage
		var d ExportDelivery
		var sentAt, queued, delivered, acked, expired int64
		if err := rows.Scan(&m.ID, &m.Scope.Kind, &m.Scope.ID, &m.Room, &m.ToUser, &m.ToName, &m.FromUser, &m.From,
			&m.FromAgent, &sentAt, &m.Body, &d.User, &d.Name, &d.State, &queued, &delivered, &acked, &expired); err != nil {
			return err
		}
		if cur == nil || cur.ID != m.ID {
			if cur != nil {
				if err := fn(*cur); err != nil {
					return err
				}
			}
			m.SentAt = fromMillis(sentAt)
			m.Deliveries = []ExportDelivery{}
			cur = &m
		}
		if d.State != "" {
			d.QueuedAt, d.DeliveredAt, d.AckedAt, d.ExpiredAt =
				optionalTime(queued), optionalTime(delivered), optionalTime(acked), optionalTime(expired)
			cur.Deliveries = append(cur.Deliveries, d)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if cur != nil {
		return fn(*cur)
	}
	return nil
}
