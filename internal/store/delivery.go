package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// A delivery is a message together with one recipient session. Its state moves:
//
//	queued    accepted by the server, not yet fetched by the recipient (Send)
//	delivered fetched by the recipient's session (FetchInbox)
//	acked     the recipient recorded that it finished handling the message (Ack)
//	expired   not fetched before the expiry period ended (Cleanup)
//
// delivered and acked are separate states so that a recipient that fetched a message and then
// lost its context (a new conversation, a compacted history) can still find the messages it
// has not finished (History with Unacked). For the same reason a delivered message never
// expires: only queued deliveries do, and a delivered one stays delivered until it is acked
// or its message is deleted.

// Delivery states.
const (
	StateQueued    = "queued"
	StateDelivered = "delivered"
	StateAcked     = "acked"
	StateExpired   = "expired"
)

// DeliveryStates lists the states in the order they are reported.
var DeliveryStates = []string{StateQueued, StateDelivered, StateAcked, StateExpired}

// ErrNoSuchMessage is returned by MessageStatus for a message the session did not send. A
// missing message, a deleted one and one sent by another session share the error on purpose:
// a session must not learn whether another session's message exists.
var ErrNoSuchMessage = errors.New("no message with that ID was sent by this session")

// ErrSessionGone is returned for a session that no longer exists: it was deleted (by the
// cleanup of unused sessions, by session remove, or with its user).
var ErrSessionGone = errors.New("the session was deleted")

// Ack records that the session finished handling the given messages: each delivered delivery
// of these messages to the session becomes acked, with the time. acked lists the IDs whose
// delivery to the session is acked now, including those acked before (their time is kept), so
// that repeating a call gives the same result. notFound lists the other IDs: a missing message,
// one not addressed to the session, one still queued (not fetched) and one expired, without
// telling which. Both keep the order of ids, without duplicates.
func (s *Store) Ack(ctx context.Context, sessionID int64, ids []int64) (acked, notFound []int64, err error) {
	acked, notFound = []int64{}, []int64{}
	err = s.tx(ctx, func(tx *sql.Tx) error {
		now := millis(s.now())
		seen := map[int64]bool{}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			if _, err := tx.ExecContext(ctx, `
				UPDATE deliveries SET state = 'acked', acked_at = ?
				WHERE message_id = ? AND recipient_session_id = ? AND state = 'delivered'`, now, id, sessionID); err != nil {
				return err
			}
			var n int
			if err := tx.QueryRowContext(ctx,
				"SELECT count(*) FROM deliveries WHERE message_id = ? AND recipient_session_id = ? AND state = 'acked'",
				id, sessionID).Scan(&n); err != nil {
				return err
			}
			if n > 0 {
				acked = append(acked, id)
			} else {
				notFound = append(notFound, id)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ack: %w", err)
	}
	return acked, notFound, nil
}

// RecipientStatus is the delivery of a message to one recipient. A time is zero when the
// delivery has not reached that state.
type RecipientStatus struct {
	Member
	State       string
	DeliveredAt time.Time
	AckedAt     time.Time
	ExpiredAt   time.Time
}

// MessageStatus is the delivery state of a message, as its sender sees it.
type MessageStatus struct {
	MessageID int64
	SentAt    time.Time
	Scope     Scope
	Room      string // empty for a message addressed to a session
	// Counts holds the number of deliveries in each of DeliveryStates.
	Counts map[string]int
	// Recipients are the deliveries in the order of the recipients' addresses as the sender
	// sees them (see package tools), up to the limit.
	Recipients []RecipientStatus
	// Truncated reports that more recipients exist than are listed.
	Truncated bool
}

// MessageStatus returns the delivery state of the message messageID sent by the session
// senderID, with up to limit recipients. It changes nothing. A message the session did not
// send fails with ErrNoSuchMessage, whether it exists or not. Deliveries to deleted sessions
// are gone, so they are neither counted nor listed.
func (s *Store) MessageStatus(ctx context.Context, senderID, messageID int64, limit int) (MessageStatus, error) {
	st := MessageStatus{MessageID: messageID, Counts: map[string]int{}, Recipients: []RecipientStatus{}}
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		var sentAt int64
		var viewer string
		err := tx.QueryRowContext(ctx, `
			SELECT m.sent_at, m.scope, m.scope_id, coalesce(r.name, ''), u.name
			FROM messages m
			JOIN sessions s ON s.id = m.sender_session_id
			JOIN users u ON u.id = s.user_id
			LEFT JOIN rooms r ON r.id = m.room_id
			WHERE m.id = ? AND m.sender_session_id = ?`, messageID, senderID).
			Scan(&sentAt, &st.Scope.Kind, &st.Scope.ID, &st.Room, &viewer)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNoSuchMessage
		case err != nil:
			return err
		}
		st.SentAt = fromMillis(sentAt)
		for _, state := range DeliveryStates {
			st.Counts[state] = 0
		}
		rows, err := tx.QueryContext(ctx, "SELECT state, count(*) FROM deliveries WHERE message_id = ? GROUP BY state", messageID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var state string
			var n int
			if err := rows.Scan(&state, &n); err != nil {
				rows.Close()
				return err
			}
			st.Counts[state] = n
		}
		if err := rows.Close(); err != nil {
			return err
		}
		// The order of the addresses: the session name for a session of the sender's user,
		// user/name for another user's (see package tools).
		rows, err = tx.QueryContext(ctx, `
			SELECT s.id, u.name, s.name, d.state, coalesce(d.delivered_at, 0), coalesce(d.acked_at, 0), coalesce(d.expired_at, 0)
			FROM deliveries d
			JOIN sessions s ON s.id = d.recipient_session_id
			JOIN users u ON u.id = s.user_id
			WHERE d.message_id = ?
			ORDER BY CASE WHEN u.name = ? THEN s.name ELSE u.name || '/' || s.name END
			LIMIT ?`, messageID, viewer, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r RecipientStatus
			var delivered, acked, expired int64
			if err := rows.Scan(&r.ID, &r.User, &r.Name, &r.State, &delivered, &acked, &expired); err != nil {
				return err
			}
			r.DeliveredAt, r.AckedAt, r.ExpiredAt = optionalTime(delivered), optionalTime(acked), optionalTime(expired)
			st.Recipients = append(st.Recipients, r)
		}
		return rows.Err()
	})
	if err != nil {
		return MessageStatus{}, fmt.Errorf("message status: %w", err)
	}
	if len(st.Recipients) > limit {
		st.Recipients, st.Truncated = st.Recipients[:limit], true
	}
	return st, nil
}

