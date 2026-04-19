package tools

import (
	"strings"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

func defineUsageTools(g *genkit.Genkit, usage usagedomain.Service) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_get_meter",
			"Get one usage meter by ID for AI Assistant usage analysis context.",
			func(ctx *genkitai.ToolContext, input idInput) (usagedomain.MeterResponse, error) {
				return usage.GetMeterByID(ctx.Context, usagedomain.GetMeterRequest{ID: strings.TrimSpace(input.ID)})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_meters",
			"List usage meters for AI Assistant usage analysis context.",
			func(ctx *genkitai.ToolContext, input meterListInput) (usagedomain.ListMeterResponse, error) {
				return usage.ListMeters(ctx.Context, usagedomain.ListMeterRequest{
					PageSize: normalizedPageSize(input.PageSize),
					Code:     strings.TrimSpace(input.Code),
					Name:     strings.TrimSpace(input.Name),
					Active:   input.Active,
				})
			},
		),
		genkit.DefineTool(g, "ai_assistant_list_usage_events",
			"List recent usage events for AI Assistant usage anomaly and billing analysis.",
			func(ctx *genkitai.ToolContext, input usageListInput) (usagedomain.ListUsageResponse, error) {
				recordedFrom, recordedTo := recentWindow(input.Days)
				return usage.ListUsage(ctx.Context, usagedomain.ListUsageRequest{
					PageSize:     normalizedPageSize(input.PageSize),
					MeterID:      strings.TrimSpace(input.MeterID),
					CustomerID:   strings.TrimSpace(input.CustomerID),
					Status:       strings.TrimSpace(input.Status),
					RecordedFrom: recordedFrom,
					RecordedTo:   recordedTo,
				})
			},
		),
	}
}
