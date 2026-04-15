package aiworkflow

import (
	"github.com/railzwaylabs/railzway/internal/aiworkflow/repository"
	"github.com/railzwaylabs/railzway/internal/aiworkflow/service"
	"go.uber.org/fx"
)

// Module provides AI workflow repository and service wiring.
var Module = fx.Module("aiworkflow",
	fx.Provide(
		repository.NewRepository,
		service.NewActionExecutor,
		service.NewService,
	),
)
