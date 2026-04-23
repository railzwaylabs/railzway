//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	customerrepo "github.com/railzwaylabs/railzway/internal/customer/repository"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	invoicerepo "github.com/railzwaylabs/railzway/internal/invoice/repository"
	invoiceservice "github.com/railzwaylabs/railzway/internal/invoice/service"
	orgdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	planrepo "github.com/railzwaylabs/railzway/internal/plan/repository"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	ratingrepo "github.com/railzwaylabs/railzway/internal/rating/repository"
	ratingservice "github.com/railzwaylabs/railzway/internal/rating/service"
	subdomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	usagerepo "github.com/railzwaylabs/railzway/internal/usage/repository"
	"gorm.io/gorm"
)

func TestUsageTieredCharges_CrossTierBoundary(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	orgID, subID, priceID, meterID, customerID := seedTieredUsageSubscription(t, tx, now, periodStart, periodEnd)

	firstUsage := usagedomain.UsageEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "api_calls",
		CustomerID: customerID,
		Value:      5,
		RecordedAt: periodStart.Add(2 * 24 * time.Hour),
		Status:     usagedomain.StatusAccepted,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&firstUsage).Error; err != nil {
		t.Fatalf("create first usage: %v", err)
	}
	secondUsage := usagedomain.UsageEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "api_calls",
		CustomerID: customerID,
		Value:      10,
		RecordedAt: periodStart.Add(10 * 24 * time.Hour),
		Status:     usagedomain.StatusAccepted,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&secondUsage).Error; err != nil {
		t.Fatalf("create second usage: %v", err)
	}

	ratingSvc := ratingservice.NewService(ratingservice.Params{
		DB:               tx,
		RatingRepo:       ratingrepo.NewRepository(tx),
		UsageRepo:        usagerepo.NewRepository(tx),
		SubscriptionRepo: subrepo.NewRepository(tx),
		PlanRepo:         planrepo.NewRepository(tx),
		CustomerRepo:     customerrepo.NewRepository(tx),
	})

	ctx := orgcontext.WithOrgID(context.Background(), orgID)
	if _, err := ratingSvc.RateUsage(ctx, ratingdomain.RateUsageRequest{UsageEventID: firstUsage.ID.String()}); err != nil {
		t.Fatalf("rate first usage: %v", err)
	}
	if _, err := ratingSvc.RateUsage(ctx, ratingdomain.RateUsageRequest{UsageEventID: secondUsage.ID.String()}); err != nil {
		t.Fatalf("rate second usage: %v", err)
	}

	invoiceSvc := invoiceservice.NewService(invoiceservice.Params{
		DB:       tx,
		Repo:     invoicerepo.NewRepository(tx),
		PlanRepo: planrepo.NewRepository(tx),
		SubRepo:  subrepo.NewRepository(tx),
	})
	resp, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IdempotencyKey: "e2e-usage-tier-boundary",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 invoice item, got %d", len(resp.Items))
	}

	expected := int64(2000) // 10*100 + 5*200 (graduated)
	if resp.Invoice.SubtotalCents != expected {
		t.Fatalf("expected subtotal %d, got %d", expected, resp.Invoice.SubtotalCents)
	}
	if resp.Items[0].AmountCents != expected {
		t.Fatalf("expected amount %d, got %d", expected, resp.Items[0].AmountCents)
	}
	if resp.Items[0].PlanPriceID == nil || *resp.Items[0].PlanPriceID != priceID {
		t.Fatalf("expected plan price %s", priceID.String())
	}
}

