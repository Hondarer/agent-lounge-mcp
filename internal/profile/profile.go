// Package profile normalizes the profile values a session declares at lounge_join and decides
// the session's repository unit.
//
// The work is split between the agent and the server. The agent runs fixed commands on its own
// machine (id -un, hostname, git remote get-url origin, git rev-parse --show-toplevel, pwd,
// echo $WSL_DISTRO_NAME) and passes their output as is. The server turns the values into one
// form with the rules in this package, because the rules are fixed and need no knowledge of the
// agent's machine. The package only processes strings: it never calls the operating system or
// reads the file system, since the server cannot check paths on the agent's machine.
//
// Limits of the rules: symbolic links are not resolved; on WSL only the default mount point of
// Windows drives (/mnt/<drive letter>) is recognized; a hostless origin (a local path or a
// file:// URL) is treated as no origin, since it names nothing that is the same across machines.
package profile

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

// MaxNameLen is the largest length of os_user and machine, in characters.
const MaxNameLen = 128

// ErrMachineRequired is returned by Repository when a folder without an origin is declared
// without a machine: the same path on another machine would otherwise share the unit.
var ErrMachineRequired = errors.New("a folder without an origin needs a machine")

// OSUser normalizes the OS user name: surrounding white space is removed and a Windows
// DOMAIN\user form is kept as is. An empty result means no value.
func OSUser(raw string) (string, error) {
	return name("os_user", strings.TrimSpace(raw))
}

// Machine normalizes the machine name: surrounding white space is removed and the name is
// lowercased. An empty result means no value.
func Machine(raw string) (string, error) {
	return name("machine", strings.ToLower(strings.TrimSpace(raw)))
}

func name(field, s string) (string, error) {
	// A variable reference such as %USERNAME% or $USER means the shell did not expand it
	// (for example echo %USERNAME% run in PowerShell).
	if unexpandedRE.MatchString(s) {
		return "", fmt.Errorf("%s %q is a variable the shell did not expand: run the command lounge_help lists for your environment and pass its output", field, s)
	}
	if n := utf8.RuneCountInString(s); n > MaxNameLen {
		return "", fmt.Errorf("%s is too long: %d characters, at most %d", field, n, MaxNameLen)
	}
	return s, nil
}

// unexpandedRE matches a value that is only a variable reference: %NAME%, $NAME, ${NAME} or
// $env:NAME.
var unexpandedRE = regexp.MustCompile(`^(%[A-Za-z_][A-Za-z0-9_]*%|\$(env:)?[A-Za-z_][A-Za-z0-9_]*|\$\{[A-Za-z_][A-Za-z0-9_]*\})$`)

// originSchemes are the URL schemes accepted for an origin.
var originSchemes = map[string]bool{"https": true, "http": true, "ssh": true, "git": true, "git+ssh": true}

// schemeRE is the form of a URL scheme. Only a scheme of this form is quoted in an error.
var schemeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)

// driveRE matches a path that starts with a Windows drive letter and a separator (or ends there).
var driveRE = regexp.MustCompile(`^[A-Za-z]:([\\/]|$)`)

// Origin normalizes the URL of a Git origin to host/path in lowercase, for example
// github.com/owner/repo. It accepts the scp-like form user@host:path, the URL forms with the
// schemes https, http, ssh, git and git+ssh, and the normalized form itself. The scheme, user
// name and credentials, port, query, fragment, trailing slashes and a trailing .git are
// removed.
//
// A hostless origin (a local path or a file:// URL) yields "" without an error: it is treated
// as no origin. A value whose host or path cannot be found is an error. Errors never quote the
// value, because it may contain credentials.
func Origin(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	var host, p string
	if scheme, _, ok := strings.Cut(s, "://"); ok {
		scheme = strings.ToLower(scheme)
		if scheme == "file" {
			return "", nil
		}
		if !originSchemes[scheme] {
			if schemeRE.MatchString(scheme) {
				return "", fmt.Errorf("origin: unsupported URL scheme %q (use https, http, ssh, git, git+ssh or user@host:path)", scheme)
			}
			return "", errors.New("origin: not a URL or user@host:path")
		}
		u, err := url.Parse(s)
		if err != nil {
			// The parse error quotes the value: do not wrap it.
			return "", errors.New("origin: the URL cannot be parsed")
		}
		host, p = u.Hostname(), u.Path
		if strings.Contains(host, ":") {
			host = "[" + host + "]" // IPv6 literal
		}
	} else {
		if isLocalPath(s) {
			return "", nil
		}
		// Drop a user name (and anything else up to the last @) before the host. The host ends
		// at the first slash at the latest.
		head := s
		if i := strings.IndexByte(s, '/'); i >= 0 {
			head = s[:i]
		}
		if i := strings.LastIndexByte(head, '@'); i >= 0 {
			s = s[i+1:]
		}
		i := strings.IndexAny(s, ":/")
		switch {
		case i < 0:
			host = s
		case s[i] == ':':
			// scp-like form host:path. A one-letter host is a Windows drive (C:repo).
			if i == 1 {
				return "", nil
			}
			host, p = s[:i], s[i+1:]
		default:
			// Normalized form host/path.
			host, p = s[:i], s[i:]
		}
		if j := strings.IndexAny(p, "?#"); j >= 0 {
			p = p[:j]
		}
	}
	if host == "" || strings.ContainsAny(host, " \t@") {
		return "", errors.New("origin: no host found (use a URL such as https://host/owner/repo or user@host:owner/repo)")
	}
	var segs []string
	for seg := range strings.SplitSeq(p, "/") {
		if seg != "" {
			segs = append(segs, seg)
		}
	}
	if n := len(segs); n > 0 {
		if segs[n-1] == ".git" {
			segs = segs[:n-1]
		} else if trimmed := strings.TrimSuffix(segs[n-1], ".git"); trimmed != "" {
			segs[n-1] = trimmed
		}
	}
	if len(segs) == 0 {
		return "", errors.New("origin: no repository path found after the host")
	}
	return strings.ToLower(host + "/" + strings.Join(segs, "/")), nil
}

