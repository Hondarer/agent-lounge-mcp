package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// User is a user of the lounge: the owner of sessions, identified by a token (or, for the user
// local, by the absence of one).
type User struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// UserSummary is a user as listed by ListUsers.
type UserSummary struct {
	User
	Sessions int
	// LastUsedAt is the last activity of the user's sessions; zero when it has none.
	LastUsedAt time.Time
}

// UserData counts the data that belongs to a user and is deleted with it.
type UserData struct {
	Sessions int
	// Messages are the messages sent by the user's sessions or addressed to one of them.
	Messages int
	// Deliveries are the delivery records of those messages and of the messages to the
	// user's sessions.
	Deliveries int
	// Memories are the memories of the user's own scope (user). Memories its sessions wrote in
	// a repository scope belong to that scope and stay.
	Memories int
}

// Errors of the user functions.
var (
	ErrNoSuchUser   = errors.New("no such user")
	ErrUserExists   = errors.New("the user already exists")
	ErrReservedUser = errors.New("the user local is reserved for connections without a token")
)

// UserNameRule describes the form of a user name.
const UserNameRule = "1 to 32 characters from a-z 0-9 . _ -, starting with a letter or digit"

// userNameRE is the form of a user name. It has no '/', so a user name and a session name
// joined by '/' (an address of another user's session) split unambiguously.
var userNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)

// ValidateUserName checks the form of a user name for adding a user. The reserved name local
// fails with ErrReservedUser.
func ValidateUserName(name string) error {
	if !userNameRE.MatchString(name) {
		return fmt.Errorf("invalid user name %q: use %s", name, UserNameRule)
	}
	if name == LocalUser {
		return ErrReservedUser
	}
	return nil
}

// checkManagedUser rejects the names that the user subcommands cannot act on.
func checkManagedUser(name string) error {
	if name == LocalUser {
		return ErrReservedUser
	}
	if !userNameRE.MatchString(name) {
		return fmt.Errorf("invalid user name %q: use %s", name, UserNameRule)
	}
	return nil
}

// AddUser creates a user with the hash of its token (see package auth) and the lounge room of
// its space.
func (s *Store) AddUser(ctx context.Context, name, tokenHash string) (User, error) {
	if err := ValidateUserName(name); err != nil {
		return User{}, err
	}
	var u User
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE name = ?", name).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return ErrUserExists
		}
		now := millis(s.now())
		res, err := tx.ExecContext(ctx, "INSERT INTO users (name, token_hash, created_at) VALUES (?, ?, ?)", name, tokenHash, now)
		if err != nil {
			return err
		}
		if u.ID, err = res.LastInsertId(); err != nil {
			return err
		}
		u.Name, u.CreatedAt = name, fromMillis(now)
		_, err = tx.ExecContext(ctx,
			"INSERT INTO rooms (scope, user_id, name, created_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING",
			ScopeUser, u.ID, LoungeRoom, now)
		return err
	})
	if err != nil {
		return User{}, fmt.Errorf("add user %q: %w", name, err)
	}
	return u, nil
}

// RotateToken replaces the token hash of the user. The previous token stops working at once;
// the user's sessions and messages stay.
func (s *Store) RotateToken(ctx context.Context, name, tokenHash string) error {
	if err := checkManagedUser(name); err != nil {
		return fmt.Errorf("rotate the token of %q: %w", name, err)
	}
	res, err := s.db.ExecContext(ctx, "UPDATE users SET token_hash = ? WHERE name = ?", tokenHash, name)
	if err != nil {
		return fmt.Errorf("rotate the token of %q: %w", name, err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("rotate the token of %q: %w", name, err)
	} else if n == 0 {
		return fmt.Errorf("rotate the token of %q: %w", name, ErrNoSuchUser)
	}
	return nil
}

// UserByTokenHash returns the user whose token has the hash, or ErrNoSuchUser.
func (s *Store) UserByTokenHash(ctx context.Context, tokenHash string) (User, error) {
	var u User
	var created int64
	// local has no token: its marker must never match, whatever the caller passes.
	err := s.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM users WHERE token_hash = ? AND name <> ?", tokenHash, LocalUser).
		Scan(&u.ID, &u.Name, &created)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return User{}, ErrNoSuchUser
	case err != nil:
		return User{}, fmt.Errorf("look up the token: %w", err)
	}
	u.CreatedAt = fromMillis(created)
	return u, nil
}

