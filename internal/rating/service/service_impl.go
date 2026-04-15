package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/config"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	"github.com/railzwaylabs/railzway/internal/db"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type service struct {
	db *gorm.DB

	ratingRepo       ratingdomain.Repository
	usageRepo        usagedomain.Repository
	subscriptionRepo subscriptiondomain.Repository
	planRepo         plandomain.Repository
	customerRepo     customerdomain.Repository
	invoiceRepo      invoicedomain.Repository
	invoiceSvc       invoicedomain.Service
	lateUsageGrace   time.Duration
	audit            *auditlog.Service
}

type Params struct {
	fx.In

	DB               *gorm.DB
	RatingRepo       ratingdomain.Repository
	UsageRepo        usagedomain.Repository
	SubscriptionRepo subscriptiondomain.Repository
	PlanRepo         plandomain.Repository
	CustomerRepo     customerdomain.Repository
	InvoiceRepo      invoicedomain.Repository `optional:"true"`
	InvoiceSvc       invoicedomain.Service    `optional:"true"`
	Config           *config.Config           `optional:"true"`
	Audit            *auditlog.Service        `optional:"true"`
}

func NewService(p Params) ratingdomain.Service {
	lateGrace := time.Duration(0)
	if p.Config != nil && p.Config.LateUsageGraceHours > 0 {
		lateGrace = time.Duration(p.Config.LateUsageGraceHours) * time.Hour
	}
	return &service{
		db:               p.DB,
		ratingRepo:       p.RatingRepo,
		usageRepo:        p.UsageRepo,
		subscriptionRepo: p.SubscriptionRepo,
		planRepo:         p.PlanRepo,
		customerRepo:     p.CustomerRepo,
		invoiceRepo:      p.InvoiceRepo,
		invoiceSvc:       p.InvoiceSvc,
		lateUsageGrace:   lateGrace,
		audit:            p.Audit,
	}
}

