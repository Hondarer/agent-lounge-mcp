package store

// The sessions a session can talk to are chosen on two axes: the scope, which is the range of
// sessions (the sessions of one user, or the sessions of one repository unit), and the target
// within that range (every other session, through the room, or one session).
//
// A scope is a kind together with an identifier that picks one range of that kind. A session
// uses only the scopes it belongs to: the identifier is always decided from the session itself
// (scopeRule.id), never given by the caller. Sessions of other users are therefore reachable
// only through a scope both sessions share.
//
// Each kind is one entry of scopeRules. Adding a kind, such as a shared scope of all users,
// means adding its name to scopeKinds and one rule: how its identifier is decided, which
// sessions it contains, and where its rooms are kept.

// Scope kinds.
const (
	// ScopeUser is the range of the sessions of one user. Its identifier is the user name.
	ScopeUser = "user"
	// ScopeRepository is the range of the sessions that declare one repository unit, whatever
	// user they belong to. Its identifier is the unit.
	ScopeRepository = "repository"
)

// Scope is a range of sessions: its kind and identifier.
type Scope struct {
	Kind string
	ID   string
}

// scopeKinds lists the kinds in the order they are reported.
var scopeKinds = []string{ScopeUser, ScopeRepository}

// scopeRule defines one kind of scope.
type scopeRule struct {
	// id returns the identifier of the scope for a session, and false when the session cannot
	// use the scope.
	id func(Session) (string, bool)
	// unavailable is the error for a session that cannot use the scope.
	unavailable error
	// members returns an SQL condition that selects the sessions of the scope with the given
	// identifier, on sessions (alias s) joined with their user (alias u), and its arguments.
	members func(id string) (cond string, args []any)
	// room returns the user_id and repository columns of the rooms of the scope.
	room func(sender Session, id string) (userID any, repository string)
}

var scopeRules = map[string]scopeRule{
	ScopeUser: {
		id:          func(s Session) (string, bool) { return s.UserName, true },
		unavailable: ErrInvalidScope, // every session has a user
		members:     func(id string) (string, []any) { return "u.name = ?", []any{id} },
		room:        func(s Session, _ string) (any, string) { return s.UserID, "" },
	},
	ScopeRepository: {
		id:          func(s Session) (string, bool) { return s.Repository, s.Repository != "" },
		unavailable: ErrNoRepository,
		members:     func(id string) (string, []any) { return "s.repository = ?", []any{id} },
		room:        func(_ Session, id string) (any, string) { return nil, id },
	},
}

// ScopeKinds returns the scope kinds, in the order they are reported.
func ScopeKinds() []string {
	return append([]string(nil), scopeKinds...)
}

// ValidScope reports whether kind is a scope kind.
func ValidScope(kind string) bool {
	_, ok := scopeRules[kind]
	return ok
}

// ScopesOf returns the scopes the session can use.
func ScopesOf(sess Session) []Scope {
	var out []Scope
	for _, kind := range scopeKinds {
		if id, ok := scopeRules[kind].id(sess); ok {
			out = append(out, Scope{Kind: kind, ID: id})
		}
	}
	return out
}

// SharedScopes returns the scopes in which a and b reach each other: those both can use with
// the same identifier.
func SharedScopes(a, b Session) []Scope {
	out := []Scope{}
	for _, kind := range scopeKinds {
		rule := scopeRules[kind]
		ida, oka := rule.id(a)
		idb, okb := rule.id(b)
		if oka && okb && ida == idb {
			out = append(out, Scope{Kind: kind, ID: ida})
		}
	}
	return out
}
