package price

import (
	"github.com/railzwaylabs/railzway/internal/price/repository"
	"github.com/railzwaylabs/railzway/internal/price/service"
	"go.uber.org/fx"
)

var Module = fx.Module("price.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
