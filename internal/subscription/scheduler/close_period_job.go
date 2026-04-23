package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"github.com/railzwaylabs/railzway/internal/telemetry"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ClosePeriodJob struct {
	db          *gorm.DB
	repo        subscriptiondomain.Repository
	invoiceSvc  invoicedomain.Service
	gracePeriod time.Duration
}

type ClosePeriodResult struct {
	ClosedPeriods     int
	GeneratedInvoices int
}

func NewClosePeriodJob(db *gorm.DB, repo subscriptiondomain.Repository, invoiceSvc invoicedomain.Service, gracePeriod time.Duration) *ClosePeriodJob {
	return &ClosePeriodJob{
		db:          db,
		repo:        repo,
		invoiceSvc:  invoiceSvc,
		gracePeriod: gracePeriod,
	}
}

func (j *ClosePeriodJob) Run(ctx context.Context, asOf time.Time, batchSize int) (result ClosePeriodResult, err error) {
	startedAt := time.Now()
	ctx, span := telemetry.StartSpan(
		ctx,
		"subscription.close_period.run",
		telemetry.StringAttr("billing.as_of", asOf.UTC().Format(time.RFC3339)),
		telemetry.Int64Attr("billing.batch_size", int64(batchSize)),
	)
	defer func() {
		span.SetAttributes(
			telemetry.Int64Attr("billing.closed_periods", int64(result.ClosedPeriods)),
			telemetry.Int64Attr("billing.generated_invoices", int64(result.GeneratedInvoices)),
		)
		telemetry.EndSpan(span, err)
		telemetry.ObserveOperation("subscription.close_period.run", time.Since(startedAt), err)
	}()

	logger := zap.L().With(
		zap.String("job", "subscription.close_period"),
		zap.Time("as_of", asOf.UTC()),
		zap.Int("batch_size", batchSize),
	)
	logger.Info("subscription close period run started")

	periods, err := j.repo.ListOpenSubscriptionPeriods(ctx, asOf, batchSize)
	if err != nil {
		logger.Warn("subscription close period period listing failed", zap.Error(err))
		return result, err
	}
	logger.Info("subscription close period periods listed", zap.Int("period_count", len(periods)))

	for _, period := range periods {
		generated, err := j.closeOne(ctx, period, asOf)
		if err != nil {
			logger.Warn("subscription close period run stopped after period failure",
				zap.String("org_id", period.OrgID.String()),
				zap.String("subscription_id", period.SubscriptionID.String()),
				zap.String("period_id", period.ID.String()),
				zap.Error(err),
			)
			return result, err
		}
		result.ClosedPeriods++
		if generated {
			result.GeneratedInvoices++
		}
	}

	logger.Info("subscription close period run completed",
		zap.Int("closed_periods", result.ClosedPeriods),
		zap.Int("generated_invoices", result.GeneratedInvoices),
	)
	return result, nil
}

func (j *ClosePeriodJob) RunForTestClock(ctx context.Context, orgID, testClockID uuid.UUID, asOf time.Time, batchSize int) (result ClosePeriodResult, err error) {
	startedAt := time.Now()
	ctx, span := telemetry.StartSpan(
		ctx,
		"subscription.close_period.run_for_test_clock",
		telemetry.UUIDAttr("billing.org_id", orgID),
		telemetry.UUIDAttr("billing.test_clock_id", testClockID),
		telemetry.StringAttr("billing.as_of", asOf.UTC().Format(time.RFC3339)),
		telemetry.Int64Attr("billing.batch_size", int64(batchSize)),
	)
	defer func() {
		span.SetAttributes(
			telemetry.Int64Attr("billing.closed_periods", int64(result.ClosedPeriods)),
			telemetry.Int64Attr("billing.generated_invoices", int64(result.GeneratedInvoices)),
		)
		telemetry.EndSpan(span, err)
		telemetry.ObserveOperation("subscription.close_period.run_for_test_clock", time.Since(startedAt), err)
	}()

	logger := zap.L().With(
		zap.String("job", "subscription.close_period"),
		zap.String("org_id", orgID.String()),
		zap.String("test_clock_id", testClockID.String()),
		zap.Time("as_of", asOf.UTC()),
		zap.Int("batch_size", batchSize),
	)
	logger.Info("test clock close period simulation started")

	periods, err := j.repo.ListOpenSubscriptionPeriodsByTestClock(ctx, orgID, testClockID, asOf, batchSize)
	if err != nil {
		logger.Warn("test clock close period period listing failed", zap.Error(err))
		return result, err
	}
	logger.Info("test clock close period periods listed", zap.Int("period_count", len(periods)))

	for _, period := range periods {
		generated, err := j.closeOne(ctx, period, asOf)
		if err != nil {
			logger.Warn("test clock close period simulation stopped after period failure",
				zap.String("subscription_id", period.SubscriptionID.String()),
				zap.String("period_id", period.ID.String()),
				zap.Error(err),
			)
			return result, err
		}
		result.ClosedPeriods++
		if generated {
			result.GeneratedInvoices++
		}
	}

	logger.Info("test clock close period simulation completed",
		zap.Int("closed_periods", result.ClosedPeriods),
		zap.Int("generated_invoices", result.GeneratedInvoices),
	)
	return result, nil
}

