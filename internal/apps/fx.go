package apps

import (
	appsdomain "github.com/railzwaylabs/railzway/internal/apps/domain"
	"github.com/railzwaylabs/railzway/internal/apps/repository"
	"github.com/railzwaylabs/railzway/internal/apps/service"
	"go.uber.org/fx"
)

// Module provides apps catalog service wiring.
var Module = fx.Module("apps",
	fx.Provide(
		repository.NewRepository,
		fx.Annotate(
			service.NewService,
			fx.As(new(appsdomain.Service)),
		),
	),
)
