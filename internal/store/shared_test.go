package store

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// Two users whose sessions declare different repositories reach each other in the shared scope,
// to one session and through the lounge.
func TestSharedScopeChat(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	bob := addUser(t, s, "bob")
	a := joinWith(t, s, JoinParams{UserID: local, Name: "a", Location: &Location{Origin: "github.com/org/a"}})
	b := joinWith(t, s, JoinParams{UserID: bob.ID, Name: "b", Location: &Location{Origin: "github.com/org/b"}})
	c := join(t, s, bob.ID, "c") // no repository

	if got := SharedScopes(a, b); !slices.Equal(got, []Scope{{Kind: ScopeShared}}) {
		t.Errorf("SharedScopes = %+v", got)
	}
	if _, err := sendTo(s, a, ScopeRepository, "bob", "b", "x"); !errors.Is(err, ErrNoSuchSession) {
		t.Errorf("repository scope across repositories: %v", err)
	}
	sent, err := sendTo(s, a, ScopeShared, "bob", "b", "direct")
	if err != nil || len(sent.Recipients) != 1 || sent.Scope != (Scope{Kind: ScopeShared}) {
		t.Fatalf("direct = %+v, %v", sent, err)
	}
	sent, err = sendRoom(s, b, ScopeShared, LoungeRoom, "room")
	if err != nil || len(sent.Recipients) != 2 {
		t.Fatalf("room = %+v, %v", sent, err)
	}
	// A session without a repository can use it as well.
	if _, err := sendRoom(s, c, ScopeShared, LoungeRoom, "no repository"); err != nil {
		t.Errorf("session without repository: %v", err)
	}
	msgs, err := s.FetchInbox(ctx, a.ID, 10)
	if err != nil || len(msgs) != 2 || msgs[0].Scope != (Scope{Kind: ScopeShared}) || msgs[0].FromUser != "bob" {
		t.Fatalf("inbox = %+v, %v", msgs, err)
	}
	_, sessions, _, err := s.VisibleSessions(ctx, a.ID, 100)
	if err != nil || len(sessions) != 3 {
		t.Errorf("visible = %+v, %v", sessions, err)
	}
}

// Shared memories are created by one user and used by another, and stay when their creator is
// removed. They count as one scope for the limit.
func TestSharedMemories(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	bob := addUser(t, s, "bob")
	a := join(t, s, local, "a")
	b := join(t, s, bob.ID, "b")

	m := createMemory(t, s, b, ScopeShared, "shared build knowledge")
	if m.Scope != (Scope{Kind: ScopeShared}) {
		t.Fatalf("scope = %+v", m.Scope)
	}
	if found := search(t, s, a, MemorySearch{Query: "build"}); len(found) != 1 || found[0].ID != m.ID {
		t.Fatalf("search by another user = %+v", found)
	}
	if found := search(t, s, a, MemorySearch{Scope: ScopeShared}); len(found) != 1 {
		t.Fatalf("search in shared = %+v", found)
	}
	upd, err := s.UpdateMemory(ctx, a.ID, m.ID, MemoryInput{Body: sp("updated by local")})
	if err != nil || upd.Body != "updated by local" {
		t.Fatalf("update = %+v, %v", upd, err)
	}
	if _, err := s.PinMemory(ctx, a.ID, m.ID, true); err != nil {
		t.Errorf("pin: %v", err)
	}

	// Removing the creator keeps the shared memory.
	if _, err := s.RemoveUser(ctx, "bob"); err != nil {
		t.Fatal(err)
	}
	if found := search(t, s, a, MemorySearch{Scope: ScopeShared}); len(found) != 1 {
		t.Fatalf("after removing the creator = %+v", found)
	}
	if err := s.ForgetMemory(ctx, a.ID, m.ID); err != nil {
		t.Errorf("delete by another user: %v", err)
	}
	if found := search(t, s, a, MemorySearch{Scope: ScopeShared}); len(found) != 0 {
		t.Errorf("after delete = %+v", found)
	}
}

func TestSharedMemoryLimit(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	s.limits.MaxMemoriesPerScope = 1
	bob := addUser(t, s, "bob")
	a := join(t, s, local, "a")
	b := join(t, s, bob.ID, "b")
	createMemory(t, s, a, ScopeShared, "one")
	if _, err := s.CreateMemory(ctx, b.ID, ScopeShared, MemoryInput{Body: sp("two")}); !errors.Is(err, ErrTooManyMemories) {
		t.Errorf("second shared memory: %v", err)
	}
	createMemory(t, s, b, ScopeUser, "own scope is separate")
}
