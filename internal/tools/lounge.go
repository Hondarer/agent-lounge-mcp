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
			"Other sessions address messages to this name. Joining with a name used before continues that session and its unread messages " +
			"(if another connection is using the name, it loses the session). After a reconnect, join again with the same name. " +
			"Pass your profile (os_user, machine, origin, folder, wsl_distro) as the output of the commands that lounge_help lists; " +
			"the server normalizes the values, decides your repository, and returns them. " +
			"When the repository or folder you work in changes, call it again with the same name. " +
			"If you do not know how to use the lounge, call lounge_help first.",
		Annotations: additive(true),
	}, d.loungeJoin)

	add(r, groupChat, &mcp.Tool{
		Name:  toolWho,
		Title: "List sessions and rooms",
		Description: "List the sessions that joined (name, agent, OS user, machine, repository, whether it is your repository, whether connected now, last activity), " +
			"your own repository, and the rooms. Use a session name as the 'to' of chat_send.",
		Annotations: readOnly(),
	}, d.loungeWho)
}

// JoinInput defines arguments for the lounge_join tool.
type JoinInput struct {
	Name  string  `json:"name,omitempty" jsonschema:"session name: 1 to 64 characters from A-Z a-z 0-9 . _ - @ (for example claude or review-task). Omit to let the server choose one"`
	Agent *string `json:"agent,omitempty" jsonschema:"what you are, shown to other sessions, for example claude-code@host-a (up to 128 characters). Omit to keep the recorded value"`
	// Profile. agent, os_user and machine: omit to keep the recorded value, "" to clear it.
	// origin, folder and wsl_distro: see joinParams.
	OSUser    *string `json:"os_user,omitempty" jsonschema:"OS user name running you, as printed by the command lounge_help lists (on WSL, the Windows user name of the host). Omit to keep the recorded value, empty to clear it"`
	Machine   *string `json:"machine,omitempty" jsonschema:"machine name, as printed by hostname (on WSL, cmd.exe /c hostname). Omit to keep the recorded value, empty to clear it"`
	Origin    *string `json:"origin,omitempty" jsonschema:"URL of the Git origin of your working folder, as printed by git remote get-url origin, without credentials; empty when there is none. origin and folder are replaced together: omit origin, folder and wsl_distro to keep the recorded ones"`
	Folder    *string `json:"folder,omitempty" jsonschema:"top of the Git working tree (git rev-parse --show-toplevel), or the working folder outside Git (pwd). origin and folder are replaced together"`
	WSLDistro *string `json:"wsl_distro,omitempty" jsonschema:"on WSL, the distribution name (echo $WSL_DISTRO_NAME); omit elsewhere. Used only to convert folder, not stored"`
}

// JoinOutput is the result of the lounge_join tool.
type JoinOutput struct {
	Name    string   `json:"name" jsonschema:"the session name you joined as"`
	Agent   string   `json:"agent" jsonschema:"recorded agent name"`
	User    string   `json:"user" jsonschema:"user the session belongs to"`
	Resumed bool     `json:"resumed" jsonschema:"true when an existing session was continued"`
	Rooms   []string `json:"rooms" jsonschema:"rooms the session is a member of"`
	// Profile as recorded.
	OSUser     string `json:"os_user" jsonschema:"recorded OS user name"`
	Machine    string `json:"machine" jsonschema:"recorded machine name"`
	Origin     string `json:"origin" jsonschema:"recorded origin, normalized to host/path (empty for none)"`
	Folder     string `json:"folder" jsonschema:"recorded folder, normalized (empty for none)"`
	Repository string `json:"repository" jsonschema:"your repository, decided by the server: the origin, else machine:folder, else empty (the repository space is not available)"`
	unreadField
}

