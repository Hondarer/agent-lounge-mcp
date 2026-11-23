package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/hondarer/agent-lounge-mcp/internal/store"
)

// RetentionOverride holds the supplied items. Nil items inherit their defaults.
type RetentionOverride struct {
	HalfLife  *time.Duration
	Threshold *float64
	Grace     *time.Duration
}

// Apply replaces only the supplied items in p.
func (o RetentionOverride) Apply(p store.RetentionPolicy) store.RetentionPolicy {
	if o.HalfLife != nil {
		p.HalfLife = *o.HalfLife
	}
	if o.Threshold != nil {
		p.Threshold = *o.Threshold
	}
	if o.Grace != nil {
		p.Grace = *o.Grace
	}
	return p
}

// RetentionPolicies returns the effective server policy for every scope kind.
func (c Config) RetentionPolicies() store.RetentionPolicies {
	p := store.RetentionPolicies{Common: c.Retention(), ByKind: make(map[string]store.RetentionPolicy)}
	for kind, o := range c.MemoryRetentionByKind {
		p.ByKind[kind] = o.Apply(p.Common)
	}
	return p
}

// ValidateRetentionOverrides checks per-kind values with the same bounds as the common options.
func (c Config) ValidateRetentionOverrides() error {
	for _, kind := range store.ScopeKinds() {
		o := c.MemoryRetentionByKind[kind]
		prefix := "--memory-" + kind
		for _, opt := range []struct {
			suffix string
			value  *time.Duration
		}{{"-half-life", o.HalfLife}, {"-grace", o.Grace}} {
			if opt.value != nil && *opt.value < 0 {
				return fmt.Errorf("%s%s %s: must not be negative (0 disables it)", prefix, opt.suffix, *opt.value)
			}
		}
		if o.Threshold != nil {
			if err := validateFadeThreshold(prefix+"-fade-threshold", *o.Threshold); err != nil {
				return err
			}
		}
	}
	return nil
}

// FormatRetentionByKind renders the three values for each supported scope kind.
func FormatRetentionByKind(policies store.RetentionPolicies) string {
	var parts []string
	for _, kind := range store.ScopeKinds() {
		p := policies.For(kind)
		parts = append(parts, fmt.Sprintf("%s %s/%g/%s", kind, FormatDuration(p.HalfLife), p.Threshold, FormatDuration(p.Grace)))
	}
	return strings.Join(parts, ", ")
}
