// Package clock provides a swappable time source for testability.
package clock

import "time"

// Clock abstracts time so tests can control it.
// Production code uses Real; tests inject Fixed to pin the current time.
type Clock interface {
	// Now returns the current time. The value is in the location of the
	// underlying implementation (UTC for Real when Loc is unset).
	Now() time.Time
}

// Real is the production clock. Loc is the app timezone; nil means UTC.
// A zero Real is a valid, always-current clock.
type Real struct{ Loc *time.Location }

// Now returns the current time in the configured location.
// The location defaults to UTC when Loc is nil.
func (r Real) Now() time.Time {
	if r.Loc == nil {
		return time.Now().UTC()
	}
	return time.Now().In(r.Loc)
}

// Fixed is a clock pinned to a specific instant (tests).
// The pinned instant is returned as UTC regardless of the input's location.
type Fixed struct{ T time.Time }

// Now returns the pinned time.
// The instant is normalized to UTC, so it equals f.T.UTC().
func (f Fixed) Now() time.Time { return f.T.UTC() }