// joinParams validates and normalizes the arguments of lounge_join.
//
// origin and folder are one set: when origin, folder or wsl_distro is given, the location is
// replaced, and a missing origin or folder counts as empty. When all three are omitted, the
// recorded location is kept.
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
	p, err := joinParams(d.UserID, in)
	if err != nil {
		return nil, JoinOutput{}, err
	}
	sess, resumed, err := d.Store.JoinSession(ctx, p)
	switch {
	case errors.Is(err, store.ErrMachineRequired):
		return nil, JoinOutput{}, errors.New("folder without origin needs machine: pass machine (the output of hostname; on WSL, cmd.exe /c hostname), " +
			"so that the same path on another machine is not taken for the same repository")
	case err != nil:
		return nil, JoinOutput{}, err
	}
	rooms, err := d.Store.ListRooms(ctx, d.UserID)
	if err != nil {
		return nil, JoinOutput{}, err
	}
	d.notify.wake(d.presence.bind(req.Session, sess)...)
	return nil, JoinOutput{
		Name: sess.Name, Agent: sess.Agent, User: store.LocalUser, Resumed: resumed, Rooms: rooms,
		OSUser: sess.OSUser, Machine: sess.Machine, Origin: sess.Origin, Folder: sess.Folder, Repository: sess.Repository,
	}, nil
}

// WhoInput defines arguments for the lounge_who tool.
type WhoInput struct{}

// WhoSession is one session in the result of lounge_who.
type WhoSession struct {
	Name       string `json:"name" jsonschema:"session name"`
	Agent      string `json:"agent" jsonschema:"agent name the session declared"`
	OSUser     string `json:"os_user" jsonschema:"OS user name the session declared"`
	Machine    string `json:"machine" jsonschema:"machine name the session declared"`
	Repository string `json:"repository" jsonschema:"repository of the session (empty for none)"`
	// SameRepository is false when either repository is empty.
	SameRepository bool   `json:"same_repository" jsonschema:"true when the session has your repository"`
	Connected      bool   `json:"connected" jsonschema:"true when a connection is joined as this session now"`
	You            bool   `json:"you" jsonschema:"true for your own session"`
	LastSeenAt     string `json:"last_seen_at" jsonschema:"time of the session's last successful tool call (RFC 3339, UTC)"`
}

// WhoOutput is the result of the lounge_who tool.
type WhoOutput struct {
	Repository string       `json:"repository" jsonschema:"your repository (empty for none)"`
	Sessions   []WhoSession `json:"sessions" jsonschema:"sessions, most recently active first"`
	Truncated  bool         `json:"truncated" jsonschema:"true when more sessions exist than are listed"`
	Rooms      []string     `json:"rooms" jsonschema:"room names"`
	unreadField
}

func (d *Deps) loungeWho(ctx context.Context, req *mcp.CallToolRequest, _ WhoInput) (*mcp.CallToolResult, WhoOutput, error) {
	me, err := d.presence.caller(req.Session)
	if err != nil {
		return nil, WhoOutput{}, err
	}
	sessions, truncated, err := d.Store.ListSessions(ctx, d.UserID, maxSessionsList)
	if err != nil {
		return nil, WhoOutput{}, err
	}
	rooms, err := d.Store.ListRooms(ctx, d.UserID)
	if err != nil {
		return nil, WhoOutput{}, err
	}
	// The profile changes only at lounge_join, which binds the connection to the updated
	// session, so the caller's session in presence is current.
	mine := me.Repository
	out := WhoOutput{Repository: mine, Sessions: make([]WhoSession, 0, len(sessions)), Truncated: truncated, Rooms: rooms}
	for _, s := range sessions {
		out.Sessions = append(out.Sessions, WhoSession{
			Name:           s.Name,
			Agent:          s.Agent,
			OSUser:         s.OSUser,
			Machine:        s.Machine,
			Repository:     s.Repository,
			SameRepository: mine != "" && s.Repository == mine,
			Connected:      d.presence.connected(s.ID),
			You:            s.ID == me.ID,
			LastSeenAt:     formatTime(s.LastSeenAt),
		})
	}
	return nil, out, nil
}

// formatTime renders times in tool output: RFC 3339 in UTC.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
