package bootstrap

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const systemBootstrapStateTable = "system_bootstrap_state"

const (
	StatusInitializing = "initializing"
	StatusActive       = "active"
)

var ErrBootstrapStateNotFound = errors.New("system bootstrap state not found")

type SystemBootstrapState struct {
	ID            bool       `gorm:"column:id"`
	Status        string     `gorm:"column:status"`
	SchemaVersion string     `gorm:"column:schema_version"`
	Checksum      *string    `gorm:"column:checksum"`
	ActivatedAt   *time.Time `gorm:"column:activated_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func loadSystemBootstrapState(ctx context.Context, db *gorm.DB) (*SystemBootstrapState, error) {
	if db == nil {
		return nil, errors.New("bootstrap state requires database handle")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var state SystemBootstrapState
	result := db.WithContext(ctx).Table(systemBootstrapStateTable).
		Select("id, status, schema_version, checksum, activated_at, created_at").
		Where("id = TRUE").
		Limit(1).
		Scan(&state)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrBootstrapStateNotFound
	}

	state.Status = strings.ToLower(strings.TrimSpace(state.Status))
	state.SchemaVersion = strings.TrimSpace(state.SchemaVersion)
	if state.Checksum != nil {
		trimmed := strings.TrimSpace(*state.Checksum)
		state.Checksum = &trimmed
	}
	return &state, nil
}
