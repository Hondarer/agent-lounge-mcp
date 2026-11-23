// Package export writes the messages and memories of the lounge as Markdown files, one file
// per conversation and one per scope of memories. It holds the file layout, the file name
// rules and the Markdown form; the data comes from store.Store.Export, and no SQL lives here.
//
// Layout under the output directory:
//
//	chat/<scope>-<scope id>/room-<room>.md            messages to a room
//	chat/<scope>-<scope id>/direct-<A>--<B>.md        messages between two sessions
//	memory/<scope>-<scope id>.md                      the memories of a scope
//
// The shared scope has no identifier, so its names carry no suffix: chat/shared/ holds its
// conversations and memory/shared.md its memories.
//
// A and B are the addresses (user/name) of the two sessions, in address order. Each name is
// passed through SafeName.
package export

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// ChatDir is the directory under the output directory that holds the conversations, and
// MemoryDir the one that holds the memories.
const (
	ChatDir   = "chat"
	MemoryDir = "memory"
)

// maxSafeNameBytes bounds the part of a file name that comes from a name. File systems limit
// a file name to 255 bytes or UTF-16 units, and macOS stores names decomposed, which can
// double the bytes of some scripts: 120 bytes leave room for that, the hash and the extension.
const maxSafeNameBytes = 120

// SafeName returns name as a file name. Unicode letters, digits and combining marks, and
// . _ - are kept, so that a name in any script stays readable; every other character is
// replaced with _. A result longer than maxSafeNameBytes is cut at a character boundary.
// When a character was replaced or the name was cut, a hyphen and the first 8 hexadecimal
// digits of the SHA-256 hash of the original name are appended, so that two names that
// differ only in what was replaced or cut (such as a/b and a:b) do not give the same file
// name.
func SafeName(name string) string {
	var b strings.Builder
	changed := false
	for _, r := range name {
		keep := unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '.' || r == '_' || r == '-'
		if !keep {
			r, changed = '_', true
		}
		if b.Len()+utf8.RuneLen(r) > maxSafeNameBytes {
			changed = true
			break
		}
		b.WriteRune(r)
	}
	if !changed {
		return b.String()
	}
	sum := sha256.Sum256([]byte(name))
	return b.String() + "-" + hex.EncodeToString(sum[:])[:8]
}

// address is how the export names a session: user/name.
func address(user, name string) string {
	return user + "/" + name
}

// conversation identifies the file of a message.
type conversation struct {
	scope store.Scope
	room  string
	// a and b are the addresses of the two sessions of a message to a session, a < b.
	a, b string
}

func conversationOf(m store.ExportMessage) conversation {
	c := conversation{scope: m.Scope, room: m.Room}
	if m.Room == "" {
		c.a, c.b = address(m.FromUser, m.From), address(m.ToUser, m.ToName)
		if c.b < c.a {
			c.a, c.b = c.b, c.a
		}
	}
	return c
}

// path returns the path of the file of the conversation, relative to the output directory.
func (c conversation) path() string {
	dir := scopeName(c.scope)
	var file string
	if c.room != "" {
		file = SafeName("room-" + c.room)
	} else {
		file = SafeName("direct-" + c.a + "--" + c.b)
	}
	return filepath.Join(ChatDir, dir, file+".md")
}

// Summary counts what WriteMarkdown wrote.
type Summary struct {
	Files    int
	Messages int
	Memories int
}

// Reader passes the data to export to its two callbacks, as store.Store.Export does: the
// messages of one conversation one after another, oldest first, then the memories of one scope
// one after another.
type Reader func(onMessage func(store.ExportMessage) error, onMemory func(store.Memory) error) error

// WriteMarkdown writes what read passes into Markdown files under dir, which is created when
// missing. A file of the same name is overwritten; other files in dir are left alone. The
// memory directory is created only when there is a memory.
func WriteMarkdown(dir string, read Reader) (Summary, error) {
	var sum Summary
	var cur *os.File
	var w *bufio.Writer
	var curConv conversation
	var curScope store.Scope
	written := map[string]bool{}
	// open starts writing the file rel, or continues it when it was written before in this run
	// (a group should come in one run; should it come again, its file is continued rather than
	// overwritten). It reports whether the file is new.
	open := func(rel string) (bool, error) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return false, err
		}
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		if written[rel] {
			flags = os.O_WRONLY | os.O_APPEND
		}
		f, err := os.OpenFile(path, flags, 0o644)
		if err != nil {
			return false, err
		}
		cur, w = f, bufio.NewWriter(f)
		if written[rel] {
			return false, nil
		}
		written[rel] = true
		sum.Files++
		return true, nil
	}
	closeCur := func() error {
		if cur == nil {
			return nil
		}
		err := w.Flush()
		if cerr := cur.Close(); err == nil {
			err = cerr
		}
		cur = nil
		return err
	}
	inMemories := false
	err := read(func(m store.ExportMessage) error {
		c := conversationOf(m)
		if cur == nil || c != curConv {
			if err := closeCur(); err != nil {
				return err
			}
			isNew, err := open(c.path())
			if err != nil {
				return err
			}
			curConv = c
			if isNew {
				writeHeader(w, c)
			}
		}
		writeMessage(w, m)
		sum.Messages++
		return nil
	}, func(m store.Memory) error {
		if !inMemories || m.Scope != curScope {
			if err := closeCur(); err != nil {
				return err
			}
			isNew, err := open(memoryPath(m.Scope))
			if err != nil {
				return err
			}
			inMemories, curScope = true, m.Scope
			if isNew {
				writeMemoryHeader(w, m.Scope)
			}
		}
		writeMemory(w, m)
		sum.Memories++
		return nil
	})
	if cerr := closeCur(); err == nil {
		err = cerr
	}
	if err != nil {
		return sum, fmt.Errorf("export: %w", err)
	}
	return sum, nil
}

