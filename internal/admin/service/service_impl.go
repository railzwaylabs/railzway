package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/config"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db      *gorm.DB
	limit   int
	timeout time.Duration
	cfg     *config.Config
}

const (
	defaultSummaryConcurrency = 4
	defaultSummaryTimeout     = 2 * time.Second
)

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	limit := defaultSummaryConcurrency
	timeout := defaultSummaryTimeout
	if cfg != nil {
		if cfg.Billing.SummaryConcurrency > 0 {
			limit = cfg.Billing.SummaryConcurrency
		}
		if cfg.Billing.SummaryTimeoutMs > 0 {
			timeout = time.Duration(cfg.Billing.SummaryTimeoutMs) * time.Millisecond
		}
	}
	return &Service{
		db:      db,
		limit:   limit,
		timeout: timeout,
		cfg:     cfg,
	}
}

func (s *Service) runSummary(group *errgroup.Group, ctx context.Context, fn func(ctx context.Context) error) {
	group.Go(func() error {
		tctx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()
		return fn(tctx)
	})
}

func (s *Service) DashboardSummary(ctx context.Context, orgID uuid.UUID) (DashboardSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return DashboardSummary{}, err
	}
	return snapshot.Dashboard, nil
}

func (s *Service) CustomersSummary(ctx context.Context, orgID uuid.UUID) (CustomersSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return CustomersSummary{}, err
	}
	return snapshot.Customers, nil
}

func (s *Service) PlansSummary(ctx context.Context, orgID uuid.UUID) (PlansSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return PlansSummary{}, err
	}
	return snapshot.Plans, nil
}

func (s *Service) SubscriptionsSummary(ctx context.Context, orgID uuid.UUID) (SubscriptionsSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return SubscriptionsSummary{}, err
	}
	return snapshot.Subscriptions, nil
}

func (s *Service) UsageSummary(ctx context.Context, orgID uuid.UUID) (UsageSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return UsageSummary{}, err
	}
	return snapshot.Usage, nil
}

func (s *Service) RatingSummary(ctx context.Context, orgID uuid.UUID) (RatingSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return RatingSummary{}, err
	}
	return snapshot.Rating, nil
}

func (s *Service) InvoicesSummary(ctx context.Context, orgID uuid.UUID) (InvoicesSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return InvoicesSummary{}, err
	}
	return snapshot.Invoices, nil
}

func (s *Service) PaymentsSummary(ctx context.Context, orgID uuid.UUID) (PaymentsSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return PaymentsSummary{}, err
	}
	return snapshot.Payments, nil
}

func (s *Service) TaxesSummary(ctx context.Context, orgID uuid.UUID) (TaxesSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return TaxesSummary{}, err
	}
	return snapshot.Taxes, nil
}

func (s *Service) AuditLogsSummary(ctx context.Context, orgID uuid.UUID) (AuditLogsSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return AuditLogsSummary{}, err
	}
	return snapshot.AuditLogs, nil
}

func (s *Service) SettingsSummary(ctx context.Context, orgID uuid.UUID) (SettingsSummary, error) {
	snapshot, err := s.getSnapshot(ctx, orgID)
	if err != nil {
		return SettingsSummary{}, err
	}
	return snapshot.Settings, nil
}

