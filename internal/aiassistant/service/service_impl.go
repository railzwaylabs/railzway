package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type service struct {
	repo              domain.Repository
	overviewMetrics   overviewMetricsProvider
	guardrailProvider guardrailProvider
	insightSvc        *InsightService
	planRanking       planRankingProvider
}

func NewService(
	repo domain.Repository,
	overviewMetrics overviewMetricsProvider,
	guardrailProvider guardrailProvider,
	insightSvc *InsightService,
	planRanking planRankingProvider,
) domain.Service {
	return &service{
		repo:              repo,
		overviewMetrics:   overviewMetrics,
		guardrailProvider: guardrailProvider,
		insightSvc:        insightSvc,
		planRanking:       planRanking,
	}
}

func (s *service) CreateRun(ctx context.Context, req domain.CreateRunRequest) (domain.RunDetailResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.RunDetailResponse{}, domain.ErrInvalidOrganization
	}

	intent := normalizeIntent(req.Intent)
	if !isValidIntent(intent) {
		return domain.RunDetailResponse{}, domain.ErrInvalidIntent
	}

	timeRange := strings.TrimSpace(req.TimeRange)
	if !isValidTimeRange(timeRange) {
		return domain.RunDetailResponse{}, domain.ErrInvalidTimeRange
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return domain.RunDetailResponse{}, domain.ErrInvalidPrompt
	}

	customerRef := strings.TrimSpace(req.CustomerRef)
	now := time.Now().UTC()
	row := domain.Run{
		ID:          uuid.New(),
		OrgID:       orgID,
		CustomerRef: customerRef,
		TimeRange:   timeRange,
		Intent:      string(intent),
		Prompt:      prompt,
		Status:      domain.StatusQueued,
		Output:      json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateRun(ctx, row); err != nil {
		return domain.RunDetailResponse{}, err
	}

	s.enqueueRun(row)

	return domain.RunDetailResponse{Run: toRunDetail(row, nil)}, nil
}

func (s *service) GetRun(ctx context.Context, req domain.GetRunRequest) (domain.RunDetailResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.RunDetailResponse{}, domain.ErrInvalidOrganization
	}

	runID, err := uuid.Parse(strings.TrimSpace(req.ID))
	if err != nil || runID == uuid.Nil {
		return domain.RunDetailResponse{}, domain.ErrInvalidID
	}

	run, err := s.repo.FindRunByID(ctx, orgID, runID)
	if err != nil {
		return domain.RunDetailResponse{}, err
	}
	if run == nil {
		return domain.RunDetailResponse{}, domain.ErrNotFound
	}

	insight, err := decodeInsight(run.Output)
	if err != nil {
		return domain.RunDetailResponse{}, err
	}

	return domain.RunDetailResponse{Run: toRunDetail(*run, insight)}, nil
}

func (s *service) ListRuns(ctx context.Context, req domain.ListRunsRequest) (domain.ListRunsResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.ListRunsResponse{}, domain.ErrInvalidOrganization
	}

	pageSize := pagination.ValidatePageSize(int(req.PageSize))
	var cursor *domain.RunCursor
	if req.PageToken != "" {
		decoded, err := pagination.DecodeCursor(req.PageToken)
		if err != nil {
			return domain.ListRunsResponse{}, domain.ErrInvalidID
		}
		if decoded != nil && decoded.CreatedAt != "" && decoded.ID != "" {
			parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
			if err != nil {
				return domain.ListRunsResponse{}, domain.ErrInvalidID
			}
			parsedID, err := uuid.Parse(decoded.ID)
			if err != nil {
				return domain.ListRunsResponse{}, domain.ErrInvalidID
			}
			cursor = &domain.RunCursor{ID: parsedID, CreatedAt: parsedTime}
		}
	}

	items, err := s.repo.ListRuns(ctx, orgID, pageSize+1, cursor)
	if err != nil {
		return domain.ListRunsResponse{}, err
	}

	resp := domain.ListRunsResponse{}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *domain.Run) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(items) > pageSize {
			items = items[:pageSize]
		}
	}

	resp.Runs = make([]domain.RunHistoryItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Runs = append(resp.Runs, toRunHistory(*item))
	}
	return resp, nil
}

func (s *service) Overview(ctx context.Context) (domain.OverviewResponse, error) {
	runs, err := s.ListRuns(ctx, domain.ListRunsRequest{PageSize: 5})
	if err != nil {
		return domain.OverviewResponse{}, err
	}

	resp := domain.OverviewResponse{
		Workspace: buildWorkspaceConfig(),
		Runs:      runs,
	}

	summaryCards, err := s.overviewMetrics.BuildSummaryCards(ctx)
	if err != nil {
		return domain.OverviewResponse{}, err
	}
	resp.SummaryCards = summaryCards

	signals, err := s.overviewMetrics.BuildSignals(ctx)
	if err != nil {
		return domain.OverviewResponse{}, err
	}
	resp.Signals = signals

	guardrails, err := s.guardrailProvider.OverviewGuardrails(ctx)
	if err != nil {
		return domain.OverviewResponse{}, err
	}
	resp.Guardrails = guardrails

	if len(runs.Runs) > 0 {
		resp.ActiveRunID = runs.Runs[0].ID
	}
	return resp, nil
}

