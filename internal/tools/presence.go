package tools

import (
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// presence maps MCP connections to the lounge sessions they joined.
//
// lounge_join is the only tool that takes a session name. Every other tool finds its caller
// here, so an agent cannot read another session's inbox by passing a wrong name. The mapping
// lives in memory only: after a reconnect or a server restart a connection is not joined, the
// tools tell the agent to call lounge_join, and joining with the same name continues the
// session with its unread messages.
type presence struct {
	mu sync.Mutex
	// byConn holds the session each connection joined.
	byConn map[*mcp.ServerSession]store.Session
	// bySession holds the connection that owns each session ID. A session has one owner.
	bySession map[int64]*mcp.ServerSession
	// takenOver remembers, for a connection that lost its session to a later join, the name
	// of that session, so its next call can say what happened.
	takenOver map[*mcp.ServerSession]string
	// watched are the connections whose end is awaited to drop their entries.
	watched map[*mcp.ServerSession]bool
}

func newPresence() *presence {
	return &presence{
		byConn:    map[*mcp.ServerSession]store.Session{},
		bySession: map[int64]*mcp.ServerSession{},
		takenOver: map[*mcp.ServerSession]string{},
		watched:   map[*mcp.ServerSession]bool{},
	}
}

// bind makes conn the owner of sess. The connection that owned sess before, if any, is no
// longer joined: when two connections join with one name, the later one takes the session over.
// A session conn joined before is released.
func (p *presence) bind(conn *mcp.ServerSession, sess store.Session) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if prev, ok := p.bySession[sess.ID]; ok && prev != conn {
		delete(p.byConn, prev)
		p.takenOver[prev] = sess.Name
	}
	if old, ok := p.byConn[conn]; ok && old.ID != sess.ID {
		delete(p.bySession, old.ID)
	}
	p.byConn[conn] = sess
	p.bySession[sess.ID] = conn
	delete(p.takenOver, conn)
	if !p.watched[conn] {
		p.watched[conn] = true
		go func() {
			conn.Wait() // returns when the connection ends
			p.release(conn)
		}()
	}
}

// release forgets conn. Its session and unread messages stay in the database.
func (p *presence) release(conn *mcp.ServerSession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if sess, ok := p.byConn[conn]; ok {
		delete(p.byConn, conn)
		if p.bySession[sess.ID] == conn {
			delete(p.bySession, sess.ID)
		}
	}
	delete(p.takenOver, conn)
	delete(p.watched, conn)
}

// lookup returns the session conn joined.
func (p *presence) lookup(conn *mcp.ServerSession) (store.Session, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	sess, ok := p.byConn[conn]
	return sess, ok
}

// caller returns the session conn joined, or an error that tells the agent how to join.
func (p *presence) caller(conn *mcp.ServerSession) (store.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if sess, ok := p.byConn[conn]; ok {
		return sess, nil
	}
	if name, ok := p.takenOver[conn]; ok {
		return store.Session{}, fmt.Errorf("not joined: another connection joined as %q and took the session over. "+
			"Call lounge_join again: with a different name to keep both connections, or with %q to take it back", name, name)
	}
	return store.Session{}, fmt.Errorf("not joined: call lounge_join first. " +
		"Pass the name used before to continue that session and receive its unread messages")
}

// connected reports whether a connection currently owns the session.
func (p *presence) connected(sessionID int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.bySession[sessionID]
	return ok
}
