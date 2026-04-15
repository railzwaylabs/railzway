package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RunCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	CreateRun(ctx context.Context, run Run) error
	UpdateRun(ctx context.Context, orgID, runID uuid.UUID, updates map[string]interface{}) error
	FindRunByID(ctx context.Context, orgID, runID uuid.UUID) (*Run, error)
	ListRuns(ctx context.Context, orgID uuid.UUID, limit int, cursor *RunCursor) ([]*Run, error)
}