func (j *ClosePeriodJob) closeOne(ctx context.Context, period subscriptiondomain.SubscriptionPeriod, asOf time.Time) (generated bool, err error) {
	startedAt := time.Now()
	ctx, span := telemetry.StartSpan(
		ctx,
		"subscription.close_period.close_one",
		telemetry.UUIDAttr("billing.org_id", period.OrgID),
		telemetry.UUIDAttr("billing.subscription_id", period.SubscriptionID),
		telemetry.UUIDAttr("billing.subscription_period_id", period.ID),
		telemetry.StringAttr("billing.as_of", asOf.UTC().Format(time.RFC3339)),
		telemetry.StringAttr("billing.period_start", period.PeriodStart.UTC().Format(time.RFC3339)),
		telemetry.StringAttr("billing.period_end", period.PeriodEnd.UTC().Format(time.RFC3339)),
	)
	defer func() {
		span.SetAttributes(telemetry.BoolAttr("billing.generated_invoice", generated))
		telemetry.EndSpan(span, err)
		telemetry.ObserveOperation("subscription.close_period.close_one", time.Since(startedAt), err)
	}()

	logger := zap.L().With(
		zap.String("job", "subscription.close_period"),
		zap.String("org_id", period.OrgID.String()),
		zap.String("subscription_id", period.SubscriptionID.String()),
		zap.String("period_id", period.ID.String()),
		zap.Time("period_start", period.PeriodStart.UTC()),
		zap.Time("period_end", period.PeriodEnd.UTC()),
		zap.Time("as_of", asOf.UTC()),
	)
	logger.Info("subscription period close started")

	err = j.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := j.repo.WithTx(tx)

		lockedPeriod, err := repo.FindSubscriptionPeriodForUpdate(ctx, period.OrgID, period.ID)
		if err != nil {
			logger.Warn("subscription period lock failed", zap.Error(err))
			return err
		}
		if lockedPeriod == nil {
			span.SetAttributes(telemetry.StringAttr("billing.skip_reason", "period_not_found"))
			logger.Info("subscription period close skipped", zap.String("reason", "period_not_found"))
			return nil
		}
		if lockedPeriod.Status != subscriptiondomain.PeriodStatusOpen || lockedPeriod.PeriodEnd.After(asOf) {
			span.SetAttributes(telemetry.StringAttr("billing.skip_reason", "period_not_due"))
			logger.Info("subscription period close skipped",
				zap.String("reason", "period_not_due"),
				zap.String("status", lockedPeriod.Status),
				zap.Time("locked_period_end", lockedPeriod.PeriodEnd.UTC()),
			)
			return nil
		}
		period = *lockedPeriod

		sub, err := repo.FindSubscriptionByID(ctx, period.OrgID, period.SubscriptionID)
		if err != nil {
			logger.Warn("subscription lookup failed", zap.Error(err))
			return err
		}
		if sub == nil {
			span.SetAttributes(telemetry.StringAttr("billing.skip_reason", "subscription_not_found"))
			logger.Info("subscription missing; closing orphaned subscription period")
			return repo.UpdateSubscriptionPeriod(ctx, period.OrgID, period.ID, map[string]interface{}{
				"status":     subscriptiondomain.PeriodStatusClosed,
				"updated_at": asOf.UTC(),
			})
		}

		asOfUTC := asOf.UTC()
		if sub.Status == subscriptiondomain.StatusTrialing && sub.TrialEnd != nil && !sub.TrialEnd.After(asOfUTC) {
			if err := repo.UpdateSubscription(ctx, period.OrgID, sub.ID, map[string]interface{}{
				"status":     subscriptiondomain.StatusActive,
				"updated_at": asOfUTC,
			}); err != nil {
				logger.Warn("subscription trial transition failed", zap.Error(err))
				return err
			}
			logger.Info("subscription transitioned from trialing to active", zap.Time("trial_end", sub.TrialEnd.UTC()))
			sub.Status = subscriptiondomain.StatusActive
		}

		shouldInvoice := true
		if sub.Status == subscriptiondomain.StatusCanceled || sub.Status == subscriptiondomain.StatusPaused {
			shouldInvoice = false
			span.SetAttributes(telemetry.StringAttr("billing.invoice_skip_reason", "subscription_not_billable_status"))
		}
		if sub.Status == subscriptiondomain.StatusTrialing && sub.TrialEnd != nil && sub.TrialEnd.After(period.PeriodEnd) {
			shouldInvoice = false
			span.SetAttributes(telemetry.StringAttr("billing.invoice_skip_reason", "trial_covers_period"))
		}
		if sub.CancelAt != nil && !sub.CancelAt.After(period.PeriodStart) {
			shouldInvoice = false
			span.SetAttributes(telemetry.StringAttr("billing.invoice_skip_reason", "cancel_at_before_period"))
		}

		if shouldInvoice {
			orgCtx := orgcontext.WithOrgID(ctx, period.OrgID)
			idempotencyKey := buildInvoiceIdempotencyKey(sub.ID, period.PeriodStart, period.PeriodEnd)
			invResp, err := j.invoiceSvc.GenerateInvoice(orgCtx, invoicedomain.GenerateInvoiceRequest{
				SubscriptionID: sub.ID.String(),
				PeriodStart:    period.PeriodStart,
				PeriodEnd:      period.PeriodEnd,
				IssueAt:        &asOf,
				IdempotencyKey: idempotencyKey,
			})
			if err != nil {
				if errors.Is(err, invoicedomain.ErrUsageNotReady) {
					span.SetAttributes(telemetry.StringAttr("billing.skip_reason", "usage_not_ready"))
					logger.Info("subscription period close deferred", zap.String("reason", "usage_not_ready"))
					return err
				}
				if errors.Is(err, invoicedomain.ErrNoBillableItems) {
					shouldInvoice = false
					span.SetAttributes(telemetry.StringAttr("billing.invoice_skip_reason", "no_billable_items"))
					logger.Info("invoice generation skipped", zap.String("reason", "no_billable_items"))
				} else {
					logger.Warn("invoice generation failed", zap.Error(err))
					return err
				}
			}
			if shouldInvoice && invResp.Invoice.Status == invoicedomain.StatusDraft && shouldOpenInvoice(asOfUTC, period.PeriodEnd, j.gracePeriod) {
				_, err = j.invoiceSvc.OpenInvoice(orgCtx, invoicedomain.OpenInvoiceRequest{
					ID: invResp.Invoice.ID.String(),
				})
				if err != nil {
					logger.Warn("invoice open failed", zap.String("invoice_id", invResp.Invoice.ID.String()), zap.Error(err))
					return err
				}
				logger.Info("invoice opened",
					zap.String("invoice_id", invResp.Invoice.ID.String()),
					zap.Int64("total_cents", invResp.Invoice.TotalCents),
				)
			}
			if shouldInvoice {
				generated = true
				span.SetAttributes(
					telemetry.UUIDAttr("billing.invoice_id", invResp.Invoice.ID),
					telemetry.Int64Attr("billing.invoice_total_cents", invResp.Invoice.TotalCents),
				)
				logger.Info("invoice generated",
					zap.String("invoice_id", invResp.Invoice.ID.String()),
					zap.String("invoice_status", invResp.Invoice.Status),
					zap.Int64("total_cents", invResp.Invoice.TotalCents),
				)
			}
		}

		if err := repo.UpdateSubscriptionPeriod(ctx, period.OrgID, period.ID, map[string]interface{}{
			"status":     subscriptiondomain.PeriodStatusClosed,
			"updated_at": asOfUTC,
		}); err != nil {
			logger.Warn("subscription period close update failed", zap.Error(err))
			return err
		}
		logger.Info("subscription period closed")

		if sub.CancelAt != nil && !sub.CancelAt.After(asOfUTC) {
			if err := repo.UpdateSubscription(ctx, period.OrgID, sub.ID, map[string]interface{}{
				"status":      subscriptiondomain.StatusCanceled,
				"canceled_at": asOfUTC,
				"ended_at":    asOfUTC,
				"updated_at":  asOfUTC,
			}); err != nil {
				logger.Warn("subscription cancel transition failed", zap.Error(err))
				return err
			}
			logger.Info("subscription canceled at period close", zap.Time("cancel_at", sub.CancelAt.UTC()))
			return nil
		}

		if !shouldAdvanceSubscription(sub.Status) {
			span.SetAttributes(telemetry.StringAttr("billing.advance_skip_reason", "status_not_advanceable"))
			logger.Info("subscription advance skipped", zap.String("reason", "status_not_advanceable"), zap.String("status", sub.Status))
			return nil
		}

		if !sub.CurrentPeriodStart.Equal(period.PeriodStart) || !sub.CurrentPeriodEnd.Equal(period.PeriodEnd) {
			span.SetAttributes(telemetry.StringAttr("billing.advance_skip_reason", "current_period_mismatch"))
			logger.Info("subscription advance skipped",
				zap.String("reason", "current_period_mismatch"),
				zap.Time("subscription_current_period_start", sub.CurrentPeriodStart.UTC()),
				zap.Time("subscription_current_period_end", sub.CurrentPeriodEnd.UTC()),
			)
			return nil
		}

		nextStart, nextEnd, ok := nextPeriod(period.PeriodStart, period.PeriodEnd)
		if !ok {
			span.SetAttributes(telemetry.StringAttr("billing.advance_skip_reason", "next_period_unresolved"))
			logger.Info("subscription advance skipped", zap.String("reason", "next_period_unresolved"))
			return nil
		}

		if err := repo.UpdateSubscription(ctx, period.OrgID, sub.ID, map[string]interface{}{
			"current_period_start": nextStart,
			"current_period_end":   nextEnd,
			"updated_at":           asOf.UTC(),
		}); err != nil {
			logger.Warn("subscription period advance failed", zap.Error(err))
			return err
		}

		nextPeriod := subscriptiondomain.SubscriptionPeriod{
			ID:             uuid.New(),
			OrgID:          period.OrgID,
			SubscriptionID: sub.ID,
			Status:         subscriptiondomain.PeriodStatusOpen,
			PeriodStart:    nextStart,
			PeriodEnd:      nextEnd,
			CreatedAt:      asOf.UTC(),
			UpdatedAt:      asOf.UTC(),
		}
		if err := repo.CreateSubscriptionPeriod(ctx, nextPeriod); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				logger.Info("next subscription period already exists", zap.String("next_period_id", nextPeriod.ID.String()))
				return nil
			}
			logger.Warn("next subscription period create failed", zap.Error(err))
			return err
		}
		span.SetAttributes(
			telemetry.UUIDAttr("billing.next_period_id", nextPeriod.ID),
			telemetry.StringAttr("billing.next_period_start", nextStart.UTC().Format(time.RFC3339)),
			telemetry.StringAttr("billing.next_period_end", nextEnd.UTC().Format(time.RFC3339)),
		)
		logger.Info("subscription advanced to next period",
			zap.String("next_period_id", nextPeriod.ID.String()),
			zap.Time("next_period_start", nextStart.UTC()),
			zap.Time("next_period_end", nextEnd.UTC()),
		)

		return nil
	})
	if err != nil {
		if errors.Is(err, invoicedomain.ErrUsageNotReady) {
			logger.Info("subscription period close deferred", zap.String("reason", "usage_not_ready"))
			return false, nil
		}
		logger.Warn("subscription period close failed", zap.Error(err))
		return false, err
	}
	logger.Info("subscription period close completed", zap.Bool("generated_invoice", generated))
	return generated, nil
}

