package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/organization/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

func (r *repository) WithTx(tx *gorm.DB) domain.Repository {
	return &repository{db: tx}
}

func (r *repository) CreateOrganization(ctx context.Context, org domain.Organization) error {
	return r.db.WithContext(ctx).Create(&org).Error
}

func (r *repository) Update(ctx context.Context, org domain.Organization) error {
	return r.db.WithContext(ctx).Model(&domain.Organization{}).Where("id = ?", org.ID).Updates(org).Error
}

func (r *repository) AddMember(ctx context.Context, member domain.OrganizationMember) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO organization_members (id, org_id, user_id, role, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		member.ID,
		member.OrgID,
		member.UserID,
		member.Role,
		member.CreatedAt,
	).Error
}

func (r *repository) ListOrganizationsByUser(ctx context.Context, userID uuid.UUID) ([]domain.OrganizationListItem, error) {
	var items []domain.OrganizationListItem
	err := r.db.WithContext(ctx).Raw(
		`SELECT o.id, o.name, m.role, o.created_at
		 FROM organizations o
		 JOIN organization_members m ON m.org_id = o.id
		 WHERE m.user_id = ?
		 ORDER BY o.created_at ASC`,
		userID,
	).Scan(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]domain.OrganizationMemberInfo, error) {
	var items []domain.OrganizationMemberInfo
	err := r.db.WithContext(ctx).
		Table("organization_members").
		Select("organization_members.user_id, users.email, users.display_name, organization_members.role, organization_members.created_at").
		Joins("JOIN users ON users.id = organization_members.user_id").
		Where("organization_members.org_id = ?", orgID).
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) IsMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.OrganizationMember{}).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) CreateInvites(ctx context.Context, invites []domain.OrganizationInvite) error {
	if len(invites) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&invites).Error
}

func (r *repository) GetInvite(ctx context.Context, inviteID uuid.UUID) (*domain.OrganizationInvite, error) {
	var invite domain.OrganizationInvite
	err := r.db.WithContext(ctx).First(&invite, "id = ?", inviteID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

func (r *repository) UpdateInvite(ctx context.Context, invite domain.OrganizationInvite) error {
	return r.db.WithContext(ctx).Save(&invite).Error
}

func (r *repository) UpsertBillingPreferences(ctx context.Context, prefs domain.OrganizationBillingPreferences) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO organization_billing_preferences (org_id, currency, timezone, invoice_prefix, invoice_number_format, invoice_sequence_scope, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (org_id)
		 DO UPDATE SET currency = EXCLUDED.currency,
		               timezone = EXCLUDED.timezone,
		               invoice_prefix = EXCLUDED.invoice_prefix,
		               invoice_number_format = EXCLUDED.invoice_number_format,
		               invoice_sequence_scope = EXCLUDED.invoice_sequence_scope,
		               updated_at = EXCLUDED.updated_at`,
		prefs.OrgID,
		prefs.Currency,
		prefs.Timezone,
		prefs.InvoicePrefix,
		prefs.InvoiceNumberFormat,
		prefs.InvoiceSequenceScope,
		prefs.CreatedAt,
		prefs.UpdatedAt,
	).Error
}

func (r *repository) CreateInvoiceNumberFormat(ctx context.Context, format domain.OrganizationInvoiceNumberFormat) error {
	return r.db.WithContext(ctx).Create(&format).Error
}

func (r *repository) ListInvoiceNumberFormats(ctx context.Context, orgID uuid.UUID) ([]domain.OrganizationInvoiceNumberFormat, error) {
	var formats []domain.OrganizationInvoiceNumberFormat
	if err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).
		Order("effective_from DESC").
		Find(&formats).Error; err != nil {
		return nil, err
	}
	return formats, nil
}

func (r *repository) GetInvoiceNumberFormatByID(ctx context.Context, orgID, formatID uuid.UUID) (*domain.OrganizationInvoiceNumberFormat, error) {
	var format domain.OrganizationInvoiceNumberFormat
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, formatID).
		First(&format).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &format, nil
}

func (r *repository) UpdateInvoiceNumberFormat(ctx context.Context, format domain.OrganizationInvoiceNumberFormat) error {
	return r.db.WithContext(ctx).Save(&format).Error
}

func (r *repository) CreateLink(ctx context.Context, link domain.OrganizationLink) error {
	return r.db.WithContext(ctx).Create(&link).Error
}

func (r *repository) ListChildLinks(ctx context.Context, parentOrgID uuid.UUID) ([]domain.OrganizationLink, error) {
	var links []domain.OrganizationLink
	if err := r.db.WithContext(ctx).
		Where("parent_org_id = ?", parentOrgID).
		Order("created_at ASC").
		Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

func (r *repository) ListParentLinks(ctx context.Context, childOrgID uuid.UUID) ([]domain.OrganizationLink, error) {
	var links []domain.OrganizationLink
	if err := r.db.WithContext(ctx).
		Where("child_org_id = ?", childOrgID).
		Order("created_at ASC").
		Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}
