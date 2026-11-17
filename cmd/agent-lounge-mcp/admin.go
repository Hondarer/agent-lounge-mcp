package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/export"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// The session, memory and export subcommands open the database file directly, as the user subcommand
// does, and work while a server uses it. They use the exit codes of the user subcommand.

func sessionUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp session list [--user NAME] [--db FILE]          list the sessions")
	fmt.Fprintln(w, "       agent-lounge-mcp session remove <user>/<name> --yes [--db FILE]  delete the session, its messages and deliveries")
	fmt.Fprintln(w, "The commands open the database file directly and work while a server uses it.")
	fmt.Fprintln(w, "A connection joined as a removed session gets an error at its next tool call and can join again.")
}

// openExisting opens the database file for a command that needs it to exist. It returns the
// exit code to use when it fails.
func openExisting(ctx context.Context, db string, stderr io.Writer) (*store.Store, int) {
	if db == "" {
		fmt.Fprintln(stderr, "agent-lounge-mcp: --db must not be empty")
		return nil, exitUsage
	}
	if _, err := os.Stat(db); errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(stderr, "agent-lounge-mcp: the database file %s does not exist: check --db\n", db)
		return nil, exitFail
	}
	st, err := store.Open(ctx, db)
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return nil, exitFail
	}
	return st, exitOK
}

// runSession runs the session subcommand with its arguments (after "session") and returns the
// exit code.
func runSession(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		sessionUsage(stderr)
		return exitUsage
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "help", "-h", "-help", "--help":
		sessionUsage(stdout)
		return exitOK
	case "list", "remove":
	default:
		fmt.Fprintf(stderr, "agent-lounge-mcp: unknown session command %q\n", cmd)
		sessionUsage(stderr)
		return exitUsage
	}
	flags := flag.NewFlagSet("agent-lounge-mcp session "+cmd, flag.ContinueOnError)
	flags.SetOutput(stderr)
	db := flags.String("db", config.Default().DBPath, "SQLite database file")
	var user *string
	var yes *bool
	if cmd == "list" {
		user = flags.String("user", "", "list only the sessions of this user")
	} else {
		yes = flags.Bool("yes", false, "remove without asking: required")
	}
	pos, err := parseInterleaved(flags, args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case err != nil:
		return exitUsage // the flag package printed the error and the usage
	}
	var owner, name string
	if cmd == "list" {
		if len(pos) != 0 {
			fmt.Fprintln(stderr, "agent-lounge-mcp: session list takes no argument")
			sessionUsage(stderr)
			return exitUsage
		}
	} else {
		var ok bool
		if len(pos) == 1 {
			owner, name, ok = strings.Cut(pos[0], "/")
		}
		if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
			fmt.Fprintln(stderr, "agent-lounge-mcp: session remove takes one session address: <user>/<name> (local/<name> for a session without a token)")
			sessionUsage(stderr)
			return exitUsage
		}
	}
	st, code := openExisting(ctx, *db, stderr)
	if st == nil {
		return code
	}
	defer st.Close()

	if cmd == "list" {
		err = sessionList(ctx, st, *user, stdout)
	} else {
		if !*yes {
			data, err := st.CountSessionData(ctx, owner, name)
			if err != nil {
				fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
				return exitFail
			}
			fmt.Fprintf(stderr, "agent-lounge-mcp: removing session %s/%s would delete %s; nothing was removed: run again with --yes to remove it\n",
				owner, name, describeSessionData(data))
			return exitFail
		}
		var data store.SessionData
		if data, err = st.RemoveSession(ctx, owner, name); err == nil {
			fmt.Fprintf(stdout, "removed session %s/%s: %s\n", owner, name, describeSessionData(data))
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
	}
	return exitOK
}

