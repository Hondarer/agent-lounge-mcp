package connection

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const invalidToken = "the token is invalid; restart the daemon with a new token"

var errUnauthorized = errors.New(invalidToken)
var errUnavailable = errors.New("cannot connect to the MCP server")
var errUnknown = errors.New("connection to the MCP server was lost; the outcome is unknown")

type requestKey struct{}

type observation struct {
	status  atomic.Int32
	gotConn atomic.Bool
	cancel  context.CancelCauseFunc
}

func observe(ctx context.Context) (context.Context, *observation, func()) {
	ctx, cancel := context.WithCancelCause(ctx)
	o := &observation{cancel: cancel}
	ctx = context.WithValue(ctx, requestKey{}, o)
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { o.gotConn.Store(true) },
	}), o, func() { cancel(nil) }
}

type upstream struct {
	url     string
	token   string
	http    *http.Client
	invalid atomic.Bool
}

func newUpstream(url, token string) *upstream {
	u := &upstream{url: url, token: token}
	u.http = &http.Client{
		Transport:     u,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return u
}

func (u *upstream) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet {
		return nil, errResumeDisabled
	}
	req = req.Clone(req.Context())
	if u.token != "" {
		req.Header.Set("Authorization", "Bearer "+u.token)
	}
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		if o, ok := req.Context().Value(requestKey{}).(*observation); ok && req.Method == http.MethodPost {
			if resp.StatusCode == http.StatusNotFound && req.Header.Get("Mcp-Session-Id") == "" {
				return resp, nil
			}
			o.status.Store(int32(resp.StatusCode))
			if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
				if id := requestID(req); id != nil {
					resp.Body = &responseStream{ReadCloser: resp.Body, reader: bufio.NewReader(resp.Body), id: id,
						onLoss: func() { o.cancel(errUnknown) }}
				}
			}
		}
		if resp.StatusCode == http.StatusUnauthorized {
			u.invalid.Store(true)
		}
	}
	return resp, err
}

func (u *upstream) connect(ctx context.Context) (*mcp.ClientSession, error) {
	if u.invalid.Load() {
		return nil, errUnauthorized
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "agent-lounge-mcp CLI", Version: version.Version}, &mcp.ClientOptions{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	s, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: u.url, HTTPClient: u.http, MaxRetries: -1, DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		if u.invalid.Load() {
			return nil, errUnauthorized
		}
		return nil, errUnavailable
	}
	return s, nil
}

type participant struct {
	joinMu   sync.Mutex
	mu       sync.Mutex
	session  *mcp.ClientSession
	join     json.RawMessage
	lastJoin string
}

func (p *participant) get(ctx context.Context, u *upstream) (*mcp.ClientSession, Reply) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx.Err() != nil {
		return nil, failure(3, errUnavailable)
	}
	if u.invalid.Load() {
		return nil, failure(3, errUnauthorized)
	}
	if p.session != nil {
		return p.session, Reply{}
	}
	s, err := u.connect(ctx)
	if err != nil {
		return nil, failure(3, err)
	}
	if p.join != nil {
		callCtx, _, stop := observe(ctx)
		result, err := s.CallTool(callCtx, &mcp.CallToolParams{Name: "lounge_join", Arguments: p.join})
		stop()
		if err != nil || result.IsError {
			_ = s.Close()
			if u.invalid.Load() {
				return nil, failure(3, errUnauthorized)
			}
			if err != nil {
				return nil, failure(3, errUnknown)
			}
			return nil, toolReply(result)
		}
	}
	p.session = s
	return s, Reply{}
}

func (p *participant) forget(s *mcp.ClientSession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.session == s {
		p.session = nil
		_ = s.Close()
	}
}

func (p *participant) remember(args json.RawMessage, result *mcp.CallToolResult) {
	var joined struct{ Name string }
	b, err := json.Marshal(result.StructuredContent)
	if err != nil || json.Unmarshal(b, &joined) != nil || joined.Name == "" {
		return
	}
	p.mu.Lock()
	p.join = args
	p.lastJoin = joined.Name
	p.mu.Unlock()
}

func failure(code int, err error) Reply {
	return Reply{Code: code, Error: err.Error()}
}

func toolReply(result *mcp.CallToolResult) Reply {
	if result.IsError {
		var message []string
		for _, content := range result.Content {
			if text, ok := content.(*mcp.TextContent); ok {
				message = append(message, text.Text)
			}
		}
		if len(message) != 0 {
			return Reply{Code: 1, Error: strings.Join(message, "\n")}
		}
	}
	b, err := json.Marshal(result)
	if err != nil {
		return failure(1, errors.New("cannot encode tool result"))
	}
	if result.IsError {
		return Reply{Code: 1, Error: string(b)}
	}
	return Reply{Result: b}
}

// Replay only a request refused before execution. Repeating a send can duplicate
// a message, and repeating an inbox read can lose messages already marked delivered.
func (p *participant) call(ctx context.Context, u *upstream, req Request) Reply {
	if req.Tool == "lounge_join" {
		p.joinMu.Lock()
		defer p.joinMu.Unlock()
	}
	for attempt := 0; attempt < 2; attempt++ {
		s, reply := p.get(ctx, u)
		if s == nil {
			return reply
		}
		callCtx, observed, stop := observe(ctx)
		result, err := s.CallTool(callCtx, &mcp.CallToolParams{Name: req.Tool, Arguments: req.Args})
		stop()
		if err == nil {
			if req.Tool == "lounge_join" && !result.IsError {
				p.remember(req.Args, result)
			}
			return toolReply(result)
		}
		if u.invalid.Load() {
			return failure(3, errUnauthorized)
		}
		if ctx.Err() != nil {
			return failure(3, ctx.Err())
		}
		if observed.status.Load() == http.StatusNotFound ||
			(!observed.gotConn.Load() && errors.Is(err, mcp.ErrSessionMissing)) {
			p.forget(s)
			continue
		}
		if !observed.gotConn.Load() {
			p.forget(s)
			_, reply := p.get(ctx, u)
			if reply.Code != 0 {
				return reply
			}
			return failure(3, errUnavailable)
		}
		p.forget(s)
		return failure(3, errUnknown)
	}
	return failure(3, errUnavailable)
}
