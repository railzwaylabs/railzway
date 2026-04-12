package clock

import (
	"context"
	"time"
)

// Clock abstracts time access for production and test implementations.
type Clock interface {
	// Now returns the current time. Implementations may use context values
	// to override time for testing.
	Now(ctx context.Context) time.Time
}
