package clock

import (
	"context"
	"time"
)

type Clock interface {
	Now(ctx context.Context) time.Time
}
