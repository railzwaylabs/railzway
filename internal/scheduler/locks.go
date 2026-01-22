package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	billingcycledomain "github.com/smallbiznis/railzway/internal/billingcycle/domain"
	invoicedomain "github.com/smallbiznis/railzway/internal/invoice/domain"
	obsmetrics "github.com/smallbiznis/railzway/internal/observability/metrics"
	subscriptiondomain "github.com/smallbiznis/railzway/internal/subscription/domain"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkSubscription struct {
	ID               snowflake.ID
	OrgID            snowflake.ID
	Status           subscriptiondomain.SubscriptionStatus
	ActivatedAt      *time.Time
	BillingCycleType string
}

type WorkBillingCycle struct {
	ID                 snowflake.ID
	OrgID              snowflake.ID
	SubscriptionID     snowflake.ID
	PeriodStart        time.Time
	PeriodEnd          time.Time
	Status             billingcycledomain.BillingCycleStatus
	ClosingStartedAt   *time.Time
	RatingCompletedAt  *time.Time
	InvoicedAt         *time.Time
	InvoiceFinalizedAt *time.Time
	ClosedAt           *time.Time
}

func (s *Scheduler) FetchSubscriptionsForWork(ctx context.Context, status subscriptiondomain.SubscriptionStatus, limit int) ([]WorkSubscription, error) {
	claimCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var subscriptions []WorkSubscription
	err := s.db.WithContext(claimCtx).Transaction(func(tx *gorm.DB) error {
		var err error
		subscriptions, err = s.fetchSubscriptionsForWork(claimCtx, tx, status, limit)
		return err
	})
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (s *Scheduler) FetchBillingCyclesForWork(ctx context.Context, status billingcycledomain.BillingCycleStatus, limit int) ([]WorkBillingCycle, error) {
	return s.fetchBillingCyclesForWork(ctx, `status = ?`, []any{status}, limit)
}

func (s *Scheduler) fetchSubscriptionsForWork(ctx context.Context, tx *gorm.DB, status subscriptiondomain.SubscriptionStatus, limit int) ([]WorkSubscription, error) {
	var subscriptions []WorkSubscription
	schedMetrics := obsmetrics.Scheduler()
	lockStart := time.Now()
	err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, status, activated_at, billing_cycle_type
		 FROM subscriptions
		 WHERE status = ?
		 ORDER BY id
		 FOR UPDATE SKIP LOCKED
		 LIMIT ?`,
		status,
		limit,
	).Scan(&subscriptions).Error
	schedMetrics.ObserveDBLockWait(obsmetrics.LockResourceSubscriptionsForWork, time.Since(lockStart))
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (s *Scheduler) fetchBillingCyclesForWork(ctx context.Context, where string, args []any, limit int) ([]WorkBillingCycle, error) {
	if limit <= 0 {
		limit = s.cfg.BatchSize
	}
	var cycles []WorkBillingCycle
	schedMetrics := obsmetrics.Scheduler()
	query := fmt.Sprintf(
		`SELECT id, org_id, subscription_id, period_start, period_end, status,
		        closing_started_at, rating_completed_at, invoiced_at,
		        invoice_finalized_at, closed_at
		 FROM billing_cycles
		 WHERE %s
		 ORDER BY period_end ASC, id ASC
		 FOR UPDATE SKIP LOCKED
		 LIMIT ?`,
		where,
	)
	args = append(args, limit)
	lockStart := time.Now()
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&cycles).Error; err != nil {
		schedMetrics.ObserveDBLockWait(obsmetrics.LockResourceBillingCyclesForWork, time.Since(lockStart))
		return cycles, err
	}
	schedMetrics.ObserveDBLockWait(obsmetrics.LockResourceBillingCyclesForWork, time.Since(lockStart))
	return cycles, nil
}

func (s *Scheduler) fetchSubscriptionsNeedingCycle(ctx context.Context, tx *gorm.DB, limit int) ([]WorkSubscription, error) {
	var subscriptions []WorkSubscription
	schedMetrics := obsmetrics.Scheduler()
	lockStart := time.Now()

	// Select Active subscriptions that DO NOT have an OPEN billing cycle
	// Note: We use FOR UPDATE SKIP LOCKED on the subscription row to ensure exclusive access
	// PostgreSQL: FOR UPDATE OF s SKIP LOCKED
	// MySQL/SQLite: FOR UPDATE SKIP LOCKED works (or striped by test)
	err := tx.WithContext(ctx).Raw(
		`SELECT s.id, s.org_id, s.status, s.activated_at, s.billing_cycle_type
		 FROM subscriptions s
		 WHERE s.status = ?
		   AND NOT EXISTS (
			   SELECT 1 FROM billing_cycles bc 
			   WHERE bc.subscription_id = s.id 
				 AND bc.status = ?
		   )
		 ORDER BY s.id
		 LIMIT ?
		 FOR UPDATE SKIP LOCKED`,
		subscriptiondomain.SubscriptionStatusActive,
		billingcycledomain.BillingCycleStatusOpen,
		limit,
	).Scan(&subscriptions).Error

	schedMetrics.ObserveDBLockWait(obsmetrics.LockResourceSubscriptionsForWork, time.Since(lockStart))
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (s *Scheduler) findOpenCycle(
	ctx context.Context,
	tx *gorm.DB,
	orgID, subscriptionID snowflake.ID,
) (*WorkBillingCycle, int, error) {

	var cycles []WorkBillingCycle
	schedMetrics := obsmetrics.Scheduler()
	lockStart := time.Now()
	err := tx.WithContext(ctx).Raw(`
		SELECT id, org_id, subscription_id, period_start, period_end, status,
		       closing_started_at, rating_completed_at, invoiced_at,
		       invoice_finalized_at, closed_at
		FROM billing_cycles
		WHERE org_id = ?
		  AND subscription_id = ?
		  AND status = ?
		ORDER BY period_end DESC
		LIMIT 2
		FOR UPDATE SKIP LOCKED
	`,
		orgID,
		subscriptionID,
		billingcycledomain.BillingCycleStatusOpen,
	).Scan(&cycles).Error
	schedMetrics.ObserveDBLockWait(obsmetrics.LockResourceOpenCycle, time.Since(lockStart))
	if err != nil {
		return nil, 0, err
	}

	switch len(cycles) {
	case 0:
		return nil, 0, nil
	case 1:
		return &cycles[0], 1, nil
	default:
		return nil, len(cycles), billingcycledomain.ErrMultipleOpenCycles
	}
}

func (s *Scheduler) findLastCycle(ctx context.Context, tx *gorm.DB, orgID, subscriptionID snowflake.ID) (*WorkBillingCycle, error) {
	var cycle WorkBillingCycle
	err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, subscription_id, period_start, period_end, status,
		        closing_started_at, rating_completed_at, invoiced_at,
		        invoice_finalized_at, closed_at
		 FROM billing_cycles
		 WHERE org_id = ? AND subscription_id = ?
		 ORDER BY period_end DESC
		 LIMIT 1`,
		orgID,
		subscriptionID,
	).Scan(&cycle).Error
	if err != nil {
		return nil, err
	}
	if cycle.ID == 0 {
		return nil, nil
	}
	return &cycle, nil
}

