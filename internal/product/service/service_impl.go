package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	"github.com/railzwaylabs/railzway/internal/product/domain"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type service struct {
	db              *gorm.DB
	repo            domain.Repository
	audit           *auditlog.Service
	plans           plandomain.Service
	productFeatures productfeaturedomain.Service
}

type Params struct {
	fx.In

	DB              *gorm.DB
	Repo            domain.Repository
	Audit           *auditlog.Service            `optional:"true"`
	Plans           plandomain.Service           `optional:"true"`
	ProductFeatures productfeaturedomain.Service `optional:"true"`
}

func NewService(p Params) domain.Service {
	return &service{
		db:              p.DB,
		repo:            p.Repo,
		audit:           p.Audit,
		plans:           p.Plans,
		productFeatures: p.ProductFeatures,
	}
}

func (s *service) Create(ctx context.Context, req domain.CreateProductRequest) (domain.ProductResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ProductResponse{}, domain.ErrInvalidOrganization
	}

	code := normalizeCode(req.Code)
	if code == "" {
		return domain.ProductResponse{}, domain.ErrInvalidCode
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.ProductResponse{}, domain.ErrInvalidName
	}

	// 1. Business Validation for Nested Prices
	if err := s.validateAggregateCreate(req); err != nil {
		return domain.ProductResponse{}, err
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err == nil && existing != nil {
			// Basic conflict check: check if name/code match
			if existing.Code != code {
				return domain.ProductResponse{}, fmt.Errorf("idempotency conflict: key %s already used for product %s", idempotencyKey, existing.Code)
			}
			return s.populateNested(ctx, toResponse(*existing), true, true)
		}
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	now := time.Now().UTC()
	product := domain.Product{
		ID:          uuid.New(),
		OrgID:       orgID,
		Code:        code,
		Name:        name,
		Description: trimOptional(req.Description),
		Active:      active,
		Metadata:    json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if idempotencyKey != "" {
		product.IdempotencyKey = &idempotencyKey
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)

		if err := txRepo.Create(ctx, product); err != nil {
			return err
		}

		if s.productFeatures != nil && len(req.FeatureIDs) > 0 {
			_, err := s.productFeatures.Replace(ctx, productfeaturedomain.ReplaceRequest{
				ProductID:  product.ID.String(),
				FeatureIDs: req.FeatureIDs,
			})
			if err != nil {
				return err
			}
		}

		if s.plans != nil && len(req.Plans) > 0 {
			for _, pInput := range req.Plans {
				pReq := s.mapToCreatePlanRequest(product.ID.String(), idempotencyKey, pInput)
				_, err := s.plans.CreatePlan(ctx, pReq)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return domain.ProductResponse{}, err
	}

	resp, _ := s.populateNested(ctx, toResponse(product), len(req.FeatureIDs) > 0, len(req.Plans) > 0)
	s.recordAudit(ctx, "product.create", "product", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) validateAggregateCreate(req domain.CreateProductRequest) error {
	for _, p := range req.Plans {
		for _, pr := range p.Prices {
			switch pr.PriceType {
			case "flat":
				if pr.MeterID != nil && *pr.MeterID != "" {
					return errors.New("meter_id must not be provided for flat pricing")
				}
				if len(pr.Amounts) == 0 {
					return errors.New("at least one amount is required for flat pricing")
				}
				if len(pr.Tiers) > 0 {
					return errors.New("tiers are forbidden for flat pricing")
				}
			case "usage":
				if pr.MeterID == nil || *pr.MeterID == "" {
					return errors.New("meter_id is required for usage-based pricing")
				}
				if len(pr.Amounts) == 0 {
					return errors.New("at least one amount is required for usage-based pricing")
				}
				if len(pr.Tiers) > 0 {
					return errors.New("tiers are forbidden for usage-based pricing")
				}
			case "tiered":
				if pr.MeterID == nil || *pr.MeterID == "" {
					return errors.New("meter_id is required for tiered pricing")
				}
				if len(pr.Tiers) == 0 {
					return errors.New("at least one tier is required for tiered pricing")
				}
				if len(pr.Amounts) > 0 {
					return errors.New("top-level amounts are forbidden for tiered pricing")
				}
				// Basic tier validation
				for i, t := range pr.Tiers {
					if i > 0 {
						prev := pr.Tiers[i-1]
						if t.StartQuantity < prev.StartQuantity {
							return errors.New("tiers must be ordered by start_quantity")
						}
					}
				}
			default:
				return fmt.Errorf("invalid price_type: %s", pr.PriceType)
			}
		}
	}
	return nil
}

func (s *service) Update(ctx context.Context, id string, req domain.UpdateProductRequest) (domain.ProductResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ProductResponse{}, domain.ErrInvalidOrganization
	}
	productID, err := parseID(id)
	if err != nil {
		return domain.ProductResponse{}, err
	}

	var product *domain.Product
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)

		updates := map[string]interface{}{}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return domain.ErrInvalidName
			}
			updates["name"] = name
		}
		if req.Description != nil {
			updates["description"] = trimOptional(req.Description)
		}
		if req.Active != nil {
			updates["active"] = *req.Active
		}

		if len(updates) > 0 {
			updates["updated_at"] = time.Now().UTC()
			if err := txRepo.Update(ctx, orgID, productID, updates); err != nil {
				return err
			}
		}

		if s.productFeatures != nil && req.FeatureIDs != nil {
			_, err := s.productFeatures.Replace(ctx, productfeaturedomain.ReplaceRequest{
				ProductID:  productID.String(),
				FeatureIDs: *req.FeatureIDs,
			})
			if err != nil {
				return err
			}
		}

		item, err := txRepo.FindByID(ctx, orgID, productID)
		if err != nil {
			return err
		}
		product = item
		return nil
	})

	if err != nil {
		return domain.ProductResponse{}, err
	}
	if product == nil {
		return domain.ProductResponse{}, domain.ErrNotFound
	}

	resp, _ := s.populateNested(ctx, toResponse(*product), req.FeatureIDs != nil, false)
	s.recordAudit(ctx, "product.update", "product", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) GetByID(ctx context.Context, req domain.GetProductRequest) (domain.ProductResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ProductResponse{}, domain.ErrInvalidOrganization
	}
	productID, err := parseID(req.ID)
	if err != nil {
		return domain.ProductResponse{}, err
	}
	item, err := s.repo.FindByID(ctx, orgID, productID)
	if err != nil {
		return domain.ProductResponse{}, err
	}
	if item == nil {
		return domain.ProductResponse{}, domain.ErrNotFound
	}

	return s.populateNested(ctx, toResponse(*item), req.ExpandFeatures, req.ExpandPlans)
}

