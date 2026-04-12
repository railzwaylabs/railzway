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
	"github.com/railzwaylabs/railzway/internal/plan/domain"
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

func (s *service) CreatePlan(ctx context.Context, req domain.CreatePlanRequest) (domain.PlanResponse, error) {
	if len(req.Prices) > 0 {
		return s.createPlanWithPrices(ctx, req)
	}

	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanResponse{}, domain.ErrInvalidOrganization
	}

	code := normalizeCode(req.Code)
	if code == "" {
		return domain.PlanResponse{}, domain.ErrInvalidCode
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.PlanResponse{}, domain.ErrInvalidName
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindPlanByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.PlanResponse{}, err
		}
		if existing != nil {
			return toPlanResponse(*existing), nil
		}
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	productID, err := parseOptionalProductID(req.ProductID)
	if err != nil {
		return domain.PlanResponse{}, err
	}

	now := time.Now().UTC()
	plan := domain.Plan{
		ID:          uuid.New(),
		OrgID:       orgID,
		ProductID:   productID,
		Code:        code,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Active:      active,
		Metadata:    json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if idempotencyKey != "" {
		plan.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindPlanByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.PlanResponse{}, findErr
			}
			if existing != nil {
				return toPlanResponse(*existing), nil
			}
		}
		return domain.PlanResponse{}, err
	}

	resp := toPlanResponse(plan)
	s.recordAudit(ctx, "plan.create", "plan", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) createPlanWithPrices(ctx context.Context, req domain.CreatePlanRequest) (domain.PlanResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanResponse{}, domain.ErrInvalidOrganization
	}

	code := normalizeCode(req.Code)
	if code == "" {
		return domain.PlanResponse{}, domain.ErrInvalidCode
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.PlanResponse{}, domain.ErrInvalidName
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	type auditEvent struct {
		action       string
		resourceType string
		resourceID   string
		before       interface{}
		after        interface{}
	}

	events := make([]auditEvent, 0, 8)
	var planResp domain.PlanResponse
	pricesResp := make([]domain.PlanPriceResponse, 0, len(req.Prices))

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)

		if idempotencyKey != "" {
			existing, err := repo.FindPlanByIdempotencyKey(ctx, orgID, idempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				planResp = toPlanResponse(*existing)
				return nil
			}
		}

		productID, err := parseOptionalProductID(req.ProductID)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		plan := domain.Plan{
			ID:          uuid.New(),
			OrgID:       orgID,
			ProductID:   productID,
			Code:        code,
			Name:        name,
			Description: strings.TrimSpace(req.Description),
			Active:      active,
			Metadata:    json.RawMessage(`{}`),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if idempotencyKey != "" {
			plan.IdempotencyKey = &idempotencyKey
		}

		if err := repo.CreatePlan(ctx, plan); err != nil {
			if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
				existing, findErr := repo.FindPlanByIdempotencyKey(ctx, orgID, idempotencyKey)
				if findErr != nil {
					return findErr
				}
				if existing != nil {
					planResp = toPlanResponse(*existing)
					return nil
				}
			}
			return err
		}

		planResp = toPlanResponse(plan)
		events = append(events, auditEvent{
			action:       "plan.create",
			resourceType: "plan",
			resourceID:   planResp.ID,
			before:       nil,
			after:        planResp,
		})

		for _, priceReq := range req.Prices {
			priceCode := normalizeCode(priceReq.Code)
			if priceCode == "" {
				return domain.ErrInvalidCode
			}

			priceType := strings.ToLower(strings.TrimSpace(priceReq.PriceType))
			if !isValidPriceType(priceType) {
				return domain.ErrInvalidPriceType
			}

			billingInterval := strings.ToLower(strings.TrimSpace(priceReq.BillingInterval))
			if !isValidBillingInterval(billingInterval) {
				return domain.ErrInvalidInterval
			}

			if priceReq.BillingIntervalCount < 1 {
				return domain.ErrInvalidIntervalCount
			}

			var meterID *uuid.UUID
			if priceReq.MeterID != nil && strings.TrimSpace(*priceReq.MeterID) != "" {
				parsed, err := uuid.Parse(strings.TrimSpace(*priceReq.MeterID))
				if err != nil || parsed == uuid.Nil {
					return domain.ErrInvalidMeter
				}
				meterID = &parsed
			}
			if (priceType == domain.PriceTypeUsage || priceType == domain.PriceTypeTiered) && meterID == nil {
				return domain.ErrInvalidMeter
			}

			active := true
			if priceReq.Active != nil {
				active = *priceReq.Active
			}

			priceIdempotencyKey := strings.TrimSpace(priceReq.IdempotencyKey)
			var price domain.PlanPrice
			if priceIdempotencyKey != "" {
				existing, err := repo.FindPlanPriceByIdempotencyKey(ctx, orgID, priceIdempotencyKey)
				if err != nil {
					return err
				}
				if existing != nil {
					price = *existing
				}
			}

			createdPrice := false
			if price.ID == uuid.Nil {
				price = domain.PlanPrice{
					ID:                   uuid.New(),
					OrgID:                orgID,
					PlanID:               plan.ID,
					MeterID:              meterID,
					Code:                 priceCode,
					Name:                 strings.TrimSpace(priceReq.Name),
					Description:          strings.TrimSpace(priceReq.Description),
					PriceType:            priceType,
					BillingInterval:      billingInterval,
					BillingIntervalCount: priceReq.BillingIntervalCount,
					AggregateUsage:       strings.TrimSpace(priceReq.AggregateUsage),
					BillingUnit:          strings.TrimSpace(priceReq.BillingUnit),
					MeterCode:            strings.TrimSpace(priceReq.MeterCode),
					Active:               active,
					Metadata:             json.RawMessage(`{}`),
					CreatedAt:            now,
					UpdatedAt:            now,
				}
				if priceIdempotencyKey != "" {
					price.IdempotencyKey = &priceIdempotencyKey
				}

				if err := repo.CreatePlanPrice(ctx, price); err != nil {
					if priceIdempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
						existing, findErr := repo.FindPlanPriceByIdempotencyKey(ctx, orgID, priceIdempotencyKey)
						if findErr != nil {
							return findErr
						}
						if existing != nil {
							price = *existing
						} else {
							return err
						}
					} else {
						return err
					}
				}
				createdPrice = true
			}

			amountsResp := make([]domain.PlanAmountResponse, 0, len(priceReq.Amounts))
			for _, amountReq := range priceReq.Amounts {
				currency := normalizeCurrency(amountReq.Currency)
				if !isValidCurrency(currency) {
					return domain.ErrInvalidCurrency
				}
				if amountReq.UnitAmountCents < 0 {
					return domain.ErrInvalidAmount
				}
				if amountReq.MinimumAmountCents != nil && *amountReq.MinimumAmountCents < 0 {
					return domain.ErrInvalidAmount
				}
				if amountReq.MaximumAmountCents != nil && *amountReq.MaximumAmountCents < 0 {
					return domain.ErrInvalidAmount
				}
				if amountReq.MinimumAmountCents != nil && amountReq.MaximumAmountCents != nil &&
					*amountReq.MinimumAmountCents > *amountReq.MaximumAmountCents {
					return domain.ErrInvalidAmount
				}

				effectiveFrom := now
				if amountReq.EffectiveFrom != nil {
					effectiveFrom = amountReq.EffectiveFrom.UTC()
				}
				var effectiveTo *time.Time
				if amountReq.EffectiveTo != nil {
					t := amountReq.EffectiveTo.UTC()
					if t.Before(effectiveFrom) {
						return domain.ErrInvalidAmount
					}
					effectiveTo = &t
				}

				amountIdempotencyKey := strings.TrimSpace(amountReq.IdempotencyKey)
				var amount domain.PlanAmount
				if amountIdempotencyKey != "" {
					existing, err := repo.FindPlanAmountByIdempotencyKey(ctx, orgID, amountIdempotencyKey)
					if err != nil {
						return err
					}
					if existing != nil {
						amount = *existing
					}
				}

				createdAmount := false
				if amount.ID == uuid.Nil {
					amount = domain.PlanAmount{
						ID:                 uuid.New(),
						OrgID:              orgID,
						PlanPriceID:        price.ID,
						Currency:           currency,
						UnitAmountCents:    amountReq.UnitAmountCents,
						MinimumAmountCents: amountReq.MinimumAmountCents,
						MaximumAmountCents: amountReq.MaximumAmountCents,
						EffectiveFrom:      effectiveFrom,
						EffectiveTo:        effectiveTo,
						Metadata:           json.RawMessage(`{}`),
						CreatedAt:          now,
						UpdatedAt:          now,
					}
					if amountIdempotencyKey != "" {
						amount.IdempotencyKey = &amountIdempotencyKey
					}

					if err := repo.CreatePlanAmount(ctx, amount); err != nil {
						if amountIdempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
							existing, findErr := repo.FindPlanAmountByIdempotencyKey(ctx, orgID, amountIdempotencyKey)
							if findErr != nil {
								return findErr
							}
							if existing != nil {
								amount = *existing
							} else {
								return err
							}
						} else {
							return err
						}
					}
					createdAmount = true
				}
				amountResp := toPlanAmountResponse(amount)
				if createdAmount {
					events = append(events, auditEvent{
						action:       "plan_amount.create",
						resourceType: "plan_amount",
						resourceID:   amountResp.ID,
						before:       nil,
						after:        amountResp,
					})
				}
				amountsResp = append(amountsResp, amountResp)
			}

			tiersResp := make([]domain.PlanTierResponse, 0, len(priceReq.Tiers))
			for _, tierReq := range priceReq.Tiers {
				tierMode := strings.ToLower(strings.TrimSpace(tierReq.TierMode))
				if !isValidTierMode(tierMode) {
					return domain.ErrInvalidTierMode
				}
				if tierReq.StartQuantity < 0 {
					return domain.ErrInvalidQuantity
				}
				if tierReq.EndQuantity != nil && *tierReq.EndQuantity < tierReq.StartQuantity {
					return domain.ErrInvalidQuantity
				}
				if tierReq.UnitAmountCents == nil && tierReq.FlatAmountCents == nil {
					return domain.ErrInvalidAmount
				}
				if tierReq.UnitAmountCents != nil && *tierReq.UnitAmountCents < 0 {
					return domain.ErrInvalidAmount
				}
				if tierReq.FlatAmountCents != nil && *tierReq.FlatAmountCents < 0 {
					return domain.ErrInvalidAmount
				}

				unit := strings.TrimSpace(tierReq.Unit)
				if unit == "" {
					return domain.ErrInvalidQuantity
				}

				tierIdempotencyKey := strings.TrimSpace(tierReq.IdempotencyKey)
				var tier domain.PlanTier
				if tierIdempotencyKey != "" {
					existing, err := repo.FindPlanTierByIdempotencyKey(ctx, orgID, tierIdempotencyKey)
					if err != nil {
						return err
					}
					if existing != nil {
						tier = *existing
					}
				}

				createdTier := false
				if tier.ID == uuid.Nil {
					tier = domain.PlanTier{
						ID:              uuid.New(),
						OrgID:           orgID,
						PlanPriceID:     price.ID,
						TierMode:        tierMode,
						StartQuantity:   tierReq.StartQuantity,
						EndQuantity:     tierReq.EndQuantity,
						UnitAmountCents: tierReq.UnitAmountCents,
						FlatAmountCents: tierReq.FlatAmountCents,
						Unit:            unit,
						Metadata:        json.RawMessage(`{}`),
						CreatedAt:       now,
						UpdatedAt:       now,
					}
					if tierIdempotencyKey != "" {
						tier.IdempotencyKey = &tierIdempotencyKey
					}

					if err := repo.CreatePlanTier(ctx, tier); err != nil {
						if tierIdempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
							existing, findErr := repo.FindPlanTierByIdempotencyKey(ctx, orgID, tierIdempotencyKey)
							if findErr != nil {
								return findErr
							}
							if existing != nil {
								tier = *existing
							} else {
								return err
							}
						} else {
							return err
						}
					}
					createdTier = true
				}
				tierResp := toPlanTierResponse(tier)
				if createdTier {
					events = append(events, auditEvent{
						action:       "plan_tier.create",
						resourceType: "plan_tier",
						resourceID:   tierResp.ID,
						before:       nil,
						after:        tierResp,
					})
				}
				tiersResp = append(tiersResp, tierResp)
			}

			priceResp := toPlanPriceResponse(price)
			if createdPrice {
				events = append(events, auditEvent{
					action:       "plan_price.create",
					resourceType: "plan_price",
					resourceID:   priceResp.ID,
					before:       nil,
					after:        priceResp,
				})
			}
			priceResp.Amounts = amountsResp
			priceResp.Tiers = tiersResp
			pricesResp = append(pricesResp, priceResp)
		}

		return nil
	})
	if err != nil {
		return domain.PlanResponse{}, err
	}

	for _, event := range events {
		s.recordAudit(ctx, event.action, event.resourceType, event.resourceID, event.before, event.after, nil)
	}

	if planResp.ID == "" {
		return domain.PlanResponse{}, domain.ErrNotFound
	}

	if len(pricesResp) > 0 {
		planResp.Prices = pricesResp
	}

	return planResp, nil
}

