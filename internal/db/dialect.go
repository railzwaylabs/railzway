package db

import (
	"fmt"

	"github.com/railzwaylabs/railzway/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Dialect(cfg *config.Config) (gorm.Dialector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing config")
	}

	switch cfg.DB.Type {
	case "mysql":
		return mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.DB.User,
			cfg.DB.Pass,
			cfg.DB.Host,
			cfg.DB.Port,
			cfg.DB.Name,
		)), nil
	case "postgres":
		return postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.DB.Host,
			cfg.DB.User,
			cfg.DB.Pass,
			cfg.DB.Name,
			cfg.DB.Port,
			cfg.DB.SSLMode,
		)), nil
	case "sqlite":
		return sqlite.Open("gorm.db"), nil
	default:
		return nil, fmt.Errorf("unsupported %s type", cfg.DB.Type)
	}

}
