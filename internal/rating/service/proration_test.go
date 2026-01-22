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
	"github.com/smallbiznis/railzway/pkg/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TestProration_MidCycleSubscriptionStart validates PRORATION RULE 1:
// Subscription starts mid-cycle, proration factor applied to flat fee
func TestProration_MidCycleSubscriptionStart(t *testing.T) {
	db, svc, node := setupProrationTest(t)

	orgID := node.Generate()
	subID := node.Generate()
	cycleID := node.Generate()
	productID := node.Generate()
	priceID := node.Generate()

	// Cycle: Jan 1 - Jan 31 (31 days)
	cycleStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	// Subscription starts Jan 16 (16 days active out of 31)
	subStart := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)

	// Expected proration: 16/31 ≈ 0.516129
	expectedFactor := 16.0 / 31.0

	// Seed data
	priceAmountStub := svc.(*Service).priceAmountRepo.(*priceAmountStub)
	priceRepoStub := svc.(*Service).priceRepo.(*priceRepoStub)
	seedProrationData(t, db, node, priceAmountStub, priceRepoStub, orgID, subID, cycleID, productID, priceID, cycleStart, cycleEnd, subStart, nil, 10000) // $100.00

	// Run rating
	err := svc.RunRating(context.Background(), cycleID.String())
	require.NoError(t, err)

	// Verify results
	var results []ratingdomain.RatingResult
	db.Where("billing_cycle_id = ?", cycleID).Find(&results)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, cycleID, result.BillingCycleID)
	assert.Equal(t, priceID, result.PriceID)

	// CRITICAL: Verify proration factor stored in quantity
	assert.InDelta(t, expectedFactor, result.Quantity, 0.0001)

	// CRITICAL: Verify prorated amount (10000 * 0.516129 ≈ 5161)
	expectedAmount := int64(10000.0 * expectedFactor)
	assert.InDelta(t, expectedAmount, result.Amount, 1) // Allow 1 cent rounding

	// CRITICAL: Verify exact time window persisted
	assert.Equal(t, subStart, result.PeriodStart)
	assert.Equal(t, cycleEnd, result.PeriodEnd)

	// CRITICAL: Verify deterministic checksum
	assert.NotEmpty(t, result.Checksum)
	firstChecksum := result.Checksum

	// Re-run rating (idempotency test)
	err = svc.RunRating(context.Background(), cycleID.String())
	require.NoError(t, err)

	var results2 []ratingdomain.RatingResult
	db.Where("billing_cycle_id = ?", cycleID).Find(&results2)
	require.Len(t, results2, 1)

	// CRITICAL: Verify identical outputs (determinism)
	assert.Equal(t, result.Quantity, results2[0].Quantity)
	assert.Equal(t, result.Amount, results2[0].Amount)
	assert.Equal(t, firstChecksum, results2[0].Checksum)
}

// TestProration_MidCycleSubscriptionEnd validates PRORATION RULE 1:
// Subscription ends mid-cycle
func TestProration_MidCycleSubscriptionEnd(t *testing.T) {
	db, svc, node := setupProrationTest(t)

	orgID := node.Generate()
	subID := node.Generate()
	cycleID := node.Generate()
	productID := node.Generate()
	priceID := node.Generate()

	// Cycle: Jan 1 - Jan 31
	cycleStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	// Subscription ends Jan 15 (15 days active)
	subEnd := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)

	expectedFactor := 15.0 / 31.0

	priceAmountStub := svc.(*Service).priceAmountRepo.(*priceAmountStub)
	priceRepoStub := svc.(*Service).priceRepo.(*priceRepoStub)
	seedProrationData(t, db, node, priceAmountStub, priceRepoStub, orgID, subID, cycleID, productID, priceID, cycleStart, cycleEnd, cycleStart, &subEnd, 10000)

	err := svc.RunRating(context.Background(), cycleID.String())
	require.NoError(t, err)

	var results []ratingdomain.RatingResult
	db.Where("billing_cycle_id = ?", cycleID).Find(&results)
	require.Len(t, results, 1)

	result := results[0]
	assert.InDelta(t, expectedFactor, result.Quantity, 0.0001)
	assert.Equal(t, cycleStart, result.PeriodStart)
	assert.Equal(t, subEnd, result.PeriodEnd)
}

