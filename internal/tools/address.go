package tools

import (
	"fmt"
	"strings"

	"github.com/hondarer/agent-lounge-mcp/internal/names"
)

// Addresses name sessions in tool input and output (to, address, reply_to, recipients,
// skipped). Session names are unique per user, and other users may use the same name, so an
// address carries the user when it differs from the viewer's:
//
//   - a session of the viewer's own user: its name, such as claude;
//   - a session of another user: user/name, such as bob/claude.
//
// Neither a user name nor a session name contains '/', so the two parts split unambiguously.
// The server builds the addresses it returns (address and reply_to) instead of leaving the
// form to agents: the rule is fixed, so the server applies it.

// address returns how a session of user viewer refers to the session name of user owner.
func address(viewer, owner, name string) string {
	if owner == viewer {
		return name
	}
	return owner + "/" + name
}

// parseAddress reads the address to given by a session of user viewer and returns the user and
// the session name it refers to. viewer/name is the same as name.
func parseAddress(viewer, to string) (user, name string, err error) {
	user, name, qualified := strings.Cut(to, "/")
	if qualified {
		user, err = names.User(user)
		if err != nil {
			return "", "", addressError(to, err)
		}
	} else {
		user, name = viewer, to
	}
	name, err = names.Session(name)
	if err != nil {
		return "", "", addressError(to, err)
	}
	return user, name, nil
}

// addressError tells the agent the two forms of an address and where to find one, with the
// rule the name broke.
func addressError(to string, err error) error {
	return fmt.Errorf("invalid to %q: use a session name for a session of your user, or user/name for a session of another user "+
		"(the address in %s): %w", to, toolWho, err)
}
