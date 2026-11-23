// agent-lounge-mcp is an MCP server that lets coding agents share conversations and short-lived
// memories. It serves MCP over Streamable HTTP and keeps all data in one SQLite file:
//
//	agent-lounge-mcp serve --listen 127.0.0.1:7788 --db lounge.db
//
// Users and their tokens are managed with the user subcommand (see user.go), sessions,
// messages and memories with the session, message and memory subcommands, and the export
// subcommand writes the messages and memories as Markdown files (see admin.go). Diagnostics
// go to standard error.
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
	"strings"
	"syscall"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/loungecli"
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
		case errors.Is(err, config.ErrRemoteNotAllowed):
			log.Print(err)
			os.Exit(2)
		case err != nil:
			log.Fatal(err)
		}
		sigs := make(chan os.Signal, 2)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		ctx, force := watchSignals(sigs)
		if err := serve(ctx, force, cfg, nil); err != nil {
			log.Fatal(err)
		}
	case "connection", "tools", "call":
		os.Exit(runLoungeCLI(os.Args[1], os.Args[2:]))
	case "user":
		os.Exit(runUser(context.Background(), os.Args[2:], os.Stdout, os.Stderr))
	case "session":
		os.Exit(runSession(context.Background(), os.Args[2:], os.Stdout, os.Stderr))
	case "memory":
		os.Exit(runMemory(context.Background(), os.Args[2:], os.Stdout, os.Stderr))
	case "message":
		os.Exit(runMessage(context.Background(), os.Args[2:], os.Stdout, os.Stderr))
	case "export":
		os.Exit(runExport(context.Background(), os.Args[2:], os.Stdout, os.Stderr))
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

func runLoungeCLI(command string, args []string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return loungecli.Run(ctx, command, args, os.Stdin, os.Stdout, os.Stderr)
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp serve [--listen ADDR] [--allow-remote] [--require-token] [--allowed-host NAME]... [--db FILE] [--inbox-wait-max DURATION]")
	fmt.Fprintln(w, "           [--expire-after DURATION] [--retain-for DURATION] [--session-idle DURATION] [--cleanup-interval DURATION]")
	fmt.Fprintln(w, "           [--max-body-bytes N] [--max-unread N] [--max-sessions-per-user N] [--max-rooms-per-scope N]")
	fmt.Fprintln(w, "           [--memory-half-life DURATION] [--memory-fade-threshold X] [--memory-grace DURATION]")
	for _, kind := range store.ScopeKinds() {
		fmt.Fprintf(w, "           [--memory-%[1]s-half-life DURATION] [--memory-%[1]s-fade-threshold X] [--memory-%[1]s-grace DURATION]\n", kind)
	}
	fmt.Fprintln(w, "           [--max-memory-bytes N] [--max-memories-per-scope N]")
	fmt.Fprintln(w, "           start the server; see agent-lounge-mcp serve -h")
	fmt.Fprintln(w, "       agent-lounge-mcp user add|list|rotate|remove ...  (manage users and their tokens; see agent-lounge-mcp user -h)")
	fmt.Fprintln(w, "       agent-lounge-mcp session list|remove ...          (list and remove sessions; see agent-lounge-mcp session -h)")
	fmt.Fprintln(w, "       agent-lounge-mcp memory list|remove ...           (list and remove memories; see agent-lounge-mcp memory -h)")
	fmt.Fprintln(w, "       agent-lounge-mcp message list|show ...          (list and show messages; see agent-lounge-mcp message -h)")
	fmt.Fprintln(w, "       agent-lounge-mcp export [filters] --out DIR [--format markdown] [--db FILE]  (write messages and memories as Markdown files)")
	fmt.Fprint(w, loungecli.Usage)
	fmt.Fprintln(w, "       agent-lounge-mcp version")
}

