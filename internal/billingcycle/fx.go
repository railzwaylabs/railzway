package billingcycle

import (
	"github.com/railzwaylabs/railzway/internal/billingcycle/service"
	"go.uber.org/fx"
)

var Module = fx.Module("billingcycle.service",
	fx.Provide(service.NewService),
)
