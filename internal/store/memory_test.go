package store

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func sp(s string) *string { return &s }

func tagsPtr(tags ...string) *[]string { return &tags }

func createMemory(t *testing.T, s *Store, sess Session, scope, body string, tags ...string) Memory {
	t.Helper()
	in := MemoryInput{Body: &body}
	if len(tags) > 0 {
		in.Tags = &tags
	}
	m, err := s.CreateMemory(context.Background(), sess.ID, scope, in)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func search(t *testing.T, s *Store, sess Session, q MemorySearch) []Memory {
	t.Helper()
	if q.Limit == 0 {
		q.Limit = 50
	}
	ms, _, err := s.SearchMemories(context.Background(), sess.ID, q)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func memoryIDs(ms []Memory) []int64 {
	out := []int64{}
	for _, m := range ms {
		out = append(out, m.ID)
	}
	return out
}

func memoryCount(t *testing.T, s *Store, table string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRowContext(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// The retention model gives the values of the proposal: with a half-life of 7 days and a
// threshold of 0.1, a memory fades about 23, 47, 70 and 93 days after it was last touched when
// it was referenced 0, 1, 3 and 7 times.
func TestRetentionModel(t *testing.T) {
	p := DefaultRetention()
	for _, c := range []struct{ refs, days int }{{0, 23}, {1, 47}, {3, 70}, {7, 93}} {
		d, ok := p.FadesAfter(c.refs)
		if got := int(math.Round(d.Hours() / 24)); !ok || got != c.days {
			t.Errorf("%d references: fades after %v (%d days), want about %d days", c.refs, d, got, c.days)
		}
	}
	base := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	st := retentionState{updatedAt: base}
	if got := p.score(st, base); got != 1 {
		t.Errorf("score at the base time = %v", got)
	}
	if got := p.score(st, base.Add(7*day)); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("score after one half-life = %v", got)
	}
	// The later of the update and the last reference is the base.
	st.accessedAt = base.Add(10 * day)
	if got := p.score(st, base.Add(10*day)); got != 1 {
		t.Errorf("score at the last reference = %v", got)
	}
	st = retentionState{updatedAt: base}
	if p.faded(st, base.Add(23*day)) || !p.faded(st, base.Add(24*day)) {
		t.Error("a memory never referenced fades between day 23 and day 24")
	}
	if p.expired(st, base.Add(53*day)) || !p.expired(st, base.Add(54*day)) {
		t.Error("a memory never referenced is deleted between day 53 and day 54 (30 days after it faded)")
	}
	pinned := retentionState{updatedAt: base, pinned: true}
	if p.score(pinned, base.Add(1000*day)) != 1 || p.faded(pinned, base.Add(1000*day)) || p.expired(pinned, base.Add(1000*day)) {
		t.Error("a pinned memory scores 1 and never fades")
	}
	off := RetentionPolicy{Threshold: 0.1, Grace: day}
	if off.score(st, base.Add(1000*day)) != 1 || off.faded(st, base.Add(1000*day)) {
		t.Error("with a half-life of 0, memories never fade")
	}
	keep := RetentionPolicy{HalfLife: 7 * day, Threshold: 0.1}
	if !keep.faded(st, base.Add(1000*day)) || keep.expired(st, base.Add(1000*day)) {
		t.Error("with a grace of 0, faded memories are kept")
	}
}

func TestMemoryCreateGetUpdate(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	clock := useClock(s)
	alice := joinWith(t, s, JoinParams{UserID: user, Name: "alice", Agent: sp("claude")})
	bob := joinWith(t, s, JoinParams{UserID: user, Name: "bob", Agent: sp("codex")})

	m, err := s.CreateMemory(ctx, alice.ID, "", MemoryInput{Body: sp("make clean fixes the build"), Title: sp("  Build fix "), Tags: tagsPtr("Build", "go", "build")})
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == 0 || m.Scope != (Scope{Kind: ScopeUser, ID: LocalUser}) || m.Title != "Build fix" || !slices.Equal(m.Tags, []string{"build", "go"}) ||
		m.CreatedBy != (Author{User: LocalUser, Session: "alice", Agent: "claude"}) || m.UpdatedBy != m.CreatedBy ||
		m.Retention != 1 || m.Faded || m.Pinned || m.AccessCount != 0 || !m.AccessedAt.IsZero() {
		t.Fatalf("created %+v", m)
	}

	clock.advance(time.Hour)
	got, err := s.GetMemory(ctx, bob.ID, m.ID)
	if err != nil || got.Body != m.Body || got.AccessCount != 1 || !got.AccessedAt.Equal(clock.now()) || !slices.Equal(got.Tags, m.Tags) {
		t.Fatalf("get: %+v, %v", got, err)
	}

	// Only the given fields change; the updater is recorded.
	clock.advance(time.Hour)
	up, err := s.UpdateMemory(ctx, bob.ID, m.ID, MemoryInput{Tags: tagsPtr()})
	if err != nil || up.Body != m.Body || up.Title != "Build fix" || len(up.Tags) != 0 ||
		up.UpdatedBy.Session != "bob" || up.CreatedBy.Session != "alice" || !up.UpdatedAt.Equal(clock.now()) || up.AccessCount != 1 {
		t.Fatalf("update tags: %+v, %v", up, err)
	}
	up, err = s.UpdateMemory(ctx, alice.ID, m.ID, MemoryInput{Body: sp("run make clean, then make"), Title: sp("")})
	if err != nil || up.Body != "run make clean, then make" || up.Title != "" || up.UpdatedBy.Session != "alice" {
		t.Fatalf("update body: %+v, %v", up, err)
	}
	if _, err := s.UpdateMemory(ctx, alice.ID, m.ID, MemoryInput{}); !errors.Is(err, ErrInvalidMemory) {
		t.Errorf("update without fields: %v", err)
	}
	// The index follows the update.
	if ms := search(t, s, alice, MemorySearch{Query: "then make"}); !slices.Equal(memoryIDs(ms), []int64{m.ID}) {
		t.Errorf("search after update = %v", memoryIDs(ms))
	}
	if ms := search(t, s, alice, MemorySearch{Query: "fixes"}); len(ms) != 0 {
		t.Errorf("the old body is still found: %v", memoryIDs(ms))
	}

	if err := s.ForgetMemory(ctx, bob.ID, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetMemory(ctx, alice.ID, m.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("get after forget: %v", err)
	}
	if err := s.ForgetMemory(ctx, bob.ID, m.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("forget twice: %v", err)
	}
	if n := memoryCount(t, s, "memories_fts") + memoryCount(t, s, "memory_tags") + memoryCount(t, s, "memory_refs"); n != 0 {
		t.Errorf("%d rows of the memory remain", n)
	}
	// IDs are not given again.
	again := createMemory(t, s, alice, "", "another")
	if again.ID <= m.ID {
		t.Errorf("new ID %d after deleting %d", again.ID, m.ID)
	}
}

func TestMemoryValidation(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	alice := join(t, s, user, "alice")
	l := DefaultLimits()
	l.MaxMemoryBytes, l.MaxMemoriesPerScope = 8, 2
	s.SetLimits(l)

	cases := []struct {
		name string
		in   MemoryInput
		want error
	}{
		{"no body", MemoryInput{}, ErrInvalidMemory},
		{"blank body", MemoryInput{Body: sp("  \n")}, ErrInvalidMemory},
		{"large body", MemoryInput{Body: sp("123456789")}, ErrMemoryTooLarge},
		{"multi-byte body over the limit", MemoryInput{Body: sp("あいう")}, ErrMemoryTooLarge},
		{"title with a newline", MemoryInput{Body: sp("x"), Title: sp("a\nb")}, ErrInvalidMemory},
		{"long title", MemoryInput{Body: sp("x"), Title: sp(strings.Repeat("あ", 201))}, ErrInvalidMemory},
		{"tag with a space", MemoryInput{Body: sp("x"), Tags: tagsPtr("a b")}, ErrInvalidMemory},
		{"tag with a comma", MemoryInput{Body: sp("x"), Tags: tagsPtr("a,b")}, ErrInvalidMemory},
		{"empty tag", MemoryInput{Body: sp("x"), Tags: tagsPtr("")}, ErrInvalidMemory},
		{"long tag", MemoryInput{Body: sp("x"), Tags: tagsPtr(strings.Repeat("x", 33))}, ErrInvalidMemory},
		{"11 tags", MemoryInput{Body: sp("x"), Tags: tagsPtr("1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11")}, ErrInvalidMemory},
	}
	for _, c := range cases {
		if _, err := s.CreateMemory(ctx, alice.ID, "", c.in); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
	if _, err := s.CreateMemory(ctx, alice.ID, "shared", MemoryInput{Body: sp("x")}); !errors.Is(err, ErrInvalidScope) {
		t.Errorf("scope shared: %v", err)
	}
	// Japanese tags, duplicates folded, 32 characters allowed.
	m, err := s.CreateMemory(ctx, alice.ID, "", MemoryInput{Body: sp("x"), Tags: tagsPtr("ビルド", "ビルド", strings.Repeat("あ", 32))})
	if err != nil || len(m.Tags) != 2 {
		t.Fatalf("Japanese tags: %+v, %v", m, err)
	}
	createMemory(t, s, alice, "", "y")
	if _, err := s.CreateMemory(ctx, alice.ID, "", MemoryInput{Body: sp("z")}); !errors.Is(err, ErrTooManyMemories) {
		t.Errorf("third memory with a limit of 2: %v", err)
	}
	// The limit is per scope: the repository scope still has room.
	repo := joinWith(t, s, JoinParams{UserID: user, Name: "repo", Location: &Location{Origin: "github.com/org/a"}})
	if _, err := s.CreateMemory(ctx, repo.ID, ScopeRepository, MemoryInput{Body: sp("z")}); err != nil {
		t.Errorf("repository memory: %v", err)
	}
	// Updating works at the limit.
	if _, err := s.UpdateMemory(ctx, alice.ID, m.ID, MemoryInput{Body: sp("new")}); err != nil {
		t.Errorf("update at the limit: %v", err)
	}
}

// Sessions use the memories of the scopes they belong to: their user, and their repository
// across users. Other memories look missing.
func TestMemoryScopes(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	other := addUser(t, s, "other").ID
	repoA := &Location{Origin: "github.com/org/a"}
	alice := joinWith(t, s, JoinParams{UserID: local, Name: "alice", Location: repoA})
	carol := joinWith(t, s, JoinParams{UserID: other, Name: "carol", Location: repoA})
	dave := joinWith(t, s, JoinParams{UserID: other, Name: "dave", Location: &Location{Origin: "github.com/org/b"}})
	erin := join(t, s, local, "erin") // no repository

	own := createMemory(t, s, alice, ScopeUser, "alice's note")
	shared := createMemory(t, s, alice, ScopeRepository, "repo note")
	if shared.Scope != (Scope{Kind: ScopeRepository, ID: "github.com/org/a"}) {
		t.Fatalf("repository memory scope = %+v", shared.Scope)
	}
	if _, err := s.CreateMemory(ctx, erin.ID, ScopeRepository, MemoryInput{Body: sp("x")}); !errors.Is(err, ErrNoRepository) {
		t.Errorf("repository memory without a repository: %v", err)
	}

	// carol (another user, same repository) reads and updates the repository memory, not
	// alice's own.
	if _, err := s.GetMemory(ctx, carol.ID, own.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("carol reads alice's user memory: %v", err)
	}
	if _, err := s.UpdateMemory(ctx, carol.ID, own.ID, MemoryInput{Body: sp("x")}); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("carol updates alice's user memory: %v", err)
	}
	up, err := s.UpdateMemory(ctx, carol.ID, shared.ID, MemoryInput{Body: sp("repo note, checked")})
	if err != nil || up.UpdatedBy.User != "other" || up.CreatedBy.User != LocalUser {
		t.Errorf("carol updates the repository memory: %+v, %v", up, err)
	}
	if ms := search(t, s, carol, MemorySearch{}); !slices.Equal(memoryIDs(ms), []int64{shared.ID}) {
		t.Errorf("carol lists %v", memoryIDs(ms))
	}
	// dave (another repository) and erin (no repository) do not see it.
	for _, sess := range []Session{dave, erin} {
		if _, err := s.GetMemory(ctx, sess.ID, shared.ID); !errors.Is(err, ErrNoSuchMemory) {
			t.Errorf("%s reads the repository memory: %v", sess.Name, err)
		}
		if err := s.ForgetMemory(ctx, sess.ID, shared.ID); !errors.Is(err, ErrNoSuchMemory) {
			t.Errorf("%s forgets the repository memory: %v", sess.Name, err)
		}
	}
	if ms := search(t, s, erin, MemorySearch{}); !slices.Equal(memoryIDs(ms), []int64{own.ID}) {
		t.Errorf("erin lists %v", memoryIDs(ms))
	}
	// alice sees both; scope narrows.
	if ms := search(t, s, alice, MemorySearch{Query: "note"}); len(ms) != 2 {
		t.Errorf("alice finds %v", memoryIDs(ms))
	}
	if ms := search(t, s, alice, MemorySearch{Query: "note", Scope: ScopeRepository}); !slices.Equal(memoryIDs(ms), []int64{shared.ID}) {
		t.Errorf("alice in the repository scope finds %v", memoryIDs(ms))
	}
	if _, _, err := s.SearchMemories(ctx, erin.ID, MemorySearch{Scope: ScopeRepository, Limit: 10}); !errors.Is(err, ErrNoRepository) {
		t.Errorf("erin searches the repository scope: %v", err)
	}
	// A session whose repository changes no longer sees the old one's memories.
	moved := joinWith(t, s, JoinParams{UserID: local, Name: "alice", Location: &Location{Origin: "github.com/org/c"}})
	if _, err := s.GetMemory(ctx, moved.ID, shared.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("after moving, alice reads the old repository memory: %v", err)
	}
}

func TestMemorySearch(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	clock := useClock(s)
	alice := join(t, s, user, "alice")
	auth := createMemory(t, s, alice, "", "認証の仕様が今週変わる", "api")
	build := createMemory(t, s, alice, "", "Build error: run MAKE clean", "build", "go")
	cpp := createMemory(t, s, alice, "", `C++ と "quoted" の扱い`)
	clock.advance(day)
	old := createMemory(t, s, alice, "", "認証 認証 認証 古いメモ")

	cases := []struct {
		query string
		tags  []string
		want  []int64
	}{
		{"認証", nil, []int64{old.ID, auth.ID}}, // two characters: no index; old was updated later, so it is kept better
		{"認証の", nil, []int64{auth.ID}},
		{"make clean", nil, []int64{build.ID}},
		{"make  CLEAN", nil, []int64{build.ID}},
		{"build mak", nil, []int64{build.ID}},
		{"build 認証", nil, []int64{}},
		{"C++", nil, []int64{cpp.ID}},
		{`"quoted"`, nil, []int64{cpp.ID}},
		{"AND", nil, []int64{}},
		{"go", nil, []int64{build.ID}}, // a tag, two characters
		{"", []string{"API"}, []int64{auth.ID}},
		{"", []string{"build", "go"}, []int64{build.ID}},
		{"", []string{"build", "api"}, []int64{}},
		{"error", []string{"go"}, []int64{build.ID}},
	}
	for _, c := range cases {
		ms := search(t, s, alice, MemorySearch{Query: c.query, Tags: c.tags})
		if got := memoryIDs(ms); !slices.Equal(got, c.want) {
			t.Errorf("search %q tags %v = %v, want %v", c.query, c.tags, got, c.want)
		}
	}

	// Listing orders by retention: the memory updated last is kept best.
	if ms := search(t, s, alice, MemorySearch{}); len(ms) != 4 || ms[0].ID != old.ID {
		t.Errorf("list = %v", memoryIDs(ms))
	}
	// limit and more.
	ms, more, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Limit: 3})
	if err != nil || len(ms) != 3 || !more {
		t.Errorf("limit 3: %d memories, more %v, %v", len(ms), more, err)
	}
	if _, more, _ := s.SearchMemories(ctx, alice.ID, MemorySearch{Limit: 4}); more {
		t.Error("more with limit 4 of 4 memories")
	}
	// Bounds of the query.
	if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Query: strings.Repeat("a ", 11), Limit: 1}); !errors.Is(err, ErrInvalidMemory) {
		t.Errorf("11 terms: %v", err)
	}
	if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Query: strings.Repeat("あ", 201), Limit: 1}); !errors.Is(err, ErrInvalidMemory) {
		t.Errorf("201 characters: %v", err)
	}
	if _, _, err := s.SearchMemories(ctx, alice.ID, MemorySearch{Scope: "shared", Limit: 1}); !errors.Is(err, ErrInvalidScope) {
		t.Errorf("scope shared: %v", err)
	}
}

