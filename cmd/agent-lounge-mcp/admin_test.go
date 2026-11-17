package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/export"
	"github.com/hondarer/agent-lounge-mcp/internal/server"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

// session runs the session subcommand and returns its exit code, stdout and stderr.
func session(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runSession(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// exportCmd runs the export subcommand and returns its exit code, stdout and stderr.
func exportCmd(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runExport(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// seed creates a database with the local sessions alice and bob, bob's message to alice and
// alice's message to the room, with the store's clock at the given time.
func seed(t *testing.T, db string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.SetClock(func() time.Time { return now })
	local, err := st.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	agent := "claude"
	alice, _, err := st.JoinSession(ctx, store.JoinParams{UserID: local.ID, Name: "alice", Agent: &agent})
	if err != nil {
		t.Fatal(err)
	}
	bob, _, err := st.JoinSession(ctx, store.JoinParams{UserID: local.ID, Name: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Send(ctx, bob.ID, store.ScopeUser, store.Target{User: store.LocalUser, Name: "alice"}, "hello ``` alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Send(ctx, alice.ID, store.ScopeUser, store.Target{Room: store.LoungeRoom}, "hello room"); err != nil {
		t.Fatal(err)
	}
}

func TestSessionCommands(t *testing.T) {
	db := filepath.Join(t.TempDir(), "lounge.db")
	if code, _, stderr := session("list", "--db", db); code != 1 || !strings.Contains(stderr, "does not exist") {
		t.Errorf("list without a database: %d %q", code, stderr)
	}
	seed(t, db, time.Now())
	if code, _, _ := user("add", "zed", "--db", db); code != 0 {
		t.Fatal("user add failed")
	}

	code, stdout, _ := session("list", "--db", db)
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if code != 0 || len(lines) != 3 || lines[0] != "ADDRESS\tAGENT\tLAST_USED\tUNREAD\tREPOSITORY" ||
		!strings.HasPrefix(lines[1], "local/alice\tclaude\t") || !strings.HasSuffix(lines[1], "\t1\t-") ||
		!strings.HasPrefix(lines[2], "local/bob\t-\t") || !strings.HasSuffix(lines[2], "\t1\t-") {
		t.Errorf("list: %d %q", code, stdout)
	}
	if code, stdout, _ := session("list", "--user", "zed", "--db", db); code != 0 || stdout != "ADDRESS\tAGENT\tLAST_USED\tUNREAD\tREPOSITORY\n" {
		t.Errorf("list --user zed: %d %q", code, stdout)
	}

	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"list", "--user", "nobody", "--db", db}, 1, "no such user"},
		{[]string{"list", "x", "--db", db}, 2, "no argument"},
		{[]string{"remove", "--yes", "--db", db}, 2, "<user>/<name>"},
		{[]string{"remove", "alice", "--yes", "--db", db}, 2, "<user>/<name>"},
		{[]string{"remove", "local/alice/x", "--yes", "--db", db}, 2, "<user>/<name>"},
		{[]string{"remove", "local/nobody", "--yes", "--db", db}, 1, "no such session"},
		{[]string{"remove", "local/nobody", "--db", db}, 1, "no such session"},
		{[]string{"list", "--yes", "--db", db}, 2, "not defined"},
		{[]string{"frobnicate"}, 2, "unknown session command"},
		{nil, 2, "usage"},
	} {
		code, stdout, stderr := session(tc.args...)
		if code != tc.code || !strings.Contains(stderr, tc.want) || stdout != "" {
			t.Errorf("%v: %d %q %q, want %d and %q", tc.args, code, stdout, stderr, tc.code, tc.want)
		}
	}

	// remove without --yes shows the counts, removes nothing and exits 1, like user remove.
	code, stdout, stderr := session("remove", "local/alice", "--db", db)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "2 messages, 2 deliveries") || !strings.Contains(stderr, "--yes") {
		t.Errorf("remove without --yes: %d %q %q", code, stdout, stderr)
	}
	code, stdout, _ = session("remove", "--db", db, "local/alice", "--yes")
	if code != 0 || stdout != "removed session local/alice: 2 messages, 2 deliveries\n" {
		t.Errorf("remove: %d %q", code, stdout)
	}
	if _, stdout, _ = session("list", "--db", db); strings.Contains(stdout, "alice") || !strings.Contains(stdout, "local/bob\t-\t") {
		t.Errorf("after remove: %q", stdout)
	}
	if code, stdout, _ := session("help"); code != 0 || !strings.Contains(stdout, "session remove") {
		t.Errorf("help: %d %q", code, stdout)
	}
}

func TestExportCommand(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "lounge.db")
	out := filepath.Join(dir, "out")
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"--db", db}, 2, "--out"},
		{[]string{"--out", out, "--format", "html", "--db", db}, 2, "markdown"},
		{[]string{"--out", out, "x", "--db", db}, 2, "no argument"},
		{[]string{"--out", out, "--db", db}, 1, "does not exist"},
	} {
		code, stdout, stderr := exportCmd(tc.args...)
		if code != tc.code || !strings.Contains(stderr, tc.want) || stdout != "" {
			t.Errorf("%v: %d %q %q, want %d and %q", tc.args, code, stdout, stderr, tc.code, tc.want)
		}
	}
	seed(t, db, time.Now())
	code, stdout, stderr := exportCmd("--out", out, "--format", "markdown", "--db", db)
	if code != 0 || stdout != "exported 2 messages and 0 memories to 2 files in "+out+"\n" || !strings.Contains(stderr, "every user") {
		t.Fatalf("export: %d %q %q", code, stdout, stderr)
	}
	room, err := os.ReadFile(filepath.Join(out, "chat", "user-local", "room-lounge.md"))
	if err != nil || !strings.Contains(string(room), "hello room") || !strings.Contains(string(room), "`local/alice` (`claude`)") {
		t.Errorf("room file: %q %v", room, err)
	}
	entries, err := os.ReadDir(filepath.Join(out, "chat", "user-local"))
	if err != nil || len(entries) != 2 || !strings.HasPrefix(entries[0].Name(), "direct-local_alice--local_bob-") {
		t.Errorf("files: %v %v", entries, err)
	}
	direct, _ := os.ReadFile(filepath.Join(out, "chat", "user-local", entries[0].Name()))
	if !strings.Contains(string(direct), "````text\nhello ``` alice\n````\n") {
		t.Errorf("direct file:\n%s", direct)
	}
}

