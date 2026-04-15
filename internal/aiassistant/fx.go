package aiassistant

import (
	"github.com/railzwaylabs/railzway/internal/aiassistant/repository"
	"github.com/railzwaylabs/railzway/internal/aiassistant/service"
	"go.uber.org/fx"
)

// Module provides AI assistant repository and service wiring.
var Module = fx.Module("aiassistant",
	fx.Provide(
		repository.NewRepository,
		service.NewOverviewMetricsProvider,
		service.NewGuardrailProvider,
		service.NewInsightReadModel,
		service.NewInsightService,
		service.NewPlanRankingProvider,
		service.NewService,
	),
)
