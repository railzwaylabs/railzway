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
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/usage/domain"
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

func (s *service) CreateMeter(ctx context.Context, req domain.CreateMeterRequest) (domain.MeterResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.MeterResponse{}, domain.ErrInvalidOrganization
	}

	code := normalizeCode(req.Code)
	if code == "" {
		return domain.MeterResponse{}, domain.ErrInvalidCode
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.MeterResponse{}, domain.ErrInvalidName
	}

	aggregation := strings.ToLower(strings.TrimSpace(req.Aggregation))
	if !isValidAggregation(aggregation) {
		return domain.MeterResponse{}, domain.ErrInvalidAggregation
	}

	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		return domain.MeterResponse{}, domain.ErrInvalidUnit
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindMeterByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.MeterResponse{}, err
		}
		if existing != nil {
			return toMeterResponse(*existing), nil
		}
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	now := time.Now().UTC()
	meter := domain.Meter{
		ID:          uuid.New(),
		OrgID:       orgID,
		Code:        code,
		Name:        name,
		Aggregation: aggregation,
		Unit:        unit,
		Active:      active,
		Metadata:    json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if idempotencyKey != "" {
		meter.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.CreateMeter(ctx, meter); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindMeterByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.MeterResponse{}, findErr
			}
			if existing != nil {
				return toMeterResponse(*existing), nil
			}
		}
		return domain.MeterResponse{}, err
	}

	resp := toMeterResponse(meter)
	s.recordAudit(ctx, "meter.create", "meter", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) UpdateMeter(ctx context.Context, id string, req domain.UpdateMeterRequest) (domain.MeterResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.MeterResponse{}, domain.ErrInvalidOrganization
	}

	meterID, err := parseID(id)
	if err != nil {
		return domain.MeterResponse{}, domain.ErrInvalidID
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return domain.MeterResponse{}, domain.ErrInvalidName
		}
		updates["name"] = name
	}
	if req.Aggregation != nil {
		aggregation := strings.ToLower(strings.TrimSpace(*req.Aggregation))
		if !isValidAggregation(aggregation) {
			return domain.MeterResponse{}, domain.ErrInvalidAggregation
		}
		updates["aggregation"] = aggregation
	}
	if req.Unit != nil {
		unit := strings.TrimSpace(*req.Unit)
		if unit == "" {
			return domain.MeterResponse{}, domain.ErrInvalidUnit
		}
		updates["unit"] = unit
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	beforeMeter, err := s.repo.FindMeterByID(ctx, orgID, meterID)
	if err != nil {
		return domain.MeterResponse{}, err
	}
	if beforeMeter == nil {
		return domain.MeterResponse{}, domain.ErrNotFound
	}

	if err := s.repo.UpdateMeter(ctx, orgID, meterID, updates); err != nil {
		return domain.MeterResponse{}, err
	}

	item, err := s.repo.FindMeterByID(ctx, orgID, meterID)
	if err != nil {
		return domain.MeterResponse{}, err
	}
	if item == nil {
		return domain.MeterResponse{}, domain.ErrNotFound
	}

	resp := toMeterResponse(*item)
	s.recordAudit(ctx, "meter.update", "meter", resp.ID, toMeterResponse(*beforeMeter), resp, nil)
	return resp, nil
}

func (s *service) GetMeterByID(ctx context.Context, req domain.GetMeterRequest) (domain.MeterResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.MeterResponse{}, domain.ErrInvalidOrganization
	}

	id, err := parseID(req.ID)
	if err != nil {
		return domain.MeterResponse{}, domain.ErrInvalidID
	}

	item, err := s.repo.FindMeterByID(ctx, orgID, id)
	if err != nil {
		return domain.MeterResponse{}, err
	}
	if item == nil {
		return domain.MeterResponse{}, domain.ErrNotFound
	}

	return toMeterResponse(*item), nil
}

func (s *service) ListMeters(ctx context.Context, req domain.ListMeterRequest) (domain.ListMeterResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListMeterResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.MeterListFilter{
		Code:   normalizeCode(req.Code),
		Name:   strings.TrimSpace(req.Name),
		Active: req.Active,
	}
	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListMeterResponse{}, err
	}

	items, err := s.repo.ListMeters(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListMeterResponse{}, err
	}

	resp := domain.ListMeterResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Meter) string {
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

	resp.Meters = make([]domain.MeterResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Meters = append(resp.Meters, toMeterResponse(*item))
	}

	return resp, nil
}