// TestProration_PlanChangeMidCycle validates PRORATION RULE 2:
// Plan change creates MULTIPLE rating rows with different periods
func TestProration_PlanChangeMidCycle(t *testing.T) {
	db, svc, node := setupProrationTest(t)

	orgID := node.Generate()
	subID := node.Generate()
	cycleID := node.Generate()

	productA := node.Generate()
	productB := node.Generate()
	priceA := node.Generate()
	priceB := node.Generate()

	// Cycle: Jan 1 - Jan 31
	cycleStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	// Plan change on Jan 16
	planChangeDate := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)

	// Seed cycle
	db.Create(&billingcycledomain.BillingCycle{
		ID:             cycleID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PeriodStart:    cycleStart,
		PeriodEnd:      cycleEnd,
		Status:         billingcycledomain.BillingCycleStatusClosing,
	})

	// Subscription (full month)
	db.Create(&subscriptiondomain.Subscription{
		ID:         subID,
		OrgID:      orgID,
		CustomerID: node.Generate(),
		Status:     subscriptiondomain.SubscriptionStatusActive,
		StartAt:    cycleStart,
	})

	// TWO subscription items (representing plan change)
	db.Create(&subscriptiondomain.SubscriptionItem{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		PriceID:        priceA,
		BillingMode:    "FLAT",
	})
	db.Create(&subscriptiondomain.SubscriptionItem{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		PriceID:        priceB,
		BillingMode:    "FLAT",
	})

	// Prices
	db.Create(&pricedomain.Price{
		ID:        priceA,
		OrgID:     orgID,
		ProductID: productA,
		Code:      "price_a",
		Active:    true,
	})
	db.Create(&pricedomain.Price{
		ID:        priceB,
		OrgID:     orgID,
		ProductID: productB,
		Code:      "price_b",
		Active:    true,
	})

	// Price amounts
	priceAmountStub := svc.(*Service).priceAmountRepo.(*priceAmountStub)
	priceAmountStub.Amounts[priceA.String()] = priceamountdomain.PriceAmount{
		PriceID:         priceA,
		UnitAmountCents: 10000, // $100
		Currency:        "USD",
	}
	priceAmountStub.Amounts[priceB.String()] = priceamountdomain.PriceAmount{
		PriceID:         priceB,
		UnitAmountCents: 15000, // $150
		Currency:        "USD",
	}

	// Configure price repo stub
	priceRepoStub := svc.(*Service).priceRepo.(*priceRepoStub)
	priceRepoStub.Prices[priceA.String()] = pricedomain.Price{
		ID:        priceA,
		ProductID: productA,
	}
	priceRepoStub.Prices[priceB.String()] = pricedomain.Price{
		ID:        priceB,
		ProductID: productB,
	}

	// TWO entitlements with different validity periods (PLAN CHANGE)
	db.Create(&subscriptiondomain.SubscriptionEntitlement{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		ProductID:      productA,
		FeatureCode:    "plan_a",
		EffectiveFrom:  cycleStart,
		EffectiveTo:    &planChangeDate, // Ends at plan change
	})
	db.Create(&subscriptiondomain.SubscriptionEntitlement{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		ProductID:      productB,
		FeatureCode:    "plan_b",
		EffectiveFrom:  planChangeDate, // Starts at plan change
		EffectiveTo:    nil,
	})

	// Run rating
	err := svc.RunRating(context.Background(), cycleID.String())
	require.NoError(t, err)

	// CRITICAL: Verify TWO rows created (plan change split)
	var results []ratingdomain.RatingResult
	db.Where("billing_cycle_id = ?", cycleID).Order("period_start").Find(&results)
	require.Len(t, results, 2, "Plan change MUST create multiple rating rows")

	// Row 1: Plan A (Jan 1 - Jan 16, 15 days)
	rowA := results[0]
	assert.Equal(t, priceA, rowA.PriceID)
	assert.Equal(t, "plan_a", rowA.FeatureCode)
	assert.Equal(t, cycleStart, rowA.PeriodStart)
	assert.Equal(t, planChangeDate, rowA.PeriodEnd)
	assert.InDelta(t, 15.0/31.0, rowA.Quantity, 0.0001)
	amountA := float64(10000) * 15.0 / 31.0
	assert.InDelta(t, amountA, float64(rowA.Amount), 1.0)

	// Row 2: Plan B (Jan 16 - Feb 1, 16 days)
	rowB := results[1]
	assert.Equal(t, priceB, rowB.PriceID)
	assert.Equal(t, "plan_b", rowB.FeatureCode)
	assert.Equal(t, planChangeDate, rowB.PeriodStart)
	assert.Equal(t, cycleEnd, rowB.PeriodEnd)
	assert.InDelta(t, 16.0/31.0, rowB.Quantity, 0.0001)
	amountB := float64(15000) * 16.0 / 31.0
	assert.InDelta(t, amountB, float64(rowB.Amount), 1.0)

	// CRITICAL: Verify different checksums
	assert.NotEqual(t, rowA.Checksum, rowB.Checksum)
}

