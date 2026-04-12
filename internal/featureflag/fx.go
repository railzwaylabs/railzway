package featureflag

import (
	"fmt"

	"github.com/railzwaylabs/railzway/internal/cache"
	"github.com/railzwaylabs/railzway/internal/featureflag/domain"
	"github.com/railzwaylabs/railzway/internal/featureflag/repository"
	"github.com/railzwaylabs/railzway/internal/featureflag/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// Module provides feature flag wiring with redis cache.
var Module = fx.Module("featureflag",
	fx.Provide(
		repository.NewRepository,
		fx.Annotate(
			func(repo domain.Repository, client *redis.Client) *service.Service {
				return service.NewService(repo, client)
			},
			fx.ParamTags("", fmt.Sprintf(`name:"%s"`, cache.ClientName)),
		),
	),
)