func (s *service) UpdatePlan(ctx context.Context, id string, req domain.UpdatePlanRequest) (domain.PlanResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanResponse{}, domain.ErrInvalidOrganization
	}

	planID, err := parseID(id)
	if err != nil {
		return domain.PlanResponse{}, err
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return domain.PlanResponse{}, domain.ErrInvalidName
		}
		updates["name"] = name
	}

	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}

	if req.Active != nil {
		updates["active"] = *req.Active
	}

	if req.ProductID != nil {
		if strings.TrimSpace(*req.ProductID) == "" {
			updates["product_id"] = nil
		} else {
			parsed, err := uuid.Parse(strings.TrimSpace(*req.ProductID))
			if err != nil || parsed == uuid.Nil {
				return domain.PlanResponse{}, domain.ErrInvalidProduct
			}
			updates["product_id"] = &parsed
		}
	}

	beforePlan, err := s.repo.FindPlanByID(ctx, orgID, planID)
	if err != nil {
		return domain.PlanResponse{}, err
	}
	if beforePlan == nil {
		return domain.PlanResponse{}, domain.ErrNotFound
	}

	if err := s.repo.UpdatePlan(ctx, orgID, planID, updates); err != nil {
		return domain.PlanResponse{}, err
	}

	plan, err := s.repo.FindPlanByID(ctx, orgID, planID)
	if err != nil {
		return domain.PlanResponse{}, err
	}
	if plan == nil {
		return domain.PlanResponse{}, domain.ErrNotFound
	}

	resp := toPlanResponse(*plan)
	s.recordAudit(ctx, "plan.update", "plan", resp.ID, toPlanResponse(*beforePlan), resp, nil)
	return resp, nil
}

