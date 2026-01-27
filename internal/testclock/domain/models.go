package domain

import (
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
)

var (
	ErrTestClockNotFound = errors.New("test clock not found")
	ErrInvalidStatus     = errors.New("invalid test clock status")
	ErrAdvanceInProgress = errors.New("test clock advance already in progress")
	ErrStateNotFound     = errors.New("test clock state not found")
	ErrInvalidState      = errors.New("invalid test clock state")
	ErrStateMismatch     = errors.New("test clock state mismatch")
)

type TestClockStatus string

const (
	TestClockStatusActive    TestClockStatus = "active"
	TestClockStatusAdvancing TestClockStatus = "advancing"
	TestClockStatusPaused    TestClockStatus = "paused"
)

type TestClockStateStatus string

const (
	TestClockStateStatusIdle      TestClockStateStatus = "idle"
	TestClockStateStatusAdvancing TestClockStateStatus = "advancing"
	TestClockStateStatusSucceeded TestClockStateStatus = "succeeded"
	TestClockStateStatusFailed    TestClockStateStatus = "failed"
)

type TestClock struct {
	ID            snowflake.ID    `gorm:"primaryKey" json:"id"`
	OrgID         snowflake.ID    `gorm:"not null;index" json:"organization_id"`
	Name          string          `gorm:"not null" json:"name"`
	SimulatedTime time.Time       `gorm:"not null;column:simulated_time" json:"current_time"`
	Status        TestClockStatus `gorm:"not null" json:"status"`
	CreatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (TestClock) TableName() string {
	return "test_clocks"
}

type TestClockState struct {
	TestClockID snowflake.ID         `gorm:"primaryKey;column:test_clock_id" json:"test_clock_id"`
	OrgID       snowflake.ID         `gorm:"not null;index" json:"organization_id"`
	CurrentTime time.Time            `gorm:"not null;column:simulated_time" json:"current_time"`
	AdvancingTo *time.Time           `gorm:"column:advancing_to" json:"advancing_to,omitempty"`
	Status      TestClockStateStatus `gorm:"not null" json:"status"`
	LastError   *string              `gorm:"column:last_error" json:"last_error,omitempty"`
	UpdatedAt   time.Time            `gorm:"not null;column:updated_at" json:"updated_at"`
}

func (TestClockState) TableName() string {
	return "test_clock_state"
}
