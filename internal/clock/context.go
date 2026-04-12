package clock

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type key string

var (
	testClockIDKey   key = "test_clock_id"
	simulatedTimeKey key = "simulated_time"
)

// WithTestClock returns a new context with the given TestClockID and SimulatedTime.
func WithTestClock(ctx context.Context, id uuid.UUID, t time.Time) context.Context {
	ctx = context.WithValue(ctx, testClockIDKey, id)
	return context.WithValue(ctx, simulatedTimeKey, t)
}

// FromContext returns the TestClockID and SimulatedTime from the context, if present.
func FromContext(ctx context.Context) (uuid.UUID, time.Time, bool) {
	id, okID := ctx.Value(testClockIDKey).(uuid.UUID)
	t, okTime := ctx.Value(simulatedTimeKey).(time.Time)

	if okID && okTime {
		return id, t, true
	}
	return uuid.Nil, time.Time{}, false
}

// TestClockIDFromContext returns the TestClockID from the context if present.
func TestClockIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(testClockIDKey).(uuid.UUID)
	return id, ok
}