func (s *Scheduler) insertCycle(ctx context.Context, tx *gorm.DB, cycleID, orgID, subscriptionID snowflake.ID, periodStart, periodEnd, now time.Time) error {
	openedAt := now
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO billing_cycles (
			id, org_id, subscription_id, period_start, period_end, status,
			opened_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cycleID,
		orgID,
		subscriptionID,
		periodStart,
		periodEnd,
		billingcycledomain.BillingCycleStatusOpen,
		openedAt,
		now,
		now,
	).Error; err != nil {
		return err
	}
	return s.upsertBillingCycleStats(ctx, tx, cycleID, orgID, periodStart, billingcycledomain.BillingCycleStatusOpen, now)
}

func (s *Scheduler) lockCycleForUpdate(
	ctx context.Context,
	tx *gorm.DB,
	cycleID snowflake.ID,
) (*WorkBillingCycle, error) {

	var cycle WorkBillingCycle
	schedMetrics := obsmetrics.Scheduler()
	lockStart := time.Now()
	err := tx.WithContext(ctx).Raw(
		`SELECT id, org_id, subscription_id, period_start, period_end, status,
		        closing_started_at, rating_completed_at, invoiced_at,
		        invoice_finalized_at, closed_at
		 FROM billing_cycles
		 WHERE id = ?
		 FOR UPDATE`,
		cycleID,
	).Scan(&cycle).Error
	schedMetrics.ObserveDBLockWait(obsmetrics.LockResourceBillingCycleByID, time.Since(lockStart))

	if err != nil {
		return nil, err
	}
	if cycle.ID == 0 {
		return nil, nil
	}
	return &cycle, nil
}

