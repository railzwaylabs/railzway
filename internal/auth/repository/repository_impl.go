package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/auth/domain"
	"gorm.io/gorm"
)

type repo struct {
	db *gorm.DB
}

func New(db *gorm.DB) (domain.Repository, domain.SessionRepository) {
	r := &repo{db: db}
	return r, r
}

func (r *repo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.User{}).Count(&count).Error
	return count, err
}

func (r *repo) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *repo) FindByExternalID(ctx context.Context, externalID string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repo) FindOne(ctx context.Context, user domain.User) (*domain.User, error) {
	var u domain.User
	err := r.db.WithContext(ctx).Where(user).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *repo) FindByID(ctx context.Context, id snowflake.ID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repo) UpdateFields(ctx context.Context, id snowflake.ID, fields map[string]any) error {
	tx := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(fields)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *repo) CreateSession(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *repo) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).Where("session_token_hash = ?", tokenHash).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *repo) UpdateLastSeen(ctx context.Context, sessionID snowflake.ID, lastSeen time.Time) error {
	tx := r.db.WithContext(ctx).Model(&domain.Session{}).Where("id = ?", sessionID).Update("last_seen_at", lastSeen)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *repo) UpdateOrgContext(ctx context.Context, sessionID snowflake.ID, activeOrgID *int64, orgIDs []int64) error {
	if orgIDs == nil {
		orgIDs = []int64{}
	}

	updates := &domain.Session{
		ActiveOrgID: activeOrgID,
		OrgIDs:      orgIDs,
	}

	tx := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("id = ?", sessionID).
		Select("active_org_id", "org_ids").
		Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *repo) RevokeSession(ctx context.Context, sessionID snowflake.ID, revokedAt time.Time) error {
	tx := r.db.WithContext(ctx).Model(&domain.Session{}).Where("id = ?", sessionID).Update("revoked_at", revokedAt)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}
