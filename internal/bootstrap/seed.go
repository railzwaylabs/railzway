package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	adminservice "github.com/railzwaylabs/railzway/internal/admin/service"
	"github.com/railzwaylabs/railzway/internal/config"
	"github.com/railzwaylabs/railzway/internal/ledger"
	"github.com/railzwaylabs/railzway/internal/organization/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func EnsureSeed(lc fx.Lifecycle, cfg *config.Config, db *gorm.DB, logger *zap.Logger) {
	if cfg == nil || db == nil {
		return
	}
	if logger == nil {
		logger = zap.L()
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := ensureAppAuthMethods(ctx, db, logger); err != nil {
				logger.Warn("bootstrap apps auth methods skipped", zap.Error(err))
			}

			if !cfg.BootstrapConfig.EnsureDefaultOrgAndUser {
				return nil
			}

			orgID, err := ensureDefaultOrg(ctx, db, cfg, logger)
			if err != nil {
				logger.Warn("bootstrap default org skipped", zap.Error(err))
				return nil
			}

			if err := ensureDefaultUser(ctx, db, cfg, orgID, logger); err != nil {
				logger.Warn("bootstrap default user skipped", zap.Error(err))
			}

			if err := ensureDefaultLedgerAccounts(ctx, db, orgID, logger); err != nil {
				logger.Warn("bootstrap default COA skipped", zap.Error(err))
			}

			return nil
		},
	})
}

func ensureDefaultOrg(ctx context.Context, db *gorm.DB, cfg *config.Config, logger *zap.Logger) (uuid.UUID, error) {
	var orgIDStr string
	err := db.WithContext(ctx).
		Raw(`SELECT id FROM organizations WHERE is_default = true LIMIT 1`).
		Scan(&orgIDStr).Error
	if err != nil {
		return uuid.Nil, err
	}
	if orgIDStr != "" {
		parsed, err := uuid.Parse(orgIDStr)
		if err != nil {
			return uuid.Nil, err
		}
		return parsed, nil
	}

	name := strings.TrimSpace(cfg.BootstrapConfig.OrgName)
	if name == "" {
		name = "Acme"
	}
	slug := slugify(name)
	if slug == "" {
		slug = "default"
	}

	var count int64
	if err := db.WithContext(ctx).
		Raw(`SELECT COUNT(1) FROM organizations WHERE slug = ?`, slug).
		Scan(&count).Error; err != nil {
		return uuid.Nil, err
	}

	orgID := uuid.New()
	if count > 0 {
		slug = fmt.Sprintf("%s-%s", slug, strings.ReplaceAll(orgID.String()[:8], "-", ""))
	}
	supportEmail := strings.TrimSpace(cfg.BootstrapConfig.UserEmail)

	if err := db.WithContext(ctx).Exec(
		`INSERT INTO organizations (id, name, slug, is_default, support_email, created_at, updated_at)
		 VALUES (?, ?, ?, true, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		orgID, name, slug, supportEmail,
	).Error; err != nil {
		return uuid.Nil, err
	}

	if err := db.WithContext(ctx).Exec(
		`INSERT INTO organization_billing_preferences (org_id, currency, timezone, created_at, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		 ON CONFLICT (org_id) DO NOTHING`,
		orgID, "USD", "UTC",
	).Error; err != nil {
		return uuid.Nil, err
	}

	var formatCount int64
	if err := db.WithContext(ctx).
		Raw(`SELECT COUNT(1) FROM organization_invoice_number_formats WHERE org_id = ?`, orgID).
		Scan(&formatCount).Error; err != nil {
		return uuid.Nil, err
	}

	fmt.Printf("formatCount: %v\n", formatCount)
	if formatCount == 0 {
		if err := db.WithContext(ctx).Exec(
			`INSERT INTO organization_invoice_number_formats (id, org_id, format, sequence_scope, effective_from, created_at, updated_at)
			 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			uuid.New(),
			orgID,
			"{PREFIX}-{YYYY}{MM}-{SEQ:6}",
			"org_month",
		).Error; err != nil {
			return uuid.Nil, err
		}
	}

	logger.Info("bootstrap default org created", zap.String("org_id", orgID.String()), zap.String("slug", slug))
	return orgID, nil
}

