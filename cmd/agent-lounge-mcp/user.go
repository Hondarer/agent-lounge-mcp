package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/auth"
	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// Exit codes of the user subcommand.
const (
	exitOK    = 0
	exitFail  = 1 // the command could not be done: no such user, an invalid name, a database error
	exitUsage = 2 // the command line is malformed
)

func userUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: agent-lounge-mcp user add <name> [--expires-in DURATION] [--db FILE]      create a user and print its token")
	fmt.Fprintln(w, "       agent-lounge-mcp user list [--db FILE]                                    list the users")
	fmt.Fprintln(w, "       agent-lounge-mcp user rotate <name> [--expires-in DURATION] [--db FILE]   print a new token; the old one stops working")
	fmt.Fprintln(w, "       agent-lounge-mcp user expiry <name> (--in DURATION | --never) [--db FILE] change only the expiry of the token")
	fmt.Fprintln(w, "       agent-lounge-mcp user remove <name> --yes [--db FILE]                     delete the user, its sessions, messages and memories")
	fmt.Fprintf(w, "A user name is %s. The commands open the database file directly and work while a server uses it.\n", store.UserNameRule)
	fmt.Fprintln(w, "A DURATION is counted from now and is at least 1s, such as 90d, 1d12h or 36h (d is 24 hours). "+
		"Without --expires-in, the token does not expire; user rotate does not keep the expiry of the previous token.")
}

// expiryFlag is a duration flag of at least a second (see config.ParseDuration), and
// records whether it was given.
type expiryFlag struct {
	d   time.Duration
	set bool
}

func (f *expiryFlag) String() string {
	if f == nil || !f.set {
		return ""
	}
	return config.FormatDuration(f.d)
}

func (f *expiryFlag) Set(s string) error {
	d, err := config.ParseDuration(s)
	if err != nil {
		return err
	}
	// The expiry is stored in milliseconds: a shorter duration would make a token that is
	// expired when it is issued.
	if d < time.Second {
		return fmt.Errorf("invalid duration %q: must be at least 1s", s)
	}
	f.d, f.set = d, true
	return nil
}

// expiresAt returns the time the duration ends, counted from now, or the zero time (no
// expiry) when the flag was not given.
func (f *expiryFlag) expiresAt(now time.Time) time.Time {
	if !f.set {
		return time.Time{}
	}
	return now.Add(f.d)
}

// formatExpiry renders an expiry for the output of the commands: RFC 3339 in UTC, or "-" for
// none.
func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

// expiryGuidance is the line of the token guidance that states the expiry.
func expiryGuidance(stderr io.Writer, name string, expiresAt time.Time) {
	if expiresAt.IsZero() {
		fmt.Fprintf(stderr, "The token of user %s does not expire.\n", name)
		return
	}
	fmt.Fprintf(stderr, "The token of user %s expires at %s.\n", name, formatExpiry(expiresAt))
}

// runUser runs the user subcommand with its arguments (after "user") and returns the exit code.
// Tokens are written to stdout only, everything else to stderr.
func runUser(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		userUsage(stderr)
		return exitUsage
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "help", "-h", "-help", "--help":
		userUsage(stdout)
		return exitOK
	case "add", "list", "rotate", "expiry", "remove":
	default:
		fmt.Fprintf(stderr, "agent-lounge-mcp: unknown user command %q\n", cmd)
		userUsage(stderr)
		return exitUsage
	}

	flags := flag.NewFlagSet("agent-lounge-mcp user "+cmd, flag.ContinueOnError)
	flags.SetOutput(stderr)
	db := flags.String("db", config.Default().DBPath, "SQLite database file")
	var yes *bool
	if cmd == "remove" {
		yes = flags.Bool("yes", false, "remove without asking: required")
	}
	var expiresIn, in expiryFlag
	var never *bool
	switch cmd {
	case "add", "rotate":
		flags.Var(&expiresIn, "expires-in", "the token expires after this `DURATION` from now (90d, 1d12h, 36h); none when omitted")
	case "expiry":
		flags.Var(&in, "in", "the token expires after this `DURATION` from now (90d, 1d12h, 36h)")
		never = flags.Bool("never", false, "the token does not expire")
	}
	pos, err := parseInterleaved(flags, args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case err != nil:
		return exitUsage // the flag package printed the error and the usage
	}
	if cmd == "expiry" && in.set == *never {
		fmt.Fprintln(stderr, "agent-lounge-mcp: user expiry takes one of --in DURATION and --never")
		userUsage(stderr)
		return exitUsage
	}
	wantArgs := 1
	if cmd == "list" {
		wantArgs = 0
	}
	if len(pos) != wantArgs {
		if wantArgs == 0 {
			fmt.Fprintf(stderr, "agent-lounge-mcp: user %s takes no argument\n", cmd)
		} else {
			fmt.Fprintf(stderr, "agent-lounge-mcp: user %s takes one user name\n", cmd)
		}
		userUsage(stderr)
		return exitUsage
	}
	if *db == "" {
		fmt.Fprintln(stderr, "agent-lounge-mcp: --db must not be empty")
		return exitUsage
	}
	// Only add may create the database: the other commands act on users that must exist.
	if cmd != "add" {
		if _, err := os.Stat(*db); errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(stderr, "agent-lounge-mcp: the database file %s does not exist: check --db\n", *db)
			return exitFail
		}
	}
	st, err := store.Open(ctx, *db)
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
	}
	defer st.Close()

	switch cmd {
	case "add":
		err = userAdd(ctx, st, pos[0], *db, expiresIn.expiresAt(time.Now()), stdout, stderr)
	case "list":
		err = userList(ctx, st, stdout)
	case "rotate":
		err = userRotate(ctx, st, pos[0], *db, expiresIn.expiresAt(time.Now()), stdout, stderr)
	case "expiry":
		err = userExpiry(ctx, st, pos[0], in.expiresAt(time.Now()), stdout)
	case "remove":
		err = userRemove(ctx, st, pos[0], *yes, stdout, stderr)
	}
	if err != nil {
		fmt.Fprintf(stderr, "agent-lounge-mcp: %v\n", err)
		return exitFail
	}
	return exitOK
}

