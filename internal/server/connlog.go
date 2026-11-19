package server

import (
	"context"
	"log"
	"strings"
	"sync"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxClientInfoRunes bounds the client name and version in the connection diagnostics.
const maxClientInfoRunes = 64

// connLog writes the connection diagnostics of --log-connections: which client connected,
// and which methods and tools it called. They let a developer tell, for each agent, which
// protocol it connected with and when it called which tool.
//
// The lines name a connection by a number counted from 1 since the server started, never by
// its MCP session ID, which would let anyone reading the output take over the connection.
// They hold no tool arguments or results, so no message, memory, token, user or session name.
type connLog struct {
	server *mcp.Server
	logf   func(format string, args ...any)

	mu   sync.Mutex
	next int
	// conns holds the state of each connection. Entries of connections that are gone are
	// dropped when a new connection initializes. The key is the session, not its ID: a
	// connection over another transport than HTTP has no ID.
	conns map[*mcp.ServerSession]*connState
}

type connState struct {
	number    int
	described bool
}

func newConnLog(s *mcp.Server, logf func(string, ...any)) *connLog {
	if logf == nil {
		logf = log.Printf
	}
	return &connLog{server: s, logf: logf, conns: map[*mcp.ServerSession]*connState{}}
}

// clientRequest is implemented by every request the server receives. Under protocol versions
// that have an initialize handshake, the client and protocol come from it; under later ones
// (2026-07-28 and after), from the metadata of each request.
type clientRequest interface {
	ClientInfo() *mcp.Implementation
	ProtocolVersion() string
}

// middleware records the requests of every connection. The client of a connection is written
// once, after the first request that names it (initialize, server/discover, or any request
// of a later protocol) was handled. A client that first asks with server/discover under a
// protocol the server cannot serve on this transport, and then connects again with initialize,
// is written as two connections: the first line shows the protocol the client asked for.
func (c *connLog) middleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ss, ok := req.GetSession().(*mcp.ServerSession)
			if !ok {
				return next(ctx, method, req)
			}
			// Taken before the request is handled: the connection may be gone, and its state
			// dropped, by the time a long request returns.
			st := c.state(ss)
			switch method {
			case "initialize", "server/discover":
				// Written as the line of the client below.
			case "tools/call":
				name := ""
				switch p := req.GetParams().(type) {
				case *mcp.CallToolParamsRaw:
					if p != nil {
						name = p.Name
					}
				case *mcp.CallToolParams:
					if p != nil {
						name = p.Name
					}
				}
				c.logf("connection %d: request tools/call %s", c.number(ss), clean(name))
			default:
				// Notifications are not written.
				if !strings.HasPrefix(method, "notifications/") {
					c.logf("connection %d: request %s", c.number(ss), clean(method))
				}
			}
			res, err := next(ctx, method, req)
			if err == nil {
				c.describe(st, req)
			}
			return res, err
		}
	}
}

// describe writes the client of a connection, once.
func (c *connLog) describe(st *connState, req mcp.Request) {
	r, ok := req.(clientRequest)
	if !ok {
		return
	}
	info := r.ClientInfo()
	if info == nil {
		return
	}
	c.mu.Lock()
	if st.described {
		c.mu.Unlock()
		return
	}
	st.described = true
	n := st.number
	c.mu.Unlock()
	c.logf("connection %d: initialized client=%q version=%q protocol=%s",
		n, clean(info.Name), clean(info.Version), clean(r.ProtocolVersion()))
}

func (c *connLog) state(ss *mcp.ServerSession) *connState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stateLocked(ss)
}

func (c *connLog) number(ss *mcp.ServerSession) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stateLocked(ss).number
}

// stateLocked returns the state of a connection, numbering it when it is new. c.mu is held.
func (c *connLog) stateLocked(ss *mcp.ServerSession) *connState {
	if st, ok := c.conns[ss]; ok {
		return st
	}
	c.pruneLocked()
	c.next++
	st := &connState{number: c.next}
	c.conns[ss] = st
	return st
}

// pruneLocked drops the state of connections the server no longer has. c.mu is held.
func (c *connLog) pruneLocked() {
	live := map[*mcp.ServerSession]bool{}
	for ss := range c.server.Sessions() {
		live[ss] = true
	}
	for ss := range c.conns {
		if !live[ss] {
			delete(c.conns, ss)
		}
	}
}

// clean makes a value sent by the client safe to write on one line: it drops control
// characters, format characters (such as those that change the direction of text), line and
// paragraph separators and quotes, and keeps at most maxClientInfoRunes characters.
func clean(v string) string {
	var b strings.Builder
	n := 0
	for _, r := range v {
		if unicode.IsControl(r) || unicode.In(r, unicode.Zl, unicode.Zp, unicode.Cf) || r == '"' || r == '\'' || r == '`' || r == unicode.ReplacementChar {
			continue
		}
		if n == maxClientInfoRunes {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
