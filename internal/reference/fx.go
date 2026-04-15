package reference

import (
	"fmt"

	"github.com/railzwaylabs/railzway/internal/cache"
	"github.com/railzwaylabs/railzway/internal/reference/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// Module provides reference lookup repository (countries/timezones).
var Module = fx.Module("reference",
	fx.Provide(
		fx.Annotate(
			NewRepository,
			fx.ResultTags(`name:"reference_base"`),
		),
		fx.Annotate(
			func(base domain.Repository, client *redis.Client) domain.Repository {
				return NewCachedRepository(base, client)
			},
			fx.ParamTags(`name:"reference_base"`, fmt.Sprintf(`name:"%s"`, cache.ClientName)),
		),
	),
)
