package integration

import (
	"github.com/railzwaylabs/railzway/internal/integration/domain"
	"github.com/railzwaylabs/railzway/internal/integration/provider/slack"
	"github.com/railzwaylabs/railzway/internal/integration/repository"
	"github.com/railzwaylabs/railzway/internal/integration/service"
	"go.uber.org/fx"
)

var Module = fx.Module("integration",
	fx.Provide(
		repository.New,
		service.New,
		service.NewDispatcher,
		func() map[string]domain.NotificationProvider {
			p := slack.NewProvider()
			return map[string]domain.NotificationProvider{
				"slack":   p,
				"discord": p,
			}
		},
	),
)
