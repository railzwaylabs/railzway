package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type service struct {
	db     *gorm.DB
	repo   domain.Repository
	meters usagedomain.Service
	audit  *auditlog.Service
}

type Params struct {
	fx.In

	DB     *gorm.DB
	Repo   domain.Repository
	Meters usagedomain.Service
	Audit  *auditlog.Service `optional:"true"`
}

func NewService(p Params) domain.Service {
	return &service{
		db:     p.DB,
		repo:   p.Repo,
		meters: p.Meters,
		audit:  p.Audit,
	}
}

func (s *service) Create(ctx context.Context, req domain.CreateFeatureRequest) (domain.FeatureResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.FeatureResponse{}, domain.ErrInvalidOrganization
	}

	code := normalizeCode(req.Code)
	if code == "" {
		return domain.FeatureResponse{}, domain.ErrInvalidCode
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.FeatureResponse{}, domain.ErrInvalidName
	}

	featureType := normalizeType(req.FeatureType)
	if featureType == "" {
		return domain.FeatureResponse{}, domain.ErrInvalidType
	}

	meterID, err := s.parseMeterID(ctx, featureType, req.MeterID)
	if err != nil {
		return domain.FeatureResponse{}, err
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.FeatureResponse{}, err
		}
		if existing != nil {
			return toResponse(*existing), nil
		}
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	now := time.Now().UTC()
	feature := domain.Feature{
		ID:          uuid.New(),
		OrgID:       orgID,
		Code:        code,
		Name:        name,
		Description: trimOptional(req.Description),
		Type:        domain.FeatureType(featureType),
		MeterID:     meterID,
		Active:      active,
		Metadata:    json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if idempotencyKey != "" {
		feature.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.Create(ctx, feature); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.FeatureResponse{}, findErr
			}
			if existing != nil {
				return toResponse(*existing), nil
			}
		}
		return domain.FeatureResponse{}, err
	}

	resp := toResponse(feature)
	s.recordAudit(ctx, "feature.create", "feature", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) Update(ctx context.Context, id string, req domain.UpdateFeatureRequest) (domain.FeatureResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.FeatureResponse{}, domain.ErrInvalidOrganization
	}
	featureID, err := parseID(id)
	if err != nil {
		return domain.FeatureResponse{}, err
	}

	updates := map[string]interface{}{}
	var current *domain.Feature
	if req.FeatureType != nil || req.MeterID != nil {
		item, err := s.repo.FindByID(ctx, orgID, featureID)
		if err != nil {
			return domain.FeatureResponse{}, err
		}
		if item == nil {
			return domain.FeatureResponse{}, domain.ErrNotFound
		}
		current = item
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return domain.FeatureResponse{}, domain.ErrInvalidName
		}
		updates["name"] = name
	}
	if req.Description != nil {
		updates["description"] = trimOptional(req.Description)
	}
	targetType := ""
	if req.FeatureType != nil {
		targetType = normalizeType(*req.FeatureType)
		if targetType == "" {
			return domain.FeatureResponse{}, domain.ErrInvalidType
		}
		updates["feature_type"] = targetType
	} else if current != nil {
		targetType = string(current.Type)
	}

	if req.MeterID != nil {
		meterID, err := s.parseMeterID(ctx, targetType, req.MeterID)
		if err != nil {
			return domain.FeatureResponse{}, err
		}
		updates["meter_id"] = meterID
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now().UTC()
	}

	if err := s.repo.Update(ctx, orgID, featureID, updates); err != nil {
		return domain.FeatureResponse{}, err
	}
	item, err := s.repo.FindByID(ctx, orgID, featureID)
	if err != nil {
		return domain.FeatureResponse{}, err
	}
	if item == nil {
		return domain.FeatureResponse{}, domain.ErrNotFound
	}
	resp := toResponse(*item)
	s.recordAudit(ctx, "feature.update", "feature", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) GetByID(ctx context.Context, req domain.GetFeatureRequest) (domain.FeatureResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.FeatureResponse{}, domain.ErrInvalidOrganization
	}
	featureID, err := parseID(req.ID)
	if err != nil {
		return domain.FeatureResponse{}, err
	}
	item, err := s.repo.FindByID(ctx, orgID, featureID)
	if err != nil {
		return domain.FeatureResponse{}, err
	}
	if item == nil {
		return domain.FeatureResponse{}, domain.ErrNotFound
	}
	return toResponse(*item), nil
}

