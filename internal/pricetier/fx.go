package pricetier

import (
	"github.com/railzwaylabs/railzway/internal/pricetier/repository"
	"github.com/railzwaylabs/railzway/internal/pricetier/service"
	"go.uber.org/fx"
)

var Module = fx.Module("pricetier.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.New),
)
