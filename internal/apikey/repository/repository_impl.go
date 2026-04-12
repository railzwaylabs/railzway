package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/apikey/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

func (r *repository) CountActiveKeys(ctx context.Context, orgID string) (int64, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Table("api_keys").
		Where("org_id = ? AND status = ?", orgID, "active").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

type keyRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	OrgID          uuid.UUID  `gorm:"column:org_id"`
	Name           string     `gorm:"column:name"`
	KeyPrefix      string     `gorm:"column:key_prefix"`
	KeyType        string     `gorm:"column:key_type"`
	Scopes         []byte     `gorm:"column:scopes"`
	AllowedIPs     []byte     `gorm:"column:allowed_ips"`
	AllowedDomains []byte     `gorm:"column:allowed_domains"`
	Status         string     `gorm:"column:status"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at"`
	RevokedAt      *time.Time `gorm:"column:revoked_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (r *repository) ListKeys(ctx context.Context, orgID string) ([]domain.APIKey, error) {
	if r.db == nil {
		return []domain.APIKey{}, nil
	}
	var rows []keyRow
	if err := r.db.WithContext(ctx).
		Raw(`SELECT id, org_id, name, key_prefix, key_type, scopes, allowed_ips, allowed_domains, status, last_used_at, revoked_at, created_at, updated_at
		     FROM api_keys WHERE org_id = ? ORDER BY created_at DESC`, orgID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.APIKey, 0, len(rows))
	for _, row := range rows {
		scopes := []string{}
		allowedIPs := []string{}
		allowedDomains := []string{}
		if len(row.Scopes) > 0 {
			_ = json.Unmarshal(row.Scopes, &scopes)
		}
		if len(row.AllowedIPs) > 0 {
			_ = json.Unmarshal(row.AllowedIPs, &allowedIPs)
		}
		if len(row.AllowedDomains) > 0 {
			_ = json.Unmarshal(row.AllowedDomains, &allowedDomains)
		}
		out = append(out, domain.APIKey{
			ID:            row.ID.String(),
			OrgID:         row.OrgID.String(),
			Name:          row.Name,
			KeyPrefix:     row.KeyPrefix,
			KeyType:       domain.KeyType(row.KeyType),
			Scopes:        scopes,
			AllowedIPs:    allowedIPs,
			AllowedDomain: allowedDomains,
			Status:        domain.KeyStatus(row.Status),
			LastUsedAt:    row.LastUsedAt,
			RevokedAt:     row.RevokedAt,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *repository) FindByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	if r.db == nil {
		return nil, nil
	}
	var row keyRow
	if err := r.db.WithContext(ctx).
		Raw(`SELECT id, org_id, name, key_prefix, key_type, scopes, allowed_ips, allowed_domains, status, last_used_at, revoked_at, created_at, updated_at
		     FROM api_keys WHERE key_hash = ? LIMIT 1`, keyHash).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	scopes := []string{}
	allowedIPs := []string{}
	allowedDomains := []string{}
	if len(row.Scopes) > 0 {
		_ = json.Unmarshal(row.Scopes, &scopes)
	}
	if len(row.AllowedIPs) > 0 {
		_ = json.Unmarshal(row.AllowedIPs, &allowedIPs)
	}
	if len(row.AllowedDomains) > 0 {
		_ = json.Unmarshal(row.AllowedDomains, &allowedDomains)
	}
	result := domain.APIKey{
		ID:            row.ID.String(),
		OrgID:         row.OrgID.String(),
		Name:          row.Name,
		KeyPrefix:     row.KeyPrefix,
		KeyType:       domain.KeyType(row.KeyType),
		Scopes:        scopes,
		AllowedIPs:    allowedIPs,
		AllowedDomain: allowedDomains,
		Status:        domain.KeyStatus(row.Status),
		LastUsedAt:    row.LastUsedAt,
		RevokedAt:     row.RevokedAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	return &result, nil
}

func (r *repository) CreateKey(ctx context.Context, key domain.APIKey, keyHash string) error {
	if r.db == nil {
		return nil
	}
	scopes, _ := json.Marshal(key.Scopes)
	allowedIPs, _ := json.Marshal(key.AllowedIPs)
	allowedDomains, _ := json.Marshal(key.AllowedDomain)
	return r.db.WithContext(ctx).
		Table("api_keys").
		Create(map[string]interface{}{
			"id":              key.ID,
			"org_id":          key.OrgID,
			"name":            key.Name,
			"key_prefix":      key.KeyPrefix,
			"key_hash":        keyHash,
			"key_type":        key.KeyType,
			"scopes":          scopes,
			"allowed_ips":     allowedIPs,
			"allowed_domains": allowedDomains,
			"status":          key.Status,
			"created_at":      key.CreatedAt,
			"updated_at":      key.UpdatedAt,
		}).Error
}

func (r *repository) RevokeKey(ctx context.Context, orgID string, id string) (*domain.APIKey, error) {
	if r.db == nil {
		return nil, nil
	}
	var row keyRow
	if err := r.db.WithContext(ctx).
		Raw(`SELECT id, org_id, name, key_prefix, key_type, scopes, allowed_ips, allowed_domains, status, last_used_at, revoked_at, created_at, updated_at
		     FROM api_keys WHERE org_id = ? AND id = ?`, orgID, id).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Table("api_keys").
		Where("org_id = ? AND id = ?", orgID, id).
		Updates(map[string]interface{}{
			"status":     "revoked",
			"revoked_at": now,
			"updated_at": now,
		}).Error; err != nil {
		return nil, err
	}
	row.Status = "revoked"
	row.RevokedAt = &now
	row.UpdatedAt = now
	scopes := []string{}
	allowedIPs := []string{}
	allowedDomains := []string{}
	if len(row.Scopes) > 0 {
		_ = json.Unmarshal(row.Scopes, &scopes)
	}
	if len(row.AllowedIPs) > 0 {
		_ = json.Unmarshal(row.AllowedIPs, &allowedIPs)
	}
	if len(row.AllowedDomains) > 0 {
		_ = json.Unmarshal(row.AllowedDomains, &allowedDomains)
	}
	result := domain.APIKey{
		ID:            row.ID.String(),
		OrgID:         row.OrgID.String(),
		Name:          row.Name,
		KeyPrefix:     row.KeyPrefix,
		KeyType:       domain.KeyType(row.KeyType),
		Scopes:        scopes,
		AllowedIPs:    allowedIPs,
		AllowedDomain: allowedDomains,
		Status:        domain.KeyStatus(row.Status),
		LastUsedAt:    row.LastUsedAt,
		RevokedAt:     row.RevokedAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	return &result, nil
}

func (r *repository) TouchLastUsed(ctx context.Context, id string) error {
	if r.db == nil {
		return nil
	}
	if strings.TrimSpace(id) == "" {
		return nil
	}
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Table("api_keys").
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_used_at": now,
			"updated_at":   now,
		}).Error
}
