package tools

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/profile"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

const (
	maxAgentLen     = 128
	maxSessionsList = 200
	// maxLocationLen bounds origin, folder and wsl_distro, in characters, before normalization.
	maxLocationLen = 1024
)

// sessionNameRule describes sessionNameRE to agents.
const sessionNameRule = "1 to 64 characters from A-Z a-z 0-9 . _ - @"

// sessionNameRE is the form of a session name.
var sessionNameRE = regexp.MustCompile(`^[A-Za-z0-9._@-]{1,64}$`)

func registerLounge(r *Registry, d *Deps) {
	add(r, groupChat, &mcp.Tool{
		Name:  toolJoin,
		Title: "Join the lounge",
		Description: "Join the lounge under a session name. Call this first, and wait for its result before calling other tools " +
			"(do not call them in parallel with it): the chat tools work only after joining. " +
			"Other sessions address messages to this name. Names are unique within your user, which the token your client sends decides (the result returns it as user). " +
			"Joining with a name used before continues that session of your user and its unread messages " +
			"(if another connection is using the name, it loses the session). After a reconnect, join again with the same name. " +
			"Pass your profile (os_user, machine, origin, folder, wsl_distro, branch) as the output of the commands that lounge_help lists; " +
			"the server normalizes the values, decides your repository, and returns them. " +
			"When the repository, folder or branch you work in changes, call it again with the same name. " +
			"After joining, call chat_inbox when a result reports unread messages, after each step of your task that changes files, " +
			"and before your final reply to the user: nothing notifies you of new messages. " +
			"If you do not know how to use the lounge, call lounge_help first.",
		Annotations: additive(true),
	}, d.loungeJoin)

	add(r, groupChat, &mcp.Tool{
		Name:  toolWho,
		Title: "List sessions and rooms",
		Description: "List the sessions you can reach: the sessions of your user, and the sessions of other users that declared your repository. " +
			"For each: name, user, address, agent, OS user, machine, repository, branch, the scopes you share with it (scope and id), whether connected now, last activity. " +
			"Also returns your scopes and the rooms. To send to a session, pass its address as the 'to' of chat_send with one of its scopes.",
		Annotations: readOnly(),
	}, d.loungeWho)
}

// JoinInput defines arguments for the lounge_join tool.
type JoinInput struct {
	Name  string  `json:"name,omitempty" jsonschema:"session name: 1 to 64 characters from A-Z a-z 0-9 . _ - @ (for example claude or review-task). Omit to let the server choose one"`
	Agent *string `json:"agent,omitempty" jsonschema:"what you are, shown to other sessions, for example claude-code@host-a (up to 128 characters). Omit to keep the recorded value"`
	// Profile. agent, os_user and machine: omit to keep the recorded value, "" to clear it.
	// origin, folder and wsl_distro: see joinParams.
	OSUser    *string `json:"os_user,omitempty" jsonschema:"OS user name running you, as printed by the command lounge_help lists for your environment (on WSL and Windows, the Windows user name). Omit to keep the recorded value, empty to clear it"`
	Machine   *string `json:"machine,omitempty" jsonschema:"machine name, as printed by the command lounge_help lists (on WSL, cmd.exe /c hostname: the Windows computer name of the host; if cmd.exe fails or is not found, cat /etc/hostname). Omit to keep the recorded value, empty to clear it"`
	Origin    *string `json:"origin,omitempty" jsonschema:"URL of the Git origin of your working folder, as printed by git remote get-url origin, without credentials; empty when there is none. origin and folder are replaced together: omit origin, folder and wsl_distro to keep the recorded ones"`
	Folder    *string `json:"folder,omitempty" jsonschema:"top of the Git working tree (git rev-parse --show-toplevel), or the working folder outside Git (pwd). origin and folder are replaced together"`
	WSLDistro *string `json:"wsl_distro,omitempty" jsonschema:"on WSL (echo $WSL_DISTRO_NAME prints a name), that distribution name; omit elsewhere. Used only to convert folder, not stored"`
	Branch    *string `json:"branch,omitempty" jsonschema:"Git branch name, as printed by git branch --show-current (empty when not on a branch or outside Git). Passing origin, folder or wsl_distro without branch clears the recorded branch; passing branch alone replaces only branch"`
}

