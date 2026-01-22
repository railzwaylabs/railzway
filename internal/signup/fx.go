package signup

import (
	"github.com/bwmarrin/snowflake"
	"github.com/smallbiznis/railzway/internal/config"
	"github.com/smallbiznis/railzway/internal/signup/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var Module = fx.Module("signup.service",
	fx.Provide(newProvisioner),
	fx.Provide(NewService),
)

func newProvisioner(cfg config.Config, db *gorm.DB, genID *snowflake.Node) domain.Provisioner {
	if !cfg.IsCloud() {
		return NewNoopProvisioner()
	}

	return NewEventProvisioner(db, genID)
}
