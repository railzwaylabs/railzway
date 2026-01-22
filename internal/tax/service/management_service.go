package service

import (
	"context"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/orgcontext"
	taxdomain "github.com/smallbiznis/railzway/internal/tax/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type serviceParams struct {
	fx.In

	Log   *zap.Logger
	GenID *snowflake.Node
	Repo  taxdomain.Repository
}

type Service struct {
	log   *zap.Logger
	genID *snowflake.Node
	repo  taxdomain.Repository
}

func NewService(p serviceParams) taxdomain.Service {
	return &Service{
		log:   p.Log.Named("tax.service"),
		genID: p.GenID,
		repo:  p.Repo,
	}
}

func (s *Service) List(ctx context.Context, req taxdomain.ListRequest) ([]taxdomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, taxdomain.ErrInvalidOrganization
	}

	filter := taxdomain.ListRequest{
		Name:      strings.TrimSpace(req.Name),
		Code:      strings.TrimSpace(req.Code),
		IsEnabled: req.IsEnabled,
		SortBy:    strings.TrimSpace(req.SortBy),
		OrderBy:   strings.TrimSpace(req.OrderBy),
	}

	items, err := s.repo.List(ctx, orgID, filter)
	if err != nil {
		return nil, err
	}

	resp := make([]taxdomain.Response, 0, len(items))
	for _, item := range items {
		resp = append(resp, toResponse(&item))
	}

	return resp, nil
}

func (s *Service) Create(ctx context.Context, req taxdomain.CreateRequest) (*taxdomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, taxdomain.ErrInvalidOrganization
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, taxdomain.ErrInvalidTaxCode
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, taxdomain.ErrInvalidName
	}

	description := strings.TrimSpace(ptrToString(req.Description))
	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	now := time.Now().UTC()
	record := &taxdomain.TaxDefinition{
		ID:          s.genID.Generate(),
		OrgID:       orgID,
		Name:        name,
		Code:        code,
		TaxMode:     normalizeTaxMode(req.TaxMode),
		Rate:        req.Rate,
		Description: descriptionPtr,
		IsEnabled:   isEnabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := record.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, err
	}

	resp := toResponse(record)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, req taxdomain.UpdateRequest) (*taxdomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, taxdomain.ErrInvalidOrganization
	}

	defID, err := snowflake.ParseString(strings.TrimSpace(req.ID))
	if err != nil {
		return nil, taxdomain.ErrInvalidID
	}

	item, err := s.repo.FindByID(ctx, orgID, defID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, taxdomain.ErrNotFound
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, taxdomain.ErrInvalidName
		}
		item.Name = name
	}
	if req.TaxMode != nil {
		item.TaxMode = normalizeTaxMode(*req.TaxMode)
	}
	if req.Rate != nil {
		item.Rate = req.Rate
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if description == "" {
			item.Description = nil
		} else {
			item.Description = &description
		}
	}

	item.UpdatedAt = time.Now().UTC()
	if err := item.Validate(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	resp := toResponse(item)
	return &resp, nil
}

func (s *Service) Disable(ctx context.Context, id string) (*taxdomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, taxdomain.ErrInvalidOrganization
	}

	defID, err := snowflake.ParseString(strings.TrimSpace(id))
	if err != nil {
		return nil, taxdomain.ErrInvalidID
	}

	item, err := s.repo.FindByID(ctx, orgID, defID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, taxdomain.ErrNotFound
	}

	item.IsEnabled = false
	item.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}

	resp := toResponse(item)
	return &resp, nil
}

func toResponse(def *taxdomain.TaxDefinition) taxdomain.Response {
	return taxdomain.Response{
		ID:             def.ID.String(),
		OrganizationID: def.OrgID.String(),
		Code:           def.Code,
		Name:           def.Name,
		TaxMode:        def.TaxMode,
		Rate:           def.Rate,
		Description:    def.Description,
		IsEnabled:      def.IsEnabled,
		CreatedAt:      def.CreatedAt,
		UpdatedAt:      def.UpdatedAt,
	}
}

func normalizeTaxMode(value taxdomain.TaxMode) taxdomain.TaxMode {
	return taxdomain.TaxMode(strings.ToLower(strings.TrimSpace(string(value))))
}

func ptrToString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