// With a term of three or more characters, a better match comes first at equal retention, and
// retention weighs in.
func TestMemorySearchOrder(t *testing.T) {
	s, user := openLocal(t)
	clock := useClock(s)
	alice := join(t, s, user, "alice")
	weak := createMemory(t, s, alice, "", "deploy notes: "+strings.Repeat("other words here ", 30)+"deploy")
	strong := createMemory(t, s, alice, "", "deploy deploy deploy")
	if ms := search(t, s, alice, MemorySearch{Query: "deploy"}); !slices.Equal(memoryIDs(ms), []int64{strong.ID, weak.ID}) {
		t.Errorf("by relevance = %v", memoryIDs(ms))
	}
	// After 50 quiet days without use, the strong match's effective elapsed time is 7 + 43/3 days
	// (the first search counted one reference: a half-life of 14 days); a weak match used just now comes first.
	clock.advance(50 * day)
	if _, err := s.GetMemory(context.Background(), alice.ID, weak.ID); err != nil {
		t.Fatal(err)
	}
	ms := search(t, s, alice, MemorySearch{Query: "deploy"})
	if !slices.Equal(memoryIDs(ms), []int64{weak.ID, strong.ID}) {
		t.Errorf("by relevance and retention = %v", memoryIDs(ms))
	}
	if ms[0].Retention != 1 || math.Abs(ms[1].Retention-math.Exp2(-(7.0+43.0/3.0)/14)) > 1e-6 {
		t.Errorf("retention = %v, %v", ms[0].Retention, ms[1].Retention)
	}
}