func (s *Service) computeDashboardSummary(ctx context.Context, orgID uuid.UUID) (DashboardSummary, error) {
	var (
		openInvoices int64
		mrrCents     int64
		usageCents   int64
		lateEvents   int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		openInvoices, err = s.count(ctx, "invoices", "org_id = ? AND status = ?", orgID, invoicedomain.StatusOpen)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		mrrCents, err = s.computeMRR(ctx, orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		usageCents, err = s.computeUsageCents(ctx, orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		lateEvents, err = s.count(ctx, "usage_events", "org_id = ? AND status != ? AND recorded_at >= ?", orgID, usagedomain.StatusRated, time.Now().UTC().Add(-24*time.Hour))
		return err
	})
	if err := group.Wait(); err != nil {
		return DashboardSummary{}, err
	}

	return DashboardSummary{
		MRRCents:     mrrCents,
		UsageCents:   usageCents,
		OpenInvoices: openInvoices,
		LateEvents:   lateEvents,
		Alerts:       []SummaryAlert{},
	}, nil
}

func (s *Service) computeCustomersSummary(ctx context.Context, orgID uuid.UUID) (CustomersSummary, error) {
	var (
		active int64
		atRisk int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		active, err = s.countDistinct(ctx, "subscriptions", "customer_id", "org_id = ? AND status IN ?", orgID, []string{subscriptiondomain.StatusActive, subscriptiondomain.StatusTrialing})
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		atRisk, err = s.countDistinct(ctx, "subscriptions", "customer_id", "org_id = ? AND status = ?", orgID, subscriptiondomain.StatusPastDue)
		return err
	})
	if err := group.Wait(); err != nil {
		return CustomersSummary{}, err
	}

	return CustomersSummary{
		Active:     active,
		AtRisk:     atRisk,
		NRRPct:     0,
		Highlights: []SummaryHighlight{},
	}, nil
}

func (s *Service) computePlansSummary(ctx context.Context, orgID uuid.UUID) (PlansSummary, error) {
	var (
		active int64
		draft  int64
		tiered int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		active, err = s.count(ctx, "plans", "org_id = ? AND active = TRUE", orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		draft, err = s.count(ctx, "plans", "org_id = ? AND active = FALSE", orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		tiered, err = s.count(ctx, "plan_prices", "org_id = ? AND price_type = ?", orgID, plandomain.PriceTypeTiered)
		return err
	})
	if err := group.Wait(); err != nil {
		return PlansSummary{}, err
	}

	return PlansSummary{
		Active:     active,
		Draft:      draft,
		Tiered:     tiered,
		Highlights: []SummaryHighlight{},
	}, nil
}

func (s *Service) computeSubscriptionsSummary(ctx context.Context, orgID uuid.UUID) (SubscriptionsSummary, error) {
	var (
		active   int64
		trialing int64
		pastDue  int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		active, err = s.count(ctx, "subscriptions", "org_id = ? AND status = ?", orgID, subscriptiondomain.StatusActive)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		trialing, err = s.count(ctx, "subscriptions", "org_id = ? AND status = ?", orgID, subscriptiondomain.StatusTrialing)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		pastDue, err = s.count(ctx, "subscriptions", "org_id = ? AND status = ?", orgID, subscriptiondomain.StatusPastDue)
		return err
	})
	if err := group.Wait(); err != nil {
		return SubscriptionsSummary{}, err
	}

	return SubscriptionsSummary{
		Active:     active,
		Trialing:   trialing,
		PastDue:    pastDue,
		Highlights: []SummaryHighlight{},
	}, nil
}

func (s *Service) computeUsageSummary(ctx context.Context, orgID uuid.UUID) (UsageSummary, error) {
	since := time.Now().UTC().Add(-1 * time.Hour)
	lastDay := time.Now().UTC().Add(-24 * time.Hour)
	var (
		eventsPerHour int64
		totalLastDay  int64
		lateLastDay   int64
		activeMeters  int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		eventsPerHour, err = s.count(ctx, "usage_events", "org_id = ? AND recorded_at >= ?", orgID, since)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		totalLastDay, err = s.count(ctx, "usage_events", "org_id = ? AND recorded_at >= ?", orgID, lastDay)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		lateLastDay, err = s.count(ctx, "usage_events", "org_id = ? AND status != ? AND recorded_at >= ?", orgID, usagedomain.StatusRated, lastDay)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		activeMeters, err = s.count(ctx, "meters", "org_id = ? AND active = TRUE", orgID)
		return err
	})
	if err := group.Wait(); err != nil {
		return UsageSummary{}, err
	}

	latePct := 0.0
	if totalLastDay > 0 {
		latePct = (float64(lateLastDay) / float64(totalLastDay)) * 100
	}

	return UsageSummary{
		EventsPerHour: eventsPerHour,
		LatePct:       latePct,
		ActiveMeters:  activeMeters,
		Highlights:    []SummaryHighlight{},
	}, nil
}

