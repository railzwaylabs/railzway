package oauth2provider

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid_request")
	ErrInvalidClient      = errors.New("invalid_client")
	ErrInvalidRedirectURI = errors.New("invalid_redirect_uri")
	ErrInvalidScope       = errors.New("invalid_scope")
	ErrInvalidCode        = errors.New("invalid_code")
	ErrCodeUsed           = errors.New("authorization_code_used")
	ErrCodeExpired        = errors.New("authorization_code_expired")
	ErrPKCEMismatch       = errors.New("pkce_verification_failed")
	ErrInvalidToken       = errors.New("invalid_token")
)
