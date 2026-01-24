package organization

import (
	"github.com/railzwaylabs/railzway/internal/organization/event"
	"github.com/railzwaylabs/railzway/internal/organization/repository"
	"github.com/railzwaylabs/railzway/internal/organization/service"
	"go.uber.org/fx"
)

var Module = fx.Module("organization.service",
	fx.Provide(repository.NewRepository),
	fx.Provide(event.NewOutboxPublisher),
	fx.Provide(service.NewService),
)
