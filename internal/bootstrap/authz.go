package bootstrap

import (
	"context"

	"github.com/railzwaylabs/railzway/internal/authz"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func ensureAdminAuthorizationPolicies(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	if db == nil {
		return nil
	}
	if _, err := authz.NewAdminAuthorizer(db.WithContext(ctx)); err != nil {
		return err
	}
	if logger != nil {
		logger.Info("bootstrap admin authorization policies ensured")
	}
	return nil
}
