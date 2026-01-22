package billingoverview

import (
	"github.com/smallbiznis/railzway/internal/billingoverview/service"
	"go.uber.org/fx"
)

var Module = fx.Module("billingoverview.service",
	fx.Provide(service.NewService),
)
