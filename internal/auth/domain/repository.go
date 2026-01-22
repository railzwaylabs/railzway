package domain

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
)

//go:generate mockgen -source=repository.go -destination=../mocks/mock_repository.go -package=mocks

type Repository interface {
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, user *User) error
	FindOne(ctx context.Context, user User) (*User, error)
	FindByExternalID(ctx context.Context, externalID string) (*User, error)
	FindByID(ctx context.Context, id snowflake.ID) (*User, error)
	UpdateFields(ctx context.Context, id snowflake.ID, fields map[string]any) error
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	UpdateLastSeen(ctx context.Context, sessionID snowflake.ID, lastSeen time.Time) error
	UpdateOrgContext(ctx context.Context, sessionID snowflake.ID, activeOrgID *int64, orgIDs []int64) error
	RevokeSession(ctx context.Context, sessionID snowflake.ID, revokedAt time.Time) error
}
