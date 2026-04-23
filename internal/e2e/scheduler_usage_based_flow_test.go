//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	apikeydomain "github.com/railzwaylabs/railzway/internal/apikey/domain"
	apikeyrepo "github.com/railzwaylabs/railzway/internal/apikey/repository"
	apikeyservice "github.com/railzwaylabs/railzway/internal/apikey/service"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	customerrepo "github.com/railzwaylabs/railzway/internal/customer/repository"
	customerservice "github.com/railzwaylabs/railzway/internal/customer/service"
	orgdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	planrepo "github.com/railzwaylabs/railzway/internal/plan/repository"
	planservice "github.com/railzwaylabs/railzway/internal/plan/service"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	productrepo "github.com/railzwaylabs/railzway/internal/product/repository"
	productservice "github.com/railzwaylabs/railzway/internal/product/service"
	subdomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	subrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
	subservice "github.com/railzwaylabs/railzway/internal/subscription/service"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
	testclockrepo "github.com/railzwaylabs/railzway/internal/testclock/repository"
	testclockservice "github.com/railzwaylabs/railzway/internal/testclock/service"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	usagerepo "github.com/railzwaylabs/railzway/internal/usage/repository"
	usageservice "github.com/railzwaylabs/railzway/internal/usage/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBillingFlow_UsageBasedSchedulerScenario(t *testing.T) {
	db, keepData := openE2EScenarioDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	periodEnd := periodStart.AddDate(0, 1, 0)
	testClockTime := periodEnd.Add(24 * time.Hour)
	recordedAt := periodStart.AddDate(0, 0, 14).Add(10 * time.Hour)
	suffix := uuid.New().String()[:8]

	orgID := uuid.New()
	org := orgdomain.Organization{
		ID:           orgID,
		Name:         "Scheduler E2E Org " + suffix,
		Slug:         "e2e-scheduler-" + suffix,
		SupportEmail: "e2e-scheduler-" + suffix + "@example.com",
		CountryCode:  "US",
		TimezoneName: "UTC",
		Metadata:     json.RawMessage(`{}`),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&org).Error; err != nil {
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
	if err := db.Create(&prefs).Error; err != nil {
		t.Fatalf("create billing preferences: %v", err)
	}

	ctx := orgcontext.WithOrgID(context.Background(), orgID)
	apiKeySvc := apikeyservice.NewService(apikeyrepo.NewRepository(db))
	testClockRepo := testclockrepo.NewRepository(db)
	testClockSvc := testclockservice.NewService(testclockservice.Params{DB: db, Repo: testClockRepo})
	customerRepo := customerrepo.NewRepository(db)
	customerSvc := customerservice.NewService(customerservice.Params{DB: db, Repo: customerRepo, Clocks: testClockRepo})
	usageSvc := usageservice.NewService(usageservice.Params{DB: db, Repo: usagerepo.NewRepository(db)})
	planSvc := planservice.NewService(planservice.Params{DB: db, Repo: planrepo.NewRepository(db)})
	productSvc := productservice.NewService(productservice.Params{
		DB:    db,
		Repo:  productrepo.NewRepository(db),
		Plans: planSvc,
	})
	subSvc := subservice.NewService(subservice.Params{
		DB:        db,
		Repo:      subrepo.NewRepository(db),
		Customers: customerRepo,
		Clocks:    testClockRepo,
	})

	apiKey, err := apiKeySvc.CreateKey(context.Background(), orgID.String(), apikeydomain.CreateAPIKeyRequest{
		Name:    "Scheduler E2E key " + suffix,
		KeyType: string(apikeydomain.KeyTypeSecret),
		Scopes: []string{
			"usage_events:write",
			"customers:write",
			"subscriptions:write",
			"invoices:read",
		},
	})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}
	if _, err := apiKeySvc.AuthorizeKey(context.Background(), apiKey.Key, apikeydomain.AuthorizeAPIKeyRequest{
		Resource: "usage_events",
		Action:   "POST",
	}); err != nil {
		t.Fatalf("authorize api key: %v", err)
	}

	testClock, err := testClockSvc.Upsert(ctx, testclockdomain.UpsertTestClockRequest{
		Name:       "Scheduler E2E clock " + suffix,
		Status:     testclockdomain.StatusActive,
		FrozenTime: testClockTime,
	})
	if err != nil {
		t.Fatalf("create test clock: %v", err)
	}
	testClockID := testClock.ID.String()

	customer, err := customerSvc.Create(ctx, customerdomain.CreateCustomerRequest{
		Name:           "Scheduler E2E Customer " + suffix,
		Email:          "scheduler-" + suffix + "@example.com",
		ExternalID:     "scheduler-" + suffix,
		Currency:       "USD",
		TestClockID:    &testClockID,
		IdempotencyKey: "e2e-scheduler-customer-" + suffix,
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	active := true
	meter, err := usageSvc.CreateMeter(ctx, usagedomain.CreateMeterRequest{
		Code:           "api_calls_" + suffix,
		Name:           "API Calls " + suffix,
		Aggregation:    usagedomain.AggregationSum,
		Unit:           "request",
		Active:         &active,
		IdempotencyKey: "e2e-scheduler-meter-" + suffix,
	})
	if err != nil {
		t.Fatalf("create meter: %v", err)
	}

	effectiveFrom := periodStart.AddDate(0, -1, 0)
	description := "Usage-based scheduler scenario"
	aggregateUsage := usagedomain.AggregationSum
	billingUnit := "request"
	product, err := productSvc.Create(ctx, productdomain.CreateProductRequest{
		Code:           "scheduler_usage_" + suffix,
		Name:           "Scheduler Usage Product " + suffix,
		Description:    &description,
		Active:         &active,
		IdempotencyKey: "e2e-scheduler-product-" + suffix,
		Plans: []productdomain.CreateProductPlanInput{
			{
				Code:   "scheduler_usage_plan_" + suffix,
				Name:   "Scheduler Usage Plan " + suffix,
				Active: &active,
				Prices: []productdomain.CreateProductPlanPriceInput{
					{
						Code:                 "api_calls_usage_" + suffix,
						Name:                 "API Calls Usage " + suffix,
						PriceType:            plandomain.PriceTypeUsage,
						BillingInterval:      plandomain.BillingIntervalMonth,
						BillingIntervalCount: 1,
						AggregateUsage:       &aggregateUsage,
						BillingUnit:          &billingUnit,
						MeterID:              &meter.ID,
						Active:               &active,
						Amounts: []productdomain.CreateProductPlanAmountInput{
							{
								Currency:        "USD",
								UnitAmountCents: 4.749975,
								EffectiveFrom:   &effectiveFrom,
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if len(product.Plans) != 1 || len(product.Plans[0].Prices) != 1 {
		t.Fatalf("expected product to include one plan with one price, got plans=%d", len(product.Plans))
	}
	plan := product.Plans[0]
	price := plan.Prices[0]

	subscription, err := subSvc.CreateSubscription(ctx, subdomain.CreateSubscriptionRequest{
		CustomerID:         customer.ID,
		PlanID:             plan.ID,
		Currency:           "USD",
		StartAt:            &periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		Status:             subdomain.StatusActive,
		IdempotencyKey:     "e2e-scheduler-subscription-" + suffix,
		Items: []subdomain.CreateSubscriptionItemInput{
			{
				PlanPriceID:    price.ID,
				Quantity:       1,
				StartAt:        &periodStart,
				IdempotencyKey: "e2e-scheduler-subscription-item-" + suffix,
			},
		},
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	usage, err := usageSvc.IngestUsage(ctx, usagedomain.IngestUsageRequest{
		MeterCode:      meter.Code,
		CustomerID:     customer.ID,
		Value:          12345,
		RecordedAt:     recordedAt,
		IdempotencyKey: "e2e-scheduler-usage-" + suffix,
	})
	if err != nil {
		t.Fatalf("ingest usage: %v", err)
	}
	if usage.Status != usagedomain.StatusAccepted {
		t.Fatalf("expected usage status accepted, got %s", usage.Status)
	}

	t.Logf("scheduler scenario keep_data=%t", keepData)
	t.Logf("org id=%s slug=%s", orgID, org.Slug)
	t.Logf("api key id=%s prefix=%s raw_key=%s", apiKey.ID, apiKey.KeyPrefix, apiKey.Key)
	t.Logf("test clock id=%s current_time=%s", testClock.ID, testClock.CurrentTime.Format(time.RFC3339))
	t.Logf("customer id=%s test_clock_id=%s", customer.ID, testClockID)
	t.Logf("meter id=%s code=%s", meter.ID, meter.Code)
	t.Logf("product id=%s plan_id=%s price_id=%s unit_amount_cents=%.8f", product.ID, plan.ID, price.ID, price.Amounts[0].UnitAmountCents)
	t.Logf("subscription id=%s period_start=%s period_end=%s", subscription.ID, periodStart.Format(time.RFC3339), periodEnd.Format(time.RFC3339))
	t.Logf("usage event id=%s status=%s value=%.2f recorded_at=%s", usage.ID, usage.Status, usage.Value, usage.RecordedAt.Format(time.RFC3339))
	if keepData {
		t.Log("next: run `RATING_JOB_INTERVAL_SEC=1 go run ./cmd/scheduler rating`, wait for usage to become rated, then stop it")
		t.Log("next: run `SUBSCRIPTION_CLOSE_PERIOD_INTERVAL_SEC=1 go run ./cmd/scheduler close-period`, wait for invoice/ledger creation, then stop it")
	}
}

func openE2EScenarioDB(t *testing.T) (*gorm.DB, bool) {
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

	keepData := os.Getenv("E2E_KEEP_DATA") == "1"
	if keepData {
		return db, true
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin tx: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	return tx, false
}
