package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/customer/domain"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type service struct {
	db    *gorm.DB
	repo  domain.Repository
	audit *auditlog.Service
}

type Params struct {
	fx.In

	DB    *gorm.DB
	Repo  domain.Repository
	Audit *auditlog.Service `optional:"true"`
}

func NewService(p Params) domain.Service {
	return &service{
		db:    p.DB,
		repo:  p.Repo,
		audit: p.Audit,
	}
}

func (s *service) Create(ctx context.Context, req domain.CreateCustomerRequest) (domain.CustomerResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.CustomerResponse{}, domain.ErrInvalidOrganization
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.CustomerResponse{}, domain.ErrInvalidName
	}

	email := strings.TrimSpace(req.Email)
	if email == "" {
		return domain.CustomerResponse{}, domain.ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return domain.CustomerResponse{}, domain.ErrInvalidEmail
	}

	currency := normalizeCurrency(req.Currency)
	if currency != "" && !isValidCurrency(currency) {
		return domain.CustomerResponse{}, domain.ErrInvalidCurrency
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.CustomerResponse{}, err
		}
		if existing != nil {
			return toResponse(*existing), nil
		}
	}

	now := time.Now().UTC()
	customer := domain.Customer{
		ID:         uuid.New(),
		OrgID:      orgID,
		ExternalID: strings.TrimSpace(req.ExternalID),
		Name:       name,
		Email:      email,
		Currency:   currency,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if idempotencyKey != "" {
		customer.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.CustomerResponse{}, findErr
			}
			if existing != nil {
				return toResponse(*existing), nil
			}
		}
		return domain.CustomerResponse{}, err
	}

	resp := toResponse(customer)
	s.recordAudit(ctx, "customer.create", "customer", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) Update(ctx context.Context, id string, req domain.UpdateCustomerRequest) (domain.CustomerResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.CustomerResponse{}, domain.ErrInvalidOrganization
	}
	customerID, err := parseID(id)
	if err != nil {
		return domain.CustomerResponse{}, err
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return domain.CustomerResponse{}, domain.ErrInvalidName
		}
		updates["name"] = name
	}

	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if email == "" {
			return domain.CustomerResponse{}, domain.ErrInvalidEmail
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return domain.CustomerResponse{}, domain.ErrInvalidEmail
		}
		updates["email"] = email
	}

	if req.ExternalID != nil {
		updates["external_id"] = strings.TrimSpace(*req.ExternalID)
	}

	if req.Currency != nil {
		currency := normalizeCurrency(*req.Currency)
		if currency != "" && !isValidCurrency(currency) {
			return domain.CustomerResponse{}, domain.ErrInvalidCurrency
		}
		updates["currency"] = currency
	}

	beforeCustomer, err := s.repo.FindByID(ctx, orgID, customerID)
	if err != nil {
		return domain.CustomerResponse{}, err
	}
	if beforeCustomer == nil {
		return domain.CustomerResponse{}, domain.ErrNotFound
	}

	if err := s.repo.Update(ctx, orgID, customerID, updates); err != nil {
		return domain.CustomerResponse{}, err
	}

	item, err := s.repo.FindByID(ctx, orgID, customerID)
	if err != nil {
		return domain.CustomerResponse{}, err
	}
	if item == nil {
		return domain.CustomerResponse{}, domain.ErrNotFound
	}

	resp := toResponse(*item)
	s.recordAudit(ctx, "customer.update", "customer", resp.ID, toResponse(*beforeCustomer), resp, nil)
	return resp, nil
}

func (s *service) GetByID(ctx context.Context, req domain.GetCustomerRequest) (domain.CustomerResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.CustomerResponse{}, domain.ErrInvalidOrganization
	}
	id, err := parseID(req.ID)
	if err != nil {
		return domain.CustomerResponse{}, err
	}

	item, err := s.repo.FindByID(ctx, orgID, id)
	if err != nil {
		return domain.CustomerResponse{}, err
	}
	if item == nil {
		return domain.CustomerResponse{}, domain.ErrNotFound
	}

	return toResponse(*item), nil
}

func (s *service) List(ctx context.Context, req domain.ListCustomerRequest) (domain.ListCustomerResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListCustomerResponse{}, domain.ErrInvalidOrganization
	}
	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.ListFilter{
		Name:        strings.TrimSpace(req.Name),
		Email:       strings.TrimSpace(req.Email),
		Currency:    normalizeCurrency(req.Currency),
		CreatedFrom: req.CreatedFrom,
		CreatedTo:   req.CreatedTo,
	}

	var cursor *domain.ListCursor
	if token := strings.TrimSpace(req.PageToken); token != "" {
		decoded, err := pagination.DecodeCursor(token)
		if err != nil {
			return domain.ListCustomerResponse{}, domain.ErrInvalidID
		}
		if decoded != nil && decoded.CreatedAt != "" && decoded.ID != "" {
			parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
			if err != nil {
				return domain.ListCustomerResponse{}, domain.ErrInvalidID
			}
			parsedID, err := uuid.Parse(decoded.ID)
			if err != nil {
				return domain.ListCustomerResponse{}, domain.ErrInvalidID
			}
			cursor = &domain.ListCursor{
				ID:        parsedID,
				CreatedAt: parsedTime,
			}
		}
	}

	items, err := s.repo.List(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListCustomerResponse{}, err
	}

	resp := domain.ListCustomerResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Customer) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(items) > pageSize {
			items = items[:pageSize]
		}
	}

	resp.Customers = make([]domain.CustomerResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Customers = append(resp.Customers, toResponse(*item))
	}

	return resp, nil
}

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidID
	}
	return id, nil
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isValidCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func toResponse(item domain.Customer) domain.CustomerResponse {
	return domain.CustomerResponse{
		ID:         item.ID.String(),
		OrgID:      item.OrgID.String(),
		ExternalID: item.ExternalID,
		Name:       item.Name,
		Email:      item.Email,
		Currency:   item.Currency,
		Metadata:   item.Metadata,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}

func (s *service) recordAudit(ctx context.Context, action, resourceType, resourceID string, before, after interface{}, meta map[string]interface{}) {
	if s.audit == nil {
		return
	}
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
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
