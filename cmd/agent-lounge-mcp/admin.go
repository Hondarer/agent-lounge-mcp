package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/export"
	"github.com/hondarer/agent-lounge-mcp/internal/names"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// The session, memory, message and export subcommands open the database file directly, as the user
// subcommand does, and work while a server uses it. They use the exit codes of the user subcommand.
// Each read is one transaction, so the rows are the database at one moment.

func sessionUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp session list [--user NAME] [--db FILE]          list the sessions")
	fmt.Fprintln(w, "       agent-lounge-mcp session remove <user>/<name> --yes [--db FILE]  delete the session, its messages and deliveries")
	fmt.Fprintln(w, "The commands open the database file directly and work while a server uses it.")
	fmt.Fprintln(w, "Put options before -- to pass an address starting with a hyphen.")
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
	if cmd == "list" {
		flags.Visit(func(f *flag.Flag) {
			if f.Name == "user" {
				*user, err = names.User(*user)
			}
		})
	} else {
		owner, err = names.User(owner)
		if err == nil {
			name, err = names.Session(name)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
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
// tabs: address (user/name), agent, last use (RFC 3339 in UTC), number of unread messages,
// repository and branch. An empty value is printed as "-".
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
	fmt.Fprintln(stdout, "ADDRESS\tAGENT\tLAST_USED\tUNREAD\tREPOSITORY\tBRANCH")
	for _, s := range sessions {
		fmt.Fprintf(stdout, "%s/%s\t%s\t%s\t%d\t%s\t%s\n", s.UserName, s.Name, dash(s.Agent), s.LastSeenAt.Format(time.RFC3339), s.Unread, dash(s.Repository), dash(s.Branch))
	}
	return nil
}

func describeSessionData(d store.SessionData) string {
	return plural(d.Messages, "message", "messages") + ", " + plural(d.Deliveries, "delivery", "deliveries")
}

func exportUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp export [--user NAME] [--scope KIND [--scope-id ID]] [--since TIME] [--until TIME] --out DIR [--format markdown] [--db FILE]")
	fmt.Fprintf(w, "Writes messages and memories of the database as Markdown files: under DIR/%s one file per conversation,\n", export.ChatDir)
	fmt.Fprintf(w, "under DIR/%s one file per scope of memories. Without a filter, every message and memory is written.\n", export.MemoryDir)
	fmt.Fprintln(w, "With a filter, only the files that contain a selected record are written.")
	fmt.Fprintln(w, "Files of the same name are overwritten; other files are left alone. It works while a server uses the database.")
	fmt.Fprintln(w, "--user, --scope, --scope-id, --since and --until mean the same as for message list. For a memory, --user matches")
	fmt.Fprintln(w, "a user scope with that identifier or a memory that user created, and --since and --until use the time it was updated.")
	fmt.Fprintln(w, "--scope-id needs --scope. Put options before -- to pass a name starting with a hyphen.")
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
	user, scope, scopeID, since, until := addFilterFlags(flags)
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
	filter, err := filterFromFlags(flags, *user, *scope, *scopeID, *since, *until)
	if err != nil {
		return reportFilter(stderr, exportUsage, err)
	}
	st, code := openExisting(ctx, *db, stderr)
	if st == nil {
		return code
	}
	defer st.Close()
	// No filter runs Export itself, which is the unfiltered read.
	read := func(onMessage func(store.ExportMessage) error, onMemory func(store.Memory) error) error {
		return st.Export(ctx, onMessage, onMemory)
	}
	if !filter.IsZero() {
		read = func(onMessage func(store.ExportMessage) error, onMemory func(store.Memory) error) error {
			return st.ExportSelected(ctx, filter, onMessage, onMemory)
		}
	}
	sum, err := export.WriteMarkdown(*out, read)
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
	}
	fmt.Fprintf(stdout, "exported %s and %s to %s in %s\n", plural(sum.Messages, "message", "messages"),
		plural(sum.Memories, "memory", "memories"), plural(sum.Files, "file", "files"), *out)
	if filter.IsZero() {
		fmt.Fprintln(stderr, "The files contain the messages and memories of every user of the database: keep them where only the people who may read them can.")
	} else {
		fmt.Fprintln(stderr, "The files contain only the messages and memories that match the filter: keep them where only the people who may read them can.")
	}
	return exitOK
}

func memoryUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp memory list [--scope user|repository|branch|shared] [--scope-id ID] [--memory-half-life DURATION] [--memory-fade-threshold X] [--db FILE]")
	for _, kind := range store.ScopeKinds() {
		fmt.Fprintf(w, "           [--memory-%[1]s-half-life DURATION] [--memory-%[1]s-fade-threshold X]\n", kind)
	}
	fmt.Fprintln(w, "       agent-lounge-mcp memory remove <ID> --yes [--db FILE]  delete the memory")
	fmt.Fprintln(w, "The commands open the database file directly and work while a server uses it.")
	fmt.Fprintln(w, "memory list uses the last recorded server settings, or built-in defaults when none are recorded. Common options override those defaults. Per-kind options take precedence.")
	fmt.Fprintln(w, "--scope-id needs --scope.")
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
	var scope, scopeID *string
	var yes *bool
	var applyRetentionFlags func()
	if cmd == "list" {
		scope = flags.String("scope", "", "list only the memories of this scope kind: user, repository, branch or shared")
		scopeID = flags.String("scope-id", "", "with --scope, list only the memories of this scope identifier")
		applyRetentionFlags = addMemoryRetentionFlags(flags, &cfg, false)
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
	var scopeIDSet bool
	var common config.RetentionOverride
	if cmd == "list" {
		flags.Visit(func(f *flag.Flag) {
			switch f.Name {
			case "scope-id":
				scopeIDSet = true
			case "memory-half-life":
				common.HalfLife = &cfg.MemoryHalfLife
			case "memory-fade-threshold":
				common.Threshold = &cfg.MemoryFadeThreshold
			}
		})
		switch {
		case len(pos) != 0:
			fmt.Fprintln(stderr, "agent-lounge-mcp: memory list takes no argument")
			memoryUsage(stderr)
			return exitUsage
		case *scope != "" && !store.ValidScope(*scope):
			fmt.Fprintf(stderr, "agent-lounge-mcp: unknown --scope %q: use user, repository, branch or shared\n", *scope)
			return exitUsage
		case scopeIDSet && *scope == "":
			fmt.Fprintln(stderr, "agent-lounge-mcp: --scope-id needs --scope")
			memoryUsage(stderr)
			return exitUsage
		}
		if err := config.ValidateFadeThreshold(cfg.MemoryFadeThreshold); err != nil {
			fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
			return exitUsage
		}
		applyRetentionFlags()
		if err := cfg.ValidateRetentionOverrides(); err != nil {
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
		policies, readErr := st.RecordedRetention(ctx)
		if readErr != nil {
			fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", readErr)
			return exitFail
		}
		for _, kind := range store.ScopeKinds() {
			policies.ByKind[kind] = cfg.MemoryRetentionByKind[kind].Apply(common.Apply(policies.For(kind)))
		}
		policies.Common = common.Apply(policies.Common)
		st.SetRetentionPolicies(policies)
		var idFilter *string
		if scopeIDSet {
			idFilter = scopeID
		}
		err = memoryList(ctx, st, *scope, idFilter, stdout)
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
func memoryList(ctx context.Context, st *store.Store, scope string, scopeID *string, stdout io.Writer) error {
	ms, err := st.ListMemories(ctx, scope, scopeID)
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
		fmt.Fprintf(stdout, "%d\t%s\t%s\t%s\t%.2f\t%s\t%s/%s\t%s\n", m.ID, m.Scope.Kind, m.Scope.ID, state, store.RoundRetention(m.Retention),
			m.UpdatedAt.Format(time.RFC3339), m.CreatedBy.User, m.CreatedBy.Session, memoryLabel(m))
	}
	return nil
}

func messageUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp message list [--user NAME] [--scope KIND [--scope-id ID]] [--since TIME] [--until TIME]")
	fmt.Fprintln(w, "                                     [--state queued|delivered|acked|expired] [--details] [--limit N] [--before-id ID] [--db FILE]")
	fmt.Fprintln(w, "       agent-lounge-mcp message show <ID> [--db FILE]")
	fmt.Fprintln(w, "The commands open the database file directly and work while a server uses it.")
	fmt.Fprintln(w, "message list prints one message per line, newest first (ID descending), without the body. --limit defaults to 100.")
	fmt.Fprintln(w, "--before-id keeps the messages whose ID is smaller. --details adds one indented line per recipient")
	fmt.Fprintln(w, "(user/name, state, and the time it entered that state).")
	fmt.Fprintln(w, "--state keeps the messages that have at least one delivery in that state.")
	fmt.Fprintln(w, "--since and --until are RFC 3339. --since is inclusive and --until is exclusive; both use the time the message was sent.")
	fmt.Fprintln(w, "--user matches the stored name of the sender, the direct recipient or a delivery recipient, including a user that was removed.")
	fmt.Fprintln(w, "--scope-id needs --scope. Put options before -- to pass a name starting with a hyphen.")
}

// usageErr is a malformed command line (exit code 2). An invalid user name is not one:
// the other commands report that as a failure (exit code 1).
type usageErr string

func (e usageErr) Error() string { return string(e) }

func addFilterFlags(fs *flag.FlagSet) (user, scope, scopeID, since, until *string) {
	user = fs.String("user", "", "select records stored with this user name")
	scope = fs.String("scope", "", "select records of this scope kind: user, repository, branch or shared")
	scopeID = fs.String("scope-id", "", "with --scope, select records of this scope identifier")
	since = fs.String("since", "", "inclusive RFC 3339 lower bound of the time range")
	until = fs.String("until", "", "exclusive RFC 3339 upper bound of the time range")
	return
}

func filterFromFlags(fs *flag.FlagSet, user, scope, scopeID, since, until string) (store.RecordFilter, error) {
	var f store.RecordFilter
	var userSet, scopeIDSet, sinceSet, untilSet bool
	fs.Visit(func(fl *flag.Flag) {
		switch fl.Name {
		case "user":
			userSet = true
		case "scope-id":
			scopeIDSet = true
		case "since":
			sinceSet = true
		case "until":
			untilSet = true
		}
	})
	if userSet {
		name, err := names.User(user)
		if err != nil {
			return store.RecordFilter{}, err
		}
		f.User = name
	}
	if scope != "" && !store.ValidScope(scope) {
		return store.RecordFilter{}, usageErr(fmt.Sprintf("unknown --scope %q: use user, repository, branch or shared", scope))
	}
	f.Scope = scope
	if scopeIDSet && scope == "" {
		return store.RecordFilter{}, usageErr("--scope-id needs --scope")
	}
	if scopeIDSet {
		f.HasScopeID = true
		f.ScopeID = scopeID
	}
	var err error
	if sinceSet {
		if f.Since, err = parseFilterTime("since", since); err != nil {
			return store.RecordFilter{}, err
		}
	}
	if untilSet {
		if f.Until, err = parseFilterTime("until", until); err != nil {
			return store.RecordFilter{}, err
		}
	}
	return f, nil
}

func parseFilterTime(flagName, value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, usageErr(fmt.Sprintf("invalid --%s %q: use RFC 3339, such as 2026-11-23T08:10:00Z", flagName, value))
	}
	return t, nil
}

