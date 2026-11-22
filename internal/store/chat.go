package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/profile"
)

const (
	// LocalUser is the user of the connections without a token (see package server). It is
	// reserved: it cannot be added, removed or given a token.
	LocalUser = "local"
	// LoungeRoom is the room every scope has. It is the only room.
	LoungeRoom = "lounge"

	// localTokenHash marks a user without a token. Token hashes are hexadecimal digests, so no
	// token can match it.
	localTokenHash = "none:" + LocalUser
)

// Limits bound what sessions can store. The server sets them from its startup options (see
// package config); the usage guide (lounge_help) states the values in effect. Lowering a limit
// deletes nothing: it applies to new messages, sessions and memories.
type Limits struct {
	// MaxBodyBytes is the largest message body, in bytes of UTF-8.
	MaxBodyBytes int
	// MaxUnread is the most unread messages one session can hold. A message to a session that
	// holds this many is refused; a room message skips that session.
	MaxUnread int
	// MaxSessionsPerUser is the most sessions one user can have. Joining under a new name fails
	// beyond it; joining under an existing name still works.
	MaxSessionsPerUser int
	// MaxMemoryBytes is the largest memory body, in bytes of UTF-8.
	MaxMemoryBytes int
	// MaxMemoriesPerScope is the most memories one scope (one user, one repository unit, or one
	// branch) can hold, faded and pinned ones included. Creating a memory fails beyond it.
	MaxMemoriesPerScope int
}

// Default limits, used until SetLimits is called.
const (
	DefaultMaxBodyBytes        = 65536
	DefaultMaxUnread           = 1000
	DefaultMaxSessionsPerUser  = 500
	DefaultMaxMemoryBytes      = 16384
	DefaultMaxMemoriesPerScope = 10000
)

// DefaultLimits returns the limits a store has when it is opened.
func DefaultLimits() Limits {
	return Limits{
		MaxBodyBytes: DefaultMaxBodyBytes, MaxUnread: DefaultMaxUnread, MaxSessionsPerUser: DefaultMaxSessionsPerUser,
		MaxMemoryBytes: DefaultMaxMemoryBytes, MaxMemoriesPerScope: DefaultMaxMemoriesPerScope,
	}
}

// SetLimits replaces the limits. Call it before the store is used concurrently.
func (s *Store) SetLimits(l Limits) { s.limits = l }

// Limits returns the limits in effect.
func (s *Store) Limits() Limits { return s.limits }

// Errors the tools turn into guidance for agents.
var (
	// ErrNoSuchSession is returned for a session target that does not exist or is outside the
	// scope. The two cases share one error on purpose: a session of another user must not learn
	// whether a session it shares no scope with exists.
	ErrNoSuchSession = errors.New("no such session in the scope")
	ErrNoSuchRoom    = errors.New("no such room")
	ErrSendToSelf    = errors.New("the destination is the sending session")
	// ErrInvalidScope is returned for a scope that is not one of ScopeKinds.
	ErrInvalidScope = errors.New("invalid scope")
	// ErrNoRepository is returned for a message or memory of scope repository from a session
	// that declared no repository.
	ErrNoRepository = errors.New("the sending session has no repository")
	// ErrNoBranch is returned for a message or memory of scope branch from a session that
	// declared no repository or no branch.
	ErrNoBranch = errors.New("the sending session has no branch")
	// ErrBodyTooLarge is returned for a body over Limits.MaxBodyBytes.
	ErrBodyTooLarge = errors.New("the message body is too large")
	// ErrInboxFull is returned for a message to a session that holds Limits.MaxUnread unread
	// messages.
	ErrInboxFull = errors.New("the recipient's inbox is full")
	// ErrTooManySessions is returned for joining under a new name when the user has
	// Limits.MaxSessionsPerUser sessions.
	ErrTooManySessions = errors.New("the user has too many sessions")
	// ErrMachineRequired is returned by JoinSession when the session would have a folder
	// without an origin and without a machine.
	ErrMachineRequired = profile.ErrMachineRequired
)

