package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultReconcileInterval = 6 * time.Hour
	defaultReconcileWindow   = 7
	defaultReconcileLimit    = 200
)

type invoiceRow struct {
	ID             uuid.UUID `gorm:"column:id"`
	OrgID          uuid.UUID `gorm:"column:org_id"`
	SubscriptionID uuid.UUID `gorm:"column:subscription_id"`
	PeriodStart    time.Time `gorm:"column:period_start"`
	PeriodEnd      time.Time `gorm:"column:period_end"`
	TotalCents     int64     `gorm:"column:total_cents"`
}

func StartReconciliationScheduler(
	lc fx.Lifecycle,
	cfg *config.Config,
	db *gorm.DB,
	logger *zap.Logger,
) {
	interval := defaultReconcileInterval
	windowDays := defaultReconcileWindow
	limit := defaultReconcileLimit
	if cfg != nil {
		if cfg.ReconciliationJobIntervalSec > 0 {
			interval = time.Duration(cfg.ReconciliationJobIntervalSec) * time.Second
		}
		if cfg.ReconciliationWindowDays > 0 {
			windowDays = cfg.ReconciliationWindowDays
		}
		if cfg.ReconciliationInvoiceLimit > 0 {
			limit = cfg.ReconciliationInvoiceLimit
		}
	}

	var cancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ticker := time.NewTicker(interval)
			runCtx, stop := context.WithCancel(context.Background())
			cancel = stop
			logger.Info("reconciliation scheduler started", zap.Duration("interval", interval), zap.Int("window_days", windowDays))

			go func() {
				defer ticker.Stop()
				for {
					reconcileInvoices(context.Background(), db, logger, windowDays, limit)

					select {
					case <-ticker.C:
						continue
					case <-runCtx.Done():
						return
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
			}
			return nil
		},
	})
}

func reconcileInvoices(ctx context.Context, db *gorm.DB, logger *zap.Logger, windowDays int, limit int) {
	if db == nil {
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -windowDays)

	var invoices []invoiceRow
	if err := db.WithContext(ctx).Raw(
		`SELECT id, org_id, subscription_id, period_start, period_end, total_cents
		 FROM invoices
		 WHERE status IN ('open','paid','uncollectible')
		   AND period_end >= ?
		 ORDER BY period_end DESC
		 LIMIT ?`,
		cutoff, limit,
	).Scan(&invoices).Error; err != nil {
		logger.Warn("reconciliation fetch failed", zap.Error(err))
		return
	}

	mismatches := 0
	for _, inv := range invoices {
		usageSum, err := sumInvoiceUsage(ctx, db, inv.ID)
		if err != nil {
			logger.Warn("reconciliation invoice usage sum failed", zap.Error(err), zap.String("invoice_id", inv.ID.String()))
			continue
		}
		ratingSum, err := sumRatingForPeriod(ctx, db, inv.OrgID, inv.SubscriptionID, inv.PeriodStart, inv.PeriodEnd)
		if err != nil {
			logger.Warn("reconciliation rating sum failed", zap.Error(err), zap.String("invoice_id", inv.ID.String()))
			continue
		}
		ledgerCredits, err := sumLedgerCredits(ctx, db, inv.ID)
		if err != nil {
			logger.Warn("reconciliation ledger sum failed", zap.Error(err), zap.String("invoice_id", inv.ID.String()))
			continue
		}

		if usageSum != ratingSum {
			mismatches++
			logger.Warn("reconciliation usage mismatch",
				zap.String("invoice_id", inv.ID.String()),
				zap.Int64("invoice_usage_cents", usageSum),
				zap.Int64("rating_cents", ratingSum),
			)
			recordReconciliationAudit(ctx, db, inv.OrgID, "reconciliation.usage_mismatch", inv, map[string]interface{}{
				"invoice_usage_cents": usageSum,
				"rating_cents":        ratingSum,
			})
		}
		if ledgerCredits != inv.TotalCents {
			mismatches++
			logger.Warn("reconciliation ledger mismatch",
				zap.String("invoice_id", inv.ID.String()),
				zap.Int64("ledger_credit_cents", ledgerCredits),
				zap.Int64("invoice_total_cents", inv.TotalCents),
			)
			recordReconciliationAudit(ctx, db, inv.OrgID, "reconciliation.ledger_mismatch", inv, map[string]interface{}{
				"ledger_credit_cents": ledgerCredits,
				"invoice_total_cents": inv.TotalCents,
			})
		}
		if mismatch, meta, err := checkCouponDiscountMatch(ctx, db, inv); err != nil {
			logger.Warn("reconciliation coupon check failed", zap.Error(err), zap.String("invoice_id", inv.ID.String()))
		} else if mismatch {
			mismatches++
			recordIntegrityMismatch(ctx, db, logger, "reconciliation.coupon_mismatch", inv, meta)
		}
		if mismatch, meta, err := checkLedgerCreditMatch(ctx, db, inv); err != nil {
			logger.Warn("reconciliation ledger credit check failed", zap.Error(err), zap.String("invoice_id", inv.ID.String()))
		} else if mismatch {
			mismatches++
			recordIntegrityMismatch(ctx, db, logger, "reconciliation.ledger_credit_mismatch", inv, meta)
		}
	}

	if len(invoices) > 0 {
		logger.Info("reconciliation completed", zap.Int("checked", len(invoices)), zap.Int("mismatches", mismatches))
	}
}

func recordReconciliationAudit(ctx context.Context, db *gorm.DB, orgID uuid.UUID, action string, inv invoiceRow, meta map[string]interface{}) {
	if db == nil {
		return
	}
	metaJSON, _ := json.Marshal(meta)
	resourceID := inv.ID.String()
	actorType := "system"
	requestID := ""
	if rid, ok := ctx.Value("request_id").(string); ok && rid != "" {
		requestID = rid
	}
	_ = db.WithContext(ctx).Exec(
		`INSERT INTO audit_logs (id, org_id, actor_type, action, resource_type, resource_id, metadata, request_id, created_at)
		 VALUES (gen_random_uuid(), ?, ?, ?, ?, ?, ?, ?, NOW())`,
		orgID,
		actorType,
		action,
		"invoice",
		&resourceID,
		metaJSON,
		requestID,
	).Error
}

func sumInvoiceUsage(ctx context.Context, db *gorm.DB, invoiceID uuid.UUID) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Raw(
		`SELECT COALESCE(SUM(amount_cents), 0)
		 FROM invoice_items
		 WHERE invoice_id = ? AND line_type = 'usage'`,
		invoiceID,
	).Scan(&total).Error
	return total, err
}

func sumRatingForPeriod(ctx context.Context, db *gorm.DB, orgID, subscriptionID uuid.UUID, periodStart, periodEnd time.Time) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Raw(
		`SELECT COALESCE(SUM(amount_cents), 0)
		 FROM rating_results
		 WHERE org_id = ? AND subscription_id = ?
		   AND window_start >= ? AND window_end <= ?`,
		orgID, subscriptionID, periodStart, periodEnd,
	).Scan(&total).Error
	return total, err
}

func sumLedgerCredits(ctx context.Context, db *gorm.DB, invoiceID uuid.UUID) (int64, error) {
	var total int64
	err := db.WithContext(ctx).Raw(
		`SELECT COALESCE(SUM(e.amount_cents), 0)
		 FROM ledger_entries e
		 JOIN ledger_transactions t ON t.id = e.transaction_id
		 WHERE t.source_type = 'billing_cycle'
		   AND t.source_id = ?
		   AND e.entry_type = 'credit'`,
		invoiceID,
	).Scan(&total).Error
	return total, err
}
