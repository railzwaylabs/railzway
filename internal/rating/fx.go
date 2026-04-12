package rating

import (
	"github.com/railzwaylabs/railzway/internal/rating/repository"
	"github.com/railzwaylabs/railzway/internal/rating/scheduler"
	"github.com/railzwaylabs/railzway/internal/rating/service"
	"go.uber.org/fx"
)

// Module provides rating repository and service wiring.
var Module = fx.Module("rating",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
	fx.Invoke(scheduler.StartRatingScheduler),
)
