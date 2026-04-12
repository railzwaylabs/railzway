package product

import (
	"github.com/railzwaylabs/railzway/internal/product/repository"
	"github.com/railzwaylabs/railzway/internal/product/service"
	"go.uber.org/fx"
)

// Module provides product repository and service wiring.
var Module = fx.Module("product",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
