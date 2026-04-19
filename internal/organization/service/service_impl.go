package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/ledger"
	"github.com/railzwaylabs/railzway/internal/organization/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const organizationCacheTTL = 15 * time.Second

type service struct {
	db    *gorm.DB
	repo  domain.Repository
	audit *auditlog.Service
	cache *redis.Client
}

type Params struct {
	fx.In

	DB    *gorm.DB
	Repo  domain.Repository
	Audit *auditlog.Service `optional:"true"`
	Cache *redis.Client     `name:"redis_cache" optional:"true"`
}

func NewService(p Params) domain.Service {
	return &service{
		db:    p.DB,
		repo:  p.Repo,
		audit: p.Audit,
		cache: p.Cache,
	}
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, req domain.CreateOrganizationRequest) (*domain.OrganizationResponse, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidUser
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidName
	}

	now := time.Now().UTC()
	orgID := uuid.New()
	org := domain.Organization{
		ID:           orgID,
		Name:         name,
		Slug:         slugify(name),
		CountryCode:  strings.TrimSpace(req.CountryCode),
		TimezoneName: strings.TrimSpace(req.TimezoneName),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	var defaultFormat *domain.OrganizationInvoiceNumberFormat
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)
		if err := repo.CreateOrganization(ctx, org); err != nil {
			return err
		}

		member := domain.OrganizationMember{
			ID:        uuid.New(),
			OrgID:     orgID,
			UserID:    userID,
			Role:      domain.RoleOwner,
			CreatedAt: now,
		}

		if err := repo.AddMember(ctx, member); err != nil {
			return err
		}

		accounts := ledger.DefaultAccounts(orgID)
		if len(accounts) > 0 {
			if err := tx.WithContext(ctx).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&accounts).Error; err != nil {
				return err
			}
		}

		prefs := defaultBillingPreferences(orgID, now, req.TimezoneName)
		if err := repo.UpsertBillingPreferences(ctx, prefs); err != nil {
			return err
		}
		created, err := ensureDefaultInvoiceFormat(ctx, repo, orgID, prefs.InvoiceNumberFormat, prefs.InvoiceSequenceScope, now)
		if err != nil {
			return err
		}
		defaultFormat = created

		return nil
	})
	if err != nil {
		return nil, err
	}

	resp := &domain.OrganizationResponse{
		ID:           orgID.String(),
		Name:         org.Name,
		Slug:         org.Slug,
		CountryCode:  org.CountryCode,
		TimezoneName: org.TimezoneName,
	}
	s.setOrganizationCache(ctx, orgID, *resp)
	s.recordAudit(ctx, orgID, "organization.create", "organization", resp.ID, nil, resp, nil)
	if defaultFormat != nil {
		s.recordAudit(ctx, orgID, "organization.invoice_format.create", "organization_invoice_number_format", defaultFormat.ID.String(), nil, defaultFormat, map[string]interface{}{
			"source": "default",
		})
	}
	return resp, nil
}

func (s *service) Update(ctx context.Context, userID uuid.UUID, orgID string, req domain.UpdateOrganizationRequest) (*domain.OrganizationResponse, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidUser
	}

	raw := strings.TrimSpace(orgID)
	if raw == "" {
		return nil, domain.ErrInvalidOrganization
	}
	parsedOrgID, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.ErrInvalidOrganization
	}

	updates := domain.Organization{
		ID:        parsedOrgID,
		UpdatedAt: time.Now().UTC(),
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidName
		}
		updates.Name = name
	}
	if req.CountryCode != nil {
		code := strings.TrimSpace(*req.CountryCode)
		if code == "" {
			return nil, domain.ErrInvalidCountry
		}
		updates.CountryCode = code
	}
	if req.TimezoneName != nil {
		tz := strings.TrimSpace(*req.TimezoneName)
		if tz == "" {
			return nil, domain.ErrInvalidTimezone
		}
		updates.TimezoneName = tz
	}

	before, err := s.GetByID(ctx, parsedOrgID.String())
	if err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, updates); err != nil {
		return nil, err
	}
	s.deleteOrganizationCache(ctx, parsedOrgID)

	updated, err := s.GetByID(ctx, parsedOrgID.String())
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, parsedOrgID, "organization.update", "organization", parsedOrgID.String(), before, updated, nil)
	return updated, nil
}

