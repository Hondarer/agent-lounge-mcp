// agent-lounge-mcp is an MCP server that lets coding agents share conversations and short-lived
// memories. It serves MCP over Streamable HTTP and keeps all data in one SQLite file:
//
//	agent-lounge-mcp serve --listen 127.0.0.1:7788 --db lounge.db
//
// Diagnostics go to standard error.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/server"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/version"
)

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("agent-lounge-mcp: ")
	log.SetFlags(0)

	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		cfg, err := parseServeFlags(flag.NewFlagSet("agent-lounge-mcp serve", flag.ContinueOnError), os.Args[2:])
		switch {
		case errors.Is(err, flag.ErrHelp):
			return
		case err != nil:
			log.Fatal(err)
		}
		sigs := make(chan os.Signal, 2)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		ctx, force := watchSignals(sigs)
		if err := serve(ctx, force, cfg, nil); err != nil {
			log.Fatal(err)
		}
	case "version", "--version", "-version":
		fmt.Println("agent-lounge-mcp", version.Version)
	case "help", "--help", "-help", "-h":
		usage(os.Stdout)
	default:
		log.Printf("unknown command %q", os.Args[1])
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp serve [--listen ADDR] [--db FILE] [--inbox-wait-max DURATION]  (start the server; see agent-lounge-mcp serve -h)")
	fmt.Fprintln(w, "       agent-lounge-mcp version")
}

// parseServeFlags builds the server configuration from the command line of the serve subcommand.
func parseServeFlags(fs *flag.FlagSet, args []string) (config.Config, error) {
	cfg := config.Default()
	fs.StringVar(&cfg.Listen, "listen", cfg.Listen, "loopback TCP address to listen on")
	fs.StringVar(&cfg.DBPath, "db", cfg.DBPath, "SQLite database file (created when missing)")
	fs.DurationVar(&cfg.InboxWaitMax, "inbox-wait-max", cfg.InboxWaitMax,
		"longest time chat_inbox may wait for a message (0 disables waiting, at most "+config.MaxInboxWaitMax.String()+
			"); keep it below the agents' tool-call time limit")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return cfg, cfg.Validate()
}

// watchSignals turns the signals received on sigs into the stop requests of serve: the first
// signal cancels the returned context (stop gracefully), the second closes force (stop now).
// Later signals are received and ignored, so they do not kill the process.
func watchSignals(sigs <-chan os.Signal) (context.Context, <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	force := make(chan struct{})
	go func() {
		<-sigs
		cancel()
		<-sigs
		close(force)
		for range sigs {
		}
	}()
	return ctx, force
}

// errForced is the cause of the shutdown context when force is closed.
var errForced = errors.New("stop forced")

// serve runs the HTTP server until ctx is cancelled, then stops it gracefully within
// cfg.ShutdownTimeout; when force is closed during that stop, it closes the listener and every
// connection at once instead of waiting. force may be nil. When ready is not nil, it receives
// the address the server listens on (useful with port 0).
func serve(ctx context.Context, force <-chan struct{}, cfg config.Config, ready chan<- net.Addr) error {
	st, err := store.Open(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()
	// Requests are not authenticated: every connection acts as the one local user.
	userID, err := st.EnsureLocalUser(ctx)
	if err != nil {
		return err
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.Listen)
	if err != nil {
		return err
	}
	sess := server.Session{Store: st, UserID: userID}
	srv := server.New(cfg, sess)
	hs := &http.Server{Handler: server.Handler(srv, sess)}

	log.Printf("version %s, database %s", version.Version, cfg.DBPath)
	log.Printf("serving MCP at http://%s%s", ln.Addr(), server.MCPPath)
	if ready != nil {
		ready <- ln.Addr()
	}

	served := make(chan error, 1)
	go func() { served <- hs.Serve(ln) }()
	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}

	// Agents keep an event stream open, so stopping usually takes the whole timeout: say so at
	// once, or the operator cannot tell whether the signal was received.
	log.Printf("shutting down (waiting up to %s for open connections; press Ctrl+C again to stop now)", cfg.ShutdownTimeout)

	// Tell agents first: tool calls fail with the shutdown error from now on, waiting chat_inbox
	// calls return it, and clients that set a log level receive it as a log notification. Keep
	// accepting requests while agents remain connected, so that their calls get that error and
	// /healthz reports "stopping" instead of the connection being refused. Then close the
	// listener, let in-flight requests finish, and drop what remains (such as open event
	// streams). All of it shares one deadline of ShutdownTimeout.
	// A second signal (force) cancels the shared deadline at once, with errForced as its cause.
	forceCtx, forceStop := context.WithCancelCause(context.Background())
	defer forceStop(nil)
	shutdownCtx, cancel := context.WithTimeout(forceCtx, cfg.ShutdownTimeout)
	defer cancel()
	go func() {
		select {
		case <-force:
			forceStop(errForced)
		case <-shutdownCtx.Done():
		}
	}()
	srv.BeginShutdown(shutdownCtx)
	srv.WaitForSessions(shutdownCtx)
	if err := hs.Shutdown(shutdownCtx); err != nil {
		hs.Close()
	}
	if errors.Is(context.Cause(shutdownCtx), errForced) {
		log.Print("stopped (forced)")
		return nil
	}
	log.Print("stopped")
	return nil
}
