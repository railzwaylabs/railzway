package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiworkflow/domain"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type service struct {
	repo     domain.Repository
	executor actionExecutor
}

func NewService(repo domain.Repository, executor actionExecutor) domain.Service {
	return &service{repo: repo, executor: executor}
}

func (s *service) CreateWorkflow(ctx context.Context, req domain.CreateWorkflowRequest) (domain.WorkflowDetailResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidOrganization
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidTitle
	}

	actions := req.Actions
	if len(actions) == 0 {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidActions
	}

	for _, action := range actions {
		if !isValidActionType(action.Type) {
			return domain.WorkflowDetailResponse{}, domain.ErrInvalidActions
		}
		if strings.TrimSpace(action.Label) == "" {
			return domain.WorkflowDetailResponse{}, domain.ErrInvalidActions
		}
	}

	var sourceRunID *uuid.UUID
	if strings.TrimSpace(req.SourceRunID) != "" {
		parsed, err := uuid.Parse(req.SourceRunID)
		if err != nil || parsed == uuid.Nil {
			return domain.WorkflowDetailResponse{}, domain.ErrInvalidID
		}
		sourceRunID = &parsed
	}

	now := time.Now().UTC()
	workflow := domain.Workflow{
		ID:          uuid.New(),
		OrgID:       orgID,
		Title:       title,
		Summary:     strings.TrimSpace(req.Summary),
		Intent:      strings.TrimSpace(req.Intent),
		Status:      domain.WorkflowPendingApproval,
		SourceRunID: sourceRunID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	actionRows := make([]domain.WorkflowActionRow, 0, len(actions))
	for idx, action := range actions {
		payload := normalizePayload(action.Payload)
		actionRows = append(actionRows, domain.WorkflowActionRow{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Type:       action.Type,
			Label:      strings.TrimSpace(action.Label),
			Status:     domain.ActionPending,
			Payload:    payload,
			Order:      idx + 1,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	if err := s.repo.CreateWorkflow(ctx, workflow, actionRows); err != nil {
		return domain.WorkflowDetailResponse{}, err
	}

	return domain.WorkflowDetailResponse{Workflow: toWorkflowDetail(workflow, actionRows, nil)}, nil
}

func (s *service) GetWorkflow(ctx context.Context, req domain.GetWorkflowRequest) (domain.WorkflowDetailResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidOrganization
	}

	workflowID, err := uuid.Parse(strings.TrimSpace(req.ID))
	if err != nil || workflowID == uuid.Nil {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidID
	}

	workflow, err := s.repo.FindWorkflowByID(ctx, orgID, workflowID)
	if err != nil {
		return domain.WorkflowDetailResponse{}, err
	}
	if workflow == nil {
		return domain.WorkflowDetailResponse{}, domain.ErrNotFound
	}

	actions, err := s.repo.ListWorkflowActions(ctx, workflowID)
	if err != nil {
		return domain.WorkflowDetailResponse{}, err
	}
	approvals, err := s.repo.ListWorkflowApprovals(ctx, workflowID)
	if err != nil {
		return domain.WorkflowDetailResponse{}, err
	}

	return domain.WorkflowDetailResponse{Workflow: toWorkflowDetail(*workflow, actions, approvals)}, nil
}

func (s *service) ListWorkflows(ctx context.Context, req domain.ListWorkflowsRequest) (domain.ListWorkflowsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListWorkflowsResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	var cursor *domain.WorkflowCursor
	if req.PageToken != "" {
		decoded, err := pagination.DecodeCursor(req.PageToken)
		if err != nil {
			return domain.ListWorkflowsResponse{}, domain.ErrInvalidID
		}
		if decoded != nil && decoded.CreatedAt != "" && decoded.ID != "" {
			parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
			if err != nil {
				return domain.ListWorkflowsResponse{}, domain.ErrInvalidID
			}
			parsedID, err := uuid.Parse(decoded.ID)
			if err != nil {
				return domain.ListWorkflowsResponse{}, domain.ErrInvalidID
			}
			cursor = &domain.WorkflowCursor{ID: parsedID, CreatedAt: parsedTime}
		}
	}

	items, err := s.repo.ListWorkflows(ctx, orgID, pageSize+1, cursor)
	if err != nil {
		return domain.ListWorkflowsResponse{}, err
	}

	resp := domain.ListWorkflowsResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Workflow) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(items) > pageSize {
			items = items[:pageSize]
		}
	}

	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	counts, err := s.repo.CountActions(ctx, ids)
	if err != nil {
		return domain.ListWorkflowsResponse{}, err
	}

	resp.Workflows = make([]domain.WorkflowListItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Workflows = append(resp.Workflows, toWorkflowListItem(*item, counts[item.ID]))
	}
	return resp, nil
}

