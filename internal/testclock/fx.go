package testclock

import (
	"github.com/railzwaylabs/railzway/internal/testclock/repository"
	"github.com/railzwaylabs/railzway/internal/testclock/service"
	"go.uber.org/fx"
)

// Module provides test clock repository and service wiring.
var Module = fx.Module("testclock",
	fx.Provide(
		repository.NewRepository,
		service.NewService,
	),
)
