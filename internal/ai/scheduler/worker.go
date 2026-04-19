package scheduler

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TaskHandler func(ctx context.Context, payload json.RawMessage) error

type Worker struct {
	db       *gorm.DB
	logger   *zap.Logger
	service  *Service
	handlers map[string]TaskHandler
	mu       sync.RWMutex
}

func NewWorker(db *gorm.DB, logger *zap.Logger, service *Service) *Worker {
	return &Worker{
		db:       db,
		logger:   logger,
		service:  service,
		handlers: make(map[string]TaskHandler),
	}
}

func (w *Worker) RegisterHandler(taskType string, handler TaskHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[taskType] = handler
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	w.logger.Info("AI Assistant Job Worker started")

	go func() {
		defer ticker.Stop()
		for {
			w.processJobs(ctx)

			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				w.logger.Info("AI Assistant Job Worker stopping")
				return
			}
		}
	}()
}

func (w *Worker) processJobs(ctx context.Context) {
	var jobs []Job
	now := time.Now().UTC()

	// Find pending jobs that are due
	if err := w.db.WithContext(ctx).
		Where("status IN (?) AND next_run_at <= ?", []JobStatus{StatusPending, StatusFailed}, now).
		Limit(10).
		Find(&jobs).Error; err != nil {
		w.logger.Error("failed to fetch pending jobs", zap.Error(err))
		return
	}

	for _, job := range jobs {
		w.runJob(ctx, job)
	}
}

func (w *Worker) runJob(ctx context.Context, job Job) {
	w.mu.RLock()
	handler, ok := w.handlers[job.TaskType]
	w.mu.RUnlock()

	if !ok {
		w.logger.Warn("no handler registered for task type", zap.String("task_type", job.TaskType))
		return
	}

	// Update to running
	if err := w.db.WithContext(ctx).Model(&job).
		Where("status = ?", job.Status).
		Updates(map[string]interface{}{
			"status":      StatusRunning,
			"last_run_at": time.Now().UTC(),
			"updated_at":  time.Now().UTC(),
		}).Error; err != nil {
		return // Likely picked up by another worker
	}

	w.logger.Info("running job", zap.String("job_id", job.ID.String()), zap.String("task_type", job.TaskType))

	// Setup context with org_id
	jobCtx := orgcontext.WithOrgID(ctx, job.OrgID)
	
	err := handler(jobCtx, job.Payload)
	
	status := StatusCompleted
	lastError := ""
	errorCount := job.ErrorCount
	
	if err != nil {
		status = StatusFailed
		lastError = err.Error()
		errorCount++
		w.logger.Error("job execution failed", zap.String("job_id", job.ID.String()), zap.Error(err))
	}

	updates := map[string]interface{}{
		"status":      status,
		"last_error":  lastError,
		"error_count": errorCount,
		"updated_at":  time.Now().UTC(),
	}

	// If it's a recurring job and it succeeded (or we want to retry failed ones on schedule)
	if job.ScheduleCron != "" && (status == StatusCompleted || errorCount < 5) {
		sch, _ := w.service.cron.Parse(job.ScheduleCron)
		updates["next_run_at"] = sch.Next(time.Now().UTC())
		updates["status"] = StatusPending // Reset to pending for next run
	}

	if err := w.db.WithContext(ctx).Model(&job).Updates(updates).Error; err != nil {
		w.logger.Error("failed to update job status", zap.String("job_id", job.ID.String()), zap.Error(err))
	}
}
