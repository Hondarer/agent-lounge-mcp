package main

import (
	"bytes"
	"context"
	"flag"
	"io"
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
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
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

	cfg, err = parse("--listen", "127.0.0.1:9000", "--db", "/tmp/x.db")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:9000" || cfg.DBPath != "/tmp/x.db" {
		t.Errorf("got %+v", cfg)
	}

	for _, args := range [][]string{{"--listen", "0.0.0.0:7788"}, {"--db", ""}, {"extra"}, {"--unknown"}} {
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
	go func() { done <- serve(ctx, cfg, ready) }()

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
