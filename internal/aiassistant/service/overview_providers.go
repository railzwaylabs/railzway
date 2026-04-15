package service

import (
	"context"

	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
)

type overviewMetricsProvider interface {
	BuildSummaryCards(ctx context.Context) ([]domain.SummaryCard, error)
	BuildSignals(ctx context.Context) (domain.SignalPanel, error)
}

type guardrailProvider interface {
	OverviewGuardrails(ctx context.Context) (domain.GuardrailPanel, error)
}

type staticGuardrailProvider struct{}

func (staticGuardrailProvider) OverviewGuardrails(ctx context.Context) (domain.GuardrailPanel, error) {
	return domain.GuardrailPanel{
		Title:       "Guardrails",
		Description: "Safety and compliance controls",
		Items: []domain.GuardrailItem{
			{ID: "pii_mask", Title: "PII masked", Detail: "Sensitive fields are masked by default"},
			{ID: "read_only", Title: "Read-only", Detail: "No write actions executed"},
			{ID: "org_scope", Title: "Org scoped", Detail: "Context limited to active org"},
			{ID: "audit", Title: "Audited", Detail: "Run activity captured in audit log"},
		},
	}, nil
}

func NewGuardrailProvider() guardrailProvider {
	return staticGuardrailProvider{}
}

func buildWorkspaceConfig() domain.WorkspaceConfig {
	return domain.WorkspaceConfig{
		CustomerPlaceholder: "Customer ID, email, or company",
		PromptPlaceholder:   "Ask the assistant to explain trends or summarize billing events.",
		DefaultPrompt:       "Explain the latest invoice, highlight main usage drivers, and mention anomalies.",
		TimeRanges: []domain.Option{
			{Value: "30d", Label: "Last 30 days"},
			{Value: "90d", Label: "Last 90 days"},
			{Value: "12m", Label: "Last 12 months"},
		},
		Intents: []domain.Option{
			{Value: string(domain.IntentBilling), Label: "Explain Billing"},
			{Value: string(domain.IntentForecast), Label: "Forecasting"},
			{Value: string(domain.IntentPlan), Label: "Plan Recommendation"},
			{Value: string(domain.IntentChurn), Label: "Customer Churn"},
			{Value: string(domain.IntentProduct), Label: "Product Recommendation"},
		},
		MaskingEnabled: true,
	}
}
