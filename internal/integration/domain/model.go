package domain

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
)

var (
	ErrIntegrationNotFound = errors.New("integration not found")
	ErrConnectionNotFound  = errors.New("connection not found")
)

// Integration Types
const (
	TypeNotification = "notification"
	TypeAccounting   = "accounting"
	TypePayment      = "payment"
	TypeCRM          = "crm"
	TypeDataWarehouse = "data_warehouse"
	TypeAnalytics     = "analytics"
)

// Auth Types
const (
	AuthTypeOAuth2 = "oauth2"
	AuthTypeAPIKey = "api_key"
	AuthTypeBasic  = "basic"
)

// CatalogItem represents an available integration in the marketplace.
type CatalogItem struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	Type        string         `json:"type" gorm:"not null"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	LogoURL     string         `json:"logo_url"`
	AuthType    string         `json:"auth_type" gorm:"not null"`
	Schema      datatypes.JSON `json:"schema" gorm:"type:jsonb;not null;default:'{}'"` // JSON Schema for config form
	IsActive    bool           `json:"is_active" gorm:"not null;default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (CatalogItem) TableName() string {
	return "integration_catalog"
}

// Connection represents an active link between an Organization and an Integration.
type Connection struct {
	ID             snowflake.ID   `json:"id" gorm:"primaryKey"`
	OrgID          snowflake.ID   `json:"org_id" gorm:"not null;index"`
	IntegrationID  string         `json:"integration_id" gorm:"not null"`
	Integration    *CatalogItem   `json:"integration,omitempty" gorm:"foreignKey:IntegrationID"`
	Name           string         `json:"name"`                                             // User defined name
	Config         datatypes.JSON `json:"config" gorm:"type:jsonb;not null;default:'{}'"`   // Public/Non-sensitive
	EncryptedCreds []byte         `json:"-" gorm:"type:bytea"`                              // Sensitive (Encrypted)
	Status         string         `json:"status" gorm:"not null;default:'active'"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	LastSyncedAt   *time.Time     `json:"last_synced_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (Connection) TableName() string {
	return "integration_connections"
}

// Repository defines data access methods.
type Repository interface {
	ListCatalog(ctx context.Context) ([]CatalogItem, error)
	GetCatalogItem(ctx context.Context, id string) (*CatalogItem, error)
	
	ListConnections(ctx context.Context, orgID snowflake.ID) ([]Connection, error)
	GetConnection(ctx context.Context, id snowflake.ID) (*Connection, error)
	SaveConnection(ctx context.Context, conn *Connection) error
	DeleteConnection(ctx context.Context, id snowflake.ID) error
}

// Service defines business logic methods.
type Service interface {
	ListCatalog(ctx context.Context) ([]CatalogItem, error)
	ListConnections(ctx context.Context, orgID snowflake.ID) ([]Connection, error)
	
	Connect(ctx context.Context, input ConnectInput) (*Connection, error)
	Disconnect(ctx context.Context, id snowflake.ID) error
	
	GetConnectionConfig(ctx context.Context, id snowflake.ID) (map[string]any, error)
}

// ConnectInput DTO
type ConnectInput struct {
	OrgID         snowflake.ID
	IntegrationID string
	Name          string
	Config        map[string]any // Public config
	Credentials   map[string]any // To be encrypted
}