// The server runs the cleanup at start-up, and the admin commands work while it serves: a
// connection whose session is removed gets an error that says so.
func TestCleanupAndAdminWhileServing(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := config.Default()
	cfg.Listen = "127.0.0.1:0"
	cfg.DBPath = filepath.Join(t.TempDir(), "lounge.db")
	cfg.ShutdownTimeout = 100 * time.Millisecond
	// Sessions last used 61 days ago: deleted at start-up with their 2 messages.
	seed(t, cfg.DBPath, time.Now().Add(-61*24*time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, nil, cfg, ready) }()
	var addr net.Addr
	select {
	case addr = <-ready:
	case err := <-done:
		t.Fatalf("serve stopped early: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		<-done
	})
	testutil.WaitFor(t, func() bool {
		return strings.Contains(logged.String(), "cleanup: 2 unread deliveries expired; 2 unused sessions deleted (2 messages, 2 deliveries)")
	}, "the start-up cleanup is logged:\n"+logged.String())
	if out := logged.String(); !strings.Contains(out, "cleanup every 1h: unread messages expire after 14d; messages are deleted 30d after their last change of state; sessions unused for 60d are deleted") ||
		strings.Contains(out, "hello") {
		t.Errorf("log:\n%s", out)
	}

	cs := testutil.ConnectHTTP(t, "http://"+addr.String()+server.MCPPath)
	testutil.Call[struct{}](t, cs, "lounge_join", map[string]any{"name": "carol"})
	if code, stdout, _ := session("list", "--db", cfg.DBPath); code != 0 || !strings.Contains(stdout, "local/carol\t") {
		t.Errorf("list while serving: %d %q", code, stdout)
	}
	if code, _, _ := exportCmd("--out", filepath.Join(t.TempDir(), "out"), "--db", cfg.DBPath); code != 0 {
		t.Errorf("export while serving: %d", code)
	}
	if code, _, _ := session("remove", "local/carol", "--yes", "--db", cfg.DBPath); code != 0 {
		t.Errorf("remove while serving: %d", code)
	}
	if msg := testutil.CallErr(t, cs, "chat_inbox", nil); !strings.Contains(msg, "was deleted") {
		t.Errorf("call after remove: %s", msg)
	}
}

