package tools

import (
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
)

var finalToolNames = []string{"lounge_help", "lounge_join", "lounge_who", "chat_send", "chat_inbox", "lounge_info"}

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
