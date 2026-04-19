package tools

import (
	"errors"
	"fmt"
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
)

func defineSubscriptionTools(g *genkit.Genkit, subscriptions subscriptiondomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_get_subscription",
			"Get one subscription by ID for AI Assistant billing lifecycle context.",
			func(ctx *genkitai.ToolContext, input idInput) (subscriptiondomain.SubscriptionResponse, error) {
				if err := ctx.Context.Err(); err != nil {
					return subscriptiondomain.SubscriptionResponse{}, fmt.Errorf("request cancelled: %w", err)
				}
				if input.ID == "" {
					return subscriptiondomain.SubscriptionResponse{}, errors.New("subscription ID is required")
				}
				return subscriptions.GetSubscriptionByID(ctx.Context, subscriptiondomain.GetSubscriptionRequest{ID: strings.TrimSpace(input.ID)})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_subscriptions",
			"List subscriptions for AI Assistant account and billing lifecycle context.",
			func(ctx *genkitai.ToolContext, input subscriptionListInput) (subscriptiondomain.ListSubscriptionResponse, error) {
				if err := ctx.Context.Err(); err != nil {
					return subscriptiondomain.ListSubscriptionResponse{}, fmt.Errorf("request cancelled: %w", err)
				}

				if input.CustomerID == "" {
					return subscriptiondomain.ListSubscriptionResponse{}, errors.New("customer ID is required")
				}

				return subscriptions.ListSubscriptions(ctx.Context, subscriptiondomain.ListSubscriptionRequest{
					PageSize:   normalizedPageSize(input.PageSize),
					CustomerID: strings.TrimSpace(input.CustomerID),
					Status:     strings.TrimSpace(input.Status),
				})
			},
		),
	}
}
