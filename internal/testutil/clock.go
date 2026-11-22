package testutil

import (
	"time"

	"github.com/kaszasg/kinex-mark/internal/clock"
)

// T0 is the fixed start time of every test clock: 2026-01-05T10:00:00Z.
var T0 = time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)

// NewClock returns a fake clock set to T0.
func NewClock() *clock.Fake {
	return clock.NewFake(T0)
}