func (s *service) RateUsage(ctx context.Context, req ratingdomain.RateUsageRequest) (resp ratingdomain.RateUsageResponse, err error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return ratingdomain.RateUsageResponse{}, ratingdomain.ErrInvalidOrganization
	}

	ctx, span := telemetry.StartSpan(
		ctx,
		"rating.rate_usage",
		telemetry.UUIDAttr("billing.org_id", orgID),
		telemetry.StringAttr("billing.usage_event_id", strings.TrimSpace(req.UsageEventID)),
	)
	defer func() { telemetry.EndSpan(span, err) }()

	startedAt := time.Now()
	defer func() { telemetry.ObserveOperation("rating.rate_usage", time.Since(startedAt), err) }()

	usageID, err := parseID(req.UsageEventID)
	if err != nil {
		return ratingdomain.RateUsageResponse{}, ratingdomain.ErrInvalidUsageEvent
	}

	existing, err := s.ratingRepo.FindRatingByUsageEvent(ctx, orgID, usageID)
	if err != nil {
		return ratingdomain.RateUsageResponse{}, err
	}
	if existing != nil {
		if err := s.ensureLateAdjustment(ctx, orgID, *existing); err != nil {
			return ratingdomain.RateUsageResponse{}, err
		}
		if err := s.usageRepo.UpdateUsageStatus(ctx, orgID, existing.UsageEventID, usagedomain.StatusRated); err != nil {
			return ratingdomain.RateUsageResponse{}, err
		}
		resp = ratingdomain.RateUsageResponse{RatingResult: toRatingResponse(*existing)}
		span.SetAttributes(
			telemetry.UUIDAttr("billing.rating_result_id", existing.ID),
			telemetry.BoolAttr("billing.idempotent_hit", true),
		)
		return resp, nil
	}

	usageEvent, err := s.findUsageEvent(ctx, orgID, usageID)
	if err != nil {
		return ratingdomain.RateUsageResponse{}, err
	}
	if usageEvent == nil {
		return ratingdomain.RateUsageResponse{}, ratingdomain.ErrUsageNotFound
	}

	customer, err := s.customerRepo.FindByID(ctx, orgID, usageEvent.CustomerID)
	if err != nil {
		return ratingdomain.RateUsageResponse{}, err
	}
	if customer == nil {
		return ratingdomain.RateUsageResponse{}, ratingdomain.ErrUsageNotFound
	}

	subscription, aggPeriodStart, aggPeriodEnd, err := s.findSubscriptionForUsage(ctx, orgID, usageEvent.CustomerID, usageEvent.RecordedAt)
	if err != nil {
		return ratingdomain.RateUsageResponse{}, err
	}
	if subscription == nil {
		return ratingdomain.RateUsageResponse{}, ratingdomain.ErrPricingNotFound
	}

	price, err := s.findPriceForMeter(ctx, orgID, subscription.ID, usageEvent.MeterID, usageEvent.RecordedAt)
	if err != nil {
		return ratingdomain.RateUsageResponse{}, err
	}
	if price == nil {
		return ratingdomain.RateUsageResponse{}, ratingdomain.ErrPricingNotFound
	}

	quantity := usageEvent.Value
	var (
		unitAmount  int64
		totalAmount int64
		planAmount  *plandomain.PlanAmount
		tierMode    string
	)

	if price.PriceType != plandomain.PriceTypeTiered {
		unitAmount, totalAmount, planAmount, tierMode, err = s.calculateAmount(ctx, orgID, price, subscription.Currency, usageEvent.RecordedAt, quantity)
		if err != nil {
			return ratingdomain.RateUsageResponse{}, err
		}
		if totalAmount < 0 || unitAmount < 0 {
			return ratingdomain.RateUsageResponse{}, ratingdomain.ErrInvalidAmount
		}
	} else {
		planAmount, err = s.pickPlanAmount(ctx, orgID, price.ID, subscription.Currency, usageEvent.RecordedAt)
		if err != nil {
			return ratingdomain.RateUsageResponse{}, err
		}
		if planAmount == nil {
			return ratingdomain.RateUsageResponse{}, ratingdomain.ErrPricingNotFound
		}
	}

	var result ratingdomain.RatingResult
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ratingRepo := s.ratingRepo.WithTx(tx)

		planAmountIDValue := planAmount.ID
		if err := lockAggregateKey(ctx, tx, buildAggregateLockKey(orgID, usageEvent.CustomerID, price.ID, planAmountIDValue, usageEvent.MeterID, aggPeriodStart, aggPeriodEnd)); err != nil {
			return err
		}
		existingAgg, err := ratingRepo.GetUsageAggregateForUpdate(ctx, orgID, usageEvent.CustomerID, price.ID, planAmountIDValue, usageEvent.MeterID, aggPeriodStart, aggPeriodEnd)
		if err != nil {
			return err
		}

		if price.PriceType == plandomain.PriceTypeTiered {
			tiers, err := s.planRepo.ListPlanTiersByPrice(ctx, orgID, price.ID)
			if err != nil {
				return err
			}
			if len(tiers) == 0 {
				return ratingdomain.ErrPricingNotFound
			}

			prevQty := float64(0)
			if existingAgg != nil {
				prevQty = existingAgg.Quantity
			}
			newQty := prevQty + quantity
			newUnit, newTotal := calculateTieredAmount(newQty, tiers)
			_, prevTotal := calculateTieredAmount(prevQty, tiers)
			unitAmount = newUnit
			totalAmount = newTotal - prevTotal
			if totalAmount < 0 {
				totalAmount = 0
			}
			tierMode = plandomain.TierModeGraduated
			if tiers[0] != nil && tiers[0].TierMode != "" {
				tierMode = tiers[0].TierMode
			}
		}

		if totalAmount < 0 || unitAmount < 0 {
			return ratingdomain.ErrInvalidAmount
		}

		now := time.Now().UTC()
		metadata := buildRatingMetadata(price, usageEvent, planAmount, tierMode)
		result = ratingdomain.RatingResult{
			ID:              uuid.New(),
			OrgID:           orgID,
			UsageEventID:    usageEvent.ID,
			CustomerID:      usageEvent.CustomerID,
			SubscriptionID:  &subscription.ID,
			PlanPriceID:     price.ID,
			PlanAmountID:    planAmountID(planAmount),
			MeterID:         usageEvent.MeterID,
			Currency:        subscription.Currency,
			Quantity:        quantity,
			UnitAmountCents: unitAmount,
			AmountCents:     totalAmount,
			Source:          ratingdomain.SourceUsage,
			WindowStart:     usageEvent.RecordedAt,
			WindowEnd:       usageEvent.RecordedAt,
			Metadata:        metadata,
			CreatedAt:       now,
		}

		if err := ratingRepo.CreateRatingResult(ctx, result); err != nil {
			return err
		}

		aggregate := ratingdomain.UsageAggregate{
			ID:             uuid.New(),
			OrgID:          orgID,
			CustomerID:     usageEvent.CustomerID,
			SubscriptionID: &subscription.ID,
			PlanPriceID:    price.ID,
			PlanAmountID:   planAmountID(planAmount),
			MeterID:        usageEvent.MeterID,
			Currency:       subscription.Currency,
			PeriodStart:    aggPeriodStart,
			PeriodEnd:      aggPeriodEnd,
			Quantity:       quantity,
			AmountCents:    totalAmount,
			LastEventAt:    &usageEvent.RecordedAt,
			Metadata:       metadata,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if existingAgg != nil {
			aggregate.ID = existingAgg.ID
			aggregate.Quantity = existingAgg.Quantity + quantity
			aggregate.AmountCents = existingAgg.AmountCents + totalAmount
			aggregate.CreatedAt = existingAgg.CreatedAt
		}

		if existingAgg == nil {
			if err := ratingRepo.CreateUsageAggregate(ctx, aggregate); err != nil {
				if db.IsDuplicateKeyErr(err) {
					lockedAgg, lockErr := ratingRepo.GetUsageAggregateForUpdate(ctx, orgID, usageEvent.CustomerID, price.ID, planAmountIDValue, usageEvent.MeterID, aggPeriodStart, aggPeriodEnd)
					if lockErr != nil {
						return lockErr
					}
					if lockedAgg != nil {
						aggregate.ID = lockedAgg.ID
						aggregate.Quantity = lockedAgg.Quantity + quantity
						aggregate.AmountCents = lockedAgg.AmountCents + totalAmount
						aggregate.CreatedAt = lockedAgg.CreatedAt
						if err := ratingRepo.UpdateUsageAggregate(ctx, aggregate); err != nil {
							return err
						}
					}
				} else {
					return err
				}
			}
		} else {
			if err := ratingRepo.UpdateUsageAggregate(ctx, aggregate); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		if db.IsDuplicateKeyErr(err) {
			existing, findErr := s.ratingRepo.FindRatingByUsageEvent(ctx, orgID, usageEvent.ID)
			if findErr != nil {
				return ratingdomain.RateUsageResponse{}, findErr
			}
			if existing != nil {
				if err := s.ensureLateAdjustment(ctx, orgID, *existing); err != nil {
					return ratingdomain.RateUsageResponse{}, err
				}
				if err := s.usageRepo.UpdateUsageStatus(ctx, orgID, existing.UsageEventID, usagedomain.StatusRated); err != nil {
					return ratingdomain.RateUsageResponse{}, err
				}
				resp = ratingdomain.RateUsageResponse{RatingResult: toRatingResponse(*existing)}
				span.SetAttributes(
					telemetry.UUIDAttr("billing.rating_result_id", existing.ID),
					telemetry.BoolAttr("billing.idempotent_hit", true),
				)
				return resp, nil
			}
		}
		return ratingdomain.RateUsageResponse{}, err
	}

	if err := s.ensureLateAdjustment(ctx, orgID, result); err != nil {
		return ratingdomain.RateUsageResponse{}, err
	}
	if err := s.usageRepo.UpdateUsageStatus(ctx, orgID, usageEvent.ID, usagedomain.StatusRated); err != nil {
		return ratingdomain.RateUsageResponse{}, err
	}

	resp = ratingdomain.RateUsageResponse{RatingResult: toRatingResponse(result)}
	span.SetAttributes(
		telemetry.UUIDAttr("billing.rating_result_id", result.ID),
		telemetry.UUIDAttr("billing.customer_id", result.CustomerID),
		telemetry.UUIDAttr("billing.meter_id", result.MeterID),
		telemetry.Int64Attr("billing.amount_cents", result.AmountCents),
	)
	s.recordAudit(ctx, "rating.result.create", "rating_result", result.ID.String(), nil, resp.RatingResult, nil)
	return resp, nil
}