func (s *service) GetPlanByID(ctx context.Context, req domain.GetPlanRequest) (domain.PlanResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanResponse{}, domain.ErrInvalidOrganization
	}

	id, err := parseID(req.ID)
	if err != nil {
		return domain.PlanResponse{}, err
	}

	plan, err := s.repo.FindPlanByID(ctx, orgID, id)
	if err != nil {
		return domain.PlanResponse{}, err
	}
	if plan == nil {
		return domain.PlanResponse{}, domain.ErrNotFound
	}

	resp := toPlanResponse(*plan)

	prices, err := s.listPlanPricesByPlan(ctx, orgID, plan.ID)
	if err != nil {
		return domain.PlanResponse{}, err
	}
	if len(prices) == 0 {
		return resp, nil
	}

	resp.Prices = make([]domain.PlanPriceResponse, 0, len(prices))
	for _, price := range prices {
		if price == nil {
			continue
		}

		priceResp := toPlanPriceResponse(*price)

		amounts, err := s.listPlanAmountsByPrice(ctx, orgID, price.ID)
		if err != nil {
			return domain.PlanResponse{}, err
		}

		if len(amounts) > 0 {
			priceResp.Amounts = make([]domain.PlanAmountResponse, 0, len(amounts))
			for _, amount := range amounts {
				if amount == nil {
					continue
				}
				priceResp.Amounts = append(priceResp.Amounts, toPlanAmountResponse(*amount))
			}
		}

		tiers, err := s.repo.ListPlanTiersByPrice(ctx, orgID, price.ID)
		if err != nil {
			return domain.PlanResponse{}, err
		}
		if len(tiers) > 0 {
			priceResp.Tiers = make([]domain.PlanTierResponse, 0, len(tiers))
			for _, tier := range tiers {
				if tier == nil {
					continue
				}
				priceResp.Tiers = append(priceResp.Tiers, toPlanTierResponse(*tier))
			}
		}

		resp.Prices = append(resp.Prices, priceResp)
	}

	return resp, nil
}

