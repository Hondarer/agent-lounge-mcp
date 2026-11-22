package export

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

func TestSafeName(t *testing.T) {
	for in, want := range map[string]string{
		"user-local":   "user-local",
		"room-lounge":  "room-lounge",
		"A.b_c-9":      "A.b_c-9",
		"repo-a/b":     "repo-a_b-",
		"repo-a:b":     "repo-a_b-",
		"x\\y z":       "x_y_z-",
		"日本":           "__-",
		"direct-a--b/": "direct-a--b_-",
	} {
		got := SafeName(in)
		if strings.HasSuffix(want, "-") {
			if !strings.HasPrefix(got, want) || len(got) != len(want)+8 || strings.Trim(got[len(want):], "0123456789abcdef") != "" {
				t.Errorf("SafeName(%q) = %q, want %q plus 8 hex digits", in, got, want)
			}
		} else if got != want {
			t.Errorf("SafeName(%q) = %q, want %q", in, got, want)
		}
	}
	if SafeName("repo-a/b") == SafeName("repo-a:b") {
		t.Error("names differing in replaced characters collide")
	}
	if SafeName("repo-a/b") != SafeName("repo-a/b") {
		t.Error("SafeName is not deterministic")
	}
}

func TestFence(t *testing.T) {
	for body, want := range map[string]string{
		"plain":          "```",
		"`one`":          "```",
		"``` code ```":   "````",
		"a ```` b ``` c": "`````",
		"````````":       "`````````",
	} {
		if got := Fence(body); got != want {
			t.Errorf("Fence(%q) = %q, want %q", body, got, want)
		}
	}
}

var t0 = time.Date(2026, 11, 18, 1, 2, 3, 0, time.UTC)

func messages() []store.ExportMessage {
	return []store.ExportMessage{
		{ID: 1, Scope: store.Scope{Kind: "repository", ID: "github.com/org/a"}, Room: "lounge", FromUser: "local", From: "alice",
			FromAgent: "claude", SentAt: t0, Body: "# not a heading\n```\ncode\n```",
			Deliveries: []store.ExportDelivery{
				{User: "bob", Name: "b", State: store.StateAcked, QueuedAt: t0, DeliveredAt: t0.Add(time.Minute), AckedAt: t0.Add(2 * time.Minute)},
				{User: "local", Name: "carol", State: store.StateQueued, QueuedAt: t0},
			}},
		{ID: 3, Scope: store.Scope{Kind: "repository", ID: "github.com/org/a"}, Room: "lounge", FromUser: "bob", From: "b",
			SentAt: t0.Add(time.Hour), Body: "second", Deliveries: []store.ExportDelivery{}},
		{ID: 2, Scope: store.Scope{Kind: "user", ID: "local"}, ToUser: "local", ToName: "alice", FromUser: "local", From: "bob",
			FromAgent: "codex", SentAt: t0, Body: "to alice",
			Deliveries: []store.ExportDelivery{{User: "local", Name: "alice", State: store.StateExpired, QueuedAt: t0, ExpiredAt: t0.Add(time.Hour)}}},
		{ID: 4, Scope: store.Scope{Kind: "user", ID: "local"}, ToUser: "local", ToName: "bob", FromUser: "local", From: "alice",
			SentAt: t0.Add(time.Hour), Body: "to bob",
			Deliveries: []store.ExportDelivery{{User: "local", Name: "bob", State: store.StateDelivered, QueuedAt: t0, DeliveredAt: t0.Add(2 * time.Hour)}}},
	}
}

func feed(msgs []store.ExportMessage, mems ...store.Memory) Reader {
	return func(fn func(store.ExportMessage) error, fm func(store.Memory) error) error {
		for _, m := range msgs {
			if err := fn(m); err != nil {
				return err
			}
		}
		for _, m := range mems {
			if err := fm(m); err != nil {
				return err
			}
		}
		return nil
	}
}

