package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	ListCatalog(ctx context.Context) ([]AppDefinition, error)
	GetCatalogByID(ctx context.Context, id string) (*AppDefinition, error)
	ListInstallations(ctx context.Context, orgID uuid.UUID) ([]AppInstallation, error)
	FindInstallationByID(ctx context.Context, orgID, id uuid.UUID) (*AppInstallation, error)
	FindInstallationByAppID(ctx context.Context, orgID uuid.UUID, appID string) (*AppInstallation, error)
	CreateInstallation(ctx context.Context, inst AppInstallation) error
	UpdateInstallation(ctx context.Context, orgID, id uuid.UUID, updates map[string]interface{}) error
}