func (s *Scheduler) markCycleClosing(ctx context.Context, cycleID snowflake.ID, now time.Time) (bool, error) {
	updated := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated, err := s.markCycleClosingTx(ctx, tx, cycleID, now)
		if !updated {
			return nil
		}
		return err
	})
	return updated, err
}

func (s *Scheduler) markCycleClosingTx(
	ctx context.Context,
	tx *gorm.DB,
	cycleID snowflake.ID,
	now time.Time,
) (bool, error) {

	result := tx.WithContext(ctx).Exec(
		`UPDATE billing_cycles
		 SET status = ?,
		     closing_started_at = COALESCE(closing_started_at, ?),
		     updated_at = ?
		 WHERE id = ?
		   AND status = ?
		   AND period_end <= ?`,
		billingcycledomain.BillingCycleStatusClosing,
		now,
		now,
		cycleID,
		billingcycledomain.BillingCycleStatusOpen,
		now,
	)

	if result.Error != nil {
		return false, result.Error
	}
	updated := result.RowsAffected > 0
	if updated {
		obsmetrics.Scheduler().IncBillingCycleTransition(
			string(billingcycledomain.BillingCycleStatusOpen),
			string(billingcycledomain.BillingCycleStatusClosing),
		)
	}
	return updated, nil
}

func (s *Scheduler) markRatingCompleted(ctx context.Context, cycleID snowflake.ID, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cycle, err := s.lockCycleForUpdate(ctx, tx, cycleID)
		if err != nil {
			return err
		}
		if cycle == nil || cycle.Status != billingcycledomain.BillingCycleStatusClosing {
			return nil
		}
		return tx.WithContext(ctx).Exec(
			`UPDATE billing_cycles
			 SET rating_completed_at = COALESCE(rating_completed_at, ?),
			     last_error = NULL,
			     last_error_at = NULL,
			     updated_at = ?
			 WHERE id = ? AND status = ?`,
			now,
			now,
			cycleID,
			billingcycledomain.BillingCycleStatusClosing,
		).Error
	})
}

func (s *Scheduler) markCycleClosed(ctx context.Context, cycleID snowflake.ID, now time.Time) (bool, error) {
	updated := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cycle, err := s.lockCycleForUpdate(ctx, tx, cycleID)
		if err != nil {
			return err
		}
		if cycle == nil || cycle.Status != billingcycledomain.BillingCycleStatusClosing {
			return nil
		}
		if cycle.RatingCompletedAt == nil {
			return invoicedomain.ErrMissingRatingResults
		}
		result := tx.WithContext(ctx).Exec(
			`UPDATE billing_cycles
			 SET status = ?, closed_at = COALESCE(closed_at, ?),
			     last_error = NULL,
			     last_error_at = NULL,
			     updated_at = ?
			 WHERE id = ? AND status = ? AND rating_completed_at IS NOT NULL`,
			billingcycledomain.BillingCycleStatusClosed,
			now,
			now,
			cycleID,
			billingcycledomain.BillingCycleStatusClosing,
		)
		if result.Error != nil {
			return result.Error
		}
		updated = result.RowsAffected > 0
		if updated {
			obsmetrics.Scheduler().IncBillingCycleTransition(
				string(billingcycledomain.BillingCycleStatusClosing),
				string(billingcycledomain.BillingCycleStatusClosed),
			)
		}
		return nil
	})
	return updated, err
}