func reportFilter(stderr io.Writer, usage func(io.Writer), err error) int {
	fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
	var u usageErr
	if errors.As(err, &u) {
		usage(stderr)
		return exitUsage
	}
	return exitFail
}

// runMessage runs the message subcommand with its arguments (after "message") and returns
// the exit code.
func runMessage(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		messageUsage(stderr)
		return exitUsage
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "help", "-h", "-help", "--help":
		messageUsage(stdout)
		return exitOK
	case "list", "show":
	default:
		fmt.Fprintf(stderr, "agent-lounge-mcp: unknown message command %q\n", cmd)
		messageUsage(stderr)
		return exitUsage
	}
	flags := flag.NewFlagSet("agent-lounge-mcp message "+cmd, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		messageUsage(stderr)
		flags.PrintDefaults()
	}
	db := flags.String("db", config.Default().DBPath, "SQLite database file")
	var user, scope, scopeID, since, until, state *string
	var details *bool
	var limit *int
	var before *int64
	if cmd == "list" {
		user, scope, scopeID, since, until = addFilterFlags(flags)
		state = flags.String("state", "", "keep messages with a delivery in this state: queued, delivered, acked or expired")
		details = flags.Bool("details", false, "print one line per recipient under each message")
		limit = flags.Int("limit", 100, "most messages to print")
		before = flags.Int64("before-id", 0, "print only messages whose ID is smaller than this")
	}
	pos, err := parseInterleaved(flags, args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case err != nil:
		return exitUsage
	}
	var id int64
	var filter store.RecordFilter
	var beforeID int64
	if cmd == "list" {
		if len(pos) != 0 {
			fmt.Fprintln(stderr, "agent-lounge-mcp: message list takes no argument")
			messageUsage(stderr)
			return exitUsage
		}
		filter, err = filterFromFlags(flags, *user, *scope, *scopeID, *since, *until)
		if err != nil {
			return reportFilter(stderr, messageUsage, err)
		}
		var stateSet, beforeSet bool
		flags.Visit(func(fl *flag.Flag) {
			switch fl.Name {
			case "state":
				stateSet = true
			case "before-id":
				beforeSet = true
			}
		})
		if stateSet && !slices.Contains(store.DeliveryStates, *state) {
			fmt.Fprintf(stderr, "agent-lounge-mcp: unknown --state %q: use queued, delivered, acked or expired\n", *state)
			messageUsage(stderr)
			return exitUsage
		}
		if *limit < 1 {
			fmt.Fprintln(stderr, "agent-lounge-mcp: --limit must be at least 1")
			messageUsage(stderr)
			return exitUsage
		}
		if beforeSet && *before < 1 {
			fmt.Fprintln(stderr, "agent-lounge-mcp: --before-id must be a positive message ID")
			messageUsage(stderr)
			return exitUsage
		}
		if beforeSet {
			beforeID = *before
		}
	} else {
		if len(pos) == 1 {
			id, err = strconv.ParseInt(pos[0], 10, 64)
		}
		if len(pos) != 1 || err != nil || id < 1 {
			fmt.Fprintln(stderr, "agent-lounge-mcp: message show takes one message ID (a positive number, as message list shows it)")
			messageUsage(stderr)
			return exitUsage
		}
	}
	st, code := openExisting(ctx, *db, stderr)
	if st == nil {
		return code
	}
	defer st.Close()
	if cmd == "list" {
		q := store.MessageQuery{Filter: filter, Limit: *limit, BeforeID: beforeID, Details: *details}
		if state != nil {
			q.State = *state
		}
		msgs, err := st.ListMessages(ctx, q)
		if err != nil {
			fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
			return exitFail
		}
		writeMessageList(stdout, msgs, *details)
		return exitOK
	}
	m, err := st.MessageAdmin(ctx, id)
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
	}
	writeMessageShow(stdout, m)
	return exitOK
}

