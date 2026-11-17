package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/profile"
)

// Names fixed while requests are not authenticated and rooms cannot be created.
const (
	// LocalUser is the one user every connection acts as. Token authentication will replace it.
	LocalUser = "local"
	// LoungeRoom is the room every space starts with.
	LoungeRoom = "lounge"
	// ScopeUser is the space visible to the sessions of one user.
	ScopeUser = "user"
	// ScopeRepository is the space of the sessions that declare the same repository unit,
	// whatever user they belong to.
	ScopeRepository = "repository"

	// localTokenHash marks a user without a token. Token hashes are hexadecimal digests, so no
	// token can match it.
	localTokenHash = "none:" + LocalUser
)

// Errors the tools turn into guidance for agents.
var (
	ErrNoSuchSession = errors.New("no such session")
	ErrNoSuchRoom    = errors.New("no such room")
	ErrSendToSelf    = errors.New("the destination is the sending session")
	// ErrNoRepository is returned for a message to the repository space from a session that
	// declared no repository.
	ErrNoRepository = errors.New("the sending session has no repository")
	// ErrMachineRequired is returned by JoinSession when the session would have a folder
	// without an origin and without a machine.
	ErrMachineRequired = profile.ErrMachineRequired
)

// Session is a named unit of work of one user. It outlives connections: joining again with the
// same name continues it, including its unread messages.
type Session struct {
	ID     int64
	UserID int64
	Name   string
	Agent  string
	Profile
	LastSeenAt time.Time
}

// Profile holds the values a session declared at lounge_join, normalized by package profile,
// and the repository unit decided from them. Empty strings mean no value.
type Profile struct {
	OSUser     string
	Machine    string
	Origin     string
	Folder     string
	Repository string
}

// Location is the place a session works in: the normalized origin and folder.
type Location struct {
	Origin string
	Folder string
}

// Sent describes an accepted message.
type Sent struct {
	MessageID  int64
	SentAt     time.Time
	Recipients []string // session names
	// RecipientIDs are the session IDs of Recipients, in the same order.
	RecipientIDs []int64
	// Scope is the space of the message.
	Scope string
	// Repository is the repository unit of a message to the repository space, else "".
	Repository string
}

// InboxMessage is a message handed to its recipient.
type InboxMessage struct {
	MessageID int64
	From      string // sender's session name
	FromAgent string
	// FromOSUser and FromMachine are the sender's declared values ("" when not declared).
	FromOSUser  string
	FromMachine string
	Scope       string
	Room        string // empty for a message addressed to the session
	// Repository is the repository unit the message was sent to, for scope repository.
	Repository string
	Body       string
	SentAt     time.Time
}

// JoinParams are the arguments of JoinSession.
type JoinParams struct {
	UserID int64
	// Name is the session name. When empty, a free name is generated.
	Name string
	// Agent, OSUser and Machine replace the recorded value when not nil ("" clears it).
	Agent   *string
	OSUser  *string
	Machine *string
	// Location replaces the recorded origin and folder together when not nil.
	Location *Location
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

// JoinSession returns the user's session named p.Name, creating it when missing, and records
// the declared values of p. resumed reports that the session existed.
//
// origin and folder are replaced as one set (p.Location), never one at a time: a session that
// moves from a repository with an origin to a folder without one must not keep the old origin,
// which would decide its repository unit. The unit is decided again from the resulting values;
// a folder without an origin and without a machine fails with ErrMachineRequired and changes
// nothing.
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
		err := tx.QueryRowContext(ctx,
			"SELECT id, agent, os_user, machine, origin, folder FROM sessions WHERE user_id = ? AND name = ?", p.UserID, name).
			Scan(&sess.ID, &sess.Agent, &sess.OSUser, &sess.Machine, &sess.Origin, &sess.Folder)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return err
		default:
			resumed = true
		}
		setIf(&sess.Agent, p.Agent)
		setIf(&sess.OSUser, p.OSUser)
		setIf(&sess.Machine, p.Machine)
		if p.Location != nil {
			sess.Origin, sess.Folder = p.Location.Origin, p.Location.Folder
		}
		if sess.Repository, err = profile.Repository(sess.Origin, sess.Machine, sess.Folder); err != nil {
			return err
		}
		if resumed {
			_, err = tx.ExecContext(ctx, `
				UPDATE sessions SET agent = ?, os_user = ?, machine = ?, origin = ?, folder = ?, repository = ?, last_seen_at = ?
				WHERE id = ?`,
				sess.Agent, sess.OSUser, sess.Machine, sess.Origin, sess.Folder, sess.Repository, millis(now), sess.ID)
			return err
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO sessions (user_id, name, agent, os_user, machine, origin, folder, repository, created_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.UserID, name, sess.Agent, sess.OSUser, sess.Machine, sess.Origin, sess.Folder, sess.Repository, millis(now), millis(now))
		if err != nil {
			return err
		}
		sess.ID, err = res.LastInsertId()
		return err
	})
	if err != nil {
		return Session{}, false, fmt.Errorf("join session: %w", err)
	}
	return sess, resumed, nil
}

