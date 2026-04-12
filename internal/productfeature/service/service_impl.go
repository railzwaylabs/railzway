package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Params struct {
	fx.In

	DB          *gorm.DB
	Log         *zap.Logger
	Repo        productfeaturedomain.Repository
	ProductRepo productdomain.Repository
	FeatureRepo featuredomain.Repository
	MeterSvc    usagedomain.Service
}

type Service struct {
	db          *gorm.DB
	log         *zap.Logger
	repo        productfeaturedomain.Repository
	productRepo productdomain.Repository
	featureRepo featuredomain.Repository
	meterSvc    usagedomain.Service
}

func New(p Params) productfeaturedomain.Service {
	logger := p.Log
	if logger == nil {
		logger = zap.L()
	}
	return &Service{
		db:          p.DB,
		log:         logger.Named("productfeature.service"),
		repo:        p.Repo,
		productRepo: p.ProductRepo,
		featureRepo: p.FeatureRepo,
		meterSvc:    p.MeterSvc,
	}
}

func (s *Service) List(ctx context.Context, req productfeaturedomain.ListRequest) ([]productfeaturedomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, productfeaturedomain.ErrInvalidOrganization
	}

	productID, err := uuid.Parse(strings.TrimSpace(req.ProductID))
	if err != nil || productID == uuid.Nil {
		return nil, productfeaturedomain.ErrInvalidProductID
	}

	product, err := s.productRepo.FindByID(ctx, orgID, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, productfeaturedomain.ErrProductNotFound
	}

	items, err := s.repo.ListByProduct(ctx, orgID, productID)
	if err != nil {
		return nil, err
	}

	resp := make([]productfeaturedomain.Response, 0, len(items))
	for _, item := range items {
		resp = append(resp, toResponse(item))
	}
	return resp, nil
}

func (s *Service) Replace(ctx context.Context, req productfeaturedomain.ReplaceRequest) ([]productfeaturedomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, productfeaturedomain.ErrInvalidOrganization
	}

	productID, err := uuid.Parse(strings.TrimSpace(req.ProductID))
	if err != nil || productID == uuid.Nil {
		return nil, productfeaturedomain.ErrInvalidProductID
	}

	product, err := s.productRepo.FindByID(ctx, orgID, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, productfeaturedomain.ErrProductNotFound
	}

	featureIDs, err := parseFeatureIDs(req.FeatureIDs)
	if err != nil {
		return nil, err
	}

	if err := s.validateFeatures(ctx, orgID, featureIDs); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.WithTx(tx).Replace(ctx, productID, featureIDs, now)
	}); err != nil {
		return nil, err
	}

	items, err := s.repo.ListByProduct(ctx, orgID, productID)
	if err != nil {
		return nil, err
	}
	resp := make([]productfeaturedomain.Response, 0, len(items))
	for _, item := range items {
		resp = append(resp, toResponse(item))
	}
	return resp, nil
}

func (s *Service) ListForProducts(ctx context.Context, req productfeaturedomain.ListForProductsRequest) ([]productfeaturedomain.Snapshot, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, productfeaturedomain.ErrInvalidOrganization
	}

	productIDs, err := parseProductIDs(req.ProductIDs)
	if err != nil {
		return nil, err
	}
	if len(productIDs) == 0 {
		return nil, nil
	}

	items, err := s.repo.ListByProducts(ctx, orgID, productIDs)
	if err != nil {
		return nil, err
	}

	resp := make([]productfeaturedomain.Snapshot, 0, len(items))
	for _, item := range items {
		if !item.Active {
			continue
		}
		resp = append(resp, toSnapshot(item))
	}
	return resp, nil
}

func parseProductIDs(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parsed, err := uuid.Parse(trimmed)
		if err != nil || parsed == uuid.Nil {
			return nil, productfeaturedomain.ErrInvalidProductID
		}
		if _, ok := seen[parsed]; ok {
			continue
		}
		seen[parsed] = struct{}{}
		ids = append(ids, parsed)
	}
	return ids, nil
}

func parseFeatureIDs(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parsed, err := uuid.Parse(trimmed)
		if err != nil || parsed == uuid.Nil {
			return nil, productfeaturedomain.ErrInvalidFeatureID
		}
		if _, ok := seen[parsed]; ok {
			continue
		}
		seen[parsed] = struct{}{}
		ids = append(ids, parsed)
	}
	return ids, nil
}

func (s *Service) validateFeatures(ctx context.Context, orgID uuid.UUID, featureIDs []uuid.UUID) error {
	if len(featureIDs) == 0 {
		return nil
	}

	features, err := s.featureRepo.ListByIDs(ctx, orgID, featureIDs)
	if err != nil {
		return err
	}
	if len(features) != len(featureIDs) {
		return productfeaturedomain.ErrFeatureNotFound
	}

	for _, item := range features {
		if item == nil {
			continue
		}
		if !item.Active {
			return productfeaturedomain.ErrFeatureInactive
		}
		if item.Type != featuredomain.FeatureTypeMetered {
			continue
		}
		if item.MeterID == nil {
			return productfeaturedomain.ErrInvalidMeterID
		}
		meter, err := s.meterSvc.GetMeterByID(ctx, usagedomain.GetMeterRequest{ID: item.MeterID.String()})
		if err != nil {
			if errors.Is(err, usagedomain.ErrNotFound) {
				return productfeaturedomain.ErrMeterNotFound
			}
			if errors.Is(err, usagedomain.ErrInvalidID) {
				return productfeaturedomain.ErrInvalidMeterID
			}
			return err
		}
		if meter.ID == "" {
			return productfeaturedomain.ErrMeterNotFound
		}
	}

	return nil
}

func toResponse(item productfeaturedomain.FeatureAssignment) productfeaturedomain.Response {
	var meterID *string
	if item.MeterID != nil {
		value := item.MeterID.String()
		meterID = &value
	}

	return productfeaturedomain.Response{
		ID:          item.FeatureID.String(),
		Code:        item.Code,
		Name:        item.Name,
		FeatureType: string(item.FeatureType),
		MeterID:     meterID,
		Active:      item.Active,
	}
}

func toSnapshot(item productfeaturedomain.FeatureAssignment) productfeaturedomain.Snapshot {
	var meterID *string
	if item.MeterID != nil {
		value := item.MeterID.String()
		meterID = &value
	}

	return productfeaturedomain.Snapshot{
		FeatureID:   item.FeatureID.String(),
		Code:        item.Code,
		Name:        item.Name,
		FeatureType: string(item.FeatureType),
		MeterID:     meterID,
		Active:      item.Active,
	}
}
