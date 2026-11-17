package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default configuration: %v", err)
	}
	for _, listen := range []string{"127.0.0.1:0", "[::1]:7788", "localhost:7788"} {
		cfg := Default()
		cfg.Listen = listen
		if err := cfg.Validate(); err != nil {
			t.Errorf("--listen %s: %v", listen, err)
		}
	}
	for _, listen := range []string{"0.0.0.0:7788", ":7788", "192.0.2.1:7788", "example.com:7788", "127.0.0.1"} {
		cfg := Default()
		cfg.Listen = listen
		if err := cfg.Validate(); err == nil {
			t.Errorf("--listen %s was accepted", listen)
		}
	}
	for _, db := range []string{"", "lounge.db?mode=memory"} {
		cfg := Default()
		cfg.DBPath = db
		if err := cfg.Validate(); err == nil {
			t.Errorf("--db %q was accepted", db)
		}
	}
	for _, wait := range []time.Duration{0, time.Second, MaxInboxWaitMax} {
		cfg := Default()
		cfg.InboxWaitMax = wait
		if err := cfg.Validate(); err != nil {
			t.Errorf("--inbox-wait-max %s: %v", wait, err)
		}
	}
	for _, wait := range []time.Duration{-time.Second, MaxInboxWaitMax + time.Second} {
		cfg := Default()
		cfg.InboxWaitMax = wait
		if err := cfg.Validate(); err == nil {
			t.Errorf("--inbox-wait-max %s was accepted", wait)
		}
	}
}

func TestRemoteListening(t *testing.T) {
	for _, listen := range []string{"0.0.0.0:7788", ":7788", "192.0.2.1:7788", "[::]:7788"} {
		cfg := Default()
		cfg.Listen = listen
		err := cfg.Validate()
		if !errors.Is(err, ErrRemoteNotAllowed) || !strings.Contains(err.Error(), "--allow-remote") {
			t.Errorf("--listen %s without --allow-remote: %v", listen, err)
		}
		cfg.AllowRemote = true
		if err := cfg.Validate(); err != nil {
			t.Errorf("--listen %s --allow-remote: %v", listen, err)
		}
		if !cfg.ListensRemote() || !cfg.TokenRequired() {
			t.Errorf("--listen %s: remote %v, token required %v", listen, cfg.ListensRemote(), cfg.TokenRequired())
		}
	}
	// --allow-remote alone changes nothing on loopback.
	cfg := Default()
	cfg.AllowRemote = true
	if err := cfg.Validate(); err != nil || cfg.ListensRemote() || cfg.TokenRequired() {
		t.Errorf("loopback with --allow-remote: %v %v %v", err, cfg.ListensRemote(), cfg.TokenRequired())
	}
}

