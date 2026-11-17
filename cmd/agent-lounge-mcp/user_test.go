package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/server"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

// user runs the user subcommand and returns its exit code, stdout and stderr.
func user(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runUser(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

var tokenLine = regexp.MustCompile(`^alm_[A-Za-z0-9_-]{43}\n$`)

func TestUserCommands(t *testing.T) {
	db := filepath.Join(t.TempDir(), "lounge.db")

	// Commands other than add need the database file.
	if code, _, stderr := user("list", "--db", db); code != 1 || !strings.Contains(stderr, "does not exist") {
		t.Errorf("list without a database: %d %q", code, stderr)
	}

	code, stdout, stderr := user("add", "alice", "--db", db)
	if code != 0 || !tokenLine.MatchString(stdout) || !strings.Contains(stderr, "only this once") || !strings.Contains(stderr, "Authorization: Bearer") {
		t.Fatalf("add: %d %q %q", code, stdout, stderr)
	}
	token := strings.TrimSpace(stdout)
	if strings.Contains(stderr, token) {
		t.Error("stderr repeats the token")
	}
	// Flags may come first.
	if code, stdout, _ := user("add", "--db", db, "bob"); code != 0 || !tokenLine.MatchString(stdout) {
		t.Errorf("add with --db first: %d %q", code, stdout)
	}

	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"add", "alice", "--db", db}, 1, "already exists"},
		{[]string{"add", "local", "--db", db}, 1, "reserved"},
		{[]string{"add", "Bad/Name", "--db", db}, 1, "invalid user name"},
		{[]string{"add", "--db", db}, 2, "one user name"},
		{[]string{"add", "a", "b", "--db", db}, 2, "one user name"},
		{[]string{"list", "x", "--db", db}, 2, "no argument"},
		{[]string{"add", "carol", "--yes", "--db", db}, 2, "not defined"},
		{[]string{"rotate", "nobody", "--db", db}, 1, "no such user"},
		{[]string{"rotate", "local", "--db", db}, 1, "reserved"},
		{[]string{"remove", "nobody", "--yes", "--db", db}, 1, "no such user"},
		{[]string{"remove", "local", "--yes", "--db", db}, 1, "reserved"},
		{[]string{"frobnicate"}, 2, "unknown user command"},
		{nil, 2, "usage"},
	} {
		code, stdout, stderr := user(tc.args...)
		if code != tc.code || !strings.Contains(stderr, tc.want) || stdout != "" {
			t.Errorf("%v: %d %q %q, want %d and %q", tc.args, code, stdout, stderr, tc.code, tc.want)
		}
	}

	// rotate prints a new token.
	code, stdout, _ = user("rotate", "alice", "--db", db)
	if code != 0 || !tokenLine.MatchString(stdout) || strings.TrimSpace(stdout) == token {
		t.Errorf("rotate: %d %q", code, stdout)
	}

	// list: tab-separated, without tokens or hashes.
	code, stdout, _ = user("list", "--db", db)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if code != 0 || len(lines) != 3 || lines[0] != "NAME\tCREATED\tSESSIONS\tLAST_USED" ||
		!strings.HasPrefix(lines[1], "alice\t") || !strings.HasSuffix(lines[1], "\t0\t-") || !strings.HasPrefix(lines[2], "bob\t") {
		t.Errorf("list: %d %q", code, stdout)
	}
	if strings.Contains(stdout, "alm_") {
		t.Error("list shows a token")
	}

	// remove without --yes shows the counts and removes nothing.
	code, stdout, stderr = user("remove", "bob", "--db", db)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "0 sessions, 0 messages, 0 deliveries") || !strings.Contains(stderr, "--yes") {
		t.Errorf("remove without --yes: %d %q %q", code, stdout, stderr)
	}
	code, stdout, _ = user("remove", "bob", "--yes", "--db", db)
	if code != 0 || stdout != "removed user bob: 0 sessions, 0 messages, 0 deliveries\n" {
		t.Errorf("remove: %d %q", code, stdout)
	}
	if _, stdout, _ = user("list", "--db", db); strings.Contains(stdout, "bob") {
		t.Errorf("bob is still listed: %q", stdout)
	}
}

// The user commands work on the database of a running server, and take effect on its next
// request: a server that requires a token accepts a token added while it runs, and rejects it
// after rotate and remove.
func TestUserCommandsWhileServing(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := config.Default()
	cfg.Listen = "127.0.0.1:0"
	cfg.DBPath = filepath.Join(t.TempDir(), "lounge.db")
	cfg.ShutdownTimeout = 100 * time.Millisecond
	cfg.RequireToken = true
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
	if !strings.Contains(logged.String(), "no user has a token yet") || strings.Contains(logged.String(), "without TLS") {
		t.Errorf("start-up log: %s", logged.String())
	}
	endpoint := "http://" + addr.String() + server.MCPPath

	if _, err := testutil.ConnectToken(t, endpoint, ""); err == nil {
		t.Fatal("connected without a token")
	}
	code, stdout, _ := user("add", "alice", "--db", cfg.DBPath)
	if code != 0 {
		t.Fatalf("add while serving: %d", code)
	}
	token := strings.TrimSpace(stdout)
	cs, err := testutil.ConnectToken(t, endpoint, token)
	if err != nil {
		t.Fatalf("token added while serving: %v", err)
	}
	type joinedUser struct {
		User string `json:"user"`
	}
	if j := testutil.Call[joinedUser](t, cs, "lounge_join", map[string]any{"name": "a"}); j.User != "alice" {
		t.Errorf("user = %q", j.User)
	}

	_, stdout, _ = user("rotate", "alice", "--db", cfg.DBPath)
	if _, err := cs.ListTools(ctx, nil); err == nil {
		t.Error("the rotated token still works")
	}
	cs, err = testutil.ConnectToken(t, endpoint, strings.TrimSpace(stdout))
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if code, stdout, _ := user("remove", "alice", "--yes", "--db", cfg.DBPath); code != 0 || !strings.Contains(stdout, "1 sessions") {
		t.Errorf("remove while serving: %d %q", code, stdout)
	}
	if _, err := cs.ListTools(ctx, nil); err == nil {
		t.Error("the removed user's token still works")
	}
}

// Listening on an address other than loopback logs the warning once.
func TestServeRemoteWarning(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	cfg := config.Default()
	cfg.Listen = "0.0.0.0:0"
	cfg.AllowRemote = true
	cfg.DBPath = filepath.Join(t.TempDir(), "lounge.db")
	cfg.ShutdownTimeout = 100 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, nil, cfg, ready) }()
	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("serve stopped early: %v", err)
	}
	cancel()
	<-done
	out := logged.String()
	want := "without TLS: tokens and messages travel unencrypted. Use it only on a trusted network, or put an HTTPS reverse proxy in front."
	if strings.Count(out, want) != 1 || !strings.Contains(out, "warning: listening on 0.0.0.0:") || !strings.Contains(out, "user add") {
		t.Errorf("log:\n%s", out)
	}
}
