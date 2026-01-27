package testclock

import (
	"github.com/railzwaylabs/railzway/internal/testclock/domain"
	"github.com/railzwaylabs/railzway/internal/testclock/service"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		domain.NewRepository,
		service.New,
	),
)
