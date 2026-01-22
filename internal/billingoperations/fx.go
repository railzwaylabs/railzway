package billingoperations

import (
	"github.com/smallbiznis/railzway/internal/billingoperations/service"
	"github.com/smallbiznis/railzway/internal/config"
	"go.uber.org/fx"
)

var Module = fx.Module("billingoperations.service",
	fx.Provide(service.NewService, config.NewBillingConfigHolder),
)
