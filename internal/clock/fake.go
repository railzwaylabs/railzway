package clock

import (
	"context"
	"time"
)

// FakeClock is a controllable clock for deterministic tests.
type FakeClock struct {
	now time.Time
}

// NewFakeClock returns a FakeClock initialized at the provided time (UTC).
func NewFakeClock(t time.Time) *FakeClock {
	return &FakeClock{now: t.UTC()}
}

// Now returns the current fake time.
func (c *FakeClock) Now(ctx context.Context) time.Time {
	return c.now
}

// Advance moves the fake time forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}
