package loungecli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/auth"
	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/connection"
	"github.com/hondarer/agent-lounge-mcp/internal/loungecli"
	"github.com/hondarer/agent-lounge-mcp/internal/server"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

const token = "alm_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

func setup(t *testing.T) server.Session {
	t.Helper()
	dir, err := os.MkdirTemp("", "al-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Setenv(connection.EnvDir, dir)
	t.Setenv(connection.EnvName, "")
	t.Setenv("AGENT_LOUNGE_AS", "")
	t.Setenv("WSL_DISTRO_NAME", "")
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "lounge.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	local, err := st.EnsureLocalUser(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddUser(t.Context(), "tester", auth.Hash(token), time.Time{}); err != nil {
		t.Fatal(err)
	}
	return server.Session{Store: st, LocalUser: local}
}

func daemon(t *testing.T, endpoint string, idle time.Duration) <-chan error {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- connection.Serve(ctx, connection.Options{Name: "default", URL: endpoint, Idle: idle}, func(*connection.State) { close(ready) })
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("daemon startup: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("daemon startup timed out")
	}
	return done
}

type result struct {
	code     int
	out, err string
}

func run(ctx context.Context, args ...string) result {
	var out, stderr bytes.Buffer
	code := loungecli.Run(ctx, args[0], args[1:], strings.NewReader(""), &out, &stderr)
	return result{code, out.String(), stderr.String()}
}

func check(t *testing.T, r result, code int, fragment string) {
	t.Helper()
	if r.code != code || !strings.Contains(r.out+r.err, fragment) {
		t.Fatalf("want code %d and %q, got %+v", code, fragment, r)
	}
}

func waiting(t *testing.T, base string, n int) bool {
	t.Helper()
	resp, err := http.Get(base + server.HealthPath)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var status struct{ Waiting int }
	return json.NewDecoder(resp.Body).Decode(&status) == nil && status.Waiting == n
}

func TestShellCalls(t *testing.T) {
	sess := setup(t)
	cfg := config.Default()
	up := httptest.NewServer(server.Handler(server.New(cfg, sess), sess))
	t.Cleanup(up.Close)
	daemon(t, up.URL+server.MCPPath, 0)
	ctx := t.Context()

	check(t, run(ctx, "connection", "status"), 0, `"participants":{}`)
	check(t, run(ctx, "call", "lounge_join"), 2, "AGENT_LOUNGE_AS")
	check(t, run(ctx, "tools"), 0, "Run one agent-lounge-mcp command per shell call")
	check(t, run(ctx, "tools", "--schema", "chat_inbox"), 0, "inputSchema")
	check(t, run(ctx, "call", "lounge_help"), 0, "Profile")
	check(t, run(ctx, "call", "--as", "alice", "unknown"), 2, "unknown tool")
	check(t, run(ctx, "call", "--as", "alice", "chat_inbox", "wait_seconds=no"), 2, "usable keys")
	check(t, run(ctx, "call", "--as", "Ａlice", "--no-profile", "lounge_join"), 0, `"name":"Alice"`)
	t.Setenv("AGENT_LOUNGE_AS", "bob")
	check(t, run(ctx, "call", "--no-profile", "lounge_join", "name=Robert"), 0, `"name":"Robert"`)
	check(t, run(ctx, "connection", "status"), 0, `"bob":{"connected":true,"last_join":"Robert"}`)
	check(t, run(ctx, "connection", "list"), 0, up.URL)
	// Options are accepted after the tool name too, and the output is the structured result
	// of the tool alone (one line), or the text of a tool that returns only text.
	after := run(ctx, "call", "lounge_who", "--as", "Alice")
	check(t, after, 0, `{"`)
	if strings.Contains(after.out, `"structuredContent"`) || strings.Contains(after.out, `"content"`) || strings.Count(after.out, "\n") != 1 {
		t.Errorf("call printed %q, want the structured result on one line", after.out)
	}
	if help := run(ctx, "call", "lounge_help"); !strings.HasPrefix(help.out, "# ") {
		t.Errorf("lounge_help printed %.40q, want the text of the guide", help.out)
	}
	check(t, run(ctx, "connection", "start", "--url", up.URL+server.MCPPath), 1, "already running")

	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan result, 1)
	go func() { done <- run(waitCtx, "call", "--as", "Alice", "chat_inbox", "wait_seconds=15") }()
	testutil.WaitFor(t, func() bool { return waiting(t, up.URL, 1) }, "inbox never started")
	check(t, run(ctx, "call", "--as", "Alice", "chat_send", "to=Robert", "body=concurrent"), 0, "message_id")
	check(t, run(ctx, "call", "chat_inbox"), 0, "concurrent")
	cancel()
	check(t, <-done, 130, "")
	testutil.WaitFor(t, func() bool { return waiting(t, up.URL, 0) }, "inbox was not canceled")
	check(t, run(ctx, "call", "--as", "Alice", "lounge_who"), 0, "Alice")

	other := testutil.ConnectHTTP(t, up.URL+server.MCPPath)
	testutil.Call[map[string]any](t, other, "lounge_join", map[string]any{"name": "Alice"})
	check(t, run(ctx, "call", "--as", "Alice", "chat_inbox"), 1, "took the session over")
	if _, err := sess.Store.RemoveSession(ctx, store.LocalUser, "Robert"); err != nil {
		t.Fatal(err)
	}
	check(t, run(ctx, "call", "chat_inbox"), 1, "was deleted")
	check(t, run(ctx, "connection", "stop"), 0, "")
}

