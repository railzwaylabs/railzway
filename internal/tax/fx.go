package tax

import (
	"github.com/smallbiznis/railzway/internal/tax/repository"
	"github.com/smallbiznis/railzway/internal/tax/service"
	"go.uber.org/fx"
)

var Module = fx.Module("tax.service",
	fx.Provide(repository.NewRepository),
	fx.Provide(service.NewResolver),
	fx.Provide(service.NewService),
)