func (s *service) ListPlans(ctx context.Context, req domain.ListPlanRequest) (domain.ListPlanResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListPlanResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	var productID *uuid.UUID
	if req.ProductID != nil && *req.ProductID != "" {
		id, err := uuid.Parse(strings.TrimSpace(*req.ProductID))
		if err == nil && id != uuid.Nil {
			productID = &id
		}
	}

	filter := domain.PlanListFilter{
		ProductID: productID,
		Code:      normalizeCode(req.Code),
		Name:      strings.TrimSpace(req.Name),
		Active:    req.Active,
	}
	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListPlanResponse{}, err
	}

	items, err := s.repo.ListPlans(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListPlanResponse{}, err
	}

	resp := domain.ListPlanResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Plan) string {
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

	resp.Plans = make([]domain.PlanResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		planResp := toPlanResponse(*item)
		prices, err := s.listPlanPricesByPlan(ctx, orgID, item.ID)
		if err != nil {
			return domain.ListPlanResponse{}, err
		}

		if len(prices) > 0 {
			planResp.Prices = make([]domain.PlanPriceResponse, 0, len(prices))
			for _, price := range prices {
				if price == nil {
					continue
				}
				priceResp := toPlanPriceResponse(*price)

				amounts, err := s.listPlanAmountsByPrice(ctx, orgID, price.ID)
				if err != nil {
					return domain.ListPlanResponse{}, err
				}
				if len(amounts) > 0 {
					priceResp.Amounts = make([]domain.PlanAmountResponse, 0, len(amounts))
					for _, amount := range amounts {
						if amount == nil {
							continue
						}
						priceResp.Amounts = append(priceResp.Amounts, toPlanAmountResponse(*amount))
					}
				}

				tiers, err := s.repo.ListPlanTiersByPrice(ctx, orgID, price.ID)
				if err != nil {
					return domain.ListPlanResponse{}, err
				}
				if len(tiers) > 0 {
					priceResp.Tiers = make([]domain.PlanTierResponse, 0, len(tiers))
					for _, tier := range tiers {
						if tier == nil {
							continue
						}
						priceResp.Tiers = append(priceResp.Tiers, toPlanTierResponse(*tier))
					}
				}

				planResp.Prices = append(planResp.Prices, priceResp)
			}
		}

		resp.Plans = append(resp.Plans, planResp)
	}

	return resp, nil
}

