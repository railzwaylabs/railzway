package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	billingcycledomain "github.com/railzwaylabs/railzway/internal/billingcycle/domain"
	"github.com/railzwaylabs/railzway/internal/bootstrap"
	pricedomain "github.com/railzwaylabs/railzway/internal/price/domain"
	priceamountdomain "github.com/railzwaylabs/railzway/internal/priceamount/domain"
	pricetierdomain "github.com/railzwaylabs/railzway/internal/pricetier/domain"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	"github.com/railzwaylabs/railzway/internal/rating/repository"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db  *gorm.DB
	log *zap.Logger

	genID           *snowflake.Node
	repo            ratingdomain.Repository
	priceRepo       pricedomain.Repository
	priceAmountRepo priceamountdomain.Repository
	orgGate         bootstrap.OrgGate
}

const defaultCurrency = "USD"

type ServiceParam struct {
	fx.In

	DB              *gorm.DB
	Log             *zap.Logger
	GenID           *snowflake.Node
	PriceRepo       pricedomain.Repository
	PriceAmountRepo priceamountdomain.Repository
	OrgGate         bootstrap.OrgGate `optional:"true"`
}

func NewService(p ServiceParam) ratingdomain.Service {
	return &Service{
		db:  p.DB,
		log: p.Log.Named("rating.service"),

		genID:           p.GenID,
		repo:            repository.NewRepository(p.DB),
		priceRepo:       p.PriceRepo,
		priceAmountRepo: p.PriceAmountRepo,
		orgGate:         p.OrgGate,
	}
}

func (s *Service) RunRating(ctx context.Context, billingCycleID string) error {
	cycleID, err := parseID(billingCycleID)
	if err != nil {
		return ratingdomain.ErrInvalidBillingCycle
	}

	cycle, err := s.repo.GetBillingCycle(ctx, cycleID)
	if err != nil {
		return err
	}
	if cycle == nil {
		return ratingdomain.ErrBillingCycleNotFound
	}
	if cycle.Status != billingcycledomain.BillingCycleStatusClosing {
		return ratingdomain.ErrBillingCycleNotClosing
	}
	if s.orgGate != nil {
		if err := s.orgGate.MustBeActive(ctx, cycle.OrgID); err != nil {
			return err
		}
	}

	subscription, err := s.repo.GetSubscription(ctx, cycle.OrgID, cycle.SubscriptionID)
	if err != nil {
		return err
	}
	if subscription == nil {
		return ratingdomain.ErrSubscriptionNotFound
	}

	items, err := s.repo.ListSubscriptionItems(ctx, cycle.OrgID, cycle.SubscriptionID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return ratingdomain.ErrNoSubscriptionItems
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := repository.NewRepository(tx)

		if err := repoTx.DeleteRatingResults(ctx, cycle.ID); err != nil {
			return err
		}

		entitlements, err := repoTx.ListEntitlements(ctx, cycle.OrgID, cycle.SubscriptionID, cycle.PeriodStart, cycle.PeriodEnd)
		if err != nil {
			return err
		}

		currency, err := s.resolveSubscriptionCurrency(ctx, tx, subscription)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		cycleDuration := cycle.PeriodEnd.Sub(cycle.PeriodStart).Seconds()

		for _, item := range items {
			price, err := s.priceRepo.FindByID(ctx, tx, cycle.OrgID, item.PriceID)
			if err != nil {
				return err
			}
			if price == nil {
				return ratingdomain.ErrMissingPriceAmount
			}

			featureCode, ent, err := s.resolveEntitlementWithWindow(ctx, tx, item, entitlements)
			if err != nil {
				return fmt.Errorf("rating failed for item %s: %w", item.ID, err)
			}

			start, end, active := resolveEffectiveWindow(
				cycle.PeriodStart, cycle.PeriodEnd,
				subscription.StartAt, subscription.EndedAt, subscription.CanceledAt,
				getEntEffectiveFrom(ent), getEntEffectiveTo(ent),
			)

			if !active {
				continue
			}

			prorationFactor := calculateProrationFactor(start, end, cycleDuration)

			if price.PricingModel == pricedomain.Flat {
				if err := s.rateFlatItem(ctx, tx, cycle, item, featureCode, start, end, prorationFactor, currency, now); err != nil {
					return err
				}
				continue
			}

			if item.MeterID == nil {
				return ratingdomain.ErrMissingMeter
			}

			windows, err := s.buildPriceWindows(ctx, tx, cycle.OrgID, item.PriceID, item.MeterID, currency, start, end)
			if err != nil {
				return err
			}

			for _, window := range windows {
				qty, err := repoTx.AggregateUsage(ctx, cycle.OrgID, cycle.SubscriptionID, *item.MeterID, window.Start, window.End)
				if err != nil {
					return err
				}

				switch price.PricingModel {
				case pricedomain.PerUnit:
					if err := s.insertRatingWindow(ctx, tx, cycle, item, window, qty, "usage_events", featureCode, currency, now); err != nil {
						return err
					}
				case pricedomain.TieredVolume, pricedomain.TieredGraduated:
					tiers, err := s.listPriceTiers(ctx, tx, cycle.OrgID, item.PriceID)
					if err != nil {
						return err
					}
					if len(tiers) == 0 {
						return ratingdomain.ErrMissingPriceTier
					}
					var amount int64
					var unitPrice int64
					if price.PricingModel == pricedomain.TieredVolume {
						amount, unitPrice, err = calculateTieredVolumeAmount(qty, tiers)
					} else {
						amount, unitPrice, err = calculateTieredGraduatedAmount(qty, tiers)
					}
					if err != nil {
						return err
					}
					if err := s.insertTieredRating(ctx, tx, cycle, item, window, qty, unitPrice, amount, currency, price.PricingModel, featureCode, now); err != nil {
						return err
					}
				default:
					return pricedomain.ErrUnsupportedPricingModel
				}
			}
		}

		return nil
	})
}

