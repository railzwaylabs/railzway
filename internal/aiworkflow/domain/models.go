package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

type WorkflowStatus string

type ActionStatus string

type ActionType string

type ApprovalStatus string

const (
	WorkflowDraft          WorkflowStatus = "draft"
	WorkflowPendingApproval WorkflowStatus = "pending_approval"
	WorkflowApproved       WorkflowStatus = "approved"
	WorkflowRejected       WorkflowStatus = "rejected"
	WorkflowExecuting      WorkflowStatus = "executing"
	WorkflowCompleted      WorkflowStatus = "completed"
	WorkflowFailed         WorkflowStatus = "failed"
)

const (
	ActionPending ActionStatus = "pending"
	ActionRunning ActionStatus = "running"
	ActionDone    ActionStatus = "done"
	ActionFailed  ActionStatus = "failed"
)

const (
	ActionNavigate     ActionType = "navigate"
	ActionCreateProduct ActionType = "create_product"
	ActionCreatePlan    ActionType = "create_plan"
	ActionCreateMeter   ActionType = "create_meter"
	ActionCreateFeature ActionType = "create_feature"
	ActionNotify        ActionType = "notify"
)

const (
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidID           = errors.New("invalid_id")
	ErrInvalidTitle        = errors.New("invalid_title")
	ErrInvalidActions      = errors.New("invalid_actions")
	ErrInvalidStatus       = errors.New("invalid_status")
	ErrNotFound            = errors.New("not_found")
)

type CreateWorkflowRequest struct {
	Title       string
	Summary     string
	Intent      string
	SourceRunID string
	Actions     []CreateWorkflowAction
}

type CreateWorkflowAction struct {
	Type    ActionType
	Label   string
	Payload json.RawMessage
}

type ApproveWorkflowRequest struct {
	ID      string
	ActorID string
	Note    string
	Status  ApprovalStatus
}

type ExecuteWorkflowRequest struct {
	ID string
}

type GetWorkflowRequest struct {
	ID string
}

type ListWorkflowsRequest struct {
	PageToken string
	PageSize  int32
}

type ListWorkflowsResponse struct {
	pagination.PageInfo
	Workflows []WorkflowListItem `json:"workflows"`
}

type WorkflowDetailResponse struct {
	Workflow WorkflowDetail `json:"workflow"`
}

type WorkflowListItem struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Intent    string        `json:"intent"`
	Summary   string        `json:"summary"`
	Status    StatusBadge   `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Actions   int           `json:"actions"`
	SourceRunID string      `json:"source_run_id,omitempty"`
}

type WorkflowDetail struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Intent      string            `json:"intent"`
	Summary     string            `json:"summary"`
	Status      StatusBadge       `json:"status"`
	SourceRunID string            `json:"source_run_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	FinishedAt  *time.Time        `json:"finished_at,omitempty"`
	Actions     []WorkflowAction  `json:"actions"`
	Approvals   []WorkflowApproval `json:"approvals"`
}

type WorkflowAction struct {
	ID        string          `json:"id"`
	Type      ActionType      `json:"type"`
	Label     string          `json:"label"`
	Status    ActionStatus    `json:"status"`
	Payload   json.RawMessage `json:"payload"`
	Order     int             `json:"order"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type WorkflowApproval struct {
	ID        string         `json:"id"`
	ActorID   string         `json:"actor_id"`
	Status    ApprovalStatus `json:"status"`
	Note      string         `json:"note,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type StatusBadge struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Tone  string `json:"tone"`
}

type Workflow struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	OrgID       uuid.UUID       `gorm:"type:uuid;not null;index" json:"org_id"`
	Title       string          `gorm:"type:text;not null" json:"title"`
	Summary     string          `gorm:"type:text" json:"summary"`
	Intent      string          `gorm:"type:text;not null" json:"intent"`
	Status      WorkflowStatus  `gorm:"type:text;not null" json:"status"`
	SourceRunID *uuid.UUID      `gorm:"type:uuid" json:"source_run_id"`
	CreatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	StartedAt   *time.Time      `gorm:"" json:"started_at"`
	FinishedAt  *time.Time      `gorm:"" json:"finished_at"`
}

func (Workflow) TableName() string { return "ai_workflows" }

type WorkflowActionRow struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	WorkflowID uuid.UUID       `gorm:"type:uuid;not null;index" json:"workflow_id"`
	Type       ActionType      `gorm:"type:text;not null" json:"type"`
	Label      string          `gorm:"type:text;not null" json:"label"`
	Status     ActionStatus    `gorm:"type:text;not null" json:"status"`
	Payload    json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`
	Order      int             `gorm:"not null" json:"order"`
	Error      string          `gorm:"type:text" json:"error"`
	CreatedAt  time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (WorkflowActionRow) TableName() string { return "ai_workflow_actions" }

type WorkflowApprovalRow struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	WorkflowID uuid.UUID      `gorm:"type:uuid;not null;index" json:"workflow_id"`
	ActorID    string         `gorm:"type:text;not null" json:"actor_id"`
	Status     ApprovalStatus `gorm:"type:text;not null" json:"status"`
	Note       string         `gorm:"type:text" json:"note"`
	CreatedAt  time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (WorkflowApprovalRow) TableName() string { return "ai_workflow_approvals" }
