package subscription

import (
	"github.com/railzwaylabs/railzway/internal/subscription/repository"
	"github.com/railzwaylabs/railzway/internal/subscription/scheduler"
	"github.com/railzwaylabs/railzway/internal/subscription/service"
	"go.uber.org/fx"
)

// Module provides subscription repository and service wiring.
var Module = fx.Module("subscription",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
	fx.Invoke(scheduler.StartClosePeriodScheduler),
)
