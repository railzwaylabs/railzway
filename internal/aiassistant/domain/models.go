package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
)

type Intent string

type RunStatus string

type ImpactLevel string

type ConfidenceLevel string

type ActionStyle string

type AnomalySeverity string

const (
	IntentBilling  Intent = "billing"
	IntentForecast Intent = "forecast"
	IntentPlan     Intent = "plan"
	IntentChurn    Intent = "churn"
	IntentProduct  Intent = "product_recommendation"
)

const (
	StatusQueued  RunStatus = "queued"
	StatusRunning RunStatus = "running"
	StatusDone    RunStatus = "done"
	StatusFailed  RunStatus = "failed"
)

const (
	ImpactHigh   ImpactLevel = "high"
	ImpactMedium ImpactLevel = "medium"
	ImpactLow    ImpactLevel = "low"
)

const (
	ConfidenceHigh   ConfidenceLevel = "high"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceLow    ConfidenceLevel = "low"
)

const (
	ActionPrimary   ActionStyle = "primary"
	ActionSecondary ActionStyle = "secondary"
)

const (
	AnomalyWatch AnomalySeverity = "watch"
	AnomalyRisk  AnomalySeverity = "risk"
)

var (
	ErrInvalidOrganization = errors.New("invalid_organization")
	ErrInvalidIntent       = errors.New("invalid_intent")
	ErrInvalidTimeRange    = errors.New("invalid_time_range")
	ErrInvalidPrompt       = errors.New("invalid_prompt")
	ErrInvalidID           = errors.New("invalid_id")
	ErrNotFound            = errors.New("not_found")
)

type CreateRunRequest struct {
	CustomerRef string
	TimeRange   string
	Intent      Intent
	Prompt      string
}

type GetRunRequest struct {
	ID string
}

type ListRunsRequest struct {
	PageToken string
	PageSize  int32
}

type ListRunsResponse struct {
	pagination.PageInfo
	Runs []RunHistoryItem `json:"runs"`
}

type OverviewResponse struct {
	Workspace    WorkspaceConfig  `json:"workspace"`
	SummaryCards []SummaryCard    `json:"summary_cards"`
	Signals      SignalPanel      `json:"signals"`
	Guardrails   GuardrailPanel   `json:"guardrails"`
	Runs         ListRunsResponse `json:"runs"`
	ActiveRunID  string           `json:"active_run_id,omitempty"`
}

type WorkspaceConfig struct {
	CustomerPlaceholder string   `json:"customer_placeholder"`
	PromptPlaceholder   string   `json:"prompt_placeholder"`
	DefaultPrompt       string   `json:"default_prompt"`
	TimeRanges          []Option `json:"time_ranges"`
	Intents             []Option `json:"intents"`
	MaskingEnabled      bool     `json:"masking_enabled"`
}

type Option struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type SummaryCard struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
	Sub   string `json:"sub"`
	Tone  string `json:"tone"`
	Delta string `json:"delta,omitempty"`
}

type SignalPanel struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Items       []SignalItem `json:"items"`
}

type SignalItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type GuardrailPanel struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Items       []GuardrailItem `json:"items"`
}

type GuardrailItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type RunDetailResponse struct {
	Run RunDetail `json:"run"`
}

type RunDetail struct {
	ID            string         `json:"id"`
	Status        StatusBadge    `json:"status"`
	Intent        string         `json:"intent"`
	TimeRange     string         `json:"time_range"`
	Prompt        string         `json:"prompt"`
	CustomerLabel string         `json:"customer_label"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	StartedAt     *time.Time     `json:"started_at,omitempty"`
	FinishedAt    *time.Time     `json:"finished_at,omitempty"`
	DurationMs    int64          `json:"duration_ms,omitempty"`
	Insight       *InsightOutput `json:"insight,omitempty"`
	Error         *RunError      `json:"error,omitempty"`
}

type RunHistoryItem struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Subtitle      string      `json:"subtitle"`
	Intent        string      `json:"intent"`
	CustomerLabel string      `json:"customer_label"`
	Status        StatusBadge `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	DurationMs    int64       `json:"duration_ms,omitempty"`
}

type StatusBadge struct {
	Code  RunStatus `json:"code"`
	Label string    `json:"label"`
	Tone  string    `json:"tone"`
}

type RunError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InsightOutput struct {
	Summary     SummaryBlock            `json:"summary"`
	Snapshot    *SnapshotBlock          `json:"snapshot,omitempty"`
	Drivers     []DriverItem            `json:"drivers"`
	Anomalies   []AnomalyItem           `json:"anomalies,omitempty"`
	Proration   *ProrationBlock         `json:"proration,omitempty"`
	PlanRec     *PlanRecommendation     `json:"plan_recommendation,omitempty"`
	Products    []ProductRecommendation `json:"product_recommendations,omitempty"`
	Confidence  ConfidenceBlock         `json:"confidence"`
	DataQuality string                  `json:"data_quality"`
	Actions     []ActionItem            `json:"actions"`
	GeneratedAt time.Time               `json:"generated_at"`
}

type SummaryBlock struct {
	Headline   string `json:"headline"`
	Metric     string `json:"metric"`
	MetricNote string `json:"metric_note"`
}

type SnapshotBlock struct {
	Label    string `json:"label"`
	Previous string `json:"previous"`
	Current  string `json:"current"`
	Delta    string `json:"delta"`
}

type DriverItem struct {
	Label  string      `json:"label"`
	Detail string      `json:"detail"`
	Impact ImpactLevel `json:"impact"`
}

type AnomalyItem struct {
	Title    string          `json:"title"`
	Detail   string          `json:"detail"`
	Severity AnomalySeverity `json:"severity"`
}

type ProrationBlock struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type ConfidenceBlock struct {
	Level ConfidenceLevel `json:"level"`
	Note  string          `json:"note"`
}

type ActionItem struct {
	Key      string      `json:"key"`
	Label    string      `json:"label"`
	Style    ActionStyle `json:"style"`
	Path     string      `json:"path"`
	Disabled bool        `json:"disabled,omitempty"`
}

type ProductRecommendation struct {
	Name                 string   `json:"name"`
	TargetSegment        string   `json:"target_segment"`
	ValueProposition     string   `json:"value_proposition"`
	PricingModel         string   `json:"pricing_model"`
	PricingHint          string   `json:"pricing_hint"`
	RequiredCapabilities []string `json:"required_capabilities"`
	ExpectedImpact       string   `json:"expected_impact"`
	Priority             string   `json:"priority"`
}

type PlanRecommendation struct {
	CurrentPlan     string `json:"current_plan"`
	RecommendedPlan string `json:"recommended_plan"`
	SavingsEstimate string `json:"savings_estimate"`
	BillingImpact   string `json:"billing_impact"`
	ReasonSummary   string `json:"reason_summary"`
}

type Run struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	OrgID       uuid.UUID       `gorm:"type:uuid;not null;index" json:"org_id"`
	CustomerRef string          `gorm:"type:text" json:"customer_ref"`
	TimeRange   string          `gorm:"type:text;not null" json:"time_range"`
	Intent      string          `gorm:"type:text;not null" json:"intent"`
	Prompt      string          `gorm:"type:text;not null" json:"prompt"`
	Status      RunStatus       `gorm:"type:text;not null" json:"status"`
	Output      json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"output"`
	ErrorCode   string          `gorm:"type:text" json:"error_code"`
	ErrorMsg    string          `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	StartedAt   *time.Time      `gorm:"" json:"started_at"`
	FinishedAt  *time.Time      `gorm:"" json:"finished_at"`
}

func (Run) TableName() string { return "ai_assistant_runs" }
