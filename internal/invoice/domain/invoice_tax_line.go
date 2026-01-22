package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// InvoiceTaxLine captures the tax applied to an invoice at finalization.
type InvoiceTaxLine struct {
	ID        snowflake.ID `gorm:"primaryKey"`
	OrgID     snowflake.ID `gorm:"not null;index"`
	InvoiceID snowflake.ID `gorm:"not null;index"`
	TaxCode   *string      `gorm:"type:text"`
	TaxName   string       `gorm:"type:text;not null"`
	TaxMode   string       `gorm:"type:text;not null"`
	TaxRate   float64      `gorm:"not null"`
	Amount    int64        `gorm:"not null"` // Tax amount in cents
	CreatedAt time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName sets the database table name.
func (InvoiceTaxLine) TableName() string { return "invoice_tax_lines" }
