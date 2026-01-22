package service

import (
	"context"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/orgcontext"
	"github.com/smallbiznis/railzway/internal/product/domain"
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
		log:   p.Log.Named("product.service"),
		repo:  p.Repo,
		genID: p.GenID,
	}
}

func (s *Service) List(ctx context.Context, req domain.ListRequest) ([]domain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	orgIDValue := int64(orgID)

	filter := domain.ListRequest{
		Name:    strings.TrimSpace(req.Name),
		Active:  req.Active,
		SortBy:  strings.TrimSpace(req.SortBy),
		OrderBy: strings.TrimSpace(req.OrderBy),
	}

	items, err := s.repo.List(ctx, s.db, orgIDValue, filter)
	if err != nil {
		return nil, err
	}

	resp := make([]domain.Response, 0, len(items))
	for _, item := range items {
		resp = append(resp, s.toResponse(&item))
	}

	return resp, nil
}

func (s *Service) Create(ctx context.Context, req domain.CreateRequest) (*domain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	orgIDValue := int64(orgID)

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, domain.ErrInvalidCode
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidName
	}

	description := strings.TrimSpace(ptrToString(req.Description))
	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	now := time.Now().UTC()
	p := &domain.Product{
		ID:          s.genID.Generate().Int64(),
		OrgID:       orgIDValue,
		Code:        code,
		Name:        name,
		Description: descriptionPtr,
		Active:      active,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.Metadata != nil {
		p.Metadata = datatypes.JSONMap(req.Metadata)
	}
	if err := s.repo.Create(ctx, s.db, p); err != nil {
		return nil, err
	}
	resp := s.toResponse(p)
	return &resp, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	orgIDValue := int64(orgID)

	productID, err := snowflake.ParseString(strings.TrimSpace(id))
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	item, err := s.repo.FindByID(ctx, s.db, orgIDValue, productID.Int64())
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	resp := s.toResponse(item)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, req domain.UpdateRequest) (*domain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	orgIDValue := int64(orgID)

	productID, err := snowflake.ParseString(strings.TrimSpace(req.ID))
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	item, err := s.repo.FindByID(ctx, s.db, orgIDValue, productID.Int64())
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

	productID, err := snowflake.ParseString(strings.TrimSpace(id))
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	item, err := s.repo.FindByID(ctx, s.db, orgIDValue, productID.Int64())
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

func (s *Service) toResponse(p *domain.Product) domain.Response {
	resp := domain.Response{
		ID:             snowflake.ID(p.ID).String(),
		OrganizationID: snowflake.ID(p.OrgID).String(),
		Code:           p.Code,
		Name:           p.Name,
		Description:    p.Description,
		Active:         p.Active,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}

	if len(p.Metadata) > 0 {
		resp.Metadata = map[string]any(p.Metadata)
	}

	return resp
}

func ptrToString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
