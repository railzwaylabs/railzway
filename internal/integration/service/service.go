package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bwmarrin/snowflake"
	auditdomain "github.com/railzwaylabs/railzway/internal/audit/domain"
	"github.com/railzwaylabs/railzway/internal/integration/domain"
	"github.com/railzwaylabs/railzway/internal/security/vault"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type Service struct {
	repo     domain.Repository
	vault    vault.Provider
	auditSvc auditdomain.Service
	node     *snowflake.Node
	log      *zap.Logger
}

func New(
	repo domain.Repository,
	vault vault.Provider,
	auditSvc auditdomain.Service,
	log *zap.Logger,
) (domain.Service, error) {
	node, err := snowflake.NewNode(1) // TODO: Get node ID from config
	if err != nil {
		return nil, err
	}
	return &Service{
		repo:     repo,
		vault:    vault,
		auditSvc: auditSvc,
		node:     node,
		log:      log,
	}, nil
}

func (s *Service) ListCatalog(ctx context.Context) ([]domain.CatalogItem, error) {
	return s.repo.ListCatalog(ctx)
}

func (s *Service) ListConnections(ctx context.Context, orgID snowflake.ID) ([]domain.Connection, error) {
	return s.repo.ListConnections(ctx, orgID)
}

func (s *Service) Connect(ctx context.Context, input domain.ConnectInput) (*domain.Connection, error) {
	// 1. Verify Integration exists
	item, err := s.repo.GetCatalogItem(ctx, input.IntegrationID)
	if err != nil {
		return nil, err
	}

	// 2. Encrypt Credentials
	credsBytes, err := json.Marshal(input.Credentials)
	if err != nil {
		return nil, err
	}
	encryptedCreds, err := s.vault.Encrypt(credsBytes)
	if err != nil {
		s.log.Error("failed to encrypt credentials", zap.Error(err))
		return nil, err
	}

	// 3. Prepare Config
	configBytes, err := json.Marshal(input.Config)
	if err != nil {
		return nil, err
	}

	// 4. Create Connection
	conn := &domain.Connection{
		ID:             s.node.Generate(),
		OrgID:          input.OrgID,
		IntegrationID:  item.ID,
		Name:           input.Name,
		Config:         datatypes.JSON(configBytes),
		EncryptedCreds: encryptedCreds,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.SaveConnection(ctx, conn); err != nil {
		return nil, err
	}

	// 5. Audit Log
	// Assuming auditdomain.Service.AuditLog signature:
	// AuditLog(ctx, orgID, actorType, actorID, action, targetType, targetID, metadata)
	targetID := conn.ID.String()
	s.auditSvc.AuditLog(
		ctx,
		&input.OrgID,
		"user", // TODO: Get actor from context
		nil,
		"integration.connected",
		"integration_connection",
		&targetID,
		map[string]any{"integration_id": item.ID},
	)

	return conn, nil
}

func (s *Service) Disconnect(ctx context.Context, id snowflake.ID) error {
	conn, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return err
	}

	conn.Status = "disconnected"
	conn.EncryptedCreds = nil // Wipe credentials
	conn.UpdatedAt = time.Now()

	if err := s.repo.SaveConnection(ctx, conn); err != nil {
		return err
	}

	targetID := conn.ID.String()
	s.auditSvc.AuditLog(
		ctx,
		&conn.OrgID,
		"user",
		nil,
		"integration.disconnected",
		"integration_connection",
		&targetID,
		nil,
	)

	return nil
}

func (s *Service) GetConnectionConfig(ctx context.Context, id snowflake.ID) (map[string]any, error) {
	conn, err := s.repo.GetConnection(ctx, id)
	if err != nil {
		return nil, err
	}

	// Decrypt creds
	decryptedBytes, err := s.vault.Decrypt(conn.EncryptedCreds)
	if err != nil {
		return nil, err
	}

	var creds map[string]any
	if err := json.Unmarshal(decryptedBytes, &creds); err != nil {
		return nil, err
	}

	// Merge with public config
	var config map[string]any
	if len(conn.Config) > 0 {
		if err := json.Unmarshal(conn.Config, &config); err != nil {
			return nil, err
		}
	} else {
		config = make(map[string]any)
	}

	// Combine: Config + Creds
	// WARNING: Be careful not to expose this to public API.
	// This method is intended for internal consumers (Dispatchers).
	for k, v := range creds {
		config[k] = v
	}

	return config, nil
}
