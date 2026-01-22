package price

import (
	"github.com/smallbiznis/railzway/internal/price/repository"
	"github.com/smallbiznis/railzway/internal/price/service"
	"go.uber.org/fx"
)

var Module = fx.Module("price.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
