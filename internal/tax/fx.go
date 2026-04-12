package tax

import (
	"github.com/railzwaylabs/railzway/internal/tax/repository"
	"github.com/railzwaylabs/railzway/internal/tax/service"
	"go.uber.org/fx"
)

// Module provides tax repository and service wiring.
var Module = fx.Module("tax",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