func (s *service) listPlanPricesByPlan(ctx context.Context, orgID, planID uuid.UUID) ([]*domain.PlanPrice, error) {
	const chunkSize = 200
	filter := domain.PlanPriceListFilter{PlanID: planID}
	var cursor *domain.ListCursor
	var out []*domain.PlanPrice

	for {
		items, err := s.repo.ListPlanPrices(ctx, orgID, filter, chunkSize+1, cursor)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}

		hasMore := false
		if len(items) > chunkSize {
			hasMore = true
			items = items[:chunkSize]
		}

		out = append(out, items...)
		if !hasMore {
			break
		}

		last := items[len(items)-1]
		if last == nil {
			break
		}
		cursor = &domain.ListCursor{ID: last.ID, CreatedAt: last.CreatedAt}
	}

	return out, nil
}

func (s *service) listPlanAmountsByPrice(ctx context.Context, orgID, priceID uuid.UUID) ([]*domain.PlanAmount, error) {
	const chunkSize = 200
	filter := domain.PlanAmountListFilter{PlanPriceID: priceID}
	var cursor *domain.ListCursor
	var out []*domain.PlanAmount

	for {
		items, err := s.repo.ListPlanAmounts(ctx, orgID, filter, chunkSize+1, cursor)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}

		hasMore := false
		if len(items) > chunkSize {
			hasMore = true
			items = items[:chunkSize]
		}

		out = append(out, items...)
		if !hasMore {
			break
		}

		last := items[len(items)-1]
		if last == nil {
			break
		}
		cursor = &domain.ListCursor{ID: last.ID, CreatedAt: last.CreatedAt}
	}

	return out, nil
}

func (s *service) CreatePlanPrice(ctx context.Context, req domain.CreatePlanPriceRequest) (domain.PlanPriceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanPriceResponse{}, domain.ErrInvalidOrganization
	}

	planID, err := parseID(req.PlanID)
	if err != nil {
		return domain.PlanPriceResponse{}, err
	}

	code := normalizeCode(req.Code)
	if code == "" {
		return domain.PlanPriceResponse{}, domain.ErrInvalidCode
	}

	priceType := strings.ToLower(strings.TrimSpace(req.PriceType))
	if !isValidPriceType(priceType) {
		return domain.PlanPriceResponse{}, domain.ErrInvalidPriceType
	}

	billingInterval := strings.ToLower(strings.TrimSpace(req.BillingInterval))
	if !isValidBillingInterval(billingInterval) {
		return domain.PlanPriceResponse{}, domain.ErrInvalidInterval
	}

	if req.BillingIntervalCount < 1 {
		return domain.PlanPriceResponse{}, domain.ErrInvalidIntervalCount
	}

	var meterID *uuid.UUID
	if req.MeterID != nil && strings.TrimSpace(*req.MeterID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.MeterID))
		if err != nil || parsed == uuid.Nil {
			return domain.PlanPriceResponse{}, domain.ErrInvalidMeter
		}
		meterID = &parsed
	}
	if (priceType == domain.PriceTypeUsage || priceType == domain.PriceTypeTiered) && meterID == nil {
		return domain.PlanPriceResponse{}, domain.ErrInvalidMeter
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindPlanPriceByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.PlanPriceResponse{}, err
		}
		if existing != nil {
			return toPlanPriceResponse(*existing), nil
		}
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	now := time.Now().UTC()
	price := domain.PlanPrice{
		ID:                   uuid.New(),
		OrgID:                orgID,
		PlanID:               planID,
		MeterID:              meterID,
		Code:                 code,
		Name:                 strings.TrimSpace(req.Name),
		Description:          strings.TrimSpace(req.Description),
		PriceType:            priceType,
		BillingInterval:      billingInterval,
		BillingIntervalCount: req.BillingIntervalCount,
		AggregateUsage:       strings.TrimSpace(req.AggregateUsage),
		BillingUnit:          strings.TrimSpace(req.BillingUnit),
		MeterCode:            strings.TrimSpace(req.MeterCode),
		Active:               active,
		Metadata:             json.RawMessage(`{}`),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if idempotencyKey != "" {
		price.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.CreatePlanPrice(ctx, price); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindPlanPriceByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.PlanPriceResponse{}, findErr
			}
			if existing != nil {
				return toPlanPriceResponse(*existing), nil
			}
		}
		return domain.PlanPriceResponse{}, err
	}

	resp := toPlanPriceResponse(price)
	s.recordAudit(ctx, "plan_price.create", "plan_price", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) GetPlanPriceByID(ctx context.Context, req domain.GetPlanPriceRequest) (domain.PlanPriceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanPriceResponse{}, domain.ErrInvalidOrganization
	}

	id, err := parseID(req.ID)
	if err != nil {
		return domain.PlanPriceResponse{}, err
	}

	price, err := s.repo.FindPlanPriceByID(ctx, orgID, id)
	if err != nil {
		return domain.PlanPriceResponse{}, err
	}
	if price == nil {
		return domain.PlanPriceResponse{}, domain.ErrNotFound
	}

	return toPlanPriceResponse(*price), nil
}

