package customer

import (
	"github.com/railzwaylabs/railzway/internal/customer/repository"
	"github.com/railzwaylabs/railzway/internal/customer/service"
	"go.uber.org/fx"
)

// Module provides customer repository and service wiring.
var Module = fx.Module("customer",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
