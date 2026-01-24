package rating

import (
	"github.com/railzwaylabs/railzway/internal/rating/service"
	"go.uber.org/fx"
)

var Module = fx.Module("rating.service",
	fx.Provide(service.NewService),
)