func TestDescribeCleanup(t *testing.T) {
	if got := describeCleanup(store.CleanupResult{Messages: 3, MessageDeliveries: 4}); got != "cleanup: 3 old messages deleted (4 deliveries)" {
		t.Errorf("describeCleanup = %q", got)
	}
	cfg := config.Default()
	cfg.ExpireAfter, cfg.RetainFor, cfg.SessionIdle, cfg.CleanupInterval = 0, 0, 0, 90*time.Second
	if got := describeCleanupPolicy(cfg); got != "cleanup every 1m30s: unread messages never expire; messages are never deleted; unused sessions are never deleted" {
		t.Errorf("describeCleanupPolicy = %q", got)
	}
}

// The cleanup runs at start and again every interval, with the store's clock, and stop ends it.
func TestStartCleanupRepeats(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	ctx := context.Background()
	db := filepath.Join(t.TempDir(), "lounge.db")
	start := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	seed(t, db, start)
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var mu sync.Mutex
	now := start
	st.SetClock(func() time.Time { mu.Lock(); defer mu.Unlock(); return now })

	cfg := config.Default()
	cfg.CleanupInterval = 10 * time.Millisecond
	stop := startCleanup(ctx, st, cfg)
	time.Sleep(50 * time.Millisecond)
	if strings.Contains(logged.String(), "cleanup:") {
		t.Errorf("cleanup logged with nothing to do:\n%s", logged.String())
	}
	mu.Lock()
	now = now.Add(15 * 24 * time.Hour)
	mu.Unlock()
	testutil.WaitFor(t, func() bool { return strings.Contains(logged.String(), "cleanup: 2 unread deliveries expired\n") }, "expiry not logged")
	mu.Lock()
	now = now.Add(46 * 24 * time.Hour)
	mu.Unlock()
	testutil.WaitFor(t, func() bool { return strings.Contains(logged.String(), "2 unused sessions deleted") }, "session deletion not logged")
	stop()
	if strings.Count(logged.String(), "cleanup:") != 2 {
		t.Errorf("log:\n%s", logged.String())
	}
}

// memory runs the memory subcommand and returns its exit code, stdout and stderr.
func memory(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runMemory(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// seedMemories adds a user memory with a title and a repository memory without one, created
// at now, and returns their IDs.
func seedMemories(t *testing.T, db string, now time.Time) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.SetClock(func() time.Time { return now })
	local, err := st.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sess, _, err := st.JoinSession(ctx, store.JoinParams{UserID: local.ID, Name: "carol", Location: &store.Location{Origin: "github.com/org/a"}})
	if err != nil {
		t.Fatal(err)
	}
	body, title, repo := "make clean fixes it", "Build\tfix", "first line of a long body that goes on and on\nsecond line"
	tags := []string{"build"}
	a, err := st.CreateMemory(ctx, sess.ID, store.ScopeUser, store.MemoryInput{Body: &body, Title: &title, Tags: &tags})
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.CreateMemory(ctx, sess.ID, store.ScopeRepository, store.MemoryInput{Body: &repo})
	if err != nil {
		t.Fatal(err)
	}
	return a.ID, b.ID
}

func TestMemoryCommands(t *testing.T) {
	db := filepath.Join(t.TempDir(), "lounge.db")
	if code, _, stderr := memory("list", "--db", db); code != 1 || !strings.Contains(stderr, "does not exist") {
		t.Errorf("list without a database: %d %q", code, stderr)
	}
	// Created 30 days ago: faded with the default retention.
	userID, repoID := seedMemories(t, db, time.Now().Add(-30*24*time.Hour))

	code, stdout, _ := memory("list", "--db", db)
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if code != 0 || len(lines) != 3 || lines[0] != "ID\tSCOPE\tSCOPE_ID\tSTATE\tRETENTION\tUPDATED\tCREATED_BY\tTITLE" ||
		!strings.HasPrefix(lines[1], fmt.Sprintf("%d\trepository\tgithub.com/org/a\tfaded\t0.05\t", repoID)) ||
		!strings.HasSuffix(lines[1], "\tlocal/carol\tfirst line of a long body that goes on a...") ||
		!strings.HasPrefix(lines[2], fmt.Sprintf("%d\tuser\tlocal\tfaded\t", userID)) || !strings.HasSuffix(lines[2], "\tBuild fix") {
		t.Errorf("list: %d %q", code, stdout)
	}
	// With a longer half-life, the same memories are active.
	if code, stdout, _ := memory("list", "--scope", "user", "--memory-half-life", "30d", "--db", db); code != 0 ||
		!strings.Contains(stdout, "\tuser\tlocal\tactive\t0.50\t") || strings.Contains(stdout, "repository") {
		t.Errorf("list --scope user --memory-half-life 30d: %d %q", code, stdout)
	}

	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"list", "x", "--db", db}, 2, "no argument"},
		{[]string{"list", "--scope", "shared", "--db", db}, 2, "--scope"},
		{[]string{"list", "--memory-fade-threshold", "1", "--db", db}, 2, "--memory-fade-threshold"},
		{[]string{"remove", "--yes", "--db", db}, 2, "one memory ID"},
		{[]string{"remove", "x", "--yes", "--db", db}, 2, "one memory ID"},
		{[]string{"remove", "999", "--yes", "--db", db}, 1, "no such memory"},
		{[]string{"remove", "999", "--db", db}, 1, "no such memory"},
		{[]string{"remove", fmt.Sprint(userID), "--db", db}, 1, "Build fix"},
		{[]string{"nope", "--db", db}, 2, "unknown memory command"},
	} {
		code, stdout, stderr := memory(tc.args...)
		if code != tc.code || !strings.Contains(stderr, tc.want) || stdout != "" {
			t.Errorf("%v: %d %q %q, want %d and %q", tc.args, code, stdout, stderr, tc.code, tc.want)
		}
	}
	if code, stdout, _ := memory("remove", fmt.Sprint(userID), "--yes", "--db", db); code != 0 || stdout != fmt.Sprintf("removed memory %d (scope user local: Build fix)\n", userID) {
		t.Errorf("remove: %d %q", code, stdout)
	}
	if code, stdout, _ := memory("list", "--db", db); code != 0 || strings.Count(stdout, "\n") != 2 {
		t.Errorf("list after remove: %d %q", code, stdout)
	}
	if code, stdout, _ := memory("help"); code != 0 || !strings.Contains(stdout, "memory remove") {
		t.Errorf("help: %d %q", code, stdout)
	}
}