func TestReconnectAndUnknownOutcome(t *testing.T) {
	sess := setup(t)
	cfg := config.Default()
	cfg.ConnectionIdle = 500 * time.Millisecond
	first := server.New(cfg, sess)
	var handler atomic.Pointer[http.Handler]
	current := server.Handler(first, sess)
	handler.Store(&current)
	var drop atomic.Bool
	var deliveries atomic.Int32
	var breakSSE atomic.Bool
	var joins atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			b, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(b))
			if bytes.Contains(b, []byte(`"name":"lounge_join"`)) && r.Header.Get("Mcp-Session-Id") != "" {
				joins.Add(1)
			}
			if bytes.Contains(b, []byte(`"name":"chat_send"`)) && drop.Load() {
				rec := httptest.NewRecorder()
				(*handler.Load()).ServeHTTP(rec, r)
				deliveries.Add(1)
				conn, rw, err := w.(http.Hijacker).Hijack()
				if err == nil {
					kind, partial := "application/json", "{"
					if breakSSE.Load() {
						kind, partial = "text/event-stream", "event: message\ndata: {"
					}
					_, _ = rw.WriteString("HTTP/1.1 200 OK\r\nContent-Type: " + kind + "\r\nContent-Length: 10000\r\n\r\n" + partial)
					_ = rw.Flush()
					_ = conn.Close()
				}
				return
			}
		}
		(*handler.Load()).ServeHTTP(w, r)
	}))
	t.Cleanup(up.Close)
	daemon(t, up.URL+server.MCPPath, 0)
	ctx := t.Context()
	alice := []string{"call", "--as", "alice"}
	call := func(args ...string) result { return run(ctx, append(append([]string{}, alice...), args...)...) }
	check(t, call("--no-profile", "lounge_join", "name=Alice", "agent=prior-profile"), 0, "prior-profile")
	testutil.WaitFor(t, func() bool {
		for range first.MCP.Sessions() {
			return false
		}
		return true
	}, "upstream idle sessions did not close")
	done := make(chan result, 4)
	for range 4 {
		go func() { done <- call("lounge_who") }()
	}
	for range 4 {
		check(t, <-done, 0, "prior-profile")
	}
	if n := joins.Load(); n != 2 {
		t.Fatalf("join attempts after concurrent reconnect = %d, want 2", n)
	}
	cfg.ConnectionIdle = 0
	restarted := server.Handler(server.New(cfg, sess), sess)
	handler.Store(&restarted)
	check(t, call("lounge_who"), 0, "prior-profile")
	if n := joins.Load(); n != 3 {
		t.Fatalf("join attempts after restart = %d, want 3", n)
	}
	check(t, run(ctx, "call", "--as", "bob", "--no-profile", "lounge_join"), 0, "bob")
	drop.Store(true)
	check(t, call("chat_send", "to=bob", "body=once"), 3, "outcome is unknown")
	if n := deliveries.Load(); n != 1 {
		t.Fatalf("send attempts = %d, want 1", n)
	}
	check(t, run(ctx, "call", "--as", "bob", "chat_inbox"), 0, "once")
	breakSSE.Store(true)
	check(t, call("chat_send", "to=bob", "body=broken-sse"), 3, "outcome is unknown")
	if n := deliveries.Load(); n != 2 {
		t.Fatalf("send attempts after SSE loss = %d, want 2", n)
	}
	check(t, run(ctx, "call", "--as", "bob", "chat_inbox"), 0, "broken-sse")
}

func TestDaemonIdleAndStaleSocket(t *testing.T) {
	sess := setup(t)
	path := filepath.Join(os.Getenv(connection.EnvDir), "default.sock")
	stale, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	stale.SetUnlinkOnClose(false)
	stale.Close()
	cfg := config.Default()
	up := httptest.NewServer(server.Handler(server.New(cfg, sess), sess))
	t.Cleanup(up.Close)
	done := daemon(t, up.URL+server.MCPPath, 700*time.Millisecond)
	ctx := t.Context()
	check(t, run(ctx, "call", "--as", "idle", "--no-profile", "lounge_join"), 0, "idle")
	result := run(ctx, "call", "--as", "idle", "chat_inbox", "wait_seconds=1")
	check(t, result, 0, "messages")
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("idle daemon did not stop")
	}
	check(t, run(ctx, "call", "lounge_help"), 3, "cannot connect to daemon")
}

