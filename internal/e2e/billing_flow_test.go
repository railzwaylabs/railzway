//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
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
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	ratingrepo "github.com/railzwaylabs/railzway/internal/rating/repository"
	ratingservice "github.com/railzwaylabs/railzway/internal/rating/service"
	subdomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	usagerepo "github.com/railzwaylabs/railzway/internal/usage/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBillingFlow_RateThenInvoice(t *testing.T) {
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
	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("db ping: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin tx: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })

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
	usageID := uuid.New()

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
		ID:        meterID,
		OrgID:     orgID,
		Code:      "requests",
		Name:      "Requests",
		Aggregation: "sum",
		Unit:      "requests",
		Active:    true,
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.Create(&meter).Error; err != nil {
		t.Fatalf("create meter: %v", err)
	}

	plan := plandomain.Plan{
		ID:        planID,
		OrgID:     orgID,
		Code:      "basic",
		Name:      "Basic",
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
		UnitAmountCents: 100,
		EffectiveFrom:   now.AddDate(0, -1, 0),
		Metadata:        json.RawMessage(`{}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&amount).Error; err != nil {
		t.Fatalf("create plan amount: %v", err)
	}

	periodStart := now.Add(-24 * time.Hour)
	periodEnd := periodStart.AddDate(0, 1, 0)
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

	usage := usagedomain.UsageEvent{
		ID:         usageID,
		OrgID:      orgID,
		MeterID:    meterID,
		MeterCode:  "requests",
		CustomerID: customerID,
		Value:      10,
		RecordedAt: periodStart.Add(1 * time.Hour),
		Status:     usagedomain.StatusAccepted,
		Metadata:   json.RawMessage(`{}`),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Create(&usage).Error; err != nil {
		t.Fatalf("create usage event: %v", err)
	}

	planRepo := planrepo.NewRepository(tx)
	subRepo := subrepo.NewRepository(tx)
	usageRepo := usagerepo.NewRepository(tx)
	ratingRepo := ratingrepo.NewRepository(tx)
	customerRepo := customerrepo.NewRepository(tx)
	invoiceRepo := invoicerepo.NewRepository(tx)

	ratingSvc := ratingservice.NewService(ratingservice.Params{
		DB:               tx,
		RatingRepo:       ratingRepo,
		UsageRepo:        usageRepo,
		SubscriptionRepo: subRepo,
		PlanRepo:         planRepo,
		CustomerRepo:     customerRepo,
	})
	invoiceSvc := invoiceservice.NewService(invoiceservice.Params{
		DB:       tx,
		Repo:     invoiceRepo,
		PlanRepo: planRepo,
		SubRepo:  subRepo,
	})

	ctx := orgcontext.WithOrgID(context.Background(), orgID)

	_, err = invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	})
	if !errors.Is(err, invoicedomain.ErrUsageNotReady) {
		t.Fatalf("expected ErrUsageNotReady, got %v", err)
	}

	_, err = ratingSvc.RateUsage(ctx, ratingdomain.RateUsageRequest{UsageEventID: usageID.String()})
	if err != nil {
		t.Fatalf("rate usage: %v", err)
	}

	resp, err := invoiceSvc.GenerateInvoice(ctx, invoicedomain.GenerateInvoiceRequest{
		SubscriptionID: subID.String(),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IdempotencyKey: "e2e-" + subID.String(),
	})
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if resp.Invoice.ID == uuid.Nil {
		t.Fatalf("expected invoice id, got nil")
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 invoice item, got %d", len(resp.Items))
	}
	if resp.Items[0].AmountCents != 1000 {
		t.Fatalf("expected amount 1000, got %d", resp.Items[0].AmountCents)
	}
}
