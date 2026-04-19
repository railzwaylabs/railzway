package ai

import (
	"context"
	"encoding/json"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/railzwaylabs/railzway/internal/ai/assistant"
	"github.com/railzwaylabs/railzway/internal/ai/tools"
	aischeduler "github.com/railzwaylabs/railzway/internal/ai/scheduler"
	genkitmodule "github.com/railzwaylabs/railzway/internal/genkit"
	"go.uber.org/fx"
)

// Module wires the shared Genkit runtime and AI Assistant tool registry.
var Module = fx.Module("ai",
	genkitmodule.Module,
	fx.Provide(
		tools.NewAssistantToolset,
		func(toolset *tools.AssistantToolset) []genkitai.ToolRef {
			if toolset == nil {
				return nil
			}
			return toolset.All()
		},
		assistant.NewThreadStore,
		assistant.NewAssistantWorkflow,
		aischeduler.NewService,
		aischeduler.NewWorker,
	),
	fx.Invoke(func(lc fx.Lifecycle, worker *aischeduler.Worker) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				// Initialize default handlers
				worker.RegisterHandler("reminder", func(ctx context.Context, payload json.RawMessage) error {
					// For now, just log the reminder
					return nil
				})
				worker.Start(context.Background())
				return nil
			},
		})
	}),
)
