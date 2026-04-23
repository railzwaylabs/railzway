package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	planfeaturedomain "github.com/railzwaylabs/railzway/internal/planfeature/domain"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Params struct {
	fx.In

	DB                 *gorm.DB
	Log                *zap.Logger
	Repo               planfeaturedomain.Repository
	PlanRepo           plandomain.Repository
	FeatureRepo        featuredomain.Repository
	ProductFeatureRepo productfeaturedomain.Repository
	MeterSvc           usagedomain.Service
}

type Service struct {
	db                 *gorm.DB
	log                *zap.Logger
	repo               planfeaturedomain.Repository
	planRepo           plandomain.Repository
	featureRepo        featuredomain.Repository
	productFeatureRepo productfeaturedomain.Repository
	meterSvc           usagedomain.Service
}

func New(p Params) planfeaturedomain.Service {
	logger := p.Log
	if logger == nil {
		logger = zap.L()
	}
	return &Service{
		db:                 p.DB,
		log:                logger.Named("planfeature.service"),
		repo:               p.Repo,
		planRepo:           p.PlanRepo,
		featureRepo:        p.FeatureRepo,
		productFeatureRepo: p.ProductFeatureRepo,
		meterSvc:           p.MeterSvc,
	}
}

func (s *Service) List(ctx context.Context, req planfeaturedomain.ListRequest) ([]planfeaturedomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, planfeaturedomain.ErrInvalidOrganization
	}
	planID, err := uuid.Parse(strings.TrimSpace(req.PlanID))
	if err != nil || planID == uuid.Nil {
		return nil, planfeaturedomain.ErrInvalidPlanID
	}
	plan, err := s.planRepo.FindPlanByID(ctx, orgID, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, planfeaturedomain.ErrPlanNotFound
	}
	items, err := s.repo.ListByPlan(ctx, orgID, planID)
	if err != nil {
		return nil, err
	}
	resp := make([]planfeaturedomain.Response, 0, len(items))
	for _, item := range items {
		resp = append(resp, toResponse(item))
	}
	return resp, nil
}

func (s *Service) Replace(ctx context.Context, req planfeaturedomain.ReplaceRequest) ([]planfeaturedomain.Response, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, planfeaturedomain.ErrInvalidOrganization
	}
	planID, err := uuid.Parse(strings.TrimSpace(req.PlanID))
	if err != nil || planID == uuid.Nil {
		return nil, planfeaturedomain.ErrInvalidPlanID
	}
	plan, err := s.planRepo.FindPlanByID(ctx, orgID, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, planfeaturedomain.ErrPlanNotFound
	}

	parsed, featureIDs, err := parseReplaceInputs(req.Features)
	if err != nil {
		return nil, err
	}
	if err := s.validateFeatures(ctx, orgID, plan, featureIDs); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	items := make([]planfeaturedomain.PlanFeature, 0, len(parsed))
	for _, item := range parsed {
		resetPeriod := normalizeResetPeriod(item.ResetPeriod)
		if resetPeriod == "" {
			return nil, planfeaturedomain.ErrInvalidResetPeriod
		}
		items = append(items, planfeaturedomain.PlanFeature{
			ID:           uuid.New(),
			OrgID:        orgID,
			PlanID:       planID,
			FeatureID:    item.FeatureID,
			Enabled:      item.Enabled,
			LimitNumeric: item.LimitNumeric,
			LimitUnit:    item.LimitUnit,
			ResetPeriod:  resetPeriod,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.WithTx(tx).Replace(ctx, orgID, planID, items, now)
	}); err != nil {
		return nil, err
	}

	assigned, err := s.repo.ListByPlan(ctx, orgID, planID)
	if err != nil {
		return nil, err
	}
	resp := make([]planfeaturedomain.Response, 0, len(assigned))
	for _, item := range assigned {
		resp = append(resp, toResponse(item))
	}
	return resp, nil
}

func (s *Service) ListForPlans(ctx context.Context, req planfeaturedomain.ListForPlansRequest) ([]planfeaturedomain.Snapshot, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, planfeaturedomain.ErrInvalidOrganization
	}
	planIDs, err := parsePlanIDs(req.PlanIDs)
	if err != nil {
		return nil, err
	}
	if len(planIDs) == 0 {
		return nil, nil
	}
	items, err := s.repo.ListByPlans(ctx, orgID, planIDs)
	if err != nil {
		return nil, err
	}
	resp := make([]planfeaturedomain.Snapshot, 0, len(items))
	for _, item := range items {
		resp = append(resp, toSnapshot(item))
	}
	return resp, nil
}

type parsedReplaceInput struct {
	FeatureID    uuid.UUID
	Enabled      bool
	LimitNumeric *float64
	LimitUnit    *string
	ResetPeriod  string
}

