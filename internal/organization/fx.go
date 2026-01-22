package organization

import (
	"github.com/smallbiznis/railzway/internal/organization/event"
	"github.com/smallbiznis/railzway/internal/organization/repository"
	"github.com/smallbiznis/railzway/internal/organization/service"
	"go.uber.org/fx"
)

var Module = fx.Module("organization.service",
	fx.Provide(repository.NewRepository),
	fx.Provide(event.NewOutboxPublisher),
	fx.Provide(service.NewService),
)
