package domain

import "context"

type Service interface {
	CreateRun(ctx context.Context, req CreateRunRequest) (RunDetailResponse, error)
	GetRun(ctx context.Context, req GetRunRequest) (RunDetailResponse, error)
	ListRuns(ctx context.Context, req ListRunsRequest) (ListRunsResponse, error)
	Overview(ctx context.Context) (OverviewResponse, error)
}
