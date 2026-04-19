package tools

import (
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
)

func defineProductTools(g *genkit.Genkit, products productdomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_create_product",
			"Create a product when the user explicitly asks to set up a new product and the required fields are present.",
			func(ctx *genkitai.ToolContext, input createProductInput) (productdomain.ProductResponse, error) {
				return products.Create(ctx.Context, productdomain.CreateProductRequest{
					Code:        strings.TrimSpace(input.Code),
					Name:        strings.TrimSpace(input.Name),
					Description: input.Description,
					Active:      input.Active,
				})
			},
		),
		genkit.DefineTool(g, "ai_assistant_get_product",
			"Get one product by ID for AI Assistant product and packaging context.",
			func(ctx *genkitai.ToolContext, input idInput) (productdomain.ProductResponse, error) {
				return products.GetByID(ctx.Context, productdomain.GetProductRequest{
					ID:             strings.TrimSpace(input.ID),
					ExpandPlans:    true,
					ExpandFeatures: true,
				})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_products",
			"List products for AI Assistant packaging context.",
			func(ctx *genkitai.ToolContext, input productListInput) (productdomain.ListProductResponse, error) {
				return products.List(ctx.Context, productdomain.ListProductRequest{
					PageSize:       normalizedPageSize(input.PageSize),
					Code:           strings.TrimSpace(input.Code),
					Name:           strings.TrimSpace(input.Name),
					Active:         input.Active,
					ExpandPlans:    true,
					ExpandFeatures: true,
				})
			},
		),
	}
}
