package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

func (r *repository) WithTx(tx *gorm.DB) domain.Repository {
	return &repository{db: tx}
}

func (r *repository) CreateRun(ctx context.Context, run domain.Run) error {
	return r.db.WithContext(ctx).Create(&run).Error
}

func (r *repository) UpdateRun(ctx context.Context, orgID, runID uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Run{}).
		Where("id = ? AND org_id = ?", runID, orgID).
		Updates(updates).Error
}

func (r *repository) FindRunByID(ctx context.Context, orgID, runID uuid.UUID) (*domain.Run, error) {
	var run domain.Run
	err := r.db.WithContext(ctx).
		Where("id = ? AND org_id = ?", runID, orgID).
		First(&run).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

func (r *repository) ListRuns(ctx context.Context, orgID uuid.UUID, limit int, cursor *domain.RunCursor) ([]*domain.Run, error) {
	var runs []*domain.Run
	stmt := r.db.WithContext(ctx).Model(&domain.Run{}).Where("org_id = ?", orgID)

	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&runs).Error; err != nil {
		return nil, err
	}
	return runs, nil
}
