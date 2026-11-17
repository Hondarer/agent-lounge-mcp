package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// startCleanup runs the periodic cleanup of the store in the background: once at once, then
// every cfg.CleanupInterval, until stop is called. stop cancels a run in progress (its
// transaction rolls back and is done again at the next start) and returns when the background
// work has ended, so the store can be closed after it.
func startCleanup(ctx context.Context, st *store.Store, cfg config.Config) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(cfg.CleanupInterval)
		defer ticker.Stop()
		for {
			cleanupOnce(ctx, st, cfg)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

// cleanupOnce runs the cleanup and logs one line when it did something, and the error when it
// failed. The line has counts only: no message bodies and no names.
func cleanupOnce(ctx context.Context, st *store.Store, cfg config.Config) {
	r, err := st.Cleanup(ctx, cfg.CleanupPolicy())
	if !r.Empty() {
		log.Print(describeCleanup(r))
	}
	if err != nil && ctx.Err() == nil {
		log.Printf("cleanup failed (tried again in %s): %v", config.FormatDuration(cfg.CleanupInterval), err)
	}
}

func describeCleanup(r store.CleanupResult) string {
	var parts []string
	if r.Expired > 0 {
		parts = append(parts, plural(r.Expired, "unread delivery", "unread deliveries")+" expired")
	}
	if r.Messages > 0 {
		parts = append(parts, plural(r.Messages, "old message", "old messages")+" deleted ("+
			plural(r.MessageDeliveries, "delivery", "deliveries")+")")
	}
	if r.Sessions > 0 {
		parts = append(parts, plural(r.Sessions, "unused session", "unused sessions")+" deleted ("+
			plural(r.SessionMessages, "message", "messages")+", "+plural(r.SessionDeliveries, "delivery", "deliveries")+")")
	}
	if r.Memories > 0 {
		parts = append(parts, plural(r.Memories, "forgotten memory", "forgotten memories")+" deleted")
	}
	return "cleanup: " + strings.Join(parts, "; ")
}

// describeCleanupPolicy states the cleanup settings for the start-up log.
func describeCleanupPolicy(cfg config.Config) string {
	parts := []string{"unread messages never expire", "messages are never deleted", "unused sessions are never deleted"}
	if cfg.ExpireAfter > 0 {
		parts[0] = "unread messages expire after " + config.FormatDuration(cfg.ExpireAfter)
	}
	if cfg.RetainFor > 0 {
		parts[1] = "messages are deleted " + config.FormatDuration(cfg.RetainFor) + " after their last change of state"
	}
	if cfg.SessionIdle > 0 {
		parts[2] = "sessions unused for " + config.FormatDuration(cfg.SessionIdle) + " are deleted"
	}
	return "cleanup every " + config.FormatDuration(cfg.CleanupInterval) + ": " + strings.Join(parts, "; ")
}

// describeMemoryPolicy states the forgetting model of memories for the start-up log.
func describeMemoryPolicy(cfg config.Config) string {
	if cfg.MemoryHalfLife == 0 {
		return "memories: forgetting is disabled (they never fade and are never deleted)"
	}
	s := fmt.Sprintf("memories: half-life %s, longer with references; they fade below a retention score of %v",
		config.FormatDuration(cfg.MemoryHalfLife), cfg.MemoryFadeThreshold)
	if cfg.MemoryGrace == 0 {
		return s + " and are never deleted"
	}
	return s + " and are deleted " + config.FormatDuration(cfg.MemoryGrace) + " after they fade"
}
