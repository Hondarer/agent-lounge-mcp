package main

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/auth"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

var expiresLine = regexp.MustCompile(`expires at (\S+)\.\n`)

// within checks that an RFC 3339 time printed by a command is d after a time between before
// and after (the output has whole seconds).
func within(t *testing.T, what, printed string, before, after time.Time, d time.Duration) {
	t.Helper()
	got, err := time.Parse(time.RFC3339, printed)
	if err != nil || !strings.HasSuffix(printed, "Z") {
		t.Errorf("%s: %q is not RFC 3339 in UTC: %v", what, printed, err)
		return
	}
	if got.Before(before.Add(d).Truncate(time.Second)) || got.After(after.Add(d)) {
		t.Errorf("%s: %v, want %v after a time between %v and %v", what, got, d, before, after)
	}
}

// listLine returns the user list line of the user, split into its columns.
func listLine(t *testing.T, db, name string) []string {
	t.Helper()
	code, stdout, stderr := user("list", "--db", db)
	if code != 0 {
		t.Fatalf("list: %d %q", code, stderr)
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		cols := strings.Split(line, "\t")
		if cols[0] == name {
			if len(cols) != 5 {
				t.Fatalf("list line %q", line)
			}
			return cols
		}
	}
	t.Fatalf("user %s is not listed: %q", name, stdout)
	return nil
}

func TestUserExpiryCommands(t *testing.T) {
	db := filepath.Join(t.TempDir(), "lounge.db")

	// add without --expires-in: no expiry.
	code, stdout, stderr := user("add", "bob", "--db", db)
	if code != 0 || !tokenLine.MatchString(stdout) || !strings.Contains(stderr, "The token of user bob does not expire.\n") {
		t.Fatalf("add: %d %q %q", code, stdout, stderr)
	}
	// add with --expires-in: only the token on stdout, the expiry on stderr.
	before := time.Now()
	code, stdout, stderr = user("add", "alice", "--expires-in", "90d", "--db", db)
	after := time.Now()
	if code != 0 || !tokenLine.MatchString(stdout) {
		t.Fatalf("add --expires-in: %d %q %q", code, stdout, stderr)
	}
	m := expiresLine.FindStringSubmatch(stderr)
	if m == nil || !strings.Contains(stderr, "The token of user alice expires at ") {
		t.Fatalf("add --expires-in guidance: %q", stderr)
	}
	within(t, "add --expires-in 90d", m[1], before, after, 90*24*time.Hour)
	if cols := listLine(t, db, "alice"); cols[4] != m[1] || cols[3] != "-" {
		t.Errorf("list after add: %q, want EXPIRES %s", cols, m[1])
	}
	if cols := listLine(t, db, "bob"); cols[4] != "-" {
		t.Errorf("bob: %q", cols)
	}
	// local, which a server creates.
	st, err := store.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.EnsureLocalUser(context.Background()); err != nil {
		t.Fatal(err)
	}
	st.Close()
	if cols := listLine(t, db, store.LocalUser); cols[4] != "-" || cols[3] != "-" {
		t.Errorf("local: %q", cols)
	}

	// Malformed command lines: usage errors.
	for _, args := range [][]string{
		{"add", "carol", "--expires-in", "0"},
		{"add", "carol", "--expires-in", "0s"},
		{"add", "carol", "--expires-in", "-1h"},
		{"add", "carol", "--expires-in", "soon"},
		{"rotate", "alice", "--expires-in", "0"},
		{"expiry", "alice"},
		{"expiry", "alice", "--in", "1d", "--never"},
		{"expiry", "alice", "--in", "0"},
		{"expiry", "alice", "--in", "-1d"},
		{"expiry", "alice", "--expires-in", "1d"},
		{"expiry", "--never"},
		{"list", "--never"},
	} {
		code, stdout, stderr := user(append(args, "--db", db)...)
		if code != 2 || stdout != "" || stderr == "" {
			t.Errorf("%v: %d %q %q", args, code, stdout, stderr)
		}
	}
	if code, _, _ := user("expiry", "alice", "--db", db); code != 2 {
		t.Errorf("expiry without --in or --never: %d", code)
	}
	if _, _, stderr := user("expiry", "alice", "--db", db); !strings.Contains(stderr, "one of --in DURATION and --never") {
		t.Errorf("expiry without --in or --never: %q", stderr)
	}
	if _, stdout, _ := user("list", "--db", db); strings.Contains(stdout, "carol") {
		t.Error("carol was added")
	}
}

