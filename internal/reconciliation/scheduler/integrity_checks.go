package scheduler

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type couponIntegrityRow struct {
	CouponID       uuid.UUID `gorm:"column:coupon_id"`
	Type           string    `gorm:"column:type"`
	AmountCents    *int64    `gorm:"column:amount_cents"`
	Percentage     *float64  `gorm:"column:percentage"`
	Duration       string    `gorm:"column:duration"`
	DurationMonths *int      `gorm:"column:duration_months"`
	Currency       *string   `gorm:"column:currency"`
	AppliedAt      time.Time `gorm:"column:applied_at"`
	SubtotalCents  int64     `gorm:"column:subtotal_cents"`
	DiscountCents  int64     `gorm:"column:discount_cents"`
}

func checkCouponDiscountMatch(ctx context.Context, db *gorm.DB, inv invoiceRow) (bool, map[string]interface{}, error) {
	if db == nil || inv.SubscriptionID == uuid.Nil {
		return false, nil, nil
	}
	var row couponIntegrityRow
	err := db.WithContext(ctx).Raw(
		`SELECT c.id AS coupon_id,
		        c.type,
		        c.amount_cents,
		        c.percentage,
		        c.duration,
		        c.duration_months,
		        c.currency,
		        sc.applied_at,
		        COALESCE(SUM(CASE WHEN ii.line_type IN ('subscription', 'usage') THEN ii.amount_cents ELSE 0 END), 0) AS subtotal_cents,
		        COALESCE(SUM(CASE WHEN ii.line_type = 'discount' THEN ii.amount_cents ELSE 0 END), 0) AS discount_cents
		 FROM subscription_coupons sc
		 JOIN coupons c ON c.org_id = sc.org_id AND c.id = sc.coupon_id
		 LEFT JOIN invoice_items ii ON ii.org_id = sc.org_id AND ii.invoice_id = ?
		 WHERE sc.org_id = ? AND sc.subscription_id = ?
		 GROUP BY c.id, c.type, c.amount_cents, c.percentage, c.duration, c.duration_months, c.currency, sc.applied_at
		 LIMIT 1`,
		inv.ID, inv.OrgID, inv.SubscriptionID,
	).Scan(&row).Error
	if err != nil {
		return false, nil, err
	}
	if row.CouponID == uuid.Nil {
		return false, nil, nil
	}

	expected := expectedCouponDiscount(row, inv.PeriodStart, inv.PeriodEnd)
	if expected != row.DiscountCents {
		return true, map[string]interface{}{
			"coupon_id":               row.CouponID.String(),
			"expected_discount_cents": expected,
			"invoice_discount_cents":  row.DiscountCents,
		}, nil
	}
	return false, nil, nil
}

func expectedCouponDiscount(row couponIntegrityRow, periodStart, periodEnd time.Time) int64 {
	if !couponActiveForPeriod(row.Duration, row.DurationMonths, row.AppliedAt, periodStart, periodEnd) || row.SubtotalCents <= 0 {
		return 0
	}
	switch row.Type {
	case "FIXED":
		if row.AmountCents == nil || *row.AmountCents <= 0 {
			return 0
		}
		if *row.AmountCents > row.SubtotalCents {
			return row.SubtotalCents
		}
		return *row.AmountCents
	case "PERCENT":
		if row.Percentage == nil || *row.Percentage <= 0 {
			return 0
		}
		amount := int64(math.Round(float64(row.SubtotalCents) * (*row.Percentage / 100.0)))
		if amount > row.SubtotalCents {
			return row.SubtotalCents
		}
		return amount
	default:
		return 0
	}
}

func couponActiveForPeriod(duration string, durationMonths *int, appliedAt, periodStart, periodEnd time.Time) bool {
	applied := appliedAt.UTC()
	start := periodStart.UTC()
	end := periodEnd.UTC()
	if end.Before(applied) || end.Equal(applied) {
		return false
	}
	switch duration {
	case "FOREVER":
		return true
	case "ONCE":
		return start.Before(applied.AddDate(0, 1, 0))
	case "REPEATING":
		if durationMonths == nil || *durationMonths <= 0 {
			return false
		}
		return start.Before(applied.AddDate(0, *durationMonths, 0))
	default:
		return false
	}
}

func checkLedgerCreditMatch(ctx context.Context, db *gorm.DB, inv invoiceRow) (bool, map[string]interface{}, error) {
	if db == nil {
		return false, nil, nil
	}
	var invoiceAdjustments int64
	if err := db.WithContext(ctx).Raw(
		`SELECT COALESCE(SUM(amount_cents), 0)
		 FROM invoice_items
		 WHERE invoice_id = ?
		   AND line_type = 'adjustment'
		   AND metadata->>'source' = 'ledger_credit_draw'`,
		inv.ID,
	).Scan(&invoiceAdjustments).Error; err != nil {
		return false, nil, err
	}
	if invoiceAdjustments == 0 {
		return false, nil, nil
	}

	var ledgerDebits int64
	if err := db.WithContext(ctx).Raw(
		`SELECT COALESCE(SUM(e.amount_cents), 0)
		 FROM ledger_entries e
		 JOIN ledger_transactions t ON t.id = e.transaction_id
		 WHERE t.source_type = 'credit_use'
		   AND t.source_id = ?
		   AND e.account_code = 'credits'
		   AND e.entry_type = 'debit'`,
		inv.ID,
	).Scan(&ledgerDebits).Error; err != nil {
		return false, nil, err
	}

	if invoiceAdjustments != ledgerDebits {
		return true, map[string]interface{}{
			"invoice_credit_adjustment_cents": invoiceAdjustments,
			"ledger_credit_debit_cents":       ledgerDebits,
		}, nil
	}
	return false, nil, nil
}

func recordIntegrityMismatch(ctx context.Context, db *gorm.DB, logger *zap.Logger, action string, inv invoiceRow, meta map[string]interface{}) {
	if len(meta) == 0 {
		return
	}
	metaJSON, _ := json.Marshal(meta)
	logger.Warn(action,
		zap.String("invoice_id", inv.ID.String()),
		zap.ByteString("metadata", metaJSON),
	)
	recordReconciliationAudit(ctx, db, inv.OrgID, action, inv, meta)
}