func ensureDefaultUser(ctx context.Context, db *gorm.DB, cfg *config.Config, orgID uuid.UUID, logger *zap.Logger) error {
	email := strings.TrimSpace(cfg.BootstrapConfig.UserEmail)
	password := strings.TrimSpace(cfg.BootstrapConfig.UserPassword)
	if email == "" || password == "" {
		return nil
	}

	var userIDStr string
	if err := db.WithContext(ctx).
		Raw(`SELECT id FROM users WHERE email = ? LIMIT 1`, email).
		Scan(&userIDStr).Error; err != nil {
		return err
	}

	var userID uuid.UUID
	if userIDStr != "" {
		parsed, err := uuid.Parse(userIDStr)
		if err != nil {
			return err
		}
		userID = parsed
	} else {
		userID = uuid.New()
		displayName := displayNameFromEmail(email)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := db.WithContext(ctx).Exec(
			`INSERT INTO users (id, provider, display_name, email, password_hash, last_password_changed, must_change_password, metadata, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, true, '{}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			userID, "local", displayName, email, string(hash),
		).Error; err != nil {
			return err
		}
		logger.Info("bootstrap default user created", zap.String("user_id", userID.String()), zap.String("email", email))
	}

	if err := db.WithContext(ctx).Exec(
		`INSERT INTO organization_members (id, org_id, user_id, role, created_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT (org_id, user_id) DO NOTHING`,
		uuid.New(), orgID, userID, domain.RoleOwner,
	).Error; err != nil {
		return err
	}

	return nil
}

func ensureDefaultLedgerAccounts(ctx context.Context, db *gorm.DB, orgID uuid.UUID, logger *zap.Logger) error {
	accounts := ledger.DefaultAccounts(orgID)
	if len(accounts) == 0 {
		return nil
	}
	if err := db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&accounts).Error; err != nil {
		return err
	}
	logger.Info("bootstrap default COA ensured", zap.String("org_id", orgID.String()), zap.Int("accounts", len(accounts)))
	return nil
}

func ensureAppAuthMethods(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	if db == nil {
		return nil
	}
	methods := []struct {
		Code        string
		Name        string
		Description string
	}{
		{Code: "api_keys", Name: "API Keys", Description: "Client provides secret/public keys or tokens."},
		{Code: "oauth2", Name: "OAuth 2.0", Description: "Redirect-based authorization code flow."},
		{Code: "basic_auth", Name: "Basic Auth", Description: "Username/password or SMTP auth."},
		{Code: "webhook_only", Name: "Webhook Only", Description: "No credentials; inbound webhooks only."},
	}

	for _, method := range methods {
		if err := db.WithContext(ctx).Exec(
			`INSERT INTO apps_auth_methods (id, code, name, description, created_at)
			 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			 ON CONFLICT (code) DO NOTHING`,
			newUUIDv7(),
			method.Code,
			method.Name,
			method.Description,
		).Error; err != nil {
			return err
		}
	}

	if logger != nil {
		logger.Info("bootstrap apps auth methods ensured", zap.Int("count", len(methods)))
	}
	return nil
}

func ensureDummySummaries(ctx context.Context, db *gorm.DB, orgID uuid.UUID, cfg *config.Config) error {
	var existing string
	if err := db.WithContext(ctx).
		Raw(`SELECT org_id FROM admin_summaries WHERE org_id = ? LIMIT 1`, orgID).
		Scan(&existing).Error; err != nil {
		return err
	}
	if existing != "" {
		return nil
	}

	snapshot := dummySummarySnapshot(orgID, cfg)
	dashboard, _ := json.Marshal(snapshot.Dashboard)
	customers, _ := json.Marshal(snapshot.Customers)
	plans, _ := json.Marshal(snapshot.Plans)
	subscriptions, _ := json.Marshal(snapshot.Subscriptions)
	usage, _ := json.Marshal(snapshot.Usage)
	rating, _ := json.Marshal(snapshot.Rating)
	invoices, _ := json.Marshal(snapshot.Invoices)
	payments, _ := json.Marshal(snapshot.Payments)
	taxes, _ := json.Marshal(snapshot.Taxes)
	auditLogs, _ := json.Marshal(snapshot.AuditLogs)
	settings, _ := json.Marshal(snapshot.Settings)

	record := adminservice.AdminSummary{
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
		Source:        "seed",
		RefreshedAt:   time.Now().UTC(),
	}

	return db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record).Error
}

func dummySummarySnapshot(orgID uuid.UUID, cfg *config.Config) adminservice.SummarySnapshot {
	now := time.Now().UTC()
	highlights := []adminservice.SummaryHighlight{
		{Name: "Starter", Note: "Most popular", Tag: "top"},
		{Name: "Growth", Note: "Highest conversion", Tag: "hot"},
	}
	invoiceHighlights := []adminservice.InvoiceHighlight{
		{Number: "INV-2026-00042", Note: "Largest invoice", Tag: "top"},
		{Number: "INV-2026-00041", Note: "Overdue 3 days", Tag: "late"},
	}
	auditEntries := []adminservice.AuditLogEntry{
		{Title: "Plan updated", Note: "Growth plan price changed", Tag: "plan"},
		{Title: "New API key", Note: "Key created by admin", Tag: "security"},
	}

	return adminservice.SummarySnapshot{
		OrgID: orgID,
		Dashboard: adminservice.DashboardSummary{
			MRRCents:     125_000,
			UsageCents:   8_750,
			OpenInvoices: 3,
			LateEvents:   2,
			Alerts: []adminservice.SummaryAlert{
				{Title: "Usage spike", Subtitle: "Last 24h up 22%", Tag: "usage"},
				{Title: "Overdue invoices", Subtitle: "3 invoices pending", Tag: "invoice"},
			},
		},
		Customers: adminservice.CustomersSummary{
			Active:     42,
			AtRisk:     3,
			NRRPct:     112.5,
			Highlights: highlights,
		},
		Plans: adminservice.PlansSummary{
			Active:     5,
			Draft:      2,
			Tiered:     2,
			Highlights: highlights,
		},
		Subscriptions: adminservice.SubscriptionsSummary{
			Active:     38,
			Trialing:   4,
			PastDue:    1,
			Highlights: highlights,
		},
		Usage: adminservice.UsageSummary{
			EventsPerHour: 1200,
			LatePct:       0.8,
			ActiveMeters:  6,
			Highlights: []adminservice.SummaryHighlight{
				{Name: "Token usage", Note: "Peak at 10:00", Tag: "trend"},
			},
		},
		Rating: adminservice.RatingSummary{
			RatedEvents:   35_000,
			AvgLatencySec: 0.85,
			ReplaysToday:  4,
			Highlights: []adminservice.SummaryHighlight{
				{Name: "Realtime", Note: "p95 900ms", Tag: "latency"},
			},
		},
		Invoices: adminservice.InvoicesSummary{
			Draft:      2,
			Open:       3,
			PaidCents:  230_000,
			Highlights: invoiceHighlights,
		},
		Payments: adminservice.PaymentsSummary{
			CollectedCents: 225_000,
			Failed:         1,
			Retries:        2,
			Highlights: []adminservice.SummaryHighlight{
				{Name: "Stripe", Note: "99.1% success", Tag: "gateway"},
			},
		},
		Taxes: adminservice.TaxesSummary{
			Profiles:        1,
			ExemptCustomers: 2,
			Highlights: []adminservice.SummaryHighlight{
				{Name: "VAT", Note: "EU profile active", Tag: "tax"},
			},
		},
		AuditLogs: adminservice.AuditLogsSummary{
			Entries: auditEntries,
		},
		Settings: adminservice.SettingsSummary{
			APIKeys:       2,
			InvoiceFormat: "INV-{YYYY}{MM}-{SEQ}",
			Timezone:      "UTC",
			Highlights: []adminservice.SummaryHighlight{
				{Name: "Keys", Note: "2 active keys", Tag: "security"},
			},
		},
		Source:      "seed",
		RefreshedAt: now,
	}
}

func slugify(input string) string {
	in := strings.TrimSpace(strings.ToLower(input))
	if in == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(in))
	prevDash := false
	for i := 0; i < len(in); i++ {
		ch := in[i]
		switch {
		case ch >= 'a' && ch <= 'z':
			b.WriteByte(ch)
			prevDash = false
		case ch >= '0' && ch <= '9':
			b.WriteByte(ch)
			prevDash = false
		case ch == ' ' || ch == '-' || ch == '_':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	return slug
}

func displayNameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 0 {
		return "Admin"
	}
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "Admin"
	}
	return name
}

func newUUIDv7() uuid.UUID {
	if id, err := uuid.NewV7(); err == nil {
		return id
	}
	return uuid.New()
}