func (s *service) List(ctx context.Context, req domain.ListProductRequest) (domain.ListProductResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListProductResponse{}, domain.ErrInvalidOrganization
	}
	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.ListFilter{
		Code:   strings.TrimSpace(req.Code),
		Name:   strings.TrimSpace(req.Name),
		Active: req.Active,
	}

	var cursor *domain.ListCursor
	if token := strings.TrimSpace(req.PageToken); token != "" {
		decoded, err := pagination.DecodeCursor(token)
		if err != nil {
			return domain.ListProductResponse{}, domain.ErrInvalidID
		}
		if decoded != nil && decoded.CreatedAt != "" && decoded.ID != "" {
			parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
			if err != nil {
				return domain.ListProductResponse{}, domain.ErrInvalidID
			}
			parsedID, err := uuid.Parse(decoded.ID)
			if err != nil {
				return domain.ListProductResponse{}, domain.ErrInvalidID
			}
			cursor = &domain.ListCursor{
				ID:        parsedID,
				CreatedAt: parsedTime,
			}
		}
	}

	items, err := s.repo.List(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListProductResponse{}, err
	}

	resp := domain.ListProductResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Product) string {
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

	resp.Products = make([]domain.ProductResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Products = append(resp.Products, toResponse(*item))
	}

	if req.ExpandPlans || req.ExpandFeatures {
		for i := range resp.Products {
			populated, err := s.populateNested(ctx, resp.Products[i], req.ExpandFeatures, req.ExpandPlans)
			if err == nil {
				resp.Products[i] = populated
			}
		}
	}

	return resp, nil
}

func (s *service) populateNested(ctx context.Context, resp domain.ProductResponse, features, plans bool) (domain.ProductResponse, error) {
	if features && s.productFeatures != nil {
		fData, err := s.productFeatures.List(ctx, productfeaturedomain.ListRequest{ProductID: resp.ID})
		if err == nil {
			resp.Features = make([]domain.ProductFeatureResponse, 0, len(fData))
			for _, f := range fData {
				resp.Features = append(resp.Features, domain.ProductFeatureResponse{
					ID:          f.ID,
					Code:        f.Code,
					Name:        f.Name,
					FeatureType: f.FeatureType,
					MeterID:     f.MeterID,
					Active:      f.Active,
				})
			}
		}
	}

	if plans && s.plans != nil {
		pData, err := s.plans.ListPlans(ctx, plandomain.ListPlanRequest{
			PageSize:  100,
			ProductID: &resp.ID,
		})
		if err == nil {
			resp.Plans = make([]domain.ProductPlanResponse, 0, len(pData.Plans))
			for _, p := range pData.Plans {
				resp.Plans = append(resp.Plans, s.mapPlanToProductPlanResponse(p))
			}
		}
	}

	return resp, nil
}

