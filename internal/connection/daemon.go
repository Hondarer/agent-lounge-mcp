// Package connection runs the local lounge daemon and its Unix socket protocol.
// One daemon holds a connection per participant because joining binds the
// participant to that connection, while the user sets up the daemon only once.
// It fetches the inbox only on request, so messages become delivered when the
// agent asks to read them. It replays only requests certainly refused before
// execution, and restores joins only after reconnecting. The agent chooses when
// to join again because its shell directory may differ from its work location.
package connection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Name  string
	URL   string
	Token string
	Idle  time.Duration
}

type daemon struct {
	upstream     *upstream
	own          participant
	state        State
	mu           sync.Mutex
	participants map[string]*participant
}

func (d *daemon) status() *State {
	state := d.state
	state.Participants = map[string]Participant{}
	d.mu.Lock()
	defer d.mu.Unlock()
	for name, p := range d.participants {
		p.mu.Lock()
		state.Participants[name] = Participant{Connected: p.session != nil && !d.upstream.invalid.Load(), LastJoin: p.lastJoin}
		p.mu.Unlock()
	}
	return &state
}

func (d *daemon) handle(ctx context.Context, req Request) Reply {
	switch req.Op {
	case "status":
		return Reply{State: d.status()}
	case "tools":
		for attempt := 0; attempt < 2; attempt++ {
			s, reply := d.own.get(ctx, d.upstream)
			if s == nil {
				return reply
			}
			var tools Reply
			callCtx, observed, stop := observe(ctx)
			defer stop()
			for tool, err := range s.Tools(callCtx, nil) {
				if err != nil {
					if d.upstream.invalid.Load() {
						return failure(3, errUnauthorized)
					}
					if ctx.Err() != nil {
						return failure(3, ctx.Err())
					}
					d.own.forget(s)
					if observed.status.Load() == 404 {
						tools = Reply{Code: 3}
						break
					}
					return failure(3, errUnavailable)
				}
				tools.Tools = append(tools.Tools, tool)
			}
			if tools.Code == 0 {
				return tools
			}
		}
		return failure(3, errUnavailable)
	case "call":
		p := &d.own
		if req.Tool != "lounge_help" {
			d.mu.Lock()
			p = d.participants[req.As]
			if p == nil {
				p = new(participant)
				d.participants[req.As] = p
			}
			d.mu.Unlock()
		}
		return p.call(ctx, d.upstream, req)
	default:
		return failure(2, errors.New("unknown daemon request"))
	}
}

// encode redacts credentials from data fields, leaving protocol keys intact.
func (d *daemon) encode(w io.Writer, value any) error {
	token := d.upstream.token
	if token != "" {
		switch v := value.(type) {
		case Reply:
			v.Error = strings.ReplaceAll(v.Error, token, "[redacted]")
			if v.State != nil {
				v.State = sanitizedState(v.State, token)
			}
			for _, tool := range v.Tools {
				tool.Description = strings.ReplaceAll(tool.Description, token, "[redacted]")
			}
			if len(v.Result) != 0 {
				var data any
				dec := json.NewDecoder(strings.NewReader(string(v.Result)))
				dec.UseNumber()
				if err := dec.Decode(&data); err != nil {
					return err
				}
				v.Result, _ = json.Marshal(redactData(data, token))
			}
			value = v
		case *State:
			value = sanitizedState(v, token)
		}
	}
	return json.NewEncoder(w).Encode(value)
}

func sanitizedState(state *State, token string) *State {
	safe := *state
	safe.URL = strings.ReplaceAll(safe.URL, token, "[redacted]")
	safe.Name = strings.ReplaceAll(safe.Name, token, "[redacted]")
	safe.Participants = make(map[string]Participant, len(state.Participants))
	for name, p := range state.Participants {
		p.LastJoin = strings.ReplaceAll(p.LastJoin, token, "[redacted]")
		safe.Participants[strings.ReplaceAll(name, token, "[redacted]")] = p
	}
	return &safe
}

