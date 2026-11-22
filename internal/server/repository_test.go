package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/testutil"
)

type profileJoined struct {
	Name       string `json:"name"`
	OSUser     string `json:"os_user"`
	Machine    string `json:"machine"`
	Origin     string `json:"origin"`
	Folder     string `json:"folder"`
	Repository string `json:"repository"`
}

type profileWho struct {
	Repository string `json:"repository"`
	Sessions   []struct {
		Name       string `json:"name"`
		OSUser     string `json:"os_user"`
		Machine    string `json:"machine"`
		Repository string `json:"repository"`
		Scopes     []struct {
			Scope string `json:"scope"`
			ID    string `json:"id"`
		} `json:"scopes"`
		You bool `json:"you"`
	} `json:"sessions"`
}

type scopedSent struct {
	Recipients []string `json:"recipients"`
	Skipped    []string `json:"skipped"`
	Scope      string   `json:"scope"`
	ScopeID    string   `json:"scope_id"`
}

type scopedInbox struct {
	Messages []struct {
		From        string `json:"from"`
		FromOSUser  string `json:"from_os_user"`
		FromMachine string `json:"from_machine"`
		FromUser    string `json:"from_user"`
		Scope       string `json:"scope"`
		ScopeID     string `json:"scope_id"`
		ReplyTo     string `json:"reply_to"`
		Room        string `json:"room"`
		Body        string `json:"body"`
	} `json:"messages"`
	WaitedSeconds *int `json:"waited_seconds"`
}