// TestProration_MeteredUsageWithWindow validates that metered usage
// respects the effective window (not full cycle)
func TestProration_MeteredUsageWithWindow(t *testing.T) {
	db, svc, node := setupProrationTest(t)

	orgID := node.Generate()
	subID := node.Generate()
	cycleID := node.Generate()
	productID := node.Generate()
	priceID := node.Generate()
	meterID := node.Generate()

	// Cycle: Jan 1 - Jan 31
	cycleStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	// Subscription starts Jan 16
	subStart := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)

	// Seed
	db.Create(&billingcycledomain.BillingCycle{
		ID:             cycleID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PeriodStart:    cycleStart,
		PeriodEnd:      cycleEnd,
		Status:         billingcycledomain.BillingCycleStatusClosing,
	})

	db.Create(&subscriptiondomain.Subscription{
		ID:         subID,
		OrgID:      orgID,
		CustomerID: node.Generate(),
		Status:     subscriptiondomain.SubscriptionStatusActive,
		StartAt:    subStart,
	})

	db.Create(&subscriptiondomain.SubscriptionItem{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		PriceID:        priceID,
		MeterID:        &meterID,
		BillingMode:    "METERED",
	})

	db.Create(&pricedomain.Price{
		ID:        priceID,
		OrgID:     orgID,
		ProductID: productID,
		Active:    true,
	})

	priceAmountStub := svc.(*Service).priceAmountRepo.(*priceAmountStub)
	priceAmountStub.Amounts[priceID.String()] = priceamountdomain.PriceAmount{
		PriceID:         priceID,
		UnitAmountCents: 100,
		Currency:        "USD",
	}

	db.Create(&subscriptiondomain.SubscriptionEntitlement{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		ProductID:      productID,
		FeatureCode:    "metered_feature",
		MeterID:        &meterID,
		EffectiveFrom:  subStart,
	})

	// Usage events: some BEFORE subscription start, some AFTER
	db.Create(&usagedomain.UsageEvent{
		ID:             node.Generate(),
		OrgID:          orgID,
		MeterID:        meterID,
		SubscriptionID: subID,
		Value:          10.0, // BEFORE sub start - should be IGNORED
		RecordedAt:     time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		Status:         usagedomain.UsageStatusEnriched,
	})
	db.Create(&usagedomain.UsageEvent{
		ID:             node.Generate(),
		OrgID:          orgID,
		MeterID:        meterID,
		SubscriptionID: subID,
		Value:          5.0, // AFTER sub start - should be INCLUDED
		RecordedAt:     time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
		Status:         usagedomain.UsageStatusEnriched,
	})

	err := svc.RunRating(context.Background(), cycleID.String())
	require.NoError(t, err)

	var results []ratingdomain.RatingResult
	db.Where("billing_cycle_id = ?", cycleID).Find(&results)
	require.Len(t, results, 1)

	result := results[0]
	// CRITICAL: Only usage AFTER subscription start counted
	assert.Equal(t, 5.0, result.Quantity, "Usage before subscription start MUST be excluded")
	assert.Equal(t, subStart, result.PeriodStart)
	assert.Equal(t, cycleEnd, result.PeriodEnd)
}

// Helper functions

