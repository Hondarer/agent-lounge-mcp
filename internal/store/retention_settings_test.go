package store

import (
	"context"
	"math"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestRetentionByKind(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	clock := useClock(s)
	s.SetRetentionPolicies(RetentionPolicies{Common: DefaultRetention(), ByKind: map[string]RetentionPolicy{
		ScopeUser:       {HalfLife: day, Threshold: 0.25, Grace: day},
		ScopeRepository: {HalfLife: 2 * day, Threshold: 0.5},
		ScopeBranch:     {HalfLife: 4 * day, Threshold: 0.25, Grace: day},
		ScopeShared:     {HalfLife: 8 * day, Threshold: 0.25, Grace: day},
	}})
	alice := joinWith(t, s, JoinParams{
		UserID: user, Name: "alice", Location: &Location{Origin: "github.com/org/repo"}, Branch: ptr("main"),
	})
	for _, kind := range scopeKinds {
		createMemory(t, s, alice, kind, "quiet "+kind)
	}
	clock.advance(8 * day)

	want := map[string]struct {
		faded bool
		score float64
	}{
		ScopeUser:       {true, math.Exp2(-10.0 / 3)},
		ScopeRepository: {true, 0.25},
		ScopeBranch:     {false, math.Exp2(-4.0 / 3)},
		ScopeShared:     {false, 0.5},
	}
	check := func(source string, rows []Memory) {
		t.Helper()
		if len(rows) != len(want) {
			t.Fatalf("%s returned %d memories, want %d", source, len(rows), len(want))
		}
		for _, m := range rows {
			w, ok := want[m.Scope.Kind]
			if !ok || m.Faded != w.faded || math.Abs(m.Retention-w.score) > 1e-8 {
				t.Errorf("%s %s: score %v, faded %v, want %+v", source, m.Scope.Kind, m.Retention, m.Faded, w)
			}
		}
	}
	listed, err := s.ListMemories(ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	check("list", listed)
	found := search(t, s, alice, MemorySearch{Limit: 4})
	if len(found) != 2 || found[0].Scope.Kind != ScopeShared || found[1].Scope.Kind != ScopeBranch {
		t.Fatalf("plain search = %+v, want only the shared and branch memories", found)
	}
	found = search(t, s, alice, MemorySearch{Limit: 4, IncludeFaded: true})
	check("search", found)
	if kinds := []string{found[0].Scope.Kind, found[1].Scope.Kind, found[2].Scope.Kind, found[3].Scope.Kind}; !slices.Equal(kinds, []string{ScopeShared, ScopeBranch, ScopeRepository, ScopeUser}) {
		t.Errorf("search order = %v", kinds)
	}

	// At day eight, only the user memory is past both fade and grace.
	if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 1 {
		t.Fatalf("cleanup deleted %d memories, want 1", r.Memories)
	}
	delete(want, ScopeUser)
	listed, err = s.ListMemories(ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	check("list after cleanup", listed)

	// Read fresh memories twice within the reference window, which exposes the score
	// from the first reference rather than refreshing it again.
	firstReads := make(map[string]Memory)
	for _, kind := range scopeKinds {
		m := createMemory(t, s, alice, kind, "get "+kind)
		firstReads[kind], err = s.GetMemory(ctx, alice.ID, m.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	clock.advance(day / 2)
	for i, kind := range scopeKinds {
		m, err := s.GetMemory(ctx, alice.ID, firstReads[kind].ID)
		halfLifeInDays := 2 << i
		expected := math.Exp2(-0.5 / float64(halfLifeInDays))
		if err != nil || m.Faded || m.AccessCount != 1 || math.Abs(m.Retention-expected) > 1e-8 {
			t.Errorf("get %s = score %v, faded %v, count %d, error %v, want score %v", kind, m.Retention, m.Faded, m.AccessCount, err, expected)
		}
	}
}

func TestRetentionMigrationAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lounge.db")
	s, err := openUnmigrated(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.applyMigrations(ctx, ms[:7]); err != nil {
		t.Fatal(err)
	}
	clock := useClock(s)
	local, err := s.EnsureLocalUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	alice := joinWith(t, s, JoinParams{
		UserID: local.ID, Name: "alice", Location: &Location{Origin: "github.com/org/repo"},
	})
	initial := RetentionPolicies{Common: DefaultRetention(), ByKind: map[string]RetentionPolicy{
		ScopeRepository: {HalfLife: day, Threshold: 0.25, Grace: day},
	}}
	s.SetRetentionPolicies(initial)
	m := createMemory(t, s, alice, ScopeRepository, "retained discount")
	clock.advance(4 * day)
	search(t, s, alice, MemorySearch{Scope: ScopeRepository, IncludeFaded: true})
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	checkDiscount := func(s *Store, want time.Duration) {
		t.Helper()
		var discount int64
		err := s.db.QueryRowContext(ctx, "SELECT discount FROM memory_scope_activity WHERE scope = ? AND scope_id = ?", m.Scope.Kind, m.Scope.ID).Scan(&discount)
		if err != nil || discount != want.Milliseconds() {
			t.Fatalf("recorded discount = %d ms, %v; want %v", discount, err, want)
		}
	}

	upgraded := open(t, path)
	upgraded.SetClock(clock.now)
	if v, err := upgraded.SchemaVersion(ctx); err != nil || v != len(ms) {
		t.Fatalf("schema = %d, %v, want %d", v, err, len(ms))
	}
	checkDiscount(upgraded, 2*day)
	if p, err := upgraded.RecordedRetention(ctx); err != nil || len(p.ByKind) != 0 || p.Common != DefaultRetention() {
		t.Fatalf("initial records = %+v, %v", p, err)
	}
	if err := upgraded.RecordRetention(ctx, initial); err != nil {
		t.Fatal(err)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatal(err)
	}

	restarted := open(t, path)
	restarted.SetClock(clock.now)
	checkRecorded := func(want RetentionPolicies) {
		t.Helper()
		p, err := restarted.RecordedRetention(ctx)
		if err != nil || len(p.ByKind) != len(scopeKinds) {
			t.Fatalf("recorded settings = %+v, %v", p, err)
		}
		for _, kind := range scopeKinds {
			if p.For(kind) != want.For(kind) {
				t.Errorf("recorded %s = %+v, want %+v", kind, p.For(kind), want.For(kind))
			}
		}
	}
	checkRecorded(initial)
	clock.advance(day)
	changed := RetentionPolicies{Common: DefaultRetention(), ByKind: map[string]RetentionPolicy{
		ScopeUser:       {Threshold: 0.2},
		ScopeRepository: {HalfLife: 2 * day, Threshold: 0.3, Grace: 7*time.Hour + time.Nanosecond},
	}}
	if err := restarted.RecordRetention(ctx, changed); err != nil {
		t.Fatal(err)
	}
	restarted.SetRetentionPolicies(changed)
	checkRecorded(changed)
	var recorded int
	err = restarted.db.QueryRowContext(ctx, "SELECT count(*) FROM memory_retention WHERE recorded_at = ?", millis(clock.now())).Scan(&recorded)
	if err != nil || recorded != len(scopeKinds) {
		t.Fatalf("rows with replacement timestamp = %d, %v", recorded, err)
	}
	checkDiscount(restarted, 2*day)
	got, err := restarted.MemoryByID(ctx, m.ID)
	if err != nil || got.Faded || math.Abs(got.Retention-math.Exp2(-1.5)) > 1e-8 {
		t.Errorf("after restart = score %v, faded %v, error %v", got.Retention, got.Faded, err)
	}
	clock.advance(4 * day)
	search(t, restarted, alice, MemorySearch{Scope: ScopeRepository, IncludeFaded: true})
	checkDiscount(restarted, 4*day)
}
