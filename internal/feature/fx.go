package feature

import (
	"github.com/railzwaylabs/railzway/internal/feature/repository"
	"github.com/railzwaylabs/railzway/internal/feature/service"
	"go.uber.org/fx"
)

// Module provides feature repository and service wiring.
var Module = fx.Module("feature",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