func parseReplaceInputs(values []planfeaturedomain.ReplaceFeatureInput) ([]parsedReplaceInput, []uuid.UUID, error) {
	items := make([]parsedReplaceInput, 0, len(values))
	ids := make([]uuid.UUID, 0, len(values))
	indexByID := make(map[uuid.UUID]int)
	for _, value := range values {
		featureID, err := uuid.Parse(strings.TrimSpace(value.FeatureID))
		if err != nil || featureID == uuid.Nil {
			return nil, nil, planfeaturedomain.ErrInvalidFeatureID
		}
		var limitNumeric *float64
		if value.LimitNumeric != nil {
			if *value.LimitNumeric < 0 {
				return nil, nil, planfeaturedomain.ErrInvalidLimit
			}
			normalized := *value.LimitNumeric
			limitNumeric = &normalized
		}
		limitUnit := normalizeOptionalString(value.LimitUnit)
		item := parsedReplaceInput{
			FeatureID:    featureID,
			Enabled:      value.Enabled,
			LimitNumeric: limitNumeric,
			LimitUnit:    limitUnit,
			ResetPeriod:  derefString(value.ResetPeriod),
		}
		if idx, ok := indexByID[featureID]; ok {
			items[idx] = item
			continue
		}
		indexByID[featureID] = len(items)
		items = append(items, item)
		ids = append(ids, featureID)
	}
	return items, ids, nil
}

func parsePlanIDs(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parsed, err := uuid.Parse(trimmed)
		if err != nil || parsed == uuid.Nil {
			return nil, planfeaturedomain.ErrInvalidPlanID
		}
		if _, ok := seen[parsed]; ok {
			continue
		}
		seen[parsed] = struct{}{}
		ids = append(ids, parsed)
	}
	return ids, nil
}

func (s *Service) validateFeatures(ctx context.Context, orgID uuid.UUID, plan *plandomain.Plan, featureIDs []uuid.UUID) error {
	if len(featureIDs) == 0 {
		return nil
	}
	features, err := s.featureRepo.ListByIDs(ctx, orgID, featureIDs)
	if err != nil {
		return err
	}
	if len(features) != len(featureIDs) {
		return planfeaturedomain.ErrFeatureNotFound
	}
	allowed := make(map[uuid.UUID]struct{}, len(featureIDs))
	if plan.ProductID == nil || *plan.ProductID == uuid.Nil {
		return planfeaturedomain.ErrFeatureNotAllowed
	}
	productFeatures, err := s.productFeatureRepo.ListByProduct(ctx, orgID, *plan.ProductID)
	if err != nil {
		return err
	}
	for _, item := range productFeatures {
		allowed[item.FeatureID] = struct{}{}
	}
	for _, item := range features {
		if item == nil {
			continue
		}
		if !item.Active {
			return planfeaturedomain.ErrFeatureInactive
		}
		if _, ok := allowed[item.ID]; !ok {
			return planfeaturedomain.ErrFeatureNotAllowed
		}
		if item.Type != featuredomain.FeatureTypeMetered {
			continue
		}
		if item.MeterID == nil {
			return planfeaturedomain.ErrInvalidMeterID
		}
		meter, err := s.meterSvc.GetMeterByID(ctx, usagedomain.GetMeterRequest{ID: item.MeterID.String()})
		if err != nil {
			if errors.Is(err, usagedomain.ErrNotFound) {
				return planfeaturedomain.ErrMeterNotFound
			}
			if errors.Is(err, usagedomain.ErrInvalidID) {
				return planfeaturedomain.ErrInvalidMeterID
			}
			return err
		}
		if meter.ID == "" {
			return planfeaturedomain.ErrMeterNotFound
		}
	}
	return nil
}

func normalizeResetPeriod(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return planfeaturedomain.ResetPeriodNone
	}
	switch normalized {
	case planfeaturedomain.ResetPeriodNone,
		planfeaturedomain.ResetPeriodDay,
		planfeaturedomain.ResetPeriodMonth,
		planfeaturedomain.ResetPeriodBillingPeriod:
		return normalized
	default:
		return ""
	}
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func toResponse(item planfeaturedomain.FeatureAssignment) planfeaturedomain.Response {
	var meterID *string
	if item.MeterID != nil {
		value := item.MeterID.String()
		meterID = &value
	}
	return planfeaturedomain.Response{
		ID:           item.FeatureID.String(),
		Code:         item.Code,
		Name:         item.Name,
		FeatureType:  string(item.FeatureType),
		MeterID:      meterID,
		Active:       item.Active,
		Enabled:      item.Enabled,
		LimitNumeric: item.LimitNumeric,
		LimitUnit:    item.LimitUnit,
		ResetPeriod:  item.ResetPeriod,
	}
}

func toSnapshot(item planfeaturedomain.FeatureAssignment) planfeaturedomain.Snapshot {
	var meterID *string
	if item.MeterID != nil {
		value := item.MeterID.String()
		meterID = &value
	}
	return planfeaturedomain.Snapshot{
		PlanID:       item.PlanID.String(),
		FeatureID:    item.FeatureID.String(),
		Code:         item.Code,
		Name:         item.Name,
		FeatureType:  string(item.FeatureType),
		MeterID:      meterID,
		Active:       item.Active,
		Enabled:      item.Enabled,
		LimitNumeric: item.LimitNumeric,
		LimitUnit:    item.LimitUnit,
		ResetPeriod:  item.ResetPeriod,
	}
}
