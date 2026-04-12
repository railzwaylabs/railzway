package organization

import (
	"github.com/railzwaylabs/railzway/internal/organization/repository"
	"github.com/railzwaylabs/railzway/internal/organization/service"
	"go.uber.org/fx"
)

// Module provides organization repository and service wiring.
var Module = fx.Module("organization",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