func (s *service) ListPlanPrices(ctx context.Context, req domain.ListPlanPriceRequest) (domain.ListPlanPriceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListPlanPriceResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.PlanPriceListFilter{
		PriceType:       strings.ToLower(strings.TrimSpace(req.PriceType)),
		Active:          req.Active,
		BillingInterval: strings.ToLower(strings.TrimSpace(req.BillingInterval)),
	}
	if req.PlanID != "" {
		id, err := parseID(req.PlanID)
		if err != nil {
			return domain.ListPlanPriceResponse{}, err
		}
		filter.PlanID = id
	}
	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListPlanPriceResponse{}, err
	}

	items, err := s.repo.ListPlanPrices(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListPlanPriceResponse{}, err
	}

	resp := domain.ListPlanPriceResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.PlanPrice) string {
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

	resp.Prices = make([]domain.PlanPriceResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Prices = append(resp.Prices, toPlanPriceResponse(*item))
	}

	return resp, nil
}

func (s *service) CreatePlanAmount(ctx context.Context, req domain.CreatePlanAmountRequest) (domain.PlanAmountResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanAmountResponse{}, domain.ErrInvalidOrganization
	}

	planPriceID, err := parseID(req.PlanPriceID)
	if err != nil {
		return domain.PlanAmountResponse{}, err
	}

	currency := normalizeCurrency(req.Currency)
	if !isValidCurrency(currency) {
		return domain.PlanAmountResponse{}, domain.ErrInvalidCurrency
	}

	if req.UnitAmountCents < 0 {
		return domain.PlanAmountResponse{}, domain.ErrInvalidAmount
	}
	if req.MinimumAmountCents != nil && *req.MinimumAmountCents < 0 {
		return domain.PlanAmountResponse{}, domain.ErrInvalidAmount
	}
	if req.MaximumAmountCents != nil && *req.MaximumAmountCents < 0 {
		return domain.PlanAmountResponse{}, domain.ErrInvalidAmount
	}
	if req.MinimumAmountCents != nil && req.MaximumAmountCents != nil && *req.MinimumAmountCents > *req.MaximumAmountCents {
		return domain.PlanAmountResponse{}, domain.ErrInvalidAmount
	}

	effectiveFrom := time.Now().UTC()
	if req.EffectiveFrom != nil {
		effectiveFrom = req.EffectiveFrom.UTC()
	}
	var effectiveTo *time.Time
	if req.EffectiveTo != nil {
		t := req.EffectiveTo.UTC()
		if t.Before(effectiveFrom) {
			return domain.PlanAmountResponse{}, domain.ErrInvalidAmount
		}
		effectiveTo = &t
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindPlanAmountByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.PlanAmountResponse{}, err
		}
		if existing != nil {
			return toPlanAmountResponse(*existing), nil
		}
	}

	now := time.Now().UTC()
	amount := domain.PlanAmount{
		ID:                 uuid.New(),
		OrgID:              orgID,
		PlanPriceID:        planPriceID,
		Currency:           currency,
		UnitAmountCents:    req.UnitAmountCents,
		MinimumAmountCents: req.MinimumAmountCents,
		MaximumAmountCents: req.MaximumAmountCents,
		EffectiveFrom:      effectiveFrom,
		EffectiveTo:        effectiveTo,
		Metadata:           json.RawMessage(`{}`),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if idempotencyKey != "" {
		amount.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.CreatePlanAmount(ctx, amount); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindPlanAmountByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.PlanAmountResponse{}, findErr
			}
			if existing != nil {
				return toPlanAmountResponse(*existing), nil
			}
		}
		return domain.PlanAmountResponse{}, err
	}

	resp := toPlanAmountResponse(amount)
	s.recordAudit(ctx, "plan_amount.create", "plan_amount", resp.ID, nil, resp, nil)
	return resp, nil
}