// JoinOutput is the result of the lounge_join tool.
type JoinOutput struct {
	Name    string   `json:"name" jsonschema:"the session name you joined as"`
	Agent   string   `json:"agent" jsonschema:"recorded agent name"`
	User    string   `json:"user" jsonschema:"user the session belongs to, decided by the token your client sends (local without a token)"`
	Resumed bool     `json:"resumed" jsonschema:"true when an existing session was continued"`
	Rooms   []string `json:"rooms" jsonschema:"rooms the session is a member of"`
	// Profile as recorded.
	OSUser     string `json:"os_user" jsonschema:"recorded OS user name"`
	Machine    string `json:"machine" jsonschema:"recorded machine name"`
	Origin     string `json:"origin" jsonschema:"recorded origin, normalized to host/path (empty for none)"`
	Folder     string `json:"folder" jsonschema:"recorded folder, normalized (empty for none)"`
	Repository string `json:"repository" jsonschema:"your repository, decided by the server: the origin, else machine:folder, else empty (the repository space is not available)"`
	Branch     string `json:"branch" jsonschema:"recorded Git branch name, normalized lowercase (empty for none)"`
	// Hint is set when the declared values suggest a step of the usage guide was skipped.
	Hint string `json:"hint,omitempty" jsonschema:"what to check about the declared profile, when something may be missing"`
	unreadField
}

// joinParams validates and normalizes the arguments of lounge_join.
//
// origin and folder are one set: when origin, folder or wsl_distro is given, the location is
// replaced, and a missing origin or folder counts as empty. When all three are omitted, the
// recorded location is kept.
//
// When a call replaces the location and omits branch, store.JoinSession clears the recorded
// branch: moving to another repository must not carry over the previous repository's branch
// name and place the session in a same-named branch of the new repository. Passing branch
// without location replaces only branch.
func joinParams(userID int64, in JoinInput) (store.JoinParams, error) {
	p := store.JoinParams{UserID: userID, Name: in.Name, Agent: in.Agent}
	if in.Name != "" && !sessionNameRE.MatchString(in.Name) {
		return p, fmt.Errorf("invalid name %q: use %s", in.Name, sessionNameRule)
	}
	if in.Agent != nil && utf8.RuneCountInString(*in.Agent) > maxAgentLen {
		return p, fmt.Errorf("agent is too long: %d characters, at most %d", utf8.RuneCountInString(*in.Agent), maxAgentLen)
	}
	var err error
	if p.OSUser, err = normalized(in.OSUser, profile.OSUser); err != nil {
		return p, err
	}
	if p.Machine, err = normalized(in.Machine, profile.Machine); err != nil {
		return p, err
	}
	if p.Branch, err = normalized(in.Branch, profile.Branch); err != nil {
		return p, err
	}
	if in.Origin == nil && in.Folder == nil && in.WSLDistro == nil {
		return p, nil
	}
	// wsl_distro only converts folder. Alone it would replace the location with an empty one,
	// which an agent that thinks of it as a property of the machine does not expect.
	if in.WSLDistro != nil && strings.TrimSpace(*in.WSLDistro) != "" && (in.Folder == nil || strings.TrimSpace(*in.Folder) == "") {
		return p, errors.New("wsl_distro is used only to convert folder: pass folder (and origin) with it, or omit wsl_distro")
	}
	for field, v := range map[string]*string{"origin": in.Origin, "folder": in.Folder, "wsl_distro": in.WSLDistro} {
		if v != nil && utf8.RuneCountInString(*v) > maxLocationLen {
			return p, fmt.Errorf("%s is too long: at most %d characters", field, maxLocationLen)
		}
	}
	var loc store.Location
	if in.Origin != nil {
		// The raw value may contain credentials: only the normalized value is kept.
		if loc.Origin, err = profile.Origin(*in.Origin); err != nil {
			return p, fmt.Errorf("%w. Pass the output of git remote get-url origin without credentials, or empty when there is no origin", err)
		}
	}
	if in.Folder != nil {
		distro := ""
		if in.WSLDistro != nil {
			distro = *in.WSLDistro
		}
		if loc.Folder, err = profile.Folder(*in.Folder, distro); err != nil {
			return p, err
		}
	}
	p.Location = &loc
	return p, nil
}