func (s *service) ListRatingResults(ctx context.Context, req ratingdomain.ListRatingResultsRequest) (ratingdomain.ListRatingResultsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return ratingdomain.ListRatingResultsResponse{}, ratingdomain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))

	filter := ratingdomain.RatingResultFilter{}

	if req.CustomerID != "" {
		id, err := parseUUID(req.CustomerID, ratingdomain.ErrInvalidCustomer)
		if err != nil {
			return ratingdomain.ListRatingResultsResponse{}, err
		}
		filter.CustomerID = id
	}

	if req.SubscriptionID != "" {
		id, err := parseUUID(req.SubscriptionID, ratingdomain.ErrInvalidSubscription)
		if err != nil {
			return ratingdomain.ListRatingResultsResponse{}, err
		}
		filter.SubscriptionID = id
	}

	if req.PlanPriceID != "" {
		id, err := parseUUID(req.PlanPriceID, ratingdomain.ErrInvalidPlanPrice)
		if err != nil {
			return ratingdomain.ListRatingResultsResponse{}, err
		}
		filter.PlanPriceID = id
	}

	if req.MeterID != "" {
		id, err := parseUUID(req.MeterID, ratingdomain.ErrInvalidMeter)
		if err != nil {
			return ratingdomain.ListRatingResultsResponse{}, err
		}
		filter.MeterID = id
	}

	if req.UsageEventID != "" {
		id, err := parseUUID(req.UsageEventID, ratingdomain.ErrInvalidUsageEvent)
		if err != nil {
			return ratingdomain.ListRatingResultsResponse{}, err
		}
		filter.UsageEventID = id
	}

	if req.WindowStartFrom != nil && req.WindowStartTo != nil && req.WindowStartTo.Before(*req.WindowStartFrom) {
		return ratingdomain.ListRatingResultsResponse{}, ratingdomain.ErrInvalidPeriod
	}

	filter.WindowStartFrom = req.WindowStartFrom
	filter.WindowStartTo = req.WindowStartTo

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return ratingdomain.ListRatingResultsResponse{}, err
	}

	items, err := s.ratingRepo.ListRatingResults(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return ratingdomain.ListRatingResultsResponse{}, err
	}

	resp := ratingdomain.ListRatingResultsResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *ratingdomain.RatingResult) string {
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

	resp.Results = make([]ratingdomain.RatingResultResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Results = append(resp.Results, toRatingResponse(*item))
	}

	return resp, nil
}