func (s *service) enqueueRun(run domain.Run) {
	go func() {
		_ = s.executeRun(context.Background(), run)
	}()
}

func (s *service) executeRun(ctx context.Context, run domain.Run) error {
	startedAt := time.Now().UTC()
	_ = s.repo.UpdateRun(ctx, run.OrgID, run.ID, map[string]interface{}{
		"status":     domain.StatusRunning,
		"started_at": startedAt,
		"updated_at": startedAt,
	})

	// Simulate async execution time.
	time.Sleep(500 * time.Millisecond)

	execCtx := orgcontext.WithOrgID(ctx, run.OrgID)
	insight, err := s.buildInsight(execCtx, run)
	if err != nil {
		finishedAt := time.Now().UTC()
		_ = s.repo.UpdateRun(ctx, run.OrgID, run.ID, map[string]interface{}{
			"status":      domain.StatusFailed,
			"error_code":  "generation_failed",
			"error_msg":   err.Error(),
			"finished_at": finishedAt,
			"updated_at":  finishedAt,
		})
		return err
	}

	payload, _ := json.Marshal(insight)
	finishedAt := time.Now().UTC()
	return s.repo.UpdateRun(ctx, run.OrgID, run.ID, map[string]interface{}{
		"status":      domain.StatusDone,
		"output":      payload,
		"finished_at": finishedAt,
		"updated_at":  finishedAt,
	})
}

func normalizeIntent(intent domain.Intent) domain.Intent {
	trimmed := strings.TrimSpace(string(intent))
	if trimmed == "" {
		return intent
	}
	return domain.Intent(strings.ToLower(trimmed))
}

func isValidIntent(intent domain.Intent) bool {
	switch intent {
	case domain.IntentBilling, domain.IntentForecast, domain.IntentPlan, domain.IntentChurn, domain.IntentProduct:
		return true
	default:
		return false
	}
}

func isValidTimeRange(value string) bool {
	switch strings.TrimSpace(value) {
	case "30d", "90d", "12m":
		return true
	default:
		return false
	}
}

func decodeInsight(raw json.RawMessage) (*domain.InsightOutput, error) {
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return nil, nil
	}
	var insight domain.InsightOutput
	if err := json.Unmarshal(raw, &insight); err != nil {
		return nil, err
	}
	return &insight, nil
}

func toRunDetail(run domain.Run, insight *domain.InsightOutput) domain.RunDetail {
	status := buildStatusBadge(run.Status)
	customerLabel := maskCustomer(run.CustomerRef)
	duration := computeDuration(run.StartedAt, run.FinishedAt)

	var runErr *domain.RunError
	if run.Status == domain.StatusFailed {
		runErr = &domain.RunError{
			Code:    run.ErrorCode,
			Message: run.ErrorMsg,
		}
	}

	return domain.RunDetail{
		ID:            run.ID.String(),
		Status:        status,
		Intent:        run.Intent,
		TimeRange:     run.TimeRange,
		Prompt:        run.Prompt,
		CustomerLabel: customerLabel,
		CreatedAt:     run.CreatedAt,
		UpdatedAt:     run.UpdatedAt,
		StartedAt:     run.StartedAt,
		FinishedAt:    run.FinishedAt,
		DurationMs:    duration,
		Insight:       insight,
		Error:         runErr,
	}
}

func toRunHistory(run domain.Run) domain.RunHistoryItem {
	status := buildStatusBadge(run.Status)
	customerLabel := maskCustomer(run.CustomerRef)
	duration := computeDuration(run.StartedAt, run.FinishedAt)
	intentLabel := strings.Title(run.Intent)

	title := "AI Insight"
	if intentLabel != "" {
		title = intentLabel + " insight"
	}
	subtitle := customerLabel
	if subtitle == "" {
		subtitle = "No customer selected"
	}

	return domain.RunHistoryItem{
		ID:            run.ID.String(),
		Title:         title,
		Subtitle:      subtitle,
		Intent:        run.Intent,
		CustomerLabel: customerLabel,
		Status:        status,
		CreatedAt:     run.CreatedAt,
		DurationMs:    duration,
	}
}

