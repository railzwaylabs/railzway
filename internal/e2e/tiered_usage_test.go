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
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	orgdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	planrepo "github.com/railzwaylabs/railzway/internal/plan/repository"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	ratingrepo "github.com/railzwaylabs/railzway/internal/rating/repository"
	ratingservice "github.com/railzwaylabs/railzway/internal/rating/service"
	subdomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	usagerepo "github.com/railzwaylabs/railzway/internal/usage/repository"
)

func TestTieredUsageBoundary_Graduated(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	orgID := uuid.New()
	customerID := uuid.New()
	meterID := uuid.New()
	planID := uuid.New()
	priceID := uuid.New()
	amountID := uuid.New()
	subID := uuid.New()
	itemID := uuid.New()
	periodID := uuid.New()

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)

	org := orgdomain.Organization{
		ID:           orgID,
		Name:         "Tiered Org",
		Slug:         "e2e-tiered-" + orgID.String()[:8],
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
		Name:      "Tiered Customer",
		Email:     "tiered@example.com",
		Currency:  "USD",
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}

	meter := usagedomain.Meter{
		ID:         meterID,
		OrgID:      orgID,
		Code:       "tokens",
		Name:       "Tokens",
		Aggregation: "sum",
		Unit:       "tokens",
		Active:     true,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&meter).Error; err != nil {
		t.Fatalf("create meter: %v", err)
	}

	plan := plandomain.Plan{
		ID:        planID,
		OrgID:     orgID,
		Code:      "tiered",
		Name:      "Tiered",
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
		Code:                 "tiered_usage",
		Name:                 "Tiered Usage",
		PriceType:            plandomain.PriceTypeTiered,
		BillingInterval:      plandomain.BillingIntervalMonth,
		BillingIntervalCount: 1,
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
		EffectiveFrom:   periodStart.AddDate(0, -1, 0),
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&amount).Error; err != nil {
		t.Fatalf("create plan amount: %v", err)
	}

	tier1Unit := int64(100)
	tier2Unit := int64(80)
	tier1End := float64(10)
	tier1 := plandomain.PlanTier{
		ID:             uuid.New(),
		OrgID:          orgID,
		PlanPriceID:    priceID,
		TierMode:       plandomain.TierModeGraduated,
		StartQuantity:  0,
		EndQuantity:    &tier1End,
		UnitAmountCents: &tier1Unit,
		Unit:           "tokens",
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	tier2 := plandomain.PlanTier{
		ID:              uuid.New(),
		OrgID:           orgID,
		PlanPriceID:     priceID,
		TierMode:        plandomain.TierModeGraduated,
		StartQuantity:   10,
		UnitAmountCents: &tier2Unit,
		Unit:            "tokens",
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&tier1).Error; err != nil {
		t.Fatalf("create tier1: %v", err)
	}
	if err := tx.Create(&tier2).Error; err != nil {
		t.Fatalf("create tier2: %v", err)
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
		t.Fatalf("create subscription period: %v", err)
	}

	event1 := usagedomain.UsageEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "tokens",
		CustomerID: customerID,
		Value:      8,
		RecordedAt: periodStart.Add(1 * time.Hour),
		Status:     usagedomain.StatusAccepted,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	event2 := usagedomain.UsageEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "tokens",
		CustomerID: customerID,
		Value:      7,
		RecordedAt: periodStart.Add(2 * time.Hour),
		Status:     usagedomain.StatusAccepted,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&event1).Error; err != nil {
		t.Fatalf("create event1: %v", err)
	}
	if err := tx.Create(&event2).Error; err != nil {
		t.Fatalf("create event2: %v", err)
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

	if _, err := ratingSvc.RateUsage(ctx, ratingdomain.RateUsageRequest{UsageEventID: event1.ID.String()}); err != nil {
		t.Fatalf("rate event1: %v", err)
	}
	if _, err := ratingSvc.RateUsage(ctx, ratingdomain.RateUsageRequest{UsageEventID: event2.ID.String()}); err != nil {
		t.Fatalf("rate event2: %v", err)
	}

	ratingRepo := ratingrepo.NewRepository(tx)
	r1, err := ratingRepo.FindRatingByUsageEvent(ctx, orgID, event1.ID)
	if err != nil || r1 == nil {
		t.Fatalf("find rating1: %v", err)
	}
	r2, err := ratingRepo.FindRatingByUsageEvent(ctx, orgID, event2.ID)
	if err != nil || r2 == nil {
		t.Fatalf("find rating2: %v", err)
	}
	if r1.AmountCents != 800 {
		t.Fatalf("expected first rating amount 800, got %d", r1.AmountCents)
	}
	if r2.AmountCents != 600 {
		t.Fatalf("expected second rating amount 600, got %d", r2.AmountCents)
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
		IdempotencyKey: "e2e-tiered-usage",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 invoice item, got %d", len(resp.Items))
	}
	if resp.Items[0].AmountCents != 1400 {
		t.Fatalf("expected invoice usage amount 1400, got %d", resp.Items[0].AmountCents)
	}
}
