package invoice

import (
	"github.com/railzwaylabs/railzway/internal/invoice/repository"
	"github.com/railzwaylabs/railzway/internal/invoice/service"
	"go.uber.org/fx"
)

// Module provides invoice repository and service wiring.
var Module = fx.Module("invoice",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
