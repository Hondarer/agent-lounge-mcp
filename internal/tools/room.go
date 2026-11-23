package tools

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/names"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// Room actions of chat_room.
const (
	roomCreate = "create"
	roomJoin   = "join"
	roomLeave  = "leave"
	roomList   = "list"
)

func registerRoom(r *Registry, d *Deps) {
	add(r, groupChat, &mcp.Tool{
		Name:  toolRoom,
		Title: "Create, join, leave or list rooms",
		Description: "Manage the rooms of a scope. Every scope has the room lounge, which reaches every session of the scope and cannot be left. " +
			"Other rooms are for topics: a message posted to one reaches only its members. " +
			"action create makes a room and joins you (an existing room is only joined); join and leave change your membership; " +
			"list returns the rooms of a scope, or of every scope you can use when scope is omitted, with whether you joined and how many sessions a post would reach. " +
			"Posting to a room with chat_send also joins you. Any session of a scope can read the messages of its rooms with chat_history, member or not. " +
			"Room names are lowercased.",
		Annotations: additive(false),
	}, d.chatRoom)
}

// RoomInput defines arguments for the chat_room tool.
type RoomInput struct {
	Action string `json:"action" jsonschema:"create, join, leave or list"`
	Scope  string `json:"scope,omitempty" jsonschema:"scope of the room: user (default), repository, branch or shared. For list, omit it to list the rooms of every scope you can use"`
	Room   string `json:"room,omitempty" jsonschema:"room name (lowercased). Required except for list"`
}

// RoomRef is a room in tool output.
type RoomRef struct {
	Scope   string `json:"scope" jsonschema:"scope of the room"`
	ScopeID string `json:"scope_id" jsonschema:"identifier of the scope (empty for shared)"`
	Room    string `json:"room" jsonschema:"room name"`
}

// RoomListEntry is one room in the result of chat_room list.
type RoomListEntry struct {
	RoomRef
	Joined  bool `json:"joined" jsonschema:"true when you are a member (always true for lounge)"`
	Members int  `json:"members" jsonschema:"number of other sessions a message posted now would reach"`
}

// RoomOutput is the result of the chat_room tool.
type RoomOutput struct {
	Scope   string          `json:"scope,omitempty" jsonschema:"scope of the room; absent for list"`
	ScopeID string          `json:"scope_id,omitempty" jsonschema:"identifier of the scope; absent for list and for shared"`
	Room    string          `json:"room,omitempty" jsonschema:"room name after normalization; absent for list"`
	Created *bool           `json:"created,omitempty" jsonschema:"for create: true when the room was made, false when it existed. Absent for other actions"`
	Joined  *bool           `json:"joined,omitempty" jsonschema:"for create, join and leave: whether you are a member after the action"`
	Rooms   []RoomListEntry `json:"rooms,omitempty" jsonschema:"for list: the rooms, by scope, lounge first and then by name"`
	unreadField
}

