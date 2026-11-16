package tools

import (
	"context"
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

const (
	maxAgentLen     = 128
	maxSessionsList = 200
)

// sessionNameRule describes sessionNameRE to agents.
const sessionNameRule = "1 to 64 characters from A-Z a-z 0-9 . _ - @"

// sessionNameRE is the form of a session name.
var sessionNameRE = regexp.MustCompile(`^[A-Za-z0-9._@-]{1,64}$`)

func registerLounge(r *Registry, d *Deps) {
	add(r, groupChat, &mcp.Tool{
		Name:  toolJoin,
		Title: "Join the lounge",
		Description: "Join the lounge under a session name. Call this first: the chat tools work only after joining. " +
			"Other sessions address messages to this name. Joining with a name used before continues that session and its unread messages " +
			"(if another connection is using the name, it loses the session). After a reconnect, join again with the same name. " +
			"If you do not know how to use the lounge, call lounge_help first.",
		Annotations: additive(true),
	}, d.loungeJoin)

	add(r, groupChat, &mcp.Tool{
		Name:        toolWho,
		Title:       "List sessions and rooms",
		Description: "List the sessions that joined (name, agent, whether connected now, last activity) and the rooms. Use a session name as the 'to' of chat_send.",
		Annotations: readOnly(),
	}, d.loungeWho)
}

// JoinInput defines arguments for the lounge_join tool.
type JoinInput struct {
	Name  string  `json:"name,omitempty" jsonschema:"session name: 1 to 64 characters from A-Z a-z 0-9 . _ - @ (for example claude or review-task). Omit to let the server choose one"`
	Agent *string `json:"agent,omitempty" jsonschema:"what you are, shown to other sessions, for example claude-code@host-a (up to 128 characters). Omit to keep the recorded value"`
}

// JoinOutput is the result of the lounge_join tool.
type JoinOutput struct {
	Name    string   `json:"name" jsonschema:"the session name you joined as"`
	Agent   string   `json:"agent" jsonschema:"recorded agent name"`
	User    string   `json:"user" jsonschema:"user the session belongs to"`
	Resumed bool     `json:"resumed" jsonschema:"true when an existing session was continued"`
	Rooms   []string `json:"rooms" jsonschema:"rooms the session is a member of"`
	unreadField
}

func (d *Deps) loungeJoin(ctx context.Context, req *mcp.CallToolRequest, in JoinInput) (*mcp.CallToolResult, JoinOutput, error) {
	if in.Name != "" && !sessionNameRE.MatchString(in.Name) {
		return nil, JoinOutput{}, fmt.Errorf("invalid name %q: use %s", in.Name, sessionNameRule)
	}
	if in.Agent != nil && utf8.RuneCountInString(*in.Agent) > maxAgentLen {
		return nil, JoinOutput{}, fmt.Errorf("agent is too long: %d characters, at most %d", utf8.RuneCountInString(*in.Agent), maxAgentLen)
	}
	sess, resumed, err := d.Store.JoinSession(ctx, store.JoinParams{UserID: d.UserID, Name: in.Name, Agent: in.Agent})
	if err != nil {
		return nil, JoinOutput{}, err
	}
	rooms, err := d.Store.ListRooms(ctx, d.UserID)
	if err != nil {
		return nil, JoinOutput{}, err
	}
	d.notify.wake(d.presence.bind(req.Session, sess)...)
	return nil, JoinOutput{Name: sess.Name, Agent: sess.Agent, User: store.LocalUser, Resumed: resumed, Rooms: rooms}, nil
}

// WhoInput defines arguments for the lounge_who tool.
type WhoInput struct{}

// WhoSession is one session in the result of lounge_who.
type WhoSession struct {
	Name       string `json:"name" jsonschema:"session name"`
	Agent      string `json:"agent" jsonschema:"agent name the session declared"`
	Connected  bool   `json:"connected" jsonschema:"true when a connection is joined as this session now"`
	You        bool   `json:"you" jsonschema:"true for your own session"`
	LastSeenAt string `json:"last_seen_at" jsonschema:"time of the session's last successful tool call (RFC 3339, UTC)"`
}

// WhoOutput is the result of the lounge_who tool.
type WhoOutput struct {
	Sessions  []WhoSession `json:"sessions" jsonschema:"sessions, most recently active first"`
	Truncated bool         `json:"truncated" jsonschema:"true when more sessions exist than are listed"`
	Rooms     []string     `json:"rooms" jsonschema:"room names"`
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
	out := WhoOutput{Sessions: make([]WhoSession, 0, len(sessions)), Truncated: truncated, Rooms: rooms}
	for _, s := range sessions {
		out.Sessions = append(out.Sessions, WhoSession{
			Name:       s.Name,
			Agent:      s.Agent,
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
