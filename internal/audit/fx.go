package audit

import (
	"github.com/smallbiznis/railzway/internal/audit/repository"
	"github.com/smallbiznis/railzway/internal/audit/service"
	"go.uber.org/fx"
)

var Module = fx.Module("audit.service",
	fx.Provide(repository.Provide),
	fx.Provide(service.NewService),
)
