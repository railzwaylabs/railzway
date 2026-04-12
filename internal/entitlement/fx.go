package entitlement

import (
	"github.com/railzwaylabs/railzway/internal/entitlement/service"
	"go.uber.org/fx"
)

var Module = fx.Module("entitlement",
	fx.Provide(service.NewService),
)
