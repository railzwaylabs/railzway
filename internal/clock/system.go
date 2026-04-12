package clock

import (
	"context"
	"time"
)

// SystemClock returns time from the system clock, unless overridden by context.
type SystemClock struct{}

// Now returns the current UTC time, or a simulated time from context if present.
func (SystemClock) Now(ctx context.Context) time.Time {
	if _, t, ok := FromContext(ctx); ok {
		return t
	}
	return time.Now().UTC()
}
