package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Names fixed while requests are not authenticated and rooms cannot be created.
const (
	// LocalUser is the one user every connection acts as. Token authentication will replace it.
	LocalUser = "local"
	// LoungeRoom is the room every space starts with.
	LoungeRoom = "lounge"
	// ScopeUser is the space visible to the sessions of one user.
	ScopeUser = "user"

	// localTokenHash marks a user without a token. Token hashes are hexadecimal digests, so no
	// token can match it.
	localTokenHash = "none:" + LocalUser
)

// Errors the tools turn into guidance for agents.
var (
	ErrNoSuchSession = errors.New("no such session")
	ErrNoSuchRoom    = errors.New("no such room")
	ErrSendToSelf    = errors.New("the destination is the sending session")
)

// Session is a named unit of work of one user. It outlives connections: joining again with the
// same name continues it, including its unread messages.
type Session struct {
	ID         int64
	UserID     int64
	Name       string
	Agent      string
	LastSeenAt time.Time
}

// Sent describes an accepted message.
type Sent struct {
	MessageID  int64
	SentAt     time.Time
	Recipients []string // session names
	// RecipientIDs are the session IDs of Recipients, in the same order.
	RecipientIDs []int64
}

// InboxMessage is a message handed to its recipient.
type InboxMessage struct {
	MessageID int64
	From      string // sender's session name
	FromAgent string
	Scope     string
	Room      string // empty for a message addressed to the session
	Body      string
	SentAt    time.Time
}

// JoinParams are the arguments of JoinSession.
type JoinParams struct {
	UserID int64
	// Name is the session name. When empty, a free name is generated.
	Name string
	// Agent replaces the recorded agent name when not nil.
	Agent *string
}

func millis(t time.Time) int64      { return t.UnixMilli() }
func fromMillis(ms int64) time.Time { return time.UnixMilli(ms).UTC() }

// EnsureLocalUser creates the local user and the lounge room of its space when missing and
// returns the user's ID.
func (s *Store) EnsureLocalUser(ctx context.Context) (int64, error) {
	var id int64
	err := s.tx(ctx, func(tx *sql.Tx) error {
		now := millis(s.now())
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO users (name, token_hash, created_at) VALUES (?, ?, ?) ON CONFLICT (name) DO NOTHING",
			LocalUser, localTokenHash, now); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE name = ?", LocalUser).Scan(&id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			"INSERT INTO rooms (scope, user_id, name, created_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING",
			ScopeUser, id, LoungeRoom, now)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("create the local user: %w", err)
	}
	return id, nil
}

// JoinSession returns the user's session named p.Name, creating it when missing. resumed
// reports that the session existed.
func (s *Store) JoinSession(ctx context.Context, p JoinParams) (sess Session, resumed bool, err error) {
	err = s.tx(ctx, func(tx *sql.Tx) error {
		now := s.now()
		name := p.Name
		if name == "" {
			if name, err = freeSessionName(ctx, tx, p.UserID); err != nil {
				return err
			}
		}
		sess = Session{UserID: p.UserID, Name: name, LastSeenAt: now.UTC().Truncate(time.Millisecond)}
		err := tx.QueryRowContext(ctx, "SELECT id, agent FROM sessions WHERE user_id = ? AND name = ?", p.UserID, name).
			Scan(&sess.ID, &sess.Agent)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if p.Agent != nil {
				sess.Agent = *p.Agent
			}
			res, err := tx.ExecContext(ctx,
				"INSERT INTO sessions (user_id, name, agent, created_at, last_seen_at) VALUES (?, ?, ?, ?, ?)",
				p.UserID, name, sess.Agent, millis(now), millis(now))
			if err != nil {
				return err
			}
			sess.ID, err = res.LastInsertId()
			return err
		case err != nil:
			return err
		}
		resumed = true
		if p.Agent != nil {
			sess.Agent = *p.Agent
		}
		_, err = tx.ExecContext(ctx, "UPDATE sessions SET agent = ?, last_seen_at = ? WHERE id = ?", sess.Agent, millis(now), sess.ID)
		return err
	})
	if err != nil {
		return Session{}, false, fmt.Errorf("join session: %w", err)
	}
	return sess, resumed, nil
}

// freeSessionName generates a session name the user does not use yet.
func freeSessionName(ctx context.Context, tx *sql.Tx, userID int64) (string, error) {
	for range 100 {
		var b [2]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		name := fmt.Sprintf("session-%04x", b)
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM sessions WHERE user_id = ? AND name = ?", userID, name).Scan(&n); err != nil {
			return "", err
		}
		if n == 0 {
			return name, nil
		}
	}
	return "", errors.New("no free session name found; pass a name")
}

// Seen records that the session was used now and returns its number of unread messages
// (deliveries still queued).
func (s *Store) Seen(ctx context.Context, sessionID int64) (unread int, err error) {
	if _, err := s.db.ExecContext(ctx, "UPDATE sessions SET last_seen_at = ? WHERE id = ?", millis(s.now()), sessionID); err != nil {
		return 0, fmt.Errorf("update session: %w", err)
	}
	err = s.db.QueryRowContext(ctx,
		"SELECT count(*) FROM deliveries WHERE recipient_session_id = ? AND state = 'queued'", sessionID).Scan(&unread)
	if err != nil {
		return 0, fmt.Errorf("count unread messages: %w", err)
	}
	return unread, nil
}

