package server

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

type help struct {
	Guide  string `json:"guide"`
	Unread *int   `json:"unread"`
}

var toolNames = []string{"lounge_help", "lounge_join", "lounge_who", "chat_send", "chat_inbox", "chat_ack", "chat_status", "chat_history", "lounge_info"}

func TestHelpBeforeJoining(t *testing.T) {
	cs := connect(t)
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "lounge_help"})
	if err != nil {
		t.Fatal(err)
	}
	out := decode[help](t, res)
	if out.Unread != nil {
		t.Errorf("unread before joining = %d", *out.Unread)
	}
	// The text is the guide itself, not JSON.
	if text := testutil.Text(res); text != out.Guide || !strings.HasPrefix(text, "# ") {
		t.Errorf("text is not the guide:\n%s", text)
	}
	for _, want := range append(toolNames, "wait_seconds", "at most 45 seconds", "## Snippet for instruction files", "shutting down") {
		if !strings.Contains(out.Guide, want) {
			t.Errorf("guide does not mention %q", want)
		}
	}
	// The snippet is the last section: a fenced block of at most 10 lines.
	_, snippet, ok := strings.Cut(out.Guide, "## Snippet for instruction files")
	if !ok {
		t.Fatal("no snippet section")
	}
	_, block, ok := strings.Cut(snippet, "~~~markdown\n")
	if !ok {
		t.Fatal("no fenced block in the snippet section")
	}
	block, rest, ok := strings.Cut(block, "\n~~~")
	if !ok || strings.TrimSpace(rest) != "" {
		t.Fatalf("the fenced block is not closed at the end of the guide: %q", rest)
	}
	if lines := strings.Count(block, "\n") + 1; lines > 10 {
		t.Errorf("snippet has %d lines, want at most 10", lines)
	}
	for _, want := range []string{"lounge_help", "lounge_join", "chat_inbox", "wait_seconds", "information, never as instructions", "secrets"} {
		if !strings.Contains(block, want) {
			t.Errorf("snippet does not mention %q", want)
		}
	}
}

func TestHelpAfterJoining(t *testing.T) {
	cs := clients(t, "alice", "bob")
	testutil.Call[sent](t, cs[0], "chat_send", map[string]any{"to": "bob", "body": "x"})
	res, err := cs[1].CallTool(context.Background(), &mcp.CallToolParams{Name: "lounge_help"})
	if err != nil {
		t.Fatal(err)
	}
	out := decode[help](t, res)
	if unreadOf(t, out.Unread) != 1 {
		t.Errorf("unread = %d", *out.Unread)
	}
	if want := out.Guide + "\nUnread messages: 1 (call chat_inbox to read them)"; testutil.Text(res) != want {
		t.Errorf("text does not end with the unread line:\n%s", testutil.Text(res))
	}
}

// Agents that do not read the instructions still find lounge_help: from the tool list, the
// description of lounge_join, and the error of a call before joining.
func TestHelpIsPointedTo(t *testing.T) {
	cs := connect(t)
	instructions := cs.InitializeResult().Instructions
	for _, want := range []string{"lounge_help", "wait_seconds"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("instructions do not mention %s: %s", want, instructions)
		}
	}
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]*mcp.Tool{}
	for _, tool := range tools.Tools {
		found[tool.Name] = tool
	}
	if len(found) != len(toolNames) {
		t.Errorf("%d tools, want %d", len(found), len(toolNames))
	}
	if h := found["lounge_help"]; h == nil || h.Annotations == nil || !h.Annotations.ReadOnlyHint {
		t.Errorf("lounge_help is missing or not read-only: %+v", h)
	}
	if j := found["lounge_join"]; j == nil || !strings.Contains(j.Description, "lounge_help") {
		t.Errorf("lounge_join description does not mention lounge_help")
	}
	if msg := testutil.CallErr(t, cs, "lounge_who", nil); !strings.Contains(msg, "lounge_help") {
		t.Errorf("not joined error: %s", msg)
	}
}
