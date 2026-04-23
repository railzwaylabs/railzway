//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
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
)

func TestBillingFlow_RatingAndProrationInvoice(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	flatItemStart := periodStart.Add(10 * 24 * time.Hour)

	orgID := uuid.New()
	customerID := uuid.New()
	meterID := uuid.New()
	planID := uuid.New()
	flatPriceID := uuid.New()
	usagePriceID := uuid.New()
	subID := uuid.New()
	flatItemID := uuid.New()
	usageItemID := uuid.New()
	usageEventID := uuid.New()

	org := orgdomain.Organization{
		ID:           orgID,
		Name:         "Rating Proration Org",
		Slug:         "e2e-rating-proration-" + orgID.String()[:8],
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
		Name:      "Rating Proration Customer",
		Email:     "rating-proration@example.com",
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
		Code:      "hybrid",
		Name:      "Hybrid Plan",
		Active:    true,
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&plan).Error; err != nil {
		t.Fatalf("create plan: %v", err)
	}

	flatPrice := planPriceFixture(flatPriceID, orgID, planID, nil, "base", "Base Fee", plandomain.PriceTypeFlat, now)
	usagePrice := planPriceFixture(usagePriceID, orgID, planID, &meterID, "api_calls", "API Calls", plandomain.PriceTypeUsage, now)
	if err := tx.Create(&flatPrice).Error; err != nil {
		t.Fatalf("create flat price: %v", err)
	}

	if err := tx.Create(&usagePrice).Error; err != nil {
		t.Fatalf("create usage price: %v", err)
	}

	flatAmount := plandomain.PlanAmount{
		ID:              uuid.New(),
		OrgID:           orgID,
		PlanPriceID:     flatPriceID,
		Currency:        "USD",
		UnitAmountCents: 10000,
		EffectiveFrom:   periodStart.AddDate(0, -1, 0),
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	usageAmount := plandomain.PlanAmount{
		ID:              uuid.New(),
		OrgID:           orgID,
		PlanPriceID:     usagePriceID,
		Currency:        "USD",
		UnitAmountCents: 125,
		EffectiveFrom:   periodStart.AddDate(0, -1, 0),
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&flatAmount).Error; err != nil {
		t.Fatalf("create flat amount: %v", err)
	}

	if err := tx.Create(&usageAmount).Error; err != nil {
		t.Fatalf("create usage amount: %v", err)
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

	flatItem := subdomain.SubscriptionItem{
		ID:             flatItemID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PlanPriceID:    flatPriceID,
		Quantity:       1,
		StartAt:        flatItemStart,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	usageItem := subdomain.SubscriptionItem{
		ID:             usageItemID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PlanPriceID:    usagePriceID,
		Quantity:       1,
		StartAt:        periodStart,
		Metadata:       json.RawMessage(`{}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := tx.Create(&flatItem).Error; err != nil {
		t.Fatalf("create flat item: %v", err)
	}

	if err := tx.Create(&usageItem).Error; err != nil {
		t.Fatalf("create usage item: %v", err)
	}

	period := subdomain.SubscriptionPeriod{
		ID:             uuid.New(),
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

	usage := usagedomain.UsageEvent{
		ID:         usageEventID,
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "api_calls",
		CustomerID: customerID,
		Value:      7,
		RecordedAt: flatItemStart.Add(2 * time.Hour),
		Status:     usagedomain.StatusAccepted,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&usage).Error; err != nil {
		t.Fatalf("create usage event: %v", err)
	}

	ctx := orgcontext.WithOrgID(context.Background(), orgID)
	ratingSvc := ratingservice.NewService(ratingservice.Params{
		DB:               tx,
		RatingRepo:       ratingrepo.NewRepository(tx),
		UsageRepo:        usagerepo.NewRepository(tx),
		SubscriptionRepo: subrepo.NewRepository(tx),
		PlanRepo:         planrepo.NewRepository(tx),
		CustomerRepo:     customerrepo.NewRepository(tx),
	})
	invoiceSvc := invoiceservice.NewService(invoiceservice.Params{
		DB:       tx,
		Repo:     invoicerepo.NewRepository(tx),
		PlanRepo: planrepo.NewRepository(tx),
		SubRepo:  subrepo.NewRepository(tx),
	})

	_, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	})
	if !errors.Is(err, invoicedomain.ErrUsageNotReady) {
		t.Fatalf("expected ErrUsageNotReady before rating, got %v", err)
	}

	ratingResp, err := ratingSvc.RateUsage(ctx, ratingdomain.RateUsageRequest{UsageEventID: usageEventID.String()})
	if err != nil {
		t.Fatalf("rate usage: %v", err)
	}

	if ratingResp.RatingResult.AmountCents != 875 {
		t.Fatalf("expected rating amount 875, got %d", ratingResp.RatingResult.AmountCents)
	}

	if ratingResp.RatingResult.UnitAmountCents != 125 {
		t.Fatalf("expected rating unit amount 125, got %.6f", ratingResp.RatingResult.UnitAmountCents)
	}

	resp, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IdempotencyKey: "e2e-rating-proration",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 invoice items, got %d", len(resp.Items))
	}

	subscriptionLine, usageLine := findInvoiceLine(resp.Items, invoicedomain.LineTypeSubscription), findInvoiceLine(resp.Items, invoicedomain.LineTypeUsage)
	if subscriptionLine == nil {
		t.Fatalf("expected subscription invoice line")
	}
	if usageLine == nil {
		t.Fatalf("expected usage invoice line")
	}

	t.Logf(
		"invoice id=%s org_id=%s subscription_id=%s customer_id=%s subtotal_cents=%d total_cents=%d item_count=%d",
		resp.Invoice.ID,
		orgID,
		subID,
		customerID,
		resp.Invoice.SubtotalCents,
		resp.Invoice.TotalCents,
		len(resp.Items),
	)
	t.Logf(
		"subscription line id=%s amount_cents=%d unit_amount_cents=%.6f quantity=%.2f period_start=%s period_end=%s",
		subscriptionLine.ID,
		subscriptionLine.AmountCents,
		subscriptionLine.UnitAmountCents,
		subscriptionLine.Quantity,
		formatTimePtr(subscriptionLine.PeriodStart),
		formatTimePtr(subscriptionLine.PeriodEnd),
	)
	t.Logf(
		"usage line id=%s usage_event_id=%s rating_result_id=%s amount_cents=%d quantity=%.2f meter_id=%s plan_price_id=%s",
		usageLine.ID,
		usageEventID,
		ratingResp.RatingResult.ID,
		usageLine.AmountCents,
		usageLine.Quantity,
		formatUUIDPtr(usageLine.MeterID),
		formatUUIDPtr(usageLine.PlanPriceID),
	)

	expectedProratedFlat := int64(6667) // $100.00 * 20 active days / 30 billing days, rounded to cents.
	expectedRatedUsage := int64(875)    // 7 calls * $1.25.
	if subscriptionLine.AmountCents != expectedProratedFlat {
		t.Fatalf("expected prorated subscription amount %d, got %d", expectedProratedFlat, subscriptionLine.AmountCents)
	}
	if usageLine.AmountCents != expectedRatedUsage {
		t.Fatalf("expected usage amount %d, got %d", expectedRatedUsage, usageLine.AmountCents)
	}
	if resp.Invoice.SubtotalCents != expectedProratedFlat+expectedRatedUsage {
		t.Fatalf("expected subtotal %d, got %d", expectedProratedFlat+expectedRatedUsage, resp.Invoice.SubtotalCents)
	}
}

func planPriceFixture(id, orgID, planID uuid.UUID, meterID *uuid.UUID, code, name, priceType string, now time.Time) plandomain.PlanPrice {
	price := plandomain.PlanPrice{
		ID:                   id,
		OrgID:                orgID,
		PlanID:               planID,
		MeterID:              meterID,
		Code:                 code,
		Name:                 name,
		PriceType:            priceType,
		BillingInterval:      plandomain.BillingIntervalMonth,
		BillingIntervalCount: 1,
		Active:               true,
		Metadata:             json.RawMessage(`{}`),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if meterID != nil {
		price.AggregateUsage = "sum"
		price.BillingUnit = "call"
		price.MeterCode = code
	}
	return price
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func formatUUIDPtr(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func findInvoiceLine(items []invoicedomain.InvoiceItem, lineType string) *invoicedomain.InvoiceItem {
	for i := range items {
		if items[i].LineType == lineType {
			return &items[i]
		}
	}
	return nil
}