func (s *service) List(ctx context.Context, req domain.ListFeatureRequest) (domain.ListFeatureResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListFeatureResponse{}, domain.ErrInvalidOrganization
	}
	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.ListFilter{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		FeatureType: normalizeType(req.FeatureType),
		Active:      req.Active,
	}

	var cursor *domain.ListCursor
	if token := strings.TrimSpace(req.PageToken); token != "" {
		decoded, err := pagination.DecodeCursor(token)
		if err != nil {
			return domain.ListFeatureResponse{}, domain.ErrInvalidID
		}
		if decoded != nil && decoded.CreatedAt != "" && decoded.ID != "" {
			parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
			if err != nil {
				return domain.ListFeatureResponse{}, domain.ErrInvalidID
			}
			parsedID, err := uuid.Parse(decoded.ID)
			if err != nil {
				return domain.ListFeatureResponse{}, domain.ErrInvalidID
			}
			cursor = &domain.ListCursor{
				ID:        parsedID,
				CreatedAt: parsedTime,
			}
		}
	}

	items, err := s.repo.List(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListFeatureResponse{}, err
	}

	resp := domain.ListFeatureResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Feature) string {
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

	resp.Features = make([]domain.FeatureResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Features = append(resp.Features, toResponse(*item))
	}
	return resp, nil
}

func toResponse(item domain.Feature) domain.FeatureResponse {
	var meterID *string
	if item.MeterID != nil {
		value := item.MeterID.String()
		meterID = &value
	}
	return domain.FeatureResponse{
		ID:          item.ID.String(),
		OrgID:       item.OrgID.String(),
		Code:        item.Code,
		Name:        item.Name,
		Description: item.Description,
		FeatureType: string(item.Type),
		MeterID:     meterID,
		Active:      item.Active,
		Metadata:    item.Metadata,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, domain.ErrInvalidID
	}
	return id, nil
}

func normalizeType(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case string(domain.FeatureTypeBoolean), string(domain.FeatureTypeMetered):
		return trimmed
	default:
		return ""
	}
}

func normalizeCode(value string) string {
	code := strings.ToLower(strings.TrimSpace(value))
	code = strings.ReplaceAll(code, " ", "-")
	return code
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *service) parseMeterID(ctx context.Context, featureType string, meterID *string) (*uuid.UUID, error) {
	if featureType != string(domain.FeatureTypeMetered) {
		return nil, nil
	}
	if meterID == nil {
		return nil, domain.ErrInvalidMeter
	}
	trimmed := strings.TrimSpace(*meterID)
	if trimmed == "" {
		return nil, domain.ErrInvalidMeter
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil || parsed == uuid.Nil {
		return nil, domain.ErrInvalidMeter
	}
	if s.meters != nil {
		if _, err := s.meters.GetMeterByID(ctx, usagedomain.GetMeterRequest{ID: parsed.String()}); err != nil {
			if errors.Is(err, usagedomain.ErrNotFound) {
				return nil, domain.ErrInvalidMeter
			}
			return nil, err
		}
	}
	return &parsed, nil
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
	reqID := strings.TrimSpace(auditlog.RequestIDFromContext(ctx))
	var reqIDPtr *string
	if reqID != "" {
		reqIDPtr = &reqID
	}

	_ = s.audit.Record(ctx, auditlog.RecordInput{
		OrgID:        orgID,
		ActorType:    actorType,
		ActorID:      actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		BeforeData:   beforeJSON,
		AfterData:    afterJSON,
		Metadata:     defaultJSON(meta),
		RequestID:    reqIDPtr,
	})
}

func defaultJSON(value map[string]interface{}) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}
