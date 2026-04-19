package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusCancelled JobStatus = "cancelled"
)

type Job struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	OrgID        uuid.UUID       `gorm:"type:uuid;index" json:"org_id"`
	UserID       *uuid.UUID      `gorm:"type:uuid" json:"user_id,omitempty"`
	TaskType     string          `gorm:"type:varchar(100);index" json:"task_type"`
	Payload      json.RawMessage `gorm:"type:jsonb" json:"payload"`
	ScheduleCron string          `gorm:"type:varchar(100)" json:"schedule_cron,omitempty"`
	NextRunAt    time.Time       `gorm:"index" json:"next_run_at"`
	LastRunAt    *time.Time      `json:"last_run_at,omitempty"`
	ErrorCount   int             `gorm:"default:0" json:"error_count"`
	Status       JobStatus       `gorm:"type:varchar(20);index;default:'pending'" json:"status"`
	LastError    string          `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (Job) TableName() string {
	return "ai_scheduled_jobs"
}

type Service struct {
	db   *gorm.DB
	cron *cron.Parser
}

func NewService(db *gorm.DB) *Service {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	return &Service{
		db:   db,
		cron: &p,
	}
}

type CreateJobInput struct {
	OrgID        uuid.UUID
	UserID       *uuid.UUID
	TaskType     string
	Payload      interface{}
	ScheduleCron string
	RunAt        *time.Time
}

func (s *Service) CreateJob(ctx context.Context, input CreateJobInput) (*Job, error) {
	payloadJSON, err := json.Marshal(input.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal_payload: %w", err)
	}

	nextRun := time.Now().UTC()
	if input.RunAt != nil {
		nextRun = *input.RunAt
	} else if input.ScheduleCron != "" {
		sch, err := s.cron.Parse(input.ScheduleCron)
		if err != nil {
			return nil, fmt.Errorf("invalid_cron: %w", err)
		}
		nextRun = sch.Next(nextRun)
	}

	job := &Job{
		ID:           uuid.New(),
		OrgID:        input.OrgID,
		UserID:       input.UserID,
		TaskType:     input.TaskType,
		Payload:      payloadJSON,
		ScheduleCron: input.ScheduleCron,
		NextRunAt:    nextRun,
		Status:       StatusPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, err
	}

	return job, nil
}

func (s *Service) ListJobs(ctx context.Context, orgID uuid.UUID, status string, pageSize int, offset int) ([]Job, int64, error) {
	var jobs []Job
	var total int64
	query := s.db.WithContext(ctx).Model(&Job{}).Where("org_id = ?", orgID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (s *Service) RetryJob(ctx context.Context, orgID uuid.UUID, jobID uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&Job{}).
		Where("id = ? AND org_id = ?", jobID, orgID).
		Updates(map[string]interface{}{
			"status":      StatusPending,
			"next_run_at": time.Now().UTC(),
			"updated_at":  time.Now().UTC(),
		}).Error
}

func (s *Service) CancelJob(ctx context.Context, orgID uuid.UUID, jobID uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&Job{}).
		Where("id = ? AND org_id = ?", jobID, orgID).
		Updates(map[string]interface{}{
			"status":     StatusCancelled,
			"updated_at": time.Now().UTC(),
		}).Error
}

func (s *Service) GetJob(ctx context.Context, orgID uuid.UUID, jobID uuid.UUID) (*Job, error) {
	var job Job
	if err := s.db.WithContext(ctx).Where("id = ? AND org_id = ?", jobID, orgID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("job_not_found")
		}
		return nil, err
	}
	return &job, nil
}
