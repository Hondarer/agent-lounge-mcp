package store

import (
	"math"
	"time"
)

// The forgetting model of memories. A memory's retention score falls from 1 toward 0 as
// effective time passes since it was last touched, and falls more slowly the more often it was
// referenced:
//
//	base      = the later of its last update and its last counted reference
//	half-life = HalfLife × (1 + log2(1 + references))
//	effective = max(0, (now - base) - (scope.discount - memory.credit) - openDiscount)
//	score     = 2 ^ ( -effective / half-life )
//
// A scope accessed at intervals up to W = HalfLife accumulates no discount, so effective time
// equals wall-clock time. When a scope stays quiet for q > W, the excess (q - W) advances the
// forgetting clock at 1/3 speed (openDiscount = 2/3 × (q - W)), so quiet scopes retain memories
// up to about 3x longer. On the next scope access, openDiscount is folded into scope.discount
// and memories whose base time moves to now record memory.credit = scope.discount, keeping
// effective elapsed time continuous across every access.
//
// A memory fades when effective >= half-life × log2(1 / Threshold) and is deleted when
// effective >= half-life × log2(1 / Threshold) + Grace. A pinned memory scores 1 and neither
// fades nor is deleted. With the default HalfLife of 7 days, Threshold of 0.1 and Grace of
// 30 days, an unused memory in an active scope fades after about 23 days (deleted after about
// 53 days), and in a quiet scope fades after about 56 days (deleted after about 146 days).
//
// The score is not stored: it depends on the current time and on the options in effect, and
// is computed whenever a memory is read, searched or cleaned up.

// RetentionPolicy holds the parameters of the forgetting model.
type RetentionPolicy struct {
	// HalfLife is the half-life of a memory never referenced. Zero disables forgetting:
	// memories score 1 and neither fade nor are deleted.
	HalfLife time.Duration
	// Threshold is the score below which a memory has faded, between 0 and 1 (exclusive).
	Threshold float64
	// Grace is how long a faded memory is kept before it is deleted. Zero keeps faded memories.
	Grace time.Duration
}

// Default retention, used until SetRetention is called.
const (
	DefaultMemoryHalfLife  = 7 * 24 * time.Hour
	DefaultMemoryThreshold = 0.1
	DefaultMemoryGrace     = 30 * 24 * time.Hour
)

// DefaultRetention returns the retention a store has when it is opened.
func DefaultRetention() RetentionPolicy {
	return RetentionPolicy{HalfLife: DefaultMemoryHalfLife, Threshold: DefaultMemoryThreshold, Grace: DefaultMemoryGrace}
}

// SetRetention replaces the retention policy. Call it before the store is used concurrently.
func (s *Store) SetRetention(p RetentionPolicy) { s.retention = p }

// Retention returns the retention policy in effect.
func (s *Store) Retention() RetentionPolicy { return s.retention }

// RoundRetention renders a retention score with two decimals for display. It rounds down, so
// that a faded memory never shows the threshold itself (0.098 shows as 0.09, not 0.10), with a
// tolerance of a millionth, so that a memory touched a moment ago shows 1, not 0.99.
func RoundRetention(score float64) float64 {
	return math.Floor(score*100+1e-6) / 100
}

// scopeActivity holds the access timing and accumulated forgetting discount of one scope.
// A zero lastAccess means no activity row exists, so effective elapsed time equals wall-clock
// elapsed time (the original curve).
type scopeActivity struct {
	lastAccess time.Time
	discount   time.Duration
}

// retentionState holds the inputs of the forgetting model for one memory.
type retentionState struct {
	pinned      bool
	updatedAt   time.Time
	accessedAt  time.Time // zero when never referenced
	accessCount int
	credit      time.Duration // scope discount at base()
}

func (st retentionState) base() time.Time {
	if st.accessedAt.After(st.updatedAt) {
		return st.accessedAt
	}
	return st.updatedAt
}

// forgets reports whether memories fade at all under p.
func (p RetentionPolicy) forgets() bool {
	return p.HalfLife > 0 && p.Threshold > 0 && p.Threshold < 1
}

func (p RetentionPolicy) halfLife(accessCount int) float64 {
	return float64(p.HalfLife) * (1 + math.Log2(1+float64(max(accessCount, 0))))
}

// gapClock maps a quiet gap q to the effective forgetting time g(q): q up to HalfLife, then
// HalfLife + (q - HalfLife) / 3.
func (p RetentionPolicy) gapClock(q time.Duration) time.Duration {
	if q <= 0 {
		return 0
	}
	if !p.forgets() || q <= p.HalfLife {
		return q
	}
	return p.HalfLife + (q-p.HalfLife)/3
}

// openDiscount returns the unclosed forgetting discount for a scope whose last access was at
// act.lastAccess, evaluated at now: 0 when q <= HalfLife, else 2/3 of (q - HalfLife).
func (p RetentionPolicy) openDiscount(act scopeActivity, now time.Time) time.Duration {
	if act.lastAccess.IsZero() || !p.forgets() {
		return 0
	}
	q := now.Sub(act.lastAccess)
	if q <= p.HalfLife {
		return 0
	}
	return q - p.gapClock(q)
}