// parseServeFlags builds the server configuration from the command line of the serve subcommand.
func parseServeFlags(fs *flag.FlagSet, args []string) (config.Config, error) {
	cfg := config.Default()
	fs.StringVar(&cfg.Listen, "listen", cfg.Listen, "TCP address to listen on; an address other than loopback needs --allow-remote")
	fs.BoolVar(&cfg.AllowRemote, "allow-remote", false,
		"allow --listen on an address other than loopback, served over plain HTTP (every request then needs a token)")
	fs.BoolVar(&cfg.RequireToken, "require-token", false, "require a token on every request, also on loopback")
	fs.Func("allowed-host", "Host header name accepted besides the loopback names, for an HTTPS reverse proxy in front "+
		"(repeatable; every request then needs a token)", func(v string) error {
		cfg.AllowedHosts = append(cfg.AllowedHosts, config.NormalizeHost(v))
		return nil
	})
	fs.StringVar(&cfg.DBPath, "db", cfg.DBPath, "SQLite database file (created when missing)")
	fs.DurationVar(&cfg.InboxWaitMax, "inbox-wait-max", cfg.InboxWaitMax,
		"longest time chat_inbox may wait for a message (0 disables waiting, at most "+config.MaxInboxWaitMax.String()+
			"); keep it below the agents' tool-call time limit")
	fs.Var(config.DurationValue{D: &cfg.ConnectionIdle}, "connection-idle",
		"time without a request after which an MCP connection is closed (0: never); the agent joins its session again "+
			"on a new connection, and nothing is deleted (unlike --session-idle)")
	fs.Var(config.DurationValue{D: &cfg.ExpireAfter}, "expire-after",
		"time after which a message nobody read expires (0: never). Durations take h, m, s and d (24h), such as 14d or 1d12h")
	fs.Var(config.DurationValue{D: &cfg.RetainFor}, "retain-for",
		"time after its last change of state (read, acked or expired) after which a message with nothing unread is deleted (0: never)")
	fs.Var(config.DurationValue{D: &cfg.SessionIdle}, "session-idle",
		"time without use after which a session is deleted with the messages it sent (0: never)")
	fs.Var(config.DurationValue{D: &cfg.CleanupInterval}, "cleanup-interval",
		"interval of the cleanup (expiry and deletion), which also runs at start-up; at least "+config.MinCleanupInterval.String())
	fs.IntVar(&cfg.MaxBodyBytes, "max-body-bytes", cfg.MaxBodyBytes,
		fmt.Sprintf("largest message body in bytes of UTF-8 (1 to %d)", config.MaxMaxBodyBytes))
	fs.IntVar(&cfg.MaxUnread, "max-unread", cfg.MaxUnread, "most unread messages one session can hold (1 or more)")
	fs.IntVar(&cfg.MaxSessionsPerUser, "max-sessions-per-user", cfg.MaxSessionsPerUser, "most sessions one user can have (1 or more)")
	fs.IntVar(&cfg.MaxRoomsPerScope, "max-rooms-per-scope", cfg.MaxRoomsPerScope,
		"most rooms one scope can have besides lounge (1 or more); lowering it deletes no room")
	applyRetentionFlags := addMemoryRetentionFlags(fs, &cfg, true)
	fs.Var(config.DurationValue{D: &cfg.MemoryGrace}, "memory-grace",
		"time after a memory faded after which the cleanup deletes it (0: never)")
	fs.IntVar(&cfg.MaxMemoryBytes, "max-memory-bytes", cfg.MaxMemoryBytes,
		fmt.Sprintf("largest memory body in bytes of UTF-8 (1 to %d)", config.MaxMaxBodyBytes))
	fs.IntVar(&cfg.MaxMemoriesPerScope, "max-memories-per-scope", cfg.MaxMemoriesPerScope,
		"most memories one scope (one user, one repository) can hold, faded and pinned ones included (1 or more)")
	fs.BoolVar(&cfg.LogConnections, "log-connections", false,
		"write diagnostics of every MCP connection to standard error: its client and version, the methods and tools it calls "+
			"(names only)")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	applyRetentionFlags()
	return cfg, cfg.Validate()
}

