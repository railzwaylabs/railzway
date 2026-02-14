package repository

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/integration/domain"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) domain.Repository {
	return &Repository{db: db}
}

func (r *Repository) ListCatalog(ctx context.Context) ([]domain.CatalogItem, error) {
	var items []domain.CatalogItem
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&items).Error
	return items, err
}

func (r *Repository) GetCatalogItem(ctx context.Context, id string) (*domain.CatalogItem, error) {
	var item domain.CatalogItem
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrIntegrationNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repository) ListConnections(ctx context.Context, orgID snowflake.ID) ([]domain.Connection, error) {
	var conns []domain.Connection
	err := r.db.WithContext(ctx).
		Where("org_id = ? AND status != 'disconnected'", orgID).
		Preload("Integration").
		Find(&conns).Error
	return conns, err
}

func (r *Repository) GetConnection(ctx context.Context, id snowflake.ID) (*domain.Connection, error) {
	var conn domain.Connection
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&conn).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrConnectionNotFound
		}
		return nil, err
	}
	return &conn, nil
}

func (r *Repository) SaveConnection(ctx context.Context, conn *domain.Connection) error {
	return r.db.WithContext(ctx).Save(conn).Error
}

func (r *Repository) DeleteConnection(ctx context.Context, id snowflake.ID) error {
	// Soft delete by setting status to disconnected? 
	// Or hard delete?
	// The domain model has a Status field, let's use that for "Disconnect" logic in Service.
	// But if we want to physically remove (e.g. cleanup), we can use Delete.
	// For now, let's implement soft delete via Status update in Service, 
	// and this DeleteConnection can be a hard delete if needed, or we just rely on Save.
	// Let's make this a hard delete for now to support "Remove" action.
	return r.db.WithContext(ctx).Delete(&domain.Connection{}, "id = ?", id).Error
}