func getEntEffectiveFrom(ent *subscriptiondomain.SubscriptionEntitlement) time.Time {
	if ent == nil {
		return time.Time{}
	}
	return ent.EffectiveFrom
}

func getEntEffectiveTo(ent *subscriptiondomain.SubscriptionEntitlement) *time.Time {
	if ent == nil {
		return nil
	}
	return ent.EffectiveTo
}

func (s *Service) resolveEntitlementWithWindow(
	ctx context.Context,
	tx *gorm.DB,
	item ratingdomain.SubscriptionItemRow,
	entitlements []subscriptiondomain.SubscriptionEntitlement,
) (string, *subscriptiondomain.SubscriptionEntitlement, error) {
	if item.MeterID != nil {
		for _, ent := range entitlements {
			if ent.MeterID != nil && *ent.MeterID == *item.MeterID {
				return ent.FeatureCode, &ent, nil
			}
		}
		return "", nil, nil
	}

	var productID snowflake.ID
	err := tx.WithContext(ctx).Raw("SELECT product_id FROM prices WHERE id = ? AND org_id = ?", item.PriceID, item.OrgID).Scan(&productID).Error
	if err != nil {
		return "", nil, err
	}

	for _, ent := range entitlements {
		if ent.ProductID == productID {
			return ent.FeatureCode, &ent, nil
		}
	}

	return "", nil, nil
}

func (s *Service) rateFlatItem(
	ctx context.Context,
	tx *gorm.DB,
	cycle *ratingdomain.BillingCycleRow,
	item ratingdomain.SubscriptionItemRow,
	featureCode string,
	periodStart, periodEnd time.Time,
	prorationFactor float64,
	currency string,
	now time.Time,
) error {
	priceAmount, err := s.resolvePriceAmountAt(ctx, tx, cycle.OrgID, item.PriceID, nil, currency, periodStart)
	if err != nil {
		return err
	}
	if priceAmount == nil {
		return ratingdomain.ErrMissingPriceAmount
	}

	baseAmount := float64(priceAmount.UnitAmountCents)
	finalAmount := roundRatingAmount(baseAmount * prorationFactor)

	checksum := buildRatingChecksum(cycle.ID, cycle.SubscriptionID, item.PriceID, item.MeterID, featureCode, periodStart, periodEnd)

	repoTx := repository.NewRepository(tx)
	return repoTx.InsertRatingResult(ctx, ratingdomain.RatingResult{
		ID:             s.genID.Generate(),
		OrgID:          cycle.OrgID,
		SubscriptionID: cycle.SubscriptionID,
		BillingCycleID: cycle.ID,
		PriceID:        item.PriceID,
		FeatureCode:    featureCode,
		MeterID:        item.MeterID,
		Source:         "flat_rate",
		Quantity:       prorationFactor,
		UnitPrice:      priceAmount.UnitAmountCents,
		Amount:         finalAmount,
		Currency:       currency,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Checksum:       checksum,
		CreatedAt:      now,
	})
}

