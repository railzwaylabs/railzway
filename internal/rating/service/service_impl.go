package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	billingcycledomain "github.com/railzwaylabs/railzway/internal/billingcycle/domain"
	"github.com/railzwaylabs/railzway/internal/bootstrap"
	priceamountdomain "github.com/railzwaylabs/railzway/internal/priceamount/domain"
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
	priceAmountRepo priceamountdomain.Repository
	orgGate         bootstrap.OrgGate
}

type ServiceParam struct {
	fx.In

	DB              *gorm.DB
	Log             *zap.Logger
	GenID           *snowflake.Node
	PriceAmountRepo priceamountdomain.Repository
	OrgGate         bootstrap.OrgGate `optional:"true"`
}

func NewService(p ServiceParam) ratingdomain.Service {
	return &Service{
		db:  p.DB,
		log: p.Log.Named("rating.service"),

		genID:           p.GenID,
		repo:            repository.NewRepository(p.DB),
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

		now := time.Now().UTC()
		cycleDuration := cycle.PeriodEnd.Sub(cycle.PeriodStart).Seconds()

		for _, item := range items {
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

			if item.MeterID == nil {
				if err := s.rateFlatItem(ctx, tx, cycle, item, featureCode, start, end, prorationFactor, now); err != nil {
					return err
				}
				continue
			}

			windows, err := s.buildPriceWindows(ctx, tx, cycle.OrgID, item.PriceID, item.MeterID, start, end)
			if err != nil {
				return err
			}

			for _, window := range windows {
				qty, err := repoTx.AggregateUsage(ctx, cycle.OrgID, cycle.SubscriptionID, *item.MeterID, window.Start, window.End)
				if err != nil {
					return err
				}

				if err := s.insertRatingWindow(ctx, tx, cycle, item, window, qty, "usage_events", featureCode, now); err != nil {
					return err
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
	now time.Time,
) error {
	priceAmount, err := s.resolvePriceAmountAt(ctx, tx, cycle.OrgID, item.PriceID, nil, periodStart)
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
		Currency:       priceAmount.Currency,
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
	periodStart, periodEnd time.Time,
) ([]priceWindow, error) {
	boundaries := []time.Time{periodStart, periodEnd}

	specific, err := s.priceAmountRepo.ListOverlapping(ctx, tx, orgID, priceID, meterID, "", periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	boundaries = appendEffectiveBoundaries(boundaries, specific, periodStart, periodEnd)

	defaults, err := s.priceAmountRepo.ListOverlapping(ctx, tx, orgID, priceID, nil, "", periodStart, periodEnd)
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

		amount, err := s.resolvePriceAmountAt(ctx, tx, orgID, priceID, meterID, start)
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
	at time.Time,
) (*priceamountdomain.PriceAmount, error) {
	amount, err := s.priceAmountRepo.FindEffectiveAt(ctx, tx, orgID, priceID, meterID, "", at)
	if err != nil {
		return nil, err
	}
	if amount != nil || meterID == nil {
		return amount, nil
	}
	return s.priceAmountRepo.FindEffectiveAt(ctx, tx, orgID, priceID, nil, "", at)
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
		Currency:       window.Amount.Currency,
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
