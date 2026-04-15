package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiworkflow/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

func (r *repository) WithTx(tx *gorm.DB) domain.Repository {
	return &repository{db: tx}
}

func (r *repository) CreateWorkflow(ctx context.Context, workflow domain.Workflow, actions []domain.WorkflowActionRow) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&workflow).Error; err != nil {
			return err
		}
		if len(actions) > 0 {
			if err := tx.Create(&actions).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) UpdateWorkflow(ctx context.Context, orgID, workflowID uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Workflow{}).
		Where("id = ? AND org_id = ?", workflowID, orgID).
		Updates(updates).Error
}

func (r *repository) FindWorkflowByID(ctx context.Context, orgID, workflowID uuid.UUID) (*domain.Workflow, error) {
	var workflow domain.Workflow
	err := r.db.WithContext(ctx).
		Where("id = ? AND org_id = ?", workflowID, orgID).
		First(&workflow).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &workflow, nil
}

func (r *repository) ListWorkflows(ctx context.Context, orgID uuid.UUID, limit int, cursor *domain.WorkflowCursor) ([]*domain.Workflow, error) {
	var workflows []*domain.Workflow
	stmt := r.db.WithContext(ctx).Model(&domain.Workflow{}).Where("org_id = ?", orgID)

	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	if err := stmt.Order("created_at desc, id desc").Limit(limit).Find(&workflows).Error; err != nil {
		return nil, err
	}
	return workflows, nil
}

func (r *repository) ListWorkflowActions(ctx context.Context, workflowID uuid.UUID) ([]domain.WorkflowActionRow, error) {
	var actions []domain.WorkflowActionRow
	if err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("\"order\" asc, created_at asc").
		Find(&actions).Error; err != nil {
		return nil, err
	}
	return actions, nil
}

func (r *repository) ListWorkflowApprovals(ctx context.Context, workflowID uuid.UUID) ([]domain.WorkflowApprovalRow, error) {
	var approvals []domain.WorkflowApprovalRow
	if err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Order("created_at asc").
		Find(&approvals).Error; err != nil {
		return nil, err
	}
	return approvals, nil
}

func (r *repository) CreateApproval(ctx context.Context, approval domain.WorkflowApprovalRow) error {
	return r.db.WithContext(ctx).Create(&approval).Error
}

func (r *repository) UpdateAction(ctx context.Context, actionID uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.WorkflowActionRow{}).
		Where("id = ?", actionID).
		Updates(updates).Error
}

func (r *repository) CountActions(ctx context.Context, workflowIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int)
	if len(workflowIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		WorkflowID uuid.UUID `gorm:"column:workflow_id"`
		Count      int       `gorm:"column:count"`
	}
	if err := r.db.WithContext(ctx).
		Model(&domain.WorkflowActionRow{}).
		Select("workflow_id, count(*) as count").
		Where("workflow_id IN ?", workflowIDs).
		Group("workflow_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.WorkflowID] = row.Count
	}
	return counts, nil
}
