// Package config holds the startup option values of agent-lounge-mcp.
package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// Config holds startup option values only. Components created at runtime (such as the store)
// are passed to the server separately.
type Config struct {
	// Listen is the TCP address the HTTP server listens on. An address other than loopback
	// needs AllowRemote.
	Listen string
	// AllowRemote allows Listen to be an address other than loopback, served over plain HTTP.
	AllowRemote bool
	// RequireToken requires a token on every MCP request even when the server listens only on
	// loopback (see TokenRequired).
	RequireToken bool
	// AllowedHosts are Host header names accepted besides the loopback names, for a reverse
	// proxy in front of a loopback server. Lowercase, without a port.
	AllowedHosts []string
	// DBPath is the SQLite database file. It is created when missing.
	DBPath string
	// ShutdownTimeout bounds how long in-flight requests may finish after a stop signal.
	ShutdownTimeout time.Duration
	// InboxWaitMax is the longest a chat_inbox call may wait for a message (its wait_seconds
	// argument is capped at it). Zero disables waiting. It must stay below the tool-call time
	// limit of the agents, or their waiting calls fail on the agent's side: Claude Code
	// 2.1.325 gives up after 60 seconds. Agents keep waiting by calling chat_inbox again.
	InboxWaitMax time.Duration

	// The periodic cleanup (see store.Cleanup) runs at start-up and every CleanupInterval.
	// A zero period disables its step.
	//
	// ExpireAfter is how long a message may wait unread before its delivery expires.
	// RetainFor is how long a message is kept after its last change of state (once nothing of
	// it is unread), so with the defaults a message lives at most ExpireAfter + RetainFor
	// (44 days). SessionIdle is how long a session may stay unused before it is deleted.
	// Deleting a session deletes the messages it sent, also from the other sessions' history,
	// so the default SessionIdle (60 days) is longer than the longest life of a message.
	ExpireAfter     time.Duration
	RetainFor       time.Duration
	SessionIdle     time.Duration
	CleanupInterval time.Duration

	// Limits of the stored data (see store.Limits). Lowering one deletes nothing: it applies
	// to new messages and new sessions.
	MaxBodyBytes       int
	MaxUnread          int
	MaxSessionsPerUser int
	// MaxMemoryBytes and MaxMemoriesPerScope bound memories in the same way.
	MaxMemoryBytes      int
	MaxMemoriesPerScope int

	// The forgetting model of memories (see store.RetentionPolicy), the same for every scope.
	// MemoryHalfLife is the half-life of a memory never referenced (0 disables forgetting),
	// MemoryFadeThreshold the retention score below which a memory has faded, and MemoryGrace
	// how long a faded memory is kept before the cleanup deletes it (0 keeps it).
	MemoryHalfLife      time.Duration
	MemoryFadeThreshold float64
	MemoryGrace         time.Duration
}

// Bounds of the options.
const (
	// MinCleanupInterval is the smallest CleanupInterval.
	MinCleanupInterval = time.Second
	// MaxMaxBodyBytes is the largest MaxBodyBytes.
	MaxMaxBodyBytes = 1 << 20
)

// Limits returns the limits of the stored data.
func (c Config) Limits() store.Limits {
	return store.Limits{
		MaxBodyBytes: c.MaxBodyBytes, MaxUnread: c.MaxUnread, MaxSessionsPerUser: c.MaxSessionsPerUser,
		MaxMemoryBytes: c.MaxMemoryBytes, MaxMemoriesPerScope: c.MaxMemoriesPerScope,
	}
}

// Retention returns the forgetting model of memories.
func (c Config) Retention() store.RetentionPolicy {
	return store.RetentionPolicy{HalfLife: c.MemoryHalfLife, Threshold: c.MemoryFadeThreshold, Grace: c.MemoryGrace}
}

// CleanupPolicy returns the periods of the periodic cleanup.
func (c Config) CleanupPolicy() store.CleanupPolicy {
	return store.CleanupPolicy{ExpireAfter: c.ExpireAfter, RetainFor: c.RetainFor, SessionIdle: c.SessionIdle}
}

// MaxInboxWaitMax is the largest value InboxWaitMax accepts.
const MaxInboxWaitMax = 10 * time.Minute

// Default returns the configuration used when no options are given.
func Default() Config {
	return Config{
		Listen:          "127.0.0.1:7788",
		DBPath:          "lounge.db",
		ShutdownTimeout: 5 * time.Second,
		InboxWaitMax:    45 * time.Second,

		ExpireAfter:     14 * Day,
		RetainFor:       30 * Day,
		SessionIdle:     60 * Day,
		CleanupInterval: time.Hour,

		MaxBodyBytes:       store.DefaultMaxBodyBytes,
		MaxUnread:          store.DefaultMaxUnread,
		MaxSessionsPerUser: store.DefaultMaxSessionsPerUser,

		MaxMemoryBytes:      store.DefaultMaxMemoryBytes,
		MaxMemoriesPerScope: store.DefaultMaxMemoriesPerScope,
		MemoryHalfLife:      store.DefaultMemoryHalfLife,
		MemoryFadeThreshold: store.DefaultMemoryThreshold,
		MemoryGrace:         store.DefaultMemoryGrace,
	}
}

