package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Rooms. Every scope has the room LoungeRoom, which every session of the scope's range
// belongs to; its membership is not recorded. Other rooms are created by sessions (CreateRoom)
// and have recorded members (room_members): a message posted to one reaches only the sessions
// of the range that are members of it at the time of sending.
//
// Membership decides what becomes unread, not what can be read: any session that can use a
// scope can list its rooms and read their messages (see History), member or not.
//
// A membership is kept when the session's repository unit or branch changes. While the session
// is outside the room's range, messages to the room do not reach it (recipients are always
// taken from the range at the time of sending), and they reach it again once it is back.

// Room errors the tools turn into guidance for agents.
var (
	// ErrTooManyRooms is returned for creating a room in a scope that holds
	// Limits.MaxRoomsPerScope created rooms.
	ErrTooManyRooms = errors.New("the scope has too many rooms")
	// ErrLeaveLounge is returned for leaving LoungeRoom, which every session belongs to.
	ErrLeaveLounge = errors.New("the lounge room cannot be left")
)

// RoomRef names a room: its scope and name.
type RoomRef struct {
	Scope Scope
	Name  string
}

// RoomInfo describes a room for a session.
type RoomInfo struct {
	RoomRef
	// Joined reports whether the session is a member (always true for LoungeRoom).
	Joined bool
	// Members is the number of other sessions a message posted now would reach.
	Members int
}

// RoomResult is the result of CreateRoom, JoinRoom and LeaveRoom.
type RoomResult struct {
	RoomRef
	// Created reports whether CreateRoom made a new room.
	Created bool
	// Joined reports whether the session is a member after the operation.
	Joined bool
}

// sessionScope returns the scope of the given kind that the session uses, or the kind's
// unavailable error.
func sessionScope(sess Session, kind string) (Scope, error) {
	rule, ok := scopeRules[kind]
	if !ok {
		return Scope{}, ErrInvalidScope
	}
	id, ok := rule.id(sess)
	if !ok {
		return Scope{}, rule.unavailable
	}
	return Scope{Kind: kind, ID: id}, nil
}

// roomColumns returns the user_id and repository columns of the rooms of scope sc.
func roomColumns(sess Session, sc Scope) (userID any, repository string) {
	return scopeRules[sc.Kind].room(sess, sc.ID)
}

// findRoom returns the ID of the room of scope sc with the given name, and false when there is
// none. For LoungeRoom it creates the row when missing (the lounge of a repository or branch
// is created at its first use).
func findRoom(ctx context.Context, tx *sql.Tx, sess Session, sc Scope, name string, now int64) (int64, bool, error) {
	userID, repository := roomColumns(sess, sc)
	if name == LoungeRoom {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO rooms (scope, user_id, repository, name, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
			sc.Kind, userID, repository, name, now); err != nil {
			return 0, false, err
		}
	}
	var id int64
	err := tx.QueryRowContext(ctx,
		"SELECT id FROM rooms WHERE scope = ? AND coalesce(user_id, 0) = coalesce(?, 0) AND repository = ? AND name = ?",
		sc.Kind, userID, repository, name).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, false, nil
	case err != nil:
		return 0, false, err
	}
	return id, true, nil
}

// joinRoom records the session as a member of the room. Joining again changes nothing.
func joinRoom(ctx context.Context, tx *sql.Tx, roomID, sessionID, now int64) error {
	_, err := tx.ExecContext(ctx,
		"INSERT INTO room_members (room_id, session_id, joined_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
		roomID, sessionID, now)
	return err
}

// roomOp runs fn for the room name of scope kind of the session sessionID in a transaction.
// name must be normalized (see package names).
func (s *Store) roomOp(ctx context.Context, sessionID int64, kind, name string,
	fn func(tx *sql.Tx, sess Session, sc Scope, now int64) (RoomResult, error)) (RoomResult, error) {
	var res RoomResult
	err := s.tx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		sc, err := sessionScope(sess, kind)
		if err != nil {
			return err
		}
		res, err = fn(tx, sess, sc, millis(s.now()))
		res.RoomRef = RoomRef{Scope: sc, Name: name}
		return err
	})
	if err != nil {
		return RoomResult{}, fmt.Errorf("room %q: %w", name, err)
	}
	return res, nil
}

// CreateRoom creates the room name in the session's scope of the given kind and makes the
// session a member. When the room exists, it only makes the session a member (Created is
// false). For LoungeRoom it does nothing: the lounge exists in every scope and has every session.
func (s *Store) CreateRoom(ctx context.Context, sessionID int64, kind, name string) (RoomResult, error) {
	return s.roomOp(ctx, sessionID, kind, name, func(tx *sql.Tx, sess Session, sc Scope, now int64) (RoomResult, error) {
		if name == LoungeRoom {
			return RoomResult{Joined: true}, nil
		}
		id, ok, err := findRoom(ctx, tx, sess, sc, name, now)
		if err != nil {
			return RoomResult{}, err
		}
		created := false
		if !ok {
			userID, repository := roomColumns(sess, sc)
			var n int
			if err := tx.QueryRowContext(ctx,
				"SELECT count(*) FROM rooms WHERE scope = ? AND coalesce(user_id, 0) = coalesce(?, 0) AND repository = ? AND name <> ?",
				sc.Kind, userID, repository, LoungeRoom).Scan(&n); err != nil {
				return RoomResult{}, err
			}
			if n >= s.limits.MaxRoomsPerScope {
				return RoomResult{}, ErrTooManyRooms
			}
			r, err := tx.ExecContext(ctx,
				"INSERT INTO rooms (scope, user_id, repository, name, created_at) VALUES (?, ?, ?, ?, ?)",
				sc.Kind, userID, repository, name, now)
			if err != nil {
				return RoomResult{}, err
			}
			if id, err = r.LastInsertId(); err != nil {
				return RoomResult{}, err
			}
			created = true
		}
		if err := joinRoom(ctx, tx, id, sess.ID, now); err != nil {
			return RoomResult{}, err
		}
		return RoomResult{Created: created, Joined: true}, nil
	})
}