// sessionList prints a header line and one line per session, in address order, separated by
// tabs: address (user/name), agent, last use (RFC 3339 in UTC), number of unread messages and
// repository. An empty value is printed as "-".
func sessionList(ctx context.Context, st *store.Store, user string, stdout io.Writer) error {
	sessions, err := st.AllSessions(ctx, user)
	if err != nil {
		return err
	}
	dash := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	fmt.Fprintln(stdout, "ADDRESS\tAGENT\tLAST_USED\tUNREAD\tREPOSITORY")
	for _, s := range sessions {
		fmt.Fprintf(stdout, "%s/%s\t%s\t%s\t%d\t%s\n", s.UserName, s.Name, dash(s.Agent), s.LastSeenAt.Format(time.RFC3339), s.Unread, dash(s.Repository))
	}
	return nil
}

func describeSessionData(d store.SessionData) string {
	return plural(d.Messages, "message", "messages") + ", " + plural(d.Deliveries, "delivery", "deliveries")
}

func exportUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp export --out DIR [--format markdown] [--db FILE]")
	fmt.Fprintf(w, "Writes every message and memory of the database, of every user, as Markdown files: under DIR/%s one file per conversation,\n", export.ChatDir)
	fmt.Fprintf(w, "under DIR/%s one file per scope of memories.\n", export.MemoryDir)
	fmt.Fprintln(w, "Files of the same name are overwritten; other files are left alone. It works while a server uses the database.")
}

// runExport runs the export subcommand with its arguments (after "export") and returns the
// exit code.
func runExport(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("agent-lounge-mcp export", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		exportUsage(stderr)
		flags.PrintDefaults()
	}
	db := flags.String("db", config.Default().DBPath, "SQLite database file")
	out := flags.String("out", "", "directory to write into (created when missing): required")
	format := flags.String("format", "markdown", "file format: markdown (the only one)")
	pos, err := parseInterleaved(flags, args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case err != nil:
		return exitUsage
	case len(pos) > 0:
		fmt.Fprintf(stderr, "agent-lounge-mcp: export takes no argument (got %q)\n", pos[0])
		exportUsage(stderr)
		return exitUsage
	case *out == "":
		fmt.Fprintln(stderr, "agent-lounge-mcp: export needs --out DIR")
		exportUsage(stderr)
		return exitUsage
	case *format != "markdown":
		fmt.Fprintf(stderr, "agent-lounge-mcp: unknown --format %q: the only format is markdown\n", *format)
		return exitUsage
	}
	st, code := openExisting(ctx, *db, stderr)
	if st == nil {
		return code
	}
	defer st.Close()
	sum, err := export.WriteMarkdown(*out, func(onMessage func(store.ExportMessage) error, onMemory func(store.Memory) error) error {
		return st.Export(ctx, onMessage, onMemory)
	})
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
	}
	fmt.Fprintf(stdout, "exported %s and %s to %s in %s\n", plural(sum.Messages, "message", "messages"),
		plural(sum.Memories, "memory", "memories"), plural(sum.Files, "file", "files"), *out)
	fmt.Fprintln(stderr, "The files contain the messages and memories of every user of the database: keep them where only the people who may read them can.")
	return exitOK
}

func memoryUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp memory list [--scope user|repository] [--memory-half-life DURATION] [--memory-fade-threshold X] [--db FILE]")
	fmt.Fprintln(w, "                                                     list the memories with their state")
	fmt.Fprintln(w, "       agent-lounge-mcp memory remove <ID> --yes [--db FILE]  delete the memory")
	fmt.Fprintln(w, "The commands open the database file directly and work while a server uses it.")
	fmt.Fprintln(w, "memory list computes the state with --memory-half-life and --memory-fade-threshold: pass the values the server uses.")
}