// Validate rejects configurations the server cannot run safely.
func (c Config) Validate() error {
	if c.DBPath == "" {
		return errors.New("--db must not be empty")
	}
	if strings.ContainsRune(c.DBPath, '?') {
		return errors.New("--db must not contain '?'")
	}
	if c.InboxWaitMax < 0 || c.InboxWaitMax > MaxInboxWaitMax {
		return fmt.Errorf("--inbox-wait-max %s: use a duration from 0 (no waiting) to %s", c.InboxWaitMax, MaxInboxWaitMax)
	}
	for _, p := range []struct {
		name string
		d    time.Duration
	}{{"--expire-after", c.ExpireAfter}, {"--retain-for", c.RetainFor}, {"--session-idle", c.SessionIdle},
		{"--memory-half-life", c.MemoryHalfLife}, {"--memory-grace", c.MemoryGrace}} {
		if p.d < 0 {
			return fmt.Errorf("%s %s: must not be negative (0 disables it)", p.name, p.d)
		}
	}
	if c.CleanupInterval < MinCleanupInterval {
		return fmt.Errorf("--cleanup-interval %s: use %s or more", FormatDuration(c.CleanupInterval), MinCleanupInterval)
	}
	if c.MaxBodyBytes < 1 || c.MaxBodyBytes > MaxMaxBodyBytes {
		return fmt.Errorf("--max-body-bytes %d: use a number from 1 to %d", c.MaxBodyBytes, MaxMaxBodyBytes)
	}
	if c.MaxUnread < 1 {
		return fmt.Errorf("--max-unread %d: use 1 or more", c.MaxUnread)
	}
	if c.MaxSessionsPerUser < 1 {
		return fmt.Errorf("--max-sessions-per-user %d: use 1 or more", c.MaxSessionsPerUser)
	}
	if err := ValidateFadeThreshold(c.MemoryFadeThreshold); err != nil {
		return err
	}
	if c.MaxMemoryBytes < 1 || c.MaxMemoryBytes > MaxMaxBodyBytes {
		return fmt.Errorf("--max-memory-bytes %d: use a number from 1 to %d", c.MaxMemoryBytes, MaxMaxBodyBytes)
	}
	if c.MaxMemoriesPerScope < 1 {
		return fmt.Errorf("--max-memories-per-scope %d: use 1 or more", c.MaxMemoriesPerScope)
	}
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return fmt.Errorf("--listen: %w", err)
	}
	if !isLoopback(host) && !c.AllowRemote {
		return fmt.Errorf("%w: --listen %s is not a loopback address, so other machines could connect over plain HTTP, "+
			"which does not encrypt tokens and messages. Pass --allow-remote to accept it (every request then needs a token: "+
			"create users with agent-lounge-mcp user add), or listen on 127.0.0.1 and put an HTTPS reverse proxy in front (--allowed-host)",
			ErrRemoteNotAllowed, c.Listen)
	}
	for _, h := range c.AllowedHosts {
		if !validHostName(h) {
			return fmt.Errorf("--allowed-host %q: give a host name or IP address without scheme, port or path (for example lounge.example.com)", h)
		}
	}
	return nil
}

// ValidateFadeThreshold checks a --memory-fade-threshold value: above 0 and below 1.
func ValidateFadeThreshold(v float64) error {
	if !(v > 0 && v < 1) {
		return fmt.Errorf("--memory-fade-threshold %v: use a number above 0 and below 1, such as 0.1", v)
	}
	return nil
}

// ErrRemoteNotAllowed is returned by Validate for an address other than loopback without
// AllowRemote.
var ErrRemoteNotAllowed = errors.New("listening on an address other than loopback needs --allow-remote")

// ListensRemote reports whether the server listens on an address other than loopback, where
// other machines can connect. Listen must be valid (see Validate).
func (c Config) ListensRemote() bool {
	host, _, err := net.SplitHostPort(c.Listen)
	return err != nil || !isLoopback(host)
}

// TokenRequired reports whether every MCP request needs a token. Without a token a request
// would be taken as the user local, which is right only for clients of this machine:
//   - an address other than loopback is reachable from other machines;
//   - with AllowedHosts, a reverse proxy forwards the requests of other machines, and they
//     arrive from loopback like local ones;
//   - RequireToken protects loopback from the other users of this machine.
func (c Config) TokenRequired() bool {
	return c.ListensRemote() || len(c.AllowedHosts) > 0 || c.RequireToken
}

// validHostName reports whether h has the form of a Host header name (without port): a DNS
// name or an IP address.
func validHostName(h string) bool {
	if net.ParseIP(strings.Trim(h, "[]")) != nil {
		return true
	}
	if h == "" || len(h) > 253 {
		return false
	}
	for _, r := range h {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '.' || r == '_') {
			return false
		}
	}
	return true
}

// NormalizeHost returns the form of a Host header name that AllowedHosts holds: lowercase,
// without surrounding space.
func NormalizeHost(h string) string {
	return strings.ToLower(strings.TrimSpace(h))
}

// IsLoopbackHost reports whether host (a Host header name without port) names the loopback
// interface: localhost or a loopback IP address.
func IsLoopbackHost(host string) bool {
	return isLoopback(strings.Trim(host, "[]"))
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