// A memory fades, leaves plain searches, comes back when read, and is deleted after the grace
// period; a pinned one stays.
func TestMemoryForgetting(t *testing.T) {
	ctx := context.Background()
	s2, user2 := openLocal(t)
	clock2 := useClock(s2)
	bob := join(t, s2, user2, "bob")
	f := createMemory(t, s2, bob, "", "fades")
	p := createMemory(t, s2, bob, "", "pinned one")
	if _, err := s2.PinMemory(ctx, bob.ID, p.ID, true); err != nil {
		t.Fatal(err)
	}
	// In a quiet scope, an unused memory fades after about 55.76 days.
	clock2.advance(56 * day)
	if ms := search(t, s2, bob, MemorySearch{}); !slices.Equal(memoryIDs(ms), []int64{p.ID}) {
		t.Errorf("day 56, plain list = %v", memoryIDs(ms))
	}
	ms := search(t, s2, bob, MemorySearch{IncludeFaded: true})
	if len(ms) != 2 || !ms[1].Faded || ms[1].ID != f.ID || ms[0].Faded {
		t.Errorf("day 56, with faded = %+v", ms)
	}
	// Searching with a term does not count a faded memory, even with include_faded.
	if ms := search(t, s2, bob, MemorySearch{Query: "fades", IncludeFaded: true}); len(ms) != 1 || !ms[0].Faded {
		t.Errorf("search with faded = %+v", ms)
	}
	got, err := s2.GetMemory(ctx, bob.ID, f.ID)
	if err != nil || got.Faded || got.AccessCount != 1 || got.Retention != 1 {
		t.Errorf("get brings it back: %+v, %v", got, err)
	}

	// Deleted by the cleanup in a quiet scope after about 145.76 days.
	g := createMemory(t, s2, bob, "", "gone")
	clock2.advance(145 * day)
	if r := cleanup(t, s2, CleanupPolicy{}); r.Memories != 0 {
		t.Errorf("day 145: %d deleted", r.Memories)
	}
	clock2.advance(day)
	r := cleanup(t, s2, CleanupPolicy{})
	if r.Memories != 1 {
		t.Errorf("day 146: %d deleted", r.Memories)
	}
	if _, err := s2.MemoryByID(ctx, g.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("deleted memory: %v", err)
	}
	if _, err := s2.MemoryByID(ctx, p.ID); err != nil {
		t.Errorf("pinned memory: %v", err)
	}
	if n := memoryCount(t, s2, "memories_fts"); n != memoryCount(t, s2, "memories") {
		t.Errorf("index rows %d for %d memories", n, memoryCount(t, s2, "memories"))
	}

	// Unpinning starts the clock again from that moment.
	clock2.advance(100 * day)
	un, err := s2.PinMemory(ctx, bob.ID, p.ID, false)
	if err != nil || un.Pinned || un.Faded || !un.AccessedAt.Equal(clock2.now()) || un.AccessCount != 0 {
		t.Errorf("unpinned: %+v, %v", un, err)
	}
	same, err := s2.PinMemory(ctx, bob.ID, p.ID, false)
	if err != nil || !same.AccessedAt.Equal(un.AccessedAt) {
		t.Errorf("unpinning again: %+v, %v", same, err)
	}

	// Forgetting disabled: nothing fades or is deleted.
	s2.SetRetention(RetentionPolicy{Threshold: 0.1, Grace: day})
	clock2.advance(1000 * day)
	if ms := search(t, s2, bob, MemorySearch{}); len(ms) != 2 {
		t.Errorf("without forgetting: %v", memoryIDs(ms))
	}
	if r := cleanup(t, s2, CleanupPolicy{}); r.Memories != 0 {
		t.Errorf("without forgetting, %d deleted", r.Memories)
	}
}