func (s *Service) buildPriceWindows(
	ctx context.Context,
	tx *gorm.DB,
	orgID, priceID snowflake.ID,
	meterID *snowflake.ID,
	currency string,
	periodStart, periodEnd time.Time,
) ([]priceWindow, error) {
	boundaries := []time.Time{periodStart, periodEnd}

	specific, err := s.priceAmountRepo.ListOverlapping(ctx, tx, orgID, priceID, meterID, currency, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	boundaries = appendEffectiveBoundaries(boundaries, specific, periodStart, periodEnd)

	defaults, err := s.priceAmountRepo.ListOverlapping(ctx, tx, orgID, priceID, nil, currency, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	boundaries = appendEffectiveBoundaries(boundaries, defaults, periodStart, periodEnd)

	boundaries = uniqueSortedTimes(boundaries)
	windows := make([]priceWindow, 0, len(boundaries)-1)
	for i := 0; i < len(boundaries)-1; i++ {
		start := boundaries[i]
		end := boundaries[i+1]
		if !end.After(start) {
			continue
		}

		amount, err := s.resolvePriceAmountAt(ctx, tx, orgID, priceID, meterID, currency, start)
		if err != nil {
			return nil, err
		}
		if amount == nil {
			return nil, ratingdomain.ErrMissingPriceAmount
		}

		windows = append(windows, priceWindow{
			Start:  start,
			End:    end,
			Amount: amount,
		})
	}

	return windows, nil
}

func (s *Service) resolvePriceAmountAt(
	ctx context.Context,
	tx *gorm.DB,
	orgID, priceID snowflake.ID,
	meterID *snowflake.ID,
	currency string,
	at time.Time,
) (*priceamountdomain.PriceAmount, error) {
	amount, err := s.priceAmountRepo.FindEffectiveAt(ctx, tx, orgID, priceID, meterID, currency, at)
	if err != nil {
		return nil, err
	}
	if amount != nil || meterID == nil {
		return amount, nil
	}
	return s.priceAmountRepo.FindEffectiveAt(ctx, tx, orgID, priceID, nil, currency, at)
}

func (s *Service) insertRatingWindow(
	ctx context.Context,
	tx *gorm.DB,
	cycle *ratingdomain.BillingCycleRow,
	item ratingdomain.SubscriptionItemRow,
	window priceWindow,
	quantity float64,
	source string,
	featureCode string,
	currency string,
	now time.Time,
) error {
	if quantity < 0 {
		return ratingdomain.ErrInvalidQuantity
	}

	unitPrice := window.Amount.UnitAmountCents
	amount := roundRatingAmount(quantity * float64(unitPrice))

	if window.Amount.MinimumAmountCents != nil && *window.Amount.MinimumAmountCents > 0 {
		if amount < *window.Amount.MinimumAmountCents {
			amount = *window.Amount.MinimumAmountCents
		}
	}
	if window.Amount.MaximumAmountCents != nil && *window.Amount.MaximumAmountCents > 0 {
		if amount > *window.Amount.MaximumAmountCents {
			amount = *window.Amount.MaximumAmountCents
		}
	}

	checksum := buildRatingChecksum(cycle.ID, cycle.SubscriptionID, item.PriceID, item.MeterID, featureCode, window.Start, window.End)

	repoTx := repository.NewRepository(tx)
	return repoTx.InsertRatingResult(ctx, ratingdomain.RatingResult{
		ID:             s.genID.Generate(),
		OrgID:          cycle.OrgID,
		SubscriptionID: cycle.SubscriptionID,
		BillingCycleID: cycle.ID,
		MeterID:        item.MeterID,
		PriceID:        item.PriceID,
		FeatureCode:    featureCode,
		Quantity:       quantity,
		UnitPrice:      unitPrice,
		Amount:         amount,
		Currency:       currency,
		PeriodStart:    window.Start,
		PeriodEnd:      window.End,
		Source:         source,
		Checksum:       checksum,
		CreatedAt:      now,
	})
}

type priceWindow struct {
	Start  time.Time
	End    time.Time
	Amount *priceamountdomain.PriceAmount
}

func (s *Service) resolveSubscriptionCurrency(ctx context.Context, tx *gorm.DB, subscription *subscriptiondomain.Subscription) (string, error) {
	if subscription == nil {
		return "", ratingdomain.ErrSubscriptionNotFound
	}
	if subscription.DefaultCurrency != nil {
		if currency := strings.ToUpper(strings.TrimSpace(*subscription.DefaultCurrency)); currency != "" {
			return currency, nil
		}
	}

	customerCurrency, err := s.loadCustomerCurrency(ctx, tx, subscription.OrgID, subscription.CustomerID)
	if err != nil {
		return "", err
	}
	if customerCurrency != "" {
		return customerCurrency, nil
	}

	orgCurrency, err := s.loadOrgCurrency(ctx, tx, subscription.OrgID)
	if err != nil {
		return "", err
	}
	if orgCurrency == "" {
		orgCurrency = defaultCurrency
	}
	return orgCurrency, nil
}

func (s *Service) loadCustomerCurrency(ctx context.Context, tx *gorm.DB, orgID, customerID snowflake.ID) (string, error) {
	if customerID == 0 {
		return "", nil
	}
	var row struct {
		Currency string `gorm:"column:currency"`
	}
	if err := tx.WithContext(ctx).Raw(
		`SELECT currency FROM customers WHERE org_id = ? AND id = ? LIMIT 1`,
		orgID,
		customerID,
	).Scan(&row).Error; err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimSpace(row.Currency)), nil
}

func (s *Service) loadOrgCurrency(ctx context.Context, tx *gorm.DB, orgID snowflake.ID) (string, error) {
	var row struct {
		Currency string `gorm:"column:currency"`
	}
	if err := tx.WithContext(ctx).Raw(
		`SELECT currency FROM organization_billing_preferences WHERE org_id = ? LIMIT 1`,
		orgID,
	).Scan(&row).Error; err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimSpace(row.Currency)), nil
}