func defaultBillingPreferences(orgID uuid.UUID, now time.Time, timezone string) domain.OrganizationBillingPreferences {
	defaultTimezone := "UTC"
	if tz := strings.TrimSpace(timezone); tz != "" {
		defaultTimezone = tz
	}
	return domain.OrganizationBillingPreferences{
		OrgID:                orgID,
		Currency:             "USD",
		Timezone:             defaultTimezone,
		InvoicePrefix:        "INV",
		InvoiceNumberFormat:  "{PREFIX}-{YYYY}{MM}-{SEQ:6}",
		InvoiceSequenceScope: "org_month",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func ensureDefaultInvoiceFormat(ctx context.Context, repo domain.Repository, orgID uuid.UUID, format, scope string, now time.Time) (*domain.OrganizationInvoiceNumberFormat, error) {
	formats, err := repo.ListInvoiceNumberFormats(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if len(formats) > 0 {
		return nil, nil
	}
	entry := domain.OrganizationInvoiceNumberFormat{
		ID:            uuid.New(),
		OrgID:         orgID,
		Format:        format,
		SequenceScope: scope,
		EffectiveFrom: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateInvoiceNumberFormat(ctx, entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*domain.OrganizationResponse, error) {
	raw := strings.TrimSpace(id)
	if raw == "" {
		return nil, domain.ErrInvalidOrganization
	}
	orgID, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.ErrInvalidOrganization
	}
	if cached, ok := s.getOrganizationCache(ctx, orgID); ok {
		return &cached, nil
	}

	var org domain.Organization
	if err := s.db.WithContext(ctx).First(&org, "id = ?", orgID).Error; err != nil {
		return nil, err
	}

	resp := domain.OrganizationResponse{
		ID:           org.ID.String(),
		Name:         org.Name,
		Slug:         org.Slug,
		CountryCode:  org.CountryCode,
		TimezoneName: org.TimezoneName,
	}
	s.setOrganizationCache(ctx, orgID, resp)
	return &resp, nil
}

func (s *service) ListOrganizationsByUser(ctx context.Context, userID uuid.UUID) ([]domain.OrganizationListResponseItem, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidUser
	}

	items, err := s.repo.ListOrganizationsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := make([]domain.OrganizationListResponseItem, 0, len(items))
	for _, item := range items {
		resp = append(resp, domain.OrganizationListResponseItem{
			ID:        item.ID.String(),
			Name:      item.Name,
			Role:      item.Role,
			CreatedAt: item.CreatedAt,
		})
	}

	return resp, nil
}

func (s *service) ListMembers(ctx context.Context, orgID string) ([]domain.OrganizationMemberInfo, error) {
	raw := strings.TrimSpace(orgID)
	if raw == "" {
		return nil, domain.ErrInvalidOrganization
	}
	parsedOrgID, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.ErrInvalidOrganization
	}

	return s.repo.ListMembers(ctx, parsedOrgID)
}

func (s *service) IsMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	return s.repo.IsMember(ctx, orgID, userID)
}

func (s *service) InviteMembers(ctx context.Context, userID uuid.UUID, orgID string, invites []domain.InviteRequest) error {
	if userID == uuid.Nil {
		return domain.ErrInvalidUser
	}

	raw := strings.TrimSpace(orgID)
	if raw == "" {
		return domain.ErrInvalidOrganization
	}
	parsedOrgID, err := uuid.Parse(raw)
	if err != nil {
		return domain.ErrInvalidOrganization
	}

	now := time.Now().UTC()
	pending := make([]domain.OrganizationInvite, 0, len(invites))
	for _, invite := range invites {
		email := strings.TrimSpace(invite.Email)
		if email == "" {
			return domain.ErrInvalidEmail
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return domain.ErrInvalidEmail
		}

		role := strings.TrimSpace(invite.Role)
		if !isValidRole(role) {
			return domain.ErrInvalidRole
		}

		pending = append(pending, domain.OrganizationInvite{
			ID:        uuid.New(),
			OrgID:     parsedOrgID,
			Email:     email,
			Role:      role,
			Status:    domain.InviteStatusPending,
			InvitedBy: userID,
			CreatedAt: now,
		})
	}

	return s.repo.CreateInvites(ctx, pending)
}

func (s *service) AcceptInvite(ctx context.Context, userID uuid.UUID, inviteID string) error {
	if userID == uuid.Nil {
		return domain.ErrInvalidUser
	}

	raw := strings.TrimSpace(inviteID)
	if raw == "" {
		return domain.ErrInvalidInvite
	}
	parsedInviteID, err := uuid.Parse(raw)
	if err != nil {
		return domain.ErrInvalidInvite
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)
		invite, err := repo.GetInvite(ctx, parsedInviteID)
		if err != nil {
			return err
		}
		if invite == nil {
			return domain.ErrInvalidInvite
		}
		if invite.Status != domain.InviteStatusPending {
			return domain.ErrInvalidInvite
		}

		invite.Status = domain.InviteStatusAccepted
		if err := repo.UpdateInvite(ctx, *invite); err != nil {
			return err
		}

		member := domain.OrganizationMember{
			ID:        uuid.New(),
			OrgID:     invite.OrgID,
			UserID:    userID,
			Role:      invite.Role,
			CreatedAt: time.Now().UTC(),
		}
		if err := repo.AddMember(ctx, member); err != nil {
			return err
		}
		s.recordAudit(ctx, invite.OrgID, "organization.invite.accept", "organization_member", member.ID.String(), nil, member, nil)
		return nil
	})
}