func writeMessageList(stdout io.Writer, msgs []store.AdminMessage, details bool) {
	fmt.Fprintln(stdout, "ID\tSENT\tFROM\tTO\tSCOPE\tSCOPE_ID\tQUEUED\tDELIVERED\tACKED\tEXPIRED")
	for _, m := range msgs {
		fmt.Fprintf(stdout, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\n",
			m.ID, formatStamp(m.SentAt), messageAddress(m.FromUser, m.FromName), messageTo(m),
			m.Scope.Kind, m.Scope.ID, m.Queued, m.Delivered, m.Acked, m.Expired)
		if details {
			writeRecipients(stdout, m.Deliveries)
		}
	}
}

func writeMessageShow(stdout io.Writer, m store.AdminMessage) {
	fmt.Fprintf(stdout, "ID: %d\n", m.ID)
	fmt.Fprintf(stdout, "From: %s\n", messageAddress(m.FromUser, m.FromName))
	fmt.Fprintf(stdout, "To: %s\n", messageTo(m))
	fmt.Fprintf(stdout, "Scope: %s %s\n", m.Scope.Kind, m.Scope.ID)
	fmt.Fprintf(stdout, "Sent: %s\n", formatStamp(m.SentAt))
	if len(m.Deliveries) == 0 {
		fmt.Fprintln(stdout, "Deliveries: none")
	} else {
		fmt.Fprintln(stdout, "Deliveries:")
		writeRecipients(stdout, m.Deliveries)
	}
	fmt.Fprintln(stdout)
	writeBody(stdout, m.Body)
}

func writeRecipients(stdout io.Writer, rs []store.AdminRecipient) {
	for _, r := range rs {
		fmt.Fprintf(stdout, "\t%s\t%s\t%s\n", messageAddress(r.User, r.Name), r.State, formatStamp(r.At))
	}
}

func messageAddress(user, name string) string { return user + "/" + name }

func messageTo(m store.AdminMessage) string {
	if m.Room != "" {
		return m.Room
	}
	return messageAddress(m.ToUser, m.ToName)
}

// formatStamp prints a UTC time as RFC 3339 with whole seconds, as session list and the
// export do. A stored fraction of a second is not shown. A filter can still name that
// instant: the bound is parsed as an RFC 3339 time (a fraction is allowed) and compared
// with the stored millisecond.
func formatStamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

func writeBody(w io.Writer, body string) {
	fmt.Fprint(w, body)
	if body == "" || !strings.HasSuffix(body, "\n") {
		fmt.Fprintln(w)
	}
}
