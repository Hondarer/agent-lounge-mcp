//go:build !windows

package connection

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/hondarer/agent-lounge-mcp/internal/ownerdir"
)

// defaultDir is $XDG_RUNTIME_DIR/agent-lounge-mcp/connections, or agent-lounge-mcp-UID/connections in the
// temporary directory. macOS uses /tmp, because its per-user temporary directory is too long
// for socket paths.
func defaultDir() (string, error) {
	var base string
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		base = filepath.Join(xdg, "agent-lounge-mcp")
	} else {
		tmp := os.TempDir()
		if runtime.GOOS == "darwin" {
			tmp = "/tmp"
		}
		base = filepath.Join(tmp, fmt.Sprintf("agent-lounge-mcp-%d", os.Getuid()))
	}
	dir := filepath.Join(base, "connections")
	for _, d := range []string{base, dir} {
		if err := ownerdir.Ensure(d); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// startDetached starts the command from newCmd in a session of its own, without a controlling
// terminal, so that it outlives the shell that ran connection start --detach.
func startDetached(newCmd func() *exec.Cmd) (*exec.Cmd, error) {
	cmd := newCmd()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd, cmd.Start()
}

func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