func (s *Service) computeRatingSummary(ctx context.Context, orgID uuid.UUID) (RatingSummary, error) {
	var (
		ratedEvents   int64
		avgLatencySec float64
		replaysToday  int64
	)

	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		ratedEvents, err = s.count(ctx, "rating_results", "org_id = ?", orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		avgLatencySec, err = s.computeAvgLatencySec(ctx, orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		replaysToday, err = s.count(ctx, "rating_results", "org_id = ? AND source != ? AND created_at >= ?", orgID, ratingdomain.SourceUsage, startOfDay)
		return err
	})
	if err := group.Wait(); err != nil {
		return RatingSummary{}, err
	}

	return RatingSummary{
		RatedEvents:   ratedEvents,
		AvgLatencySec: avgLatencySec,
		ReplaysToday:  replaysToday,
		Highlights:    []SummaryHighlight{},
	}, nil
}

func (s *Service) computeInvoicesSummary(ctx context.Context, orgID uuid.UUID) (InvoicesSummary, error) {
	var (
		draft     int64
		open      int64
		paidCents int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		draft, err = s.count(ctx, "invoices", "org_id = ? AND status = ?", orgID, invoicedomain.StatusDraft)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		open, err = s.count(ctx, "invoices", "org_id = ? AND status = ?", orgID, invoicedomain.StatusOpen)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		paidCents, err = s.sumInt64(ctx, "invoices", "amount_paid_cents", "org_id = ? AND status = ?", orgID, invoicedomain.StatusPaid)
		return err
	})
	if err := group.Wait(); err != nil {
		return InvoicesSummary{}, err
	}

	return InvoicesSummary{
		Draft:      draft,
		Open:       open,
		PaidCents:  paidCents,
		Highlights: []InvoiceHighlight{},
	}, nil
}

func (s *Service) computePaymentsSummary(ctx context.Context, orgID uuid.UUID) (PaymentsSummary, error) {
	var (
		collectedCents int64
		failed         int64
		retries        int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		collectedCents, err = s.sumInt64(ctx, "payments", "amount_cents", "org_id = ? AND status = ?", orgID, paymentdomain.StatusSucceeded)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		failed, err = s.count(ctx, "payments", "org_id = ? AND status = ?", orgID, paymentdomain.StatusFailed)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		retries, err = s.count(ctx, "payments", "org_id = ? AND status = ?", orgID, paymentdomain.StatusPending)
		return err
	})
	if err := group.Wait(); err != nil {
		return PaymentsSummary{}, err
	}

	return PaymentsSummary{
		CollectedCents: collectedCents,
		Failed:         failed,
		Retries:        retries,
		Highlights:     []SummaryHighlight{},
	}, nil
}

func (s *Service) computeTaxesSummary(ctx context.Context, orgID uuid.UUID) (TaxesSummary, error) {
	var profiles int64
	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		profiles, err = s.count(ctx, "tax_rates", "org_id = ?", orgID)
		return err
	})
	if err := group.Wait(); err != nil {
		return TaxesSummary{}, err
	}

	return TaxesSummary{
		Profiles:        profiles,
		ExemptCustomers: 0,
		Highlights:      []SummaryHighlight{},
	}, nil
}

func (s *Service) computeAuditLogsSummary(ctx context.Context, _ uuid.UUID) (AuditLogsSummary, error) {
	return AuditLogsSummary{Entries: []AuditLogEntry{}}, nil
}

func (s *Service) computeSettingsSummary(ctx context.Context, orgID uuid.UUID) (SettingsSummary, error) {
	var (
		timezone string
		format   string
		apiKeys  int64
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		timezone, err = s.lookupTimezone(ctx, orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		format, err = s.lookupInvoiceFormat(ctx, orgID)
		return err
	})
	s.runSummary(group, gctx, func(ctx context.Context) error {
		var err error
		apiKeys, err = s.count(ctx, "api_keys", "org_id = ? AND status = ?", orgID, "active")
		return err
	})
	if err := group.Wait(); err != nil {
		return SettingsSummary{}, err
	}

	return SettingsSummary{
		APIKeys:       apiKeys,
		InvoiceFormat: format,
		Timezone:      timezone,
		Highlights:    []SummaryHighlight{},
	}, nil
}

