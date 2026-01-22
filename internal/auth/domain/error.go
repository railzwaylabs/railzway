package domain

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrMustChangePassword = errors.New("password change required")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrSessionRevoked     = errors.New("session revoked")
	ErrInvalidSession     = errors.New("invalid session")
)