func (s *service) IngestUsage(ctx context.Context, req domain.IngestUsageRequest) (domain.UsageEventResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.UsageEventResponse{}, domain.ErrInvalidOrganization
	}

	meterCodeRaw := strings.TrimSpace(req.MeterCode)
	if meterCodeRaw == "" {
		return domain.UsageEventResponse{}, domain.ErrInvalidMeter
	}

	customerID, err := parseID(req.CustomerID)
	if err != nil {
		return domain.UsageEventResponse{}, domain.ErrInvalidCustomer
	}
	if req.RecordedAt.IsZero() {
		return domain.UsageEventResponse{}, domain.ErrInvalidRecordedAt
	}
	if req.Value < 0 {
		return domain.UsageEventResponse{}, domain.ErrInvalidValue
	}

	var meter *domain.Meter
	code := normalizeCode(meterCodeRaw)
	meter, err = s.repo.FindMeterByCode(ctx, orgID, code)
	if err != nil {
		return domain.UsageEventResponse{}, err
	}
	if meter == nil {
		return domain.UsageEventResponse{}, domain.ErrInvalidMeter
	}
	if !meter.Active {
		return domain.UsageEventResponse{}, domain.ErrInvalidMeter
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindUsageByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.UsageEventResponse{}, err
		}
		if existing != nil {
			return toUsageEventResponse(*existing), nil
		}
	}

	now := time.Now().UTC()
	event := domain.UsageEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		MeterID:    meter.ID,
		MeterCode:  meter.Code,
		CustomerID: customerID,
		Value:      req.Value,
		RecordedAt: req.RecordedAt.UTC(),
		Status:     domain.StatusAccepted,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if idempotencyKey != "" {
		event.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.CreateUsageEvent(ctx, event); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindUsageByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.UsageEventResponse{}, findErr
			}
			if existing != nil {
				return toUsageEventResponse(*existing), nil
			}
		}
		return domain.UsageEventResponse{}, err
	}

	return toUsageEventResponse(event), nil
}

func (s *service) ListUsage(ctx context.Context, req domain.ListUsageRequest) (domain.ListUsageResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListUsageResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.UsageListFilter{
		Status: strings.ToLower(strings.TrimSpace(req.Status)),
	}
	if req.MeterID != "" {
		id, err := parseID(req.MeterID)
		if err != nil {
			return domain.ListUsageResponse{}, domain.ErrInvalidMeter
		}
		filter.MeterID = id
	}
	if req.CustomerID != "" {
		id, err := parseID(req.CustomerID)
		if err != nil {
			return domain.ListUsageResponse{}, domain.ErrInvalidCustomer
		}
		filter.CustomerID = id
	}
	if filter.Status != "" && !isValidStatus(filter.Status) {
		return domain.ListUsageResponse{}, domain.ErrInvalidStatus
	}
	filter.RecordedFrom = req.RecordedFrom
	filter.RecordedTo = req.RecordedTo

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListUsageResponse{}, err
	}

	items, err := s.repo.ListUsageEvents(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListUsageResponse{}, err
	}

	resp := domain.ListUsageResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.UsageEvent) string {
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

	resp.Events = make([]domain.UsageEventResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Events = append(resp.Events, toUsageEventResponse(*item))
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

func decodeCursor(token string) (*domain.ListCursor, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil
	}
	decoded, err := pagination.DecodeCursor(raw)
	if err != nil {
		return nil, domain.ErrInvalidID
	}
	if decoded == nil || decoded.ID == "" || decoded.CreatedAt == "" {
		return nil, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
	if err != nil {
		return nil, domain.ErrInvalidID
	}
	parsedID, err := uuid.Parse(decoded.ID)
	if err != nil {
		return nil, domain.ErrInvalidID
	}
	return &domain.ListCursor{
		ID:        parsedID,
		CreatedAt: parsedTime,
	}, nil
}

func normalizeCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidAggregation(value string) bool {
	switch value {
	case domain.AggregationSum, domain.AggregationCount, domain.AggregationMax, domain.AggregationLast, domain.AggregationAvg:
		return true
	default:
		return false
	}
}

func isValidStatus(value string) bool {
	switch value {
	case domain.StatusAccepted, domain.StatusEnriched, domain.StatusRated:
		return true
	default:
		return false
	}
}

func toMeterResponse(meter domain.Meter) domain.MeterResponse {
	return domain.MeterResponse{
		ID:          meter.ID.String(),
		OrgID:       meter.OrgID.String(),
		Code:        meter.Code,
		Name:        meter.Name,
		Aggregation: meter.Aggregation,
		Unit:        meter.Unit,
		Active:      meter.Active,
		Metadata:    meter.Metadata,
		CreatedAt:   meter.CreatedAt,
		UpdatedAt:   meter.UpdatedAt,
	}
}

func toUsageEventResponse(event domain.UsageEvent) domain.UsageEventResponse {
	return domain.UsageEventResponse{
		ID:         event.ID.String(),
		OrgID:      event.OrgID.String(),
		MeterID:    event.MeterID.String(),
		MeterCode:  event.MeterCode,
		CustomerID: event.CustomerID.String(),
		Value:      event.Value,
		RecordedAt: event.RecordedAt,
		Status:     event.Status,
		Metadata:   event.Metadata,
		CreatedAt:  event.CreatedAt,
		UpdatedAt:  event.UpdatedAt,
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
