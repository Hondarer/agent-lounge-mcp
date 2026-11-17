package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Day is the unit d of the duration options: 24 hours.
const Day = 24 * time.Hour

// dayRE matches a number followed by the unit d. No unit of time.ParseDuration contains the
// letter d, so the match is never part of another unit.
var dayRE = regexp.MustCompile(`([0-9]*\.?[0-9]+)d`)

// ParseDuration parses a duration option: the form of time.ParseDuration (such as 90m or 36h)
// with the added unit d of 24 hours (such as 14d or 1d12h). A negative duration is an error.
func ParseDuration(s string) (time.Duration, error) {
	in := strings.TrimSpace(s)
	var convErr error
	conv := dayRE.ReplaceAllStringFunc(in, func(m string) string {
		v, err := strconv.ParseFloat(strings.TrimSuffix(m, "d"), 64)
		if err != nil {
			convErr = err
			return m
		}
		return strconv.FormatFloat(v*24, 'f', -1, 64) + "h"
	})
	d, err := time.ParseDuration(conv)
	if err != nil || convErr != nil {
		return 0, fmt.Errorf("invalid duration %q: use a number with a unit, such as 90m, 36h, 14d or 1d12h (d is 24 hours)", s)
	}
	if d < 0 {
		return 0, fmt.Errorf("invalid duration %q: must not be negative", s)
	}
	return d, nil
}

// FormatDuration renders a non-negative duration as the options accept it, without zero
// units: 14d, 1d12h, 1h, 1m30s, 0. A part below a second is rendered as time.Duration does.
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return d.String()
	}
	var b strings.Builder
	if d >= Day {
		b.WriteString(strconv.FormatInt(int64(d/Day), 10) + "d")
		d %= Day
	}
	if d%time.Second != 0 {
		if d > 0 {
			b.WriteString(d.String())
		}
		return b.String()
	}
	for _, u := range []struct {
		unit time.Duration
		name string
	}{{time.Hour, "h"}, {time.Minute, "m"}, {time.Second, "s"}} {
		if n := d / u.unit; n > 0 {
			b.WriteString(strconv.FormatInt(int64(n), 10) + u.name)
			d %= u.unit
		}
	}
	return b.String()
}

// DurationValue is a flag.Value for a duration option (see ParseDuration).
type DurationValue struct{ D *time.Duration }

func (v DurationValue) String() string {
	if v.D == nil {
		return ""
	}
	return FormatDuration(*v.D)
}

func (v DurationValue) Set(s string) error {
	d, err := ParseDuration(s)
	if err != nil {
		return errors.New(strings.TrimPrefix(err.Error(), "invalid duration "+strconv.Quote(s)+": "))
	}
	*v.D = d
	return nil
}
