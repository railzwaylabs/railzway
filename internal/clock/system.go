package clock

import (
	"context"
	"time"

	testclockctx "github.com/railzwaylabs/railzway/internal/testclock/context"
)

type SystemClock struct{}

func (SystemClock) Now(ctx context.Context) time.Time {
	if _, t, ok := testclockctx.FromContext(ctx); ok {
		return t
	}
	return time.Now().UTC()
}