func (s *service) ListUsageAggregates(ctx context.Context, req ratingdomain.ListUsageAggregatesRequest) (ratingdomain.ListUsageAggregatesResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return ratingdomain.ListUsageAggregatesResponse{}, ratingdomain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))

	filter := ratingdomain.UsageAggregateFilter{}

	if req.CustomerID != "" {
		id, err := parseUUID(req.CustomerID, ratingdomain.ErrInvalidCustomer)
		if err != nil {
			return ratingdomain.ListUsageAggregatesResponse{}, err
		}
		filter.CustomerID = id
	}

	if req.SubscriptionID != "" {
		id, err := parseUUID(req.SubscriptionID, ratingdomain.ErrInvalidSubscription)
		if err != nil {
			return ratingdomain.ListUsageAggregatesResponse{}, err
		}
		filter.SubscriptionID = id
	}

	if req.PlanPriceID != "" {
		id, err := parseUUID(req.PlanPriceID, ratingdomain.ErrInvalidPlanPrice)
		if err != nil {
			return ratingdomain.ListUsageAggregatesResponse{}, err
		}
		filter.PlanPriceID = id
	}

	if req.MeterID != "" {
		id, err := parseUUID(req.MeterID, ratingdomain.ErrInvalidMeter)
		if err != nil {
			return ratingdomain.ListUsageAggregatesResponse{}, err
		}
		filter.MeterID = id
	}

	if req.PeriodStartFrom != nil && req.PeriodStartTo != nil && req.PeriodStartTo.Before(*req.PeriodStartFrom) {
		return ratingdomain.ListUsageAggregatesResponse{}, ratingdomain.ErrInvalidPeriod
	}

	filter.PeriodStartFrom = req.PeriodStartFrom
	filter.PeriodStartTo = req.PeriodStartTo

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return ratingdomain.ListUsageAggregatesResponse{}, err
	}

	items, err := s.ratingRepo.ListUsageAggregates(ctx, orgID, filter, pageSize+1, cursor)
	if err != nil {
		return ratingdomain.ListUsageAggregatesResponse{}, err
	}

	resp := ratingdomain.ListUsageAggregatesResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *ratingdomain.UsageAggregate) string {
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

	resp.Aggregates = make([]ratingdomain.UsageAggregateResponse, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Aggregates = append(resp.Aggregates, toUsageAggregateResponse(*item))
	}

	return resp, nil
}