func (s *Scheduler) markCycleInvoiced(ctx context.Context, cycleID snowflake.ID, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cycle, err := s.lockCycleForUpdate(ctx, tx, cycleID)
		if err != nil {
			return err
		}
		if cycle == nil || cycle.Status != billingcycledomain.BillingCycleStatusClosed {
			return nil
		}
		return tx.WithContext(ctx).Exec(
			`UPDATE billing_cycles
			 SET invoiced_at = COALESCE(invoiced_at, ?),
			     last_error = NULL,
			     last_error_at = NULL,
			     updated_at = ?
			 WHERE id = ? AND status = ?`,
			now,
			now,
			cycleID,
			billingcycledomain.BillingCycleStatusClosed,
		).Error
	})
}

func (s *Scheduler) markCycleInvoiceFinalized(ctx context.Context, cycleID snowflake.ID, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cycle, err := s.lockCycleForUpdate(ctx, tx, cycleID)
		if err != nil {
			return err
		}
		if cycle == nil || cycle.Status != billingcycledomain.BillingCycleStatusClosed {
			return nil
		}
		return tx.WithContext(ctx).Exec(
			`UPDATE billing_cycles
			 SET invoice_finalized_at = COALESCE(invoice_finalized_at, ?),
			     updated_at = ?
			 WHERE id = ? AND status = ?`,
			now,
			now,
			cycleID,
			billingcycledomain.BillingCycleStatusClosed,
		).Error
	})
}

func (s *Scheduler) recordCycleErrorWithMetrics(ctx context.Context, cycleID snowflake.ID, stage string, err error) error {
	if err == nil {
		return nil
	}
	obsmetrics.Scheduler().IncBillingCycleError(stage, err)
	return s.recordCycleError(ctx, cycleID, err)
}

func (s *Scheduler) recordCycleError(ctx context.Context, cycleID snowflake.ID, err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	now := time.Now().UTC()
	if updateErr := s.db.WithContext(ctx).Exec(
		`UPDATE billing_cycles
		 SET last_error = ?, last_error_at = ?, updated_at = ?
		 WHERE id = ?`,
		message,
		now,
		now,
		cycleID,
	).Error; updateErr != nil {
		s.log.Warn("failed to record cycle error", zap.String("cycle_id", cycleID.String()), zap.Error(updateErr))
		return updateErr
	}
	return nil
}

func (s *Scheduler) hasRatingResults(ctx context.Context, cycleID snowflake.ID) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(1)
		 FROM rating_results
		 WHERE billing_cycle_id = ?`,
		cycleID,
	).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Scheduler) canEndSubscription(ctx context.Context, orgID, subscriptionID snowflake.ID) (bool, error) {
	var openCount int64
	if err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(1)
		 FROM billing_cycles
		 WHERE org_id = ? AND subscription_id = ? AND status IN (?, ?)`,
		orgID,
		subscriptionID,
		billingcycledomain.BillingCycleStatusOpen,
		billingcycledomain.BillingCycleStatusClosing,
	).Scan(&openCount).Error; err != nil {
		return false, err
	}
	if openCount > 0 {
		return false, nil
	}

	var invoiceCount int64
	if err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(1)
		 FROM billing_cycles bc
		 LEFT JOIN invoices i ON i.billing_cycle_id = bc.id
		 WHERE bc.org_id = ? AND bc.subscription_id = ? AND bc.status = ?
		   AND (i.id IS NULL OR i.status NOT IN (?, ?))`,
		orgID,
		subscriptionID,
		billingcycledomain.BillingCycleStatusClosed,
		invoicedomain.InvoiceStatusFinalized,
		invoicedomain.InvoiceStatusVoid,
	).Scan(&invoiceCount).Error; err != nil {
		return false, err
	}
	if invoiceCount > 0 {
		return false, nil
	}

	return true, nil
}
