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
	fmt.Fprintln(w, "usage: agent-lounge-mcp user add <name> [--db FILE]          create a user and print its token")
	fmt.Fprintln(w, "       agent-lounge-mcp user list [--db FILE]                list the users")
	fmt.Fprintln(w, "       agent-lounge-mcp user rotate <name> [--db FILE]       print a new token; the old one stops working")
	fmt.Fprintln(w, "       agent-lounge-mcp user remove <name> --yes [--db FILE] delete the user, its sessions and messages")
	fmt.Fprintf(w, "A user name is %s. The commands open the database file directly and work while a server uses it.\n", store.UserNameRule)
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
	case "add", "list", "rotate", "remove":
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
	pos, err := parseInterleaved(flags, args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case err != nil:
		return exitUsage // the flag package printed the error and the usage
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
		err = userAdd(ctx, st, pos[0], *db, stdout, stderr)
	case "list":
		err = userList(ctx, st, stdout)
	case "rotate":
		err = userRotate(ctx, st, pos[0], *db, stdout, stderr)
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

func userAdd(ctx context.Context, st *store.Store, name, db string, stdout, stderr io.Writer) error {
	if err := store.ValidateUserName(name); err != nil {
		return err
	}
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	if _, err := st.AddUser(ctx, name, auth.Hash(token)); err != nil {
		return err
	}
	fmt.Fprintln(stdout, token)
	tokenGuidance(stderr, name, db)
	return nil
}

func userRotate(ctx context.Context, st *store.Store, name, db string, stdout, stderr io.Writer) error {
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	if err := st.RotateToken(ctx, name, auth.Hash(token)); err != nil {
		return err
	}
	fmt.Fprintln(stdout, token)
	fmt.Fprintf(stderr, "The previous token of user %s no longer works; its sessions and messages are kept.\n", name)
	tokenGuidance(stderr, name, db)
	return nil
}

// userList prints one line per user, separated by tabs: name, creation time, number of
// sessions, and the last activity of its sessions ("-" for none). Times are RFC 3339 in UTC.
func userList(ctx context.Context, st *store.Store, stdout io.Writer) error {
	users, err := st.ListUsers(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "NAME\tCREATED\tSESSIONS\tLAST_USED")
	for _, u := range users {
		last := "-"
		if !u.LastUsedAt.IsZero() {
			last = u.LastUsedAt.Format(time.RFC3339)
		}
		fmt.Fprintf(stdout, "%s\t%s\t%d\t%s\n", u.Name, u.CreatedAt.Format(time.RFC3339), u.Sessions, last)
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
	return fmt.Sprintf("%d sessions, %d messages, %d deliveries", d.Sessions, d.Messages, d.Deliveries)
}
