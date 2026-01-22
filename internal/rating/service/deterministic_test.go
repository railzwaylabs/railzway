package service

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	billingcycledomain "github.com/smallbiznis/railzway/internal/billingcycle/domain"
	pricedomain "github.com/smallbiznis/railzway/internal/price/domain"
	priceamountdomain "github.com/smallbiznis/railzway/internal/priceamount/domain"
	ratingdomain "github.com/smallbiznis/railzway/internal/rating/domain"
	subscriptiondomain "github.com/smallbiznis/railzway/internal/subscription/domain"
	usagedomain "github.com/smallbiznis/railzway/internal/usage/domain"
	"github.com/smallbiznis/railzway/pkg/db/option"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestRating_Deterministic_Idempotency(t *testing.T) {
	// 1. Setup DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate
	err = db.AutoMigrate(
		&ratingdomain.RatingResult{},
		// &usagedomain.UsageEvent{}, // The service queries usage_events table
		&subscriptiondomain.SubscriptionItem{},
		&subscriptiondomain.Subscription{},
		&subscriptiondomain.SubscriptionEntitlement{},
		&billingcycledomain.BillingCycle{},
		&pricedomain.Price{},
		// PriceAmount table not strictly needed if we stub repo, but good for consistency
	)
	assert.NoError(t, err)

	// Usage table needs manual creation or migrate
	err = db.AutoMigrate(&usagedomain.UsageEvent{})
	assert.NoError(t, err)

	node, _ := snowflake.NewNode(1)
	logger := zap.NewNop()

	priceAmountStub := &priceAmountStub{
		Amounts: make(map[string]priceamountdomain.PriceAmount),
	}

	svc := NewService(ServiceParam{
		DB:              db,
		Log:             logger,
		GenID:           node,
		PriceAmountRepo: priceAmountStub,
	})

	// 2. Seed Data
	orgID := node.Generate()
	subID := node.Generate()
	cycleID := node.Generate()

	productID := node.Generate()
	priceID := node.Generate()
	meterID := node.Generate()

	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)
	end := now

	// Cycle
	db.Create(&billingcycledomain.BillingCycle{
		ID:             cycleID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PeriodStart:    start,
		PeriodEnd:      end,
		Status:         billingcycledomain.BillingCycleStatusClosing,
	})

	// Price (Allowed Snapshot)
	db.Create(&pricedomain.Price{
		ID:        priceID,
		OrgID:     orgID,
		ProductID: productID,
		Code:      "price_123",
		Active:    true,
	})

	// Price Amount (via stub)
	priceAmountStub.Amounts[priceID.String()] = priceamountdomain.PriceAmount{
		PriceID:         priceID,
		UnitAmountCents: 100, // $1.00
	}

	// Subscription Item (Metered)
	db.Create(&subscriptiondomain.SubscriptionItem{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		PriceID:        priceID,
		MeterID:        &meterID,
		Quantity:       1,
		BillingMode:    "METERED",
	})

	// Entitlement (Snapshot)
	featureCode := "feat_metered"
	db.Create(&subscriptiondomain.SubscriptionEntitlement{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		ProductID:      productID,
		FeatureCode:    featureCode,
		MeterID:        &meterID,
		EffectiveFrom:  start.Add(-1 * time.Hour),
	})

	// Usage Events
	db.Create(&usagedomain.UsageEvent{
		ID:             node.Generate(),
		OrgID:          orgID,
		MeterID:        meterID,
		SubscriptionID: subID, // Value, not pointer
		Value:          10.0,
		RecordedAt:     start.Add(1 * time.Hour),
		Status:         usagedomain.UsageStatusEnriched, // Correct status for aggregation
	})
	db.Create(&usagedomain.UsageEvent{
		ID:             node.Generate(),
		OrgID:          orgID,
		MeterID:        meterID,
		SubscriptionID: subID,
		Value:          5.0,
		RecordedAt:     start.Add(2 * time.Hour),
		Status:         usagedomain.UsageStatusEnriched,
	})

	// 3. Run Rating (First Time)
	err = svc.RunRating(context.Background(), cycleID.String())
	assert.NoError(t, err)

	var results []ratingdomain.RatingResult
	db.Where("billing_cycle_id = ?", cycleID).Find(&results)
	if assert.Len(t, results, 1) {
		assert.Equal(t, float64(15.0), results[0].Quantity)
		assert.Equal(t, featureCode, results[0].FeatureCode)
	} else {
		return
	}
	firstRunID := results[0].ID

	// 4. Run Rating (Second Time - Idempotency)
	// Should produce SAME content. ID CAN Change due to DELETE-INSERT.
	// But check Deterministic outputs.
	err = svc.RunRating(context.Background(), cycleID.String())
	assert.NoError(t, err)

	var results2 []ratingdomain.RatingResult
	db.Where("billing_cycle_id = ?", cycleID).Find(&results2)
	assert.Len(t, results2, 1)
	assert.Equal(t, float64(15.0), results2[0].Quantity)
	assert.Equal(t, featureCode, results2[0].FeatureCode)

	// Assert ID changed (proving we did a full replacements)
	assert.NotEqual(t, firstRunID, results2[0].ID, "ID should change due to Delete-Insert")
}