func shouldAdvanceSubscription(status string) bool {
	switch status {
	case subscriptiondomain.StatusActive, subscriptiondomain.StatusTrialing, subscriptiondomain.StatusPastDue:
		return true
	default:
		return false
	}
}

func nextPeriod(start, end time.Time) (time.Time, time.Time, bool) {
	if end.Before(start) {
		return time.Time{}, time.Time{}, false
	}

	for count := 1; count <= 12; count++ {
		if end.Equal(start.AddDate(0, count, 0)) {
			nextStart := end
			return nextStart, nextStart.AddDate(0, count, 0), true
		}
	}
	for count := 1; count <= 5; count++ {
		if end.Equal(start.AddDate(count, 0, 0)) {
			nextStart := end
			return nextStart, nextStart.AddDate(count, 0, 0), true
		}
	}
	for count := 1; count <= 31; count++ {
		if end.Equal(start.AddDate(0, 0, count)) {
			nextStart := end
			return nextStart, nextStart.AddDate(0, 0, count), true
		}
	}

	duration := end.Sub(start)
	if duration <= 0 {
		return time.Time{}, time.Time{}, false
	}
	nextStart := end
	return nextStart, nextStart.Add(duration), true
}

func buildInvoiceIdempotencyKey(subscriptionID uuid.UUID, start, end time.Time) string {
	return fmt.Sprintf("period:%s:%s:%s", subscriptionID.String(), start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
}

func shouldOpenInvoice(asOf, periodEnd time.Time, grace time.Duration) bool {
	if grace <= 0 {
		return true
	}
	return !asOf.Before(periodEnd.Add(grace))
}
