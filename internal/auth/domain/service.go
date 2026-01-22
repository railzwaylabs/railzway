package domain

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
)

//go:generate mockgen -source=service.go -destination=../mocks/mock_service.go -package=mocks
type Service interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
	Login(ctx context.Context, req LoginRequest) (*LoginResult, error)
	Logout(ctx context.Context, rawToken string) error
	Authenticate(ctx context.Context, rawToken string) (*Session, error)
	UpdateSessionOrgContext(ctx context.Context, sessionID snowflake.ID, activeOrgID *int64, orgIDs []int64) error
	ChangePassword(ctx context.Context, userID string, newPassword string) error
	CurrentUser(ctx context.Context) (*User, error)
}

type CreateUserRequest struct {
	Username    string // deprecated
	Email       string
	Password    string
	DisplayName string
}

type LoginRequest struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

type LoginResult struct {
	Session   *SessionView
	RawToken  string
	ExpiresAt time.Time
	SessionID snowflake.ID
}