// export writes the memories, one file per scope, after the conversations.
func TestExportMemories(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "lounge.db")
	out := filepath.Join(dir, "out")
	seedMemories(t, db, time.Now())
	code, stdout, _ := exportCmd("--out", out, "--db", db)
	if code != 0 || stdout != "exported 0 messages and 2 memories to 2 files in "+out+"\n" {
		t.Fatalf("export: %d %q", code, stdout)
	}
	b, err := os.ReadFile(filepath.Join(out, "memory", "user-local.md"))
	if err != nil || !strings.Contains(string(b), "## Memory 1: `Build\tfix`") || !strings.Contains(string(b), "- Tags: `build`") ||
		!strings.Contains(string(b), "by `local/carol`") || !strings.Contains(string(b), "```text\nmake clean fixes it\n```") {
		t.Errorf("user memories:\n%s %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(out, "memory", export.SafeName("repository-github.com/org/a")+".md")); err != nil {
		t.Errorf("repository memories: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "chat")); err == nil {
		t.Error("a chat directory without messages")
	}
}

func TestPlural(t *testing.T) {
	for n, want := range map[int]string{0: "0 memories", 1: "1 memory", 2: "2 memories"} {
		if got := plural(n, "memory", "memories"); got != want {
			t.Errorf("plural(%d) = %q", n, got)
		}
	}
	if got := describeCleanup(store.CleanupResult{Expired: 1, Sessions: 1, SessionMessages: 1, SessionDeliveries: 2, Memories: 1}); got !=
		"cleanup: 1 unread delivery expired; 1 unused session deleted (1 message, 2 deliveries); 1 forgotten memory deleted" {
		t.Errorf("describeCleanup = %q", got)
	}
	cfg := config.Default()
	if got := describeMemoryPolicy(cfg); got != "memories: half-life 7d, longer with references; they fade below a retention score of 0.1 and are deleted 30d after they fade" {
		t.Errorf("describeMemoryPolicy = %q", got)
	}
	cfg.MemoryGrace = 0
	if got := describeMemoryPolicy(cfg); !strings.HasSuffix(got, "are never deleted") {
		t.Errorf("describeMemoryPolicy without grace = %q", got)
	}
	cfg.MemoryHalfLife = 0
	if got := describeMemoryPolicy(cfg); !strings.Contains(got, "disabled") {
		t.Errorf("describeMemoryPolicy without forgetting = %q", got)
	}
}
