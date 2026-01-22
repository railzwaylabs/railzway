package pricetier

import (
	"github.com/smallbiznis/railzway/internal/pricetier/repository"
	"github.com/smallbiznis/railzway/internal/pricetier/service"
	"go.uber.org/fx"
)

var Module = fx.Module("pricetier.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
