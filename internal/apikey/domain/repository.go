package domain

import (
	"context"
)

type Repository interface {
	CountActiveKeys(ctx context.Context, orgID string) (int64, error)
	ListKeys(ctx context.Context, orgID string) ([]APIKey, error)
	FindByHash(ctx context.Context, keyHash string) (*APIKey, error)
	CreateKey(ctx context.Context, key APIKey, keyHash string) error
	RevokeKey(ctx context.Context, orgID string, id string) (*APIKey, error)
	TouchLastUsed(ctx context.Context, id string) error
}
