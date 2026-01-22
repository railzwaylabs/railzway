package migration

import (
	"github.com/smallbiznis/railzway/internal/config"
	"github.com/smallbiznis/railzway/internal/seed"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var Module = fx.Module("migrations",
	fx.Invoke(func(conn *gorm.DB, cfg config.Config) error {
		sqlDB, err := conn.DB()
		if err != nil {
			return err
		}

		if err := RunMigrations(sqlDB); err != nil {
			return err
		}

		if cfg.DefaultOrgID != 0 {
			if err := seed.EnsureMainOrgWithID(conn, cfg.DefaultOrgID); err != nil {
				return err
			}
		} else {
			if err := seed.EnsureMainOrg(conn); err != nil {
				return err
			}
		}
		if !cfg.IsCloud() && cfg.Bootstrap.EnsureDefaultOrgAndUser {
			return seed.EnsureMainOrgAndAdmin(conn)
		}
		return nil
	}),
)