// effectiveElapsed returns the effective forgetting time of st at now under act.
func (p RetentionPolicy) effectiveElapsed(st retentionState, act scopeActivity, now time.Time) time.Duration {
	wall := now.Sub(st.base())
	if wall <= 0 {
		return 0
	}
	if act.lastAccess.IsZero() {
		return wall
	}
	paid := act.discount - st.credit
	open := p.openDiscount(act, now)
	eff := wall - paid - open
	if eff <= 0 {
		return 0
	}
	return eff
}

// score returns the retention score of a memory at now without scope activity (active scope).
func (p RetentionPolicy) score(st retentionState, now time.Time) float64 {
	return p.scoreAt(st, scopeActivity{}, now)
}

// scoreAt returns the retention score of a memory at now under act: 1 for a pinned memory or
// when forgetting is disabled.
func (p RetentionPolicy) scoreAt(st retentionState, act scopeActivity, now time.Time) float64 {
	if st.pinned || !p.forgets() {
		return 1
	}
	eff := p.effectiveElapsed(st, act, now)
	if eff <= 0 {
		return 1
	}
	return math.Exp2(-float64(eff) / p.halfLife(st.accessCount))
}

// fadesAt returns the moment a memory fades in an active scope, and false when it never fades.
func (p RetentionPolicy) fadesAt(st retentionState) (time.Time, bool) {
	if st.pinned || !p.forgets() {
		return time.Time{}, false
	}
	return st.base().Add(p.fadeAfter(st.accessCount)), true
}

// FadesAfter returns how long after it was last touched a memory with the given number of
// references fades in an active scope, and false when memories do not fade under p.
func (p RetentionPolicy) FadesAfter(accessCount int) (time.Duration, bool) {
	if !p.forgets() {
		return 0, false
	}
	return p.fadeAfter(accessCount), true
}

// quietWall converts an effective duration eff into the wall-clock duration for a scope that
// stays quiet from the memory's base time: eff when eff <= HalfLife, else 3*eff - 2*HalfLife
// (saturating at math.MaxInt64).
func (p RetentionPolicy) quietWall(eff time.Duration) time.Duration {
	if eff <= p.HalfLife {
		return eff
	}
	f := 3*float64(eff) - 2*float64(p.HalfLife)
	if f >= math.MaxInt64 {
		return math.MaxInt64
	}
	return time.Duration(f)
}

// FadesAfterQuiet returns how long after it was last touched a memory fades in a scope that
// receives no further memory access, and false when memories do not fade under p.
func (p RetentionPolicy) FadesAfterQuiet(accessCount int) (time.Duration, bool) {
	if !p.forgets() {
		return 0, false
	}
	return p.quietWall(p.fadeAfter(accessCount)), true
}

// DeletesAfterQuiet returns how long after it was last touched a memory is deleted in a scope
// that receives no further memory access, and false when memories are not deleted under p.
func (p RetentionPolicy) DeletesAfterQuiet(accessCount int) (time.Duration, bool) {
	if !p.forgets() || p.Grace <= 0 {
		return 0, false
	}
	fade := p.fadeAfter(accessCount)
	total := fade + p.Grace
	if total < fade {
		return math.MaxInt64, true
	}
	return p.quietWall(total), true
}

// fadeAfter returns how long after it was last touched a memory with the given reference
// count fades: its half-life × log2(1 / Threshold). forgets must be true.
//
// The result saturates at the longest Duration (about 292 years): a large half-life, a tiny
// threshold or many references would otherwise overflow, and an overflowed duration is
// negative, which would make the memory fade at once.
func (p RetentionPolicy) fadeAfter(accessCount int) time.Duration {
	d := p.halfLife(accessCount) * math.Log2(1/p.Threshold)
	if d >= math.MaxInt64 {
		return math.MaxInt64
	}
	return time.Duration(d)
}

// faded reports whether a memory has faded at now in an active scope.
func (p RetentionPolicy) faded(st retentionState, now time.Time) bool {
	return p.fadedAt(st, scopeActivity{}, now)
}

// fadedAt reports whether a memory has faded at now under act: its effective elapsed time has
// reached fadeAfter.
func (p RetentionPolicy) fadedAt(st retentionState, act scopeActivity, now time.Time) bool {
	if st.pinned || !p.forgets() {
		return false
	}
	return p.effectiveElapsed(st, act, now) >= p.fadeAfter(st.accessCount)
}

// expired reports whether a faded memory is due for deletion at now in an active scope.
func (p RetentionPolicy) expired(st retentionState, now time.Time) bool {
	return p.expiredAt(st, scopeActivity{}, now)
}

// expiredAt reports whether a faded memory is due for deletion at now under act: effective
// elapsed time has reached fadeAfter + Grace. It is never due when Grace is zero.
func (p RetentionPolicy) expiredAt(st retentionState, act scopeActivity, now time.Time) bool {
	if st.pinned || !p.forgets() || p.Grace <= 0 {
		return false
	}
	fade := p.fadeAfter(st.accessCount)
	total := fade + p.Grace
	if total < fade {
		return false // saturated
	}
	return p.effectiveElapsed(st, act, now) >= total
}