// wslHint asks an agent that declared a Linux path to check that it is not on WSL. The server
// cannot tell plain Linux from WSL, and agents on WSL tend to take the plain Linux commands.
const wslHint = "folder is a Linux or macOS path. If echo $WSL_DISTRO_NAME prints a name, you are on WSL: " +
	"call " + toolJoin + " again with the same name, wsl_distro (that name), folder, origin, branch, " +
	"os_user (cmd.exe /c \"echo %USERNAME%\") and machine (cmd.exe /c hostname), " +
	"so that Windows and WSL sessions on one machine get the same values. If it prints nothing, ignore this hint."

// branchHint asks an agent that declared a location without branch to pass git branch --show-current.
const branchHint = "branch was not passed. If you are in a Git working tree, run git branch --show-current and call " +
	toolJoin + " again with the same name and branch (that output, or an empty string when not on a branch) to use the branch scope."

// joinHint returns the hint for the recorded profile, or "".
func joinHint(sess store.Session, locationGiven, branchGiven bool) string {
	var hints []string
	if strings.HasPrefix(sess.Folder, "/") {
		hints = append(hints, wslHint)
	}
	if locationGiven && !branchGiven && sess.Repository != "" {
		hints = append(hints, branchHint)
	}
	return strings.Join(hints, " ")
}

// normalized applies normalize to an optional value; nil stays nil (keep the recorded value).
func normalized(v *string, normalize func(string) (string, error)) (*string, error) {
	if v == nil {
		return nil, nil
	}
	s, err := normalize(*v)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Deps) loungeJoin(ctx context.Context, req *mcp.CallToolRequest, in JoinInput) (*mcp.CallToolResult, JoinOutput, error) {
	user := d.RequestUser(req)
	p, err := joinParams(user.ID, in)
	if err != nil {
		return nil, JoinOutput{}, err
	}
	sess, resumed, err := d.Store.JoinSession(ctx, p)
	switch {
	case errors.Is(err, store.ErrMachineRequired):
		return nil, JoinOutput{}, errors.New("folder without origin needs machine: pass machine (on WSL the output of cmd.exe /c hostname, or of cat /etc/hostname if cmd.exe fails or is not found; on Windows of hostname, on Linux and macOS of uname -n), " +
			"so that the same path on another machine is not taken for the same repository")
	case errors.Is(err, store.ErrTooManySessions):
		return nil, JoinOutput{}, fmt.Errorf("your user has %d sessions, the most allowed: join with the name of an existing session (%s lists them)",
			d.Cfg.MaxSessionsPerUser, toolWho)
	case err != nil:
		return nil, JoinOutput{}, err
	}
	rooms, err := d.Store.ListRooms(ctx, user.ID)
	if err != nil {
		return nil, JoinOutput{}, err
	}
	d.notify.wake(d.presence.bind(req.Session, sess)...)
	return nil, JoinOutput{
		Name: sess.Name, Agent: sess.Agent, User: sess.UserName, Resumed: resumed, Rooms: rooms,
		OSUser: sess.OSUser, Machine: sess.Machine, Origin: sess.Origin, Folder: sess.Folder, Repository: sess.Repository, Branch: sess.Branch,
		Hint: joinHint(sess, p.Location != nil, in.Branch != nil),
	}, nil
}

// WhoInput defines arguments for the lounge_who tool.
type WhoInput struct{}

// ScopeOutput is a scope in tool output: its kind and identifier.
type ScopeOutput struct {
	Scope string `json:"scope" jsonschema:"kind of scope: user (the sessions of one user), repository (the sessions of one repository, whatever their user) or branch (the sessions of one repository and branch, whatever their user)"`
	ID    string `json:"id" jsonschema:"identifier of the scope: the user name for user, the repository for repository, repository:branch for branch"`
}

