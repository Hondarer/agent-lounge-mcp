package server

import (
	"slices"
	"strings"
	"testing"

	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

func TestBranchScopeEndToEnd(t *testing.T) {
	s, _ := newServer(t)
	alice := testutil.Connect(t, s.MCP, nil)
	bob := testutil.Connect(t, s.MCP, nil)
	carol := testutil.Connect(t, s.MCP, nil)

	type branchJoined struct {
		Name       string `json:"name"`
		Repository string `json:"repository"`
		Branch     string `json:"branch"`
		Hint       string `json:"hint"`
	}
	type branchWho struct {
		Repository string `json:"repository"`
		Branch     string `json:"branch"`
		Scopes     []struct {
			Scope string `json:"scope"`
			ID    string `json:"id"`
		} `json:"scopes"`
		Sessions []struct {
			Name   string `json:"name"`
			Branch string `json:"branch"`
			Scopes []struct {
				Scope string `json:"scope"`
				ID    string `json:"id"`
			} `json:"scopes"`
		} `json:"sessions"`
	}

	// Invalid branch value is rejected with guidance.
	if msg := testutil.CallErr(t, alice, "lounge_join", map[string]any{
		"name": "alice", "origin": "github.com/org/repo", "branch": "* main",
	}); !strings.Contains(msg, "git branch --show-current") {
		t.Errorf("invalid branch error: %q", msg)
	}

	j := testutil.Call[branchJoined](t, alice, "lounge_join", map[string]any{
		"name": "alice", "origin": "https://github.com/Org/Repo.git", "branch": "Feature/X\n",
	})
	if j.Repository != "github.com/org/repo" || j.Branch != "feature/x" || j.Hint != "" {
		t.Fatalf("alice join = %+v", j)
	}
	testutil.Call[branchJoined](t, bob, "lounge_join", map[string]any{
		"name": "bob", "origin": "git@github.com:org/repo.git", "branch": "feature/x",
	})
	testutil.Call[branchJoined](t, carol, "lounge_join", map[string]any{
		"name": "carol", "origin": "github.com/org/repo", "branch": "main",
	})

	// lounge_who reports top-level branch, each session's branch, and shared branch scope.
	w := testutil.Call[branchWho](t, alice, "lounge_who", nil)
	if w.Branch != "feature/x" || len(w.Scopes) != 4 || w.Scopes[2].Scope != "branch" || w.Scopes[2].ID != "github.com/org/repo:feature/x" {
		t.Fatalf("who = %+v", w)
	}
	byName := map[string]string{}
	for _, sess := range w.Sessions {
		byName[sess.Name] = sess.Branch
	}
	if byName["bob"] != "feature/x" || byName["carol"] != "main" {
		t.Errorf("who branches = %+v", byName)
	}

	// Branch room reaches bob (same branch) and not carol (other branch).
	sent := testutil.Call[scopedSent](t, alice, "chat_send", map[string]any{
		"room": "lounge", "scope": "branch", "body": "only feat",
	})
	if sent.Scope != "branch" || sent.ScopeID != "github.com/org/repo:feature/x" || !slices.Equal(sent.Recipients, []string{"bob"}) {
		t.Errorf("branch send = %+v", sent)
	}

	// Branch memory is visible to bob and hidden from carol.
	type memPutOut struct {
		Memory struct {
			ID      int64  `json:"id"`
			Scope   string `json:"scope"`
			ScopeID string `json:"scope_id"`
		} `json:"memory"`
	}
	mp := testutil.Call[memPutOut](t, alice, "memory_put", map[string]any{
		"scope": "branch", "body": "branch-only note",
	})
	if mp.Memory.Scope != "branch" || mp.Memory.ScopeID != "github.com/org/repo:feature/x" {
		t.Errorf("memory_put = %+v", mp)
	}
	if msg := testutil.CallErr(t, carol, "memory_get", map[string]any{"id": mp.Memory.ID}); !strings.Contains(msg, "no memory") {
		t.Errorf("carol memory_get = %q", msg)
	}

	// Clearing branch via location-only join disables branch scope with guidance.
	j = testutil.Call[branchJoined](t, alice, "lounge_join", map[string]any{
		"name": "alice", "origin": "github.com/org/repo",
	})
	if j.Branch != "" || !strings.Contains(j.Hint, "git branch --show-current") {
		t.Errorf("location-only join = %+v", j)
	}
	if msg := testutil.CallErr(t, alice, "chat_send", map[string]any{
		"room": "lounge", "scope": "branch", "body": "fail",
	}); !strings.Contains(msg, "git branch --show-current") || !strings.Contains(msg, "detached HEAD") {
		t.Errorf("branch send without branch = %q", msg)
	}
	if msg := testutil.CallErr(t, alice, "memory_put", map[string]any{
		"scope": "branch", "body": "fail",
	}); !strings.Contains(msg, "git branch --show-current") || !strings.Contains(msg, "detached HEAD") {
		t.Errorf("branch memory_put without branch = %q", msg)
	}
}
