package store

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// userRooms returns the names of the rooms of the user scope of userID.
func userRooms(t *testing.T, s *Store, userID int64) []string {
	t.Helper()
	rows, err := s.db.Query("SELECT name FROM rooms WHERE scope = ? AND user_id = ? ORDER BY name", ScopeUser, userID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

// recipientNames returns the session names of the recipients of a message.
func recipientNames(sent Sent) []string {
	var out []string
	for _, r := range sent.Recipients {
		out = append(out, r.Name)
	}
	return out
}

// roomList returns the rooms of the session as scope:name, with * for the rooms it joined.
func roomList(t *testing.T, s *Store, sess Session, kind string, joinedOnly bool) []string {
	t.Helper()
	rooms, err := s.Rooms(context.Background(), sess.ID, kind, joinedOnly)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range rooms {
		name := r.Scope.Kind + ":" + r.Name
		if r.Joined {
			name += "*"
		}
		out = append(out, name)
	}
	return out
}

// A message to a created room reaches its members only; creating and posting join the session,
// and leaving stops the delivery.
func TestRoomMembers(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	a := join(t, s, local, "a")
	b := join(t, s, local, "b")
	c := join(t, s, local, "c")

	res, err := s.CreateRoom(ctx, a.ID, ScopeUser, "release")
	if err != nil || !res.Created || !res.Joined || res.Name != "release" || res.Scope != (Scope{Kind: ScopeUser, ID: LocalUser}) {
		t.Fatalf("create = %+v, %v", res, err)
	}
	// Creating again only joins.
	if res, err := s.CreateRoom(ctx, b.ID, ScopeUser, "release"); err != nil || res.Created || !res.Joined {
		t.Fatalf("create again = %+v, %v", res, err)
	}
	sent, err := sendRoom(s, a, ScopeUser, "release", "to members")
	if err != nil || !slices.Equal(recipientNames(sent), []string{"b"}) {
		t.Fatalf("to members = %+v, %v", sent, err)
	}
	// The lounge still reaches everyone.
	if sent, err := sendRoom(s, a, ScopeUser, LoungeRoom, "to all"); err != nil || !slices.Equal(recipientNames(sent), []string{"b", "c"}) {
		t.Fatalf("lounge = %+v, %v", sent, err)
	}
	// Posting joins c, so that the reply of a reaches it.
	if _, err := sendRoom(s, c, ScopeUser, "release", "from c"); err != nil {
		t.Fatal(err)
	}
	if sent, err := sendRoom(s, a, ScopeUser, "release", "reply"); err != nil || !slices.Equal(recipientNames(sent), []string{"b", "c"}) {
		t.Fatalf("after c posted = %+v, %v", sent, err)
	}
	if res, err := s.LeaveRoom(ctx, b.ID, ScopeUser, "release"); err != nil || res.Joined {
		t.Fatalf("leave = %+v, %v", res, err)
	}
	if res, err := s.LeaveRoom(ctx, b.ID, ScopeUser, "release"); err != nil || res.Joined {
		t.Fatalf("leave twice = %+v, %v", res, err)
	}
	if sent, err := sendRoom(s, a, ScopeUser, "release", "after leave"); err != nil || !slices.Equal(recipientNames(sent), []string{"c"}) {
		t.Fatalf("after leave = %+v, %v", sent, err)
	}
	if res, err := s.JoinRoom(ctx, b.ID, ScopeUser, "release"); err != nil || !res.Joined {
		t.Fatalf("join = %+v, %v", res, err)
	}

	// Errors: a missing room, leaving the lounge.
	if _, err := sendRoom(s, a, ScopeUser, "missing", "x"); !errors.Is(err, ErrNoSuchRoom) {
		t.Errorf("post to a missing room: %v", err)
	}
	if _, err := s.JoinRoom(ctx, a.ID, ScopeUser, "missing"); !errors.Is(err, ErrNoSuchRoom) {
		t.Errorf("join a missing room: %v", err)
	}
	if _, err := s.LeaveRoom(ctx, a.ID, ScopeUser, LoungeRoom); !errors.Is(err, ErrLeaveLounge) {
		t.Errorf("leave the lounge: %v", err)
	}
	if res, err := s.CreateRoom(ctx, a.ID, ScopeUser, LoungeRoom); err != nil || res.Created || !res.Joined {
		t.Errorf("create the lounge = %+v, %v", res, err)
	}

	// Listing: the lounge first, then the created rooms; joined ones are marked.
	if got := roomList(t, s, c, ScopeUser, false); !slices.Equal(got, []string{"user:lounge*", "user:release*"}) {
		t.Errorf("rooms of c = %v", got)
	}
	if _, err := s.CreateRoom(ctx, a.ID, ScopeUser, "alpha"); err != nil {
		t.Fatal(err)
	}
	if got := roomList(t, s, c, "", false); !slices.Equal(got, []string{"user:lounge*", "user:alpha", "user:release*", "shared:lounge*"}) {
		t.Errorf("all rooms of c = %v", got)
	}
	if got := roomList(t, s, c, "", true); !slices.Equal(got, []string{"user:lounge*", "user:release*", "shared:lounge*"}) {
		t.Errorf("joined rooms of c = %v", got)
	}
	rooms, err := s.Rooms(ctx, a.ID, ScopeUser, false)
	if err != nil || rooms[0].Members != 2 || rooms[1].Name != "alpha" || rooms[1].Members != 0 || rooms[2].Members != 2 {
		t.Errorf("members = %+v, %v", rooms, err)
	}
}

// Rooms belong to their scope: a repository room reaches only the members that declare the
// repository now, and a member that moves away keeps its membership for when it comes back.
func TestRoomScopeAndMembership(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	bob := addUser(t, s, "bob")
	repo := &Location{Origin: "github.com/org/app"}
	a := joinWith(t, s, JoinParams{UserID: local, Name: "a", Location: repo})
	b := joinWith(t, s, JoinParams{UserID: bob.ID, Name: "b", Location: repo})

	if _, err := s.CreateRoom(ctx, a.ID, ScopeRepository, "review"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.JoinRoom(ctx, b.ID, ScopeRepository, "review"); err != nil {
		t.Fatal(err)
	}
	// The same name in another scope is another room.
	if _, err := sendRoom(s, a, ScopeUser, "review", "x"); !errors.Is(err, ErrNoSuchRoom) {
		t.Errorf("user scope: %v", err)
	}
	if sent, err := sendRoom(s, a, ScopeRepository, "review", "1"); err != nil || len(sent.Recipients) != 1 {
		t.Fatalf("before the move = %+v, %v", sent, err)
	}
	// b moves to another repository: it no longer gets the room's messages.
	joinWith(t, s, JoinParams{UserID: bob.ID, Name: "b", Location: &Location{Origin: "github.com/org/other"}})
	if sent, err := sendRoom(s, a, ScopeRepository, "review", "2"); err != nil || len(sent.Recipients) != 0 {
		t.Fatalf("after the move = %+v, %v", sent, err)
	}
	// Back in the repository, its membership counts again.
	joinWith(t, s, JoinParams{UserID: bob.ID, Name: "b", Location: repo})
	if sent, err := sendRoom(s, a, ScopeRepository, "review", "3"); err != nil || len(sent.Recipients) != 1 {
		t.Fatalf("after coming back = %+v, %v", sent, err)
	}
	// A session without a repository cannot use the scope.
	c := join(t, s, local, "c")
	if _, err := s.CreateRoom(ctx, c.ID, ScopeRepository, "x"); !errors.Is(err, ErrNoRepository) {
		t.Errorf("no repository: %v", err)
	}
	if _, err := s.Rooms(ctx, c.ID, ScopeBranch, false); !errors.Is(err, ErrNoBranch) {
		t.Errorf("no branch: %v", err)
	}
}

// The number of created rooms of a scope is limited; the lounge does not count.
func TestRoomLimit(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	l := s.Limits()
	l.MaxRoomsPerScope = 2
	s.SetLimits(l)
	a := join(t, s, local, "a")
	for _, name := range []string{"one", "two"} {
		if _, err := s.CreateRoom(ctx, a.ID, ScopeUser, name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CreateRoom(ctx, a.ID, ScopeUser, "three"); !errors.Is(err, ErrTooManyRooms) {
		t.Errorf("third room: %v", err)
	}
	// Joining an existing room and the shared scope's own limit still work.
	if res, err := s.CreateRoom(ctx, a.ID, ScopeUser, "one"); err != nil || res.Created {
		t.Errorf("existing room = %+v, %v", res, err)
	}
	if _, err := s.CreateRoom(ctx, a.ID, ScopeShared, "three"); err != nil {
		t.Errorf("another scope: %v", err)
	}
}

// The cleanup deletes the created rooms without members and messages, and never the lounge.
func TestCleanupDeletesUnusedRooms(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	a := join(t, s, local, "a")
	for _, name := range []string{"empty", "talked", "kept"} {
		if _, err := s.CreateRoom(ctx, a.ID, ScopeUser, name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sendRoom(s, a, ScopeUser, "talked", "x"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"empty", "talked"} {
		if _, err := s.LeaveRoom(ctx, a.ID, ScopeUser, name); err != nil {
			t.Fatal(err)
		}
	}
	r, err := s.Cleanup(ctx, CleanupPolicy{})
	if err != nil || r.Rooms != 1 {
		t.Fatalf("cleanup = %+v, %v", r, err)
	}
	if got := userRooms(t, s, local); !slices.Equal(got, []string{"kept", LoungeRoom, "talked"}) {
		t.Errorf("rooms = %v", got)
	}
	// The name of a deleted room can be used again.
	if res, err := s.CreateRoom(ctx, a.ID, ScopeUser, "empty"); err != nil || !res.Created {
		t.Errorf("create again = %+v, %v", res, err)
	}
}
