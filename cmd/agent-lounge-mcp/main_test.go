package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/server"
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
	if cfg != config.Default() {
		t.Errorf("no options: got %+v, want the defaults", cfg)
	}

	cfg, err = parse("--listen", "127.0.0.1:9000", "--db", "/tmp/x.db", "--inbox-wait-max", "2m")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:9000" || cfg.DBPath != "/tmp/x.db" || cfg.InboxWaitMax != 2*time.Minute {
		t.Errorf("got %+v", cfg)
	}
	if cfg, err = parse("--inbox-wait-max", "0"); err != nil || cfg.InboxWaitMax != 0 {
		t.Errorf("--inbox-wait-max 0: %+v %v", cfg, err)
	}

	for _, args := range [][]string{{"--listen", "0.0.0.0:7788"}, {"--db", ""}, {"extra"}, {"--unknown"},
		{"--inbox-wait-max", "11m"}, {"--inbox-wait-max", "-1s"}, {"--inbox-wait-max", "60"}} {
		if _, err := parse(args...); err == nil {
			t.Errorf("%v was accepted", args)
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

	cs := testutil.ConnectHTTP(t, "http://"+addr.String()+server.MCPPath)
	out := testutil.Call[struct {
		SchemaVersion int `json:"schema_version"`
	}](t, cs, "lounge_info", nil)
	if out.SchemaVersion < 1 {
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
// error, the client receives the log notification, and the server stops within the timeout.
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

	logs := make(chan *mcp.LoggingMessageParams, 1)
	cs := testutil.ConnectHTTPWith(t, "http://"+addr.String()+server.MCPPath, &mcp.ClientOptions{
		LoggingMessageHandler: func(_ context.Context, req *mcp.LoggingMessageRequest) { logs <- req.Params },
	})
	if err := cs.SetLoggingLevel(context.Background(), &mcp.SetLoggingLevelParams{Level: "warning"}); err != nil {
		t.Fatal(err)
	}
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
	case p := <-logs:
		if p.Level != "warning" || p.Data != tools.ShutdownMessage {
			t.Errorf("log notification = %+v", p)
		}
	case <-time.After(5 * time.Second):
		t.Error("no log notification")
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
