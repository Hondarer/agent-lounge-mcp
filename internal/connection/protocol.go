package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/ownerdir"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const EnvName = "AGENT_LOUNGE_CONNECTION"
const EnvDir = "AGENT_LOUNGE_CONNECTION_DIR"

var validName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,32}$`)

func Name() string {
	if name := os.Getenv(EnvName); name != "" {
		return name
	}
	return "default"
}

func CheckName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid connection name %q: use 1 to 32 letters, digits, '.', '_' or '-'", name)
	}
	return nil
}

func directory() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		absolute, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		return absolute, ownerdir.Ensure(absolute)
	}
	return defaultDir()
}

func socketPath(name string) (string, error) {
	if err := CheckName(name); err != nil {
		return "", err
	}
	dir, err := directory()
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(dir, name+".sock"))
	if err != nil {
		return "", err
	}
	if len(path) > 103 {
		return "", fmt.Errorf("socket path is too long; set %s to a shorter directory", EnvDir)
	}
	return path, nil
}

type State struct {
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	Started      time.Time              `json:"started"`
	PID          int                    `json:"pid"`
	Participants map[string]Participant `json:"participants"`
}

type Participant struct {
	Connected bool   `json:"connected"`
	LastJoin  string `json:"last_join,omitempty"`
}

type Request struct {
	Op   string
	Tool string
	As   string
	Args json.RawMessage
}

type Reply struct {
	Code   int
	Error  string
	State  *State
	Tools  []*mcp.Tool
	Result json.RawMessage
}

// Exchange uses one socket per request. Closing it, including on process exit,
// cancels its upstream call independently of other requests.
func Exchange(ctx context.Context, name string, req Request) (Reply, error) {
	var reply Reply
	path, err := socketPath(name)
	if err != nil {
		return reply, err
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return reply, fmt.Errorf("cannot connect to daemon %q; check agent-lounge-mcp connection list", name)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return reply, err
	}
	if err := json.NewDecoder(conn).Decode(&reply); err != nil {
		return reply, fmt.Errorf("lost connection to daemon %q; the outcome is unknown", name)
	}
	return reply, nil
}

func List(ctx context.Context) ([]*State, error) {
	dir, err := directory()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var states []*State
	for _, entry := range entries {
		name, ok := strings.CutSuffix(entry.Name(), ".sock")
		if !ok || CheckName(name) != nil {
			continue
		}
		probe, cancel := context.WithTimeout(ctx, time.Second)
		reply, err := Exchange(probe, name, Request{Op: "status"})
		cancel()
		if err == nil && reply.State != nil {
			states = append(states, reply.State)
		}
	}
	return states, ctx.Err()
}