// References: memory_get and the first three results of a search with terms, once a day per
// session and memory; listing does not count; reading a faded memory always counts.
func TestMemoryReferences(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	clock := useClock(s)
	alice := join(t, s, user, "alice")
	bob := join(t, s, user, "bob")
	var ms []Memory
	for i := range 5 {
		ms = append(ms, createMemory(t, s, alice, "", "topic "+strings.Repeat("x", i)))
	}
	count := func(m Memory) int {
		t.Helper()
		got, err := s.MemoryByID(ctx, m.ID)
		if err != nil {
			t.Fatal(err)
		}
		return got.AccessCount
	}

	search(t, s, alice, MemorySearch{})
	for _, m := range ms {
		if count(m) != 0 {
			t.Fatal("listing counted a reference")
		}
	}
	res := search(t, s, alice, MemorySearch{Query: "topic"})
	counted := 0
	for i, m := range res {
		if res[i].AccessCount != 0 {
			t.Error("the search returned the counts after counting")
		}
		counted += count(m)
		if want := map[bool]int{true: 1, false: 0}[i < 3]; count(m) != want {
			t.Errorf("result %d counted %d", i, count(m))
		}
	}
	if counted != 3 {
		t.Errorf("%d references counted", counted)
	}
	// The same session again within a day: not counted. Another session: counted.
	search(t, s, alice, MemorySearch{Query: "topic"})
	if _, err := s.GetMemory(ctx, alice.ID, res[0].ID); err != nil {
		t.Fatal(err)
	}
	if count(res[0]) != 1 {
		t.Errorf("counted again within a day: %d", count(res[0]))
	}
	if _, err := s.GetMemory(ctx, bob.ID, res[0].ID); err != nil {
		t.Fatal(err)
	}
	if count(res[0]) != 2 {
		t.Errorf("bob's reference: %d", count(res[0]))
	}
	// A day later, alice's reference counts again; the cleanup drops old reference records.
	clock.advance(day)
	if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 0 {
		t.Fatal(r)
	}
	if n := memoryCount(t, s, "memory_refs"); n != 0 {
		t.Errorf("%d reference records after a day", n)
	}
	if _, err := s.GetMemory(ctx, alice.ID, res[0].ID); err != nil {
		t.Fatal(err)
	}
	if count(res[0]) != 3 {
		t.Errorf("a day later: %d", count(res[0]))
	}
	// A faded memory counts even within the day: reading it brings it back.
	last := res[4]
	clock.advance(60 * day)
	if _, err := s.GetMemory(ctx, alice.ID, last.ID); err != nil {
		t.Fatal(err)
	}
	clock.advance(time.Hour)
	if got, _ := s.MemoryByID(ctx, last.ID); got.Faded {
		t.Fatal("read memory is faded")
	}
	// Make it fade again within alice's window by pushing the clock past its quiet fade time:
	// with one reference it fades after about 125.52 days in a quiet scope.
	clock.advance(130 * day)
	if _, err := s.db.ExecContext(ctx, "UPDATE memory_refs SET counted_at = ?", millis(clock.now())); err != nil {
		t.Fatal(err)
	}
	before := count(last)
	got, err := s.GetMemory(ctx, alice.ID, last.ID)
	if err != nil || got.AccessCount != before+1 || got.Faded {
		t.Errorf("reading a faded memory within the window: %+v, %v (before %d)", got, err, before)
	}
}