func TestTokenRequired(t *testing.T) {
	cfg := Default()
	if cfg.TokenRequired() {
		t.Error("the default configuration requires a token")
	}
	cfg.RequireToken = true
	if !cfg.TokenRequired() {
		t.Error("--require-token does not require a token")
	}
	cfg = Default()
	cfg.AllowedHosts = []string{"lounge.example.com"}
	if err := cfg.Validate(); err != nil || !cfg.TokenRequired() {
		t.Errorf("--allowed-host: %v, token required %v", err, cfg.TokenRequired())
	}
	for _, h := range []string{"", "https://x", "x:443", "x/y", "a b"} {
		cfg.AllowedHosts = []string{h}
		if err := cfg.Validate(); err == nil {
			t.Errorf("--allowed-host %q was accepted", h)
		}
	}
	for _, h := range []string{"lounge.example.com", "192.0.2.1", "[2001:db8::1]", "my_host"} {
		cfg.AllowedHosts = []string{h}
		if err := cfg.Validate(); err != nil {
			t.Errorf("--allowed-host %q: %v", h, err)
		}
	}
}

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"0":         0,
		"90m":       90 * time.Minute,
		"36h":       36 * time.Hour,
		"14d":       14 * Day,
		"1d12h":     36 * time.Hour,
		"12h1d":     36 * time.Hour,
		"1.5d":      36 * time.Hour,
		"2d30m10s":  2*Day + 30*time.Minute + 10*time.Second,
		" 30d ":     30 * Day,
		"1d12h0m0s": 36 * time.Hour,
	} {
		got, err := ParseDuration(in)
		if err != nil || got != want {
			t.Errorf("ParseDuration(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "d", "14", "-1d", "-1s", "1w", "1dd", "1d-2h", "abc"} {
		if d, err := ParseDuration(in); err == nil {
			t.Errorf("ParseDuration(%q) = %v, want an error", in, d)
		}
	}
	for d, want := range map[time.Duration]string{0: "0s", 14 * Day: "14d", time.Hour: "1h", 36 * time.Hour: "1d12h", time.Second: "1s",
		90 * time.Second: "1m30s", Day + 1500*time.Millisecond: "1d1.5s", 2*Day + time.Minute: "2d1m"} {
		if got := FormatDuration(d); got != want {
			t.Errorf("FormatDuration(%v) = %q, want %q", d, got, want)
		}
		if back, err := ParseDuration(FormatDuration(d)); err != nil || back != d {
			t.Errorf("ParseDuration(FormatDuration(%v)) = %v, %v", d, back, err)
		}
	}
	var d time.Duration
	v := DurationValue{&d}
	if err := v.Set("1d"); err != nil || d != Day || v.String() != "1d" {
		t.Errorf("DurationValue: %v %v %q", d, err, v.String())
	}
	if err := v.Set("-1d"); err == nil || strings.Contains(err.Error(), `"-1d"`) {
		t.Errorf("DurationValue.Set(-1d) = %v", err)
	}
}

func TestValidateCleanupAndLimits(t *testing.T) {
	cfg := Default()
	if cfg.ExpireAfter != 14*Day || cfg.RetainFor != 30*Day || cfg.SessionIdle != 60*Day || cfg.CleanupInterval != time.Hour ||
		cfg.MaxBodyBytes != 65536 || cfg.MaxUnread != 1000 || cfg.MaxSessionsPerUser != 500 {
		t.Errorf("defaults: %+v", cfg)
	}
	// The default session period is longer than the longest life of a message.
	if cfg.SessionIdle <= cfg.ExpireAfter+cfg.RetainFor {
		t.Error("SessionIdle is not longer than ExpireAfter + RetainFor")
	}
	for _, ok := range []func(*Config){
		func(c *Config) { c.ExpireAfter, c.RetainFor, c.SessionIdle = 0, 0, 0 },
		func(c *Config) { c.CleanupInterval = time.Second },
		func(c *Config) { c.MaxBodyBytes = 1 },
		func(c *Config) { c.MaxBodyBytes = MaxMaxBodyBytes },
		func(c *Config) { c.MaxUnread, c.MaxSessionsPerUser = 1, 1 },
	} {
		cfg := Default()
		ok(&cfg)
		if err := cfg.Validate(); err != nil {
			t.Errorf("%+v: %v", cfg, err)
		}
	}
	for want, bad := range map[string]func(*Config){
		"--expire-after":          func(c *Config) { c.ExpireAfter = -time.Second },
		"--retain-for":            func(c *Config) { c.RetainFor = -time.Second },
		"--session-idle":          func(c *Config) { c.SessionIdle = -time.Second },
		"--cleanup-interval":      func(c *Config) { c.CleanupInterval = time.Second - 1 },
		"--max-body-bytes 0":      func(c *Config) { c.MaxBodyBytes = 0 },
		"--max-body-bytes 104857": func(c *Config) { c.MaxBodyBytes = MaxMaxBodyBytes + 1 },
		"--max-unread":            func(c *Config) { c.MaxUnread = 0 },
		"--max-sessions-per-user": func(c *Config) { c.MaxSessionsPerUser = 0 },
	} {
		cfg := Default()
		bad(&cfg)
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", want, err)
		}
	}
	for want, bad := range map[string]func(*Config){
		"--memory-half-life":       func(c *Config) { c.MemoryHalfLife = -time.Second },
		"--memory-grace":           func(c *Config) { c.MemoryGrace = -time.Second },
		"--memory-fade-threshold":  func(c *Config) { c.MemoryFadeThreshold = 0 },
		"--max-memory-bytes":       func(c *Config) { c.MaxMemoryBytes = MaxMaxBodyBytes + 1 },
		"--max-memories-per-scope": func(c *Config) { c.MaxMemoriesPerScope = 0 },
	} {
		cfg := Default()
		bad(&cfg)
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", want, err)
		}
	}
	if p := Default().Retention(); p.HalfLife != 7*Day || p.Threshold != 0.1 || p.Grace != 30*Day {
		t.Errorf("Retention = %+v", p)
	}
	if l := Default().Limits(); l.MaxMemoryBytes != 16384 || l.MaxMemoriesPerScope != 10000 {
		t.Errorf("memory limits = %+v", l)
	}
	if l := Default().Limits(); l.MaxBodyBytes != 65536 || l.MaxUnread != 1000 || l.MaxSessionsPerUser != 500 {
		t.Errorf("Limits = %+v", l)
	}
	if p := Default().CleanupPolicy(); p.ExpireAfter != 14*Day || p.RetainFor != 30*Day || p.SessionIdle != 60*Day {
		t.Errorf("CleanupPolicy = %+v", p)
	}
}
