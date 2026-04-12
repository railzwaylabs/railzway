//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	invoicerepo "github.com/railzwaylabs/railzway/internal/invoice/repository"
	invoiceservice "github.com/railzwaylabs/railzway/internal/invoice/service"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	orgdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	planrepo "github.com/railzwaylabs/railzway/internal/plan/repository"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	subdomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"gorm.io/gorm"
)

func TestInvoiceUsageCharges_ExcludeDuringTrial(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	trialEnd := periodStart.Add(10 * 24 * time.Hour)

	orgID, subID, priceID, meterID, customerID := seedUsageSubscription(t, tx, now, periodStart, periodEnd, trialEnd)

	usageBeforeAt := periodStart.Add(5 * 24 * time.Hour)
	usageAfterAt := trialEnd.Add(2 * 24 * time.Hour)

	beforeUsageID := uuid.New()
	afterUsageID := uuid.New()

	beforeUsage := usagedomain.UsageEvent{
		ID:         beforeUsageID,
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "api_calls",
		CustomerID: customerID,
		Value:      5,
		RecordedAt: usageBeforeAt,
		Status:     usagedomain.StatusRated,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&beforeUsage).Error; err != nil {
		t.Fatalf("create usage before trial: %v", err)
	}

	afterUsage := usagedomain.UsageEvent{
		ID:         afterUsageID,
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "api_calls",
		CustomerID: customerID,
		Value:      7,
		RecordedAt: usageAfterAt,
		Status:     usagedomain.StatusRated,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&afterUsage).Error; err != nil {
		t.Fatalf("create usage after trial: %v", err)
	}

	beforeRating := ratingdomain.RatingResult{
		ID:              uuid.New(),
		OrgID:           orgID,
		UsageEventID:    beforeUsageID,
		CustomerID:      customerID,
		SubscriptionID:  &subID,
		PlanPriceID:     priceID,
		MeterID:         meterID,
		Currency:        "USD",
		Quantity:        5,
		UnitAmountCents: 100,
		AmountCents:     500,
		Source:          ratingdomain.SourceUsage,
		WindowStart:     usageBeforeAt,
		WindowEnd:       usageBeforeAt,
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
	}
	if err := tx.Create(&beforeRating).Error; err != nil {
		t.Fatalf("create rating before trial: %v", err)
	}

	afterRating := ratingdomain.RatingResult{
		ID:              uuid.New(),
		OrgID:           orgID,
		UsageEventID:    afterUsageID,
		CustomerID:      customerID,
		SubscriptionID:  &subID,
		PlanPriceID:     priceID,
		MeterID:         meterID,
		Currency:        "USD",
		Quantity:        7,
		UnitAmountCents: 100,
		AmountCents:     700,
		Source:          ratingdomain.SourceUsage,
		WindowStart:     usageAfterAt,
		WindowEnd:       usageAfterAt,
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
	}
	if err := tx.Create(&afterRating).Error; err != nil {
		t.Fatalf("create rating after trial: %v", err)
	}

	invoiceSvc := invoiceservice.NewService(invoiceservice.Params{
		DB:       tx,
		Repo:     invoicerepo.NewRepository(tx),
		PlanRepo: planrepo.NewRepository(tx),
		SubRepo:  subrepo.NewRepository(tx),
	})

	ctx := orgcontext.WithOrgID(context.Background(), orgID)
	resp, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IdempotencyKey: "e2e-trial-usage-excluded",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 invoice item, got %d", len(resp.Items))
	}
	if resp.Items[0].AmountCents != 700 {
		t.Fatalf("expected usage amount 700, got %d", resp.Items[0].AmountCents)
	}
	if resp.Invoice.SubtotalCents != 700 {
		t.Fatalf("expected subtotal 700, got %d", resp.Invoice.SubtotalCents)
	}
}

func seedUsageSubscription(
	t *testing.T,
	tx *gorm.DB,
	now time.Time,
	periodStart time.Time,
	periodEnd time.Time,
	trialEnd time.Time,
) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	orgID := uuid.New()
	customerID := uuid.New()
	planID := uuid.New()
	priceID := uuid.New()
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
		Code:      "usage-plan",
		Name:      "Usage Plan",
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
		Code:                 "usage",
		Name:                 "Usage",
		PriceType:            plandomain.PriceTypeUsage,
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

	sub := subdomain.Subscription{
		ID:                 subID,
		OrgID:              orgID,
		CustomerID:         customerID,
		PlanID:             planID,
		Status:             subdomain.StatusTrialing,
		Currency:           "USD",
		StartAt:            periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		TrialEnd:           &trialEnd,
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
