package tools

import (
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
)

func definePlanTools(g *genkit.Genkit, plans plandomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_get_plan",
			"Get one plan by ID for AI Assistant plan recommendation and pricing context.",
			func(ctx *genkitai.ToolContext, input idInput) (plandomain.PlanResponse, error) {
				return plans.GetPlanByID(ctx.Context, plandomain.GetPlanRequest{ID: strings.TrimSpace(input.ID)})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_plans",
			"List plans for AI Assistant pricing and recommendation context.",
			func(ctx *genkitai.ToolContext, input planListInput) (plandomain.ListPlanResponse, error) {
				return plans.ListPlans(ctx.Context, plandomain.ListPlanRequest{
					PageSize:  normalizedPageSize(input.PageSize),
					ProductID: input.ProductID,
					Code:      strings.TrimSpace(input.Code),
					Name:      strings.TrimSpace(input.Name),
					Active:    input.Active,
				})
			},
		),
	}
}