func (s *service) findUsageEvent(ctx context.Context, orgID, usageID uuid.UUID) (*usagedomain.UsageEvent, error) {
	var event usagedomain.UsageEvent
	err := s.db.WithContext(ctx).
		Where("org_id = ? AND id = ?", orgID, usageID).
		First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

type subscriptionPeriodRow struct {
	subscriptiondomain.Subscription
	PeriodStart time.Time `gorm:"column:period_start"`
	PeriodEnd   time.Time `gorm:"column:period_end"`
}

func (s *service) findSubscriptionForUsage(ctx context.Context, orgID, customerID uuid.UUID, at time.Time) (*subscriptiondomain.Subscription, time.Time, time.Time, error) {
	var row subscriptionPeriodRow
	err := s.db.WithContext(ctx).Raw(
		`SELECT s.*, p.period_start, p.period_end
		 FROM subscription_periods p
		 JOIN subscriptions s ON s.id = p.subscription_id
		 WHERE p.org_id = ? AND s.customer_id = ? AND p.period_start <= ? AND p.period_end >= ?
		 ORDER BY p.period_start DESC
		 LIMIT 1`,
		orgID, customerID, at, at,
	).Scan(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, time.Time{}, time.Time{}, nil
		}
		return nil, time.Time{}, time.Time{}, err
	}
	if row.ID == uuid.Nil {
		return nil, time.Time{}, time.Time{}, nil
	}
	if at.Before(row.StartAt) {
		return nil, time.Time{}, time.Time{}, nil
	}
	if row.CancelAt != nil && at.After(row.CancelAt.UTC()) {
		return nil, time.Time{}, time.Time{}, nil
	}
	if row.CanceledAt != nil && at.After(row.CanceledAt.UTC()) {
		return nil, time.Time{}, time.Time{}, nil
	}
	if row.EndedAt != nil && at.After(row.EndedAt.UTC()) {
		return nil, time.Time{}, time.Time{}, nil
	}
	return &row.Subscription, row.PeriodStart, row.PeriodEnd, nil
}

func (s *service) findPriceForMeter(ctx context.Context, orgID, subscriptionID, meterID uuid.UUID, at time.Time) (*plandomain.PlanPrice, error) {
	var price plandomain.PlanPrice
	err := s.db.WithContext(ctx).
		Raw(
			`SELECT p.*
			 FROM subscription_items i
			 JOIN plan_prices p ON p.id = i.plan_price_id
			 WHERE i.org_id = ? AND i.subscription_id = ? AND p.meter_id = ?
			   AND i.start_at <= ?
			   AND (i.end_at IS NULL OR i.end_at >= ?)
			 LIMIT 1`,
			orgID, subscriptionID, meterID, at, at,
		).
		Scan(&price).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if price.ID == uuid.Nil {
		return nil, nil
	}
	return &price, nil
}

