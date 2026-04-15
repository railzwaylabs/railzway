package domain

import "context"

type Service interface {
	CreateWorkflow(ctx context.Context, req CreateWorkflowRequest) (WorkflowDetailResponse, error)
	GetWorkflow(ctx context.Context, req GetWorkflowRequest) (WorkflowDetailResponse, error)
	ListWorkflows(ctx context.Context, req ListWorkflowsRequest) (ListWorkflowsResponse, error)
	ApproveWorkflow(ctx context.Context, req ApproveWorkflowRequest) (WorkflowDetailResponse, error)
	ExecuteWorkflow(ctx context.Context, req ExecuteWorkflowRequest) (WorkflowDetailResponse, error)
}
