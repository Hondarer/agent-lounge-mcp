package store

import (
	"math"
	"time"
)

// The forgetting model of memories. A memory's retention score falls from 1 toward 0 as time
// passes since it was last touched, and falls more slowly the more often it was referenced:
//
//	base      = the later of its last update and its last counted reference
//	half-life = HalfLife × (1 + log2(1 + references))
//	score     = 2 ^ ( -(now - base) / half-life )
//
// A memory fades when its score falls below Threshold: it no longer appears in plain searches,
// but can still be fetched by ID, which brings it back. Solving score = Threshold for the time
// gives the moment it fades:
//
//	fades at  = base + half-life × log2(1 / Threshold)
//
// It is deleted Grace after it faded. A pinned memory scores 1 and neither fades nor is
// deleted. With the default HalfLife of 7 days and Threshold of 0.1, a memory never referenced
// fades about 23 days after it was last touched (7 × 3.32), one referenced once after about 47
// days, three times after about 70 days and seven times after about 93 days.
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

// retentionState holds the inputs of the forgetting model for one memory.
type retentionState struct {
	pinned      bool
	updatedAt   time.Time
	accessedAt  time.Time // zero when never referenced
	accessCount int
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

// score returns the retention score of a memory at now: 1 for a pinned memory or when
// forgetting is disabled.
func (p RetentionPolicy) score(st retentionState, now time.Time) float64 {
	if st.pinned || !p.forgets() {
		return 1
	}
	elapsed := float64(now.Sub(st.base()))
	if elapsed <= 0 {
		return 1
	}
	return math.Exp2(-elapsed / p.halfLife(st.accessCount))
}

// fadesAt returns the moment a memory fades, and false when it never fades (pinned, or
// forgetting is disabled).
func (p RetentionPolicy) fadesAt(st retentionState) (time.Time, bool) {
	if st.pinned || !p.forgets() {
		return time.Time{}, false
	}
	return st.base().Add(p.fadeAfter(st.accessCount)), true
}

// FadesAfter returns how long after it was last touched a memory with the given number of
// references fades, and false when memories do not fade under p.
func (p RetentionPolicy) FadesAfter(accessCount int) (time.Duration, bool) {
	if !p.forgets() {
		return 0, false
	}
	return p.fadeAfter(accessCount), true
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

// faded reports whether a memory has faded at now: its score is below the threshold.
func (p RetentionPolicy) faded(st retentionState, now time.Time) bool {
	at, ok := p.fadesAt(st)
	return ok && now.After(at)
}

// expired reports whether a faded memory is due for deletion at now: Grace has passed since
// it faded. It is never due when Grace is zero.
func (p RetentionPolicy) expired(st retentionState, now time.Time) bool {
	if p.Grace <= 0 {
		return false
	}
	at, ok := p.fadesAt(st)
	return ok && now.After(at.Add(p.Grace))
}
