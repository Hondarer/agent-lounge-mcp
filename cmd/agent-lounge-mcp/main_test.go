package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/server"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
	"github.com/hondarer/agent-lounge-mcp/internal/tools"
)

func parse(args ...string) (config.Config, error) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return parseServeFlags(fs, args)
}

func TestParseServeFlags(t *testing.T) {
	cfg, err := parse()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, config.Default()) {
		t.Errorf("no options: got %+v, want the defaults", cfg)
	}

	cfg, err = parse("--listen", "127.0.0.1:9000", "--db", "/tmp/x.db", "--inbox-wait-max", "2m")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:9000" || cfg.DBPath != "/tmp/x.db" || cfg.InboxWaitMax != 2*time.Minute {
		t.Errorf("got %+v", cfg)
	}
	cfg, err = parse("--memory-half-life", "3d", "--memory-fade-threshold", "0.25", "--memory-grace", "0", "--max-memory-bytes", "100", "--max-memories-per-scope", "5")
	if err != nil || cfg.MemoryHalfLife != 3*config.Day || cfg.MemoryFadeThreshold != 0.25 || cfg.MemoryGrace != 0 ||
		cfg.MaxMemoryBytes != 100 || cfg.MaxMemoriesPerScope != 5 {
		t.Errorf("memory options: %+v %v", cfg, err)
	}
	if cfg, err := parse("--log-connections"); err != nil || !cfg.LogConnections {
		t.Errorf("--log-connections: %+v %v", cfg, err)
	}
	for _, bad := range [][]string{{"--memory-fade-threshold", "0"}, {"--memory-fade-threshold", "1"}, {"--max-memory-bytes", "0"},
		{"--max-memories-per-scope", "0"}, {"--memory-half-life", "-1d"}} {
		if _, err := parse(bad...); err == nil {
			t.Errorf("%v: no error", bad)
		}
	}
	if cfg, err = parse("--inbox-wait-max", "0"); err != nil || cfg.InboxWaitMax != 0 {
		t.Errorf("--inbox-wait-max 0: %+v %v", cfg, err)
	}

	cfg, err = parse("--listen", "0.0.0.0:7788", "--allow-remote", "--require-token", "--allowed-host", "Lounge.Example.com", "--allowed-host", "b.example")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowRemote || !cfg.RequireToken || !reflect.DeepEqual(cfg.AllowedHosts, []string{"lounge.example.com", "b.example"}) {
		t.Errorf("got %+v", cfg)
	}
	if _, err := parse("--listen", "0.0.0.0:7788"); !errors.Is(err, config.ErrRemoteNotAllowed) {
		t.Errorf("non-loopback without --allow-remote: %v", err)
	}

	cfg, err = parse("--expire-after", "1d12h", "--retain-for", "0", "--session-idle", "90m", "--cleanup-interval", "1s",
		"--max-body-bytes", "1048576", "--max-unread", "1", "--max-sessions-per-user", "2")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ExpireAfter != 36*time.Hour || cfg.RetainFor != 0 || cfg.SessionIdle != 90*time.Minute || cfg.CleanupInterval != time.Second ||
		cfg.MaxBodyBytes != 1048576 || cfg.MaxUnread != 1 || cfg.MaxSessionsPerUser != 2 {
		t.Errorf("cleanup and limit options: %+v", cfg)
	}
	if cfg, err = parse("--expire-after", "14d"); err != nil || cfg.ExpireAfter != 14*24*time.Hour {
		t.Errorf("--expire-after 14d: %+v %v", cfg, err)
	}
	if cfg, err = parse(); err != nil || cfg.ConnectionIdle != time.Hour {
		t.Errorf("--connection-idle default: %v %v", cfg.ConnectionIdle, err)
	}
	for v, want := range map[string]time.Duration{"0": 0, "30s": 30 * time.Second, "1d": 24 * time.Hour} {
		if cfg, err = parse("--connection-idle", v); err != nil || cfg.ConnectionIdle != want {
			t.Errorf("--connection-idle %s: %v %v", v, cfg.ConnectionIdle, err)
		}
	}

	for _, args := range [][]string{{"--listen", "0.0.0.0:7788"}, {"--db", ""}, {"--allowed-host", "x:1"}, {"extra"}, {"--unknown"},
		{"--inbox-wait-max", "11m"}, {"--inbox-wait-max", "-1s"}, {"--inbox-wait-max", "60"},
		{"--connection-idle", "-1s"}, {"--connection-idle", "x"}, {"--expire-after", "-1d"}, {"--retain-for", "1w"}, {"--session-idle", "x"}, {"--cleanup-interval", "999ms"}, {"--cleanup-interval", "0"},
		{"--max-body-bytes", "0"}, {"--max-body-bytes", "1048577"}, {"--max-unread", "0"}, {"--max-sessions-per-user", "0"}, {"--max-unread", "x"}} {
		if _, err := parse(args...); err == nil {
			t.Errorf("%v was accepted", args)
		}
	}
}