// memoryPath returns the path of the file of the memories of a scope, relative to the output
// directory.
func memoryPath(sc store.Scope) string {
	return filepath.Join(MemoryDir, scopeName(sc)+".md")
}

// scopeName returns the file or folder name of a scope: <scope>-<scope id>, or only the scope
// for the shared scope, which has no identifier.
func scopeName(sc store.Scope) string {
	if sc.Kind == store.ScopeShared {
		return SafeName(sc.Kind)
	}
	return SafeName(sc.Kind + "-" + sc.ID)
}

func writeMemoryHeader(w io.Writer, sc store.Scope) {
	fmt.Fprintln(w, "# agent-lounge memories")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Scope: %s\n", code(sc.Kind))
	fmt.Fprintf(w, "- Scope ID: %s\n", code(sc.ID))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Memories are written by the sessions named with them. Times are UTC.")
}

func writeMemory(w io.Writer, m store.Memory) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "## Memory %d", m.ID)
	if m.Title != "" {
		fmt.Fprintf(w, ": %s", code(m.Title))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	if len(m.Tags) > 0 {
		tags := make([]string, len(m.Tags))
		for i, t := range m.Tags {
			tags[i] = code(t)
		}
		fmt.Fprintf(w, "- Tags: %s\n", strings.Join(tags, ", "))
	} else {
		fmt.Fprintln(w, "- Tags: none")
	}
	fmt.Fprintf(w, "- Pinned: %s\n", map[bool]string{true: "yes", false: "no"}[m.Pinned])
	fmt.Fprintf(w, "- Created: %s by %s\n", formatTime(m.CreatedAt), author(m.CreatedBy))
	fmt.Fprintf(w, "- Updated: %s by %s\n", formatTime(m.UpdatedAt), author(m.UpdatedBy))
	if m.AccessedAt.IsZero() {
		fmt.Fprintln(w, "- Last referenced: never")
	} else {
		fmt.Fprintf(w, "- Last referenced: %s\n", formatTime(m.AccessedAt))
	}
	fmt.Fprintf(w, "- References: %d\n", m.AccessCount)
	fmt.Fprintln(w)
	fence := Fence(m.Body)
	fmt.Fprintln(w, fence+"text")
	fmt.Fprintln(w, m.Body)
	fmt.Fprintln(w, fence)
}

// author renders who wrote a memory: user/session, with the agent name when there is one.
func author(a store.Author) string {
	s := code(address(a.User, a.Session))
	if a.Agent != "" {
		s += " (" + code(a.Agent) + ")"
	}
	return s
}

func writeHeader(w io.Writer, c conversation) {
	fmt.Fprintln(w, "# agent-lounge conversation")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Scope: %s\n", code(c.scope.Kind))
	fmt.Fprintf(w, "- Scope ID: %s\n", code(c.scope.ID))
	if c.room != "" {
		fmt.Fprintf(w, "- Room: %s\n", code(c.room))
	} else {
		fmt.Fprintf(w, "- Between: %s and %s\n", code(c.a), code(c.b))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Messages are written by the sending sessions. Times are UTC.")
}

func writeMessage(w io.Writer, m store.ExportMessage) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "## %s %s", formatTime(m.SentAt), code(address(m.FromUser, m.From)))
	if m.FromAgent != "" {
		fmt.Fprintf(w, " (%s)", code(m.FromAgent))
	}
	fmt.Fprintf(w, " message %d\n\n", m.ID)
	if len(m.Deliveries) == 0 {
		fmt.Fprintln(w, "Deliveries: none")
	} else {
		fmt.Fprintln(w, "Deliveries:")
		fmt.Fprintln(w)
		for _, d := range m.Deliveries {
			fmt.Fprintf(w, "- %s: %s %s\n", code(address(d.User, d.Name)), d.State, formatTime(stateTime(d)))
		}
	}
	fmt.Fprintln(w)
	fence := Fence(m.Body)
	fmt.Fprintln(w, fence+"text")
	fmt.Fprintln(w, m.Body)
	fmt.Fprintln(w, fence)
}

// stateTime returns the time a delivery reached its state.
func stateTime(d store.ExportDelivery) time.Time {
	switch d.State {
	case store.StateDelivered:
		return d.DeliveredAt
	case store.StateAcked:
		return d.AckedAt
	case store.StateExpired:
		return d.ExpiredAt
	default:
		return d.QueuedAt
	}
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// longestRun returns the length of the longest run of c in s.
func longestRun(s string, c rune) int {
	longest, run := 0, 0
	for _, r := range s {
		if r == c {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return longest
}

// Fence returns the backtick fence of a code block that holds body: at least three backticks,
// and more than the longest run of backticks in body, so that nothing in the body ends the
// block.
func Fence(body string) string {
	return strings.Repeat("`", max(3, longestRun(body, '`')+1))
}

// code renders s as an inline code span whose delimiter is longer than any run of backticks in
// s, so that names cannot break the Markdown around them.
func code(s string) string {
	delim := strings.Repeat("`", longestRun(s, '`')+1)
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return delim + " " + s + " " + delim
	}
	return delim + s + delim
}