func setupProrationTest(t *testing.T) (*gorm.DB, ratingdomain.Service, *snowflake.Node) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&ratingdomain.RatingResult{},
		&subscriptiondomain.SubscriptionItem{},
		&subscriptiondomain.Subscription{},
		&subscriptiondomain.SubscriptionEntitlement{},
		&billingcycledomain.BillingCycle{},
		&pricedomain.Price{},
		&usagedomain.UsageEvent{},
	)
	require.NoError(t, err)

	node, _ := snowflake.NewNode(1)
	logger := zap.NewNop()

	priceAmountStub := &priceAmountStub{
		Amounts: make(map[string]priceamountdomain.PriceAmount),
	}

	priceRepoStub := &priceRepoStub{
		Prices: make(map[string]pricedomain.Price),
	}

	svc := &Service{
		db:              db,
		log:             logger,
		genID:           node,
		priceAmountRepo: priceAmountStub,
		priceRepo:       priceRepoStub,
		ratingrepo:      nil, // Not used in tests
	}

	return db, svc, node
}

// Price repo stub
type priceRepoStub struct {
	Prices map[string]pricedomain.Price
}

func (s *priceRepoStub) WithTrx(tx *gorm.DB) repository.Repository[pricedomain.Price] {
	return s
}

func (s *priceRepoStub) Find(ctx context.Context, query *pricedomain.Price, opts ...option.QueryOption) ([]*pricedomain.Price, error) {
	return nil, nil
}

func (s *priceRepoStub) FindOne(ctx context.Context, query *pricedomain.Price, opts ...option.QueryOption) (*pricedomain.Price, error) {
	if p, ok := s.Prices[query.ID.String()]; ok {
		return &p, nil
	}
	return nil, nil
}

func (s *priceRepoStub) Create(ctx context.Context, entity *pricedomain.Price) error {
	return nil
}

func (s *priceRepoStub) Update(ctx context.Context, resourceID string, resource any) error {
	return nil
}

func (s *priceRepoStub) Delete(ctx context.Context, resourceID string) error {
	return nil
}

func (s *priceRepoStub) BatchCreate(ctx context.Context, entities []*pricedomain.Price) error {
	return nil
}

func (s *priceRepoStub) BatchUpdate(ctx context.Context, entities []*pricedomain.Price) error {
	return nil
}

func (s *priceRepoStub) Count(ctx context.Context, query *pricedomain.Price) (int64, error) {
	return 0, nil
}

// Price amount stub (existing)

func seedProrationData(
	t *testing.T,
	db *gorm.DB,
	node *snowflake.Node,
	priceAmountStub *priceAmountStub,
	priceRepoStub *priceRepoStub,
	orgID, subID, cycleID, productID, priceID snowflake.ID,
	cycleStart, cycleEnd, subStart time.Time,
	subEnd *time.Time,
	unitAmountCents int64,
) {
	db.Create(&billingcycledomain.BillingCycle{
		ID:             cycleID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PeriodStart:    cycleStart,
		PeriodEnd:      cycleEnd,
		Status:         billingcycledomain.BillingCycleStatusClosing,
	})

	sub := &subscriptiondomain.Subscription{
		ID:         subID,
		OrgID:      orgID,
		CustomerID: node.Generate(),
		Status:     subscriptiondomain.SubscriptionStatusActive,
		StartAt:    subStart,
	}
	if subEnd != nil {
		sub.EndedAt = subEnd
	}
	db.Create(sub)

	db.Create(&subscriptiondomain.SubscriptionItem{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		PriceID:        priceID,
		BillingMode:    "FLAT",
	})

	db.Create(&pricedomain.Price{
		ID:        priceID,
		OrgID:     orgID,
		ProductID: productID,
		Code:      "test_price",
		Active:    true,
	})

	// Configure price repo stub
	priceRepoStub.Prices[priceID.String()] = pricedomain.Price{
		ID:        priceID,
		ProductID: productID,
	}

	// Configure price amount stub
	priceAmountStub.Amounts[priceID.String()] = priceamountdomain.PriceAmount{
		PriceID:         priceID,
		UnitAmountCents: unitAmountCents,
		Currency:        "USD",
	}

	db.Create(&subscriptiondomain.SubscriptionEntitlement{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		ProductID:      productID,
		FeatureCode:    "test_feature",
		EffectiveFrom:  cycleStart,
	})
}
