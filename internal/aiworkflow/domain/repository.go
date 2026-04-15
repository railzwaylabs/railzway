package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkflowCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type Repository interface {
	WithTx(tx *gorm.DB) Repository
	CreateWorkflow(ctx context.Context, workflow Workflow, actions []WorkflowActionRow) error
	UpdateWorkflow(ctx context.Context, orgID, workflowID uuid.UUID, updates map[string]interface{}) error
	FindWorkflowByID(ctx context.Context, orgID, workflowID uuid.UUID) (*Workflow, error)
	ListWorkflows(ctx context.Context, orgID uuid.UUID, limit int, cursor *WorkflowCursor) ([]*Workflow, error)
	ListWorkflowActions(ctx context.Context, workflowID uuid.UUID) ([]WorkflowActionRow, error)
	ListWorkflowApprovals(ctx context.Context, workflowID uuid.UUID) ([]WorkflowApprovalRow, error)
	CreateApproval(ctx context.Context, approval WorkflowApprovalRow) error
	UpdateAction(ctx context.Context, actionID uuid.UUID, updates map[string]interface{}) error
	CountActions(ctx context.Context, workflowIDs []uuid.UUID) (map[uuid.UUID]int, error)
}