// Session is a named unit of work of one user. It outlives connections: joining again with the
// same name continues it, including its unread messages.
type Session struct {
	ID       int64
	UserID   int64
	UserName string
	Name     string
	Agent    string
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
	Branch     string
}

// Location is the place a session works in: the normalized origin and folder.
type Location struct {
	Origin string
	Folder string
}

// Member identifies a session together with its user, as a recipient of a message.
type Member struct {
	ID   int64
	User string
	Name string
}

// Target is where a message goes: a room of the scope, or one session of the scope given by
// its user and name. Exactly one of Room and Name is set.
type Target struct {
	Room string
	User string
	Name string
}

// Sent describes an accepted message.
type Sent struct {
	MessageID int64
	SentAt    time.Time
	Scope     Scope
	// Recipients are the sessions the message is queued for, ordered by name.
	Recipients []Member
	// Skipped are the sessions of a room that did not get the message because their inbox
	// was full (Limits.MaxUnread).
	Skipped []Member
}

// InboxMessage is a message handed to its recipient.
type InboxMessage struct {
	MessageID int64
	FromUser  string // sender's user name
	From      string // sender's session name
	// FromAgent, FromOSUser and FromMachine are the values the sender declared when it sent
	// the message ("" when not declared).
	FromAgent   string
	FromOSUser  string
	FromMachine string
	Scope       Scope
	Room        string // empty for a message addressed to the session
	Body        string
	SentAt      time.Time
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
	// Location replaces the recorded origin and folder together when not nil. When Location is
	// not nil and Branch is nil, the recorded branch is cleared so moving to another repository
	// never carries the previous repository's branch.
	Location *Location
	// Branch replaces the recorded branch when not nil ("" clears it).
	Branch *string
}

func millis(t time.Time) int64      { return t.UnixMilli() }
func fromMillis(ms int64) time.Time { return time.UnixMilli(ms).UTC() }

// sessionColumns selects a Session from sessions s joined with users u; see scanSession.
const sessionColumns = `s.id, s.user_id, u.name, s.name, s.agent, s.os_user, s.machine, s.origin, s.folder,
	s.repository, s.branch, s.last_seen_at`

// sessionFrom is the FROM clause that goes with sessionColumns.
const sessionFrom = ` FROM sessions s JOIN users u ON u.id = s.user_id `

type rowScanner interface{ Scan(dest ...any) error }

func scanSession(r rowScanner) (Session, error) {
	var sess Session
	var seen int64
	err := r.Scan(&sess.ID, &sess.UserID, &sess.UserName, &sess.Name, &sess.Agent, &sess.OSUser, &sess.Machine,
		&sess.Origin, &sess.Folder, &sess.Repository, &sess.Branch, &seen)
	sess.LastSeenAt = fromMillis(seen)
	return sess, err
}

// EnsureLocalUser creates the local user and the lounge room of its space when missing and
// returns the user.
func (s *Store) EnsureLocalUser(ctx context.Context) (User, error) {
	var u User
	err := s.tx(ctx, func(tx *sql.Tx) error {
		now := millis(s.now())
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO users (name, token_hash, created_at) VALUES (?, ?, ?) ON CONFLICT (name) DO NOTHING",
			LocalUser, localTokenHash, now); err != nil {
			return err
		}
		var created int64
		if err := tx.QueryRowContext(ctx, "SELECT id, name, created_at FROM users WHERE name = ?", LocalUser).
			Scan(&u.ID, &u.Name, &created); err != nil {
			return err
		}
		u.CreatedAt = fromMillis(created)
		_, err := tx.ExecContext(ctx,
			"INSERT INTO rooms (scope, user_id, name, created_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING",
			ScopeUser, u.ID, LoungeRoom, now)
		return err
	})
	if err != nil {
		return User{}, fmt.Errorf("create the local user: %w", err)
	}
	return u, nil
}