func TestDetachedBinary(t *testing.T) {
	sess := setup(t)
	cfg := config.Default()
	cfg.RequireToken = true
	h := server.Handler(server.New(cfg, sess), sess)
	var invalid atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if invalid.Load() && r.Method == "POST" {
			b, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(b))
			if bytes.Contains(b, []byte(`"name":"lounge_who"`)) {
				http.Error(w, token, http.StatusUnauthorized)
				return
			}
		}
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(up.Close)

	bin := filepath.Join(t.TempDir(), "agent-lounge-mcp")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "../../cmd/agent-lounge-mcp")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	command := func(input string, args ...string) result {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Stdin = strings.NewReader(input)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		b, err := cmd.Output()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		r := result{code, string(b), stderr.String()}
		if strings.Contains(r.out+r.err, token) {
			t.Fatal("CLI leaked credential")
		}
		return r
	}
	endpoint := up.URL + server.MCPPath
	check(t, command("wrong\n", "connection", "start", "--detach", "--url", endpoint, "--token-stdin"), 1, "token is invalid")
	check(t, command(token+"\nignored\n", "connection", "start", "--detach", "--url", endpoint, "--token-stdin"), 0, endpoint)
	defer command("", "connection", "stop")
	check(t, command(`{"name":123}`, "call", "--as", "one", "--json", "-", "--no-profile", "lounge_join"), 2, "usable keys")
	check(t, command(`{"agent":"ignored"}`, "call", "--as", "one", "--json", "-", "--no-profile", "lounge_join", "agent=@@literal"), 0, `"agent":"@literal"`)
	check(t, command("", "call", "--as", "one", "lounge_who"), 0, `"name":"one"`)
	check(t, command("", "call", "--as", "one", "lounge_who"), 0, `"you":true`)
	check(t, command("", "connection", "status"), 0, `"last_join":"one"`)
	killed := exec.CommandContext(t.Context(), bin, "call", "--as", "one", "chat_inbox", "wait_seconds=20")
	if err := killed.Start(); err != nil {
		t.Fatal(err)
	}
	defer killed.Process.Kill()
	testutil.WaitFor(t, func() bool { return waiting(t, up.URL, 1) }, "binary inbox never started")
	if err := killed.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = killed.Wait()
	testutil.WaitFor(t, func() bool { return waiting(t, up.URL, 0) }, "killed call was not canceled")
	check(t, command("", "call", "--as", "one", "lounge_who"), 0, `"you":true`)
	for _, suffix := range []string{".json", ".log"} {
		b, err := os.ReadFile(filepath.Join(os.Getenv(connection.EnvDir), "default"+suffix))
		if err != nil || bytes.Contains(b, []byte(token)) {
			t.Fatalf("credential file check for %s: %v", suffix, err)
		}
	}
	invalid.Store(true)
	check(t, command("", "call", "--as", "one", "lounge_who"), 3, "restart the daemon with a new token")
	invalid.Store(false)
	check(t, command("", "call", "--as", "one", "lounge_who"), 3, "restart the daemon with a new token")
	check(t, command("", "connection", "stop"), 0, "")
}

func TestJoinProfileFromCallDirectory(t *testing.T) {
	sess := setup(t)
	cfg := config.Default()
	h := server.Handler(server.New(cfg, sess), sess)
	joins := make(chan map[string]any, 3)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			b, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(b))
			var message struct {
				Params struct {
					Name      string
					Arguments map[string]any
				}
			}
			if json.Unmarshal(b, &message) == nil && message.Params.Name == "lounge_join" {
				joins <- message.Params.Arguments
			}
		}
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(up.Close)
	daemon(t, up.URL+server.MCPPath, 0)
	root := t.TempDir()
	t.Chdir(root)
	for _, args := range [][]string{
		{"init", "-b", "test-branch"},
		{"remote", "add", "origin", "https://user:secret@example.com/Team/Repo.git"},
	} {
		if b, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, b)
		}
	}
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	ctx := t.Context()
	check(t, run(ctx, "call", "--as", "profile", "lounge_join", "machine=explicit-host"), 0, "example.com/team/repo")
	args := <-joins
	if args["machine"] != "explicit-host" || strings.TrimRight(args["origin"].(string), "\r\n") != "https://example.com/Team/Repo.git" ||
		strings.TrimRight(args["branch"].(string), "\r\n") != "test-branch" || strings.Contains(strings.TrimSpace(args["folder"].(string)), "nested") {
		t.Fatalf("profile sent = %#v", args)
	}
	check(t, run(ctx, "call", "--as", "profile", "--no-profile", "lounge_join", "agent=explicit-agent"), 0, "explicit-agent")
	if args := <-joins; len(args) != 2 || args["name"] != "profile" || args["agent"] != "explicit-agent" {
		t.Fatalf("profile with --no-profile = %#v", args)
	}
}