func files(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

func TestWriteMarkdown(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out") // created by WriteMarkdown
	sum, err := WriteMarkdown(dir, feed(messages()))
	if err != nil {
		t.Fatal(err)
	}
	if sum != (Summary{Files: 2, Messages: 4}) {
		t.Errorf("summary = %+v", sum)
	}
	repoDir := "chat/" + SafeName("repository-github.com/org/a")
	direct := "chat/user-local/" + SafeName("direct-local/alice--local/bob") + ".md"
	if got, want := files(t, dir), []string{repoDir + "/room-lounge.md", direct}; !slices.Equal(got, want) {
		t.Fatalf("files = %v, want %v", got, want)
	}

	room, _ := os.ReadFile(filepath.Join(dir, repoDir, "room-lounge.md"))
	text := string(room)
	for _, want := range []string{
		"- Scope: `repository`\n", "- Scope ID: `github.com/org/a`\n", "- Room: `lounge`\n",
		"## 2026-11-18T01:02:03Z `local/alice` (`claude`) message 1\n",
		"- `bob/b`: acked 2026-11-18T01:04:03Z\n", "- `local/carol`: queued 2026-11-18T01:02:03Z\n",
		"````text\n# not a heading\n```\ncode\n```\n````\n",
		"## 2026-11-18T02:02:03Z `bob/b` message 3\n\nDeliveries: none\n\n```text\nsecond\n```\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("room file does not contain %q:\n%s", want, text)
		}
	}
	// Outside the code blocks, the only headings are the file heading and the message
	// headings: the body cannot close its block early.
	var headings []string
	fence := ""
	for _, line := range strings.Split(text, "\n") {
		switch {
		case fence == "" && strings.HasPrefix(line, "```"):
			fence = strings.TrimSuffix(line, "text")
		case fence != "" && line == fence:
			fence = ""
		case fence == "" && strings.HasPrefix(line, "#"):
			headings = append(headings, line)
		}
	}
	if len(headings) != 3 {
		t.Errorf("headings = %q", headings)
	}

	b, _ := os.ReadFile(filepath.Join(dir, direct))
	text = string(b)
	for _, want := range []string{"- Between: `local/alice` and `local/bob`\n", "- `local/alice`: expired 2026-11-18T02:02:03Z\n",
		"- `local/bob`: delivered 2026-11-18T03:02:03Z\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("direct file does not contain %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "to alice") > strings.Index(text, "to bob") {
		t.Error("messages are not oldest first")
	}

	// Writing again overwrites the files of the same name and leaves other files alone.
	other := filepath.Join(dir, "chat", "keep.txt")
	if err := os.WriteFile(other, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteMarkdown(dir, feed(messages()[:1])); err != nil {
		t.Fatal(err)
	}
	room, _ = os.ReadFile(filepath.Join(dir, repoDir, "room-lounge.md"))
	if strings.Contains(string(room), "second") || strings.Count(string(room), "# agent-lounge conversation") != 1 {
		t.Errorf("room file was not overwritten:\n%s", room)
	}
	if b, err := os.ReadFile(other); err != nil || string(b) != "keep" {
		t.Errorf("other file: %q %v", b, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, direct)); !strings.Contains(string(b), "to bob") {
		t.Error("a file not written again was changed")
	}
}

// A conversation that comes in two runs is continued, not overwritten.
func TestWriteMarkdownInterleaved(t *testing.T) {
	dir := t.TempDir()
	ms := messages()
	sum, err := WriteMarkdown(dir, feed([]store.ExportMessage{ms[0], ms[2], ms[1]}))
	if err != nil || sum.Files != 2 {
		t.Fatalf("%+v %v", sum, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "chat", SafeName("repository-github.com/org/a"), "room-lounge.md"))
	if !strings.Contains(string(b), "message 1") || !strings.Contains(string(b), "message 3") || strings.Count(string(b), "# agent-lounge") != 1 {
		t.Errorf("room file:\n%s", b)
	}
}

func TestWriteMarkdownMemories(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 11, 18, 1, 2, 3, 0, time.UTC)
	author := store.Author{User: "local", Session: "alice", Agent: "claude"}
	mems := []store.Memory{
		{ID: 1, Scope: store.Scope{Kind: "repository", ID: "github.com/org/a"}, Body: "repo", Tags: []string{}, CreatedAt: at, UpdatedAt: at, CreatedBy: author, UpdatedBy: author},
		{ID: 2, Scope: store.Scope{Kind: "user", ID: "local"}, Title: "Fix", Body: "has ``` inside", Tags: []string{"build", "go"}, Pinned: true,
			CreatedAt: at, UpdatedAt: at, AccessedAt: at, AccessCount: 2, CreatedBy: author, UpdatedBy: store.Author{User: "local", Session: "bob"}},
		{ID: 3, Scope: store.Scope{Kind: "user", ID: "local"}, Body: "third", Tags: []string{}, CreatedAt: at, UpdatedAt: at, CreatedBy: author, UpdatedBy: author},
	}
	sum, err := WriteMarkdown(dir, feed(messages()[:1], mems...))
	if err != nil {
		t.Fatal(err)
	}
	if sum != (Summary{Files: 3, Messages: 1, Memories: 3}) {
		t.Errorf("summary = %+v", sum)
	}
	b, err := os.ReadFile(filepath.Join(dir, "memory", "user-local.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "# agent-lounge memories\n\n- Scope: `user`\n- Scope ID: `local`\n\nMemories are written by the sessions named with them. Times are UTC.\n\n" +
		"## Memory 2: `Fix`\n\n- Tags: `build`, `go`\n- Pinned: yes\n- Created: 2026-11-18T01:02:03Z by `local/alice` (`claude`)\n" +
		"- Updated: 2026-11-18T01:02:03Z by `local/bob`\n- Last referenced: 2026-11-18T01:02:03Z\n- References: 2\n\n````text\nhas ``` inside\n````\n\n" +
		"## Memory 3\n\n- Tags: none\n- Pinned: no\n- Created: 2026-11-18T01:02:03Z by `local/alice` (`claude`)\n" +
		"- Updated: 2026-11-18T01:02:03Z by `local/alice` (`claude`)\n- Last referenced: never\n- References: 0\n\n```text\nthird\n```\n"
	if string(b) != want {
		t.Errorf("user-local.md:\n%s\nwant:\n%s", b, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "memory", SafeName("repository-github.com/org/a")+".md")); err != nil {
		t.Error(err)
	}
}

func TestWriteMarkdownBranchScope(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	branchScope := store.Scope{Kind: store.ScopeBranch, ID: "github.com/org/a:feature/x"}
	msgs := []store.ExportMessage{{
		ID: 10, Scope: branchScope, Room: "lounge", FromUser: "local", From: "alice",
		SentAt: t0, Body: "branch message", Deliveries: []store.ExportDelivery{},
	}}
	mems := []store.Memory{{
		ID: 20, Scope: branchScope, Title: "note", Body: "branch memory", Tags: []string{},
		CreatedAt: t0, UpdatedAt: t0,
		CreatedBy: store.Author{User: "local", Session: "alice"},
		UpdatedBy: store.Author{User: "local", Session: "alice"},
	}}
	sum, err := WriteMarkdown(dir, feed(msgs, mems...))
	if err != nil {
		t.Fatal(err)
	}
	if sum != (Summary{Files: 2, Messages: 1, Memories: 1}) {
		t.Fatalf("summary = %+v", sum)
	}
	wantDir := SafeName("branch-github.com/org/a:feature/x")
	wantFiles := []string{
		"chat/" + wantDir + "/room-lounge.md",
		"memory/" + wantDir + ".md",
	}
	if got := files(t, dir); !slices.Equal(got, wantFiles) {
		t.Errorf("files = %v, want %v", got, wantFiles)
	}
}
