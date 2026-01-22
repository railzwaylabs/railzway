package invoice

import (
	"github.com/smallbiznis/railzway/internal/invoice/render"
	"github.com/smallbiznis/railzway/internal/invoice/service"
	"github.com/smallbiznis/railzway/internal/tax"
	"go.uber.org/fx"
)

var Module = fx.Module("invoice.service",
	tax.Module,
	fx.Provide(render.NewRenderer),
	fx.Provide(service.NewService),
)