func (s *service) SetBillingPreferences(ctx context.Context, userID uuid.UUID, orgID string, req domain.BillingPreferencesRequest) error {
	if userID == uuid.Nil {
		return domain.ErrInvalidUser
	}

	raw := strings.TrimSpace(orgID)
	if raw == "" {
		return domain.ErrInvalidOrganization
	}
	parsedOrgID, err := uuid.Parse(raw)
	if err != nil {
		return domain.ErrInvalidOrganization
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		return domain.ErrInvalidCurrency
	}

	timezone := strings.TrimSpace(req.Timezone)
	if timezone == "" {
		return domain.ErrInvalidTimezone
	}

	invoicePrefix := strings.TrimSpace(req.InvoicePrefix)
	if invoicePrefix == "" {
		invoicePrefix = "INV"
	}

	invoiceNumberFormat := strings.TrimSpace(req.InvoiceNumberFormat)
	if invoiceNumberFormat == "" {
		invoiceNumberFormat = "{PREFIX}-{YYYY}{MM}-{SEQ:6}"
	}

	invoiceSequenceScope := strings.TrimSpace(req.InvoiceSequenceScope)
	if invoiceSequenceScope == "" {
		invoiceSequenceScope = "org_month"
	}

	now := time.Now().UTC()
	prefs := domain.OrganizationBillingPreferences{
		OrgID:                parsedOrgID,
		Currency:             currency,
		Timezone:             timezone,
		InvoicePrefix:        invoicePrefix,
		InvoiceNumberFormat:  invoiceNumberFormat,
		InvoiceSequenceScope: invoiceSequenceScope,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := s.repo.UpsertBillingPreferences(ctx, prefs); err != nil {
		return err
	}
	s.recordAudit(ctx, parsedOrgID, "organization.billing_preferences.upsert", "organization_billing_preferences", parsedOrgID.String(), nil, prefs, nil)
	return nil
}

func (s *service) CreateInvoiceNumberFormat(ctx context.Context, userID uuid.UUID, orgID string, req domain.InvoiceNumberFormatRequest) (*domain.InvoiceNumberFormatResponse, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidUser
	}

	raw := strings.TrimSpace(orgID)
	if raw == "" {
		return nil, domain.ErrInvalidOrganization
	}
	parsedOrgID, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.ErrInvalidOrganization
	}

	format := strings.TrimSpace(req.Format)
	if format == "" || !strings.Contains(format, "{SEQ") {
		return nil, domain.ErrInvalidFormat
	}

	sequenceScope := normalizeSequenceScope(req.SequenceScope)
	if sequenceScope == "" {
		return nil, domain.ErrInvalidSequenceScope
	}

	if req.EffectiveFrom.IsZero() {
		return nil, domain.ErrInvalidEffectiveRange
	}
	if req.EffectiveTo != nil && req.EffectiveTo.Before(req.EffectiveFrom) {
		return nil, domain.ErrInvalidEffectiveRange
	}

	existing, err := s.repo.ListInvoiceNumberFormats(ctx, parsedOrgID)
	if err != nil {
		return nil, err
	}
	for _, item := range existing {
		if rangesOverlap(req.EffectiveFrom, req.EffectiveTo, item.EffectiveFrom, item.EffectiveTo) {
			return nil, domain.ErrOverlappingFormat
		}
	}

	now := time.Now().UTC()
	entry := domain.OrganizationInvoiceNumberFormat{
		ID:            uuid.New(),
		OrgID:         parsedOrgID,
		Format:        format,
		SequenceScope: sequenceScope,
		EffectiveFrom: req.EffectiveFrom.UTC(),
		EffectiveTo:   toUTC(req.EffectiveTo),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreateInvoiceNumberFormat(ctx, entry); err != nil {
		return nil, err
	}

	resp := &domain.InvoiceNumberFormatResponse{
		ID:            entry.ID.String(),
		OrgID:         entry.OrgID.String(),
		Format:        entry.Format,
		SequenceScope: entry.SequenceScope,
		EffectiveFrom: entry.EffectiveFrom,
		EffectiveTo:   entry.EffectiveTo,
		CreatedAt:     entry.CreatedAt,
		UpdatedAt:     entry.UpdatedAt,
	}
	s.recordAudit(ctx, parsedOrgID, "organization.invoice_format.create", "organization_invoice_number_format", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) ListInvoiceNumberFormats(ctx context.Context, orgID string) ([]domain.InvoiceNumberFormatResponse, error) {
	raw := strings.TrimSpace(orgID)
	if raw == "" {
		return nil, domain.ErrInvalidOrganization
	}
	parsedOrgID, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.ErrInvalidOrganization
	}

	formats, err := s.repo.ListInvoiceNumberFormats(ctx, parsedOrgID)
	if err != nil {
		return nil, err
	}

	resp := make([]domain.InvoiceNumberFormatResponse, 0, len(formats))
	for _, item := range formats {
		resp = append(resp, domain.InvoiceNumberFormatResponse{
			ID:            item.ID.String(),
			OrgID:         item.OrgID.String(),
			Format:        item.Format,
			SequenceScope: item.SequenceScope,
			EffectiveFrom: item.EffectiveFrom,
			EffectiveTo:   item.EffectiveTo,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return resp, nil
}

func (s *service) CloseInvoiceNumberFormat(ctx context.Context, userID uuid.UUID, orgID string, formatID string, effectiveTo time.Time) (*domain.InvoiceNumberFormatResponse, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidUser
	}

	raw := strings.TrimSpace(orgID)
	if raw == "" {
		return nil, domain.ErrInvalidOrganization
	}
	parsedOrgID, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.ErrInvalidOrganization
	}

	formatRaw := strings.TrimSpace(formatID)
	if formatRaw == "" {
		return nil, domain.ErrInvalidFormatID
	}
	parsedFormatID, err := uuid.Parse(formatRaw)
	if err != nil {
		return nil, domain.ErrInvalidFormatID
	}

	if effectiveTo.IsZero() {
		return nil, domain.ErrInvalidEffectiveRange
	}

	format, err := s.repo.GetInvoiceNumberFormatByID(ctx, parsedOrgID, parsedFormatID)
	if err != nil {
		return nil, err
	}
	if format == nil {
		return nil, domain.ErrFormatNotFound
	}

	if effectiveTo.Before(format.EffectiveFrom) {
		return nil, domain.ErrInvalidEffectiveRange
	}

	existing, err := s.repo.ListInvoiceNumberFormats(ctx, parsedOrgID)
	if err != nil {
		return nil, err
	}
	for _, item := range existing {
		if item.ID == parsedFormatID {
			continue
		}
		if rangesOverlap(format.EffectiveFrom, &effectiveTo, item.EffectiveFrom, item.EffectiveTo) {
			return nil, domain.ErrOverlappingFormat
		}
	}

	format.EffectiveTo = toUTC(&effectiveTo)
	format.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateInvoiceNumberFormat(ctx, *format); err != nil {
		return nil, err
	}

	resp := &domain.InvoiceNumberFormatResponse{
		ID:            format.ID.String(),
		OrgID:         format.OrgID.String(),
		Format:        format.Format,
		SequenceScope: format.SequenceScope,
		EffectiveFrom: format.EffectiveFrom,
		EffectiveTo:   format.EffectiveTo,
		CreatedAt:     format.CreatedAt,
		UpdatedAt:     format.UpdatedAt,
	}
	s.recordAudit(ctx, parsedOrgID, "organization.invoice_format.close", "organization_invoice_number_format", resp.ID, format, resp, nil)
	return resp, nil
}

func (s *service) LinkChildOrganization(ctx context.Context, parentOrgID, childOrgID uuid.UUID, mode, role string) error {
	if parentOrgID == uuid.Nil || childOrgID == uuid.Nil || parentOrgID == childOrgID {
		return domain.ErrInvalidLink
	}

	mode = strings.TrimSpace(strings.ToUpper(mode))
	if mode == "" {
		return domain.ErrInvalidLink
	}
	if mode != domain.LinkModeManaged && mode != domain.LinkModeIndependent {
		return domain.ErrInvalidLink
	}

	role = strings.TrimSpace(strings.ToUpper(role))
	if role == "" {
		return domain.ErrInvalidRole
	}

	link := domain.OrganizationLink{
		ID:          uuid.New(),
		ParentOrgID: parentOrgID,
		ChildOrgID:  childOrgID,
		Mode:        mode,
		Role:        role,
		Status:      domain.LinkStatusActive,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateLink(ctx, link); err != nil {
		return err
	}
	s.recordAudit(ctx, parentOrgID, "organization.link.create", "organization_link", link.ID.String(), nil, link, nil)
	return nil
}

func isValidRole(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleFinance, domain.RoleOperations, domain.RoleDeveloper, domain.RoleCustomerSupport, domain.RoleAuditor:
		return true
	default:
		return false
	}
}

func normalizeSequenceScope(value string) string {
	scope := strings.ToLower(strings.TrimSpace(value))
	if scope == "" {
		return "org_month"
	}
	switch scope {
	case "org_month", "org_year", "org_global":
		return scope
	default:
		return ""
	}
}

func rangesOverlap(startA time.Time, endA *time.Time, startB time.Time, endB *time.Time) bool {
	if endA != nil && endA.Before(startB) {
		return false
	}
	if endB != nil && endB.Before(startA) {
		return false
	}
	return true
}

func toUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	v := value.UTC()
	return &v
}

