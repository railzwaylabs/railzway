package domain

import "time"

type InsightKind string

const (
	InsightKindPlanRecommendation InsightKind = "plan_recommendation"
	InsightKindChurnRisk          InsightKind = "churn_risk"
)

type InsightSeverity string

const (
	SeverityInfo    InsightSeverity = "info"
	SeverityWarning InsightSeverity = "warning"
	SeverityRisk    InsightSeverity = "risk"
)

type InsightConfidence string

const (
	InsightConfidenceLow    InsightConfidence = "low"
	InsightConfidenceMedium InsightConfidence = "medium"
	InsightConfidenceHigh   InsightConfidence = "high"
)

type InsightActionType string

const (
	ActionInspectUsage     InsightActionType = "inspect_usage"
	ActionReviewPlans      InsightActionType = "review_plans"
	ActionReviewCustomers  InsightActionType = "review_customers"
	ActionFlagReview       InsightActionType = "flag_review"
	ActionDraftOutreach    InsightActionType = "draft_outreach"
	ActionDraftUpgradeNote InsightActionType = "draft_upgrade_note"
)

type InsightDriver struct {
	Code   string `json:"code"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}

type InsightAction struct {
	Type   InsightActionType `json:"type"`
	Label  string            `json:"label"`
	Reason string            `json:"reason"`
}

type Insight struct {
	ID         string            `json:"id"`
	Kind       InsightKind       `json:"kind"`
	CustomerID string            `json:"customer_id,omitempty"`
	Title      string            `json:"title"`
	Summary    string            `json:"summary"`
	Severity   InsightSeverity   `json:"severity"`
	Confidence InsightConfidence `json:"confidence"`
	Drivers    []InsightDriver   `json:"drivers"`
	Actions    []InsightAction   `json:"actions"`
	ObservedAt time.Time         `json:"observed_at"`
}

type PlanFitSnapshot struct {
	CustomerID           string
	SubscriptionID       string
	PlanID               string
	PlanName             string
	WindowDays           int
	UsagePctOfIncluded   float64
	ConsecutiveHighUsage int
	ConsecutiveLowUsage  int
	OveragePeriods       int
	AvgInvoiceAmount     int64
	LastInvoiceAmount    int64
	Currency             string
	LastObservedAt       time.Time
}

type ChurnRiskSnapshot struct {
	CustomerID           string
	SubscriptionID       string
	PlanID               string
	WindowDays           int
	UsageDropPct         float64
	HasOverdueInvoice    bool
	OverdueDays          int
	LatePaymentCount90d  int
	InvoiceAmountDropPct float64
	LastObservedAt       time.Time
}
