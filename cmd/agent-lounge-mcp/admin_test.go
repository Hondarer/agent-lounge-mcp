package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
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
	if code != 0 || stdout != "exported 2 messages to 2 files in "+filepath.Join(out, "chat")+"\n" || !strings.Contains(stderr, "every user") {
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
