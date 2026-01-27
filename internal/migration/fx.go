package migration

import (
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var Module = fx.Module("migrations",
	fx.Invoke(func(conn *gorm.DB) error {
		sqlDB, err := conn.DB()
		if err != nil {
			return err
		}

		return RunMigrations(sqlDB)
	}),
)