// optionalTime converts a stored time where 0 means none.
func optionalTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return fromMillis(ms)
}

// HistoryQuery selects the messages History returns. Zero values do not filter.
type HistoryQuery struct {
	// Scope keeps the messages of one scope kind.
	Scope string
	// Room keeps the messages to the room of that name.
	Room string
	// WithUser and WithName keep the messages between the session and that session, in either
	// direction, addressed to a session (not to a room). Set both or neither.
	WithUser, WithName string
	// Unacked keeps the messages delivered to the session and not acked.
	Unacked bool
	// Before keeps the messages with a smaller ID (older ones).
	Before int64
	// Limit is the most messages to return; it must be positive.
	Limit int
}

// HistoryMessage is a message as History returns it.
type HistoryMessage struct {
	InboxMessage
	// Sent reports that the session sent the message.
	Sent bool
	// Others reports a room message the session neither sent nor received: a message of a room
	// of one of its scopes, posted before it entered the range or to a room it did not join.
	// A message that is neither Sent nor Others was delivered to the session.
	Others bool
	// ToUser and ToName are the recipient of a message addressed to a session; empty for a
	// room message.
	ToUser, ToName string
	// State is the state of the delivery to the session, for a received message.
	State string
}

// History returns the messages of the session sessionID, newest first, up to q.Limit, and
// whether older ones remain. It changes nothing.
//
// The session reads the messages it sent, the messages delivered to it (delivered, acked or
// expired), and every message of the rooms of the scopes it can use now, whether it is a member
// or not and whether it was in the range when the message was sent: information on the server
// is shared among the sessions of a scope, and room membership only decides what becomes unread.
// Messages addressed to other sessions are not returned. Messages queued for the session are
// not returned: they are read with FetchInbox, which records that they were delivered.
//
// A WithUser and WithName that name no session give an empty list, the same as a session
// without messages, so that the result does not tell whether another user's session exists.
func (s *Store) History(ctx context.Context, sessionID int64, q HistoryQuery) (msgs []HistoryMessage, more bool, err error) {
	msgs = []HistoryMessage{}
	err = s.readTx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		// The rooms of the scopes the session can use now.
		var roomConds []string
		var roomArgs []any
		for _, sc := range ScopesOf(sess) {
			userID, repository := roomColumns(sess, sc)
			roomConds = append(roomConds, "(r2.scope = ? AND coalesce(r2.user_id, 0) = coalesce(?, 0) AND r2.repository = ?)")
			roomArgs = append(roomArgs, sc.Kind, userID, repository)
		}
		var peer int64
		if q.WithName != "" {
			err := tx.QueryRowContext(ctx, "SELECT s.id"+sessionFrom+"WHERE u.name = ? AND s.name = ?", q.WithUser, q.WithName).Scan(&peer)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				return nil // no such session: no messages
			case err != nil:
				return err
			}
		}
		before := q.Before
		if before <= 0 {
			before = 1<<63 - 1
		}
		var conds []string
		// The arguments of the CTE, of the select list and of the join with the session's
		// deliveries come first, in the order of the query text.
		args := []any{sessionID, before, sessionID, before, before}
		args = append(args, roomArgs...)
		args = append(args, sessionID, sessionID, sessionID)
		if q.Scope != "" {
			conds = append(conds, "m.scope = ?")
			args = append(args, q.Scope)
		}
		if q.Room != "" {
			conds = append(conds, "r.name = ?")
			args = append(args, q.Room)
		}
		if q.WithName != "" {
			conds = append(conds, "((m.sender_session_id = ? AND m.to_session_id = ?) OR (m.sender_session_id = ? AND m.to_session_id = ?))")
			args = append(args, sessionID, peer, peer, sessionID)
		}
		if q.Unacked {
			conds = append(conds, "d.state = 'delivered'")
		}
		where := ""
		if len(conds) > 0 {
			where = "WHERE " + strings.Join(conds, " AND ")
		}
		args = append(args, q.Limit+1)
		rows, err := tx.QueryContext(ctx, `
			WITH mine(id) AS (
				SELECT id FROM messages WHERE sender_session_id = ? AND id < ?
				UNION
				SELECT message_id FROM deliveries
				WHERE recipient_session_id = ? AND message_id < ? AND state IN ('delivered', 'acked', 'expired')
				UNION
				SELECT m2.id FROM messages m2 JOIN rooms r2 ON r2.id = m2.room_id
				WHERE m2.id < ? AND (`+strings.Join(roomConds, " OR ")+`)
					AND NOT EXISTS (SELECT 1 FROM deliveries q
						WHERE q.message_id = m2.id AND q.recipient_session_id = ? AND q.state = 'queued')
			)
			SELECT m.id, u.name, s.name, m.sender_agent, m.sender_os_user, m.sender_machine, m.scope, m.scope_id,
				coalesce(r.name, ''), m.body, m.sent_at, coalesce(tu.name, ''), coalesce(ts.name, ''), coalesce(d.state, ''),
				m.sender_session_id = ?
			FROM mine
			JOIN messages m ON m.id = mine.id
			JOIN sessions s ON s.id = m.sender_session_id
			JOIN users u ON u.id = s.user_id
			LEFT JOIN rooms r ON r.id = m.room_id
			LEFT JOIN sessions ts ON ts.id = m.to_session_id
			LEFT JOIN users tu ON tu.id = ts.user_id
			LEFT JOIN deliveries d ON d.message_id = m.id AND d.recipient_session_id = ?
			`+where+`
			ORDER BY m.id DESC LIMIT ?`, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m HistoryMessage
			var sentAt int64
			if err := rows.Scan(&m.MessageID, &m.FromUser, &m.From, &m.FromAgent, &m.FromOSUser, &m.FromMachine,
				&m.Scope.Kind, &m.Scope.ID, &m.Room, &m.Body, &sentAt, &m.ToUser, &m.ToName, &m.State, &m.Sent); err != nil {
				return err
			}
			m.SentAt = fromMillis(sentAt)
			m.Others = !m.Sent && m.State == ""
			msgs = append(msgs, m)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, false, fmt.Errorf("history: %w", err)
	}
	if len(msgs) > q.Limit {
		return msgs[:q.Limit], true, nil
	}
	return msgs, false, nil
}
