package connection

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/hondarer/agent-lounge-mcp/internal/ownerdir"
)

// defaultDir is %LOCALAPPDATA%\agent-lounge-mcp\connections.
func defaultDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var err error
		if base, err = os.UserCacheDir(); err != nil {
			return "", err
		}
	}
	dir := filepath.Join(base, "agent-lounge-mcp", "connections")
	return dir, ownerdir.Ensure(dir)
}

// startDetached starts the command from newCmd without a window, in a process group of its own,
// so that Ctrl+C in the shell that ran session start --detach does not reach it. It also leaves
// the job object of that shell when the job allows it, because closing the job would otherwise
// end the session.
func startDetached(newCmd func() *exec.Cmd) (*exec.Cmd, error) {
	flags := uint32(windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW)
	cmd := newCmd()
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags | windows.CREATE_BREAKAWAY_FROM_JOB}
	if err := cmd.Start(); err == nil {
		return cmd, nil
	}
	// The job does not permit breakaway.
	cmd = newCmd()
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags}
	return cmd, cmd.Start()
}

func lockFile(f *os.File) error {
	return windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, new(windows.Overlapped))
}