func (s *Service) count(ctx context.Context, table, where string, args ...interface{}) (int64, error) {
	var count int64
	stmt := s.db.WithContext(ctx).Table(table)
	if where != "" {
		stmt = stmt.Where(where, args...)
	}
	if err := stmt.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) countDistinct(ctx context.Context, table, column, where string, args ...interface{}) (int64, error) {
	var count int64
	stmt := s.db.WithContext(ctx).Table(table).Select(fmt.Sprintf("COUNT(DISTINCT %s)", column))
	if where != "" {
		stmt = stmt.Where(where, args...)
	}
	if err := stmt.Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) sumInt64(ctx context.Context, table, field, where string, args ...interface{}) (int64, error) {
	var total sql.NullInt64
	stmt := s.db.WithContext(ctx).Table(table).
		Select(fmt.Sprintf("COALESCE(SUM(%s), 0)", field))
	if where != "" {
		stmt = stmt.Where(where, args...)
	}
	if err := stmt.Scan(&total).Error; err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}

func (s *Service) lookupTimezone(ctx context.Context, orgID uuid.UUID) (string, error) {
	var value sql.NullString
	if err := s.db.WithContext(ctx).
		Table("organization_billing_preferences").
		Select("timezone").
		Where("org_id = ?", orgID).
		Limit(1).
		Scan(&value).Error; err != nil {
		return "", err
	}
	if value.Valid {
		return value.String, nil
	}
	return "", nil
}

func (s *Service) lookupInvoiceFormat(ctx context.Context, orgID uuid.UUID) (string, error) {
	var value sql.NullString
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).
		Table("organization_invoice_number_formats").
		Select("format").
		Where("org_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)", orgID, now, now).
		Order("effective_from desc").
		Limit(1).
		Scan(&value).Error; err != nil {
		return "", err
	}
	if value.Valid {
		return value.String, nil
	}
	return "", nil
}

func (s *Service) ConfigWarnings(ctx context.Context, orgID uuid.UUID) (ConfigWarningsResponse, error) {
	var warnings []ConfigWarning

	warnings = append(warnings, s.sessionWarnings(orgID)...)
	warnings = append(warnings, s.transportWarnings(orgID)...)
	warnings = append(warnings, s.appsWarnings(orgID)...)

	format, err := s.lookupInvoiceFormat(ctx, orgID)
	if err != nil {
		return ConfigWarningsResponse{}, err
	}
	if strings.TrimSpace(format) == "" {
		warnings = append(warnings, ConfigWarning{
			Module: "billing",
			Code:   "invoice_format_missing",
			Link:   orgPath(orgID, "settings"),
		})
	}

	taxCount, err := s.count(ctx, "tax_rates", "org_id = ?", orgID)
	if err != nil {
		return ConfigWarningsResponse{}, err
	}
	if taxCount == 0 {
		warnings = append(warnings, ConfigWarning{
			Module: "tax",
			Code:   "tax_rate_missing",
			Link:   orgPath(orgID, "taxes"),
		})
	}

	paymentApps, err := s.countPaymentApps(ctx, orgID)
	if err != nil {
		return ConfigWarningsResponse{}, err
	}
	if paymentApps == 0 {
		warnings = append(warnings, ConfigWarning{
			Module: "payments",
			Code:   "payment_app_missing",
			Link:   orgPath(orgID, "apps"),
		})
	}

	reconciliationCount, err := s.countReconciliationMismatches(ctx, orgID)
	if err != nil {
		return ConfigWarningsResponse{}, err
	}
	if reconciliationCount > 0 {
		windowDays := s.reconciliationWindowDays()
		warnings = append(warnings, ConfigWarning{
			Module: "reconciliation",
			Code:   "reconciliation_mismatch",
			Link:   orgDashboardPath(orgID, "reconciliation"),
			Metadata: map[string]interface{}{
				"count":       reconciliationCount,
				"window_days": windowDays,
			},
		})
	}

	apiKeyCount, err := s.count(ctx, "api_keys", "org_id = ? AND status = ?", orgID, "active")
	if err != nil {
		return ConfigWarningsResponse{}, err
	}
	if apiKeyCount == 0 {
		warnings = append(warnings, ConfigWarning{
			Module: "api",
			Code:   "api_key_missing",
			Link:   orgPath(orgID, "settings"),
		})
	}

	return ConfigWarningsResponse{Warnings: warnings}, nil
}

