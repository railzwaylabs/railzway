package tools

import (
	"errors"
	"fmt"
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
)

func defineInvoiceTools(g *genkit.Genkit, invoices invoicedomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_get_invoice",
			"Retrieve a single invoice by ID including its line items, amounts, and status. "+
				"Use this when the user asks about a specific invoice. Requires a valid invoice ID.",
			func(ctx *genkitai.ToolContext, input idInput) (invoicedomain.GetInvoiceResponse, error) {
				if err := ctx.Context.Err(); err != nil {
					return invoicedomain.GetInvoiceResponse{}, fmt.Errorf("request cancelled: %w", err)
				}

				id := strings.TrimSpace(input.ID)
				if id == "" {
					return invoicedomain.GetInvoiceResponse{}, errors.New("invoice ID is required")
				}

				return invoices.GetInvoice(ctx.Context, invoicedomain.GetInvoiceRequest{ID: strings.TrimSpace(input.ID)})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_invoices",
			"List invoices with optional filters. Use this when the user asks about multiple invoices, "+
				"overdue payments, or billing history. Supports filtering by customer, subscription, status, or invoice number.",
			func(ctx *genkitai.ToolContext, input invoiceListInput) (invoicedomain.ListInvoicesResponse, error) {
				if err := ctx.Context.Err(); err != nil {
					return invoicedomain.ListInvoicesResponse{}, fmt.Errorf("request cancelled: %w", err)
				}

				if input.CustomerID == "" && input.SubscriptionID == "" && input.Status == "" && input.Number == "" {
					return invoicedomain.ListInvoicesResponse{}, errors.New("at least one filter is required")
				}

				return invoices.ListInvoices(ctx.Context, invoicedomain.ListInvoicesRequest{
					PageSize:       normalizedPageSize(input.PageSize),
					CustomerID:     strings.TrimSpace(input.CustomerID),
					SubscriptionID: strings.TrimSpace(input.SubscriptionID),
					Status:         strings.TrimSpace(input.Status),
					Number:         strings.TrimSpace(input.Number),
				})
			},
		),
	}
}
