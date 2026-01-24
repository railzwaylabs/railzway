package subscription

import (
	"github.com/railzwaylabs/railzway/internal/subscription/repository"
	"github.com/railzwaylabs/railzway/internal/subscription/service"
	"go.uber.org/fx"
)

var Module = fx.Module("subscription.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.NewService),
)