func (s *service) mapPlanToProductPlanResponse(p plandomain.PlanResponse) domain.ProductPlanResponse {
	prices := make([]domain.ProductPlanPrice, 0, len(p.Prices))
	for _, pr := range p.Prices {
		amounts := make([]domain.ProductPlanAmount, 0, len(pr.Amounts))
		for _, am := range pr.Amounts {
			amounts = append(amounts, domain.ProductPlanAmount{
				ID:                 am.ID,
				Currency:           am.Currency,
				UnitAmountCents:    am.UnitAmountCents,
				MinimumAmountCents: am.MinimumAmountCents,
				MaximumAmountCents: am.MaximumAmountCents,
				EffectiveFrom:      am.EffectiveFrom,
				EffectiveTo:        am.EffectiveTo,
			})
		}
		tiers := make([]domain.ProductPlanTier, 0, len(pr.Tiers))
		for _, t := range pr.Tiers {
			tiers = append(tiers, domain.ProductPlanTier{
				ID:              t.ID,
				TierMode:        t.TierMode,
				StartQuantity:   t.StartQuantity,
				EndQuantity:     t.EndQuantity,
				UnitAmountCents: t.UnitAmountCents,
				FlatAmountCents: t.FlatAmountCents,
				Unit:            t.Unit,
			})
		}
		prices = append(prices, domain.ProductPlanPrice{
			ID:                   pr.ID,
			Code:                 pr.Code,
			Name:                 pr.Name,
			Description:          pr.Description,
			PriceType:            pr.PriceType,
			BillingInterval:      pr.BillingInterval,
			BillingIntervalCount: pr.BillingIntervalCount,
			AggregateUsage:       pr.AggregateUsage,
			BillingUnit:          pr.BillingUnit,
			MeterID:              pr.MeterID,
			MeterCode:            pr.MeterCode,
			Active:               pr.Active,
			Amounts:              amounts,
			Tiers:                tiers,
		})
	}

	return domain.ProductPlanResponse{
		ID:          p.ID,
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Active:      p.Active,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Prices:      prices,
	}
}

func (s *service) mapToCreatePlanRequest(productID, idempotencyKey string, input domain.CreateProductPlanInput) plandomain.CreatePlanRequest {
	prices := make([]plandomain.CreatePlanPriceInput, 0, len(input.Prices))
	for _, pr := range input.Prices {
		amounts := make([]plandomain.CreatePlanAmountInput, 0, len(pr.Amounts))
		for _, am := range pr.Amounts {
			amounts = append(amounts, plandomain.CreatePlanAmountInput{
				Currency:           am.Currency,
				UnitAmountCents:    am.UnitAmountCents,
				MinimumAmountCents: am.MinimumAmountCents,
				MaximumAmountCents: am.MaximumAmountCents,
				EffectiveFrom:      am.EffectiveFrom,
				EffectiveTo:        am.EffectiveTo,
			})
		}
		tiers := make([]plandomain.CreatePlanTierInput, 0, len(pr.Tiers))
		for _, t := range pr.Tiers {
			tiers = append(tiers, plandomain.CreatePlanTierInput{
				TierMode:        t.TierMode,
				StartQuantity:   t.StartQuantity,
				EndQuantity:     t.EndQuantity,
				UnitAmountCents: t.UnitAmountCents,
				FlatAmountCents: t.FlatAmountCents,
				Unit:            t.Unit,
			})
		}
		
		desc := ""
		if pr.Description != nil {
			desc = *pr.Description
		}
		mID := ""
		if pr.MeterID != nil {
			mID = *pr.MeterID
		}

		prices = append(prices, plandomain.CreatePlanPriceInput{
			Code:                 pr.Code,
			Name:                 pr.Name,
			Description:          desc,
			PriceType:            pr.PriceType,
			BillingInterval:      pr.BillingInterval,
			BillingIntervalCount: pr.BillingIntervalCount,
			MeterID:              &mID,
			Active:               pr.Active,
			Amounts:              amounts,
			Tiers:                tiers,
		})
	}

	pDesc := ""
	if input.Description != nil {
		pDesc = *input.Description
	}

	return plandomain.CreatePlanRequest{
		Code:           input.Code,
		Name:           input.Name,
		Description:    pDesc,
		Active:         input.Active,
		ProductID:      &productID,
		IdempotencyKey: idempotencyKey,
		Prices:         prices,
	}
}

func toResponse(item domain.Product) domain.ProductResponse {
	return domain.ProductResponse{
		ID:          item.ID.String(),
		OrgID:       item.OrgID.String(),
		Code:        item.Code,
		Name:        item.Name,
		Description: item.Description,
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
