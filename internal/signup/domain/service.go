package domain

import (
	"context"
	"errors"
	"time"

	authdomain "github.com/smallbiznis/railzway/internal/auth/domain"
)

type Service interface {
	Signup(ctx context.Context, req Request) (*Result, error)
}

type Request struct {
	OrgName   string `json:"org_name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	UserAgent string `json:"-"`
	IPAddress string `json:"-"`
}

type Result struct {
	Session   *authdomain.SessionView
	RawToken  string
	ExpiresAt time.Time
	OrgID     string
	UserID    string
}

type Provisioner interface {
	Provision(ctx context.Context, organizationID string) error
}

var ErrInvalidRequest = errors.New("invalid signup request")