func slugify(s string) string {
	raw := strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "org"
	}
	return out
}

func (s *service) recordAudit(ctx context.Context, orgID uuid.UUID, action, resourceType, resourceID string, before, after interface{}, meta map[string]interface{}) {
	if s.audit == nil {
		return
	}
	if orgID == uuid.Nil {
		return
	}

	actorType, actorID := auditlog.ActorFromContext(ctx)
	if strings.TrimSpace(actorType) == "" {
		actorType = "system"
	}

	var beforeJSON []byte
	if before != nil {
		beforeJSON, _ = json.Marshal(before)
	}
	var afterJSON []byte
	if after != nil {
		afterJSON, _ = json.Marshal(after)
	}

	var metaJSON []byte
	merged := mergeMetadata(meta, auditlog.MetadataFromContext(ctx))
	if merged != nil {
		metaJSON, _ = json.Marshal(merged)
	}

	var resourcePtr *string
	if strings.TrimSpace(resourceID) != "" {
		resourcePtr = &resourceID
	}

	requestID := strings.TrimSpace(auditlog.RequestIDFromContext(ctx))
	var requestPtr *string
	if requestID != "" {
		requestPtr = &requestID
	}

	reason := strings.TrimSpace(auditlog.ReasonFromContext(ctx))
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}

	_ = s.audit.Record(ctx, auditlog.RecordInput{
		OrgID:        orgID,
		ActorType:    actorType,
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourcePtr,
		BeforeData:   beforeJSON,
		AfterData:    afterJSON,
		Metadata:     metaJSON,
		Reason:       reasonPtr,
		RequestID:    requestPtr,
	})
}