// JoinSession returns the user's session named p.Name, creating it when missing, and records
// the declared values of p. resumed reports that the session existed. Session names are unique
// per user, so joining under a name continues only a session of the same user. A new name fails
// with ErrTooManySessions when the user has Limits.MaxSessionsPerUser sessions.
//
// origin and folder are replaced as one set (p.Location), never one at a time: a session that
// moves from a repository with an origin to a folder without one must not keep the old origin,
// which would decide its repository unit. The unit is decided again from the resulting values;
// a folder without an origin and without a machine fails with ErrMachineRequired and changes
// nothing.
func (s *Store) JoinSession(ctx context.Context, p JoinParams) (sess Session, resumed bool, err error) {
	err = s.tx(ctx, func(tx *sql.Tx) error {
		now := s.now()
		var userName string
		err := tx.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", p.UserID).Scan(&userName)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNoSuchUser
		case err != nil:
			return err
		}
		name := p.Name
		if name == "" {
			if name, err = freeSessionName(ctx, tx, p.UserID); err != nil {
				return err
			}
		}
		sess = Session{UserID: p.UserID, UserName: userName, Name: name, LastSeenAt: now.UTC().Truncate(time.Millisecond)}
		err = tx.QueryRowContext(ctx,
			"SELECT id, agent, os_user, machine, origin, folder, branch FROM sessions WHERE user_id = ? AND name = ?", p.UserID, name).
			Scan(&sess.ID, &sess.Agent, &sess.OSUser, &sess.Machine, &sess.Origin, &sess.Folder, &sess.Branch)
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
			if p.Branch == nil {
				sess.Branch = ""
			}
		}
		setIf(&sess.Branch, p.Branch)
		if sess.Repository, err = profile.Repository(sess.Origin, sess.Machine, sess.Folder); err != nil {
			return err
		}
		if resumed {
			_, err = tx.ExecContext(ctx, `
				UPDATE sessions SET agent = ?, os_user = ?, machine = ?, origin = ?, folder = ?, repository = ?, branch = ?, last_seen_at = ?
				WHERE id = ?`,
				sess.Agent, sess.OSUser, sess.Machine, sess.Origin, sess.Folder, sess.Repository, sess.Branch, millis(now), sess.ID)
			return err
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM sessions WHERE user_id = ?", p.UserID).Scan(&count); err != nil {
			return err
		}
		if count >= s.limits.MaxSessionsPerUser {
			return ErrTooManySessions
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO sessions (user_id, name, agent, os_user, machine, origin, folder, repository, branch, created_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.UserID, name, sess.Agent, sess.OSUser, sess.Machine, sess.Origin, sess.Folder, sess.Repository, sess.Branch, millis(now), millis(now))
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

// Touch records that sess was used now and reports whether it is still recorded: the same ID,
// user and name. The user and name are compared too because a deleted session's ID can be
// given to a new session. One statement does both, so that every tool call of a joined
// connection costs a single query to notice a deleted session (deleted by the cleanup of
// unused sessions, by session remove, or with its user) and to keep the session in use.
func (s *Store) Touch(ctx context.Context, sess Session) (exists bool, err error) {
	res, err := s.db.ExecContext(ctx, "UPDATE sessions SET last_seen_at = ? WHERE id = ? AND user_id = ? AND name = ?",
		millis(s.now()), sess.ID, sess.UserID, sess.Name)
	if err != nil {
		return false, fmt.Errorf("update session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("update session: %w", err)
	}
	return n > 0, nil
}

// Unread returns the session's number of unread messages: deliveries still queued. Expired
// deliveries are not unread.
func (s *Store) Unread(ctx context.Context, sessionID int64) (unread int, err error) {
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
	sessions, truncated, err = s.listSessions(ctx, s.db, "s.user_id = ?", []any{userID}, limit)
	if err != nil {
		return nil, false, fmt.Errorf("list sessions: %w", err)
	}
	return sessions, truncated, nil
}

// VisibleSessions returns the session viewerID as it is recorded now, and up to limit sessions
// it can reach in any of its scopes, most recently used first: the union of the scopes'
// members, the viewer included. truncated reports that more exist.
func (s *Store) VisibleSessions(ctx context.Context, viewerID int64, limit int) (viewer Session, sessions []Session, truncated bool, err error) {
	err = s.tx(ctx, func(tx *sql.Tx) error {
		if viewer, err = loadSession(ctx, tx, viewerID); err != nil {
			return err
		}
		var conds []string
		var args []any
		for _, sc := range ScopesOf(viewer) {
			cond, condArgs := scopeRules[sc.Kind].members(sc.ID)
			conds = append(conds, "("+cond+")")
			args = append(args, condArgs...)
		}
		sessions, truncated, err = s.listSessions(ctx, tx, strings.Join(conds, " OR "), args, limit)
		return err
	})
	if err != nil {
		return Session{}, nil, false, fmt.Errorf("list sessions: %w", err)
	}
	return viewer, sessions, truncated, nil
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// listSessions returns up to limit sessions that meet cond, most recently used first.
func (s *Store) listSessions(ctx context.Context, q querier, cond string, args []any, limit int) ([]Session, bool, error) {
	rows, err := q.QueryContext(ctx, "SELECT "+sessionColumns+sessionFrom+"WHERE "+cond+
		" ORDER BY s.last_seen_at DESC, s.id DESC LIMIT ?", append(args, limit+1)...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	sessions := []Session{}
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, false, err
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(sessions) > limit {
		return sessions[:limit], true, nil
	}
	return sessions, false, nil
}

// loadSession reads a session as it is recorded now.
func loadSession(ctx context.Context, tx *sql.Tx, id int64) (Session, error) {
	sess, err := scanSession(tx.QueryRowContext(ctx, "SELECT "+sessionColumns+sessionFrom+"WHERE s.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, errors.New("the session no longer exists: join again")
	}
	return sess, err
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

// Send queues a message from the session senderID in the given scope (one of ScopeKinds).
//
// The scope is the sender's own: its identifier is decided from the sender as recorded at the
// time of sending (see scopeRule), so a session can reach only the ranges it belongs to. The
// recipients are the sessions in that range now, connected or not. A session that enters the
// range later does not receive the message, and one that leaves it keeps what it received.
//
// To a room (target.Room, which must be LoungeRoom), the message is queued for every other
// session of the range, except those whose inbox is full (Sent.Skipped). To one session
// (target.User and target.Name), the session must be in the range: a session outside it and a
// missing session both fail with ErrNoSuchSession, and a full inbox fails with ErrInboxFull.
func (s *Store) Send(ctx context.Context, senderID int64, scope string, target Target, body string) (Sent, error) {
	rule, ok := scopeRules[scope]
	if !ok {
		return Sent{}, ErrInvalidScope
	}
	if len(body) > s.limits.MaxBodyBytes {
		return Sent{}, ErrBodyTooLarge
	}
	var sent Sent
	err := s.tx(ctx, func(tx *sql.Tx) error {
		sender, err := loadSession(ctx, tx, senderID)
		if err != nil {
			return err
		}
		id, ok := rule.id(sender)
		if !ok {
			return rule.unavailable
		}
		sc := Scope{Kind: scope, ID: id}
		if target.Room != "" {
			sent, err = s.sendToRoom(ctx, tx, sender, sc, target.Room, body)
		} else {
			sent, err = s.sendToSession(ctx, tx, sender, sc, target, body)
		}
		return err
	})
	if err != nil {
		if target.Room != "" {
			return Sent{}, fmt.Errorf("send to room %q: %w", target.Room, err)
		}
		return Sent{}, fmt.Errorf("send to session %q: %w", target.Name, err)
	}
	return sent, nil
}

// candidate is a session of a scope with its number of unread messages.
type candidate struct {
	Member
	unread int
}

// scopeMembers returns the sessions of the scope that meet the extra condition, ordered by
// session name.
func scopeMembers(ctx context.Context, tx *sql.Tx, sc Scope, extra string, extraArgs ...any) ([]candidate, error) {
	cond, args := scopeRules[sc.Kind].members(sc.ID)
	rows, err := tx.QueryContext(ctx, `
		SELECT s.id, u.name, s.name,
			(SELECT count(*) FROM deliveries d WHERE d.recipient_session_id = s.id AND d.state = 'queued')`+
		sessionFrom+"WHERE ("+cond+") AND "+extra+" ORDER BY s.name, u.name, s.id", append(args, extraArgs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ID, &c.User, &c.Name, &c.unread); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) sendToSession(ctx context.Context, tx *sql.Tx, sender Session, sc Scope, target Target, body string) (Sent, error) {
	found, err := scopeMembers(ctx, tx, sc, "u.name = ? AND s.name = ?", target.User, target.Name)
	switch {
	case err != nil:
		return Sent{}, err
	case len(found) == 0:
		return Sent{}, ErrNoSuchSession
	case found[0].ID == sender.ID:
		return Sent{}, ErrSendToSelf
	case found[0].unread >= s.limits.MaxUnread:
		return Sent{}, ErrInboxFull
	}
	return s.store(ctx, tx, sender, sc, "to_session_id", found[0].ID, body, found[:1], nil)
}

func (s *Store) sendToRoom(ctx context.Context, tx *sql.Tx, sender Session, sc Scope, room, body string) (Sent, error) {
	if room != LoungeRoom {
		return Sent{}, ErrNoSuchRoom
	}
	// A room is created at the first message of its scope (the lounge of a user is created
	// together with the user).
	userID, repository := scopeRules[sc.Kind].room(sender, sc.ID)
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO rooms (scope, user_id, repository, name, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
		sc.Kind, userID, repository, room, millis(s.now())); err != nil {
		return Sent{}, err
	}
	var roomID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT id FROM rooms WHERE scope = ? AND coalesce(user_id, 0) = coalesce(?, 0) AND repository = ? AND name = ?",
		sc.Kind, userID, repository, room).Scan(&roomID); err != nil {
		return Sent{}, err
	}
	found, err := scopeMembers(ctx, tx, sc, "s.id <> ?", sender.ID)
	if err != nil {
		return Sent{}, err
	}
	var recipients, skipped []candidate
	for _, c := range found {
		if c.unread >= s.limits.MaxUnread {
			skipped = append(skipped, c)
		} else {
			recipients = append(recipients, c)
		}
	}
	return s.store(ctx, tx, sender, sc, "room_id", roomID, body, recipients, skipped)
}

// store inserts a message addressed by column (to_session_id or room_id) and queues it for
// the recipients.
func (s *Store) store(ctx context.Context, tx *sql.Tx, sender Session, sc Scope, column string, dest int64, body string,
	recipients, skipped []candidate) (Sent, error) {
	now := millis(s.now())
	// The sender's declared values are copied as they are now (see schema version 4).
	res, err := tx.ExecContext(ctx,
		"INSERT INTO messages (scope, scope_id, sender_session_id, "+column+`, body, sent_at, sender_agent, sender_os_user, sender_machine)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sc.Kind, sc.ID, sender.ID, dest, body, now, sender.Agent, sender.OSUser, sender.Machine)
	if err != nil {
		return Sent{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Sent{}, err
	}
	sent := Sent{MessageID: id, SentAt: fromMillis(now), Scope: sc, Recipients: []Member{}, Skipped: []Member{}}
	for _, r := range recipients {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO deliveries (message_id, recipient_session_id, queued_at) VALUES (?, ?, ?)", id, r.ID, now); err != nil {
			return Sent{}, err
		}
		sent.Recipients = append(sent.Recipients, r.Member)
	}
	for _, r := range skipped {
		sent.Skipped = append(sent.Skipped, r.Member)
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
			SELECT m.id, u.name, s.name, m.sender_agent, m.sender_os_user, m.sender_machine, m.scope, m.scope_id, coalesce(r.name, ''),
				m.body, m.sent_at
			FROM deliveries d
			JOIN messages m ON m.id = d.message_id
			JOIN sessions s ON s.id = m.sender_session_id
			JOIN users u ON u.id = s.user_id
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
			if err := rows.Scan(&m.MessageID, &m.FromUser, &m.From, &m.FromAgent, &m.FromOSUser, &m.FromMachine,
				&m.Scope.Kind, &m.Scope.ID, &m.Room, &m.Body, &sentAt); err != nil {
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
