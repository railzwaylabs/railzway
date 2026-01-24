package billingoperations

import (
	"github.com/railzwaylabs/railzway/internal/billingoperations/service"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/fx"
)

var Module = fx.Module("billingoperations.service",
	fx.Provide(service.NewService, config.NewBillingConfigHolder),
)
