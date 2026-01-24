package invoice

import (
	"github.com/railzwaylabs/railzway/internal/invoice/render"
	"github.com/railzwaylabs/railzway/internal/invoice/service"
	"github.com/railzwaylabs/railzway/internal/tax"
	"go.uber.org/fx"
)

var Module = fx.Module("invoice.service",
	tax.Module,
	fx.Provide(render.NewRenderer),
	fx.Provide(service.NewService),
)
