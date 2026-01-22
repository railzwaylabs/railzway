package scheduler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	"github.com/prometheus/client_golang/prometheus"
	auditdomain "github.com/smallbiznis/railzway/internal/audit/domain"
	billingcycledomain "github.com/smallbiznis/railzway/internal/billingcycle/domain"
	"github.com/smallbiznis/railzway/internal/clock"
	invoicedomain "github.com/smallbiznis/railzway/internal/invoice/domain"
	ledgerdomain "github.com/smallbiznis/railzway/internal/ledger/domain"
	obsmetrics "github.com/smallbiznis/railzway/internal/observability/metrics"
	subscriptiondomain "github.com/smallbiznis/railzway/internal/subscription/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Mocks for dependencies

type mockRatingSvc struct {
	db *gorm.DB
}

func (m *mockRatingSvc) RunRating(ctx context.Context, cycleID string) error {
	// Side-effect: insert rating result so hasRatingResults returns true and ledger logic works
	return m.db.Exec(`
		INSERT INTO rating_results (billing_cycle_id, org_id, meter_id, currency, amount) 
		VALUES (?, ?, ?, ?, ?)
	`, cycleID, 2010735548360036353, nil, "USD", 100.0).Error
}

type mockInvoiceSvc struct {
	genFunc func(ctx context.Context, cycleID string) (*invoicedomain.Invoice, error)
	finFunc func(ctx context.Context, invoiceID string) error
}

func (m *mockInvoiceSvc) List(context.Context, invoicedomain.ListInvoiceRequest) (invoicedomain.ListInvoiceResponse, error) {
	return invoicedomain.ListInvoiceResponse{}, nil
}
func (m *mockInvoiceSvc) GetByID(ctx context.Context, id string) (invoicedomain.Invoice, error) {
	return invoicedomain.Invoice{}, nil
}
func (m *mockInvoiceSvc) RenderInvoice(ctx context.Context, invoiceID string) (invoicedomain.RenderInvoiceResponse, error) {
	return invoicedomain.RenderInvoiceResponse{}, nil
}
func (m *mockInvoiceSvc) GenerateInvoice(ctx context.Context, billingCycleID string) (*invoicedomain.Invoice, error) {
	if m.genFunc != nil {
		return m.genFunc(ctx, billingCycleID)
	}
	return nil, nil
}
func (m *mockInvoiceSvc) FinalizeInvoice(ctx context.Context, invoiceID string) error {
	if m.finFunc != nil {
		return m.finFunc(ctx, invoiceID)
	}
	return nil
}
func (m *mockInvoiceSvc) VoidInvoice(ctx context.Context, invoiceID string, reason string) error {
	return nil
}

type mockLedgerSvc struct{}

func (m *mockLedgerSvc) CreateEntry(ctx context.Context, orgID snowflake.ID, sourceType string, sourceID snowflake.ID, currency string, occurredAt time.Time, lines []ledgerdomain.LedgerEntryLine) error {
	return nil
}

type mockSubscriptionSvc struct{}

func (m *mockSubscriptionSvc) List(context.Context, subscriptiondomain.ListSubscriptionRequest) (subscriptiondomain.ListSubscriptionResponse, error) {
	return subscriptiondomain.ListSubscriptionResponse{}, nil
}
func (m *mockSubscriptionSvc) Create(context.Context, subscriptiondomain.CreateSubscriptionRequest) (subscriptiondomain.CreateSubscriptionResponse, error) {
	return subscriptiondomain.CreateSubscriptionResponse{}, nil
}
func (m *mockSubscriptionSvc) ReplaceItems(context.Context, subscriptiondomain.ReplaceSubscriptionItemsRequest) (subscriptiondomain.CreateSubscriptionResponse, error) {
	return subscriptiondomain.CreateSubscriptionResponse{}, nil
}
func (m *mockSubscriptionSvc) GetByID(context.Context, string) (subscriptiondomain.Subscription, error) {
	return subscriptiondomain.Subscription{}, nil
}
func (m *mockSubscriptionSvc) GetActiveByCustomerID(context.Context, subscriptiondomain.GetActiveByCustomerIDRequest) (subscriptiondomain.Subscription, error) {
	return subscriptiondomain.Subscription{}, nil
}
func (m *mockSubscriptionSvc) GetSubscriptionItem(context.Context, subscriptiondomain.GetSubscriptionItemRequest) (subscriptiondomain.SubscriptionItem, error) {
	return subscriptiondomain.SubscriptionItem{}, nil
}
func (m *mockSubscriptionSvc) TransitionSubscription(ctx context.Context, id string, status subscriptiondomain.SubscriptionStatus, reason subscriptiondomain.TransitionReason) error {
	return nil
}
func (m *mockSubscriptionSvc) ValidateUsageEntitlement(ctx context.Context, subscriptionID, meterID snowflake.ID, at time.Time) error {
	return nil
}
func (m *mockSubscriptionSvc) ChangePlan(ctx context.Context, req subscriptiondomain.ChangePlanRequest) error {
	return nil
}

