// Package export writes the messages of the lounge as Markdown files, one file per
// conversation. It holds the file layout, the file name rules and the Markdown form; the
// messages come from store.Store.Export, and no SQL lives here.
//
// Layout under the output directory:
//
//	chat/<scope>-<scope id>/room-<room>.md            messages to a room
//	chat/<scope>-<scope id>/direct-<A>--<B>.md        messages between two sessions
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

	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// ChatDir is the directory under the output directory that holds the conversations.
const ChatDir = "chat"

// SafeName returns name as a file name: every character other than A-Z a-z 0-9 . _ - is
// replaced with _. When a character is replaced, a hyphen and the first 8 hexadecimal digits
// of the SHA-256 hash of the original name are appended, so that two names that differ only in
// replaced characters (such as a/b and a:b) do not give the same file name.
func SafeName(name string) string {
	var b strings.Builder
	replaced := false
	for _, r := range name {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
			replaced = true
		}
	}
	if !replaced {
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
	dir := SafeName(c.scope.Kind + "-" + c.scope.ID)
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
}

// WriteMarkdown writes the messages that read passes to its callback into Markdown files under
// dir, which is created when missing. read must pass the messages of one conversation one after
// another, oldest first (store.Store.Export does). A file of the same name is overwritten;
// other files in dir are left alone.
func WriteMarkdown(dir string, read func(fn func(store.ExportMessage) error) error) (Summary, error) {
	var sum Summary
	var cur *os.File
	var w *bufio.Writer
	var curConv conversation
	written := map[string]bool{}
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
	err := read(func(m store.ExportMessage) error {
		c := conversationOf(m)
		if cur == nil || c != curConv {
			if err := closeCur(); err != nil {
				return err
			}
			rel := c.path()
			path := filepath.Join(dir, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			// A conversation comes in one run; should it come again, its file is continued
			// rather than overwritten.
			flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			if written[rel] {
				flags = os.O_WRONLY | os.O_APPEND
			}
			f, err := os.OpenFile(path, flags, 0o644)
			if err != nil {
				return err
			}
			cur, w, curConv = f, bufio.NewWriter(f), c
			if !written[rel] {
				written[rel] = true
				sum.Files++
				writeHeader(w, c)
			}
		}
		writeMessage(w, m)
		sum.Messages++
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
