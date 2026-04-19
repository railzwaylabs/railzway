package tools

import (
	"errors"
	"fmt"
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
)

func defineCustomerTools(g *genkit.Genkit, customers customerdomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_get_customer",
			"Get one customer by ID for AI Assistant billing context.",
			func(ctx *genkitai.ToolContext, input idInput) (customerdomain.CustomerResponse, error) {
				if err := ctx.Context.Err(); err != nil {
					return customerdomain.CustomerResponse{}, fmt.Errorf("request cancelled: %w", err)
				}
				return customers.GetByID(ctx.Context, customerdomain.GetCustomerRequest{ID: strings.TrimSpace(input.ID)})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_customers",
			"List customers for AI Assistant context and mention resolution.",
			func(ctx *genkitai.ToolContext, input customerListInput) (customerdomain.ListCustomerResponse, error) {
				if err := ctx.Context.Err(); err != nil {
					return customerdomain.ListCustomerResponse{}, fmt.Errorf("request cancelled: %w", err)
				}

				if input.Name == "" && input.Email == "" && input.Currency == "" {
					return customerdomain.ListCustomerResponse{}, errors.New("at least one filter is required")
				}

				return customers.List(ctx.Context, customerdomain.ListCustomerRequest{
					PageSize: normalizedPageSize(input.PageSize),
					Name:     strings.TrimSpace(input.Name),
					Email:    strings.TrimSpace(input.Email),
					Currency: strings.TrimSpace(input.Currency),
				})
			},
		),
	}
}
