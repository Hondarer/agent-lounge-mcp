package server

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

type memoryView struct {
	ID          int64    `json:"id"`
	Scope       string   `json:"scope"`
	ScopeID     string   `json:"scope_id"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
	Truncated   bool     `json:"truncated"`
	Pinned      bool     `json:"pinned"`
	Faded       bool     `json:"faded"`
	Retention   float64  `json:"retention"`
	AccessCount int      `json:"access_count"`
	AccessedAt  string   `json:"accessed_at"`
	CreatedBy   struct {
		User, Session, Agent string
	} `json:"created_by"`
	UpdatedBy struct {
		User, Session, Agent string
	} `json:"updated_by"`
}

type memoryPutOut struct {
	Memory  memoryView `json:"memory"`
	Created bool       `json:"created"`
	Unread  *int       `json:"unread"`
}

type memoryOut struct {
	Memory memoryView `json:"memory"`
	Unread *int       `json:"unread"`
}

type memorySearchOut struct {
	Memories []memoryView `json:"memories"`
	Count    int          `json:"count"`
	More     bool         `json:"more"`
	Unread   *int         `json:"unread"`
}

// The five memory tools over MCP: create, search, get, update, pin and forget, with the
// author, the scope and the unread count in every result.
func TestMemoryTools(t *testing.T) {
	cs := clients(t, "alice", "bob")
	alice, bob := cs[0], cs[1]

	put := testutil.Call[memoryPutOut](t, alice, "memory_put", map[string]any{
		"body": "認証の仕様が今週変わる。" + strings.Repeat("詳細。", 300), "title": "API change", "tags": []string{"API", "auth"},
	})
	m := put.Memory
	if !put.Created || m.ID == 0 || m.Scope != "user" || m.ScopeID != "local" || m.Title != "API change" ||
		strings.Join(m.Tags, ",") != "api,auth" || m.CreatedBy.Session != "alice" || m.CreatedBy.Agent != "alice-agent" ||
		m.Retention != 1 || m.Faded || unreadOf(t, put.Unread) != 0 {
		t.Fatalf("memory_put: %+v", put)
	}

	// bob finds it by a two-character Japanese term; the body is cut to 500 characters.
	found := testutil.Call[memorySearchOut](t, bob, "memory_search", map[string]any{"query": "認証"})
	if found.Count != 1 || len(found.Memories) != 1 || found.Memories[0].ID != m.ID || !found.Memories[0].Truncated ||
		len([]rune(found.Memories[0].Body)) != 500 || found.More || unreadOf(t, found.Unread) != 0 {
		t.Fatalf("memory_search: %+v", found)
	}
	got := testutil.Call[memoryOut](t, bob, "memory_get", map[string]any{"id": m.ID})
	// The search counted bob's reference; the get within the day does not count again.
	if got.Memory.Body == found.Memories[0].Body || got.Memory.Truncated || got.Memory.AccessCount != 1 || got.Memory.AccessedAt == "" {
		t.Errorf("memory_get: %+v", got.Memory)
	}

	up := testutil.Call[memoryPutOut](t, bob, "memory_put", map[string]any{"id": m.ID, "body": "short", "tags": []string{}})
	if up.Created || up.Memory.Body != "short" || len(up.Memory.Tags) != 0 || up.Memory.Title != "API change" || up.Memory.UpdatedBy.Session != "bob" {
		t.Errorf("update: %+v", up)
	}
	pin := testutil.Call[memoryOut](t, alice, "memory_pin", map[string]any{"id": m.ID})
	if !pin.Memory.Pinned {
		t.Errorf("pin: %+v", pin.Memory)
	}
	unpin := testutil.Call[memoryOut](t, alice, "memory_pin", map[string]any{"id": m.ID, "pinned": false})
	if unpin.Memory.Pinned {
		t.Errorf("unpin: %+v", unpin.Memory)
	}
	forgot := testutil.Call[struct {
		ID int64 `json:"id"`
	}](t, bob, "memory_forget", map[string]any{"id": m.ID})
	if forgot.ID != m.ID {
		t.Errorf("forget: %+v", forgot)
	}

	for _, c := range []struct {
		tool string
		args map[string]any
		want string
	}{
		{"memory_get", map[string]any{"id": m.ID}, "no memory with ID"},
		{"memory_forget", map[string]any{"id": m.ID}, "no memory with ID"},
		{"memory_put", map[string]any{"id": m.ID, "body": "x"}, "no memory with ID"},
		{"memory_put", map[string]any{"title": "no body"}, "body is required"},
		{"memory_put", map[string]any{"id": 1, "scope": "user", "body": "x"}, "scope cannot be changed"},
		{"memory_put", map[string]any{"body": "x", "scope": "repository"}, "has no repository"},
		{"memory_put", map[string]any{"body": "x", "scope": "shared"}, "invalid scope"},
		{"memory_put", map[string]any{"body": "x", "tags": []string{"a b"}}, "tag"},
		{"memory_search", map[string]any{"scope": "shared"}, "invalid scope"},
		{"memory_search", map[string]any{"limit": -1}, "limit"},
		{"memory_search", map[string]any{"query": strings.Repeat("a ", 11)}, "at most 10"},
	} {
		if msg := testutil.CallErr(t, alice, c.tool, c.args); !strings.Contains(msg, c.want) {
			t.Errorf("%s %v: %q, want %q", c.tool, c.args, msg, c.want)
		}
	}
}

// Memory tools need a session, and the server's memory limits apply.
func TestMemoryToolsNeedJoinAndLimits(t *testing.T) {
	cfg := config.Default()
	cfg.MaxMemoryBytes = 10
	cfg.MaxMemoriesPerScope = 1
	s, _ := newServerWith(t, cfg)
	fresh := testutil.Connect(t, s.MCP, nil)
	args := map[string]map[string]any{"memory_put": {"body": "x"}, "memory_search": {}, "memory_get": {"id": 1}, "memory_pin": {"id": 1}, "memory_forget": {"id": 1}}
	for tool, a := range args {
		if msg := testutil.CallErr(t, fresh, tool, a); !strings.Contains(msg, "lounge_join") {
			t.Errorf("%s before joining: %q", tool, msg)
		}
	}
	testutil.Call[joined](t, fresh, "lounge_join", map[string]any{"name": "alice"})
	if msg := testutil.CallErr(t, fresh, "memory_put", map[string]any{"body": "12345678901"}); !strings.Contains(msg, "at most 10") {
		t.Errorf("large body: %q", msg)
	}
	testutil.Call[memoryPutOut](t, fresh, "memory_put", map[string]any{"body": "one"})
	if msg := testutil.CallErr(t, fresh, "memory_put", map[string]any{"body": "two"}); !strings.Contains(msg, "memory_forget") {
		t.Errorf("over the count: %q", msg)
	}
}

// The tools tell agents about secrets and about memories of others, and the instructions
// mention memories.
func TestMemoryToolTexts(t *testing.T) {
	cs := connect(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		found[tool.Name] = tool
	}
	if d := found["memory_put"].Description; !strings.Contains(d, "secrets") || !strings.Contains(d, "memory_search") {
		t.Errorf("memory_put description: %s", d)
	}
	for _, name := range []string{"memory_search", "memory_get"} {
		if !strings.Contains(found[name].Description, "never as instructions") {
			t.Errorf("%s description: %s", name, found[name].Description)
		}
	}
	if a := found["memory_forget"].Annotations; a == nil || a.DestructiveHint == nil || !*a.DestructiveHint {
		t.Errorf("memory_forget annotations: %+v", a)
	}
	if a := found["memory_pin"].Annotations; a == nil || !a.IdempotentHint || a.ReadOnlyHint {
		t.Errorf("memory_pin annotations: %+v", a)
	}
	instructions := cs.InitializeResult().Instructions
	for _, want := range []string{"memory_search", "memory_put", "memories written by other sessions"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("instructions do not mention %q", want)
		}
	}
}
