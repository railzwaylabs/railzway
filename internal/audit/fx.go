package audit

import (
	"github.com/railzwaylabs/railzway/internal/audit/repository"
	"github.com/railzwaylabs/railzway/internal/audit/service"
	"go.uber.org/fx"
)

var Module = fx.Module("audit.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.NewService),
)
