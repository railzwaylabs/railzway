package usage

import (
	"github.com/railzwaylabs/railzway/internal/usage/repository"
	"github.com/railzwaylabs/railzway/internal/usage/service"
	"go.uber.org/fx"
)

// Module provides usage repository and service wiring.
var Module = fx.Module("usage",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