// runMemory runs the memory subcommand with its arguments (after "memory") and returns the
// exit code.
func runMemory(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		memoryUsage(stderr)
		return exitUsage
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "help", "-h", "-help", "--help":
		memoryUsage(stdout)
		return exitOK
	case "list", "remove":
	default:
		fmt.Fprintf(stderr, "agent-lounge-mcp: unknown memory command %q\n", cmd)
		memoryUsage(stderr)
		return exitUsage
	}
	flags := flag.NewFlagSet("agent-lounge-mcp memory "+cmd, flag.ContinueOnError)
	flags.SetOutput(stderr)
	db := flags.String("db", config.Default().DBPath, "SQLite database file")
	cfg := config.Default()
	var scope *string
	var yes *bool
	if cmd == "list" {
		scope = flags.String("scope", "", "list only the memories of this scope kind: user or repository")
		addMemoryRetentionFlags(flags, &cfg)
	} else {
		yes = flags.Bool("yes", false, "remove without asking: required")
	}
	pos, err := parseInterleaved(flags, args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case err != nil:
		return exitUsage // the flag package printed the error and the usage
	}
	var id int64
	if cmd == "list" {
		switch {
		case len(pos) != 0:
			fmt.Fprintln(stderr, "agent-lounge-mcp: memory list takes no argument")
			memoryUsage(stderr)
			return exitUsage
		case *scope != "" && !store.ValidScope(*scope):
			fmt.Fprintf(stderr, "agent-lounge-mcp: unknown --scope %q: use user or repository\n", *scope)
			return exitUsage
		}
		if err := config.ValidateFadeThreshold(cfg.MemoryFadeThreshold); err != nil {
			fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
			return exitUsage
		}
	} else {
		if len(pos) == 1 {
			id, err = strconv.ParseInt(pos[0], 10, 64)
		}
		if len(pos) != 1 || err != nil || id < 1 {
			fmt.Fprintln(stderr, "agent-lounge-mcp: memory remove takes one memory ID (a positive number, as memory list shows it)")
			memoryUsage(stderr)
			return exitUsage
		}
	}
	st, code := openExisting(ctx, *db, stderr)
	if st == nil {
		return code
	}
	defer st.Close()

	if cmd == "list" {
		st.SetRetention(cfg.Retention())
		err = memoryList(ctx, st, *scope, stdout)
	} else {
		if !*yes {
			m, err := st.MemoryByID(ctx, id)
			if err != nil {
				fmt.Fprintf(stderr, "agent-lounge-mcp: memory %d: %v\n", id, err)
				return exitFail
			}
			fmt.Fprintf(stderr, "agent-lounge-mcp: removing memory %d (scope %s %s: %s) would delete it; nothing was removed: run again with --yes to remove it\n",
				id, m.Scope.Kind, m.Scope.ID, memoryLabel(m))
			return exitFail
		}
		var m store.Memory
		if m, err = st.RemoveMemory(ctx, id); err == nil {
			fmt.Fprintf(stdout, "removed memory %d (scope %s %s: %s)\n", id, m.Scope.Kind, m.Scope.ID, memoryLabel(m))
		} else {
			err = fmt.Errorf("memory %d: %w", id, err)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
	}
	return exitOK
}

// memoryLabel names a memory in one line: its title, or the first line of its body, cut to 40
// characters, with tabs replaced by spaces.
func memoryLabel(m store.Memory) string {
	s := m.Title
	if s == "" {
		s, _, _ = strings.Cut(strings.TrimSpace(m.Body), "\n")
		s = strings.TrimSpace(s)
	}
	s = strings.ReplaceAll(s, "\t", " ")
	if r := []rune(s); len(r) > 40 {
		s = string(r[:40]) + "..."
	}
	return s
}

// memoryList prints a header line and one line per memory, in the order scope, scope ID, ID,
// separated by tabs: ID, scope kind, scope ID, state (pinned, active or faded), retention
// score, last update (RFC 3339 in UTC), creator (user/session) and title.
func memoryList(ctx context.Context, st *store.Store, scope string, stdout io.Writer) error {
	ms, err := st.ListMemories(ctx, scope)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "ID\tSCOPE\tSCOPE_ID\tSTATE\tRETENTION\tUPDATED\tCREATED_BY\tTITLE")
	for _, m := range ms {
		state := "active"
		switch {
		case m.Pinned:
			state = "pinned"
		case m.Faded:
			state = "faded"
		}
		fmt.Fprintf(stdout, "%d\t%s\t%s\t%s\t%.2f\t%s\t%s/%s\t%s\n", m.ID, m.Scope.Kind, m.Scope.ID, state, m.Retention,
			m.UpdatedAt.Format(time.RFC3339), m.CreatedBy.User, m.CreatedBy.Session, memoryLabel(m))
	}
	return nil
}
