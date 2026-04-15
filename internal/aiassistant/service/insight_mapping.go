package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
)

func buildPlanInsightOutput(insight *domain.Insight, planRec *domain.PlanRecommendation) domain.InsightOutput {
	now := time.Now().UTC()
	output := domain.InsightOutput{
		GeneratedAt: now,
		Drivers:     []domain.DriverItem{},
		Actions:     []domain.ActionItem{},
		Summary: domain.SummaryBlock{
			Headline: "Plan recommendation unavailable",
			Metric:   "—",
		},
		Confidence:  domain.ConfidenceBlock{Level: domain.ConfidenceLow, Note: "Limited data available"},
		DataQuality: "Insufficient usage and invoice data for plan recommendation.",
	}

	if insight == nil {
		return output
	}

	metric := normalizePlanRecommendationLabel(insight.Title)
	output.Summary = domain.SummaryBlock{
		Headline:   insight.Summary,
		Metric:     metric,
		MetricNote: "recommendation",
	}
	output.Drivers = mapInsightDrivers(insight)
	output.Actions = mapInsightActions(insight.Actions)
	output.Confidence = domain.ConfidenceBlock{
		Level: domain.ConfidenceLevel(insight.Confidence),
		Note:  "Based on recent usage and billing patterns",
	}
	output.DataQuality = "Usage and invoice data from the last 90 days."
	output.PlanRec = planRec

	return output
}

func buildChurnInsightOutput(insight *domain.Insight) domain.InsightOutput {
	now := time.Now().UTC()
	output := domain.InsightOutput{
		GeneratedAt: now,
		Drivers:     []domain.DriverItem{},
		Actions:     []domain.ActionItem{},
		Summary: domain.SummaryBlock{
			Headline: "Churn risk signals unavailable",
			Metric:   "—",
		},
		Confidence:  domain.ConfidenceBlock{Level: domain.ConfidenceLow, Note: "Limited data available"},
		DataQuality: "Insufficient usage and payment data for churn insights.",
	}

	if insight == nil {
		return output
	}

	metric := strings.Title(string(insight.Severity))
	output.Summary = domain.SummaryBlock{
		Headline:   insight.Summary,
		Metric:     metric,
		MetricNote: "risk level",
	}
	output.Drivers = mapInsightDrivers(insight)
	output.Actions = mapInsightActions(insight.Actions)
	output.Confidence = domain.ConfidenceBlock{
		Level: domain.ConfidenceLevel(insight.Confidence),
		Note:  "Based on usage change and payment behavior",
	}
	output.DataQuality = "Usage and payment data from the last 28-90 days."

	return output
}

func mapInsightDrivers(insight *domain.Insight) []domain.DriverItem {
	if insight == nil || len(insight.Drivers) == 0 {
		return []domain.DriverItem{}
	}
	items := make([]domain.DriverItem, 0, len(insight.Drivers))
	for i, driver := range insight.Drivers {
		impact := domain.ImpactLow
		switch {
		case i == 0:
			impact = domain.ImpactHigh
		case i == 1:
			impact = domain.ImpactMedium
		}
		items = append(items, domain.DriverItem{
			Label:  driver.Label,
			Detail: driver.Detail,
			Impact: impact,
		})
	}
	return items
}

func mapInsightActions(actions []domain.InsightAction) []domain.ActionItem {
	if len(actions) == 0 {
		return []domain.ActionItem{}
	}
	items := make([]domain.ActionItem, 0, len(actions))
	for i, action := range actions {
		style := domain.ActionSecondary
		if i == 0 {
			style = domain.ActionPrimary
		}
		items = append(items, domain.ActionItem{
			Key:   string(action.Type),
			Label: action.Label,
			Style: style,
			Path:  actionPath(action.Type),
		})
	}
	return items
}

func actionPath(actionType domain.InsightActionType) string {
	switch actionType {
	case domain.ActionInspectUsage:
		return "/usage"
	case domain.ActionReviewPlans:
		return "/plans"
	case domain.ActionReviewCustomers:
		return "/customers"
	case domain.ActionFlagReview:
		return "/audit-logs"
	case domain.ActionDraftOutreach:
		return "/customers"
	case domain.ActionDraftUpgradeNote:
		return "/customers"
	default:
		return ""
	}
}

func selectInsight(insights []domain.Insight, kind domain.InsightKind, customerRef string) *domain.Insight {
	var byKind []domain.Insight
	for _, insight := range insights {
		if insight.Kind == kind {
			byKind = append(byKind, insight)
		}
	}
	if len(byKind) == 0 {
		return nil
	}

	trimmed := strings.TrimSpace(customerRef)
	if trimmed != "" {
		if parsed, err := uuid.Parse(trimmed); err == nil {
			for _, insight := range byKind {
				if insight.CustomerID == parsed.String() {
					return &insight
				}
			}
		}
	}

	return &byKind[0]
}

func selectPlanSnapshot(snaps []domain.PlanFitSnapshot, customerRef string) *domain.PlanFitSnapshot {
	if len(snaps) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(customerRef)
	if trimmed != "" {
		if parsed, err := uuid.Parse(trimmed); err == nil {
			for _, snap := range snaps {
				if snap.CustomerID == parsed.String() {
					return &snap
				}
			}
		}
	}
	return &snaps[0]
}

func formatMoneyLabel(cents int64, currency string) string {
	if currency == "" {
		if cents == 0 {
			return "—"
		}
		return fmt.Sprintf("%d", cents)
	}
	amount := float64(cents) / 100.0
	switch strings.ToUpper(currency) {
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, strings.ToUpper(currency))
	}
}

func percentChange(current, previous int64) string {
	if previous == 0 {
		return "0%"
	}
	delta := (float64(current-previous) / float64(previous)) * 100
	if delta == 0 {
		return "0%"
	}
	if delta > 0 {
		return fmt.Sprintf("+%.0f%%", math.Round(delta))
	}
	return fmt.Sprintf("%.0f%%", math.Round(delta))
}