// user expiry changes only the expiry; user rotate does not keep it.
func TestUserExpiryKeepsToken(t *testing.T) {
	ctx := context.Background()
	db := filepath.Join(t.TempDir(), "lounge.db")
	_, stdout, _ := user("add", "alice", "--expires-in", "1h", "--db", db)
	token := strings.TrimSpace(stdout)

	// --in: one line on stdout, the same token.
	before := time.Now()
	code, stdout, stderr := user("expiry", "alice", "--in", "1d12h", "--db", db)
	after := time.Now()
	m := regexp.MustCompile(`^the token of user alice expires at (\S+)\n$`).FindStringSubmatch(stdout)
	if code != 0 || m == nil || stderr != "" {
		t.Fatalf("expiry --in: %d %q %q", code, stdout, stderr)
	}
	within(t, "expiry --in 1d12h", m[1], before, after, 36*time.Hour)
	if cols := listLine(t, db, "alice"); cols[4] != m[1] {
		t.Errorf("list after expiry --in: %q", cols)
	}
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.UserByTokenHash(ctx, auth.Hash(token)); err != nil {
		t.Errorf("token after expiry --in: %v", err)
	}
	st.Close()

	// --never.
	code, stdout, _ = user("expiry", "alice", "--never", "--db", db)
	if code != 0 || stdout != "the token of user alice does not expire\n" {
		t.Errorf("expiry --never: %d %q", code, stdout)
	}
	if cols := listLine(t, db, "alice"); cols[4] != "-" {
		t.Errorf("list after expiry --never: %q", cols)
	}

	// Unknown users and local fail like the other commands.
	for name, want := range map[string]string{"nobody": "no such user", store.LocalUser: "reserved", "Bad/Name": "invalid user name"} {
		code, stdout, stderr := user("expiry", name, "--never", "--db", db)
		if code != 1 || stdout != "" || !strings.Contains(stderr, want) {
			t.Errorf("expiry %s: %d %q %q", name, code, stdout, stderr)
		}
	}
	if code, _, stderr := user("expiry", "alice", "--never", "--db", filepath.Join(t.TempDir(), "none.db")); code != 1 || !strings.Contains(stderr, "does not exist") {
		t.Errorf("expiry without a database: %d %q", code, stderr)
	}

	// rotate with an expiry, then without: no expiry.
	code, stdout, stderr = user("rotate", "alice", "--expires-in", "2h", "--db", db)
	if code != 0 || !tokenLine.MatchString(stdout) || expiresLine.FindStringSubmatch(stderr) == nil {
		t.Errorf("rotate --expires-in: %d %q %q", code, stdout, stderr)
	}
	if cols := listLine(t, db, "alice"); cols[4] == "-" {
		t.Errorf("list after rotate --expires-in: %q", cols)
	}
	code, stdout, stderr = user("rotate", "alice", "--db", db)
	if code != 0 || !tokenLine.MatchString(stdout) || !strings.Contains(stderr, "The token of user alice does not expire.\n") {
		t.Errorf("rotate: %d %q %q", code, stdout, stderr)
	}
	if cols := listLine(t, db, "alice"); cols[4] != "-" {
		t.Errorf("list after rotate without --expires-in: %q", cols)
	}

	// The usage names the command.
	if _, stdout, _ := user("help"); !strings.Contains(stdout, "user expiry <name> (--in DURATION | --never)") || !strings.Contains(stdout, "--expires-in DURATION") {
		t.Errorf("usage: %q", stdout)
	}
}
