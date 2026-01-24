package billingoverview

import (
	"github.com/railzwaylabs/railzway/internal/billingoverview/service"
	"go.uber.org/fx"
)

var Module = fx.Module("billingoverview.service",
	fx.Provide(service.NewService),
)
