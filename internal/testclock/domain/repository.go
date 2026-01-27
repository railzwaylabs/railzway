package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/repository"
	"gorm.io/gorm"
)

type Repository interface {
	repository.Repository[TestClock]
	FindActiveByOrg(ctx context.Context, db *gorm.DB, orgID snowflake.ID) ([]TestClock, error)
	FindByIDAndOrg(ctx context.Context, db *gorm.DB, id, orgID snowflake.ID) (*TestClock, error)
}

type repo struct {
	repository.Repository[TestClock]
}

func NewRepository(db *gorm.DB) Repository {
	return &repo{
		Repository: repository.ProvideStore[TestClock](db),
	}
}

func (r *repo) FindActiveByOrg(ctx context.Context, db *gorm.DB, orgID snowflake.ID) ([]TestClock, error) {
	var items []TestClock
	err := db.WithContext(ctx).Where("org_id = ?", orgID).Find(&items).Error
	return items, err
}

func (r *repo) FindByIDAndOrg(ctx context.Context, db *gorm.DB, id, orgID snowflake.ID) (*TestClock, error) {
	var item TestClock
	err := db.WithContext(ctx).Where("id = ? AND org_id = ?", id, orgID).First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}