func setIf(dst *string, v *string) {
	if v != nil {
		*dst = *v
	}
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
		`SELECT id, name, agent, os_user, machine, origin, folder, repository, last_seen_at
		FROM sessions WHERE user_id = ? ORDER BY last_seen_at DESC, id DESC LIMIT ?`,
		userID, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		sess := Session{UserID: userID}
		var seen int64
		if err := rows.Scan(&sess.ID, &sess.Name, &sess.Agent, &sess.OSUser, &sess.Machine, &sess.Origin, &sess.Folder,
			&sess.Repository, &seen); err != nil {
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
		sent = Sent{MessageID: id, SentAt: fromMillis(millis(now)), Recipients: []string{to}, RecipientIDs: []int64{recipient}, Scope: ScopeUser}
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
		sent, err = s.postToRoom(ctx, tx, sender.ID, ScopeUser, roomID, body,
			"SELECT id FROM sessions WHERE user_id = ? AND id <> ?", sender.UserID, sender.ID)
		return err
	})
	if err != nil {
		return Sent{}, fmt.Errorf("send to room %q: %w", room, err)
	}
	return sent, nil
}

// SendToRepositoryRoom queues a message for every other session that records the sender's
// repository unit now, whatever its user and whether it is connected or not. The unit is read
// from the database at the time of sending, so it is the one the sender declared last. A session
// that declares the unit later does not receive the message, and one that changes its
// declaration keeps what it received. Only the room named lounge exists in a repository space;
// it is created at the first message to the unit.
func (s *Store) SendToRepositoryRoom(ctx context.Context, sender Session, room, body string) (Sent, error) {
	if room != LoungeRoom {
		return Sent{}, fmt.Errorf("send to room %q: %w", room, ErrNoSuchRoom)
	}
	var sent Sent
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var repository string
		if err := tx.QueryRowContext(ctx, "SELECT repository FROM sessions WHERE id = ?", sender.ID).Scan(&repository); err != nil {
			return err
		}
		if repository == "" {
			return ErrNoRepository
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO rooms (scope, user_id, repository, name, created_at) VALUES (?, NULL, ?, ?, ?) ON CONFLICT DO NOTHING",
			ScopeRepository, repository, room, millis(s.now())); err != nil {
			return err
		}
		var roomID int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM rooms WHERE scope = ? AND repository = ? AND name = ?",
			ScopeRepository, repository, room).Scan(&roomID); err != nil {
			return err
		}
		var err error
		sent, err = s.postToRoom(ctx, tx, sender.ID, ScopeRepository, roomID, body,
			"SELECT id FROM sessions WHERE repository = ? AND id <> ?", repository, sender.ID)
		sent.Repository = repository
		return err
	})
	if err != nil {
		return Sent{}, fmt.Errorf("send to the %s room of the repository: %w", room, err)
	}
	return sent, nil
}

// postToRoom stores a message to a room and queues it for the sessions selected by
// recipientsQuery (with recipientsArgs), which returns session IDs.
func (s *Store) postToRoom(ctx context.Context, tx *sql.Tx, senderID int64, scope string, roomID int64, body string,
	recipientsQuery string, recipientsArgs ...any) (Sent, error) {
	now := s.now()
	res, err := tx.ExecContext(ctx,
		"INSERT INTO messages (scope, sender_session_id, room_id, body, sent_at) VALUES (?, ?, ?, ?, ?)",
		scope, senderID, roomID, body, millis(now))
	if err != nil {
		return Sent{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Sent{}, err
	}
	args := append([]any{id, millis(now)}, recipientsArgs...)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO deliveries (message_id, recipient_session_id, queued_at)
		SELECT ?, id, ? FROM (`+recipientsQuery+`)`, args...); err != nil {
		return Sent{}, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT s.id, s.name FROM deliveries d JOIN sessions s ON s.id = d.recipient_session_id
		WHERE d.message_id = ? ORDER BY s.name, s.id`, id)
	if err != nil {
		return Sent{}, err
	}
	defer rows.Close()
	sent := Sent{MessageID: id, SentAt: fromMillis(millis(now)), Recipients: []string{}, Scope: scope}
	for rows.Next() {
		var recipientID int64
		var name string
		if err := rows.Scan(&recipientID, &name); err != nil {
			return Sent{}, err
		}
		sent.Recipients = append(sent.Recipients, name)
		sent.RecipientIDs = append(sent.RecipientIDs, recipientID)
	}
	return sent, rows.Err()
}

// FetchInbox hands the session its oldest unread messages, up to limit, and marks them
// delivered. Selecting and marking share one write transaction, so concurrent calls never hand
// out the same message twice.
func (s *Store) FetchInbox(ctx context.Context, sessionID int64, limit int) ([]InboxMessage, error) {
	msgs := []InboxMessage{}
	err := s.tx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT m.id, s.name, s.agent, s.os_user, s.machine, m.scope, coalesce(r.name, ''), coalesce(r.repository, ''),
				m.body, m.sent_at
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
			if err := rows.Scan(&m.MessageID, &m.From, &m.FromAgent, &m.FromOSUser, &m.FromMachine, &m.Scope, &m.Room,
				&m.Repository, &m.Body, &sentAt); err != nil {
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
