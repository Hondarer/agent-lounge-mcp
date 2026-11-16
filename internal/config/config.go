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
	// Listen is the TCP address the HTTP server listens on.
	Listen string
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
	// Requests are not authenticated yet, so only loopback clients may connect.
	if !isLoopback(host) {
		return fmt.Errorf("--listen %s: only loopback addresses (127.0.0.1, [::1], localhost) are supported until token authentication is implemented", c.Listen)
	}
	return nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
