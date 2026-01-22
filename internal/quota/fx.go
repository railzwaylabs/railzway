package quota

import (
	"github.com/smallbiznis/railzway/internal/quota/domain"
	"github.com/smallbiznis/railzway/internal/quota/service"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		domain.LoadFromEnv,
		service.NewService,
	),
	fx.Invoke(func(s domain.Service) {
		// Eager load service
	}),
)
