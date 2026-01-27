package scheduler

import (
	"context"

	"gorm.io/gorm"

	testclockctx "github.com/railzwaylabs/railzway/internal/testclock/context"
)

// applyTestClockScope filters the query based on the test_clock_id in the context.
// If a TestClockID is present, it filters for records matching that ID.
// If no TestClockID is present, it filters for records where test_clock_id IS NULL (Production/Real mode).
func applyTestClockScope(ctx context.Context, db *gorm.DB) *gorm.DB {
	if id, _, ok := testclockctx.FromContext(ctx); ok {
		return db.Where("test_clock_id = ?", id)
	}
	return db.Where("test_clock_id IS NULL")
}
