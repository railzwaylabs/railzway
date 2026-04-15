package service

import (
	"fmt"
	"math"
	"strings"

	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
)

func EvaluatePlanRecommendation(s domain.PlanFitSnapshot) *domain.Insight {
	if s.ConsecutiveHighUsage >= 3 || s.OveragePeriods >= 2 {
		return &domain.Insight{
			ID:         "plan_upgrade_" + s.CustomerID,
			Kind:       domain.InsightKindPlanRecommendation,
			CustomerID: s.CustomerID,
			Title:      "Upgrade plan recommended",
			Summary:    "Customer consistently operates near or above plan capacity.",
			Severity:   domain.SeverityWarning,
			Confidence: derivePlanConfidence(s),
			Drivers: []domain.InsightDriver{
				{
					Code:   "high_usage_ratio",
					Label:  "Usage near plan capacity",
					Detail: fmt.Sprintf("Usage reached %.0f%% of included capacity", s.UsagePctOfIncluded*100),
				},
				{
					Code:   "repeated_pattern",
					Label:  "Sustained pattern",
					Detail: fmt.Sprintf("High usage observed for %d consecutive periods", s.ConsecutiveHighUsage),
				},
			},
			Actions: []domain.InsightAction{
				{
					Type:   domain.ActionReviewPlans,
					Label:  "Review available plans",
					Reason: "Customer may be a strong fit for a higher tier",
				},
				{
					Type:   domain.ActionDraftUpgradeNote,
					Label:  "Draft upgrade recommendation",
					Reason: "Prepare a customer-facing upgrade explanation",
				},
			},
			ObservedAt: s.LastObservedAt,
		}
	}

	if s.ConsecutiveLowUsage >= 3 {
		return &domain.Insight{
			ID:         "plan_downsize_" + s.CustomerID,
			Kind:       domain.InsightKindPlanRecommendation,
			CustomerID: s.CustomerID,
			Title:      "Plan may be oversized",
			Summary:    "Customer consistently uses far below included plan capacity.",
			Severity:   domain.SeverityInfo,
			Confidence: derivePlanConfidence(s),
			Drivers: []domain.InsightDriver{
				{
					Code:   "low_usage_ratio",
					Label:  "Low utilization",
					Detail: fmt.Sprintf("Usage stayed at %.0f%% of included capacity", s.UsagePctOfIncluded*100),
				},
			},
			Actions: []domain.InsightAction{
				{
					Type:   domain.ActionInspectUsage,
					Label:  "Inspect usage trend",
					Reason: "Validate whether low usage is persistent",
				},
				{
					Type:   domain.ActionReviewPlans,
					Label:  "Review lower-tier plans",
					Reason: "Customer may be overpaying for unused capacity",
				},
			},
			ObservedAt: s.LastObservedAt,
		}
	}

	return nil
}

