// Package domain contains persistence models for test clocks.
package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive = "active"
	StatusPaused = "paused"
)

type TestClock struct {
	ID          uuid.UUID `gorm:"primaryKey" json:"id"`
	OrgID       uuid.UUID `gorm:"not null;index" json:"org_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Status      string    `gorm:"type:text;not null" json:"status"`
	CurrentTime time.Time `gorm:"column:clock_time;not null" json:"current_time"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (TestClock) TableName() string { return "test_clocks" }