func buildStatusBadge(status domain.RunStatus) domain.StatusBadge {
	switch status {
	case domain.StatusDone:
		return domain.StatusBadge{Code: status, Label: "Done", Tone: "success"}
	case domain.StatusRunning:
		return domain.StatusBadge{Code: status, Label: "Running", Tone: "info"}
	case domain.StatusFailed:
		return domain.StatusBadge{Code: status, Label: "Failed", Tone: "danger"}
	default:
		return domain.StatusBadge{Code: status, Label: "Queued", Tone: "muted"}
	}
}

func computeDuration(start, end *time.Time) int64 {
	if start == nil || end == nil {
		return 0
	}
	return end.Sub(*start).Milliseconds()
}

func maskCustomer(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "@") {
		parts := strings.Split(trimmed, "@")
		if len(parts) >= 2 && len(parts[0]) > 1 {
			return parts[0][:1] + "***@" + parts[1]
		}
		return "***@" + parts[len(parts)-1]
	}
	if len(trimmed) <= 4 {
		return "***"
	}
	return trimmed[:2] + "***" + trimmed[len(trimmed)-2:]
}

func (s *service) buildInsight(ctx context.Context, run domain.Run) (*domain.InsightOutput, error) {
	now := time.Now().UTC()
	insight := &domain.InsightOutput{
		GeneratedAt: now,
	}

	intent := domain.Intent(run.Intent)
	switch intent {
	case domain.IntentForecast:
		insight.Actions = []domain.ActionItem{
			{Key: "usage_details", Label: "View usage details", Style: domain.ActionPrimary, Path: "/usage"},
			{Key: "invoice_trends", Label: "Review invoice trends", Style: domain.ActionSecondary, Path: "/invoices"},
			{Key: "flag_review", Label: "Flag for review", Style: domain.ActionSecondary, Path: "/audit-logs"},
		}
		insight.Summary = domain.SummaryBlock{
			Headline:   "Revenue forecast is stable with slight upside.",
			Metric:     "+6%",
			MetricNote: "next 90 days",
		}
		insight.Snapshot = &domain.SnapshotBlock{
			Label:    "Projected revenue",
			Previous: "$310k",
			Current:  "$328k",
			Delta:    "+$18k",
		}
		insight.Drivers = []domain.DriverItem{
			{Label: "Usage growth", Detail: "Top 10 customers trending up", Impact: domain.ImpactHigh},
			{Label: "Renewals", Detail: "Two renewals pending next month", Impact: domain.ImpactMedium},
			{Label: "Churn risk", Detail: "One enterprise account at risk", Impact: domain.ImpactLow},
		}
		insight.Confidence = domain.ConfidenceBlock{Level: domain.ConfidenceMedium, Note: "Forecast assumes current usage slope"}
		insight.DataQuality = "Usage coverage at 95% for the last 30 days."
	case domain.IntentPlan:
		if s.insightSvc != nil {
			insights, err := s.insightSvc.ListInsights(ctx)
			if err != nil {
				return nil, err
			}
			planInsight := selectInsight(insights, domain.InsightKindPlanRecommendation, run.CustomerRef)

			var snap *domain.PlanFitSnapshot
			snaps, err := s.insightSvc.ListPlanFitSnapshots(ctx, 50)
			if err != nil {
				return nil, err
			}
			snap = selectPlanSnapshot(snaps, run.CustomerRef)

			var planRec *domain.PlanRecommendation
			if s.planRanking != nil {
				planRec, err = s.planRanking.Recommend(ctx, planInsight, snap)
				if err != nil {
					return nil, err
				}
			}

			output := buildPlanInsightOutput(planInsight, planRec)
			return &output, nil
		}

		insight.Summary = domain.SummaryBlock{
			Headline:   "Plan recommendation unavailable",
			Metric:     "—",
			MetricNote: "recommendation",
		}
		insight.Confidence = domain.ConfidenceBlock{Level: domain.ConfidenceLow, Note: "Limited data available"}
		insight.DataQuality = "Insufficient data for plan recommendation."
	case domain.IntentChurn:
		if s.insightSvc != nil {
			insights, err := s.insightSvc.ListInsights(ctx)
			if err != nil {
				return nil, err
			}
			churnInsight := selectInsight(insights, domain.InsightKindChurnRisk, run.CustomerRef)
			output := buildChurnInsightOutput(churnInsight)
			return &output, nil
		}

		insight.Summary = domain.SummaryBlock{
			Headline:   "Churn risk signals unavailable",
			Metric:     "—",
			MetricNote: "risk level",
		}
		insight.Confidence = domain.ConfidenceBlock{Level: domain.ConfidenceLow, Note: "Limited data available"}
		insight.DataQuality = "Insufficient data for churn insight."
	case domain.IntentProduct:
		insight.Summary = domain.SummaryBlock{
			Headline:   "Usage patterns point to unmet needs in compliance reporting and cost controls.",
			Metric:     "3 ideas",
			MetricNote: "validated opportunities",
		}
		insight.Products = []domain.ProductRecommendation{
			{
				Name:                 "Usage Guardrails",
				TargetSegment:        "Mid-market SaaS",
				ValueProposition:     "Prevent overages with automated usage caps and alerts.",
				PricingModel:         "Usage-based add-on",
				PricingHint:          "$0.003 per event with monthly minimum",
				RequiredCapabilities: []string{"Real-time meter alerts", "Policy thresholds", "Webhook notifications"},
				ExpectedImpact:       "+4% retention, -12% bill shock tickets",
				Priority:             "high",
			},
			{
				Name:                 "Compliance Export Pack",
				TargetSegment:        "Enterprise finance",
				ValueProposition:     "One-click exports for SOC2, ISO, and tax audits.",
				PricingModel:         "Per workspace license",
				PricingHint:          "$499 / month per org",
				RequiredCapabilities: []string{"Ledger export templates", "Audit log bundling", "Role-based approval"},
				ExpectedImpact:       "+$18k MRR, lower churn in regulated accounts",
				Priority:             "medium",
			},
			{
				Name:                 "Cost Forecast Studio",
				TargetSegment:        "Growth teams",
				ValueProposition:     "Scenario modeling for upcoming usage spikes.",
				PricingModel:         "Tiered bundle",
				PricingHint:          "Included in Growth+ tiers",
				RequiredCapabilities: []string{"Forecast modeling", "Scenario snapshots", "Budget alerts"},
				ExpectedImpact:       "+2% upsell, +1.5% NRR",
				Priority:             "low",
			},
		}
		insight.Drivers = []domain.DriverItem{
			{Label: "Usage alerts demand", Detail: "42% of customers enable manual overage reviews", Impact: domain.ImpactHigh},
			{Label: "Compliance requests", Detail: "8 finance teams asked for export packs this quarter", Impact: domain.ImpactMedium},
			{Label: "Forecasting gap", Detail: "Usage spikes drive 3 of top 5 churn events", Impact: domain.ImpactMedium},
		}
		insight.Confidence = domain.ConfidenceBlock{Level: domain.ConfidenceMedium, Note: "Derived from last 90 days of usage and support tags"}
		insight.DataQuality = "Support tagging coverage at 86%; usage data complete."
		insight.Actions = []domain.ActionItem{
			{Key: "create_product", Label: "Create product draft", Style: domain.ActionPrimary, Path: "/products/new"},
			{Key: "create_plan", Label: "Create plan", Style: domain.ActionSecondary, Path: "/plans/new"},
			{Key: "create_meter", Label: "Create meter", Style: domain.ActionSecondary, Path: "/meters/new"},
			{Key: "create_feature", Label: "Create feature", Style: domain.ActionSecondary, Path: "/features/new"},
		}
	default:
		insight.Actions = []domain.ActionItem{
			{Key: "usage_details", Label: "View usage details", Style: domain.ActionPrimary, Path: "/usage"},
			{Key: "rating_results", Label: "Inspect rating results", Style: domain.ActionSecondary, Path: "/rating"},
			{Key: "customer_explain", Label: "Generate customer explanation", Style: domain.ActionSecondary, Path: "/customers"},
			{Key: "flag_review", Label: "Flag for review", Style: domain.ActionSecondary, Path: "/audit-logs"},
		}
		insight.Summary = domain.SummaryBlock{
			Headline:   "Invoice increased due to API usage spikes on core meters.",
			Metric:     "+18%",
			MetricNote: "vs last cycle",
		}
		insight.Snapshot = &domain.SnapshotBlock{
			Label:    "Invoice total",
			Previous: "$86.4k",
			Current:  "$102.1k",
			Delta:    "+$15.7k",
		}
		insight.Drivers = []domain.DriverItem{
			{Label: "API Usage", Detail: "Traffic up across 3 high-volume meters", Impact: domain.ImpactHigh},
			{Label: "Overage Fees", Detail: "Exceeded included tier by 12%", Impact: domain.ImpactMedium},
			{Label: "Discounts", Detail: "Promotional credits unchanged", Impact: domain.ImpactLow},
		}
		insight.Anomalies = []domain.AnomalyItem{{Title: "Usage burst", Detail: "02:00–04:00 UTC spike on /v1/usage", Severity: domain.AnomalyWatch}}
		insight.Proration = &domain.ProrationBlock{Title: "Proration applied", Detail: "Plan upgrade mid-cycle. 12 days prorated."}
		insight.Confidence = domain.ConfidenceBlock{Level: domain.ConfidenceHigh, Note: "98% of usage events processed"}
		insight.DataQuality = "2% events delayed; final totals may shift slightly."
	}

	return insight, nil
}