func TestServeRetentionOptions(t *testing.T) {
	common := []string{"--memory-half-life", "5d", "--memory-fade-threshold", "0.3", "--memory-grace", "8d"}
	cfg, err := parse(common...)
	if err != nil {
		t.Fatal(err)
	}
	fallback := store.RetentionPolicy{HalfLife: 5 * config.Day, Threshold: 0.3, Grace: 8 * config.Day}
	if p, uniform := cfg.RetentionPolicies().Uniform(); !uniform || p != fallback {
		t.Fatalf("common options = %+v, uniform %t, want %+v", p, uniform, fallback)
	}

	cfg, err = parse(append(common, "--memory-user-half-life", "0",
		"--memory-repository-fade-threshold", "0.4", "--memory-branch-grace", "0",
		"--memory-shared-half-life", "2d", "--memory-shared-fade-threshold", "0.2", "--memory-shared-grace", "3d")...)
	if err != nil {
		t.Fatal(err)
	}
	for kind, want := range map[string]store.RetentionPolicy{
		store.ScopeUser:       {Threshold: 0.3, Grace: 8 * config.Day},
		store.ScopeRepository: {HalfLife: 5 * config.Day, Threshold: 0.4, Grace: 8 * config.Day},
		store.ScopeBranch:     {HalfLife: 5 * config.Day, Threshold: 0.3},
		store.ScopeShared:     {HalfLife: 2 * config.Day, Threshold: 0.2, Grace: 3 * config.Day},
	} {
		if p := cfg.RetentionPolicies().For(kind); p != want {
			t.Errorf("%s = %+v, want %+v", kind, p, want)
		}
	}
	for _, kind := range store.ScopeKinds() {
		for item, value := range map[string]string{"half-life": "-1d", "grace": "bad", "fade-threshold": "NaN"} {
			option := "--memory-" + kind + "-" + item
			if _, err := parse(option, value); err == nil || !strings.Contains(err.Error(), option[1:]) {
				t.Errorf("%s %s: error = %v", option, value, err)
			}
		}
	}
}

