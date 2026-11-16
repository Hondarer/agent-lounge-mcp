package tools

import (
	"errors"
	"sync"
)

// ShutdownMessage is the text of the error every tool call returns while the server stops, and
// of the log notification sent to connected clients when the stop begins.
const ShutdownMessage = "agent-lounge-mcp is shutting down. Messages are kept. " +
	"After the server is started again, call lounge_join with the same session name. " +
	"If the server cannot be reached, tell the user instead of retrying."

// errShuttingDown is returned by every tool call once the server is stopping.
var errShuttingDown = errors.New(ShutdownMessage)

// notifier wakes chat_inbox calls that wait for new messages, in memory: the database is never
// polled.
//
// A waiting call registers a waiter for its session before it checks for unread messages, and
// waits only when the check finds nothing. chat_send wakes the waiters of its recipients after
// the message is stored, so a message stored between the registration and the check is either
// found by the check or wakes the waiter afterwards: none is missed. A wake only means "check
// again": every woken call reads the inbox, the database hands each message to one of them, and
// the others wait on.
//
// Besides new messages, a waiter is woken when its session is joined by another connection (the
// woken call finds its connection no longer joined and returns the takeover error) and when the
// server begins to stop (stopped is closed and the call returns the shutdown error).
type notifier struct {
	mu sync.Mutex
	// waiters holds the registered waiters of each session ID.
	waiters map[int64]map[*waiter]struct{}
	// stopped is closed when the server begins to stop.
	stopped  chan struct{}
	stopOnce sync.Once
}

// waiter is one waiting chat_inbox call.
type waiter struct {
	sessionID int64
	// C receives a value when the call should check its inbox again. It has room for one value,
	// so a wake that comes before the call starts to wait is kept.
	C chan struct{}
}

func newNotifier() *notifier {
	return &notifier{waiters: map[int64]map[*waiter]struct{}{}, stopped: make(chan struct{})}
}

// register adds a waiter for the session. The caller must call unregister when it stops waiting.
func (n *notifier) register(sessionID int64) *waiter {
	w := &waiter{sessionID: sessionID, C: make(chan struct{}, 1)}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.waiters[sessionID] == nil {
		n.waiters[sessionID] = map[*waiter]struct{}{}
	}
	n.waiters[sessionID][w] = struct{}{}
	return w
}

// unregister removes a waiter added by register.
func (n *notifier) unregister(w *waiter) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.waiters[w.sessionID], w)
	if len(n.waiters[w.sessionID]) == 0 {
		delete(n.waiters, w.sessionID)
	}
}

// wake wakes every waiter of the given sessions.
func (n *notifier) wake(sessionIDs ...int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, id := range sessionIDs {
		for w := range n.waiters[id] {
			select {
			case w.C <- struct{}{}:
			default: // already woken and not yet checked
			}
		}
	}
}

// stop switches to the stopping state: from now on every tool call fails with errShuttingDown,
// and waiting calls return it.
func (n *notifier) stop() {
	n.stopOnce.Do(func() { close(n.stopped) })
}

// stopping reports whether stop was called.
func (n *notifier) stopping() bool {
	select {
	case <-n.stopped:
		return true
	default:
		return false
	}
}

// WaitingCalls returns the number of chat_inbox calls waiting for messages now.
func (d *Deps) WaitingCalls() int {
	d.init()
	d.notify.mu.Lock()
	defer d.notify.mu.Unlock()
	n := 0
	for _, ws := range d.notify.waiters {
		n += len(ws)
	}
	return n
}