func organizationCacheKey(orgID uuid.UUID) string {
	return fmt.Sprintf("cache:organization:%s", orgID.String())
}

func (s *service) getOrganizationCache(ctx context.Context, orgID uuid.UUID) (domain.OrganizationResponse, bool) {
	if s.cache == nil {
		return domain.OrganizationResponse{}, false
	}
	raw, err := s.cache.Get(ctx, organizationCacheKey(orgID)).Result()
	if err != nil {
		return domain.OrganizationResponse{}, false
	}
	var cached domain.OrganizationResponse
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return domain.OrganizationResponse{}, false
	}
	return cached, true
}

func (s *service) setOrganizationCache(ctx context.Context, orgID uuid.UUID, resp domain.OrganizationResponse) {
	if s.cache == nil {
		return
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = s.cache.Set(ctx, organizationCacheKey(orgID), raw, organizationCacheTTL).Err()
}

func (s *service) deleteOrganizationCache(ctx context.Context, orgID uuid.UUID) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, organizationCacheKey(orgID)).Err()
}

func mergeMetadata(primary, secondary map[string]interface{}) map[string]interface{} {
	if len(primary) == 0 && len(secondary) == 0 {
		return nil
	}
	merged := map[string]interface{}{}
	for key, value := range secondary {
		merged[key] = value
	}
	for key, value := range primary {
		merged[key] = value
	}
	return merged
}
