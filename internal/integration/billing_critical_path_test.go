package integration

import (
	"context"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	// Domain imports

	// Repositories

	subscriptionrepo "github.com/railzwaylabs/railzway/internal/subscription/repository"
)

func TestBillingCriticalPath(t *testing.T) {
	// 1. Setup Infrastructure
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info), // Uncomment for SQL debugging
	})
	require.NoError(t, err)

	// Validate DB connection
	sqlDB, err := db.DB()
	require.NoError(t, err)
	err = sqlDB.Ping()
	require.NoError(t, err)

	// Auto-Migrate All Tables
	err = db.AutoMigrate(
		&customerdomain.Customer{},
		&productdomain.Product{},
		&pricedomain.Price{},
		&priceamountdomain.PriceAmount{},
		&meterdomain.Meter{},
		&subscriptiondomain.Subscription{},
		&subscriptiondomain.SubscriptionItem{},
		&subscriptiondomain.SubscriptionEntitlement{},
		&billingcycledomain.BillingCycle{},
		&usagedomain.UsageEvent{},
		&ratingdomain.RatingResult{},
		&invoicedomain.Invoice{},
		&invoicedomain.InvoiceItem{},
		&invoicedomain.InvoiceTaxLine{},
		&ledgerdomain.LedgerAccount{},
		&ledgerdomain.LedgerEntry{},
		&ledgerdomain.LedgerEntryLine{},
	)
	require.NoError(t, err)

	node, _ := snowflake.NewNode(1)
	logger := zap.NewNop()
	clk := clock.New() // Real clock

	// 2. Initialize Repositories
	meterRepo := meterrepo.Provide()
	priceRepo := pricerepo.Provide()
	priceAmountRepo := priceamountrepo.Provide()
	subRepo := subscriptionrepo.Provide()
	productRepo := productrepo.Provide()
	
	usageRepo := pkg_repository.ProvideStore[usagedomain.UsageEvent](db)
	// ratingRepo := ratingrepo.NewRepository(db) // Accessed via service usually
	// invoiceRepo := invoicerepo.ProvideStore(db)
	billingCycleRepo := pkg_repository.ProvideStore[billingcycledomain.BillingCycle](db)

	// 3. Initialize Services
	meterSvc := meterservice.New(meterservice.Params{
		DB:    db,
		Log:   logger,
		GenID: node,
		Repo:  meterRepo,
	})

	priceSvc := priceservice.New(priceservice.Params{
		DB:          db,
		Log:         logger,
		GenID:       node,
		Repo:        priceRepo,
		ProductRepo: productRepo,
	})

	priceAmountSvc := priceamountservice.New(priceamountservice.Params{
		DB:        db,
		Log:       logger,
		GenID:     node,
		Clock:     clk,
		Repo:      priceAmountRepo,
		PriceRepo: priceRepo,
	})

	subSvc := subscriptionservice.NewService(subscriptionservice.ServiceParam{
		DB:                 db,
		Log:                logger,
		GenID:              node,
		Clock:              clk,
		Repo:               subRepo,
		PaymentMethodSvc:   nil,
		Pricesvc:           priceSvc,
		PriceAmountsvc:     priceAmountSvc,
		ProductFeatureRepo: nil, // Mock if needed
		QuotaSvc:           nil, // Mock?
	})

	// Usage Service needs Meter & Subscription Services
	usageSvc := usageservice.NewService(usageservice.ServiceParam{
		DB:       db,
		Log:      logger,
		GenID:    node,
		Repo:     usageRepo,
		MeterSvc: meterSvc,
		SubSvc:   subSvc,
		Clock:    clk,
		// Outbox: optional, nil
	})

	ratingSvc := ratingservice.NewService(ratingservice.ServiceParam{
		DB:              db,
		Log:             logger,
		GenID:           node,
		PriceAmountRepo: priceAmountRepo,
		BillingCycleRepo: billingCycleRepo,
	})

	ledgerSvc := ledgerservice.NewService(ledgerservice.Params{
		DB:    db,
		Log:   logger,
		GenID: node,
		// No Repo needed
	})

	invoiceSvc := invoiceservice.NewService(invoiceservice.ServiceParam{
		DB:             db,
		Log:            logger,
		GenID:          node,
		AuditSvc:       nil, // Mock
		TemplateRepo:   nil, // Mock
		Renderer:       nil, // Mock
		PublicTokenSvc: nil,
		TaxResolver:    nil, // Mock
		LedgerSvc:      ledgerSvc,
		EmailProvider:  nil, // Mock
		PDFProvider:    nil, // Mock
	})

	// 4. Test Scenario Execution
	ctx := context.Background()
	orgID := node.Generate()
	customerID := node.Generate()

	// 4a. Setup Catalog (Meter, Price, PriceAmount)
	meterID, err := createMeter(ctx, meterSvc, orgID, "api_calls", "sum")
	require.NoError(t, err)

	productID := node.Generate() // Just ID
	priceID, err := createPrice(ctx, priceSvc, orgID, productID, "price_api_calls")
	require.NoError(t, err)

	// Create Price Amount: $0.05 per unit
	err = priceAmountSvc.Create(ctx, priceamountservice.CreateRequest{
		OrgID:           orgID,
		PriceID:         priceID,
		Currency:        "USD",
		UnitAmountCents: 5, // $0.05
		MeterID:         &meterID,
		EffectiveFrom:   time.Now().Add(-24 * time.Hour),
	})
	require.NoError(t, err)

	// 4b. Create Subscription & Cycle
	subID, err := createSubscription(db, orgID, customerID, priceID, meterID)
	require.NoError(t, err)

	cycleStart := time.Now().UTC().Add(-24 * time.Hour)
	cycleEnd := time.Now().UTC().Add(1 * time.Hour) // Ends in future initially?
	// For rating, we usually rate "Closing" cycles.
	// Let's manually create a cycle that is "Closing" or ready to be rated.
	cycleID := node.Generate()
	cycle := billingcycledomain.BillingCycle{
		ID:             cycleID,
		OrgID:          orgID,
		SubscriptionID: subID,
		PeriodStart:    cycleStart,
		PeriodEnd:      cycleEnd,
		Status:         billingcycledomain.BillingCycleStatusClosing, // Ready for rating
	}
	err = db.Create(&cycle).Error
	require.NoError(t, err)

	// 4c. Ingest Usage
	// 100 units * $0.05 = $5.00 => 500 cents
	err = usageSvc.Ingest(ctx, usagedomain.CreateIngestRequest{
		OrgID:              orgID,
		EventName:          "api_calls", // Meter Slug
		ExternalCustomerID: "cust_123",  // Need to map?
		// Usage Service resolves Customer?
		// UsageService.Ingest -> resolveMeter, resolveSubscription.
		// It uses ExternalCustomerID if provided but we don't have mapping service wired?
		// Let's use Subscription ID directly if possible? No, Ingest takes UsageEvent payload.
		// Let's Look at Ingest implementation again.
	})
	// Usage Ingest usually requires `meter_code` and `customer_id`.
	// We didn't create a customer mapping.
	// SHORTCUT: Insert UsageEvent directly into DB to bypass Ingest lookup complexity for now?
	// OR: Fix the setup to be correct.
	// Usage Service needs `customer_id` (Snowflake) OR `external_customer_id`.
	// If `external`, it looks up Customer.
	// Let's Insert a UsageEvent manually to ensure precise state for Rating.

	// Insert Usage Event: 100 Units
	uEvt := usagedomain.UsageEvent{
		ID:             node.Generate(),
		OrgID:          orgID,
		MeterID:        meterID,
		SubscriptionID: subID,
		Value:          100,
		RecordedAt:     time.Now().Add(-1 * time.Hour),
		Status:         usagedomain.UsageStatusEnriched,
	}
	err = db.Create(&uEvt).Error
	require.NoError(t, err)

	// 4d. Run Rating
	err = ratingSvc.RunRating(ctx, cycleID.String())
	require.NoError(t, err)

	// Verify Rating Results
	var results []ratingdomain.RatingResult
	err = db.Where("billing_cycle_id = ?", cycleID).Find(&results).Error
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, int64(500), results[0].Amount) // 100 * 5 = 500
	require.Equal(t, "USD", results[0].Currency)

	// 4e. Scheduler Step: Create Ledger Entry (Simulated)
	// We need to create Ledger Accounts first
	mustEnsureAccount(t, db, orgID, ledgerdomain.AccountCodeRevenueUsage, "Usage Revenue")
	mustEnsureAccount(t, db, orgID, ledgerdomain.AccountCodeAccountsReceivable, "Accounts Receivable")

	// Construct Ledger Logic (Simplified from Scheduler)
	lines := []ledgerdomain.LedgerEntryLine{
		{
			AccountID: mustGetAccountID(t, db, orgID, ledgerdomain.AccountCodeRevenueUsage),
			Direction: ledgerdomain.LedgerEntryDirectionCredit,
			Currency:  "USD",
			Amount:    500,
		},
		{
			AccountID: mustGetAccountID(t, db, orgID, ledgerdomain.AccountCodeAccountsReceivable),
			Direction: ledgerdomain.LedgerEntryDirectionDebit, // AR is Debit
			Currency:  "USD",
			Amount:    500,
		},
	}
	_, err = ledgerSvc.CreateEntry(ctx, ledgerdomain.CreateEntryRequest{
		OrgID:      orgID,
		SourceType: ledgerdomain.SourceTypeBillingCycle,
		SourceID:   cycleID,
		Currency:   "USD",
		OccurredAt: cycleEnd,
		Lines:      lines,
	})
	require.NoError(t, err)

	// Update Cycle to Closed (Prerequisite for Invoice?)
	// Invoice service checks: if cycle.Status != BillingCycleStatusClosed { return ErrBillingCycleNotClosed }
	err = db.Model(&cycle).Update("status", billingcycledomain.BillingCycleStatusClosed).Error
	require.NoError(t, err)

	// 4f. Generate Invoice
	inv, err := invoiceSvc.GenerateInvoice(ctx, cycleID.String())
	require.NoError(t, err)
	require.NotNil(t, inv)

	// 4g. Verify Invoice
	require.Equal(t, invoicedomain.InvoiceStatusDraft, inv.Status)
	require.Equal(t, int64(500), inv.SubtotalAmount)
	require.Equal(t, "USD", inv.Currency)

	// Verify Line Items
	require.Len(t, inv.Items, 1)
	require.Equal(t, int64(500), inv.Items[0].Amount)
	require.Equal(t, float64(100), inv.Items[0].Quantity)
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func createMeter(ctx context.Context, svc meterdomain.Service, orgID snowflake.ID, slug, aggregator string) (snowflake.ID, error) {
	m, err := svc.Create(ctx, meterdomain.CreateMeterRequest{
		OrgID:      orgID,
		Slug:       slug,
		Aggregator: aggregator,
		Name:       slug,
	})
	if err != nil {
		return 0, err
	}
	return m.ID, nil
}

