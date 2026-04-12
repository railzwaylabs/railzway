package domain

import (
	"context"
	"errors"
	"time"
)

type UpsertTestClockRequest struct {
	CurrentTime time.Time
	Status      string
}

type AdvanceTestClockRequest struct {
	AdvanceBy time.Duration
}

type Service interface {
	Get(ctx context.Context) (*TestClock, error)
	Upsert(ctx context.Context, req UpsertTestClockRequest) (TestClock, error)
	Advance(ctx context.Context, req AdvanceTestClockRequest) (TestClock, error)
	Pause(ctx context.Context) (TestClock, error)
	Resume(ctx context.Context) (TestClock, error)
	ListActive(ctx context.Context) ([]TestClock, error)
}

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidTime         = errors.New("invalid_time")
	ErrInvalidAdvance      = errors.New("invalid_advance")
	ErrInvalidStatus       = errors.New("invalid_status")
	ErrNotFound            = errors.New("not_found")
)
