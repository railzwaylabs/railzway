package usage

import (
	"github.com/railzwaylabs/railzway/internal/cache"
	"github.com/railzwaylabs/railzway/internal/usage/liveevents"
	"github.com/railzwaylabs/railzway/internal/usage/repository"
	"github.com/railzwaylabs/railzway/internal/usage/service"
	"github.com/railzwaylabs/railzway/internal/usage/snapshot"
	"go.uber.org/fx"
)

var Module = fx.Module("usage.service",
	fx.Provide(cache.NewUsageResolverCache),
	fx.Provide(repository.ProvideSnapshot),
	liveevents.Module,
	fx.Provide(service.NewService),
	snapshot.Module,
)
