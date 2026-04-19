package tools

import (
	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
)

func defineOrganizationTools(g *genkit.Genkit, organizations organizationdomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_get_organization",
			"Get one organization by ID for AI Assistant workspace context.",
			func(ctx *genkitai.ToolContext, input organizationIDInput) (*organizationdomain.OrganizationResponse, error) {
				return organizations.GetByID(ctx.Context, input.identifier())
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_organization_members",
			"List organization members for AI Assistant workspace context.",
			func(ctx *genkitai.ToolContext, input organizationIDInput) ([]organizationdomain.OrganizationMemberInfo, error) {
				return organizations.ListMembers(ctx.Context, input.identifier())
			},
		),
	}
}
