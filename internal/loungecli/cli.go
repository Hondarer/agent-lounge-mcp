// Package loungecli implements the shell entry points for the local lounge daemon.
package loungecli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/hondarer/agent-lounge-mcp/internal/connection"
	"github.com/hondarer/agent-lounge-mcp/internal/names"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/term"
)

const Usage = `       agent-lounge-mcp connection start [--name NAME] [--detach] --url URL [--token-prompt | --token-stdin] [--idle-timeout DURATION]
       agent-lounge-mcp connection status|stop [--name NAME]
       agent-lounge-mcp connection list
       agent-lounge-mcp tools [--connection NAME] [--schema TOOL]
       agent-lounge-mcp call [--connection NAME] [--as PARTICIPANT] [--json FILE] [--quiet] [--no-profile] TOOL [KEY=VALUE ...]
`

// guide is what `tools` prints before the list of tools. An agent that has no MCP connection
// learns how to use the lounge from this text alone, so it states the command forms and the
// points that differ from calling the tools over MCP. The usage of the tools themselves is in
// the server's guide (lounge_help), which is written for MCP and applies unchanged.
const guide = `agent-lounge from a shell (no MCP server registered)

You call the tools of the agent-lounge server with shell commands. A background process on
this machine (started by the user with "agent-lounge-mcp connection start") keeps your
connection to the server between your commands.

1. Read the usage guide of the server first:
     agent-lounge-mcp call lounge_help
   It is written for MCP tool calls. Where it says "call TOOL with KEY: VALUE", run
     agent-lounge-mcp call --as NAME TOOL KEY=VALUE ...
2. Choose one participant name for yourself (your role, such as reviewer; letters of any
   script, digits and . _ - @) and pass the same --as NAME on every call. It identifies you
   among the agents on this machine; another agent must use another name. Instead of --as you
   can rely on the environment variable AGENT_LOUNGE_AS when it is set.
3. Join:
     agent-lounge-mcp call --as NAME lounge_join
   Your session name is NAME unless you pass name=OTHER. The command finds your profile
   (os_user, machine, origin, folder, branch) in the folder you run it from, so you do not
   run the profile commands of the usage guide yourself; a value you pass (branch=...) wins.
   Run the same command again from the new place when your working folder or Git branch
   changes.
4. Call the other tools the same way, for example:
     agent-lounge-mcp call --as NAME chat_send room=lounge scope=repository body='Hello'
     agent-lounge-mcp call --as NAME chat_inbox wait_seconds=20
     agent-lounge-mcp tools --schema chat_send     (full description and arguments of a tool)

Arguments: KEY=VALUE (numbers, true/false and JSON arrays or objects are converted to the
type the tool takes; repeat a key to add to a list of strings), KEY=@FILE (the file's
content is the value), or --json FILE with all arguments as one JSON object (- reads standard
input; use it for text with quotes or line breaks).

Output: the result of the tool as one line of JSON on standard output. Exit code 0: success.
1: the tool returned an error (the message, on standard error, says what to do). 2: wrong
usage. 3: the background process or the server cannot be reached, the token is invalid, or
the connection broke and the outcome is unknown (check with chat_history before you send
again). When the background process is not running, ask the user to start it.

Keep wait_seconds of chat_inbox shorter than the time limit your shell puts on a command.
Run one agent-lounge-mcp command per shell call, not several joined with && or ;.

Tools (name, then what it does):
`

type streams struct {
	in       io.Reader
	out, err io.Writer
}

func (s streams) fail(code int, err error) int {
	fmt.Fprintln(s.err, err)
	return code
}

func (s streams) flags(command string) *flag.FlagSet {
	fs := flag.NewFlagSet("agent-lounge-mcp "+command, flag.ContinueOnError)
	fs.SetOutput(s.err)
	return fs
}

func parseCode(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	return 2
}

// Run implements the CLI commands and returns the shell exit code.
func Run(ctx context.Context, command string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	s := streams{stdin, stdout, stderr}
	if command == "connection" {
		return s.connection(ctx, args)
	}
	return s.toolsOrCall(ctx, command, args)
}

func (s streams) connection(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(s.err, Usage)
		return 2
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Fprint(s.out, Usage)
		return 0
	case "start":
		return s.start(ctx, args[1:])
	case "status", "stop", "list":
		fs := s.flags("connection " + args[0])
		name := connection.Name()
		if args[0] != "list" {
			fs.StringVar(&name, "name", name, "daemon name")
		}
		if err := fs.Parse(args[1:]); err != nil {
			return parseCode(err)
		}
		if fs.NArg() != 0 {
			return s.fail(2, errors.New("unexpected connection arguments"))
		}
		if args[0] == "list" {
			states, err := connection.List(ctx)
			if err != nil {
				return s.fail(1, err)
			}
			for _, state := range states {
				if err := json.NewEncoder(s.out).Encode(state); err != nil {
					return s.fail(1, err)
				}
			}
			return 0
		}
		if err := connection.CheckName(name); err != nil {
			return s.fail(2, err)
		}
		reply, err := connection.Exchange(ctx, name, connection.Request{Op: args[0]})
		if err != nil {
			return s.fail(1, err)
		}
		if reply.Error != "" {
			return s.fail(1, errors.New(reply.Error))
		}
		if reply.State != nil {
			if err := json.NewEncoder(s.out).Encode(reply.State); err != nil {
				return s.fail(1, err)
			}
		}
		return 0
	default:
		return s.fail(2, errors.New("use connection start, status, list or stop"))
	}
}

