package domain

import (
	"context"
	"time"
)

type Service interface {
	Create(ctx context.Context, req CreateTestClockRequest) (*TestClock, error)
	Advance(ctx context.Context, id string, duration time.Duration) (*TestClock, error)
	Get(ctx context.Context, id string) (*TestClock, error)
	List(ctx context.Context) ([]TestClock, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, req UpdateTestClockRequest) (*TestClock, error)
}

type CreateTestClockRequest struct {
	Name        string
	InitialTime time.Time
}

type UpdateTestClockRequest struct {
	Name string
}