// parseInterleaved parses flags that may come before or after the positional arguments, as in
// "user add alice --db x.db", and returns the positional arguments.
func parseInterleaved(flags *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		args = flags.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

// tokenGuidance is written to stderr after a token is printed.
func tokenGuidance(stderr io.Writer, name, db string) {
	fmt.Fprintf(stderr, "The token of user %s is printed above. It is shown only this once: the database keeps only its hash, so it cannot be read again.\n", name)
	fmt.Fprintln(stderr, "Register the server in the user's agents with the HTTP header \"Authorization: Bearer <token>\" (see the user guide for each agent).")
	fmt.Fprintf(stderr, "If the token leaks, issue a new one with: agent-lounge-mcp user rotate %s --db %s\n", name, db)
}

func userAdd(ctx context.Context, st *store.Store, name, db string, expiresAt time.Time, stdout, stderr io.Writer) error {
	if err := store.ValidateUserName(name); err != nil {
		return err
	}
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	if _, err := st.AddUser(ctx, name, auth.Hash(token), expiresAt); err != nil {
		return err
	}
	fmt.Fprintln(stdout, token)
	tokenGuidance(stderr, name, db)
	expiryGuidance(stderr, name, expiresAt)
	return nil
}

func userRotate(ctx context.Context, st *store.Store, name, db string, expiresAt time.Time, stdout, stderr io.Writer) error {
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	if err := st.RotateToken(ctx, name, auth.Hash(token), expiresAt); err != nil {
		return err
	}
	fmt.Fprintln(stdout, token)
	fmt.Fprintf(stderr, "The previous token of user %s no longer works; its sessions and messages are kept.\n", name)
	tokenGuidance(stderr, name, db)
	expiryGuidance(stderr, name, expiresAt)
	return nil
}

// userExpiry changes the expiry of the user's token (the zero time for none) and prints the
// result in one line. The token stays the same; an expired token works again when the new
// expiry is later than now.
func userExpiry(ctx context.Context, st *store.Store, name string, expiresAt time.Time, stdout io.Writer) error {
	if err := st.SetTokenExpiry(ctx, name, expiresAt); err != nil {
		return err
	}
	if expiresAt.IsZero() {
		fmt.Fprintf(stdout, "the token of user %s does not expire\n", name)
	} else {
		fmt.Fprintf(stdout, "the token of user %s expires at %s\n", name, formatExpiry(expiresAt))
	}
	return nil
}

// userList prints one line per user, separated by tabs: name, creation time, number of
// sessions, last use, and the expiry of its token ("-" for none, and for local). The last use
// is the later of the last request with the user's token and the last activity of its sessions
// ("-" for neither). Times are RFC 3339 in UTC.
func userList(ctx context.Context, st *store.Store, stdout io.Writer) error {
	users, err := st.ListUsers(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "NAME\tCREATED\tSESSIONS\tLAST_USED\tEXPIRES")
	for _, u := range users {
		last := "-"
		if !u.LastUsedAt.IsZero() {
			last = u.LastUsedAt.Format(time.RFC3339)
		}
		fmt.Fprintf(stdout, "%s\t%s\t%d\t%s\t%s\n", u.Name, u.CreatedAt.Format(time.RFC3339), u.Sessions, last, formatExpiry(u.TokenExpiresAt))
	}
	return nil
}

func userRemove(ctx context.Context, st *store.Store, name string, yes bool, stdout, stderr io.Writer) error {
	if !yes {
		data, err := st.CountUserData(ctx, name)
		if err != nil {
			return err
		}
		return fmt.Errorf("removing user %s would delete %s; nothing was removed: run again with --yes to remove them", name, describeData(data))
	}
	data, err := st.RemoveUser(ctx, name)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed user %s: %s\n", name, describeData(data))
	return nil
}

func describeData(d store.UserData) string {
	return plural(d.Sessions, "session", "sessions") + ", " + plural(d.Messages, "message", "messages") + ", " +
		plural(d.Deliveries, "delivery", "deliveries") + ", " + plural(d.Memories, "memory", "memories")
}

// plural renders a count with the form of the noun it takes in English: 1 session, 2 sessions.
// The administration commands and the log lines use it for every count they print.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
