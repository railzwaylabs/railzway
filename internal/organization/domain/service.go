package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	RoleOwner           = "OWNER"
	RoleAdmin           = "ADMIN"
	RoleFinance         = "FINANCE"
	RoleOperations      = "OPERATIONS"
	RoleDeveloper       = "DEVELOPER"
	RoleCustomerSupport = "CUSTOMER_SUPPORT"
	RoleAuditor         = "AUDITOR"

	LinkModeManaged     = "MANAGED"
	LinkModeIndependent = "INDEPENDENT"

	LinkStatusActive    = "ACTIVE"
	LinkStatusSuspended = "SUSPENDED"

	InviteStatusPending  = "PENDING"
	InviteStatusAccepted = "ACCEPTED"
	InviteStatusDeclined = "DECLINED"
)

type Service interface {
	Create(ctx context.Context, userID uuid.UUID, req CreateOrganizationRequest) (*OrganizationResponse, error)
	Update(ctx context.Context, userID uuid.UUID, orgID string, req UpdateOrganizationRequest) (*OrganizationResponse, error)
	GetByID(ctx context.Context, id string) (*OrganizationResponse, error)
	ListOrganizationsByUser(ctx context.Context, userID uuid.UUID) ([]OrganizationListResponseItem, error)
	ListMembers(ctx context.Context, orgID string) ([]OrganizationMemberInfo, error)
	IsMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error)
	InviteMembers(ctx context.Context, userID uuid.UUID, orgID string, invites []InviteRequest) error
	AcceptInvite(ctx context.Context, userID uuid.UUID, inviteID string) error
	SetBillingPreferences(ctx context.Context, userID uuid.UUID, orgID string, req BillingPreferencesRequest) error
	CreateInvoiceNumberFormat(ctx context.Context, userID uuid.UUID, orgID string, req InvoiceNumberFormatRequest) (*InvoiceNumberFormatResponse, error)
	ListInvoiceNumberFormats(ctx context.Context, orgID string) ([]InvoiceNumberFormatResponse, error)
	CloseInvoiceNumberFormat(ctx context.Context, userID uuid.UUID, orgID string, formatID string, effectiveTo time.Time) (*InvoiceNumberFormatResponse, error)
	LinkChildOrganization(ctx context.Context, parentOrgID, childOrgID uuid.UUID, mode, role string) error
}

type CreateOrganizationRequest struct {
	Name         string
	CountryCode  string
	TimezoneName string
}

type UpdateOrganizationRequest struct {
	Name         *string
	CountryCode  *string
	TimezoneName *string
}

type InviteRequest struct {
	Email string
	Role  string
}

type BillingPreferencesRequest struct {
	Currency             string
	Timezone             string
	InvoicePrefix        string
	InvoiceNumberFormat  string
	InvoiceSequenceScope string
}

type InvoiceNumberFormatRequest struct {
	Format        string
	SequenceScope string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

type InvoiceNumberFormatResponse struct {
	ID            string     `json:"id"`
	OrgID         string     `json:"org_id"`
	Format        string     `json:"format"`
	SequenceScope string     `json:"sequence_scope"`
	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type OrganizationResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	CountryCode  string `json:"country_code"`
	TimezoneName string `json:"timezone_name"`
}

type OrganizationListResponseItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	ErrInvalidName           = errors.New("invalid_name")
	ErrInvalidCountry        = errors.New("invalid_country")
	ErrInvalidTimezone       = errors.New("invalid_timezone")
	ErrInvalidCurrency       = errors.New("invalid_currency")
	ErrInvalidUser           = errors.New("invalid_user")
	ErrInvalidOrganization   = errors.New("invalid_organization")
	ErrInvalidEmail          = errors.New("invalid_email")
	ErrInvalidRole           = errors.New("invalid_role")
	ErrInvalidInvite         = errors.New("invalid_invite")
	ErrInvalidLink           = errors.New("invalid_link")
	ErrInvalidFormat         = errors.New("invalid_format")
	ErrInvalidFormatID       = errors.New("invalid_format_id")
	ErrInvalidSequenceScope  = errors.New("invalid_sequence_scope")
	ErrInvalidEffectiveRange = errors.New("invalid_effective_range")
	ErrOverlappingFormat     = errors.New("overlapping_invoice_format")
	ErrFormatNotFound        = errors.New("invoice_format_not_found")
	ErrForbidden             = errors.New("forbidden")
)
