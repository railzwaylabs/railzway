package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizationListItem struct {
	ID        uuid.UUID
	Name      string
	Role      string
	CreatedAt time.Time
}

type OrganizationMemberInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	CreateOrganization(ctx context.Context, org Organization) error
	Update(ctx context.Context, org Organization) error
	AddMember(ctx context.Context, member OrganizationMember) error
	ListOrganizationsByUser(ctx context.Context, userID uuid.UUID) ([]OrganizationListItem, error)
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]OrganizationMemberInfo, error)
	IsMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error)

	CreateInvites(ctx context.Context, invites []OrganizationInvite) error
	GetInvite(ctx context.Context, inviteID uuid.UUID) (*OrganizationInvite, error)
	UpdateInvite(ctx context.Context, invite OrganizationInvite) error

	UpsertBillingPreferences(ctx context.Context, prefs OrganizationBillingPreferences) error
	CreateInvoiceNumberFormat(ctx context.Context, format OrganizationInvoiceNumberFormat) error
	ListInvoiceNumberFormats(ctx context.Context, orgID uuid.UUID) ([]OrganizationInvoiceNumberFormat, error)
	GetInvoiceNumberFormatByID(ctx context.Context, orgID, formatID uuid.UUID) (*OrganizationInvoiceNumberFormat, error)
	UpdateInvoiceNumberFormat(ctx context.Context, format OrganizationInvoiceNumberFormat) error

	CreateLink(ctx context.Context, link OrganizationLink) error
	ListChildLinks(ctx context.Context, parentOrgID uuid.UUID) ([]OrganizationLink, error)
	ListParentLinks(ctx context.Context, childOrgID uuid.UUID) ([]OrganizationLink, error)
}
