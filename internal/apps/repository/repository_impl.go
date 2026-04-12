package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/apps/domain"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.Repository {
	return &repository{db: db}
}

type appRow struct {
	ID                string `gorm:"column:id"`
	Name              string `gorm:"column:name"`
	Category          string `gorm:"column:category"`
	Provider          string `gorm:"column:provider"`
	Description       string `gorm:"column:description"`
	Capabilities      []byte `gorm:"column:capabilities"`
	AuthMethods       []byte `gorm:"column:auth_methods"`
	CredentialsSchema []byte `gorm:"column:credentials_schema"`
	Status            string `gorm:"column:status"`
	Version           string `gorm:"column:version"`
}

func (r *repository) ListCatalog(ctx context.Context) ([]domain.AppDefinition, error) {
	if r.db == nil {
		return []domain.AppDefinition{}, nil
	}
	var rows []appRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, name, category, provider, description, capabilities, auth_methods, credentials_schema, status, version
		     FROM apps_catalog
		     ORDER BY category, provider`).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.AppDefinition, 0, len(rows))
	for _, row := range rows {
		caps := []string{}
		if len(row.Capabilities) > 0 {
			_ = json.Unmarshal(row.Capabilities, &caps)
		}
		authMethods := []string{}
		if len(row.AuthMethods) > 0 {
			_ = json.Unmarshal(row.AuthMethods, &authMethods)
		}
		credSchema := map[string][]string{}
		if len(row.CredentialsSchema) > 0 {
			_ = json.Unmarshal(row.CredentialsSchema, &credSchema)
		}
		out = append(out, domain.AppDefinition{
			ID:                row.ID,
			Name:              row.Name,
			Category:          domain.AppCategory(row.Category),
			Provider:          row.Provider,
			Description:       row.Description,
			Capabilities:      caps,
			AuthMethods:       authMethods,
			CredentialsSchema: credSchema,
			Status:            domain.AppStatus(row.Status),
			Version:           row.Version,
		})
	}

	return out, nil
}

func (r *repository) GetCatalogByID(ctx context.Context, id string) (*domain.AppDefinition, error) {
	if r.db == nil {
		return nil, nil
	}
	var row appRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, name, category, provider, description, capabilities, auth_methods, credentials_schema, status, version
		     FROM apps_catalog WHERE id = ?`, id).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, nil
	}
	caps := []string{}
	if len(row.Capabilities) > 0 {
		_ = json.Unmarshal(row.Capabilities, &caps)
	}
	authMethods := []string{}
	if len(row.AuthMethods) > 0 {
		_ = json.Unmarshal(row.AuthMethods, &authMethods)
	}
	credSchema := map[string][]string{}
	if len(row.CredentialsSchema) > 0 {
		_ = json.Unmarshal(row.CredentialsSchema, &credSchema)
	}
	return &domain.AppDefinition{
		ID:                row.ID,
		Name:              row.Name,
		Category:          domain.AppCategory(row.Category),
		Provider:          row.Provider,
		Description:       row.Description,
		Capabilities:      caps,
		AuthMethods:       authMethods,
		CredentialsSchema: credSchema,
		Status:            domain.AppStatus(row.Status),
		Version:           row.Version,
	}, nil
}

type installRow struct {
	ID          uuid.UUID `gorm:"column:id"`
	OrgID       uuid.UUID `gorm:"column:org_id"`
	AppID       string    `gorm:"column:app_id"`
	Status      string    `gorm:"column:status"`
	AuthMethod  string    `gorm:"column:auth_method"`
	Config      []byte    `gorm:"column:config"`
	Credentials []byte    `gorm:"column:credentials"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (r *repository) ListInstallations(ctx context.Context, orgID uuid.UUID) ([]domain.AppInstallation, error) {
	if r.db == nil {
		return []domain.AppInstallation{}, nil
	}
	var rows []installRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, org_id, app_id, status, auth_method, config, credentials, created_at, updated_at
		     FROM apps_installations WHERE org_id = ? ORDER BY created_at DESC`, orgID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return mapInstallations(rows), nil
}

func (r *repository) FindInstallationByID(ctx context.Context, orgID, id uuid.UUID) (*domain.AppInstallation, error) {
	if r.db == nil {
		return nil, nil
	}
	var rows []installRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, org_id, app_id, status, auth_method, config, credentials, created_at, updated_at
		     FROM apps_installations WHERE org_id = ? AND id = ?`, orgID, id).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	installations := mapInstallations(rows)
	if len(installations) == 0 {
		return nil, nil
	}
	return &installations[0], nil
}

func (r *repository) FindInstallationByAppID(ctx context.Context, orgID uuid.UUID, appID string) (*domain.AppInstallation, error) {
	if r.db == nil {
		return nil, nil
	}
	var rows []installRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT id, org_id, app_id, status, auth_method, config, credentials, created_at, updated_at
		     FROM apps_installations WHERE org_id = ? AND app_id = ?`, orgID, appID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	installations := mapInstallations(rows)
	if len(installations) == 0 {
		return nil, nil
	}
	return &installations[0], nil
}

func (r *repository) CreateInstallation(ctx context.Context, inst domain.AppInstallation) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO apps_installations (id, org_id, app_id, status, auth_method, config, credentials, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inst.ID,
		inst.OrgID,
		inst.AppID,
		inst.Status,
		inst.AuthMethod,
		inst.Config,
		inst.Credentials,
		inst.CreatedAt,
		inst.UpdatedAt,
	).Error
}

func (r *repository) UpdateInstallation(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Table("apps_installations").
		Where("org_id = ? AND id = ?", orgID, id).
		Updates(updates).Error
}

func mapInstallations(rows []installRow) []domain.AppInstallation {
	out := make([]domain.AppInstallation, 0, len(rows))
	for _, row := range rows {
		config := json.RawMessage(`{}`)
		if len(row.Config) > 0 {
			config = json.RawMessage(row.Config)
		}
		credentials := json.RawMessage(`{}`)
		if len(row.Credentials) > 0 {
			credentials = json.RawMessage(row.Credentials)
		}
		out = append(out, domain.AppInstallation{
			ID:          row.ID,
			OrgID:       row.OrgID,
			AppID:       row.AppID,
			Status:      domain.InstallationStatus(row.Status),
			AuthMethod:  row.AuthMethod,
			Config:      config,
			Credentials: credentials,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return out
}