// Starts the server as the serve subcommand does, calls a tool over HTTP and stops it.
func TestServe(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := config.Default()
	cfg.Listen = "127.0.0.1:0"
	cfg.DBPath = filepath.Join(t.TempDir(), "lounge.db")
	cfg.ShutdownTimeout = 100 * time.Millisecond

	zero := time.Duration(0)
	cfg.MemoryRetentionByKind = map[string]config.RetentionOverride{store.ScopeBranch: {HalfLife: &zero}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, nil, cfg, ready) }()

	var addr net.Addr
	select {
	case addr = <-ready:
	case err := <-done:
		t.Fatalf("serve stopped early: %v", err)
	}

	st, err := store.Open(ctx, cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	policies, err := st.RecordedRetention(ctx)
	st.Close()
	if err != nil || len(policies.ByKind) != 4 || policies.For(store.ScopeBranch).HalfLife != 0 {
		t.Fatalf("serve recorded %+v, %v", policies, err)
	}

	cs := testutil.ConnectHTTP(t, "http://"+addr.String()+server.MCPPath)
	out := testutil.Call[tools.InfoOutput](t, cs, "lounge_info", nil)
	if out.SchemaVersion != 9 || len(out.MemoryRetention) != 4 || out.MemoryRetention[store.ScopeBranch].HalfLife != "0s" {
		t.Fatalf("got %+v", out)
	}

	// The client keeps an event stream open; the server still stops within the timeout, and
	// announces the shutdown before it waits.
	cancel()
	testutil.WaitFor(t, func() bool { return strings.Contains(logged.String(), "shutting down (waiting up to 100ms") },
		"the shutdown is announced")
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop")
	}
}

// Stops the server while a client waits in chat_inbox: the waiting call returns the shutdown
// error, and the server stops within the timeout.
func TestServeStopsWhileWaiting(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := config.Default()
	cfg.Listen = "127.0.0.1:0"
	cfg.DBPath = filepath.Join(t.TempDir(), "lounge.db")
	cfg.ShutdownTimeout = 500 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, nil, cfg, ready) }()
	var addr net.Addr
	select {
	case addr = <-ready:
	case err := <-done:
		t.Fatalf("serve stopped early: %v", err)
	}

	cs := testutil.ConnectHTTP(t, "http://"+addr.String()+server.MCPPath)
	testutil.Call[struct{}](t, cs, "lounge_join", map[string]any{"name": "waiter"})

	waited := make(chan *mcp.CallToolResult, 1)
	go func() {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "chat_inbox", Arguments: map[string]any{"wait_seconds": 30}})
		if err != nil {
			res = &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}
		}
		waited <- res
	}()
	// Let the call start waiting. If it has not yet, it fails with the same error on arrival.
	time.Sleep(200 * time.Millisecond)

	cancel()
	select {
	case res := <-waited:
		if !res.IsError || testutil.Text(res) != tools.ShutdownMessage {
			t.Errorf("waiting chat_inbox returned %q", testutil.Text(res))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting chat_inbox did not return")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop")
	}
	if out := logged.String(); !strings.Contains(out, "shutting down") || !strings.Contains(out, "stopped") {
		t.Errorf("log output:\n%s", out)
	}
}

// startServe runs serve in the background with cfg on a free loopback port. It returns the
// address, the function that stops the server (as a signal does), and the result of serve.
func startServe(t *testing.T, cfg config.Config, force <-chan struct{}) (net.Addr, context.CancelFunc, <-chan error) {
	t.Helper()
	cfg.Listen = "127.0.0.1:0"
	cfg.DBPath = filepath.Join(t.TempDir(), "lounge.db")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	ready := make(chan net.Addr, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, force, cfg, ready) }()
	select {
	case addr := <-ready:
		return addr, cancel, done
	case err := <-done:
		t.Fatalf("serve stopped early: %v", err)
		return nil, nil, nil
	}
}

