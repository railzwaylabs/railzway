package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Params struct {
	fx.In

	DB    *gorm.DB
	Log   *zap.Logger
	GenID *snowflake.Node
	Repo  domain.Repository
}

type Service struct {
	db    *gorm.DB
	log   *zap.Logger
	repo  domain.Repository
	genID *snowflake.Node
}

func New(p Params) domain.Service {
	return &Service{
		db:    p.DB,
		log:   p.Log.Named("feature.service"),
		repo:  p.Repo,
		genID: p.GenID,
	}
}

func (s *Service) List(ctx context.Context, req domain.ListRequest) (domain.ListResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.ListResponse{}, domain.ErrInvalidOrganization
	}
	orgIDValue := int64(orgID)

	filter := domain.ListRequest{
		Name:        strings.TrimSpace(req.Name),
		Code:        strings.TrimSpace(req.Code),
		FeatureType: req.FeatureType,
		Active:      req.Active,
		SortBy:      strings.TrimSpace(req.SortBy),
		OrderBy:     strings.TrimSpace(req.OrderBy),
		PageToken:   req.PageToken,
		PageSize:    req.PageSize,
	}

	pageSize := req.PageSize
	if pageSize < 0 {
		pageSize = 0
	} else if pageSize == 0 {
		pageSize = 50
	}

	items, err := s.repo.List(ctx, s.db, orgIDValue, filter, pagination.Pagination{
		PageToken: req.PageToken,
		PageSize:  int(pageSize),
	})
	if err != nil {
		return domain.ListResponse{}, err
	}

	var pageInfo *pagination.PageInfo
	if pageSize > 0 {
		pageInfo = pagination.BuildCursorPageInfo(items, pageSize, func(item *domain.Feature) string {
			token, err := pagination.EncodeCursor(pagination.Cursor{
				ID:        item.ID.String(),
				CreatedAt: item.CreatedAt.Format(time.RFC3339),
			})
			if err != nil {
				return ""
			}
			return token
		})
		if pageInfo != nil && pageInfo.HasMore && len(items) > int(pageSize) {
			items = items[:pageSize]
		}
	}

	resp := make([]domain.Response, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, s.toResponse(item))
	}

	out := domain.ListResponse{Features: resp}
	if pageInfo != nil {
		out.PageInfo = *pageInfo
	}

	return out, nil
}

func (s *Service) Create(ctx context.Context, req domain.CreateRequest) (*domain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, s.db, int64(orgID), idempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			resp := s.toResponse(existing)
			return &resp, nil
		}
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, domain.ErrInvalidCode
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidName
	}

	featureType, err := normalizeFeatureType(req.FeatureType)
	if err != nil {
		return nil, err
	}

	description := strings.TrimSpace(ptrToString(req.Description))
	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	var meterID *snowflake.ID
	if req.MeterID != nil {
		parsed, err := parseID(*req.MeterID)
		if err != nil {
			return nil, domain.ErrInvalidMeterID
		}
		meterID = &parsed
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	now := time.Now().UTC()
	record := &domain.Feature{
		ID:          s.genID.Generate(),
		OrgID:       orgID,
		Code:        code,
		Name:        name,
		Description: descriptionPtr,
		Type:        featureType,
		MeterID:     meterID,
		Active:      active,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if idempotencyKey != "" {
		record.IdempotencyKey = &idempotencyKey
	}
	if req.Metadata != nil {
		record.Metadata = datatypes.JSONMap(req.Metadata)
	}

	if err := s.repo.Create(ctx, s.db, record); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindByIdempotencyKey(ctx, s.db, int64(orgID), idempotencyKey)
			if findErr != nil {
				return nil, findErr
			}
			if existing != nil {
				resp := s.toResponse(existing)
				return &resp, nil
			}
		}
		return nil, err
	}

	resp := s.toResponse(record)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, req domain.UpdateRequest) (*domain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	orgIDValue := int64(orgID)

	featureID, err := snowflake.ParseString(strings.TrimSpace(req.ID))
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	item, err := s.repo.FindByID(ctx, s.db, orgIDValue, featureID.Int64())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidName
		}
		item.Name = name
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if description == "" {
			item.Description = nil
		} else {
			item.Description = &description
		}
	}
	if req.FeatureType != nil {
		featureType, err := normalizeFeatureType(*req.FeatureType)
		if err != nil {
			return nil, err
		}
		item.Type = featureType
	}
	if req.MeterID != nil {
		meterValue := strings.TrimSpace(*req.MeterID)
		if meterValue == "" {
			item.MeterID = nil
		} else {
			parsed, err := parseID(meterValue)
			if err != nil {
				return nil, domain.ErrInvalidMeterID
			}
			item.MeterID = &parsed
		}
	}
	if req.Active != nil {
		item.Active = *req.Active
	}
	if req.Metadata != nil {
		item.Metadata = datatypes.JSONMap(req.Metadata)
	}

	item.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, s.db, item); err != nil {
		return nil, err
	}

	resp := s.toResponse(item)
	return &resp, nil
}

func (s *Service) Archive(ctx context.Context, id string) (*domain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	orgIDValue := int64(orgID)

	featureID, err := snowflake.ParseString(strings.TrimSpace(id))
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	item, err := s.repo.FindByID(ctx, s.db, orgIDValue, featureID.Int64())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	item.Active = false
	item.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, s.db, item); err != nil {
		return nil, err
	}

	resp := s.toResponse(item)
	return &resp, nil
}

func (s *Service) toResponse(f *domain.Feature) domain.Response {
	resp := domain.Response{
		ID:             f.ID.String(),
		OrganizationID: f.OrgID.String(),
		Code:           f.Code,
		Name:           f.Name,
		Description:    f.Description,
		FeatureType:    f.Type,
		Active:         f.Active,
		CreatedAt:      f.CreatedAt,
		UpdatedAt:      f.UpdatedAt,
	}
	if f.MeterID != nil && *f.MeterID != 0 {
		meter := f.MeterID.String()
		resp.MeterID = &meter
	}
	if len(f.Metadata) > 0 {
		resp.Metadata = map[string]any(f.Metadata)
	}
	return resp
}

func normalizeFeatureType(value domain.FeatureType) (domain.FeatureType, error) {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case string(domain.FeatureTypeBoolean):
		return domain.FeatureTypeBoolean, nil
	case string(domain.FeatureTypeMetered):
		return domain.FeatureTypeMetered, nil
	default:
		return "", domain.ErrInvalidType
	}
}

func parseID(value string) (snowflake.ID, error) {
	parsed, err := snowflake.ParseString(strings.TrimSpace(value))
	if err != nil || parsed == 0 {
		return 0, errors.New("invalid_id")
	}
	return parsed, nil
}

func ptrToString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
