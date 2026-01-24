package billingdashboard

import (
	"github.com/railzwaylabs/railzway/internal/billingdashboard/rollup"
	"github.com/railzwaylabs/railzway/internal/billingdashboard/service"
	"go.uber.org/fx"
)

var Module = fx.Module("billingdashboard.service",
	fx.Provide(service.NewService),
	fx.Provide(rollup.NewService),
)
