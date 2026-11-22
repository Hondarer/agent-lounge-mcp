package profile

import (
	"errors"
	"strings"
	"testing"
)

func TestOSUser(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"honda", "honda"},
		{"  Honda\r\n", "Honda"},
		{`CORP\Honda`, `CORP\Honda`},
		{"", ""},
		{"   ", ""},
		{strings.Repeat("あ", MaxNameLen), strings.Repeat("あ", MaxNameLen)},
	} {
		got, err := OSUser(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("OSUser(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
	if _, err := OSUser(strings.Repeat("a", MaxNameLen+1)); err == nil {
		t.Error("OSUser accepted a name longer than the limit")
	}
}

func TestMachine(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"HOST-A\r\n", "host-a"},
		{" MacBook-Pro.local ", "macbook-pro.local"},
		{"", ""},
	} {
		got, err := Machine(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Machine(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
	if _, err := Machine(strings.Repeat("a", MaxNameLen+1)); err == nil {
		t.Error("Machine accepted a name longer than the limit")
	}
}

func TestBranch(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"main", "main"},
		{"  Feature/Foo-Bar  \r\n", "feature/foo-bar"},
		{"warning line\n  Release/1.0 \n\n", "release/1.0"},
		{"", ""},
		{"   \r\n\t ", ""},
		{"feature/日本語", "feature/日本語"},
		{"a@b", "a@b"},
		{"a#b", "a#b"},
		{strings.Repeat("a", MaxBranchLen), strings.Repeat("a", MaxBranchLen)},
	} {
		got, err := Branch(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Branch(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
	for _, in := range []string{
		strings.Repeat("a", MaxBranchLen+1),
		"a:b",
		"a b",
		"a\tb",
		"a\x00b",
		"a~1",
		"a^2",
		"a?b",
		"a*b",
		"a[b",
		`a\b`,
		"* main",
		"HEAD",
		"head",
		"  HeAd \n",
	} {
		got, err := Branch(in)
		if err == nil {
			t.Errorf("Branch(%q) = %q, want an error", in, got)
		} else if !strings.Contains(err.Error(), "git branch --show-current") {
			t.Errorf("Branch(%q) error %q does not mention git branch --show-current", in, err.Error())
		}
	}
}

func TestOrigin(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		// The examples of the specification.
		{"https://github.com/Owner/Repo.git", "github.com/owner/repo"},
		{"git@github.com:owner/repo.git", "github.com/owner/repo"},
		{"ssh://git@github.com:22/owner/repo", "github.com/owner/repo"},
		{"https://user:token@github.com/owner/repo/", "github.com/owner/repo"},
		{"github.com/owner/repo", "github.com/owner/repo"},
		// Each accepted scheme.
		{"http://gitlab.example.com/group/sub/repo.git", "gitlab.example.com/group/sub/repo"},
		{"git://example.org/repo.git", "example.org/repo"},
		{"git+ssh://git@example.org/owner/repo.git", "example.org/owner/repo"},
		{"SSH://git@GitHub.com/Owner/Repo", "github.com/owner/repo"},
		// Surrounding white space, port, query, fragment, trailing slashes and .git.
		{"  https://github.com/owner/repo.git\r\n", "github.com/owner/repo"},
		{"https://github.com:443/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo.git?ref=main#readme", "github.com/owner/repo"},
		{"https://github.com/owner/repo.git/", "github.com/owner/repo"},
		{"https://github.com/owner/repo//", "github.com/owner/repo"},
		{"https://github.com//owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo/.git", "github.com/owner/repo"},
		{"https://oauth2:ghp_SECRET@github.com/owner/repo.git", "github.com/owner/repo"},
		// scp-like form without user, with a path from the root, and with credentials.
		{"github.com:Owner/Repo", "github.com/owner/repo"},
		{"git@github.com:/owner/repo.git", "github.com/owner/repo"},
		{"user:secret@github.com:owner/repo.git", "github.com/owner/repo"},
		{"github.com/owner/repo.git?x=1#y", "github.com/owner/repo"},
		{"git@github.com/owner/repo", "github.com/owner/repo"},
		{"https://[::1]:8080/owner/repo", "[::1]/owner/repo"},
		// Hostless origins are no origin.
		{"", ""},
		{"   ", ""},
		{"/srv/git/repo.git", ""},
		{"../other", ""},
		{"./repo", ""},
		{"~/repo.git", ""},
		{`C:\git\repo`, ""},
		{"C:/git/repo", ""},
		{"C:repo", ""},
		{`\\server\share\repo`, ""},
		{"file:///srv/git/repo.git", ""},
		{"FILE://host/srv/repo", ""},
	} {
		got, err := Origin(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Origin(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

// The normalized form host/path is read as a host only when its first segment looks like a
// host name; another value without a scheme, a colon or a user is a relative local path.
func TestOriginRelativePath(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"github.com/owner/repo", "github.com/owner/repo"},
		{"localhost/owner/repo", "localhost/owner/repo"},
		{"LocalHost/Owner/Repo", "localhost/owner/repo"},
		{"foo/bar", ""},
		{"gitserver/team/repo", ""},
		{"foo", ""},
		{"repo", ""},
		{"repo.git", ""},
		{"my.app", ""},
		{"https://gitserver/team/repo", "gitserver/team/repo"},
		{"git@gitserver:team/repo", "gitserver/team/repo"},
		{"gitserver:team/repo", "gitserver/team/repo"},
		{"user@host/path", "host/path"},
	} {
		got, err := Origin(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Origin(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestOriginErrors(t *testing.T) {
	for _, in := range []string{
		"https://github.com",       // no path
		"https://github.com/.git",  // no path left
		"https:///owner/repo",      // no host
		"ftp://example.org/repo",   // unsupported scheme
		"https://user:to%zz@h/o/r", // cannot be parsed
		"git@:owner/repo",          // empty host
		"has space:owner/repo",     // not a host
	} {
		if got, err := Origin(in); err == nil {
			t.Errorf("Origin(%q) = %q, want an error", in, got)
		}
	}
}

// A credentialed origin never appears in an error.
func TestOriginErrorsHideCredentials(t *testing.T) {
	for _, in := range []string{
		"https://user:SECRET@github.com",
		"https://user:SECRET%zz@github.com/o/r",
		"https://user:SECRET#x@github.com/o/r",
		"weird!scheme://user:SECRET@host/o/r",
		"user:SECRET@host",
	} {
		_, err := Origin(in)
		if err == nil {
			t.Errorf("Origin(%q) succeeded, want an error", in)
			continue
		}
		if strings.Contains(err.Error(), "SECRET") {
			t.Errorf("Origin(%q) error quotes the credentials: %v", in, err)
		}
	}
}

// A Windows drive seen through the WSL share is the drive path.
func TestFolderShareDrive(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`\\wsl.localhost\Ubuntu\mnt\c\Work\App`, `c:\work\app`},
		{`//wsl.localhost/Ubuntu/mnt/c/Work/App/`, `c:\work\app`},
		{`\\wsl$\Ubuntu\mnt\d`, `d:\`},
		{`\\wsl.localhost\Ubuntu\mnt\data\x`, `\\wsl.localhost\ubuntu\mnt\data\x`},
		{`\\wsl.localhost\Ubuntu\home\..\mnt\c\x`, `c:\x`},
		{`\\wsl.localhost\Ubuntu\mnt`, `\\wsl.localhost\ubuntu\mnt`},
		{`\\wsl.localhost\Ubuntu`, `\\wsl.localhost\ubuntu`},
		{`\\wsl.localhost\Ubuntu\home\User\App`, `\\wsl.localhost\ubuntu\home\User\App`},
	} {
		got, err := Folder(tc.in, "")
		if err != nil || got != tc.want {
			t.Errorf("Folder(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
	// The same folder declared from inside WSL gives the same value.
	fromWSL, _ := Folder("/mnt/c/Work/App", "Ubuntu")
	fromShare, _ := Folder(`\\wsl.localhost\Ubuntu\mnt\c\Work\App`, "")
	if fromWSL != fromShare {
		t.Errorf("from WSL %q, from the share %q", fromWSL, fromShare)
	}
}

func TestFolder(t *testing.T) {
	for _, tc := range []struct{ in, distro, want string }{
		// The rows of the specification.
		{"/mnt/c/work/App", "Ubuntu", `c:\work\app`},
		{"/home/user/app", "Ubuntu", `\\wsl.localhost\ubuntu\home\user\app`},
		{`C:\work\App`, "", `c:\work\app`},
		{"C:/work/App/", "", `c:\work\app`},
		{`\\wsl$\Ubuntu\home\user\app`, "", `\\wsl.localhost\ubuntu\home\user\app`},
		{`\\wsl.localhost\Ubuntu\home\user\app`, "", `\\wsl.localhost\ubuntu\home\user\app`},
		{`\\server\share\Dir`, "", `\\server\share\dir`},
		{"/home/user/app/", "", "/home/user/app"},
		// WSL: drive roots, other mounts, case of the Linux part, cleaning.
		{"/mnt/c", "Ubuntu", `c:\`},
		{"/mnt/D/Data/", "Ubuntu", `d:\data`},
		{"/mnt/cdrom/x", "Ubuntu", `\\wsl.localhost\ubuntu\mnt\cdrom\x`},
		{"/home/User/App//src/../", "Ubuntu-24.04", `\\wsl.localhost\ubuntu-24.04\home\User\App`},
		{"/", "Ubuntu", `\\wsl.localhost\ubuntu`},
		{" /home/user/app\n", " Ubuntu\r\n", `\\wsl.localhost\ubuntu\home\user\app`},
		// Windows: drive roots, repeated separators, dot segments, the distro is ignored.
		{"C:", "", `c:\`},
		{`C:\`, "", `c:\`},
		{`c:\\work\\.\App\..\App2\`, "", `c:\work\app2`},
		{`C:\work\App`, "Ubuntu", `c:\work\app`},
		// WSL shares: only the prefix through the distribution is lowercased.
		{`\\WSL$\Ubuntu\Home\User\App\`, "", `\\wsl.localhost\ubuntu\Home\User\App`},
		{`//wsl.localhost/Ubuntu/home/user/app`, "", `\\wsl.localhost\ubuntu\home\user\app`},
		{`\\wsl.localhost\Ubuntu`, "", `\\wsl.localhost\ubuntu`},
		// Other UNC paths, also with forward slashes (git rev-parse on Windows).
		{`\\Server\Share`, "", `\\server\share`},
		{"//Server/Share/Dir/", "", `\\server\share\dir`},
		// Native POSIX: cleaned, case kept.
		{"/Users/Honda/Work//App/./", "", "/Users/Honda/Work/App"},
		{"/a/b/../c", "", "/a/c"},
		{"/", "", "/"},
		{"", "Ubuntu", ""},
	} {
		got, err := Folder(tc.in, tc.distro)
		if err != nil || got != tc.want {
			t.Errorf("Folder(%q, %q) = %q, %v; want %q", tc.in, tc.distro, got, err, tc.want)
		}
	}
}

func TestFolderErrors(t *testing.T) {
	for _, tc := range []struct{ in, distro string }{
		{"work/app", ""},
		{"./app", ""},
		{"~/app", ""},
		{"C:work", ""},
		{"app", "Ubuntu"},
		{`\\server`, ""},
		{`\\server\`, ""},
		{"/home/user/app", `Ub\untu`},
		{"/home/user/app", "a/b"},
	} {
		if got, err := Folder(tc.in, tc.distro); err == nil {
			t.Errorf("Folder(%q, %q) = %q, want an error", tc.in, tc.distro, got)
		}
	}
}

// One real folder gives one value, whether the agent runs on Windows or in WSL.
func TestFolderSameFromWindowsAndWSL(t *testing.T) {
	for _, group := range [][][2]string{
		{
			{"/mnt/c/work/App", "Ubuntu"},
			{`C:\work\App`, ""},
			{"C:/work/App/", ""},
			{`c:\WORK\app`, ""},
		},
		{
			{"/home/user/app", "Ubuntu"},
			{`\\wsl$\Ubuntu\home\user\app`, ""},
			{`\\wsl.localhost\UBUNTU\home\user\app\`, ""},
			{"//wsl.localhost/Ubuntu/home/user/app", ""},
		},
	} {
		var first string
		for i, in := range group {
			got, err := Folder(in[0], in[1])
			if err != nil {
				t.Fatalf("Folder(%q, %q): %v", in[0], in[1], err)
			}
			if i == 0 {
				first = got
			} else if got != first {
				t.Errorf("Folder(%q, %q) = %q, want %q (same as %q)", in[0], in[1], got, first, group[0][0])
			}
		}
	}
	// Different Linux case is a different folder; different distributions are different folders.
	a, _ := Folder("/home/user/App", "Ubuntu")
	b, _ := Folder("/home/user/app", "Ubuntu")
	c, _ := Folder("/home/user/app", "Debian")
	if a == b || b == c {
		t.Errorf("distinct folders share a value: %q %q %q", a, b, c)
	}
}

func TestRepository(t *testing.T) {
	for _, tc := range []struct{ origin, machine, folder, want string }{
		{"github.com/owner/repo", "host-a", `c:\work\app`, "github.com/owner/repo"},
		{"github.com/owner/repo", "", "", "github.com/owner/repo"},
		{"", "host-a", `c:\work\app`, `host-a:c:\work\app`},
		{"", "host-a", "/home/user/app", "host-a:/home/user/app"},
		{"", "host-a", "", ""},
		{"", "", "", ""},
	} {
		got, err := Repository(tc.origin, tc.machine, tc.folder)
		if err != nil || got != tc.want {
			t.Errorf("Repository(%q, %q, %q) = %q, %v; want %q", tc.origin, tc.machine, tc.folder, got, err, tc.want)
		}
	}
	if _, err := Repository("", "", `c:\work\app`); !errors.Is(err, ErrMachineRequired) {
		t.Errorf("folder without machine: err = %v, want ErrMachineRequired", err)
	}
}

func TestUnexpandedVariable(t *testing.T) {
	for _, v := range []string{"%USERNAME%", "$USER", "${USER}", "$env:USERNAME", " %COMPUTERNAME%\r\n"} {
		if _, err := OSUser(v); err == nil {
			t.Errorf("OSUser(%q): no error", v)
		}
		if _, err := Machine(v); err == nil {
			t.Errorf("Machine(%q): no error", v)
		}
	}
	for _, v := range []string{"alice", `CORP\alice`, "a%b", "user$"} {
		if got, err := OSUser(v); err != nil || got != v {
			t.Errorf("OSUser(%q) = %q, %v", v, got, err)
		}
	}
}

func TestMachineRejectsGenericNames(t *testing.T) {
	for _, v := range []string{"sandbox", "Sandbox", "LOCALHOST", "localhost.localdomain", " Localhost.LocalDomain\r\n"} {
		_, err := Machine(v)
		if err == nil {
			t.Errorf("Machine(%q): no error", v)
			continue
		}
		if !strings.Contains(err.Error(), "does not identify a machine") || !strings.Contains(err.Error(), "cat /etc/hostname") {
			t.Errorf("Machine(%q): error %q", v, err)
		}
	}
	if got, err := Machine("sandbox-1"); err != nil || got != "sandbox-1" {
		t.Errorf("Machine(sandbox-1) = %q, %v", got, err)
	}
}

func TestLastLine(t *testing.T) {
	in := "'\\\\wsl.localhost\\Ubuntu\\home\\u'\r\nCMD.EXE was started with the above path as the current directory.\r\nUNC paths are not supported.\r\nHOST-A\r\n\r\n"
	if got, err := Machine(in); err != nil || got != "host-a" {
		t.Errorf("Machine = %q, %v", got, err)
	}
	if got, err := OSUser(strings.ReplaceAll(in, "HOST-A", `CORP\alice`)); err != nil || got != `CORP\alice` {
		t.Errorf("OSUser = %q, %v", got, err)
	}
}