// A memory outlives its session; removing a user removes the memories of its user scope but
// not those of a repository.
func TestMemoryDeletionWithSessionsAndUsers(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	bobUser := addUser(t, s, "bob")
	repoA := &Location{Origin: "github.com/org/a"}
	alice := joinWith(t, s, JoinParams{UserID: local, Name: "alice", Location: repoA})
	bob := joinWith(t, s, JoinParams{UserID: bobUser.ID, Name: "bob", Location: repoA})
	own := createMemory(t, s, bob, ScopeUser, "bob's own")
	repo := createMemory(t, s, bob, ScopeRepository, "bob's repository note")
	if _, err := s.GetMemory(ctx, bob.ID, own.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := s.RemoveSession(ctx, "bob", "bob"); err != nil {
		t.Fatal(err)
	}
	if n := memoryCount(t, s, "memory_refs"); n != 0 {
		t.Errorf("%d references of the removed session", n)
	}
	if _, err := s.MemoryByID(ctx, own.ID); err != nil {
		t.Errorf("the memory of a removed session: %v", err)
	}

	data, err := s.CountUserData(ctx, "bob")
	if err != nil || data.Memories != 1 {
		t.Errorf("count of bob's data: %+v, %v", data, err)
	}
	if data, err := s.RemoveUser(ctx, "bob"); err != nil || data.Memories != 1 {
		t.Errorf("remove bob: %+v, %v", data, err)
	}
	if _, err := s.MemoryByID(ctx, own.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("bob's user memory after removing bob: %v", err)
	}
	got, err := s.GetMemory(ctx, alice.ID, repo.ID)
	if err != nil || got.CreatedBy.User != "bob" {
		t.Errorf("the repository memory after removing bob: %+v, %v", got, err)
	}
	if n := memoryCount(t, s, "memories_fts"); n != 1 {
		t.Errorf("%d index rows, want 1", n)
	}
}

func TestMemoryAdminAndExport(t *testing.T) {
	ctx := context.Background()
	s, local := openLocal(t)
	alice := joinWith(t, s, JoinParams{UserID: local, Name: "alice", Location: &Location{Origin: "github.com/org/a"}})
	u2 := createMemory(t, s, alice, ScopeUser, "second", "b", "a")
	r1 := createMemory(t, s, alice, ScopeRepository, "repo")
	u3 := createMemory(t, s, alice, ScopeUser, "third")

	all, err := s.ListMemories(ctx, "")
	if err != nil || !slices.Equal(memoryIDs(all), []int64{r1.ID, u2.ID, u3.ID}) {
		t.Errorf("list = %v, %v", memoryIDs(all), err)
	}
	if user, _ := s.ListMemories(ctx, ScopeUser); !slices.Equal(memoryIDs(user), []int64{u2.ID, u3.ID}) {
		t.Errorf("list user = %v", memoryIDs(user))
	}
	if _, err := s.ListMemories(ctx, "nope"); !errors.Is(err, ErrInvalidScope) {
		t.Errorf("list nope: %v", err)
	}

	var got []Memory
	var msgs int
	err = s.Export(ctx, func(ExportMessage) error { msgs++; return nil }, func(m Memory) error { got = append(got, m); return nil })
	if err != nil || msgs != 0 || !slices.Equal(memoryIDs(got), []int64{r1.ID, u2.ID, u3.ID}) || !slices.Equal(got[1].Tags, []string{"b", "a"}) {
		t.Errorf("export = %+v, %v", got, err)
	}

	if m, err := s.RemoveMemory(ctx, u2.ID); err != nil || m.Body != "second" {
		t.Errorf("remove: %+v, %v", m, err)
	}
	if _, err := s.RemoveMemory(ctx, u2.ID); !errors.Is(err, ErrNoSuchMemory) {
		t.Errorf("remove twice: %v", err)
	}
	if n := memoryCount(t, s, "memories_fts"); n != 2 {
		t.Errorf("%d index rows", n)
	}
}

// A database of schema version 4 migrates to the latest version: the data stays, and memories
// work.
func TestMigrateVersion4(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lounge.db")
	s, err := openUnmigrated(path)
	if err != nil {
		t.Fatal(err)
	}
	ms, err := loadMigrations(schemaFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 5 {
		t.Fatal("no version 5 script")
	}
	if err := s.applyMigrations(ctx, ms[:4]); err != nil {
		t.Fatal(err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, name, token_hash, created_at) VALUES (1, 'local', 'none:local', 10);
		INSERT INTO rooms (id, scope, user_id, repository, name, created_at) VALUES (2, 'user', 1, '', 'lounge', 11);
		INSERT INTO sessions (id, user_id, name, agent, created_at, last_seen_at) VALUES (1, 1, 'alice', 'claude', 12, 20), (2, 1, 'bob', '', 13, 21);
		INSERT INTO messages (id, scope, scope_id, sender_session_id, to_session_id, room_id, body, sent_at, sender_agent) VALUES
			(1, 'user', 'local', 1, 2, NULL, 'direct', 30, 'claude');
		INSERT INTO deliveries (message_id, recipient_session_id, state, queued_at) VALUES (1, 2, 'queued', 30);`)
	if err != nil {
		t.Fatal(err)
	}
	dump := func() string {
		return dumpRows(t, s, []string{
			"SELECT id, name, token_hash, created_at FROM users ORDER BY id", "SELECT id, user_id, name, agent, created_at, last_seen_at, os_user, machine, origin, folder, repository FROM sessions ORDER BY id", "SELECT * FROM rooms ORDER BY id",
			"SELECT * FROM messages ORDER BY id", "SELECT * FROM deliveries ORDER BY message_id",
		})
	}
	before := dump()
	s.Close()

	s = open(t, path)
	if v, _ := s.SchemaVersion(ctx); v != len(ms) {
		t.Fatalf("schema version = %d, want %d", v, len(ms))
	}
	if after := dump(); after != before {
		t.Errorf("data changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	m, err := s.CreateMemory(ctx, 1, "", MemoryInput{Body: sp("migrated")})
	if err != nil || m.CreatedBy.Agent != "claude" {
		t.Fatalf("create after migration: %+v, %v", m, err)
	}
	if got := search(t, s, Session{ID: 2}, MemorySearch{Query: "migrated"}); len(got) != 1 {
		t.Errorf("search after migration = %v", memoryIDs(got))
	}
}

func TestRoundRetention(t *testing.T) {
	for in, want := range map[float64]float64{1: 1, 1 - 1e-9: 1, 0.0999: 0.09, 0.5: 0.5, 0.123: 0.12, 0: 0} {
		if got := RoundRetention(in); got != want {
			t.Errorf("RoundRetention(%v) = %v, want %v", in, got, want)
		}
	}
}

// Durations that would overflow saturate: a memory with many references under a long
// half-life does not fade at once, and the cleanup does not misread the cutoff.
func TestRetentionSaturates(t *testing.T) {
	p := RetentionPolicy{HalfLife: 3650 * day, Threshold: 0.1, Grace: 30 * day}
	for _, refs := range []int{0, 300, 1 << 30} {
		if d, ok := p.FadesAfter(refs); !ok || d <= 0 {
			t.Errorf("%d references: fades after %v", refs, d)
		}
	}
	now := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	st := retentionState{updatedAt: now.Add(-time.Hour), accessCount: 300}
	if p.faded(st, now) || p.expired(st, now) {
		t.Error("a memory with 300 references faded at once")
	}
	tiny := RetentionPolicy{HalfLife: 7 * day, Threshold: 1e-300, Grace: math.MaxInt64}
	s, user := openLocal(t)
	useClock(s)
	s.SetRetention(tiny)
	createMemory(t, s, join(t, s, user, "alice"), "", "kept")
	if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 0 {
		t.Errorf("deleted %d with an overflowing cutoff", r.Memories)
	}
}

// Only the first three results count, faded ones among them excepted: a faded result in the
// top three does not let the fourth count.
func TestSearchCountsTopThreeOnly(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	clock := useClock(s)
	alice := join(t, s, user, "alice")
	bob := join(t, s, user, "bob")
	// The faded memory matches strongly and has just faded (retention just below 0.1); the
	// others match weakly and are half a day younger (retention just above 0.1), so that
	// relevance times retention ranks the faded one first.
	faded := createMemory(t, s, alice, "", "topic topic topic")
	clock.advance(12 * time.Hour)
	for i := range 4 {
		createMemory(t, s, alice, "", "topic and some other words "+strings.Repeat("y", i+1))
	}
	// In a quiet scope, faded (created 12h earlier) fades at 55.76d while the other 4 stay active.
	clock.advance(55*day + 12*time.Hour)
	res := search(t, s, bob, MemorySearch{Query: "topic", IncludeFaded: true})
	if len(res) != 5 {
		t.Fatalf("results = %v", memoryIDs(res))
	}
	pos := slices.Index(memoryIDs(res), faded.ID)
	if pos != 0 || !res[0].Faded || res[1].Faded {
		t.Fatalf("the faded memory ranks %d (%+v); the case needs it first", pos, res[0])
	}
	counted := 0
	for i, m := range res {
		got, err := s.MemoryByID(ctx, m.ID)
		if err != nil {
			t.Fatal(err)
		}
		if want := i < 3 && m.ID != faded.ID; (got.AccessCount == 1) != want {
			t.Errorf("result %d (faded %v) counted %d", i, m.Faded, got.AccessCount)
		}
		counted += got.AccessCount
	}
	if counted != 2 {
		t.Errorf("%d references counted, want 2", counted)
	}
}

// Terms shorter than a trigram ignore the case of every letter, as the index does.
func TestShortTermsFoldEveryLetter(t *testing.T) {
	s, user := openLocal(t)
	alice := join(t, s, user, "alice")
	m := createMemory(t, s, alice, "", "ÄÖÜ ΩΜΈΓΑ Éa")
	for _, q := range []string{"äöü", "ωμέ", "äö", "ωμ", "éa", "ÉA", "Ω"} {
		if got := search(t, s, alice, MemorySearch{Query: q}); !slices.Equal(memoryIDs(got), []int64{m.ID}) {
			t.Errorf("search %q = %v", q, memoryIDs(got))
		}
	}
}

// The cleanup deletes a referenced memory at its own, later time, and keeps faded memories
// when the grace period is 0.
func TestCleanupReferencedAndNoGrace(t *testing.T) {
	ctx := context.Background()
	s, user := openLocal(t)
	clock := useClock(s)
	alice := join(t, s, user, "alice")
	m := createMemory(t, s, alice, "", "used once")
	if _, err := s.GetMemory(ctx, alice.ID, m.ID); err != nil {
		t.Fatal(err)
	}
	clock.advance(215 * day) // one reference in a quiet scope: deleted after about 215.52 days
	if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 0 {
		t.Fatalf("day 215: %d deleted", r.Memories)
	}
	clock.advance(day)
	if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 1 {
		t.Errorf("day 216: %d deleted", r.Memories)
	}

	keep := createMemory(t, s, alice, "", "kept faded")
	s.SetRetention(RetentionPolicy{HalfLife: 7 * day, Threshold: 0.1})
	clock.advance(1000 * day)
	if r := cleanup(t, s, CleanupPolicy{}); r.Memories != 0 {
		t.Errorf("with no grace, %d deleted", r.Memories)
	}
	if got, err := s.MemoryByID(ctx, keep.ID); err != nil || !got.Faded {
		t.Errorf("kept memory: %+v, %v", got, err)
	}
}
