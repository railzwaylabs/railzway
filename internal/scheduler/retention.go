package scheduler

import (
	"context"

	paymentdomain "github.com/smallbiznis/railzway/internal/payment/domain"
	"go.uber.org/zap"
)

func (s *Scheduler) ResizeWebhookLogsJob(ctx context.Context) error {
	ctx, run, owner := s.ensureJobRun(ctx, "cleanup_webhook_logs", 1) // 1 batch, we just run a delete query
	if owner {
		s.logJobStart(ctx, run)
		defer s.logJobFinish(ctx, run)
	}

	retentionDays := s.cfg.WebhookRetentionDays
	if retentionDays <= 0 {
		s.log.Info("logging retention disabled or invalid", zap.Int("days", retentionDays))
		return nil
	}

	cutoff := s.clock.Now().AddDate(0, 0, -retentionDays)
	s.log.Info("cleaning up webhook logs", zap.Time("cutoff", cutoff))

	// Assuming we have a repository method or we use DB directly.
	// Since scheduler has access to DB, we can use it.
	// Target table: payment_event_records (based on domain exploration)
	
	result := s.db.WithContext(ctx).Delete(&paymentdomain.EventRecord{}, "received_at < ?", cutoff)
	if result.Error != nil {
		s.logSchedulerError(ctx, run, "scheduler.cleanup.failed", "cleanup_webhook_logs", 0, result.Error)
		return result.Error
	}

	deleted := int(result.RowsAffected)
	s.log.Info("cleanup webhook logs completed", zap.Int("deleted", deleted))
	run.AddProcessed(deleted)

	return nil
}