func (s *service) GetPlanAmountByID(ctx context.Context, req domain.GetPlanAmountRequest) (domain.PlanAmountResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanAmountResponse{}, domain.ErrInvalidOrganization
	}

	id, err := parseID(req.ID)
	if err != nil {
		return domain.PlanAmountResponse{}, err
	}

	amount, err := s.repo.FindPlanAmountByID(ctx, orgID, id)
	if err != nil {
		return domain.PlanAmountResponse{}, err
	}
	if amount == nil {
		return domain.PlanAmountResponse{}, domain.ErrNotFound
	}

	return toPlanAmountResponse(*amount), nil
}

func (s *service) ListPlanAmounts(ctx context.Context, req domain.ListPlanAmountRequest) (domain.ListPlanAmountResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListPlanAmountResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.PlanAmountListFilter{
		Currency: normalizeCurrency(req.Currency),
	}
	if req.PlanPriceID != "" {
		id, err := parseID(req.PlanPriceID)
		if err != nil {
			return domain.ListPlanAmountResponse{}, err
		}
		filter.PlanPriceID = id
	}
	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListPlanAmountResponse{}, err
	}

	items, err := s.repo.ListPlanAmounts(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListPlanAmountResponse{}, err
	}

	resp := domain.ListPlanAmountResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.PlanAmount) string {
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

	resp.Amounts = make([]domain.PlanAmountResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Amounts = append(resp.Amounts, toPlanAmountResponse(*item))
	}

	return resp, nil
}

func (s *service) CreatePlanTier(ctx context.Context, req domain.CreatePlanTierRequest) (domain.PlanTierResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanTierResponse{}, domain.ErrInvalidOrganization
	}

	planPriceID, err := parseID(req.PlanPriceID)
	if err != nil {
		return domain.PlanTierResponse{}, err
	}

	tierMode := strings.ToLower(strings.TrimSpace(req.TierMode))
	if !isValidTierMode(tierMode) {
		return domain.PlanTierResponse{}, domain.ErrInvalidTierMode
	}
	if req.StartQuantity < 0 {
		return domain.PlanTierResponse{}, domain.ErrInvalidQuantity
	}
	if req.EndQuantity != nil && *req.EndQuantity < req.StartQuantity {
		return domain.PlanTierResponse{}, domain.ErrInvalidQuantity
	}
	if req.UnitAmountCents == nil && req.FlatAmountCents == nil {
		return domain.PlanTierResponse{}, domain.ErrInvalidAmount
	}
	if req.UnitAmountCents != nil && *req.UnitAmountCents < 0 {
		return domain.PlanTierResponse{}, domain.ErrInvalidAmount
	}
	if req.FlatAmountCents != nil && *req.FlatAmountCents < 0 {
		return domain.PlanTierResponse{}, domain.ErrInvalidAmount
	}

	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		return domain.PlanTierResponse{}, domain.ErrInvalidQuantity
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.FindPlanTierByIdempotencyKey(ctx, orgID, idempotencyKey)
		if err != nil {
			return domain.PlanTierResponse{}, err
		}
		if existing != nil {
			return toPlanTierResponse(*existing), nil
		}
	}

	now := time.Now().UTC()
	tier := domain.PlanTier{
		ID:              uuid.New(),
		OrgID:           orgID,
		PlanPriceID:     planPriceID,
		TierMode:        tierMode,
		StartQuantity:   req.StartQuantity,
		EndQuantity:     req.EndQuantity,
		UnitAmountCents: req.UnitAmountCents,
		FlatAmountCents: req.FlatAmountCents,
		Unit:            unit,
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if idempotencyKey != "" {
		tier.IdempotencyKey = &idempotencyKey
	}

	if err := s.repo.CreatePlanTier(ctx, tier); err != nil {
		if idempotencyKey != "" && errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := s.repo.FindPlanTierByIdempotencyKey(ctx, orgID, idempotencyKey)
			if findErr != nil {
				return domain.PlanTierResponse{}, findErr
			}
			if existing != nil {
				return toPlanTierResponse(*existing), nil
			}
		}
		return domain.PlanTierResponse{}, err
	}

	resp := toPlanTierResponse(tier)
	s.recordAudit(ctx, "plan_tier.create", "plan_tier", resp.ID, nil, resp, nil)
	return resp, nil
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

func (s *service) GetPlanTierByID(ctx context.Context, req domain.GetPlanTierRequest) (domain.PlanTierResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.PlanTierResponse{}, domain.ErrInvalidOrganization
	}

	id, err := parseID(req.ID)
	if err != nil {
		return domain.PlanTierResponse{}, err
	}

	tier, err := s.repo.FindPlanTierByID(ctx, orgID, id)
	if err != nil {
		return domain.PlanTierResponse{}, err
	}
	if tier == nil {
		return domain.PlanTierResponse{}, domain.ErrNotFound
	}

	return toPlanTierResponse(*tier), nil
}