// healthState returns the state reported by /healthz, or "" when it does not answer.
func healthState(addr net.Addr) string {
	res, err := http.Get("http://" + addr.String() + server.HealthPath)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	var body struct {
		State string `json:"state"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(res.Body).Decode(&body) != nil {
		return ""
	}
	return body.State
}

// While an agent stays connected, the server keeps answering after the stop begins: a new tool
// call gets the shutdown error and /healthz reports "stopping". It stops within the timeout.
func TestServeGracePeriod(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	cfg := config.Default()
	cfg.ShutdownTimeout = 500 * time.Millisecond
	addr, stop, done := startServe(t, cfg, nil)

	cs := testutil.ConnectHTTP(t, "http://"+addr.String()+server.MCPPath)
	testutil.Call[struct{}](t, cs, "lounge_join", map[string]any{"name": "agent"})
	if got := healthState(addr); got != "ok" {
		t.Fatalf("health before the stop = %q", got)
	}

	start := time.Now()
	stop()
	testutil.WaitFor(t, func() bool { return healthState(addr) == "stopping" }, "/healthz does not report stopping")
	if msg := testutil.CallErr(t, cs, "lounge_who", nil); msg != tools.ShutdownMessage {
		t.Errorf("tool call while stopping: %q", msg)
	}
	if elapsed := time.Since(start); elapsed >= cfg.ShutdownTimeout {
		t.Fatalf("the checks took %s, longer than the grace period", elapsed)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop")
	}
	if elapsed := time.Since(start); elapsed > cfg.ShutdownTimeout+time.Second {
		t.Errorf("stopping took %s, want about %s", elapsed, cfg.ShutdownTimeout)
	}
}

// Without connected agents, the server stops at once instead of waiting for the timeout.
func TestServeStopsPromptlyWithoutClients(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	cfg := config.Default()
	cfg.ShutdownTimeout = 5 * time.Second
	addr, stop, done := startServe(t, cfg, nil)
	if got := healthState(addr); got != "ok" {
		t.Fatalf("health = %q", got)
	}

	start := time.Now()
	stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("stopping without clients took %s", elapsed)
	}
}

// A second signal during the stop closes the connections at once instead of waiting for the
// connected agent, and serve still ends without an error (exit code 0).
func TestServeSecondSignalStopsNow(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	cfg := config.Default()
	cfg.ShutdownTimeout = 30 * time.Second
	force := make(chan struct{})
	addr, stop, done := startServe(t, cfg, force)

	cs := testutil.ConnectHTTP(t, "http://"+addr.String()+server.MCPPath)
	testutil.Call[struct{}](t, cs, "lounge_join", map[string]any{"name": "agent"})

	stop()
	testutil.WaitFor(t, func() bool {
		return strings.Contains(logged.String(),
			"shutting down (waiting up to 30s for open connections; press Ctrl+C again to stop now)")
	}, "the shutdown is not announced with the hint for a second Ctrl+C")
	if healthState(addr) != "stopping" {
		t.Fatal("the server is not in its grace period")
	}
	start := time.Now()
	close(force)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop after the second signal")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("forced stop took %s", elapsed)
	}
	if out := logged.String(); !strings.Contains(out, "stopped (forced)") {
		t.Errorf("log output:\n%s", out)
	}
	if healthState(addr) != "" {
		t.Error("the listener is still open")
	}
}

// Without a second signal, the stop is reported as a normal one.
func TestServeStopWithoutSecondSignalIsNotForced(t *testing.T) {
	logged := &syncBuffer{}
	log.SetOutput(logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	force := make(chan struct{})
	_, stop, done := startServe(t, config.Default(), force)
	stop()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if out := logged.String(); strings.Contains(out, "forced") || !strings.Contains(out, "stopped") {
		t.Errorf("log output:\n%s", out)
	}
}

func TestWatchSignals(t *testing.T) {
	sigs := make(chan os.Signal)
	ctx, force := watchSignals(sigs)
	select {
	case <-ctx.Done():
		t.Fatal("stopped before a signal")
	case <-force:
		t.Fatal("forced before a signal")
	default:
	}
	sigs <- os.Interrupt
	<-ctx.Done()
	select {
	case <-force:
		t.Fatal("forced after one signal")
	case <-time.After(20 * time.Millisecond):
	}
	sigs <- os.Interrupt
	<-force
	// Further signals are received (they do not block the sender) and ignored.
	select {
	case sigs <- os.Interrupt:
	case <-time.After(5 * time.Second):
		t.Fatal("a third signal is not received")
	}
}

// syncBuffer collects log output written from the server's goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
