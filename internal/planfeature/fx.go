package planfeature

import (
	"github.com/railzwaylabs/railzway/internal/planfeature/repository"
	"github.com/railzwaylabs/railzway/internal/planfeature/service"
	"go.uber.org/fx"
)

var Module = fx.Module("planfeature",
	fx.Provide(
		repository.NewRepository,
		service.New,
	),
)
