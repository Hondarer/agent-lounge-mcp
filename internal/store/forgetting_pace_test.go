package store

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestForgettingPaceFormula(t *testing.T) {
	p := DefaultRetention() // 7d, 0.1, 30d
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	// g(q) table from BL-043.
	for _, tc := range []struct {
		qDays    float64
		wantDays float64
	}{
		{1, 1},
		{7, 7},
		{14, 7 + 7.0/3.0},
		{30, 7 + 23.0/3.0},
		{90, 7 + 83.0/3.0},
		{180, 7 + 173.0/3.0},
	} {
		got := p.gapClock(time.Duration(tc.qDays * float64(day)))
		want := time.Duration(tc.wantDays * float64(day))
		if diff := got - want; diff < -time.Second || diff > time.Second {
			t.Errorf("gapClock(%vd) = %v, want %v", tc.qDays, got, want)
		}
	}

	// Quiet scope from base time, 0 references.
	st := retentionState{updatedAt: base}
	act := scopeActivity{lastAccess: base}
	for _, tc := range []struct {
		wallDays float64
		effDays  float64
		score    float64
		faded    bool
		expired  bool
	}{
		{7, 7, 0.5, false, false},
		{21, 7 + 14.0/3.0, 0.314980, false, false},
		{30, 7 + 23.0/3.0, 0.234, false, false},
		{55.77, 23.256667, 0.1, true, false},
		{100, 38, 0.023, true, false},
		{145.77, 53.256667, 0.005, true, true},
	} {
		now := base.Add(time.Duration(tc.wallDays * float64(day)))
		eff := p.effectiveElapsed(st, act, now)
		wantEff := time.Duration(tc.effDays * float64(day))
		if diff := eff - wantEff; diff < -time.Minute || diff > time.Minute {
			t.Errorf("day %v: effective = %v, want %v", tc.wallDays, eff, wantEff)
		}
		sc := p.scoreAt(st, act, now)
		if math.Abs(sc-tc.score) > 0.002 {
			t.Errorf("day %v: score = %.6f, want ~%.6f", tc.wallDays, sc, tc.score)
		}
		if got := p.fadedAt(st, act, now); got != tc.faded {
			t.Errorf("day %v: faded = %v, want %v", tc.wallDays, got, tc.faded)
		}
		if got := p.expiredAt(st, act, now); got != tc.expired {
			t.Errorf("day %v: expired = %v, want %v", tc.wallDays, got, tc.expired)
		}
	}

	// Quiet fades/deletes helpers match BL-043 table.
	for _, tc := range []struct {
		refs       int
		fadeDays   float64
		deleteDays float64
	}{
		{0, 55.760490, 145.760490},
		{1, 125.520980, 215.520980},
		{3, 195.281470, 285.281470},
		{7, 265.041960, 355.041960},
	} {
		fd, ok := p.FadesAfterQuiet(tc.refs)
		if !ok || math.Abs(fd.Hours()/24-tc.fadeDays) > 0.01 {
			t.Errorf("FadesAfterQuiet(%d) = %.4fd, ok=%v; want %.4fd", tc.refs, fd.Hours()/24, ok, tc.fadeDays)
		}
		dd, ok := p.DeletesAfterQuiet(tc.refs)
		if !ok || math.Abs(dd.Hours()/24-tc.deleteDays) > 0.01 {
			t.Errorf("DeletesAfterQuiet(%d) = %.4fd, ok=%v; want %.4fd", tc.refs, dd.Hours()/24, ok, tc.deleteDays)
		}
	}
}