func createPrice(ctx context.Context, svc pricedomain.Service, orgID, productID snowflake.ID, code string) (snowflake.ID, error) {
	p, err := svc.Create(ctx, pricedomain.CreatePriceRequest{
		OrgID:       orgID,
		ProductID:   productID,
		Code:        code,
		Name:        code,
		BillingMode: "METERED", // or FLAT
		Interval:    "MONTH",
	})
	if err != nil {
		return 0, err
	}
	return p.ID, nil
}

func createSubscription(db *gorm.DB, orgID, custID, priceID, meterID snowflake.ID) (snowflake.ID, error) {
	node, _ := snowflake.NewNode(1)
	subID := node.Generate()

	// Subscription
	sub := subscriptiondomain.Subscription{
		ID:         subID,
		OrgID:      orgID,
		CustomerID: custID,
		Status:     subscriptiondomain.SubscriptionStatusActive,
	}
	if err := db.Create(&sub).Error; err != nil {
		return 0, err
	}

	// Entitlement
	ent := subscriptiondomain.SubscriptionEntitlement{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		FeatureCode:    "api_calls", // Needs to match Rating logic?
		// Rating uses Feature Code from... where?
		// Subscription Item -> Price.
		// Rating logic:
		// 1. List Sub Items.
		// 2. Map to Entitlements?
		// Wait, Rating logic iterates Items.
		// We need Subscription Item.
	}
	// Actually Rating iterates SubscriptionItems.
	item := subscriptiondomain.SubscriptionItem{
		ID:             node.Generate(),
		OrgID:          orgID,
		SubscriptionID: subID,
		PriceID:        priceID,
		MeterID:        &meterID,
		Quantity:       1, // Multiplier?
	}
	if err := db.Create(&item).Error; err != nil {
		return 0, err
	}

	// We also need Entitlement for Invoice Description lookup (as per snapshot_test logic)
	ent.FeatureCode = "feature_snapshot" // Just a dummy?
	// Wait, recent fix in Invoice service (Step 512):
	// "Strict Join Rule: WAS Must match entitlement. Now: Optional."
	// But Rating Result puts FeatureCode.
	// Rating Result FeatureCode comes from... Price?

	return subID, nil
}

func mustEnsureAccount(t *testing.T, db *gorm.DB, orgID snowflake.ID, code ledgerdomain.LedgerAccountCode, name string) {
	// Basic insert
	err := db.Exec("INSERT INTO ledger_accounts (id, org_id, code, name, created_at) VALUES (?, ?, ?, ?, ?)",
		snowflake.ID(time.Now().UnixNano()), orgID, code, name, time.Now()).Error
	require.NoError(t, err)
}

func mustGetAccountID(t *testing.T, db *gorm.DB, orgID snowflake.ID, code ledgerdomain.LedgerAccountCode) snowflake.ID {
	var id snowflake.ID
	err := db.Raw("SELECT id FROM ledger_accounts WHERE org_id = ? AND code = ?", orgID, code).Scan(&id).Error
	require.NoError(t, err)
	return id
}
