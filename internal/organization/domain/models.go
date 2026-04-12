// Package domain contains persistence models for the organization service.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant.
type Organization struct {
	ID           uuid.UUID       `gorm:"primaryKey" json:"id"`
	Name         string          `gorm:"type:text;not null" json:"name"`
	Slug         string          `gorm:"type:text;not null;uniqueIndex:ux_organizations_slug" json:"slug"`
	SupportEmail string          `gorm:"type:text;column:support_email" json:"support_email"`
	IsDefault    bool            `gorm:"column:is_default" json:"is_default"`
	CountryCode  string          `gorm:"column:country_code" json:"country_code"`
	TimezoneName string          `gorm:"column:timezone_name" json:"timezone_name"`
	Metadata     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt    time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the database table name.
func (Organization) TableName() string { return "organizations" }

// OrganizationMember represents membership of a user in an organization.
type OrganizationMember struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	OrgID     uuid.UUID `gorm:"not null;index;uniqueIndex:ux_org_user,priority:1" json:"org_id"`
	UserID    uuid.UUID `gorm:"not null;index;uniqueIndex:ux_org_user,priority:2" json:"user_id"`
	Role      string    `gorm:"type:text;not null" json:"role"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName sets the database table name.
func (OrganizationMember) TableName() string { return "organization_members" }

// OrganizationInvite tracks a pending invite to an organization.
type OrganizationInvite struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	OrgID     uuid.UUID `gorm:"not null;index" json:"org_id"`
	Email     string    `gorm:"type:text;not null" json:"email"`
	Role      string    `gorm:"type:text;not null" json:"role"`
	Status    string    `gorm:"type:text;not null" json:"status"`
	InvitedBy uuid.UUID `gorm:"column:invited_by;not null;index" json:"invited_by"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName sets the database table name.
func (OrganizationInvite) TableName() string { return "organization_invites" }

// OrganizationBillingPreferences stores billing defaults for an organization.
type OrganizationBillingPreferences struct {
	OrgID                uuid.UUID `gorm:"primaryKey" json:"org_id"`
	Currency             string    `gorm:"type:text;not null" json:"currency"`
	Timezone             string    `gorm:"type:text;not null" json:"timezone"`
	InvoicePrefix        string    `gorm:"type:text;not null;default:'INV'" json:"invoice_prefix"`
	InvoiceNumberFormat  string    `gorm:"type:text;not null;default:'{PREFIX}-{YYYY}{MM}-{SEQ:6}'" json:"invoice_number_format"`
	InvoiceSequenceScope string    `gorm:"type:text;not null;default:'org_month'" json:"invoice_sequence_scope"`
	CreatedAt            time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt            time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the database table name.
func (OrganizationBillingPreferences) TableName() string { return "organization_billing_preferences" }

// OrganizationLink represents a parent-child organization relationship.
type OrganizationLink struct {
	ID          uuid.UUID `gorm:"primaryKey" json:"id"`
	ParentOrgID uuid.UUID `gorm:"not null;index;uniqueIndex:ux_organization_links_parent_child,priority:1" json:"parent_org_id"`
	ChildOrgID  uuid.UUID `gorm:"not null;index;uniqueIndex:ux_organization_links_parent_child,priority:2" json:"child_org_id"`
	Mode        string    `gorm:"type:text;not null" json:"mode"`
	Role        string    `gorm:"type:text;not null" json:"role"`
	Status      string    `gorm:"type:text;not null" json:"status"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName sets the database table name.
func (OrganizationLink) TableName() string { return "organization_links" }

// OrganizationInvoiceNumberFormat stores versioned invoice number formats.
type OrganizationInvoiceNumberFormat struct {
	ID            uuid.UUID  `gorm:"primaryKey" json:"id"`
	OrgID         uuid.UUID  `gorm:"not null;index" json:"org_id"`
	Format        string     `gorm:"type:text;not null" json:"format"`
	SequenceScope string     `gorm:"type:text;not null" json:"sequence_scope"`
	EffectiveFrom time.Time  `gorm:"not null" json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
	CreatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName sets the database table name.
func (OrganizationInvoiceNumberFormat) TableName() string {
	return "organization_invoice_number_formats"
}
