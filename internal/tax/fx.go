package tax

import (
	"github.com/railzwaylabs/railzway/internal/tax/repository"
	"github.com/railzwaylabs/railzway/internal/tax/service"
	"go.uber.org/fx"
)

var Module = fx.Module("tax.service",
	fx.Provide(repository.NewRepository),
	fx.Provide(service.NewResolver),
	fx.Provide(service.NewService),
)