func seedTieredUsageSubscription(
	t *testing.T,
	tx *gorm.DB,
	now time.Time,
	periodStart time.Time,
	periodEnd time.Time,
) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	orgID := uuid.New()
	customerID := uuid.New()
	planID := uuid.New()
	priceID := uuid.New()
	amountID := uuid.New()
	subID := uuid.New()
	itemID := uuid.New()
	periodID := uuid.New()
	meterID := uuid.New()

	org := orgdomain.Organization{
		ID:           orgID,
		Name:         "E2E Org",
		Slug:         "e2e-" + orgID.String()[:8],
		SupportEmail: "e2e@example.com",
		CountryCode:  "US",
		TimezoneName: "UTC",
		Metadata:     json.RawMessage(`{}`),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := tx.Create(&org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}

	prefs := orgdomain.OrganizationBillingPreferences{
		OrgID:                orgID,
		Currency:             "USD",
		Timezone:             "UTC",
		InvoicePrefix:        "INV",
		InvoiceNumberFormat:  "{PREFIX}-{YYYY}{MM}-{SEQ:6}",
		InvoiceSequenceScope: "org_month",
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := tx.Create(&prefs).Error; err != nil {
		t.Fatalf("create billing prefs: %v", err)
	}

	customer := customerdomain.Customer{
		ID:        customerID,
		OrgID:     orgID,
		Name:      "E2E Customer",
		Email:     "customer@example.com",
		Currency:  "USD",
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	meter := usagedomain.Meter{
		ID:          meterID,
		OrgID:       orgID,
		Code:        "api_calls",
		Name:        "API Calls",
		Aggregation: "sum",
		Unit:        "call",
		Active:      true,
		Metadata:    json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tx.Create(&meter).Error; err != nil {
		t.Fatalf("create meter: %v", err)
	}

	plan := plandomain.Plan{
		ID:        planID,
		OrgID:     orgID,
		Code:      "tier-plan",
		Name:      "Tier Plan",
		Active:    true,
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&plan).Error; err != nil {
		t.Fatalf("create plan: %v", err)
	}

	price := plandomain.PlanPrice{
		ID:                   priceID,
		OrgID:                orgID,
		PlanID:               planID,
		MeterID:              &meterID,
		Code:                 "tiered",
		Name:                 "Tiered Usage",
		PriceType:            plandomain.PriceTypeTiered,
		BillingInterval:      "month",
		BillingIntervalCount: 1,
		AggregateUsage:       "sum",
		BillingUnit:          "call",
		MeterCode:            "api_calls",
		Active:               true,
		Metadata:             json.RawMessage(`{}`),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := tx.Create(&price).Error; err != nil {
		t.Fatalf("create plan price: %v", err)
	}

	amount := plandomain.PlanAmount{
		ID:              amountID,
		OrgID:           orgID,
		PlanPriceID:     priceID,
		Currency:        "USD",
		UnitAmountCents: 0,
		EffectiveFrom:   periodStart,
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&amount).Error; err != nil {
		t.Fatalf("create plan amount: %v", err)
	}

	tier1End := float64(10)
	tier1Unit := float64(100)
	tier2Unit := float64(200)
	tiers := []plandomain.PlanTier{
		{
			ID:              uuid.New(),
			OrgID:           orgID,
			PlanPriceID:     priceID,
			TierMode:        plandomain.TierModeGraduated,
			StartQuantity:   0,
			EndQuantity:     &tier1End,
			UnitAmountCents: &tier1Unit,
			Unit:            "call",
			Metadata:        json.RawMessage(`{}`),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              uuid.New(),
			OrgID:           orgID,
			PlanPriceID:     priceID,
			TierMode:        plandomain.TierModeGraduated,
			StartQuantity:   tier1End,
			UnitAmountCents: &tier2Unit,
			Unit:            "call",
			Metadata:        json.RawMessage(`{}`),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
	for _, tier := range tiers {
		if err := tx.Create(&tier).Error; err != nil {
			t.Fatalf("create plan tier: %v", err)
		}
	}

	sub := subdomain.Subscription{
		ID:                 subID,
		OrgID:              orgID,
		CustomerID:         customerID,
		PlanID:             planID,
		Status:             subdomain.StatusActive,
		Currency:           "USD",
		StartAt:            periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Metadata:           json.RawMessage(`{}`),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := tx.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	period := subdomain.SubscriptionPeriod{
		ID:             periodID,
		OrgID:          orgID,
		SubscriptionID: subID,
		Status:         subdomain.PeriodStatusOpen,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := tx.Create(&period).Error; err != nil {
		t.Fatalf("create period: %v", err)
	}

	item := subdomain.SubscriptionItem{
		ID:             itemID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PlanPriceID:    priceID,
		Quantity:       1,
		StartAt:        periodStart,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := tx.Create(&item).Error; err != nil {
		t.Fatalf("create subscription item: %v", err)
	}

	return orgID, subID, priceID, meterID, customerID
}