func (d *Deps) chatRoom(ctx context.Context, req *mcp.CallToolRequest, in RoomInput) (*mcp.CallToolResult, RoomOutput, error) {
	me, err := d.presence.caller(req.Session)
	if err != nil {
		return nil, RoomOutput{}, err
	}
	if in.Scope != "" && !store.ValidScope(in.Scope) {
		return nil, RoomOutput{}, fmt.Errorf("invalid scope %q: %s", in.Scope, scopeChoices)
	}
	if in.Action == roomList {
		rooms, err := d.Store.Rooms(ctx, me.ID, in.Scope, false)
		if err != nil {
			return nil, RoomOutput{}, scopeError(err, in.Scope)
		}
		out := RoomOutput{Rooms: make([]RoomListEntry, 0, len(rooms))}
		for _, r := range rooms {
			out.Rooms = append(out.Rooms, RoomListEntry{RoomRef: roomRef(r.RoomRef), Joined: r.Joined, Members: r.Members})
		}
		return nil, out, nil
	}
	var op func(context.Context, int64, string, string) (store.RoomResult, error)
	switch in.Action {
	case roomCreate:
		op = d.Store.CreateRoom
	case roomJoin:
		op = d.Store.JoinRoom
	case roomLeave:
		op = d.Store.LeaveRoom
	default:
		return nil, RoomOutput{}, fmt.Errorf("invalid action %q: use %s, %s, %s or %s", in.Action, roomCreate, roomJoin, roomLeave, roomList)
	}
	if in.Room == "" {
		return nil, RoomOutput{}, fmt.Errorf("give the room name (room) for action %s", in.Action)
	}
	room, err := names.Room(in.Room)
	if err != nil {
		return nil, RoomOutput{}, err
	}
	scope := cmp.Or(in.Scope, store.ScopeUser)
	res, err := op(ctx, me.ID, scope, room)
	switch {
	case errors.Is(err, store.ErrNoSuchRoom):
		return nil, RoomOutput{}, noRoomError(room, scope)
	case errors.Is(err, store.ErrLeaveLounge):
		return nil, RoomOutput{}, fmt.Errorf("%s cannot be left: every session of the scope belongs to it", store.LoungeRoom)
	case errors.Is(err, store.ErrTooManyRooms):
		return nil, RoomOutput{}, fmt.Errorf("scope %s already has %d rooms besides %s, the limit of this server: post to an existing room (see action %s)", scope, d.Cfg.MaxRoomsPerScope, store.LoungeRoom, roomList)
	case err != nil:
		return nil, RoomOutput{}, scopeError(err, scope)
	}
	ref := roomRef(res.RoomRef)
	out := RoomOutput{Scope: ref.Scope, ScopeID: ref.ScopeID, Room: ref.Room, Joined: &res.Joined}
	if in.Action == roomCreate {
		out.Created = &res.Created
	}
	return nil, out, nil
}

// roomRef converts a store room reference to tool output.
func roomRef(r store.RoomRef) RoomRef {
	return RoomRef{Scope: r.Scope.Kind, ScopeID: r.Scope.ID, Room: r.Name}
}

// joinedRooms returns the rooms the session belongs to, for lounge_join and lounge_who.
func (d *Deps) joinedRooms(ctx context.Context, sessionID int64) ([]RoomRef, error) {
	rooms, err := d.Store.Rooms(ctx, sessionID, "", true)
	if err != nil {
		return nil, err
	}
	out := make([]RoomRef, 0, len(rooms))
	for _, r := range rooms {
		out = append(out, roomRef(r.RoomRef))
	}
	return out, nil
}

// scopeChoices lists the values of a scope argument for error messages.
var scopeChoices = fmt.Sprintf("use %s (default), %s, %s or %s", store.ScopeUser, store.ScopeRepository, store.ScopeBranch, store.ScopeShared)

// scopeError turns the store's errors for a scope the session cannot use into guidance.
func scopeError(err error, scope string) error {
	switch {
	case errors.Is(err, store.ErrInvalidScope):
		return fmt.Errorf("invalid scope %q: %s", scope, scopeChoices)
	case errors.Is(err, store.ErrNoRepository):
		return fmt.Errorf("your session has no repository, so scope %s cannot be used: call %s with the same name and your origin, "+
			"or your folder and machine (see %s), then try again", store.ScopeRepository, toolJoin, toolHelp)
	case errors.Is(err, store.ErrNoBranch):
		return fmt.Errorf("your session cannot use scope %s: call %s with the same name, your origin (or folder and machine), "+
			"and branch (the output of git branch --show-current), then try again. Scope %s is not available when not on a branch (detached HEAD)",
			store.ScopeBranch, toolJoin, store.ScopeBranch)
	}
	return err
}

// noRoomError is the error for a room name that does not exist in the scope. A scope of ""
// stands for every scope the session can use.
func noRoomError(room, scope string) error {
	where := "any of your scopes"
	if scope != "" {
		where = "scope " + scope
	}
	return fmt.Errorf("no room named %q in %s: create it with %s action %s, or see the rooms with %s action %s",
		room, where, toolRoom, roomCreate, toolRoom, roomList)
}
