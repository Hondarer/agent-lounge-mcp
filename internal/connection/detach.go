package connection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const childEnv = "AGENT_LOUNGE_INTERNAL_CONNECTION_CHILD"

// Detach re-executes the CLI and writes its credential to the child's stdin pipe.
// The child's stdout is a separate pipe used only for the startup result.
func Detach(ctx context.Context, opt Options, args []string) (*State, error) {
	logPath, err := LogPath(opt.Name)
	if err != nil {
		return nil, err
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer log.Close()
	input, tokenWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer input.Close()
	defer tokenWriter.Close()
	output, readyWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer output.Close()
	defer readyWriter.Close()
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cmd, err := startDetached(func() *exec.Cmd {
		cmd := exec.Command(self, append([]string{"connection", "start"}, args...)...)
		cmd.Env = append(os.Environ(), childEnv+"="+strconv.Itoa(os.Getpid()))
		cmd.Stdin, cmd.Stdout, cmd.Stderr = input, readyWriter, log
		return cmd
	})
	if err != nil {
		return nil, err
	}
	input.Close()
	readyWriter.Close()
	go func() { _ = cmd.Wait() }()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() {
		_ = cmd.Process.Kill()
		tokenWriter.Close()
		output.Close()
	})
	defer stop()
	if _, err := fmt.Fprintln(tokenWriter, opt.Token); err != nil {
		return nil, errors.New("cannot pass token to detached daemon")
	}
	tokenWriter.Close()
	var reply Reply
	if err := json.NewDecoder(output).Decode(&reply); err != nil {
		return nil, fmt.Errorf("daemon did not start; check %s", logPath)
	}
	if reply.Error != "" {
		return nil, errors.New(reply.Error)
	}
	return reply.State, nil
}

func IsChild() bool {
	return os.Getenv(childEnv) == strconv.Itoa(os.Getppid())
}