func (s *service) calculateAmount(ctx context.Context, orgID uuid.UUID, price *plandomain.PlanPrice, currency string, at time.Time, quantity float64) (int64, int64, *plandomain.PlanAmount, string, error) {
	amount, err := s.pickPlanAmount(ctx, orgID, price.ID, currency, at)
	if err != nil {
		return 0, 0, nil, "", err
	}
	if amount == nil {
		return 0, 0, nil, "", ratingdomain.ErrPricingNotFound
	}

	switch price.PriceType {
	case plandomain.PriceTypeTiered:
		tiers, err := s.planRepo.ListPlanTiersByPrice(ctx, orgID, price.ID)
		if err != nil {
			return 0, 0, nil, "", err
		}
		if len(tiers) == 0 {
			return 0, 0, nil, "", ratingdomain.ErrPricingNotFound
		}
		total, unit := calculateTieredAmount(quantity, tiers)
		tierMode := plandomain.TierModeGraduated
		if tiers[0] != nil && tiers[0].TierMode != "" {
			tierMode = tiers[0].TierMode
		}
		return unit, total, amount, tierMode, nil
	default:
		unit := amount.UnitAmountCents
		total := int64(roundAmount(quantity * float64(unit)))
		return unit, total, amount, "", nil
	}
}

func (s *service) pickPlanAmount(ctx context.Context, orgID, planPriceID uuid.UUID, currency string, at time.Time) (*plandomain.PlanAmount, error) {
	var amount plandomain.PlanAmount
	err := s.db.WithContext(ctx).
		Where("org_id = ? AND plan_price_id = ? AND currency = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)",
			orgID, planPriceID, currency, at, at).
		Order("effective_from desc").
		First(&amount).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &amount, nil
}

func calculateTieredAmount(quantity float64, tiers []*plandomain.PlanTier) (int64, int64) {
	if quantity <= 0 {
		return 0, 0
	}

	mode := plandomain.TierModeGraduated
	if tiers[0] != nil && tiers[0].TierMode != "" {
		mode = tiers[0].TierMode
	}

	switch mode {
	case plandomain.TierModeVolume:
		return calculateTieredVolume(quantity, tiers)
	default:
		return calculateTieredGraduated(quantity, tiers)
	}
}

func calculateTieredVolume(quantity float64, tiers []*plandomain.PlanTier) (int64, int64) {
	var matched *plandomain.PlanTier
	for _, tier := range tiers {
		if tier == nil {
			continue
		}
		if quantity < tier.StartQuantity {
			continue
		}
		if tier.EndQuantity != nil && quantity > *tier.EndQuantity {
			continue
		}
		matched = tier
		break
	}
	if matched == nil {
		matched = tiers[len(tiers)-1]
	}

	unit := int64(0)
	if matched.UnitAmountCents != nil {
		unit = *matched.UnitAmountCents
	}
	total := int64(roundAmount(quantity * float64(unit)))
	if matched.FlatAmountCents != nil {
		total += *matched.FlatAmountCents
	}
	return unit, total
}

func calculateTieredGraduated(quantity float64, tiers []*plandomain.PlanTier) (int64, int64) {
	var total int64
	var lastUnit int64
	for _, tier := range tiers {
		if tier == nil {
			continue
		}
		if quantity <= tier.StartQuantity {
			continue
		}
		end := quantity
		if tier.EndQuantity != nil && *tier.EndQuantity < end {
			end = *tier.EndQuantity
		}
		tierQty := end - tier.StartQuantity
		if tierQty <= 0 {
			continue
		}
		unit := int64(0)
		if tier.UnitAmountCents != nil {
			unit = *tier.UnitAmountCents
		}
		lastUnit = unit
		total += int64(roundAmount(tierQty * float64(unit)))
		if tier.FlatAmountCents != nil {
			total += *tier.FlatAmountCents
		}
	}
	return lastUnit, total
}