// addMemoryRetentionFlags registers the options shared by serve and memory list. The
// returned function collects the supplied per-kind items after parsing, including zeros.
func addMemoryRetentionFlags(fs *flag.FlagSet, cfg *config.Config, withGrace bool) func() {
	fs.Var(config.DurationValue{D: &cfg.MemoryHalfLife}, "memory-half-life",
		"half-life of the retention score of a memory never referenced; references lengthen it (0: memories never fade)")
	fs.Float64Var(&cfg.MemoryFadeThreshold, "memory-fade-threshold", cfg.MemoryFadeThreshold,
		"retention score below which a memory has faded and leaves plain searches (above 0, below 1)")
	type kindFlags struct {
		halfLife, grace time.Duration
		threshold       float64
	}
	valuesByKind := make(map[string]*kindFlags)
	for _, kind := range store.ScopeKinds() {
		prefix := "memory-" + kind + "-"
		values := &kindFlags{}
		valuesByKind[kind] = values
		fs.Var(config.DurationValue{D: &values.halfLife}, prefix+"half-life", "override the half-life for "+kind+" (0: never fades)")
		fs.Lookup(prefix + "half-life").DefValue = ""
		fs.Float64Var(&values.threshold, prefix+"fade-threshold", 0, "override the fade threshold for "+kind+" (above 0, below 1)")
		if withGrace {
			fs.Var(config.DurationValue{D: &values.grace}, prefix+"grace", "override the deletion grace for "+kind+" (0: never deletes faded memories)")
			fs.Lookup(prefix + "grace").DefValue = ""
		}
	}
	return func() {
		fs.Visit(func(f *flag.Flag) {
			name, memoryFlag := strings.CutPrefix(f.Name, "memory-")
			kind, item, _ := strings.Cut(name, "-")
			values := valuesByKind[kind]
			if !memoryFlag || values == nil {
				return
			}
			o := cfg.MemoryRetentionByKind[kind]
			switch item {
			case "half-life":
				o.HalfLife = &values.halfLife
			case "fade-threshold":
				o.Threshold = &values.threshold
			case "grace":
				o.Grace = &values.grace
			}
			if cfg.MemoryRetentionByKind == nil {
				cfg.MemoryRetentionByKind = make(map[string]config.RetentionOverride)
			}
			cfg.MemoryRetentionByKind[kind] = o
		})
	}
}

// listenAddr returns the address to show for a listener: the host as configured (the listener
// reports 0.0.0.0 as [::]) with the port actually bound (useful with port 0).
func listenAddr(configured string, actual net.Addr) string {
	host, _, err := net.SplitHostPort(configured)
	_, port, err2 := net.SplitHostPort(actual.String())
	if err != nil || err2 != nil {
		return actual.String()
	}
	return net.JoinHostPort(host, port)
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
	// Requests without a token act as the user local, unless a token is required.
	local, err := st.EnsureLocalUser(ctx)
	if err != nil {
		return err
	}
	var tokenUsers int
	if cfg.TokenRequired() {
		if tokenUsers, err = st.TokenUserCount(ctx); err != nil {
			return err
		}
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.Listen)
	if err != nil {
		return err
	}
	defer ln.Close()
	if err := st.RecordRetention(ctx, cfg.RetentionPolicies()); err != nil {
		return err
	}
	sess := server.Session{Store: st, LocalUser: local}
	// server.New applies the limits and the retention of memories to the store: the cleanup,
	// which reads them, starts after it. It runs while the server serves, and stops before the
	// store is closed.
	srv := server.New(cfg, sess)
	stopCleanup := startCleanup(ctx, st, cfg)
	defer stopCleanup()
	hs := &http.Server{Handler: server.Handler(srv, sess)}

	log.Printf("version %s, database %s", version.Version, cfg.DBPath)
	addr := listenAddr(cfg.Listen, ln.Addr())
	log.Printf("serving MCP at http://%s%s", addr, server.MCPPath)
	log.Print(describeCleanupPolicy(cfg))
	log.Print(describeMemoryPolicy(cfg))
	if cfg.ListensRemote() {
		log.Printf("warning: listening on %s without TLS: tokens and messages travel unencrypted. "+
			"Use it only on a trusted network, or put an HTTPS reverse proxy in front.", addr)
	}
	if cfg.TokenRequired() {
		log.Print("every request needs a token")
		if tokenUsers == 0 {
			log.Printf("no user has a token yet, so every request is rejected: create one with agent-lounge-mcp user add <name> --db %s", cfg.DBPath)
		}
	}
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

	// Tell agents first: tool calls fail with the shutdown error from now on, and waiting
	// chat_inbox calls return it. Keep
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
	srv.BeginShutdown()
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