// JoinRoom makes the session a member of the existing room name of its scope of the given kind.
// A missing room fails with ErrNoSuchRoom.
func (s *Store) JoinRoom(ctx context.Context, sessionID int64, kind, name string) (RoomResult, error) {
	return s.roomOp(ctx, sessionID, kind, name, func(tx *sql.Tx, sess Session, sc Scope, now int64) (RoomResult, error) {
		if name == LoungeRoom {
			return RoomResult{Joined: true}, nil
		}
		id, ok, err := findRoom(ctx, tx, sess, sc, name, now)
		switch {
		case err != nil:
			return RoomResult{}, err
		case !ok:
			return RoomResult{}, ErrNoSuchRoom
		}
		return RoomResult{Joined: true}, joinRoom(ctx, tx, id, sess.ID, now)
	})
}

// LeaveRoom ends the session's membership of the room name of its scope of the given kind.
// Leaving a room the session is not a member of succeeds. A missing room fails with
// ErrNoSuchRoom, and LoungeRoom with ErrLeaveLounge.
func (s *Store) LeaveRoom(ctx context.Context, sessionID int64, kind, name string) (RoomResult, error) {
	return s.roomOp(ctx, sessionID, kind, name, func(tx *sql.Tx, sess Session, sc Scope, now int64) (RoomResult, error) {
		if name == LoungeRoom {
			return RoomResult{}, ErrLeaveLounge
		}
		id, ok, err := findRoom(ctx, tx, sess, sc, name, now)
		switch {
		case err != nil:
			return RoomResult{}, err
		case !ok:
			return RoomResult{}, ErrNoSuchRoom
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM room_members WHERE room_id = ? AND session_id = ?", id, sess.ID)
		return RoomResult{Joined: false}, err
	})
}

// memberCondition is the extra condition of scopeMembers that selects the members of a created
// room (first argument) other than one session (second argument).
const memberCondition = "s.id <> ? AND s.id IN (SELECT session_id FROM room_members WHERE room_id = ?)"

// Rooms returns the rooms of the session's scopes, in the order of ScopeKinds, each scope's
// LoungeRoom first and then its created rooms by name. kind limits the result to one scope; ""
// means every scope the session can use. joinedOnly limits it to the rooms the session belongs to.
func (s *Store) Rooms(ctx context.Context, sessionID int64, kind string, joinedOnly bool) ([]RoomInfo, error) {
	out := []RoomInfo{}
	err := s.readTx(ctx, func(tx *sql.Tx) error {
		sess, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		var scopes []Scope
		if kind != "" {
			sc, err := sessionScope(sess, kind)
			if err != nil {
				return err
			}
			scopes = []Scope{sc}
		} else {
			scopes = ScopesOf(sess)
		}
		for _, sc := range scopes {
			everyone, err := scopeMembers(ctx, tx, sc, "s.id <> ?", sess.ID)
			if err != nil {
				return err
			}
			out = append(out, RoomInfo{RoomRef: RoomRef{Scope: sc, Name: LoungeRoom}, Joined: true, Members: len(everyone)})
			userID, repository := roomColumns(sess, sc)
			rows, err := tx.QueryContext(ctx, `
				SELECT r.id, r.name, EXISTS (SELECT 1 FROM room_members m WHERE m.room_id = r.id AND m.session_id = ?)
				FROM rooms r
				WHERE r.scope = ? AND coalesce(r.user_id, 0) = coalesce(?, 0) AND r.repository = ? AND r.name <> ?
				ORDER BY r.name`, sess.ID, sc.Kind, userID, repository, LoungeRoom)
			if err != nil {
				return err
			}
			type room struct {
				id     int64
				name   string
				joined bool
			}
			var rooms []room
			for rows.Next() {
				var r room
				if err := rows.Scan(&r.id, &r.name, &r.joined); err != nil {
					rows.Close()
					return err
				}
				rooms = append(rooms, r)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
			for _, r := range rooms {
				if joinedOnly && !r.joined {
					continue
				}
				members, err := scopeMembers(ctx, tx, sc, memberCondition, sess.ID, r.id)
				if err != nil {
					return err
				}
				out = append(out, RoomInfo{RoomRef: RoomRef{Scope: sc, Name: r.name}, Joined: r.joined, Members: len(members)})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	return out, nil
}

// RoomExists reports whether a room named name exists in the session's scope of the given kind,
// or in any scope the session can use when kind is "". LoungeRoom always exists.
func (s *Store) RoomExists(ctx context.Context, sessionID int64, kind, name string) (bool, error) {
	if name == LoungeRoom {
		return true, nil
	}
	rooms, err := s.Rooms(ctx, sessionID, kind, false)
	if err != nil {
		return false, err
	}
	for _, r := range rooms {
		if r.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// deleteUnusedRooms deletes the created rooms that have no member and no message. LoungeRoom is
// never deleted.
func deleteUnusedRooms(ctx context.Context, tx *sql.Tx) (int, error) {
	res, err := tx.ExecContext(ctx, `
		DELETE FROM rooms WHERE name <> ?
			AND NOT EXISTS (SELECT 1 FROM room_members m WHERE m.room_id = rooms.id)
			AND NOT EXISTS (SELECT 1 FROM messages g WHERE g.room_id = rooms.id)`, LoungeRoom)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}