// Stub
type priceAmountStub struct {
	Amounts map[string]priceamountdomain.PriceAmount
}

// Implement required methods
func (s *priceAmountStub) Insert(ctx context.Context, db *gorm.DB, amount *priceamountdomain.PriceAmount) error {
	return nil
}
func (s *priceAmountStub) FindOne(ctx context.Context, db *gorm.DB, amount *priceamountdomain.PriceAmount) (*priceamountdomain.PriceAmount, error) {
	return nil, nil
}
func (s *priceAmountStub) FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*priceamountdomain.PriceAmount, error) {
	return nil, nil
}
func (s *priceAmountStub) List(ctx context.Context, db *gorm.DB, f priceamountdomain.PriceAmount, opts ...option.QueryOption) ([]priceamountdomain.PriceAmount, error) {
	return nil, nil
}
func (s *priceAmountStub) Update(ctx context.Context, db *gorm.DB, amount *priceamountdomain.PriceAmount) (*priceamountdomain.PriceAmount, error) {
	return nil, nil
}

func (s *priceAmountStub) FindEffectiveAt(ctx context.Context, db *gorm.DB, orgID, priceID snowflake.ID, meterID *snowflake.ID, currency string, at time.Time) (*priceamountdomain.PriceAmount, error) {
	// Simple mock lookup by PriceID
	if v, ok := s.Amounts[priceID.String()]; ok {
		return &v, nil
	}
	return nil, nil
}

func (s *priceAmountStub) FindPrevious(ctx context.Context, db *gorm.DB, orgID, priceID snowflake.ID, meterID *snowflake.ID, currency string, before time.Time) (*priceamountdomain.PriceAmount, error) {
	return nil, nil
}
func (s *priceAmountStub) FindNext(ctx context.Context, db *gorm.DB, orgID, priceID snowflake.ID, meterID *snowflake.ID, currency string, after time.Time) (*priceamountdomain.PriceAmount, error) {
	return nil, nil
}
func (s *priceAmountStub) ListOverlapping(ctx context.Context, db *gorm.DB, orgID, priceID snowflake.ID, meterID *snowflake.ID, currency string, start, end time.Time) ([]priceamountdomain.PriceAmount, error) {
	return nil, nil
}
func (s *priceAmountStub) FindLatestByPriceAndCurrency(ctx context.Context, db *gorm.DB, orgID, priceID snowflake.ID, currency string) (*priceamountdomain.PriceAmount, error) {
	return nil, nil
}
func (s *priceAmountStub) FindUpcoming(ctx context.Context, db *gorm.DB, orgID, priceID snowflake.ID, meterID *snowflake.ID, currency string) (*priceamountdomain.PriceAmount, error) {
	return nil, nil
}
