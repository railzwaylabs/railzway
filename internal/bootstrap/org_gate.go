package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type OrgGate interface {
	MustBeActive(ctx context.Context, orgID snowflake.ID) error
}

type orgGate struct {
	db *gorm.DB
}

func NewOrgGate(db *gorm.DB) OrgGate {
	return &orgGate{db: db}
}

func (g *orgGate) MustBeActive(ctx context.Context, orgID snowflake.ID) error {
	if orgID == 0 {
		return errors.New("org id is required")
	}
	var state OrgBootstrapState
	result := g.db.WithContext(ctx).Table(orgBootstrapStateTable).
		Select("org_id, status, activated_at, suspended_at, terminated_at").
		Where("org_id = ?", orgID).
		Limit(1).
		Scan(&state)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrgBootstrapStateNotFound
	}

	status := strings.ToLower(strings.TrimSpace(state.Status))
	if status != string(OrgStatusActive) {
		return fmt.Errorf("%w: status=%s", ErrOrgNotActive, status)
	}

	return nil
}
