package genkit

import (
	"context"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/railzwaylabs/railzway/internal/config"
	"go.uber.org/zap"
)

// NewGenkit initializes a Genkit runtime backed by Gemini for AI workflow planning.
func NewGenkit(cfg *config.Config) (*genkit.Genkit, error) {
	if cfg == nil || !cfg.AIWorkflowConfig.AIWorkflowGenkitEnabled {
		return nil, nil
	}

	apiKey := strings.TrimSpace(cfg.AIWorkflowConfig.AIWorkflowAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("aiworkflow: AI_WORKFLOW_API_KEY is required when AI_WORKFLOW_GENKIT_ENABLED=true")
	}

	model := strings.TrimSpace(cfg.AIWorkflowConfig.AIWorkflowModel)
	if model == "" {
		model = "googleai/gemini-2.5-flash"
	}

	ctx := context.Background()
	g := genkit.Init(ctx,
		genkit.WithDefaultModel(model),
		genkit.WithPlugins(
			&googlegenai.GoogleAI{APIKey: apiKey},
		),
	)

	zap.L().Info("aiworkflow genkit initialized", zap.String("model", model))
	return g, nil
}
