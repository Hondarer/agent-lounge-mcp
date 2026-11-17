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