type ratingMetadata struct {
	MeterCode            string `json:"meter_code,omitempty"`
	PlanPriceCode        string `json:"plan_price_code,omitempty"`
	PriceType            string `json:"price_type,omitempty"`
	BillingInterval      string `json:"billing_interval,omitempty"`
	BillingIntervalCount int    `json:"billing_interval_count,omitempty"`
	BillingUnit          string `json:"billing_unit,omitempty"`
	PlanAmountID         string `json:"plan_amount_id,omitempty"`
	TierMode             string `json:"tier_mode,omitempty"`
}

func buildRatingMetadata(price *plandomain.PlanPrice, usageEvent *usagedomain.UsageEvent, amount *plandomain.PlanAmount, tierMode string) json.RawMessage {
	meta := ratingMetadata{}
	if usageEvent != nil {
		meta.MeterCode = usageEvent.MeterCode
	}
	if price != nil {
		meta.PlanPriceCode = price.Code
		meta.PriceType = price.PriceType
		meta.BillingInterval = price.BillingInterval
		meta.BillingIntervalCount = price.BillingIntervalCount
		meta.BillingUnit = price.BillingUnit
	}
	if amount != nil {
		meta.PlanAmountID = amount.ID.String()
	}
	if tierMode != "" {
		meta.TierMode = tierMode
	}

	raw, err := json.Marshal(meta)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func planAmountID(amount *plandomain.PlanAmount) *uuid.UUID {
	if amount == nil {
		return nil
	}
	id := amount.ID
	return &id
}

func roundAmount(value float64) float64 {
	if value < 0 {
		return 0
	}
	return float64(int64(value + 0.5))
}

func parseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ratingdomain.ErrInvalidUsageEvent
	}
	return id, nil
}

func parseUUID(value string, invalidErr error) (uuid.UUID, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return uuid.Nil, invalidErr
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, invalidErr
	}
	return id, nil
}

func decodeCursor(token string) (*ratingdomain.ListCursor, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil
	}
	decoded, err := pagination.DecodeCursor(raw)
	if err != nil {
		return nil, ratingdomain.ErrInvalidCursor
	}
	if decoded == nil || decoded.ID == "" || decoded.CreatedAt == "" {
		return nil, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
	if err != nil {
		return nil, ratingdomain.ErrInvalidCursor
	}
	parsedID, err := uuid.Parse(decoded.ID)
	if err != nil {
		return nil, ratingdomain.ErrInvalidCursor
	}
	return &ratingdomain.ListCursor{
		ID:        parsedID,
		CreatedAt: parsedTime,
	}, nil
}

func toRatingResponse(result ratingdomain.RatingResult) ratingdomain.RatingResultResponse {
	var subID *string
	if result.SubscriptionID != nil {
		val := result.SubscriptionID.String()
		subID = &val
	}
	return ratingdomain.RatingResultResponse{
		ID:              result.ID.String(),
		UsageEventID:    result.UsageEventID.String(),
		CustomerID:      result.CustomerID.String(),
		SubscriptionID:  subID,
		PlanPriceID:     result.PlanPriceID.String(),
		MeterID:         result.MeterID.String(),
		Currency:        result.Currency,
		Quantity:        result.Quantity,
		UnitAmountCents: result.UnitAmountCents,
		AmountCents:     result.AmountCents,
		Source:          result.Source,
		WindowStart:     result.WindowStart,
		WindowEnd:       result.WindowEnd,
		CreatedAt:       result.CreatedAt,
	}
}

