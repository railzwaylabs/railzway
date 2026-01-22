package db

import (
	"fmt"

	"github.com/smallbiznis/railzway/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Dialect(cfg config.Config) (gorm.Dialector, error) {
	switch cfg.DBType {
	case "mysql":
		return mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
		)), nil
	case "postgres":
		return postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost,
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBPort,
			cfg.DBSSLMode,
		)), nil
	case "sqlite":
		return sqlite.Open("gorm.db"), nil
	default:
		return nil, fmt.Errorf("unsupported %s type", cfg.DBType)
	}

}
