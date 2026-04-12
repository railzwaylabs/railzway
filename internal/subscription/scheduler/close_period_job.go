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

func (j *ClosePeriodJob) Run(ctx context.Context, asOf time.Time, batchSize int) (ClosePeriodResult, error) {
	periods, err := j.repo.ListOpenSubscriptionPeriods(ctx, asOf, batchSize)
	if err != nil {
		return ClosePeriodResult{}, err
	}

	result := ClosePeriodResult{}
	for _, period := range periods {
		generated, err := j.closeOne(ctx, period, asOf)
		if err != nil {
			return result, err
		}
		result.ClosedPeriods++
		if generated {
			result.GeneratedInvoices++
		}
	}

	return result, nil
}

func (j *ClosePeriodJob) closeOne(ctx context.Context, period subscriptiondomain.SubscriptionPeriod, asOf time.Time) (bool, error) {
	generated := false
	err := j.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := j.repo.WithTx(tx)

		lockedPeriod, err := repo.FindSubscriptionPeriodForUpdate(ctx, period.OrgID, period.ID)
		if err != nil {
			return err
		}
		if lockedPeriod == nil {
			return nil
		}
		if lockedPeriod.Status != subscriptiondomain.PeriodStatusOpen || lockedPeriod.PeriodEnd.After(asOf) {
			return nil
		}
		period = *lockedPeriod

		sub, err := repo.FindSubscriptionByID(ctx, period.OrgID, period.SubscriptionID)
		if err != nil {
			return err
		}
		if sub == nil {
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
				return err
			}
			sub.Status = subscriptiondomain.StatusActive
		}

		shouldInvoice := true
		if sub.Status == subscriptiondomain.StatusCanceled || sub.Status == subscriptiondomain.StatusPaused {
			shouldInvoice = false
		}
		if sub.Status == subscriptiondomain.StatusTrialing && sub.TrialEnd != nil && sub.TrialEnd.After(period.PeriodEnd) {
			shouldInvoice = false
		}
		if sub.CancelAt != nil && !sub.CancelAt.After(period.PeriodStart) {
			shouldInvoice = false
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
					return err
				}
				return err
			}
			if invResp.Invoice.Status == invoicedomain.StatusDraft && shouldOpenInvoice(asOfUTC, period.PeriodEnd, j.gracePeriod) {
				_, err = j.invoiceSvc.OpenInvoice(orgCtx, invoicedomain.OpenInvoiceRequest{
					ID: invResp.Invoice.ID.String(),
				})
				if err != nil {
					return err
				}
			}
			generated = true
		}

		if err := repo.UpdateSubscriptionPeriod(ctx, period.OrgID, period.ID, map[string]interface{}{
			"status":     subscriptiondomain.PeriodStatusClosed,
			"updated_at": asOfUTC,
		}); err != nil {
			return err
		}

		if sub.CancelAt != nil && !sub.CancelAt.After(asOfUTC) {
			if err := repo.UpdateSubscription(ctx, period.OrgID, sub.ID, map[string]interface{}{
				"status":      subscriptiondomain.StatusCanceled,
				"canceled_at": asOfUTC,
				"ended_at":    asOfUTC,
				"updated_at":  asOfUTC,
			}); err != nil {
				return err
			}
			return nil
		}

		if !shouldAdvanceSubscription(sub.Status) {
			return nil
		}

		if !sub.CurrentPeriodStart.Equal(period.PeriodStart) || !sub.CurrentPeriodEnd.Equal(period.PeriodEnd) {
			return nil
		}

		nextStart, nextEnd, ok := nextPeriod(period.PeriodStart, period.PeriodEnd)
		if !ok {
			return nil
		}

		if err := repo.UpdateSubscription(ctx, period.OrgID, sub.ID, map[string]interface{}{
			"current_period_start": nextStart,
			"current_period_end":   nextEnd,
			"updated_at":           asOf.UTC(),
		}); err != nil {
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
				return nil
			}
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, invoicedomain.ErrUsageNotReady) {
			return false, nil
		}
		return false, err
	}
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
