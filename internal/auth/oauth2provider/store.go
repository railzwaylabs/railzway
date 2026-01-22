package oauth2provider

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Store provides persistence for OAuth2 provider data.
type Store interface {
	CreateAuthorizationCode(ctx context.Context, code *AuthorizationCode) error
	GetAuthorizationCode(ctx context.Context, codeHash string) (*AuthorizationCode, error)
	MarkAuthorizationCodeUsed(ctx context.Context, codeHash string, usedAt time.Time) (bool, error)
	CreateAccessToken(ctx context.Context, token *AccessToken) error
	GetAccessToken(ctx context.Context, tokenHash string) (*AccessToken, error)
}

type gormStore struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &gormStore{db: db}
}

func (s *gormStore) CreateAuthorizationCode(ctx context.Context, code *AuthorizationCode) error {
	return s.db.WithContext(ctx).Create(code).Error
}

func (s *gormStore) GetAuthorizationCode(ctx context.Context, codeHash string) (*AuthorizationCode, error) {
	var code AuthorizationCode
	err := s.db.WithContext(ctx).Where("code_hash = ?", codeHash).First(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCode
	}
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (s *gormStore) MarkAuthorizationCodeUsed(ctx context.Context, codeHash string, usedAt time.Time) (bool, error) {
	tx := s.db.WithContext(ctx).
		Model(&AuthorizationCode{}).
		Where("code_hash = ? AND used_at IS NULL", codeHash).
		Update("used_at", usedAt)
	if tx.Error != nil {
		return false, tx.Error
	}
	if tx.RowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

func (s *gormStore) CreateAccessToken(ctx context.Context, token *AccessToken) error {
	return s.db.WithContext(ctx).Create(token).Error
}

func (s *gormStore) GetAccessToken(ctx context.Context, tokenHash string) (*AccessToken, error) {
	var token AccessToken
	err := s.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}
