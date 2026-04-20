package coupon

import (
	"github.com/railzwaylabs/railzway/internal/coupon/repository"
	"github.com/railzwaylabs/railzway/internal/coupon/service"
	"go.uber.org/fx"
)

var Module = fx.Module("coupon",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