func toUsageAggregateResponse(agg ratingdomain.UsageAggregate) ratingdomain.UsageAggregateResponse {
	var subID *string
	if agg.SubscriptionID != nil {
		val := agg.SubscriptionID.String()
		subID = &val
	}
	var amountID *string
	if agg.PlanAmountID != nil {
		val := agg.PlanAmountID.String()
		amountID = &val
	}
	return ratingdomain.UsageAggregateResponse{
		ID:             agg.ID.String(),
		OrgID:          agg.OrgID.String(),
		CustomerID:     agg.CustomerID.String(),
		SubscriptionID: subID,
		PlanPriceID:    agg.PlanPriceID.String(),
		PlanAmountID:   amountID,
		MeterID:        agg.MeterID.String(),
		Currency:       agg.Currency,
		PeriodStart:    agg.PeriodStart,
		PeriodEnd:      agg.PeriodEnd,
		Quantity:       agg.Quantity,
		AmountCents:    agg.AmountCents,
		LastEventAt:    agg.LastEventAt,
		CreatedAt:      agg.CreatedAt,
		UpdatedAt:      agg.UpdatedAt,
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

func (s *service) ensureLateAdjustment(ctx context.Context, orgID uuid.UUID, result ratingdomain.RatingResult) error {
	if s.invoiceRepo == nil || s.invoiceSvc == nil {
		return nil
	}
	if result.AmountCents <= 0 {
		return nil
	}
	if result.SubscriptionID == nil || *result.SubscriptionID == uuid.Nil {
		return nil
	}

	period, err := s.subscriptionRepo.FindSubscriptionPeriodByTime(ctx, orgID, *result.SubscriptionID, result.WindowStart)
	if err != nil {
		return err
	}
	if period == nil {
		return nil
	}

	inv, err := s.invoiceRepo.FindInvoiceBySubscriptionPeriod(ctx, orgID, *result.SubscriptionID, period.PeriodStart, period.PeriodEnd)
	if err != nil || inv == nil {
		return err
	}
	if inv.Status == invoicedomain.StatusVoid {
		return nil
	}
	if inv.Status == invoicedomain.StatusDraft {
		if s.lateUsageGrace > 0 && result.WindowEnd.After(period.PeriodEnd.Add(s.lateUsageGrace)) {
			// Draft is beyond grace window; treat as late adjustment instead of mutating draft.
		} else {
			_, err := s.invoiceSvc.RecalculateDraftInvoice(ctx, invoicedomain.RecalculateDraftInvoiceRequest{
				ID: inv.ID.String(),
			})
			return err
		}
	}

	ratingID := result.ID.String()
	planPriceID := result.PlanPriceID.String()
	meterID := result.MeterID.String()
	windowStart := result.WindowStart
	windowEnd := result.WindowEnd
	line := invoicedomain.AdjustmentLine{
		RatingResultID:  &ratingID,
		PlanPriceID:     &planPriceID,
		MeterID:         &meterID,
		Quantity:        result.Quantity,
		UnitAmountCents: result.UnitAmountCents,
		AmountCents:     result.AmountCents,
		Currency:        result.Currency,
		Description:     "Late usage adjustment",
		WindowStart:     &windowStart,
		WindowEnd:       &windowEnd,
	}
	req := invoicedomain.CreateAdjustmentInvoiceRequest{
		SubscriptionID: result.SubscriptionID.String(),
		PeriodStart:    period.PeriodStart,
		PeriodEnd:      period.PeriodEnd,
		BaseInvoiceID:  inv.ID.String(),
		Reason:         "late_usage",
		Lines:          []invoicedomain.AdjustmentLine{line},
		IdempotencyKey: "late_usage:" + ratingID,
	}
	_, err = s.invoiceSvc.CreateAdjustmentInvoice(ctx, req)
	return err
}

func buildAggregateLockKey(orgID, customerID, priceID, planAmountID, meterID uuid.UUID, periodStart, periodEnd time.Time) string {
	return fmt.Sprintf(
		"rating.aggregate:%s:%s:%s:%s:%s:%s:%s",
		orgID.String(),
		customerID.String(),
		priceID.String(),
		planAmountID.String(),
		meterID.String(),
		periodStart.UTC().Format(time.RFC3339Nano),
		periodEnd.UTC().Format(time.RFC3339Nano),
	)
}

func lockAggregateKey(ctx context.Context, tx *gorm.DB, key string) error {
	if tx == nil || tx.Dialector == nil || tx.Dialector.Name() != "postgres" {
		return nil
	}
	lockKey := int64(crc32.ChecksumIEEE([]byte(key)))
	return tx.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", lockKey).Error
}