func (s *service) ApproveWorkflow(ctx context.Context, req domain.ApproveWorkflowRequest) (domain.WorkflowDetailResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidOrganization
	}

	workflowID, err := uuid.Parse(strings.TrimSpace(req.ID))
	if err != nil || workflowID == uuid.Nil {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidID
	}

	status := req.Status
	if status != domain.ApprovalApproved && status != domain.ApprovalRejected {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidStatus
	}

	workflow, err := s.repo.FindWorkflowByID(ctx, orgID, workflowID)
	if err != nil {
		return domain.WorkflowDetailResponse{}, err
	}
	if workflow == nil {
		return domain.WorkflowDetailResponse{}, domain.ErrNotFound
	}
	if workflow.Status != domain.WorkflowPendingApproval {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidStatus
	}

	actor := strings.TrimSpace(req.ActorID)
	if actor == "" {
		actor = "system"
	}

	now := time.Now().UTC()
	approval := domain.WorkflowApprovalRow{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		ActorID:    actor,
		Status:     status,
		Note:       strings.TrimSpace(req.Note),
		CreatedAt:  now,
	}
	if err := s.repo.CreateApproval(ctx, approval); err != nil {
		return domain.WorkflowDetailResponse{}, err
	}

	nextStatus := domain.WorkflowApproved
	if status == domain.ApprovalRejected {
		nextStatus = domain.WorkflowRejected
	}
	if err := s.repo.UpdateWorkflow(ctx, orgID, workflowID, map[string]interface{}{
		"status":     nextStatus,
		"updated_at": now,
	}); err != nil {
		return domain.WorkflowDetailResponse{}, err
	}

	return s.GetWorkflow(ctx, domain.GetWorkflowRequest{ID: workflowID.String()})
}

func (s *service) ExecuteWorkflow(ctx context.Context, req domain.ExecuteWorkflowRequest) (domain.WorkflowDetailResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidOrganization
	}

	workflowID, err := uuid.Parse(strings.TrimSpace(req.ID))
	if err != nil || workflowID == uuid.Nil {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidID
	}

	workflow, err := s.repo.FindWorkflowByID(ctx, orgID, workflowID)
	if err != nil {
		return domain.WorkflowDetailResponse{}, err
	}
	if workflow == nil {
		return domain.WorkflowDetailResponse{}, domain.ErrNotFound
	}
	if workflow.Status != domain.WorkflowApproved {
		return domain.WorkflowDetailResponse{}, domain.ErrInvalidStatus
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateWorkflow(ctx, orgID, workflowID, map[string]interface{}{
		"status":     domain.WorkflowExecuting,
		"started_at": now,
		"updated_at": now,
	}); err != nil {
		return domain.WorkflowDetailResponse{}, err
	}

	s.enqueueExecution(workflow.OrgID, workflow.ID)

	return s.GetWorkflow(ctx, domain.GetWorkflowRequest{ID: workflowID.String()})
}

func (s *service) enqueueExecution(orgID, workflowID uuid.UUID) {
	go func() {
		_ = s.executeWorkflow(context.Background(), orgID, workflowID)
	}()
}

func (s *service) executeWorkflow(ctx context.Context, orgID, workflowID uuid.UUID) error {
	workflow, err := s.repo.FindWorkflowByID(ctx, orgID, workflowID)
	if err != nil || workflow == nil {
		return err
	}

	actions, err := s.repo.ListWorkflowActions(ctx, workflowID)
	if err != nil {
		return err
	}

	execCtx := orgcontext.WithOrgID(ctx, orgID)
	for _, action := range actions {
		now := time.Now().UTC()
		_ = s.repo.UpdateAction(ctx, action.ID, map[string]interface{}{
			"status":     domain.ActionRunning,
			"updated_at": now,
		})

		if err := s.executor.Execute(execCtx, *workflow, action); err != nil {
			failedAt := time.Now().UTC()
			_ = s.repo.UpdateAction(ctx, action.ID, map[string]interface{}{
				"status":     domain.ActionFailed,
				"error":      err.Error(),
				"updated_at": failedAt,
			})
			_ = s.repo.UpdateWorkflow(ctx, orgID, workflowID, map[string]interface{}{
				"status":      domain.WorkflowFailed,
				"finished_at": failedAt,
				"updated_at":  failedAt,
			})
			return err
		}

		doneAt := time.Now().UTC()
		_ = s.repo.UpdateAction(ctx, action.ID, map[string]interface{}{
			"status":     domain.ActionDone,
			"updated_at": doneAt,
		})
	}

	finishedAt := time.Now().UTC()
	return s.repo.UpdateWorkflow(ctx, orgID, workflowID, map[string]interface{}{
		"status":      domain.WorkflowCompleted,
		"finished_at": finishedAt,
		"updated_at":  finishedAt,
	})
}

