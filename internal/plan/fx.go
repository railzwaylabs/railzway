package plan

import (
	"github.com/railzwaylabs/railzway/internal/plan/repository"
	"github.com/railzwaylabs/railzway/internal/plan/service"
	"go.uber.org/fx"
)

// Module provides plan repository and service wiring.
var Module = fx.Module("plan",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