func readToken(in io.Reader) (string, error) {
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", errors.New("cannot read token")
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func (s streams) start(ctx context.Context, args []string) int {
	fs := s.flags("connection start")
	name := fs.String("name", connection.Name(), "daemon name")
	endpoint := fs.String("url", "", "Streamable HTTP MCP endpoint")
	detach := fs.Bool("detach", false, "start in the background")
	prompt := fs.Bool("token-prompt", false, "read the token from a terminal without echo")
	stdin := fs.Bool("token-stdin", false, "read the token from the first line of stdin")
	idle := fs.Duration("idle-timeout", 0, "stop after this long without a client")
	if err := fs.Parse(args); err != nil {
		return parseCode(err)
	}
	if fs.NArg() != 0 || *idle < 0 || (*prompt && *stdin) {
		return s.fail(2, errors.New("invalid connection start arguments"))
	}
	if err := connection.CheckName(*name); err != nil {
		return s.fail(2, err)
	}
	u, err := url.Parse(*endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return s.fail(2, errors.New("--url must be an HTTP or HTTPS MCP endpoint without user credentials"))
	}
	opt := connection.Options{Name: *name, URL: *endpoint, Idle: *idle}
	child := *detach && connection.IsChild()
	switch {
	case child || *stdin:
		opt.Token, err = readToken(s.in)
	case *prompt:
		f, ok := s.in.(*os.File)
		if !ok || !term.IsTerminal(int(f.Fd())) {
			return s.fail(1, errors.New("--token-prompt requires a terminal on stdin"))
		}
		fmt.Fprint(s.err, "Token: ")
		var b []byte
		b, err = term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(s.err)
		opt.Token = string(b)
	}
	if err != nil {
		return s.fail(1, errors.New("cannot read token"))
	}
	if *detach && !child {
		state, err := connection.Detach(ctx, opt, args)
		if ctx.Err() != nil {
			return 130
		}
		if err != nil {
			return s.fail(1, err)
		}
		if err := json.NewEncoder(s.out).Encode(state); err != nil {
			return s.fail(1, err)
		}
		return 0
	}
	var ready func(*connection.State)
	if child {
		ready = func(state *connection.State) {
			_ = json.NewEncoder(s.out).Encode(connection.Reply{State: state})
		}
	}
	if err := connection.Serve(ctx, opt, ready); err != nil {
		if opt.Token != "" {
			err = errors.New(strings.ReplaceAll(err.Error(), opt.Token, "[redacted]"))
		}
		if child {
			_ = json.NewEncoder(s.out).Encode(connection.Reply{Code: 1, Error: err.Error()})
		}
		return s.fail(1, err)
	}
	return 0
}

func (s streams) toolsOrCall(ctx context.Context, command string, args []string) int {
	fs := s.flags(command)
	name := fs.String("connection", connection.Name(), "daemon name")
	var tool, as, file string
	var noProfile bool
	if command == "tools" {
		fs.StringVar(&tool, "schema", "", "show a tool's description and input schema")
	} else {
		fs.StringVar(&as, "as", os.Getenv("AGENT_LOUNGE_AS"), "participant name")
		fs.StringVar(&file, "json", "", "read arguments from a JSON file, or - for stdin")
		// The lounge server sends neither progress nor log notifications.
		fs.Bool("quiet", false, "suppress progress messages")
		fs.BoolVar(&noProfile, "no-profile", false, "skip collecting the join profile")
	}
	// Agents write the options before or after the tool name: accept both.
	if err := fs.Parse(optionsFirst(fs, args)); err != nil {
		return parseCode(err)
	}
	if err := connection.CheckName(*name); err != nil {
		return s.fail(2, err)
	}
	pairs := fs.Args()
	if command == "tools" && len(pairs) != 0 {
		return s.fail(2, errors.New("unexpected tools arguments"))
	}
	if command == "call" {
		if len(pairs) == 0 {
			return s.fail(2, errors.New("call requires a tool name"))
		}
		tool, pairs = pairs[0], pairs[1:]
		if tool != "lounge_help" {
			if as == "" {
				return s.fail(2, errors.New("specify a participant with --as NAME or set AGENT_LOUNGE_AS"))
			}
			var err error
			as, err = names.Session(as)
			if err != nil {
				return s.fail(2, err)
			}
		}
	}

	reply, code := s.exchange(ctx, *name, connection.Request{Op: "tools"})
	if code != 0 {
		return code
	}
	if command == "tools" && tool == "" {
		fmt.Fprint(s.out, guide)
		for _, t := range reply.Tools {
			fmt.Fprintf(s.out, "%s\t%s\n", t.Name, firstSentence(t.Description))
		}
		return 0
	}
	var definition *mcp.Tool
	for _, t := range reply.Tools {
		if t.Name == tool {
			definition = t
			break
		}
	}
	if definition == nil {
		return s.fail(2, fmt.Errorf("unknown tool %q; run agent-lounge-mcp tools", tool))
	}
	if command == "tools" {
		return s.json(map[string]any{"description": definition.Description, "inputSchema": definition.InputSchema})
	}
	argsJSON, err := s.arguments(ctx, definition, as, file, pairs, noProfile)
	if ctx.Err() != nil {
		return 130
	}
	if err != nil {
		return s.fail(2, err)
	}
	reply, code = s.exchange(ctx, *name, connection.Request{Op: "call", Tool: tool, As: as, Args: argsJSON})
	if code != 0 {
		return code
	}
	return s.result(reply.Result)
}

// optionsFirst moves the options of fs, with their values, in front of the other arguments,
// so that the flag package also parses the ones written after the tool name. The other
// arguments are a tool name and KEY=VALUE pairs, which never start with a hyphen. Everything
// after -- is left as it is.
func optionsFirst(fs *flag.FlagSet, args []string) []string {
	var options, rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			rest = append(rest, arg)
			continue
		}
		options = append(options, arg)
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		f := fs.Lookup(name)
		if f == nil || hasValue {
			continue // an unknown option is reported by the flag package
		}
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			continue
		}
		if i+1 < len(args) {
			i++
			options = append(options, args[i])
		}
	}
	return append(append(options, "--"), rest...)
}

