package meter

import (
	"github.com/smallbiznis/railzway/internal/meter/repository"
	"github.com/smallbiznis/railzway/internal/meter/service"
	"go.uber.org/fx"
)

var Module = fx.Module("meter.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
