// Package clock provides the time source used by every package that records
// timestamps, so that tests can run against a controllable fake clock.
package clock

import "time"

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// Real is the production clock. It always returns UTC.
type Real struct{}

// Now returns the current wall-clock time in UTC.
func (Real) Now() time.Time { return time.Now().UTC() }
