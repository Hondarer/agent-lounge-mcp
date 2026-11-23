package tools

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

func TestRetentionHelpAndInfoByKind(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "lounge.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	half, grace, threshold := 9*config.Day, 12*config.Day, 0.25
	cfg := config.Default()
	cfg.MemoryRetentionByKind = make(map[string]config.RetentionOverride)
	for _, kind := range store.ScopeKinds() {
		cfg.MemoryRetentionByKind[kind] = config.RetentionOverride{HalfLife: &half, Grace: &grace, Threshold: &threshold}
	}
	common := config.Default()
	common.MemoryHalfLife, common.MemoryGrace, common.MemoryFadeThreshold = half, grace, threshold
	if got := guide(cfg); got != guide(common) {
		t.Error("equal per-kind settings did not render the common help text")
	}

	info := func(cfg config.Config) InfoOutput {
		t.Helper()
		deps := Deps{Cfg: cfg, Store: st}
		_, out, err := deps.loungeInfo(ctx, nil, InfoInput{})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	out := info(cfg)
	data, err := json.Marshal(out)
	if err != nil || strings.Contains(string(data), "memory_retention") {
		t.Fatalf("uniform info = %s, %v", data, err)
	}

	zero := time.Duration(0)
	cfg.MemoryRetentionByKind[store.ScopeUser] = config.RetentionOverride{HalfLife: &zero}
	cfg.MemoryRetentionByKind[store.ScopeBranch] = config.RetentionOverride{Grace: &zero}
	g := guide(cfg)
	for _, want := range []string{"user 0s/0.1/30d", "repository 9d/0.25/12d", "branch 7d/0.1/0s"} {
		if !strings.Contains(g, want) {
			t.Errorf("guide is missing %q", want)
		}
	}
	for kind, want := range map[string]MemoryRetentionInfo{
		store.ScopeUser:       {HalfLife: "0s", Threshold: 0.1, Grace: "30d"},
		store.ScopeRepository: {HalfLife: "9d", Threshold: 0.25, Grace: "12d"},
		store.ScopeBranch:     {HalfLife: "7d", Threshold: 0.1, Grace: "0s"},
	} {
		if got := info(cfg).MemoryRetention[kind]; got != want {
			t.Errorf("info for %s = %+v, want %+v", kind, got, want)
		}
	}
	if len(g) > maxGuideBytes {
		t.Errorf("mixed guide is %d bytes, limit %d", len(g), maxGuideBytes)
	}

	long, small := time.Duration(math.MaxInt64), 1.2345678901234567e-200
	cfg.MemoryHalfLife, cfg.MemoryGrace, cfg.MemoryFadeThreshold = long, long, small
	cfg.MemoryRetentionByKind = map[string]config.RetentionOverride{store.ScopeUser: {Threshold: &threshold}}
	if n := len(guide(cfg)); n > maxGuideBytes {
		t.Errorf("mixed guide with long values is %d bytes, limit %d", n, maxGuideBytes)
	}
}