func isValidActionType(actionType domain.ActionType) bool {
	switch actionType {
	case domain.ActionNavigate,
		domain.ActionCreateProduct,
		domain.ActionCreatePlan,
		domain.ActionCreateMeter,
		domain.ActionCreateFeature,
		domain.ActionNotify:
		return true
	default:
		return false
	}
}

func normalizePayload(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 || string(payload) == "null" {
		return json.RawMessage(`{}`)
	}
	return payload
}

func toWorkflowListItem(workflow domain.Workflow, actionCount int) domain.WorkflowListItem {
	var source string
	if workflow.SourceRunID != nil {
		source = workflow.SourceRunID.String()
	}
	return domain.WorkflowListItem{
		ID:          workflow.ID.String(),
		Title:       workflow.Title,
		Intent:      workflow.Intent,
		Summary:     workflow.Summary,
		Status:      buildStatusBadge(workflow.Status),
		CreatedAt:   workflow.CreatedAt,
		UpdatedAt:   workflow.UpdatedAt,
		Actions:     actionCount,
		SourceRunID: source,
	}
}

func toWorkflowDetail(workflow domain.Workflow, actions []domain.WorkflowActionRow, approvals []domain.WorkflowApprovalRow) domain.WorkflowDetail {
	var source string
	if workflow.SourceRunID != nil {
		source = workflow.SourceRunID.String()
	}

	actionItems := make([]domain.WorkflowAction, 0, len(actions))
	for _, action := range actions {
		actionItems = append(actionItems, domain.WorkflowAction{
			ID:        action.ID.String(),
			Type:      action.Type,
			Label:     action.Label,
			Status:    action.Status,
			Payload:   action.Payload,
			Order:     action.Order,
			Error:     action.Error,
			CreatedAt: action.CreatedAt,
			UpdatedAt: action.UpdatedAt,
		})
	}

	approvalItems := make([]domain.WorkflowApproval, 0, len(approvals))
	for _, approval := range approvals {
		approvalItems = append(approvalItems, domain.WorkflowApproval{
			ID:        approval.ID.String(),
			ActorID:   approval.ActorID,
			Status:    approval.Status,
			Note:      approval.Note,
			CreatedAt: approval.CreatedAt,
		})
	}

	return domain.WorkflowDetail{
		ID:          workflow.ID.String(),
		Title:       workflow.Title,
		Intent:      workflow.Intent,
		Summary:     workflow.Summary,
		Status:      buildStatusBadge(workflow.Status),
		SourceRunID: source,
		CreatedAt:   workflow.CreatedAt,
		UpdatedAt:   workflow.UpdatedAt,
		StartedAt:   workflow.StartedAt,
		FinishedAt:  workflow.FinishedAt,
		Actions:     actionItems,
		Approvals:   approvalItems,
	}
}

func buildStatusBadge(status domain.WorkflowStatus) domain.StatusBadge {
	switch status {
	case domain.WorkflowPendingApproval:
		return domain.StatusBadge{Code: string(status), Label: "Pending approval", Tone: "warning"}
	case domain.WorkflowApproved:
		return domain.StatusBadge{Code: string(status), Label: "Approved", Tone: "success"}
	case domain.WorkflowRejected:
		return domain.StatusBadge{Code: string(status), Label: "Rejected", Tone: "danger"}
	case domain.WorkflowExecuting:
		return domain.StatusBadge{Code: string(status), Label: "Executing", Tone: "info"}
	case domain.WorkflowCompleted:
		return domain.StatusBadge{Code: string(status), Label: "Completed", Tone: "success"}
	case domain.WorkflowFailed:
		return domain.StatusBadge{Code: string(status), Label: "Failed", Tone: "danger"}
	default:
		return domain.StatusBadge{Code: string(status), Label: "Draft", Tone: "muted"}
	}
}
