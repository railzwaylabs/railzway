package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/railzwaylabs/railzway/internal/config"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	"github.com/railzwaylabs/railzway/internal/seed"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EnsureDefaultOrgAndUser creates the default organization and admin user when explicitly enabled.
// This is intended for OSS/dev setups that want an explicit, env-gated bootstrap.
func EnsureDefaultOrgAndUser(cfg config.Config, db *gorm.DB, orgState OrgStateService, log *zap.Logger) error {
	if !cfg.Bootstrap.EnsureDefaultOrgAndUser {
		return nil
	}
	if db == nil {
		return errors.New("bootstrap requires database handle")
	}

	orgID := cfg.Bootstrap.DefaultOrgID
	createAdmin := true
	if cfg.IsCloud() {
		createAdmin = false
	}
	if err := seed.EnsureOrgAndAdminWithOptions(db, seed.OrgSeedOptions{
		OrgID:           orgID,
		Name:            cfg.Bootstrap.DefaultOrgName,
		Slug:            cfg.Bootstrap.DefaultOrgSlug,
		AdminEmail:      cfg.Bootstrap.AdminEmail,
		AdminPassword:   cfg.Bootstrap.AdminPassword,
		CreateAdminUser: &createAdmin,
	}); err != nil {
		return err
	}

	if orgState == nil {
		return nil
	}

	ctx := context.Background()
	var org organizationdomain.Organization
	query := db.WithContext(ctx).Model(&organizationdomain.Organization{})
	if orgID != 0 {
		query = query.Where("id = ?", orgID)
	} else if cfg.Bootstrap.DefaultOrgSlug != "" {
		query = query.Where("slug = ?", cfg.Bootstrap.DefaultOrgSlug)
	} else {
		query = query.Where("is_default = ?", true).Order("created_at ASC")
	}
	if err := query.First(&org).Error; err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := orgState.Initialize(ctx, org.ID, now); err != nil {
		return err
	}
	if err := orgState.Activate(ctx, org.ID, now); err != nil {
		return err
	}

	if log != nil {
		log.Info("default org bootstrap ensured",
			zap.String("org_id", org.ID.String()),
		)
	}
	return nil
}