func (s *Service) listPriceTiers(ctx context.Context, tx *gorm.DB, orgID, priceID snowflake.ID) ([]pricetierdomain.PriceTier, error) {
	var tiers []pricetierdomain.PriceTier
	if err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, price_id, tier_mode, start_quantity, end_quantity, unit_amount_cents, flat_amount_cents, unit, metadata, created_at, updated_at
		 FROM price_tiers
		 WHERE org_id = ? AND price_id = ?
		 ORDER BY start_quantity ASC`,
		orgID,
		priceID,
	).Scan(&tiers).Error; err != nil {
		return nil, err
	}
	return tiers, nil
}

func (s *Service) insertTieredRating(
	ctx context.Context,
	tx *gorm.DB,
	cycle *ratingdomain.BillingCycleRow,
	item ratingdomain.SubscriptionItemRow,
	window priceWindow,
	quantity float64,
	unitPrice int64,
	amount int64,
	currency string,
	pricingModel pricedomain.PricingModel,
	featureCode string,
	now time.Time,
) error {
	source := "tiered"
	switch pricingModel {
	case pricedomain.TieredVolume:
		source = "tiered_volume"
	case pricedomain.TieredGraduated:
		source = "tiered_graduated"
	}

	checksum := buildRatingChecksum(cycle.ID, cycle.SubscriptionID, item.PriceID, item.MeterID, featureCode, window.Start, window.End)

	repoTx := repository.NewRepository(tx)
	return repoTx.InsertRatingResult(ctx, ratingdomain.RatingResult{
		ID:             s.genID.Generate(),
		OrgID:          cycle.OrgID,
		SubscriptionID: cycle.SubscriptionID,
		BillingCycleID: cycle.ID,
		MeterID:        item.MeterID,
		PriceID:        item.PriceID,
		FeatureCode:    featureCode,
		Quantity:       quantity,
		UnitPrice:      unitPrice,
		Amount:         amount,
		Currency:       currency,
		PeriodStart:    window.Start,
		PeriodEnd:      window.End,
		Source:         source,
		Checksum:       checksum,
		CreatedAt:      now,
	})
}

func calculateTieredVolumeAmount(quantity float64, tiers []pricetierdomain.PriceTier) (int64, int64, error) {
	if quantity <= 0 {
		return 0, 0, nil
	}
	sorted := append([]pricetierdomain.PriceTier(nil), tiers...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartQuantity < sorted[j].StartQuantity })

	var matched *pricetierdomain.PriceTier
	for i := range sorted {
		tier := sorted[i]
		if quantity < tier.StartQuantity {
			continue
		}
		if tier.EndQuantity != nil && quantity > *tier.EndQuantity {
			continue
		}
		matched = &tier
	}
	if matched == nil {
		return 0, 0, ratingdomain.ErrMissingPriceTier
	}

	var amount int64
	if matched.UnitAmountCents != nil {
		amount += roundRatingAmount(quantity * float64(*matched.UnitAmountCents))
	}
	if matched.FlatAmountCents != nil {
		amount += *matched.FlatAmountCents
	}

	unitPrice := int64(0)
	if quantity > 0 {
		unitPrice = roundRatingAmount(float64(amount) / quantity)
	}
	return amount, unitPrice, nil
}

func calculateTieredGraduatedAmount(quantity float64, tiers []pricetierdomain.PriceTier) (int64, int64, error) {
	if quantity <= 0 {
		return 0, 0, nil
	}
	sorted := append([]pricetierdomain.PriceTier(nil), tiers...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartQuantity < sorted[j].StartQuantity })

	var amount int64
	matched := false
	for i := range sorted {
		tier := sorted[i]
		tierQty := tierQuantity(quantity, tier.StartQuantity, tier.EndQuantity)
		if tierQty <= 0 {
			if quantity < tier.StartQuantity {
				break
			}
			continue
		}
		matched = true
		if tier.UnitAmountCents != nil {
			amount += roundRatingAmount(tierQty * float64(*tier.UnitAmountCents))
		}
		if tier.FlatAmountCents != nil {
			amount += *tier.FlatAmountCents
		}
	}

	if !matched {
		return 0, 0, ratingdomain.ErrMissingPriceTier
	}

	unitPrice := int64(0)
	if quantity > 0 {
		unitPrice = roundRatingAmount(float64(amount) / quantity)
	}
	return amount, unitPrice, nil
}

func tierQuantity(total float64, start float64, end *float64) float64 {
	if total < start {
		return 0
	}
	upper := total
	if end != nil && *end < upper {
		upper = *end
	}
	if upper < start {
		return 0
	}
	return math.Max(0, upper-start+1)
}

func appendEffectiveBoundaries(
	boundaries []time.Time,
	amounts []priceamountdomain.PriceAmount,
	periodStart, periodEnd time.Time,
) []time.Time {
	for _, amount := range amounts {
		start := amount.EffectiveFrom.UTC()
		if start.After(periodStart) && start.Before(periodEnd) {
			boundaries = append(boundaries, start)
		}
		if amount.EffectiveTo != nil {
			end := amount.EffectiveTo.UTC()
			if end.After(periodStart) && end.Before(periodEnd) {
				boundaries = append(boundaries, end)
			}
		}
	}
	return boundaries
}

func uniqueSortedTimes(times []time.Time) []time.Time {
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	out := make([]time.Time, 0, len(times))
	for _, t := range times {
		if len(out) == 0 || !t.Equal(out[len(out)-1]) {
			out = append(out, t)
		}
	}
	return out
}

func parseID(value string) (snowflake.ID, error) {
	return snowflake.ParseString(strings.TrimSpace(value))
}