type mockAuditSvc struct{}

func (m *mockAuditSvc) AuditLog(ctx context.Context, orgID *snowflake.ID, userID string, actorID *string, action string, targetType string, targetID *string, metadata map[string]any) error {
	return nil
}

func (m *mockAuditSvc) List(ctx context.Context, req auditdomain.ListAuditLogRequest) (auditdomain.ListAuditLogResponse, error) {
	return auditdomain.ListAuditLogResponse{}, nil
}

type mockAuthzSvc struct{}

func (m *mockAuthzSvc) Authorize(ctx context.Context, subject, domain, object, action string) error {
	return nil
}

// TestScheduler_RunOnce_FakeClock_30Days verifies scheduler behavior over a simulated 30-day period
func TestScheduler_RunOnce_FakeClock_30Days(t *testing.T) {
	// 1. Setup DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	// SQLite support hack: remove FOR UPDATE clauses
	db.Callback().Query().Before("gorm:query").Register("sqlite_skip_locked", func(d *gorm.DB) {
		sql := d.Statement.SQL.String()
		if strings.Contains(sql, "FOR UPDATE") {
			newSQL := strings.ReplaceAll(sql, "FOR UPDATE SKIP LOCKED", "")
			newSQL = strings.ReplaceAll(newSQL, "FOR UPDATE", "")
			d.Statement.SQL.Reset()
			d.Statement.SQL.WriteString(newSQL)
		}
	})
	db.Callback().Row().Before("gorm:row").Register("sqlite_skip_locked_row", func(d *gorm.DB) {
		sql := d.Statement.SQL.String()
		if strings.Contains(sql, "FOR UPDATE") {
			newSQL := strings.ReplaceAll(sql, "FOR UPDATE SKIP LOCKED", "")
			newSQL = strings.ReplaceAll(newSQL, "FOR UPDATE", "")
			d.Statement.SQL.Reset()
			d.Statement.SQL.WriteString(newSQL)
		}
	})

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	// Create tables needed by Scheduler
	// subscriptions table
	if err := db.Exec(`
		CREATE TABLE subscriptions (
			id INTEGER PRIMARY KEY,
			org_id INTEGER,
			status TEXT,
			activated_at DATETIME,
			billing_cycle_type TEXT
		)
	`).Error; err != nil {
		t.Fatalf("create subscriptions table: %v", err)
	}
	// billing_cycles table
	if err := db.Exec(`
		CREATE TABLE billing_cycles (
			id INTEGER PRIMARY KEY,
			org_id INTEGER,
			subscription_id INTEGER,
			period_start DATETIME,
			period_end DATETIME,
			status TEXT,
			opened_at DATETIME,
			closing_started_at DATETIME,
			rating_completed_at DATETIME,
			invoiced_at DATETIME,
			invoice_finalized_at DATETIME,
			closed_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			last_error TEXT,
			last_error_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create billing_cycles table: %v", err)
	}
	// invoices table
	if err := db.Exec(`
		CREATE TABLE invoices (
			id INTEGER PRIMARY KEY,
			billing_cycle_id INTEGER,
			status TEXT,
			finalized_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create invoices table: %v", err)
	}
	// rating_results (for hasRatingResults check)
	if err := db.Exec(`
		CREATE TABLE rating_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			billing_cycle_id INTEGER,
			org_id INTEGER,
			meter_id TEXT,
			currency TEXT,
			amount REAL
		)
	`).Error; err != nil {
		t.Fatalf("create rating_results table: %v", err)
	}
	// billing_cycle_stats
	if err := db.Exec(`
		CREATE TABLE billing_cycle_stats (
			billing_cycle_id INTEGER PRIMARY KEY,
			org_id INTEGER,
			period_start DATETIME,
			status TEXT,
			total_revenue REAL,
			invoice_count INTEGER,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create billing_cycle_stats table: %v", err)
	}

	// ledger_accounts
	if err := db.Exec(`
		CREATE TABLE ledger_accounts (
			id INTEGER PRIMARY KEY,
			org_id INTEGER,
			code TEXT,
			type TEXT,
			name TEXT,
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create ledger_accounts table: %v", err)
	}

	// 2. Setup Dependencies
	node, _ := snowflake.NewNode(1)
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(startTime)

	// Seed ledger accounts
	testOrgID := snowflake.ID(2010735548360036353)
	accounts := []struct {
		Code string
		Type string
	}{
		{"revenue_flat", "income"},
		{"revenue_usage", "income"},
		{"accounts_receivable", "assets"},
	}
	for _, acc := range accounts {
		if err := db.Exec(`
			INSERT INTO ledger_accounts (id, org_id, code, type, name, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, node.Generate(), testOrgID, acc.Code, acc.Type, acc.Code, startTime).Error; err != nil {
			t.Fatalf("seed ledger account %s: %v", acc.Code, err)
		}
	}

	// Reset metrics
	registry := prometheus.NewRegistry()
	restore := swapPrometheusRegistry(registry)
	defer restore()
	obsmetrics.ResetSchedulerMetricsForTest()
	obsmetrics.SchedulerWithConfig(obsmetrics.Config{ServiceName: "test", Environment: "test"})

	// Mocks
	ratingSvc := &mockRatingSvc{db: db}
	invoiceSvc := &mockInvoiceSvc{
		genFunc: func(ctx context.Context, cycleID string) (*invoicedomain.Invoice, error) {
			// Simulate invoice generation
			invID := node.Generate()
			return &invoicedomain.Invoice{
				ID:     invID,
				Status: invoicedomain.InvoiceStatusDraft,
			}, nil
		},
		finFunc: func(ctx context.Context, invoiceID string) error {
			// Simulate finalization
			return nil
		},
	}

	scheduler, err := New(Params{
		DB:              db,
		Log:             zap.NewNop(),
		RatingSvc:       ratingSvc,
		InvoiceSvc:      invoiceSvc,
		LedgerSvc:       &mockLedgerSvc{},
		SubscriptionSvc: &mockSubscriptionSvc{},
		AuditSvc:        &mockAuditSvc{},
		AuthzSvc:        &mockAuthzSvc{},
		GenID:           node,
		Clock:           fakeClock,
		Config: Config{
			BatchSize:           10,
			MaxCloseBatchSize:   10,
			MaxRatingBatchSize:  10,
			MaxInvoiceBatchSize: 10,
			FinalizeInvoices:    true,
		},
	})
	if err != nil {
		t.Fatalf("New scheduler: %v", err)
	}

	// 3. Seed Initial Data
	// Create an active Monthly subscription starting at startTime
	subID := node.Generate()
	db.Exec(`
		INSERT INTO subscriptions (id, org_id, status, activated_at, billing_cycle_type)
		VALUES (?, ?, ?, ?, ?)
	`, subID, testOrgID, subscriptiondomain.SubscriptionStatusActive, startTime, "monthly")

	// 4. Run Loop for 30 Days (actually 32 days to cover full month transition)
	// We want to verify:
	// - Cycle created at T0 (period: Jan 1 - Feb 1)
	// - At Feb 1, cycle should close, rate, invoice

	ctx := context.Background()

	// Step 1: Initial Run at Jan 1
	// Should create the first billing cycle
	if err := scheduler.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce failed at start: %v", err)
	}

	// Verify cycle created
	var cycle WorkBillingCycle
	if err := db.Raw("SELECT * FROM billing_cycles WHERE subscription_id = ?", subID).Scan(&cycle).Error; err != nil {
		t.Fatalf("check cycle: %v", err)
	}
	if cycle.ID == 0 {
		t.Fatal("expected billing cycle to be created on day 1")
	}
	if cycle.Status != billingcycledomain.BillingCycleStatusOpen {
		t.Fatalf("expected cycle to be OPEN, got %s", cycle.Status)
	}
	expectedEnd := startTime.AddDate(0, 1, 0) // Feb 1
	if !cycle.PeriodEnd.Equal(expectedEnd) {
		t.Fatalf("expected period end %v, got %v", expectedEnd, cycle.PeriodEnd)
	}

	t.Logf("Cycle %s created. Period: %v - %v", cycle.ID, cycle.PeriodStart, cycle.PeriodEnd)

	// Step 2: Advance time ensuring we pass period end
	// Let's advance day by day
	targetDate := startTime.AddDate(0, 0, 32) // Feb 2

	for fakeClock.Now().Before(targetDate) {
		fakeClock.Advance(24 * time.Hour)
		// Run scheduler each "day"
		// In reality scheduler runs every 30s, but we just need to hit the transition point
		if err := scheduler.RunOnce(ctx); err != nil {
			t.Fatalf("RunOnce failed at %v: %v", fakeClock.Now(), err)
		}
	}

	// 5. Verify Final State
	// At Feb 2, the Jan cycle should be:
	// - Closed
	// - Rated
	// - Invoiced
	// - Invoice Finalized (since cfg.FinalizeInvoices = true)

	// Re-fetch cycle
	if err := db.Raw("SELECT * FROM billing_cycles WHERE id = ?", cycle.ID).Scan(&cycle).Error; err != nil {
		t.Fatalf("refetch cycle: %v", err)
	}

	// Check status chain
	// Cycle should be Closed (after rating and ensureLedger)
	// Actually, wait:
	// - CloseCyclesJob marks it Closing
	// - RatingJob runs (marks RatingCompleted)
	// - CloseAfterRatingJob runs (checks rating, ensures ledger, marks Closed)
	// - InvoiceJob runs (generates invoice, marks Invoiced)

	// In scheduler.go:
	// s.runJob("invoice") checks status=Closed AND invoiced_at IS NULL (line 505)
	// So it MUST be Closed before Invoicing.

	if cycle.Status != billingcycledomain.BillingCycleStatusClosed {
		t.Errorf("expected cycle to be Closed, got %s", cycle.Status)
	}
	if cycle.ClosingStartedAt == nil {
		t.Error("expected ClosingStartedAt to be set")
	}
	if cycle.RatingCompletedAt == nil {
		t.Error("expected RatingCompletedAt to be set")
	}
	if cycle.ClosedAt == nil {
		t.Error("expected ClosedAt to be set")
	}
	if cycle.InvoicedAt == nil {
		t.Error("expected InvoicedAt to be set")
	}
	if cycle.InvoiceFinalizedAt == nil {
		t.Error("expected InvoiceFinalizedAt to be set")
	}

	// Check if a NEW cycle was created for Feb?
	// EnsureBillingCyclesJob should have run.
	// Logic: findLastCycle. If LastCycle.PeriodEnd < Now (Feb 2), it should check if it can open next.
	// LastCycle (Jan) PeriodEnd is Feb 1. Now is Feb 2.
	// So new cycle should be created starting Feb 1.

	var cycles []WorkBillingCycle
	db.Raw("SELECT * FROM billing_cycles WHERE subscription_id = ? ORDER BY period_start", subID).Scan(&cycles)
	if len(cycles) != 2 {
		t.Errorf("expected 2 cycles (Jan and Feb), got %d", len(cycles))
	} else {
		febCycle := cycles[1]
		if febCycle.Status != billingcycledomain.BillingCycleStatusOpen {
			t.Errorf("expected Feb cycle to be Open, got %s", febCycle.Status)
		}
		expectedFebStart := expectedEnd // Feb 1
		if !febCycle.PeriodStart.Equal(expectedFebStart) {
			t.Errorf("expected Feb cycle start %v, got %v", expectedFebStart, febCycle.PeriodStart)
		}
	}
}