func (s *Service) sessionWarnings(orgID uuid.UUID) []ConfigWarning {
	if s.cfg == nil {
		return []ConfigWarning{{
			Module: "auth",
			Code:   "session_secret_missing",
			Link:   orgPath(orgID, "settings"),
		}}
	}
	secret := strings.TrimSpace(s.cfg.Session.Secret)
	if secret == "" {
		return []ConfigWarning{{
			Module: "auth",
			Code:   "session_secret_missing",
			Link:   orgPath(orgID, "settings"),
		}}
	}
	if secret == "dev-secret-change-me" {
		return []ConfigWarning{{
			Module: "auth",
			Code:   "session_secret_default",
			Link:   orgPath(orgID, "settings"),
		}}
	}
	if len(secret) < 16 {
		return []ConfigWarning{{
			Module: "auth",
			Code:   "session_secret_weak",
			Link:   orgPath(orgID, "settings"),
		}}
	}
	return nil
}

func (s *Service) transportWarnings(orgID uuid.UUID) []ConfigWarning {
	if s.cfg == nil {
		return nil
	}
	if s.cfg.App.Env.IsProduction() && s.cfg.App.TLSMode.IsDisabled() {
		return []ConfigWarning{{
			Module: "transport",
			Code:   "tls_disabled",
			Link:   orgPath(orgID, "settings"),
		}}
	}
	return nil
}

