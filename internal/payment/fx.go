package payment

import (
	"github.com/railzwaylabs/railzway/internal/payment/repository"
	"github.com/railzwaylabs/railzway/internal/payment/service"
	"go.uber.org/fx"
)

// Module provides payment repository and service wiring.
var Module = fx.Module("payment",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
