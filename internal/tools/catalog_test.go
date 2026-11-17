package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
)

var finalToolNames = []string{"lounge_help", "lounge_join", "lounge_who", "chat_send", "chat_inbox", "chat_ack", "chat_status", "chat_history",
	"memory_put", "memory_search", "memory_get", "memory_pin", "memory_forget", "lounge_info"}

func TestFinalToolNames(t *testing.T) {
	catalog := Catalog()
	for _, name := range finalToolNames {
		if _, ok := catalog[name]; !ok {
			t.Errorf("missing tool %s", name)
		}
	}
	if len(catalog) != len(finalToolNames) {
		t.Errorf("catalog has %d tools, want %d", len(catalog), len(finalToolNames))
	}
}

// The guide lists every tool and takes its values from the configuration.
func TestGuide(t *testing.T) {
	g := guide(config.Default())
	_, tools, _ := strings.Cut(g, "## Tools")
	tools, _, _ = strings.Cut(tools, "\n## ")
	for name := range Catalog() {
		if !strings.Contains(tools, "- "+name+": ") {
			t.Errorf("the Tools section does not list %s", name)
		}
	}
	for _, want := range []string{"at most 45 seconds", "up to 20 messages", "at most 100", ShutdownMessage, sessionNameRule} {
		if !strings.Contains(g, want) {
			t.Errorf("guide does not contain %q", want)
		}
	}

	cfg := config.Default()
	cfg.InboxWaitMax = 90 * time.Second
	if g := guide(cfg); !strings.Contains(g, "at most 90 seconds") {
		t.Error("guide does not follow --inbox-wait-max")
	}
	cfg.InboxWaitMax = 0
	if g := guide(cfg); !strings.Contains(g, "Waiting is disabled on this server") || strings.Contains(g, "at most 0 seconds") {
		t.Error("guide does not say that waiting is disabled")
	}
}

// The guide states the retention periods and the limits in effect.
func TestGuideRetentionAndLimits(t *testing.T) {
	g := guide(config.Default())
	for _, want := range []string{"within 14d of being sent", "deleted 30d after its last change", "not used for 60d", "documents of the repository",
		"at most 65536 bytes", "at most 1000 unread", "at most 500 sessions", "up to 100 IDs", "chat_ack", "unacked: true"} {
		if !strings.Contains(g, want) {
			t.Errorf("guide does not contain %q", want)
		}
	}
	cfg := config.Default()
	cfg.ExpireAfter, cfg.RetainFor, cfg.SessionIdle = 0, 36*time.Hour, 0
	cfg.MaxBodyBytes = 1024
	g = guide(cfg)
	for _, want := range []string{"Unread messages do not expire.", "deleted 1d12h after", "Unused sessions are not deleted automatically.", "at most 1024 bytes"} {
		if !strings.Contains(g, want) {
			t.Errorf("guide with changed options does not contain %q", want)
		}
	}
	cfg.RetainFor = 0
	if g := guide(cfg); !strings.Contains(g, "Messages are not deleted for their age.") {
		t.Error("guide does not say that messages are kept")
	}
}

// The guide states the forgetting model and the memory limits in effect.
func TestGuideMemory(t *testing.T) {
	g := guide(config.Default())
	for _, want := range []string{"## Memory", "falling by half every 7d", "below 0.1", "about 23 days", "about 47 days",
		"deleted 30d after it faded", "at most 16384 bytes", "at most 10000 memories", "up to 10 memories", "first 3 results",
		"memory_search with words of the task"} {
		if !strings.Contains(g, want) {
			t.Errorf("guide does not contain %q", want)
		}
	}
	cfg := config.Default()
	cfg.MemoryGrace = 0
	if g := guide(cfg); !strings.Contains(g, "Faded memories are not deleted on this server") {
		t.Error("guide does not follow --memory-grace 0")
	}
	cfg.MemoryHalfLife = 0
	if g := guide(cfg); !strings.Contains(g, "Memories do not fade on this server") || strings.Contains(g, "falling by half") {
		t.Error("guide does not follow --memory-half-life 0")
	}
}
