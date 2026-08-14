// Package clock provides a swappable time source for testability.
package clock

import "time"

// Clock abstracts time so tests can control it.
type Clock interface {
	Now() time.Time
}

// Real is the production clock. Loc is the app timezone; nil means UTC.
type Real struct{ Loc *time.Location }

// Now returns the current time in the configured location.
func (r Real) Now() time.Time {
	if r.Loc == nil {
		return time.Now().UTC()
	}
	return time.Now().In(r.Loc)
}

// Fixed is a clock pinned to a specific instant (tests).
type Fixed struct{ T time.Time }

// Now returns the pinned time.
func (f Fixed) Now() time.Time { return f.T.UTC() }
