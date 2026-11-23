package loungecli

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Collect only at explicit joins. The call's directory may differ from the
// agent's work location, so inspecting it on each call could change its profile.
func collectProfile(ctx context.Context, args map[string]any) {
	set := func(key string, collect func() (string, bool)) {
		if _, explicit := args[key]; !explicit {
			if value, ok := collect(); ok {
				args[key] = value
			}
		}
	}
	run := func(command string, argv ...string) (string, bool) {
		b, err := exec.CommandContext(ctx, command, argv...).Output()
		return string(b), err == nil
	}
	distro := os.Getenv("WSL_DISTRO_NAME")
	switch {
	case runtime.GOOS == "windows":
		set("os_user", func() (string, bool) { return os.LookupEnv("USERNAME") })
		set("machine", func() (string, bool) {
			name, err := os.Hostname()
			return name, err == nil
		})
	case runtime.GOOS == "linux" && distro != "":
		set("wsl_distro", func() (string, bool) { return distro, true })
		set("os_user", func() (string, bool) { return run("cmd.exe", "/c", "echo %USERNAME%") })
		set("machine", func() (string, bool) {
			if name, ok := run("cmd.exe", "/c", "hostname"); ok {
				return name, true
			}
			b, err := os.ReadFile("/etc/hostname")
			return string(b), err == nil
		})
	default:
		set("os_user", func() (string, bool) { return run("id", "-un") })
		set("machine", func() (string, bool) { return run("uname", "-n") })
	}
	git := func(argv ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", argv...)
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		return cmd.Output()
	}
	set("origin", func() (string, bool) {
		b, err := git("remote", "get-url", "origin")
		if err == nil {
			return cleanOrigin(string(b))
		}
		if exit, ok := err.(*exec.ExitError); ok {
			diagnostic := string(exit.Stderr)
			if strings.Contains(diagnostic, "No such remote") || strings.Contains(diagnostic, "not a git repository") {
				return "", true
			}
		}
		return "", false
	})
	set("folder", func() (string, bool) {
		b, err := git("rev-parse", "--show-toplevel")
		if err == nil {
			return string(b), true
		}
		if exit, ok := err.(*exec.ExitError); ok && strings.Contains(string(exit.Stderr), "not a git repository") {
			dir, err := os.Getwd()
			return dir, err == nil
		}
		return "", false
	})
	set("branch", func() (string, bool) {
		b, err := git("branch", "--show-current")
		if exit, ok := err.(*exec.ExitError); ok && strings.Contains(string(exit.Stderr), "not a git repository") {
			return "", true
		}
		return string(b), err == nil
	})
}

func cleanOrigin(raw string) (string, bool) {
	value := strings.TrimRight(raw, "\r\n")
	suffix := raw[len(value):]
	if _, rest, ok := strings.Cut(value, "://"); ok {
		u, err := url.Parse(value)
		if err != nil {
			return "", false
		}
		host, path, _ := strings.Cut(rest, "/")
		if at := strings.LastIndexByte(host, '@'); u.User != nil && at >= 0 {
			value = value[:len(value)-len(rest)] + host[at+1:]
			if len(rest) > len(host) {
				value += "/" + path
			}
		}
		if end := strings.IndexAny(value, "?#"); end >= 0 {
			value = value[:end]
		}
		return value + suffix, true
	}
	head, _, _ := strings.Cut(value, "/")
	if at := strings.LastIndexByte(head, '@'); at >= 0 {
		value = value[at+1:]
	}
	return value + suffix, true
}