func EvaluateChurnRisk(s domain.ChurnRiskSnapshot) *domain.Insight {
	drivers := make([]domain.InsightDriver, 0, 4)

	if s.UsageDropPct >= 15 {
		drivers = append(drivers, domain.InsightDriver{
			Code:   "usage_drop",
			Label:  "Usage decline",
			Detail: fmt.Sprintf("Usage dropped %.0f%% over %d days", s.UsageDropPct, s.WindowDays),
		})
	}

	if s.InvoiceAmountDropPct >= 15 {
		drivers = append(drivers, domain.InsightDriver{
			Code:   "invoice_drop",
			Label:  "Revenue decline",
			Detail: fmt.Sprintf("Invoice totals down %.0f%% in the last %d days", s.InvoiceAmountDropPct, s.WindowDays),
		})
	}

	if s.HasOverdueInvoice {
		detail := "Invoice past due"
		if s.OverdueDays > 0 {
			detail = fmt.Sprintf("Invoice past due for %d days", s.OverdueDays)
		}
		drivers = append(drivers, domain.InsightDriver{
			Code:   "overdue_invoice",
			Label:  "Overdue invoice",
			Detail: detail,
		})
	}

	if s.LatePaymentCount90d > 0 {
		drivers = append(drivers, domain.InsightDriver{
			Code:   "late_payments",
			Label:  "Late payments",
			Detail: fmt.Sprintf("%d late payments in the last 90 days", s.LatePaymentCount90d),
		})
	}

	if len(drivers) == 0 {
		return nil
	}

	severity := domain.SeverityInfo
	if s.HasOverdueInvoice && (s.UsageDropPct >= 30 || s.InvoiceAmountDropPct >= 30) {
		severity = domain.SeverityRisk
	} else if s.UsageDropPct >= 25 || s.InvoiceAmountDropPct >= 25 || s.LatePaymentCount90d >= 2 || s.HasOverdueInvoice {
		severity = domain.SeverityWarning
	}

	title := "Churn risk detected"
	summary := "Usage and billing signals indicate potential churn risk."
	switch severity {
	case domain.SeverityRisk:
		title = "High churn risk"
		summary = "Material usage decline combined with billing risk signals."
	case domain.SeverityWarning:
		title = "Churn risk rising"
		summary = "Usage decline and payment behavior suggest elevated risk."
	}

	actions := []domain.InsightAction{
		{
			Type:   domain.ActionReviewCustomers,
			Label:  "Review at-risk customers",
			Reason: "Focus outreach on accounts with recent declines",
		},
		{
			Type:   domain.ActionInspectUsage,
			Label:  "Inspect usage drivers",
			Reason: "Validate whether usage drop is sustained",
		},
	}

	if severity == domain.SeverityRisk {
		actions = append(actions, domain.InsightAction{
			Type:   domain.ActionDraftOutreach,
			Label:  "Draft outreach message",
			Reason: "Prepare retention outreach while risk is high",
		})
	}

	if s.HasOverdueInvoice || s.LatePaymentCount90d > 0 {
		actions = append(actions, domain.InsightAction{
			Type:   domain.ActionFlagReview,
			Label:  "Flag for review",
			Reason: "Escalate payment risk for follow-up",
		})
	}

	return &domain.Insight{
		ID:         "churn_risk_" + s.CustomerID,
		Kind:       domain.InsightKindChurnRisk,
		CustomerID: s.CustomerID,
		Title:      title,
		Summary:    summary,
		Severity:   severity,
		Confidence: deriveChurnConfidence(s),
		Drivers:    drivers,
		Actions:    actions,
		ObservedAt: s.LastObservedAt,
	}
}

func derivePlanConfidence(s domain.PlanFitSnapshot) domain.InsightConfidence {
	score := 0
	if s.UsagePctOfIncluded > 0 {
		score++
	}
	if s.ConsecutiveHighUsage >= 3 || s.ConsecutiveLowUsage >= 3 {
		score++
	}
	if s.OveragePeriods >= 2 {
		score++
	}
	if s.AvgInvoiceAmount > 0 && s.LastInvoiceAmount > 0 {
		score++
	}

	return confidenceFromScore(score)
}

func deriveChurnConfidence(s domain.ChurnRiskSnapshot) domain.InsightConfidence {
	score := 0
	if s.UsageDropPct > 0 {
		score++
	}
	if s.InvoiceAmountDropPct > 0 {
		score++
	}
	if s.HasOverdueInvoice || s.LatePaymentCount90d > 0 {
		score++
	}
	if s.WindowDays >= 28 {
		score++
	}

	return confidenceFromScore(score)
}

func confidenceFromScore(score int) domain.InsightConfidence {
	switch {
	case score >= 4:
		return domain.InsightConfidenceHigh
	case score >= 2:
		return domain.InsightConfidenceMedium
	default:
		return domain.InsightConfidenceLow
	}
}

func normalizePlanRecommendationLabel(title string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.Contains(lower, "upgrade"):
		return "Upgrade"
	case strings.Contains(lower, "oversized") || strings.Contains(lower, "down"):
		return "Downsize"
	default:
		return "Plan fit"
	}
}

func planSavingsEstimate(usagePct float64) string {
	if usagePct <= 0 {
		return "—"
	}
	if usagePct >= 0.8 {
		return fmt.Sprintf("Usage %.0f%% of plan", math.Round(usagePct*100))
	}
	savings := math.Max(0, (1-usagePct)*100)
	return fmt.Sprintf("Est. %.0f%% savings", math.Round(savings))
}