// ListSessions returns up to limit sessions of the user, most recently used first. truncated
// reports that the user has more.
func (s *Store) ListSessions(ctx context.Context, userID int64, limit int) (sessions []Session, truncated bool, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, agent, last_seen_at FROM sessions WHERE user_id = ? ORDER BY last_seen_at DESC, id DESC LIMIT ?",
		userID, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		sess := Session{UserID: userID}
		var seen int64
		if err := rows.Scan(&sess.ID, &sess.Name, &sess.Agent, &seen); err != nil {
			return nil, false, fmt.Errorf("list sessions: %w", err)
		}
		sess.LastSeenAt = fromMillis(seen)
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("list sessions: %w", err)
	}
	if len(sessions) > limit {
		return sessions[:limit], true, nil
	}
	return sessions, false, nil
}

// ListRooms returns the names of the rooms in the user's own space.
func (s *Store) ListRooms(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name FROM rooms WHERE scope = ? AND user_id = ? ORDER BY name", ScopeUser, userID)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("list rooms: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// SendToSession queues a message for the sender's user's session named to.
func (s *Store) SendToSession(ctx context.Context, sender Session, to, body string) (Sent, error) {
	var sent Sent
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var recipient int64
		err := tx.QueryRowContext(ctx, "SELECT id FROM sessions WHERE user_id = ? AND name = ?", sender.UserID, to).Scan(&recipient)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNoSuchSession
		case err != nil:
			return err
		case recipient == sender.ID:
			return ErrSendToSelf
		}
		now := s.now()
		res, err := tx.ExecContext(ctx,
			"INSERT INTO messages (scope, sender_session_id, to_session_id, body, sent_at) VALUES (?, ?, ?, ?, ?)",
			ScopeUser, sender.ID, recipient, body, millis(now))
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO deliveries (message_id, recipient_session_id, queued_at) VALUES (?, ?, ?)", id, recipient, millis(now)); err != nil {
			return err
		}
		sent = Sent{MessageID: id, SentAt: fromMillis(millis(now)), Recipients: []string{to}, RecipientIDs: []int64{recipient}}
		return nil
	})
	if err != nil {
		return Sent{}, fmt.Errorf("send to session %q: %w", to, err)
	}
	return sent, nil
}

// SendToRoom queues a message for every session of the sender's user that exists now, except
// the sender. Rooms keep no member list: a session of the space is a member of its rooms, whether
// it is connected or not.
func (s *Store) SendToRoom(ctx context.Context, sender Session, room, body string) (Sent, error) {
	var sent Sent
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var roomID int64
		err := tx.QueryRowContext(ctx, "SELECT id FROM rooms WHERE scope = ? AND user_id = ? AND name = ?", ScopeUser, sender.UserID, room).Scan(&roomID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNoSuchRoom
		case err != nil:
			return err
		}
		now := s.now()
		res, err := tx.ExecContext(ctx,
			"INSERT INTO messages (scope, sender_session_id, room_id, body, sent_at) VALUES (?, ?, ?, ?, ?)",
			ScopeUser, sender.ID, roomID, body, millis(now))
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO deliveries (message_id, recipient_session_id, queued_at)
			SELECT ?, id, ? FROM sessions WHERE user_id = ? AND id <> ?`, id, millis(now), sender.UserID, sender.ID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `
			SELECT s.id, s.name FROM deliveries d JOIN sessions s ON s.id = d.recipient_session_id
			WHERE d.message_id = ? ORDER BY s.name`, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		recipients := []string{}
		var recipientIDs []int64
		for rows.Next() {
			var recipientID int64
			var name string
			if err := rows.Scan(&recipientID, &name); err != nil {
				return err
			}
			recipients = append(recipients, name)
			recipientIDs = append(recipientIDs, recipientID)
		}
		sent = Sent{MessageID: id, SentAt: fromMillis(millis(now)), Recipients: recipients, RecipientIDs: recipientIDs}
		return rows.Err()
	})
	if err != nil {
		return Sent{}, fmt.Errorf("send to room %q: %w", room, err)
	}
	return sent, nil
}

// FetchInbox hands the session its oldest unread messages, up to limit, and marks them
// delivered. Selecting and marking share one write transaction, so concurrent calls never hand
// out the same message twice.
func (s *Store) FetchInbox(ctx context.Context, sessionID int64, limit int) ([]InboxMessage, error) {
	msgs := []InboxMessage{}
	err := s.tx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT m.id, s.name, s.agent, m.scope, coalesce(r.name, ''), m.body, m.sent_at
			FROM deliveries d
			JOIN messages m ON m.id = d.message_id
			JOIN sessions s ON s.id = m.sender_session_id
			LEFT JOIN rooms r ON r.id = m.room_id
			WHERE d.recipient_session_id = ? AND d.state = 'queued'
			ORDER BY d.message_id LIMIT ?`, sessionID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m InboxMessage
			var sentAt int64
			if err := rows.Scan(&m.MessageID, &m.From, &m.FromAgent, &m.Scope, &m.Room, &m.Body, &sentAt); err != nil {
				return err
			}
			m.SentAt = fromMillis(sentAt)
			msgs = append(msgs, m)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		now := millis(s.now())
		for _, m := range msgs {
			if _, err := tx.ExecContext(ctx, `
				UPDATE deliveries SET state = 'delivered', delivered_at = ?
				WHERE message_id = ? AND recipient_session_id = ? AND state = 'queued'`, now, m.MessageID, sessionID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetch inbox: %w", err)
	}
	return msgs, nil
}