func TestForgettingPaceActiveVsQuietScope(t *testing.T) {
	ctx := context.Background()

	// 1. Active scope (another memory accessed daily): untouched 0-ref memory fades between day 23 and 24,
	//    deleted between day 53 and 54.
	t.Run("active_scope_matches_original_curve", func(t *testing.T) {
		s, user := openLocal(t)
		clock := useClock(s)
		alice := join(t, s, user, "alice")
		target := createMemory(t, s, alice, "", "untouched memory")
		for d := 1; d <= 23; d++ {
			clock.advance(day)
			// Keep scope active without referencing target (empty query list does not count refs).
			if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Scope: ScopeUser, Limit: 1}); err != nil {
				t.Fatal(err)
			}
		}
		m, err := s.MemoryByID(ctx, target.ID)
		if err != nil || m.Faded {
			t.Fatalf("day 23 active: faded=%v, err=%v", m.Faded, err)
		}
		clock.advance(day) // day 24
		m, err = s.MemoryByID(ctx, target.ID)
		if err != nil || !m.Faded {
			t.Fatalf("day 24 active: faded=%v, err=%v", m.Faded, err)
		}
		for d := 25; d <= 53; d++ {
			clock.advance(day)
			if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Scope: ScopeUser, Limit: 1}); err != nil {
				t.Fatal(err)
			}
		}
		if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 0 {
			t.Fatalf("day 53 active: deleted %d, want 0", r.Memories)
		}
		clock.advance(day) // day 54
		if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 1 {
			t.Fatalf("day 54 active: deleted %d, want 1", r.Memories)
		}
	})

	// 2. Quiet scope: stays alive at day 30, fades at day 56, stays at day 145, deleted at day 146;
	//    closing a 100-day gap keeps effective elapsed at 38 days, then daily access advances 1d/day.
	t.Run("quiet_scope_and_gap_closing", func(t *testing.T) {
		s, user := openLocal(t)
		clock := useClock(s)
		alice := join(t, s, user, "alice")
		m := createMemory(t, s, alice, "", "quiet note")

		// Day 30: still not faded, cleanup does not advance last_access.
		clock.advance(30 * day)
		if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 0 {
			t.Fatalf("day 30 cleanup deleted %d", r.Memories)
		}
		got, err := s.MemoryByID(ctx, m.ID)
		if err != nil || got.Faded {
			t.Fatalf("day 30 quiet: %+v, err=%v", got, err)
		}

		// Day 55: still not faded. Day 56: faded.
		clock.advance(25 * day) // day 55
		if got, err := s.MemoryByID(ctx, m.ID); err != nil || got.Faded {
			t.Fatalf("day 55 quiet: faded=%v, err=%v", got.Faded, err)
		}
		clock.advance(day) // day 56
		if got, err := s.MemoryByID(ctx, m.ID); err != nil || !got.Faded {
			t.Fatalf("day 56 quiet: faded=%v, err=%v", got.Faded, err)
		}

		// Day 100: before closing gap vs right after closing gap (via list search) have identical retention (eff = 38d).
		clock.advance(44 * day) // day 100
		before, err := s.MemoryByID(ctx, m.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Scope: ScopeUser, Limit: 1}); err != nil {
			t.Fatal(err)
		}
		after, err := s.MemoryByID(ctx, m.ID)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(before.Retention-after.Retention) > 1e-9 {
			t.Errorf("gap close changed retention: before=%.9f after=%.9f", before.Retention, after.Retention)
		}
		wantScore38 := math.Exp2(-38.0 / 7.0)
		if math.Abs(after.Retention-wantScore38) > 1e-6 {
			t.Errorf("day 100 retention = %.9f, want %.9f (eff=38d)", after.Retention, wantScore38)
		}

		// After closing at day 100 (eff=38d), daily access advances eff by 1d/day -> deleted at eff=53.25d (+16d -> day 116).
		for range 15 {
			clock.advance(day)
			if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Scope: ScopeUser, Limit: 1}); err != nil {
				t.Fatal(err)
			}
		}
		if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 0 {
			t.Fatalf("eff=53d: deleted %d, want 0", r.Memories)
		}
		clock.advance(day) // eff=54d
		if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 1 {
			t.Fatalf("eff=54d: deleted %d, want 1", r.Memories)
		}
	})

	// 3. Scope independence: searching user scope does not advance repository or branch scope.
	t.Run("scope_independence", func(t *testing.T) {
		s, user := openLocal(t)
		clock := useClock(s)
		alice := joinWith(t, s, JoinParams{
			UserID: user, Name: "alice",
			Location: &Location{Origin: "github.com/org/repo"},
			Branch:   ptr("feat"),
		})
		uMem := createMemory(t, s, alice, ScopeUser, "user mem")
		rMem := createMemory(t, s, alice, ScopeRepository, "repo mem")
		bMem := createMemory(t, s, alice, ScopeBranch, "branch mem")

		// Touch only user scope daily for 25 days.
		for range 25 {
			clock.advance(day)
			if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Scope: ScopeUser, Limit: 1}); err != nil {
				t.Fatal(err)
			}
		}
		uGot, _ := s.MemoryByID(ctx, uMem.ID)
		rGot, _ := s.MemoryByID(ctx, rMem.ID)
		bGot, _ := s.MemoryByID(ctx, bMem.ID)
		if !uGot.Faded {
			t.Errorf("active user memory should be faded at day 25: %+v", uGot)
		}
		if rGot.Faded || bGot.Faded {
			t.Errorf("quiet repo/branch memories should not be faded at day 25: repo=%v branch=%v", rGot.Faded, bGot.Faded)
		}
	})
}

func TestMigrationV6ToV7CreatesScopeActivity(t *testing.T) {
	ctx := context.Background()
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v6-activity.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	s := &Store{db: db, now: time.Now, limits: DefaultLimits(), retention: DefaultRetention()}
	if err := s.applyMigrations(ctx, ms[:6]); err != nil {
		db.Close()
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	tRecent := millis(now.Add(-5 * day))
	tQuiet := millis(now.Add(-30 * day))
	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (id, name, token_hash, created_at) VALUES (1, 'local', 'none:local', 1000);
		INSERT INTO memories (id, scope, scope_id, user_id, body, created_at, updated_at, created_user, created_session, updated_user, updated_session) VALUES
			(1, 'user', 'local', 1, 'recent', ?, ?, 'local', 'a', 'local', 'a'),
			(2, 'repository', 'github.com/o/r', NULL, 'quiet 30d', ?, ?, 'local', 'a', 'local', 'a');
		INSERT INTO memories_fts (rowid, title, body, tags) VALUES (1, '', 'recent', ''), (2, '', 'quiet 30d', '');
	`, tRecent, tRecent, tQuiet, tQuiet); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()

	upgraded := open(t, path)
	upgraded.SetClock(func() time.Time { return now })

	// Recent scope (5d <= 7d): open discount is 0, score matches original curve.
	m1, err := upgraded.MemoryByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantRecent := math.Exp2(-5.0 / 7.0)
	if math.Abs(m1.Retention-wantRecent) > 1e-6 || m1.Faded {
		t.Errorf("recent memory = %.6f faded=%v, want %.6f", m1.Retention, m1.Faded, wantRecent)
	}

	// Quiet scope (30d): effective elapsed is 7 + 23/3 = 14.6667d -> score ~0.234, not faded!
	m2, err := upgraded.MemoryByID(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	wantQuiet := math.Exp2(-(7.0 + 23.0/3.0) / 7.0)
	if math.Abs(m2.Retention-wantQuiet) > 1e-4 || m2.Faded {
		t.Errorf("quiet 30d memory = %.6f faded=%v, want %.6f", m2.Retention, m2.Faded, wantQuiet)
	}
}
