package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// ChangeRequestStatus represents the approval workflow state.
type ChangeRequestStatus string

const (
	ChangeRequestStatusPending  ChangeRequestStatus = "PENDING"
	ChangeRequestStatusApproved ChangeRequestStatus = "APPROVED"
	ChangeRequestStatusRejected ChangeRequestStatus = "REJECTED"
	ChangeRequestStatusExecuted ChangeRequestStatus = "EXECUTED"
)

// ChangeRequestType defines the type of billing change.
type ChangeRequestType string

const (
	ChangeRequestTypeReRating ChangeRequestType = "RE_RATING"
)

// BillingChangeRequest represents a request to modify billing data.
type BillingChangeRequest struct {
	ID              snowflake.ID        `gorm:"primaryKey"`
	OrgID           snowflake.ID        `gorm:"not null;index"`
	BillingCycleID  snowflake.ID        `gorm:"not null;index"`
	Type            ChangeRequestType   `gorm:"type:text;not null"`
	Status          ChangeRequestStatus `gorm:"type:text;not null;default:'PENDING'"`
	RequestedBy     snowflake.ID        `gorm:"not null"`
	RequestedByName string              `gorm:"type:text"`
	ApprovedBy      *snowflake.ID       `gorm:"index"`
	ApprovedByName  *string             `gorm:"type:text"`
	Reason          string              `gorm:"type:text;not null"`
	RejectionReason *string             `gorm:"type:text"`
	CreatedAt       time.Time           `gorm:"not null;default:CURRENT_TIMESTAMP"`
	ApprovedAt      *time.Time          `gorm:""`
	ExecutedAt      *time.Time          `gorm:""`
}

// TableName sets the database table name.
func (BillingChangeRequest) TableName() string { return "billing_change_requests" }

// CanApprove checks if a user can approve this request (four-eyes principle).
func (r *BillingChangeRequest) CanApprove(userID snowflake.ID) bool {
	if r.Status != ChangeRequestStatusPending {
		return false
	}
	// Four-eyes: requester cannot approve their own request
	return r.RequestedBy != userID
}
