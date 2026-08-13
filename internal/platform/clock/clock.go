// Package clock provides a swappable time source for testability.
package clock

import "time"

// Clock abstracts time so tests can control it.
type Clock interface {
	Now() time.Time
}

// Real is the production clock.
type Real struct{}

// Now returns the current UTC time.
func (Real) Now() time.Time { return time.Now().UTC() }

// Fixed is a clock pinned to a specific instant (tests).
type Fixed struct{ T time.Time }

// Now returns the pinned time.
func (f Fixed) Now() time.Time { return f.T.UTC() }