func (s *Service) appsWarnings(orgID uuid.UUID) []ConfigWarning {
	if s.cfg == nil {
		return []ConfigWarning{{
			Module: "apps",
			Code:   "apps_credentials_key_missing",
			Link:   orgPath(orgID, "apps"),
		}}
	}
	key := strings.TrimSpace(s.cfg.Integrations.AppsCredentialsKey)
	if key == "" {
		return []ConfigWarning{{
			Module: "apps",
			Code:   "apps_credentials_key_missing",
			Link:   orgPath(orgID, "apps"),
		}}
	}
	if len(key) == 32 {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(decoded) != 32 {
		return []ConfigWarning{{
			Module: "apps",
			Code:   "apps_credentials_key_invalid",
			Link:   orgPath(orgID, "apps"),
		}}
	}
	return nil
}

func (s *Service) countPaymentApps(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("apps_installations").
		Joins("JOIN apps_catalog ON apps_catalog.id = apps_installations.app_id").
		Where("apps_installations.org_id = ? AND apps_catalog.category = ?", orgID, "payment").
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) ReconciliationSummary(ctx context.Context, orgID uuid.UUID) (ReconciliationSummaryResponse, error) {
	windowDays := s.reconciliationWindowDays()
	cutoff := time.Now().UTC().AddDate(0, 0, -windowDays)

	usageCount, err := s.countReconciliationAction(ctx, orgID, "reconciliation.usage_mismatch", cutoff)
	if err != nil {
		return ReconciliationSummaryResponse{}, err
	}
	ledgerCount, err := s.countReconciliationAction(ctx, orgID, "reconciliation.ledger_mismatch", cutoff)
	if err != nil {
		return ReconciliationSummaryResponse{}, err
	}

	type row struct {
		Action    string          `gorm:"column:action"`
		InvoiceID string          `gorm:"column:invoice_id"`
		Metadata  json.RawMessage `gorm:"column:metadata"`
		CreatedAt time.Time       `gorm:"column:created_at"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(
		`SELECT action, resource_id as invoice_id, metadata, created_at
		 FROM audit_logs
		 WHERE org_id = ? AND action IN (?, ?) AND created_at >= ?
		 ORDER BY created_at DESC
		 LIMIT 10`,
		orgID, "reconciliation.usage_mismatch", "reconciliation.ledger_mismatch", cutoff,
	).Scan(&rows).Error; err != nil {
		return ReconciliationSummaryResponse{}, err
	}

	latest := make([]ReconciliationMismatch, 0, len(rows))
	for _, item := range rows {
		latest = append(latest, ReconciliationMismatch{
			Action:    item.Action,
			InvoiceID: item.InvoiceID,
			Metadata:  item.Metadata,
			CreatedAt: item.CreatedAt,
		})
	}

	total := usageCount + ledgerCount
	return ReconciliationSummaryResponse{
		WindowDays:       windowDays,
		UsageMismatches:  usageCount,
		LedgerMismatches: ledgerCount,
		TotalMismatches:  total,
		Latest:           latest,
	}, nil
}

func (s *Service) countReconciliationMismatches(ctx context.Context, orgID uuid.UUID) (int64, error) {
	windowDays := s.reconciliationWindowDays()
	cutoff := time.Now().UTC().AddDate(0, 0, -windowDays)
	usageCount, err := s.countReconciliationAction(ctx, orgID, "reconciliation.usage_mismatch", cutoff)
	if err != nil {
		return 0, err
	}
	ledgerCount, err := s.countReconciliationAction(ctx, orgID, "reconciliation.ledger_mismatch", cutoff)
	if err != nil {
		return 0, err
	}
	return usageCount + ledgerCount, nil
}

func (s *Service) countReconciliationAction(ctx context.Context, orgID uuid.UUID, action string, cutoff time.Time) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).
		Table("audit_logs").
		Where("org_id = ? AND action = ? AND created_at >= ?", orgID, action, cutoff).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) reconciliationWindowDays() int {
	windowDays := 7
	if s.cfg != nil && s.cfg.Billing.ReconciliationWindowDays > 0 {
		windowDays = s.cfg.Billing.ReconciliationWindowDays
	}
	return windowDays
}

func orgPath(orgID uuid.UUID, path string) string {
	suffix := strings.TrimPrefix(strings.TrimSpace(path), "/")
	if suffix == "" {
		return fmt.Sprintf("/organizations/%s", orgID.String())
	}
	return fmt.Sprintf("/organizations/%s/%s", orgID.String(), suffix)
}

func orgDashboardPath(orgID uuid.UUID, anchor string) string {
	base := orgPath(orgID, "dashboard")
	if strings.TrimSpace(anchor) == "" {
		return base
	}
	return fmt.Sprintf("%s#%s", base, strings.TrimPrefix(anchor, "#"))
}

func (s *Service) computeUsageCents(ctx context.Context, orgID uuid.UUID) (int64, error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)
	return s.sumInt64(
		ctx,
		"rating_results",
		"amount_cents",
		"org_id = ? AND window_start >= ? AND window_start < ?",
		orgID,
		startOfMonth,
		startOfNextMonth,
	)
}

func (s *Service) computeMRR(ctx context.Context, orgID uuid.UUID) (int64, error) {
	type result struct {
		Value sql.NullFloat64 `gorm:"column:mrr_cents"`
	}
	now := time.Now().UTC()
	row := result{}
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(
				si.quantity * pa.unit_amount_cents *
				CASE
					WHEN pp.billing_interval = 'day' THEN (30.0 / pp.billing_interval_count)
					WHEN pp.billing_interval = 'week' THEN (4.0 / pp.billing_interval_count)
					WHEN pp.billing_interval = 'month' THEN (1.0 / pp.billing_interval_count)
					WHEN pp.billing_interval = 'year' THEN (1.0 / (12.0 * pp.billing_interval_count))
					ELSE 0
				END
			), 0) AS mrr_cents
		FROM subscription_items si
		JOIN subscriptions s ON s.id = si.subscription_id AND s.org_id = ?
		JOIN plan_prices pp ON pp.id = si.plan_price_id AND pp.org_id = s.org_id
		JOIN LATERAL (
			SELECT pa1.unit_amount_cents
			FROM plan_amounts pa1
			WHERE pa1.plan_price_id = pp.id
				AND pa1.org_id = s.org_id
				AND pa1.currency = s.currency
				AND pa1.effective_from <= ?
				AND (pa1.effective_to IS NULL OR pa1.effective_to >= ?)
			ORDER BY pa1.effective_from DESC, pa1.created_at DESC
			LIMIT 1
		) pa ON TRUE
		WHERE s.status = ?
			AND pp.price_type = ?
			AND pp.active = TRUE
			AND si.start_at <= ?
			AND (si.end_at IS NULL OR si.end_at >= ?)
	`, orgID, now, now, subscriptiondomain.StatusActive, plandomain.PriceTypeFlat, now, now).Scan(&row).Error
	if err != nil {
		return 0, err
	}
	if row.Value.Valid {
		return int64(row.Value.Float64 + 0.5), nil
	}
	return 0, nil
}

func (s *Service) computeAvgLatencySec(ctx context.Context, orgID uuid.UUID) (float64, error) {
	type row struct {
		RecordedAt time.Time `gorm:"column:recorded_at"`
		RatedAt    time.Time `gorm:"column:rated_at"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Table("rating_results AS r").
		Select("r.created_at AS rated_at, u.recorded_at").
		Joins("JOIN usage_events u ON u.id = r.usage_event_id").
		Where("r.org_id = ?", orgID).
		Order("r.created_at desc").
		Limit(500).
		Scan(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	var total float64
	var count float64
	for _, item := range rows {
		if item.RatedAt.IsZero() || item.RecordedAt.IsZero() {
			continue
		}
		if item.RatedAt.Before(item.RecordedAt) {
			continue
		}
		total += item.RatedAt.Sub(item.RecordedAt).Seconds()
		count++
	}
	if count == 0 {
		return 0, nil
	}
	return total / count, nil
}

func (s *Service) getSnapshot(ctx context.Context, orgID uuid.UUID) (*SummarySnapshot, error) {
	snapshot, err := s.loadSummarySnapshot(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if snapshot != nil {
		return snapshot, nil
	}
	return s.RefreshSummary(ctx, orgID)
}

func (s *Service) loadSummarySnapshot(ctx context.Context, orgID uuid.UUID) (*SummarySnapshot, error) {
	var row AdminSummary
	if err := s.db.WithContext(ctx).Where("org_id = ?", orgID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	snapshot := SummarySnapshot{
		OrgID:       row.OrgID,
		Source:      row.Source,
		RefreshedAt: row.RefreshedAt,
	}

	if err := unmarshalJSON(row.Dashboard, &snapshot.Dashboard); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Customers, &snapshot.Customers); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Plans, &snapshot.Plans); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Subscriptions, &snapshot.Subscriptions); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Usage, &snapshot.Usage); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Rating, &snapshot.Rating); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Invoices, &snapshot.Invoices); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Payments, &snapshot.Payments); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Taxes, &snapshot.Taxes); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.AuditLogs, &snapshot.AuditLogs); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(row.Settings, &snapshot.Settings); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (s *Service) RefreshSummary(ctx context.Context, orgID uuid.UUID) (*SummarySnapshot, error) {
	var (
		dashboard     DashboardSummary
		customers     CustomersSummary
		plans         PlansSummary
		subscriptions SubscriptionsSummary
		usage         UsageSummary
		rating        RatingSummary
		invoices      InvoicesSummary
		payments      PaymentsSummary
		taxes         TaxesSummary
		auditLogs     AuditLogsSummary
		settings      SettingsSummary
	)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	group.Go(func() error {
		var err error
		dashboard, err = s.computeDashboardSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		customers, err = s.computeCustomersSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		plans, err = s.computePlansSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		subscriptions, err = s.computeSubscriptionsSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		usage, err = s.computeUsageSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		rating, err = s.computeRatingSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		invoices, err = s.computeInvoicesSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		payments, err = s.computePaymentsSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		taxes, err = s.computeTaxesSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		auditLogs, err = s.computeAuditLogsSummary(gctx, orgID)
		return err
	})
	group.Go(func() error {
		var err error
		settings, err = s.computeSettingsSummary(gctx, orgID)
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}

	snapshot := SummarySnapshot{
		OrgID:         orgID,
		Dashboard:     dashboard,
		Customers:     customers,
		Plans:         plans,
		Subscriptions: subscriptions,
		Usage:         usage,
		Rating:        rating,
		Invoices:      invoices,
		Payments:      payments,
		Taxes:         taxes,
		AuditLogs:     auditLogs,
		Settings:      settings,
		Source:        "materialized",
		RefreshedAt:   time.Now().UTC(),
	}

	if err := s.saveSummarySnapshot(ctx, snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (s *Service) RefreshAllSummaries(ctx context.Context) error {
	orgIDs, err := s.listOrgIDs(ctx)
	if err != nil {
		return err
	}
	if len(orgIDs) == 0 {
		return nil
	}

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(s.limit)
	for _, orgID := range orgIDs {
		s.runSummary(group, gctx, func(ctx context.Context) error {
			_, err := s.RefreshSummary(ctx, orgID)
			return err
		})
	}
	return group.Wait()
}

func (s *Service) listOrgIDs(ctx context.Context) ([]uuid.UUID, error) {
	var orgIDs []uuid.UUID
	if err := s.db.WithContext(ctx).Table("organizations").Select("id").Scan(&orgIDs).Error; err != nil {
		return nil, err
	}
	return orgIDs, nil
}

func (s *Service) saveSummarySnapshot(ctx context.Context, snapshot SummarySnapshot) error {
	dashboard, err := marshalJSON(snapshot.Dashboard)
	if err != nil {
		return err
	}
	customers, err := marshalJSON(snapshot.Customers)
	if err != nil {
		return err
	}
	plans, err := marshalJSON(snapshot.Plans)
	if err != nil {
		return err
	}
	subscriptions, err := marshalJSON(snapshot.Subscriptions)
	if err != nil {
		return err
	}
	usage, err := marshalJSON(snapshot.Usage)
	if err != nil {
		return err
	}
	rating, err := marshalJSON(snapshot.Rating)
	if err != nil {
		return err
	}
	invoices, err := marshalJSON(snapshot.Invoices)
	if err != nil {
		return err
	}
	payments, err := marshalJSON(snapshot.Payments)
	if err != nil {
		return err
	}
	taxes, err := marshalJSON(snapshot.Taxes)
	if err != nil {
		return err
	}
	auditLogs, err := marshalJSON(snapshot.AuditLogs)
	if err != nil {
		return err
	}
	settings, err := marshalJSON(snapshot.Settings)
	if err != nil {
		return err
	}

	row := AdminSummary{
		OrgID:         snapshot.OrgID,
		Dashboard:     dashboard,
		Customers:     customers,
		Plans:         plans,
		Subscriptions: subscriptions,
		Usage:         usage,
		Rating:        rating,
		Invoices:      invoices,
		Payments:      payments,
		Taxes:         taxes,
		AuditLogs:     auditLogs,
		Settings:      settings,
		Source:        snapshot.Source,
		RefreshedAt:   snapshot.RefreshedAt,
	}

	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "org_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"dashboard",
			"customers",
			"plans",
			"subscriptions",
			"usage",
			"rating",
			"invoices",
			"payments",
			"taxes",
			"audit_logs",
			"settings",
			"source",
			"refreshed_at",
		}),
	}).Create(&row).Error
}

func marshalJSON(value any) (json.RawMessage, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func unmarshalJSON(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}