func TestJoinProfile(t *testing.T) {
	s, sess := newServer(t)
	cs := testutil.Connect(t, s.MCP, nil)

	// Values are passed as the commands print them and come back normalized.
	j := testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{
		"name": "alice", "os_user": "Honda\r\n", "machine": "HOST-A\r\n",
		"origin": "https://user:SECRET@GitHub.com/Owner/Repo.git\n", "folder": "/mnt/c/Work/Repo", "wsl_distro": "Ubuntu\n",
	})
	want := profileJoined{Name: "alice", OSUser: "Honda", Machine: "host-a", Origin: "github.com/owner/repo",
		Folder: `c:\work\repo`, Repository: "github.com/owner/repo"}
	if j != want {
		t.Fatalf("join = %+v, want %+v", j, want)
	}
	// The credentialed raw origin is stored nowhere.
	sessions, _, err := sess.Store.ListSessions(context.Background(), sess.LocalUser.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := json.Marshal(sessions); strings.Contains(string(b), "SECRET") {
		t.Errorf("stored session contains the credentials: %s", b)
	}

	// Omitted values are kept.
	j = testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "alice"})
	if j != want {
		t.Errorf("rejoin without values = %+v, want %+v", j, want)
	}

	// origin and folder are one set: a folder alone clears the origin, and the repository
	// becomes machine:folder.
	j = testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "alice", "folder": `C:\Scratch\`})
	if j.Origin != "" || j.Folder != `c:\scratch` || j.Repository != `host-a:c:\scratch` || j.OSUser != "Honda" {
		t.Errorf("after moving to a folder without origin: %+v", j)
	}
	// An origin alone clears the folder.
	j = testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "alice", "origin": "git@github.com:owner/other.git"})
	if j.Origin != "github.com/owner/other" || j.Folder != "" || j.Repository != "github.com/owner/other" {
		t.Errorf("after moving to another origin: %+v", j)
	}
	// A hostless origin is no origin.
	j = testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "alice", "origin": "/srv/git/repo.git", "folder": "/home/u/repo"})
	if j.Origin != "" || j.Folder != "/home/u/repo" || j.Repository != "host-a:/home/u/repo" {
		t.Errorf("hostless origin: %+v", j)
	}
	// wsl_distro only converts folder: without folder it is refused and changes nothing.
	if msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": "alice", "wsl_distro": "Ubuntu"}); !strings.Contains(msg, "wsl_distro") {
		t.Errorf("wsl_distro alone: %q", msg)
	}
	if msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": "alice", "origin": "", "wsl_distro": "Ubuntu"}); !strings.Contains(msg, "wsl_distro") {
		t.Errorf("wsl_distro with origin only: %q", msg)
	}
	j = testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "alice"})
	if j.Folder != "/home/u/repo" || j.Repository != "host-a:/home/u/repo" {
		t.Errorf("after refused wsl_distro: %+v", j)
	}
	// A value the shell did not expand is refused.
	if msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": "alice", "os_user": "%USERNAME%"}); !strings.Contains(msg, "did not expand") {
		t.Errorf("unexpanded os_user: %q", msg)
	}
	// Empty values clear.
	j = testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "alice", "os_user": "", "machine": "", "origin": "", "folder": ""})
	if j != (profileJoined{Name: "alice"}) {
		t.Errorf("after clearing: %+v", j)
	}
}

func TestJoinProfileErrors(t *testing.T) {
	cs := connect(t)
	for _, tc := range []struct {
		args map[string]any
		want string
	}{
		{map[string]any{"name": "a", "folder": "/home/u/repo"}, "machine"},
		{map[string]any{"name": "a", "machine": "h", "folder": "work/repo"}, "absolute"},
		{map[string]any{"name": "a", "origin": "ftp://example.org/repo"}, "scheme"},
		{map[string]any{"name": "a", "origin": "https://user:SECRET@github.com"}, "origin"},
		{map[string]any{"name": "a", "os_user": strings.Repeat("u", 129)}, "too long"},
		{map[string]any{"name": "a", "machine": strings.Repeat("m", 129)}, "too long"},
		{map[string]any{"name": "a", "machine": "h", "folder": "/" + strings.Repeat("f", 1024)}, "too long"},
	} {
		msg := testutil.CallErr(t, cs, "lounge_join", tc.args)
		if !strings.Contains(msg, tc.want) {
			t.Errorf("%v: error %q does not mention %q", tc.args, msg, tc.want)
		}
		if strings.Contains(msg, "SECRET") {
			t.Errorf("%v: error quotes the credentials: %q", tc.args, msg)
		}
	}
	// A failed join does not join.
	if msg := testutil.CallErr(t, cs, "lounge_who", nil); !strings.Contains(msg, "not joined") {
		t.Errorf("after failed joins: %q", msg)
	}
	// A recorded folder without origin cannot lose its machine.
	testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "a", "machine": "h", "folder": "/x"})
	if msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": "a", "machine": ""}); !strings.Contains(msg, "machine") {
		t.Errorf("clearing the machine: %q", msg)
	}
}

// A machine name that does not tell machines apart is rejected, and the recorded profile stays.
func TestJoinRejectsGenericMachine(t *testing.T) {
	cs := connect(t)
	for _, m := range []string{"sandbox", "Sandbox", "LOCALHOST", "localhost.localdomain"} {
		msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": "a", "machine": m})
		if !strings.Contains(msg, "does not identify a machine") || !strings.Contains(msg, "cat /etc/hostname") {
			t.Errorf("machine %q: error %q", m, msg)
		}
	}
	// A rejected value does not join.
	if msg := testutil.CallErr(t, cs, "lounge_who", nil); !strings.Contains(msg, "not joined") {
		t.Errorf("after rejected joins: %q", msg)
	}
	// A rejected value leaves the recorded profile unchanged.
	want := testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "a", "os_user": "Honda", "machine": "HOST-A", "folder": "/x"})
	if msg := testutil.CallErr(t, cs, "lounge_join", map[string]any{"name": "a", "os_user": "Other", "machine": "sandbox", "folder": "/y"}); !strings.Contains(msg, "does not identify") {
		t.Errorf("rejoin: %q", msg)
	}
	if got := testutil.Call[profileJoined](t, cs, "lounge_join", map[string]any{"name": "a"}); got != want {
		t.Errorf("profile after rejected join = %+v, want %+v", got, want)
	}
}

// Sessions of one repository, declared in different forms, share the repository room; sessions
// of another repository and sessions without one do not receive its messages.
func TestRepositoryRoom(t *testing.T) {
	s, _ := waitServer(t, 10*time.Second)
	join := func(args map[string]any) *mcp.ClientSession {
		cs := testutil.Connect(t, s.MCP, nil)
		testutil.Call[profileJoined](t, cs, "lounge_join", args)
		return cs
	}
	alice := join(map[string]any{"name": "alice", "os_user": "honda", "machine": "host-a", "origin": "https://github.com/Org/A.git"})
	bob := join(map[string]any{"name": "bob", "origin": "git@github.com:org/a.git"})
	carol := join(map[string]any{"name": "carol", "origin": "git@github.com:org/b.git"})
	dave := join(map[string]any{"name": "dave"})
	// Without origin: the same folder from WSL and from Windows on one machine is one unit.
	wsl := join(map[string]any{"name": "wsl", "machine": "PC", "folder": "/home/u/app", "wsl_distro": "Ubuntu"})
	win := join(map[string]any{"name": "win", "machine": "pc", "folder": `\\wsl$\Ubuntu\home\u\app`})
	other := join(map[string]any{"name": "other", "machine": "pc2", "folder": `\\wsl$\Ubuntu\home\u\app`})

	w := testutil.Call[profileWho](t, alice, "lounge_who", nil)
	if w.Repository != "github.com/org/a" {
		t.Errorf("who repository = %q", w.Repository)
	}
	same := map[string]bool{}
	for _, sess := range w.Sessions {
		// The repository scope is shared exactly with the sessions of alice's repository.
		same[sess.Name] = false
		for _, sc := range sess.Scopes {
			if sc.Scope == "repository" && sc.ID == "github.com/org/a" {
				same[sess.Name] = true
			}
		}
		if sess.Name == "alice" && (sess.OSUser != "honda" || sess.Machine != "host-a" || !sess.You) {
			t.Errorf("alice in who = %+v", sess)
		}
	}
	if want := map[string]bool{"alice": true, "bob": true, "carol": false, "dave": false, "wsl": false, "win": false, "other": false}; !mapsEqual(same, want) {
		t.Errorf("shares the repository scope = %v, want %v", same, want)
	}

	// bob waits; alice posts to the repository room and the wait ends with the message.
	ch := callAsync(context.Background(), bob, "chat_inbox", map[string]any{"wait_seconds": 10})
	waitingCalls(t, s, 1)
	out := testutil.Call[scopedSent](t, alice, "chat_send", map[string]any{"room": "lounge", "scope": "repository", "body": "repo a"})
	if !slices.Equal(out.Recipients, []string{"bob"}) || out.Scope != "repository" || out.ScopeID != "github.com/org/a" {
		t.Errorf("sent = %+v", out)
	}
	got := decode[scopedInbox](t, receive(t, ch))
	if len(got.Messages) != 1 || got.WaitedSeconds == nil {
		t.Fatalf("bob's inbox = %+v", got)
	}
	m := got.Messages[0]
	if m.Scope != "repository" || m.ScopeID != "github.com/org/a" || m.Room != "lounge" || m.From != "alice" || m.ReplyTo != "alice" ||
		m.FromOSUser != "honda" || m.FromMachine != "host-a" || m.Body != "repo a" {
		t.Errorf("bob's message = %+v", m)
	}
	for _, cs := range []*mcp.ClientSession{carol, dave, wsl, win, other} {
		if in := testutil.Call[scopedInbox](t, cs, "chat_inbox", nil); len(in.Messages) != 0 {
			t.Errorf("a session outside the repository received %+v", in.Messages)
		}
	}

	// The folder unit: wsl reaches win only.
	out = testutil.Call[scopedSent](t, wsl, "chat_send", map[string]any{"room": "lounge", "scope": "repository", "body": "folder"})
	if !slices.Equal(out.Recipients, []string{"win"}) || out.ScopeID != `pc:\\wsl.localhost\ubuntu\home\u\app` {
		t.Errorf("folder unit: %+v", out)
	}
	// Messages without scope or with scope user are unchanged; their scope id is the user.
	out = testutil.Call[scopedSent](t, alice, "chat_send", map[string]any{"room": "lounge", "body": "user room"})
	if out.Scope != "user" || out.ScopeID != "local" || len(out.Recipients) != 6 {
		t.Errorf("user room: %+v", out)
	}
	res, err := carol.CallTool(context.Background(), &mcp.CallToolParams{Name: "chat_inbox"})
	if err != nil {
		t.Fatal(err)
	}
	if text := testutil.Text(res); !strings.Contains(text, `"scope":"user"`) || !strings.Contains(text, `"scope_id":"local"`) {
		t.Errorf("user message text: %s", text)
	}
	out = testutil.Call[scopedSent](t, alice, "chat_send", map[string]any{"room": "lounge", "scope": "user", "body": "user room"})
	if out.Scope != "user" {
		t.Errorf("scope user: %+v", out)
	}
}

func TestRepositorySendErrors(t *testing.T) {
	cs := clients(t, "alice", "bob")
	for _, tc := range []struct {
		args map[string]any
		want string
	}{
		{map[string]any{"room": "lounge", "scope": "repository", "body": "x"}, "lounge_join"},
		{map[string]any{"to": "bob", "scope": "repository", "body": "x"}, "lounge_join"},
		{map[string]any{"room": "lounge", "scope": "shared", "body": "x"}, "invalid scope"},
	} {
		if msg := testutil.CallErr(t, cs[0], "chat_send", tc.args); !strings.Contains(msg, tc.want) {
			t.Errorf("%v: error %q does not mention %q", tc.args, msg, tc.want)
		}
	}
	testutil.Call[profileJoined](t, cs[0], "lounge_join", map[string]any{"name": "alice", "origin": "github.com/o/r"})
	if msg := testutil.CallErr(t, cs[0], "chat_send", map[string]any{"room": "other", "scope": "repository", "body": "x"}); !strings.Contains(msg, "no room") {
		t.Errorf("unknown room: %q", msg)
	}
	// Alone in the repository: accepted with no recipients.
	out := testutil.Call[scopedSent](t, cs[0], "chat_send", map[string]any{"room": "lounge", "scope": "repository", "body": "x"})
	if out.Recipients == nil || len(out.Recipients) != 0 {
		t.Errorf("recipients = %#v, want an empty list", out.Recipients)
	}
}

// /healthz reports the number of chat_inbox calls that wait now.
func TestHealthReportsWaiting(t *testing.T) {
	cfg := config.Default()
	s, sess := newServerWith(t, cfg)
	hs := httptest.NewServer(Handler(s, sess))
	t.Cleanup(hs.Close)
	waiting := func() int {
		res, err := http.Get(hs.URL + HealthPath)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var body struct {
			Waiting *int `json:"waiting"`
		}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil || body.Waiting == nil {
			t.Fatalf("health without waiting: %v", err)
		}
		return *body.Waiting
	}
	if n := waiting(); n != 0 {
		t.Fatalf("waiting = %d before any call", n)
	}
	cs := testutil.ConnectHTTP(t, hs.URL+MCPPath)
	testutil.Call[joined](t, cs, "lounge_join", map[string]any{"name": "bob"})
	ctx, cancel := context.WithCancel(context.Background())
	ch := callAsync(ctx, cs, "chat_inbox", map[string]any{"wait_seconds": 30})
	testutil.WaitFor(t, func() bool { return waiting() == 1 }, "/healthz does not report the waiting call")
	cancel()
	<-ch
	testutil.WaitFor(t, func() bool { return waiting() == 0 }, "/healthz still reports a waiting call")
}

func TestHelpDescribesProfileAndRepository(t *testing.T) {
	cs := connect(t)
	guide := guideOf(t, cs)
	for _, want := range []string{
		"## Profile", "## Scopes and targets", "## Users", "os_user", "machine", "origin", "folder", "wsl_distro",
		"id -un", "uname -n", "you are on WSL", "$env:USERNAME", "echo $USERNAME", `cmd.exe /c "echo %USERNAME%"`, "cmd.exe /c hostname", "git remote get-url origin",
		"cat /etc/hostname", "pwd -W", "git rev-parse --show-toplevel", "echo $WSL_DISTRO_NAME", "scope repository", "credentials",
		"before you call other tools", "not verified", "not carried over", "/mnt/<drive letter>",
	} {
		if !strings.Contains(guide, want) {
			t.Errorf("guide does not mention %q", want)
		}
	}
	_, snippet, _ := strings.Cut(guide, "## Snippet for instruction files")
	if !strings.Contains(snippet, "Call it again with the same name when the repository, folder or branch you work in changes") {
		t.Errorf("snippet does not tell to join again when the folder or branch changes:\n%s", snippet)
	}
	if !strings.Contains(cs.InitializeResult().Instructions, "profile") {
		t.Error("instructions do not mention the profile")
	}
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "lounge_join" && !strings.Contains(tool.Description, "wait for its result before calling other tools") {
			t.Errorf("lounge_join description does not tell to wait for its result: %s", tool.Description)
		}
		if tool.Name == "chat_send" && !strings.Contains(tool.Description, "repository") {
			t.Errorf("chat_send description does not mention the repository scope: %s", tool.Description)
		}
	}
}

func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// A Linux path as folder gets the hint to check for WSL; a location without branch gets the
// hint to declare branch when the session has a repository.
func TestJoinHint(t *testing.T) {
	s, _ := newServer(t)
	cs := testutil.Connect(t, s.MCP, nil)
	type hinted struct {
		Hint string `json:"hint"`
	}
	j := testutil.Call[hinted](t, cs, "lounge_join", map[string]any{"name": "a", "machine": "m", "folder": "/home/u/app"})
	if !strings.Contains(j.Hint, "WSL_DISTRO_NAME") || !strings.Contains(j.Hint, "git branch --show-current") {
		t.Errorf("Linux path without branch: hint %q", j.Hint)
	}
	j = testutil.Call[hinted](t, cs, "lounge_join", map[string]any{"name": "a", "machine": "m", "folder": `C:\work`})
	if strings.Contains(j.Hint, "WSL_DISTRO_NAME") || !strings.Contains(j.Hint, "git branch --show-current") {
		t.Errorf("Windows path without branch: hint %q", j.Hint)
	}
	for _, args := range []map[string]any{
		{"name": "a", "machine": "m", "folder": "/home/u/app", "wsl_distro": "Ubuntu", "branch": "main"},
		{"name": "a", "machine": "m", "folder": `C:\work`, "branch": ""},
		{"name": "a", "origin": "", "folder": ""},
		{"name": "a"},
	} {
		if j := testutil.Call[hinted](t, cs, "lounge_join", args); j.Hint != "" {
			t.Errorf("%v: hint %q", args, j.Hint)
		}
	}
}
