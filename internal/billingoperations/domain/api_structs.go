package domain

import (
	"encoding/json"
	"time"
)

// Constants
const (
	PeriodTypeDaily   = "daily"
	PeriodTypeWeekly  = "weekly"
	PeriodTypeMonthly = "monthly"

	ScoringVersionV1EqualWeight = "v1_equal_weight"
)

type GetPerformanceRequest struct {
	PeriodType string    `json:"period_type" form:"period_type" binding:"required"`
	From       time.Time `json:"from" form:"from"`
	To         time.Time `json:"to" form:"to"`
	Limit      int       `json:"limit" form:"limit"`
}

type PerformanceResponse struct {
	UserID         string        `json:"user_id"`
	PeriodType     string        `json:"period_type"`
	ScoringVersion string        `json:"scoring_version"`
	Snapshots      []APISnapshot `json:"snapshots"`
}

type APISnapshot struct {
	PeriodStart time.Time         `json:"period_start"`
	PeriodEnd   time.Time         `json:"period_end"`
	Metrics     APIMetrics        `json:"metrics"`
	Scores      PerformanceScores `json:"scores"`
	TotalScore  int               `json:"total_score"`
}

type APIMetrics struct {
	AvgResponseMinutes float64 `json:"avg_response_minutes"`
	CompletionRatio    float64 `json:"completion_ratio"`
	EscalationRatio    float64 `json:"escalation_ratio"`
	ExposureHandled    int64   `json:"exposure_handled"`
}

type TeamPerformanceResponse struct {
	PeriodType string              `json:"period_type"`
	TeamSize   int                 `json:"team_size"`
	Snapshots  []TeamMemberSummary `json:"snapshots"`
}

type TeamMemberSummary struct {
	UserID         string     `json:"user_id"`
	AvgScore       int        `json:"avg_score"`
	MetricsSummary APIMetrics `json:"metrics_summary"`
}

// Helper methods for JSON unmarshalling if needed by repository mapping
func (m *PerformanceMetrics) UnmarshalJSON(data []byte) error {
	type Alias PerformanceMetrics
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	return json.Unmarshal(data, &aux)
}

func (s *PerformanceScores) UnmarshalJSON(data []byte) error {
	type Alias PerformanceScores
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	return json.Unmarshal(data, &aux)
}
