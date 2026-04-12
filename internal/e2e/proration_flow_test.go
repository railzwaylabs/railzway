//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
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
	subdomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestInvoiceProration_ItemStartsMidPeriod(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	itemStart := periodStart.Add(10 * 24 * time.Hour)

	orgID, subID := seedFlatSubscription(t, tx, now, periodStart, periodEnd, itemStart, nil)

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
		IdempotencyKey: "e2e-proration-item-start",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 invoice item, got %d", len(resp.Items))
	}

	expected := proratedAmount(10000, itemStart, periodEnd, periodStart, periodEnd)
	if resp.Items[0].AmountCents != expected {
		t.Fatalf("expected amount %d, got %d", expected, resp.Items[0].AmountCents)
	}
}

func TestInvoiceProration_SubscriptionCanceledMidPeriod(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	cancelAt := periodStart.Add(12 * 24 * time.Hour)

	orgID, subID := seedFlatSubscription(t, tx, now, periodStart, periodEnd, periodStart, &cancelAt)

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
		IdempotencyKey: "e2e-proration-cancel",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 invoice item, got %d", len(resp.Items))
	}

	expected := proratedAmount(10000, periodStart, cancelAt, periodStart, periodEnd)
	if resp.Items[0].AmountCents != expected {
		t.Fatalf("expected amount %d, got %d", expected, resp.Items[0].AmountCents)
	}
}

func TestInvoiceProration_PriceChangeMidPeriod(t *testing.T) {
	tx := openE2ETx(t)
	now := time.Now().UTC().Truncate(time.Second)

	periodStart := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	switchAt := periodStart.Add(15 * 24 * time.Hour)

	orgID, subID := seedFlatSubscription(t, tx, now, periodStart, periodEnd, periodStart, nil)

	var priceID uuid.UUID
	if err := tx.Raw("SELECT id FROM plan_prices WHERE org_id = ? LIMIT 1", orgID).Scan(&priceID).Error; err != nil {
		t.Fatalf("find plan price: %v", err)
	}

	if err := tx.Exec("DELETE FROM plan_amounts WHERE org_id = ? AND plan_price_id = ?", orgID, priceID).Error; err != nil {
		t.Fatalf("reset amounts: %v", err)
	}

	first := plandomain.PlanAmount{
		ID:              uuid.New(),
		OrgID:           orgID,
		PlanPriceID:     priceID,
		Currency:        "USD",
		UnitAmountCents: 10000,
		EffectiveFrom:   periodStart,
		EffectiveTo:     &switchAt,
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	second := plandomain.PlanAmount{
		ID:              uuid.New(),
		OrgID:           orgID,
		PlanPriceID:     priceID,
		Currency:        "USD",
		UnitAmountCents: 20000,
		EffectiveFrom:   switchAt,
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&first).Error; err != nil {
		t.Fatalf("create first amount: %v", err)
	}
	if err := tx.Create(&second).Error; err != nil {
		t.Fatalf("create second amount: %v", err)
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
		IdempotencyKey: "e2e-proration-price-change",
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 invoice items, got %d", len(resp.Items))
	}

	firstExpected := proratedAmount(10000, periodStart, switchAt, periodStart, periodEnd)
	secondExpected := proratedAmount(20000, switchAt, periodEnd, periodStart, periodEnd)
	expectedTotal := firstExpected + secondExpected
	if resp.Invoice.SubtotalCents != expectedTotal {
		t.Fatalf("expected subtotal %d, got %d", expectedTotal, resp.Invoice.SubtotalCents)
	}
}

func openE2ETx(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("E2E_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("E2E_DATABASE_URL or DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin tx: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	return tx
}

func seedFlatSubscription(
	t *testing.T,
	tx *gorm.DB,
	now time.Time,
	periodStart time.Time,
	periodEnd time.Time,
	itemStart time.Time,
	cancelAt *time.Time,
) (uuid.UUID, uuid.UUID) {
	t.Helper()

	orgID := uuid.New()
	customerID := uuid.New()
	planID := uuid.New()
	priceID := uuid.New()
	amountID := uuid.New()
	subID := uuid.New()
	itemID := uuid.New()
	periodID := uuid.New()

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

	plan := plandomain.Plan{
		ID:        planID,
		OrgID:     orgID,
		Code:      "flat",
		Name:      "Flat Plan",
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
		Code:                 "flat",
		Name:                 "Flat Fee",
		PriceType:            plandomain.PriceTypeFlat,
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
		UnitAmountCents: 10000,
		EffectiveFrom:   periodStart.AddDate(0, -1, 0),
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&amount).Error; err != nil {
		t.Fatalf("create plan amount: %v", err)
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
		CancelAt:           cancelAt,
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
		StartAt:        itemStart,
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

	_ = customerrepo.NewRepository(tx)
	return orgID, subID
}

func proratedAmount(unitAmount int64, activeStart, activeEnd, periodStart, periodEnd time.Time) int64 {
	periodSeconds := periodEnd.Sub(periodStart).Seconds()
	if periodSeconds <= 0 {
		return 0
	}
	activeSeconds := activeEnd.Sub(activeStart).Seconds()
	if activeSeconds <= 0 {
		return 0
	}
	factor := activeSeconds / periodSeconds
	if factor > 1 {
		factor = 1
	}
	if factor < 0 {
		factor = 0
	}
	raw := float64(unitAmount) * factor
	return int64(raw + 0.5)
}