// TokenValid reports whether tokenHash is still the token hash of the user with the ID: false
// after user rotate, and after user remove (also when a new user got the ID, since its token
// differs).
func (s *Store) TokenValid(ctx context.Context, userID int64, tokenHash string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE id = ? AND token_hash = ? AND name <> ?",
		userID, tokenHash, LocalUser).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("look up the token: %w", err)
	}
	return n > 0, nil
}

// ListUsers returns every user, in name order, with the number of its sessions and their last
// activity.
func (s *Store) ListUsers(ctx context.Context) ([]UserSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.name, u.created_at, count(s.id), coalesce(max(s.last_seen_at), 0)
		FROM users u LEFT JOIN sessions s ON s.user_id = u.id
		GROUP BY u.id ORDER BY u.name`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []UserSummary
	for rows.Next() {
		var u UserSummary
		var created, lastUsed int64
		if err := rows.Scan(&u.ID, &u.Name, &created, &u.Sessions, &lastUsed); err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}
		u.CreatedAt = fromMillis(created)
		if lastUsed != 0 {
			u.LastUsedAt = fromMillis(lastUsed)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// TokenUserCount returns the number of users that have a token, that is every user but local.
func (s *Store) TokenUserCount(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE name <> ?", LocalUser).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// CountUserData counts what RemoveUser would delete.
func (s *Store) CountUserData(ctx context.Context, name string) (UserData, error) {
	if err := checkManagedUser(name); err != nil {
		return UserData{}, fmt.Errorf("user %q: %w", name, err)
	}
	var data UserData
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var err error
		data, err = countUserData(ctx, tx, name)
		return err
	})
	if err != nil {
		return UserData{}, fmt.Errorf("user %q: %w", name, err)
	}
	return data, nil
}

func countUserData(ctx context.Context, tx *sql.Tx, name string) (UserData, error) {
	var id int64
	err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE name = ?", name).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return UserData{}, ErrNoSuchUser
	case err != nil:
		return UserData{}, err
	}
	d, err := countSessionData(ctx, tx, "SELECT id FROM sessions WHERE user_id = ?", id)
	if err != nil {
		return UserData{}, err
	}
	err = tx.QueryRowContext(ctx, "SELECT count(*) FROM memories WHERE user_id = ?", id).Scan(&d.Memories)
	return d, err
}

// RemoveUser deletes the user with its sessions, the messages they sent or that were addressed
// to them, the delivery records of those, and the memories of its user scope, and returns what it deleted. Connections of the
// user are rejected from their next request, since the token no longer matches any user.
func (s *Store) RemoveUser(ctx context.Context, name string) (UserData, error) {
	if err := checkManagedUser(name); err != nil {
		return UserData{}, fmt.Errorf("remove user %q: %w", name, err)
	}
	var data UserData
	err := s.tx(ctx, func(tx *sql.Tx) error {
		var err error
		if data, err = countUserData(ctx, tx, name); err != nil {
			return err
		}
		// Foreign keys cascade from the user to its sessions, rooms and memories, and from those
		// to the messages, deliveries, tags and references. The full-text index of the memories
		// is a virtual table without foreign keys: its rows are deleted first.
		if _, err := tx.ExecContext(ctx, `DELETE FROM memories_fts WHERE rowid IN (
			SELECT m.id FROM memories m JOIN users u ON u.id = m.user_id WHERE u.name = ?)`, name); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE name = ?", name)
		return err
	})
	if err != nil {
		return UserData{}, fmt.Errorf("remove user %q: %w", name, err)
	}
	return data, nil
}
