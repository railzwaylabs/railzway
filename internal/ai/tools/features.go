package tools

import (
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
)

func defineFeatureTools(g *genkit.Genkit, features featuredomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_get_feature",
			"Get one feature by ID for AI Assistant entitlement context.",
			func(ctx *genkitai.ToolContext, input idInput) (featuredomain.FeatureResponse, error) {
				return features.GetByID(ctx.Context, featuredomain.GetFeatureRequest{ID: strings.TrimSpace(input.ID)})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_features",
			"List features for AI Assistant entitlement context.",
			func(ctx *genkitai.ToolContext, input featureListInput) (featuredomain.ListFeatureResponse, error) {
				return features.List(ctx.Context, featuredomain.ListFeatureRequest{
					PageSize:    normalizedPageSize(input.PageSize),
					Code:        strings.TrimSpace(input.Code),
					Name:        strings.TrimSpace(input.Name),
					FeatureType: strings.TrimSpace(input.FeatureType),
					Active:      input.Active,
				})
			},
		),
	}
}