func redactData(value any, token string) any {
	switch v := value.(type) {
	case string:
		return strings.ReplaceAll(v, token, "[redacted]")
	case []any:
		for i := range v {
			v[i] = redactData(v[i], token)
		}
	case map[string]any:
		for key, item := range v {
			v[key] = redactData(item, token)
		}
	}
	return value
}

// Serve initializes MCP before accepting requests and owns the socket until
// all in-flight requests and upstream sessions have closed.
func Serve(ctx context.Context, opt Options, ready func(*State)) error {
	var stopClients []net.Conn
	defer func() {
		for _, conn := range stopClients {
			_ = json.NewEncoder(conn).Encode(Reply{})
			conn.Close()
		}
	}()
	path, err := socketPath(opt.Name)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(path, ".sock")
	// Keep a stable lock inode across starts so stale-socket removal and binding
	// cannot race with another process starting or shutting down.
	lock, err := os.OpenFile(base+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := lockFile(lock); err != nil {
		return fmt.Errorf("connection %q is already running or starting", opt.Name)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	u := newUpstream(opt.URL, opt.Token)
	s, err := u.connect(ctx)
	if err != nil {
		return err
	}
	d := &daemon{upstream: u, state: State{Name: opt.Name, URL: opt.URL, PID: os.Getpid(), Started: time.Now().UTC()}, participants: map[string]*participant{}}
	d.own.session = s
	defer func() {
		var closes sync.WaitGroup
		for _, p := range d.participants {
			if p.session != nil {
				closes.Go(func() { _ = p.session.Close() })
			}
		}
		if d.own.session != nil {
			closes.Go(func() { _ = d.own.session.Close() })
		}
		closes.Wait()
	}()

	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return err
	}
	ln.SetUnlinkOnClose(false)
	defer ln.Close()
	defer os.Remove(path)
	statePath := base + ".json"
	stateFile, err := os.OpenFile(statePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(statePath)
	err = d.encode(stateFile, d.status())
	closeErr := stateFile.Close()
	if err := errors.Join(err, closeErr); err != nil {
		return err
	}
	if ready != nil {
		state := d.status()
		if opt.Token != "" {
			state = sanitizedState(state, opt.Token)
		}
		ready(state)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stopAccept := context.AfterFunc(ctx, func() { ln.Close() })
	defer stopAccept()
	var requests sync.WaitGroup
	var idleMu sync.Mutex
	var idle *time.Timer
	var generation int
	var active int
	adjustIdle := func(delta int) {
		idleMu.Lock()
		defer idleMu.Unlock()
		active += delta
		generation++
		if idle != nil {
			idle.Stop()
		}
		if active == 0 && opt.Idle > 0 && ctx.Err() == nil {
			current := generation
			idle = time.AfterFunc(opt.Idle, func() {
				idleMu.Lock()
				defer idleMu.Unlock()
				if current == generation {
					cancel()
				}
			})
		}
	}
	adjustIdle(0)
	for {
		conn, err := ln.AcceptUnix()
		if err != nil {
			cancel()
			requests.Wait()
			break
		}
		adjustIdle(1)
		requests.Go(func() {
			defer adjustIdle(-1)
			stopRead := context.AfterFunc(ctx, func() { conn.Close() })
			defer stopRead()
			reader := json.NewDecoder(conn)
			var req Request
			if err := reader.Decode(&req); err != nil {
				conn.Close()
				return
			}
			if req.Op == "stop" && stopRead() {
				d.mu.Lock()
				stopClients = append(stopClients, conn)
				d.mu.Unlock()
				cancel()
				return
			}
			defer conn.Close()
			callCtx, stopCall := context.WithCancel(ctx)
			defer stopCall()
			disconnected := make(chan struct{})
			go func() {
				_, _ = io.Copy(io.Discard, conn)
				stopCall()
				close(disconnected)
			}()
			_ = d.encode(conn, d.handle(callCtx, req))
			conn.Close()
			<-disconnected
		})
	}
	return nil
}

func LogPath(name string) (string, error) {
	path, err := socketPath(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), name+".log"), nil
}