// isLocalPath reports whether an origin without a scheme is a local path.
func isLocalPath(s string) bool {
	return strings.HasPrefix(s, "/") || strings.HasPrefix(s, `\`) || strings.HasPrefix(s, ".") ||
		strings.HasPrefix(s, "~") || driveRE.MatchString(s)
}

// wslShareRE matches the WSL share prefixes \\wsl$\<distro> and \\wsl.localhost\<distro>
// (after slashes are turned into backslashes).
var wslShareRE = regexp.MustCompile(`(?i)^\\\\(wsl\$|wsl\.localhost)\\([^\\]+)(\\|$)`)

// mntDriveRE matches a Windows drive mounted in WSL: /mnt/<one letter>.
var mntDriveRE = regexp.MustCompile(`^/mnt/([A-Za-z])(/|$)`)

// Folder normalizes a folder path so that one real folder has one value, whether it is given
// from Windows or from WSL. Windows paths are the common form:
//
//   - With wslDistro (the path comes from WSL): /mnt/<letter>/... becomes <letter>:\..., and
//     another absolute path becomes \\wsl.localhost\<distro>\....
//   - A drive path (C:\work, C:/work/) and a UNC path (\\server\share\dir) use backslashes and
//     are lowercased as a whole, since Windows ignores case.
//   - A WSL share (\\wsl$\<distro>\... or \\wsl.localhost\<distro>\...) becomes
//     \\wsl.localhost\<distro>\...; only the part through the distribution name is lowercased,
//     since the rest is a Linux path.
//   - Without wslDistro, an absolute POSIX path (Linux, macOS) is cleaned and keeps its case.
//
// Repeated and trailing separators are removed, and . and .. are resolved lexically. A path
// that is not absolute is an error. wslDistro is used only for this conversion.
func Folder(raw, wslDistro string) (string, error) {
	s := strings.TrimSpace(raw)
	distro := strings.TrimSpace(wslDistro)
	if s == "" {
		return "", nil
	}
	if distro != "" && (strings.ContainsAny(distro, `\/`) || distro == "." || distro == "..") {
		return "", fmt.Errorf("wsl_distro %q is not a distribution name (pass the output of echo $WSL_DISTRO_NAME)", distro)
	}
	switch {
	case strings.HasPrefix(s, `\\`) || strings.HasPrefix(s, "//"):
		return uncPath(strings.ReplaceAll(s, "/", `\`))
	case driveRE.MatchString(s):
		return windowsPath(strings.ToLower(s[:2]), s[2:], true), nil
	case strings.HasPrefix(s, "/"):
		clean := path.Clean(s)
		if distro == "" {
			return clean, nil
		}
		if m := mntDriveRE.FindStringSubmatch(clean); m != nil {
			return windowsPath(strings.ToLower(m[1])+":", clean[len("/mnt/")+1:], true), nil
		}
		return windowsPath(`\\wsl.localhost\`+strings.ToLower(distro), clean, false), nil
	default:
		return "", fmt.Errorf("folder %q is not an absolute path (pass the output of git rev-parse --show-toplevel, or of pwd)", s)
	}
}

// uncPath normalizes a path that starts with two backslashes.
func uncPath(s string) (string, error) {
	if m := wslShareRE.FindStringSubmatch(s); m != nil {
		return windowsPath(`\\wsl.localhost\`+strings.ToLower(m[2]), s[len(m[0])-len(m[3]):], false), nil
	}
	parts := strings.SplitN(strings.TrimLeft(s, `\`), `\`, 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("folder %q is not a complete UNC path (\\\\server\\share\\...)", s)
	}
	rest := ""
	if len(parts) == 3 {
		rest = parts[2]
	}
	return windowsPath(strings.ToLower(`\\`+parts[0]+`\`+parts[1]), rest, true), nil
}

// windowsPath joins prefix (a drive such as c: or a share such as \\server\share) and the
// cleaned segments of rest with backslashes. A drive root keeps its separator (c:\).
func windowsPath(prefix, rest string, lower bool) string {
	var segs []string
	for seg := range strings.FieldsFuncSeq(rest, func(r rune) bool { return r == '\\' || r == '/' }) {
		switch seg {
		case ".":
		case "..":
			if len(segs) > 0 {
				segs = segs[:len(segs)-1]
			}
		default:
			segs = append(segs, seg)
		}
	}
	out := prefix
	if len(segs) > 0 || !strings.HasPrefix(prefix, `\\`) {
		out += `\` + strings.Join(segs, `\`)
	}
	if lower {
		out = strings.ToLower(out)
	}
	return out
}

// Repository decides the repository unit of a session from its normalized values: the origin
// when there is one, otherwise machine and folder joined by a colon (for example
// host-a:c:\work\app), otherwise "" (the session cannot use the repository space). A folder
// without an origin and without a machine is ErrMachineRequired.
//
// A clone with an origin and a copy without one are different units, and a unit is not carried
// over when an origin is added later.
func Repository(origin, machine, folder string) (string, error) {
	switch {
	case origin != "":
		return origin, nil
	case folder == "":
		return "", nil
	case machine == "":
		return "", ErrMachineRequired
	default:
		return machine + ":" + folder, nil
	}
}