func (s *service) ListPlanTiers(ctx context.Context, req domain.ListPlanTierRequest) (domain.ListPlanTierResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListPlanTierResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	filter := domain.PlanTierListFilter{
		TierMode: strings.ToLower(strings.TrimSpace(req.TierMode)),
	}
	if req.PlanPriceID != "" {
		id, err := parseID(req.PlanPriceID)
		if err != nil {
			return domain.ListPlanTierResponse{}, err
		}
		filter.PlanPriceID = id
	}
	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return domain.ListPlanTierResponse{}, err
	}

	items, err := s.repo.ListPlanTiers(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return domain.ListPlanTierResponse{}, err
	}

	resp := domain.ListPlanTierResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.PlanTier) string {
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

	resp.Tiers = make([]domain.PlanTierResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Tiers = append(resp.Tiers, toPlanTierResponse(*item))
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

func isValidPriceType(value string) bool {
	switch value {
	case domain.PriceTypeFlat, domain.PriceTypeUsage, domain.PriceTypeTiered:
		return true
	default:
		return false
	}
}

func isValidBillingInterval(value string) bool {
	switch value {
	case domain.BillingIntervalDay, domain.BillingIntervalWeek, domain.BillingIntervalMonth, domain.BillingIntervalYear:
		return true
	default:
		return false
	}
}

func isValidTierMode(value string) bool {
	switch value {
	case domain.TierModeGraduated, domain.TierModeVolume:
		return true
	default:
		return false
	}
}

func toPlanResponse(plan domain.Plan) domain.PlanResponse {
	resp := domain.PlanResponse{
		ID:          plan.ID.String(),
		OrgID:       plan.OrgID.String(),
		Code:        plan.Code,
		Name:        plan.Name,
		Description: plan.Description,
		Active:      plan.Active,
		Metadata:    plan.Metadata,
		CreatedAt:   plan.CreatedAt,
		UpdatedAt:   plan.UpdatedAt,
	}
	if plan.ProductID != nil {
		pid := plan.ProductID.String()
		resp.ProductID = &pid
	}
	return resp
}

func parseOptionalProductID(raw *string) (*uuid.UUID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(*raw))
	if err != nil || parsed == uuid.Nil {
		return nil, domain.ErrInvalidProduct
	}
	return &parsed, nil
}

func toPlanPriceResponse(price domain.PlanPrice) domain.PlanPriceResponse {
	var meterID *string
	if price.MeterID != nil {
		val := price.MeterID.String()
		meterID = &val
	}
	return domain.PlanPriceResponse{
		ID:                   price.ID.String(),
		OrgID:                price.OrgID.String(),
		PlanID:               price.PlanID.String(),
		MeterID:              meterID,
		Code:                 price.Code,
		Name:                 price.Name,
		Description:          price.Description,
		PriceType:            price.PriceType,
		BillingInterval:      price.BillingInterval,
		BillingIntervalCount: price.BillingIntervalCount,
		AggregateUsage:       price.AggregateUsage,
		BillingUnit:          price.BillingUnit,
		MeterCode:            price.MeterCode,
		Active:               price.Active,
		Metadata:             price.Metadata,
		CreatedAt:            price.CreatedAt,
		UpdatedAt:            price.UpdatedAt,
	}
}

func toPlanAmountResponse(amount domain.PlanAmount) domain.PlanAmountResponse {
	return domain.PlanAmountResponse{
		ID:                 amount.ID.String(),
		OrgID:              amount.OrgID.String(),
		PlanPriceID:        amount.PlanPriceID.String(),
		Currency:           amount.Currency,
		UnitAmountCents:    amount.UnitAmountCents,
		MinimumAmountCents: amount.MinimumAmountCents,
		MaximumAmountCents: amount.MaximumAmountCents,
		EffectiveFrom:      amount.EffectiveFrom,
		EffectiveTo:        amount.EffectiveTo,
		Metadata:           amount.Metadata,
		CreatedAt:          amount.CreatedAt,
		UpdatedAt:          amount.UpdatedAt,
	}
}

func toPlanTierResponse(tier domain.PlanTier) domain.PlanTierResponse {
	return domain.PlanTierResponse{
		ID:              tier.ID.String(),
		OrgID:           tier.OrgID.String(),
		PlanPriceID:     tier.PlanPriceID.String(),
		TierMode:        tier.TierMode,
		StartQuantity:   tier.StartQuantity,
		EndQuantity:     tier.EndQuantity,
		UnitAmountCents: tier.UnitAmountCents,
		FlatAmountCents: tier.FlatAmountCents,
		Unit:            tier.Unit,
		Metadata:        tier.Metadata,
		CreatedAt:       tier.CreatedAt,
		UpdatedAt:       tier.UpdatedAt,
	}
}