func scopesOutput(scopes []store.Scope) []ScopeOutput {
	out := make([]ScopeOutput, 0, len(scopes))
	for _, sc := range scopes {
		out = append(out, ScopeOutput{Scope: sc.Kind, ID: sc.ID})
	}
	return out
}

// WhoSession is one session in the result of lounge_who.
type WhoSession struct {
	Name       string `json:"name" jsonschema:"session name"`
	User       string `json:"user" jsonschema:"user the session belongs to"`
	Address    string `json:"address" jsonschema:"value to pass as the 'to' of chat_send to reach this session: its name for a session of your user, user/name for a session of another user"`
	Agent      string `json:"agent" jsonschema:"agent name the session declared"`
	OSUser     string `json:"os_user" jsonschema:"OS user name the session declared"`
	Machine    string `json:"machine" jsonschema:"machine name the session declared"`
	Repository string `json:"repository" jsonschema:"repository of the session (empty for none)"`
	Branch     string `json:"branch" jsonschema:"branch of the session (empty for none)"`
	// Scopes are those in which the caller reaches the session.
	Scopes     []ScopeOutput `json:"scopes" jsonschema:"scopes you share with the session, each with its id: pass one as the scope of chat_send to reach it"`
	Connected  bool          `json:"connected" jsonschema:"true when a connection is joined as this session now"`
	You        bool          `json:"you" jsonschema:"true for your own session"`
	LastSeenAt string        `json:"last_seen_at" jsonschema:"time of the last tool call of a connection joined as the session (RFC 3339, UTC)"`
}

// WhoOutput is the result of the lounge_who tool.
type WhoOutput struct {
	Repository string        `json:"repository" jsonschema:"your repository (empty for none)"`
	Branch     string        `json:"branch" jsonschema:"your branch (empty for none)"`
	Scopes     []ScopeOutput `json:"scopes" jsonschema:"scopes you can use, each with its id"`
	Sessions   []WhoSession  `json:"sessions" jsonschema:"sessions you can reach, most recently active first"`
	Truncated  bool          `json:"truncated" jsonschema:"true when more sessions exist than are listed"`
	Rooms      []string      `json:"rooms" jsonschema:"room names"`
	unreadField
}

func (d *Deps) loungeWho(ctx context.Context, req *mcp.CallToolRequest, _ WhoInput) (*mcp.CallToolResult, WhoOutput, error) {
	me, err := d.presence.caller(req.Session)
	if err != nil {
		return nil, WhoOutput{}, err
	}
	viewer, sessions, truncated, err := d.Store.VisibleSessions(ctx, me.ID, maxSessionsList)
	if err != nil {
		return nil, WhoOutput{}, err
	}
	rooms, err := d.Store.ListRooms(ctx, viewer.UserID)
	if err != nil {
		return nil, WhoOutput{}, err
	}
	out := WhoOutput{
		Repository: viewer.Repository, Branch: viewer.Branch, Scopes: scopesOutput(store.ScopesOf(viewer)),
		Sessions: make([]WhoSession, 0, len(sessions)), Truncated: truncated, Rooms: rooms,
	}
	for _, s := range sessions {
		out.Sessions = append(out.Sessions, WhoSession{
			Name:       s.Name,
			User:       s.UserName,
			Address:    address(viewer.UserName, s.UserName, s.Name),
			Agent:      s.Agent,
			OSUser:     s.OSUser,
			Machine:    s.Machine,
			Repository: s.Repository,
			Branch:     s.Branch,
			Scopes:     scopesOutput(store.SharedScopes(viewer, s)),
			Connected:  d.presence.connected(s.ID),
			You:        s.ID == me.ID,
			LastSeenAt: formatTime(s.LastSeenAt),
		})
	}
	return nil, out, nil
}

// formatTime renders times in tool output: RFC 3339 in UTC.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
