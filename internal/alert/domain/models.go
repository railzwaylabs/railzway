package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type AlertFrequency string

const (
	AlertFrequencyPerCustomer AlertFrequency = "per_customer"
)

type AlertApplyTo string

const (
	AlertApplyToAllCustomers AlertApplyTo = "all_customers"
)

type Alert struct {
	ID        snowflake.ID   `gorm:"primaryKey"`
	OrgID     snowflake.ID   `gorm:"index"`
	Name      string         `gorm:"not null"`
	MeterID   snowflake.ID   `gorm:"not null;index"`
	Threshold float64        `gorm:"not null"`
	Frequency AlertFrequency `gorm:"not null"`
	ApplyTo   AlertApplyTo   `gorm:"not null"`
	Active    bool           `gorm:"not null;default:false"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
}