// result prints what the tool returned, as an agent connected over MCP receives it: the
// structured result as one line of JSON, or, for a tool that has no structured result
// (lounge_help returns only text), the text.
func (s streams) result(raw json.RawMessage) int {
	var r struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Structured json.RawMessage `json:"structuredContent"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return s.fail(1, err)
	}
	var line bytes.Buffer
	if len(r.Structured) > 0 {
		if err := json.Compact(&line, r.Structured); err != nil {
			return s.fail(1, err)
		}
	}
	// A tool that returns only text has no structured result; an empty one counts as none.
	if structured := line.String(); structured != "" && structured != "null" && structured != "{}" {
		fmt.Fprintln(s.out, structured)
		return 0
	}
	for _, c := range r.Content {
		if c.Type == "text" {
			fmt.Fprintln(s.out, strings.TrimRight(c.Text, "\n"))
		}
	}
	return 0
}

func (s streams) json(value any) int {
	if err := json.NewEncoder(s.out).Encode(value); err != nil {
		return s.fail(1, err)
	}
	return 0
}

func (s streams) exchange(ctx context.Context, name string, req connection.Request) (connection.Reply, int) {
	reply, err := connection.Exchange(ctx, name, req)
	if ctx.Err() != nil {
		return reply, 130
	}
	if err != nil {
		return reply, s.fail(3, err)
	}
	if reply.Code != 0 {
		return reply, s.fail(reply.Code, errors.New(reply.Error))
	}
	return reply, 0
}

func (s streams) arguments(ctx context.Context, tool *mcp.Tool, as, file string, pairs []string, noProfile bool) (json.RawMessage, error) {
	shape, err := parseSchema(tool.InputSchema)
	if err != nil {
		return nil, err
	}
	withKeys := func(err error) error {
		return fmt.Errorf("%w; usable keys: %s", err, strings.Join(shape.keys(), ", "))
	}
	var base map[string]any
	if file != "" {
		base, err = readArgsFile(file, func() ([]byte, error) { return io.ReadAll(s.in) })
		if err != nil {
			return nil, withKeys(err)
		}
	}
	args, err := buildArgs(shape, base, pairs, os.ReadFile)
	if err != nil {
		return nil, withKeys(err)
	}
	for key := range args {
		if shape.Properties[key] == nil {
			return nil, withKeys(fmt.Errorf("unknown argument %q", key))
		}
	}
	if tool.Name == "lounge_join" {
		if _, exists := args["name"]; !exists {
			args["name"] = as
		}
		if !noProfile {
			collectProfile(ctx, args)
		}
	}
	rawSchema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return nil, err
	}
	var full jsonschema.Schema
	if err := json.Unmarshal(rawSchema, &full); err != nil {
		return nil, err
	}
	resolved, err := full.Resolve(nil)
	if err != nil {
		return nil, err
	}
	if err := resolved.Validate(args); err != nil {
		return nil, withKeys(err)
	}
	b, err := json.Marshal(args)
	if err != nil {
		return nil, withKeys(err)
	}
	return b, nil
}

func firstSentence(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}
