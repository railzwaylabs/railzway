package usage

import (
	"github.com/smallbiznis/railzway/internal/cache"
	"github.com/smallbiznis/railzway/internal/usage/liveevents"
	"github.com/smallbiznis/railzway/internal/usage/repository"
	"github.com/smallbiznis/railzway/internal/usage/service"
	"github.com/smallbiznis/railzway/internal/usage/snapshot"
	"go.uber.org/fx"
)

var Module = fx.Module("usage.service",
	fx.Provide(cache.NewUsageResolverCache),
	fx.Provide(repository.ProvideSnapshot),
	liveevents.Module,
	fx.Provide(service.NewService),
	snapshot.Module,
)
