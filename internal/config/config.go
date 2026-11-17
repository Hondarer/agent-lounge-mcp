// Package config holds the startup option values of agent-lounge-mcp.
package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
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
