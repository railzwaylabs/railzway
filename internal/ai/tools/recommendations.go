package tools

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

type CustomerRecommendation struct {
	CustomerID              string   `json:"customer_id"`
	LifecycleStage          string   `json:"lifecycle_stage"`
	TenureDays              int      `json:"tenure_days"`
	WindowDays              int      `json:"window_days"`
	ActiveSubscriptionCount int      `json:"active_subscription_count"`
	RecentUsageEvents       int      `json:"recent_usage_events"`
	OpenInvoiceCount        int      `json:"open_invoice_count"`
	PaidInvoiceCount        int      `json:"paid_invoice_count"`
	Headline                string   `json:"headline"`
	Rationale               []string `json:"rationale"`
	Recommendations         []string `json:"recommendations"`
}

type PlanFitAnalysis struct {
	CustomerID         string   `json:"customer_id"`
	SubscriptionID     string   `json:"subscription_id,omitempty"`
	PlanID             string   `json:"plan_id,omitempty"`
	SubscriptionStatus string   `json:"subscription_status,omitempty"`
	CurrentPeriodStart string   `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   string   `json:"current_period_end,omitempty"`
	SubscriptionItems  int      `json:"subscription_items"`
	CommittedQuantity  float64  `json:"committed_quantity"`
	RecentUsageEvents  int      `json:"recent_usage_events"`
	RecentUsageValue   float64  `json:"recent_usage_value"`
	UsagePerDay        float64  `json:"usage_per_day"`
	Fit                string   `json:"fit"`
	Confidence         string   `json:"confidence"`
	Headline           string   `json:"headline"`
	Signals            []string `json:"signals"`
	Recommendations    []string `json:"recommendations"`
}

type usageAnomalyInput struct {
	CustomerID   string  `json:"customer_id" jsonschema:"description=Customer identifier"`
	Days         int     `json:"days,omitempty" jsonschema:"description=Recent comparison window in days"`
	ThresholdPct float64 `json:"threshold_pct,omitempty" jsonschema:"description=Minimum percent change required to flag an anomaly"`
}

type UsageAnomalyMetric struct {
	MeterID        string  `json:"meter_id,omitempty"`
	MeterCode      string  `json:"meter_code,omitempty"`
	RecentEvents   int     `json:"recent_events"`
	BaselineEvents int     `json:"baseline_events"`
	RecentValue    float64 `json:"recent_value"`
	BaselineValue  float64 `json:"baseline_value"`
	ChangePct      float64 `json:"change_pct"`
	Direction      string  `json:"direction"`
	Flagged        bool    `json:"flagged"`
}

type UsageAnomalyReport struct {
	CustomerID      string               `json:"customer_id"`
	WindowDays      int                  `json:"window_days"`
	ThresholdPct    float64              `json:"threshold_pct"`
	Flagged         bool                 `json:"flagged"`
	Headline        string               `json:"headline"`
	Metrics         []UsageAnomalyMetric `json:"metrics"`
	Recommendations []string             `json:"recommendations"`
}

type customerOptionalInput struct {
	CustomerID string `json:"customer_id,omitempty" jsonschema:"description=Optional customer identifier"`
}

type InvoiceAgingBucket struct {
	Bucket         string `json:"bucket"`
	InvoiceCount   int    `json:"invoice_count"`
	AmountDueCents int64  `json:"amount_due_cents"`
}

type InvoiceAgingReport struct {
	CustomerID          string               `json:"customer_id,omitempty"`
	AsOf                string               `json:"as_of"`
	TotalOpenInvoices   int                  `json:"total_open_invoices"`
	TotalAmountDueCents int64                `json:"total_amount_due_cents"`
	Buckets             []InvoiceAgingBucket `json:"buckets"`
	Headline            string               `json:"headline"`
	Recommendations     []string             `json:"recommendations"`
}

type SubscriptionHealth struct {
	SubscriptionID           string   `json:"subscription_id"`
	CustomerID               string   `json:"customer_id"`
	PlanID                   string   `json:"plan_id"`
	Status                   string   `json:"status"`
	CurrentPeriodStart       string   `json:"current_period_start"`
	CurrentPeriodEnd         string   `json:"current_period_end"`
	DaysUntilRenewal         int      `json:"days_until_renewal"`
	UpcomingRenewal          bool     `json:"upcoming_renewal"`
	CancelScheduled          bool     `json:"cancel_scheduled"`
	OpenInvoiceCount         int      `json:"open_invoice_count"`
	OpenAmountDueCents       int64    `json:"open_amount_due_cents"`
	CurrentPeriodUsageEvents int      `json:"current_period_usage_events"`
	CurrentPeriodUsageValue  float64  `json:"current_period_usage_value"`
	Health                   string   `json:"health"`
	Headline                 string   `json:"headline"`
	Signals                  []string `json:"signals"`
	Recommendations          []string `json:"recommendations"`
}

func defineRecommendationTools(
	g *genkit.Genkit,
	customers customerdomain.Service,
	subscriptions subscriptiondomain.Service,
	invoices invoicedomain.Service,
	usage usagedomain.Service,
) []genkitai.ToolRef {
	return []genkitai.ToolRef{
		genkit.DefineTool(g, "ai_assistant_analyze_plan_fit",
			"Check whether a customer's current subscription plan broadly matches their actual usage pattern. "+
				"Use when user asks if a customer is over or under their plan, or to recommend a plan change. "+
				"Input id must be a customer ID.",
			func(ctx *genkitai.ToolContext, input idInput) (PlanFitAnalysis, error) {
				if err := ctx.Context.Err(); err != nil {
					return PlanFitAnalysis{}, fmt.Errorf("request cancelled: %w", err)
				}
				customerID := strings.TrimSpace(input.ID)
				if customerID == "" {
					return PlanFitAnalysis{}, errors.New("customer id is required")
				}

				subscriptionResp, err := subscriptions.ListSubscriptions(ctx.Context, subscriptiondomain.ListSubscriptionRequest{
					PageSize:   maxToolPageSize,
					CustomerID: customerID,
				})
				if err != nil {
					return PlanFitAnalysis{}, fmt.Errorf("failed to fetch subscriptions for customer %s: %w", customerID, err)
				}
				subscription, ok := pickPrimarySubscription(subscriptionResp.Subscriptions)
				if !ok {
					return PlanFitAnalysis{}, errors.New("no subscription found for customer")
				}

				usageResp, err := usage.ListUsage(ctx.Context, usagedomain.ListUsageRequest{
					PageSize:     maxToolPageSize,
					CustomerID:   customerID,
					RecordedFrom: &subscription.CurrentPeriodStart,
					RecordedTo:   &subscription.CurrentPeriodEnd,
				})
				if err != nil {
					return PlanFitAnalysis{}, fmt.Errorf("failed to fetch usage for customer %s: %w", customerID, err)
				}

				return buildPlanFitAnalysis(customerID, subscription, usageResp.Events), nil
			},
		),
		genkit.DefineTool(g, "ai_assistant_detect_usage_anomaly",
			"Compare a customer's recent usage against their historical average and flag significant spikes or drops. "+
				"Use when user asks why a bill is high, or if usage looks unusual.",
			func(ctx *genkitai.ToolContext, input usageAnomalyInput) (UsageAnomalyReport, error) {
				if err := ctx.Context.Err(); err != nil {
					return UsageAnomalyReport{}, fmt.Errorf("request cancelled: %w", err)
				}
				customerID := strings.TrimSpace(input.CustomerID)
				if customerID == "" {
					return UsageAnomalyReport{}, errors.New("customer_id is required")
				}

				windowDays := input.Days
				if windowDays <= 0 {
					windowDays = 30
				}
				if windowDays > 180 {
					windowDays = 180
				}
				thresholdPct := input.ThresholdPct
				if thresholdPct <= 0 {
					thresholdPct = 50
				}

				now := time.Now().UTC()
				recentFrom := now.AddDate(0, 0, -windowDays)
				baselineFrom := recentFrom.AddDate(0, 0, -windowDays)

				recentResp, err := usage.ListUsage(ctx.Context, usagedomain.ListUsageRequest{
					PageSize:     maxToolPageSize,
					CustomerID:   customerID,
					RecordedFrom: &recentFrom,
					RecordedTo:   &now,
				})
				if err != nil {
					return UsageAnomalyReport{}, fmt.Errorf("failed to fetch recent usage for customer %s: %w", customerID, err)
				}
				baselineResp, err := usage.ListUsage(ctx.Context, usagedomain.ListUsageRequest{
					PageSize:     maxToolPageSize,
					CustomerID:   customerID,
					RecordedFrom: &baselineFrom,
					RecordedTo:   &recentFrom,
				})
				if err != nil {
					return UsageAnomalyReport{}, fmt.Errorf("failed to fetch baseline usage for customer %s: %w", customerID, err)
				}

				return buildUsageAnomalyReport(customerID, recentResp.Events, baselineResp.Events, windowDays, thresholdPct), nil
			},
		),
		genkit.DefineTool(g, "ai_assistant_invoice_aging",
			"Return an aging summary of unpaid invoices grouped by days overdue (0-30, 31-60, 61-90, 90+). "+
				"Use when user asks about overdue invoices, collection risk, or cash flow exposure.",
			func(ctx *genkitai.ToolContext, input customerOptionalInput) (InvoiceAgingReport, error) {
				if err := ctx.Context.Err(); err != nil {
					return InvoiceAgingReport{}, fmt.Errorf("request cancelled: %w", err)
				}
				invoiceResp, err := invoices.ListInvoices(ctx.Context, invoicedomain.ListInvoicesRequest{
					PageSize:   maxToolPageSize,
					CustomerID: strings.TrimSpace(input.CustomerID),
					Status:     invoicedomain.StatusOpen,
				})
				if err != nil {
					return InvoiceAgingReport{}, fmt.Errorf("failed to fetch open invoices: %w", err)
				}
				return buildInvoiceAgingReport(strings.TrimSpace(input.CustomerID), invoiceResp.Invoices), nil
			},
		),
		genkit.DefineTool(g, "ai_assistant_get_subscription_health",
			"Check the health of a subscription including status, current period, upcoming renewal, "+
				"and whether usage is within plan limits. Use when user asks if a subscription is at risk or due for renewal.",
			func(ctx *genkitai.ToolContext, input idInput) (SubscriptionHealth, error) {
				if err := ctx.Context.Err(); err != nil {
					return SubscriptionHealth{}, fmt.Errorf("request cancelled: %w", err)
				}
				subscriptionID := strings.TrimSpace(input.ID)
				if subscriptionID == "" {
					return SubscriptionHealth{}, errors.New("subscription id is required")
				}

				subscription, err := subscriptions.GetSubscriptionByID(ctx.Context, subscriptiondomain.GetSubscriptionRequest{ID: subscriptionID})
				if err != nil {
					return SubscriptionHealth{}, fmt.Errorf("failed to fetch subscription %s: %w", subscriptionID, err)
				}
				usageResp, err := usage.ListUsage(ctx.Context, usagedomain.ListUsageRequest{
					PageSize:     maxToolPageSize,
					CustomerID:   subscription.CustomerID,
					RecordedFrom: &subscription.CurrentPeriodStart,
					RecordedTo:   &subscription.CurrentPeriodEnd,
				})
				if err != nil {
					return SubscriptionHealth{}, fmt.Errorf("failed to fetch usage for subscription %s: %w", subscriptionID, err)
				}
				invoiceResp, err := invoices.ListInvoices(ctx.Context, invoicedomain.ListInvoicesRequest{
					PageSize:       maxToolPageSize,
					SubscriptionID: subscription.ID,
					Status:         invoicedomain.StatusOpen,
				})
				if err != nil {
					return SubscriptionHealth{}, fmt.Errorf("failed to fetch invoices for subscription %s: %w", subscriptionID, err)
				}

				return buildSubscriptionHealth(subscription, usageResp.Events, invoiceResp.Invoices), nil
			},
		),
		genkit.DefineTool(g, "ai_assistant_recommend_customer_strategy",
			"Analyze a customer's health and recommend next actions. "+
				"Returns lifecycle stage, billing risk signals, and actionable recommendations. "+
				"Use this when the user asks what to do with a customer, how to grow an account, or how to handle billing risk. "+
				"Requires a valid customer ID. Optionally accepts a window in days (default 30, max 180).",
			func(ctx *genkitai.ToolContext, input customerRecommendationInput) (CustomerRecommendation, error) {
				if err := ctx.Context.Err(); err != nil {
					return CustomerRecommendation{}, fmt.Errorf("request cancelled: %w", err)
				}

				customerID := strings.TrimSpace(input.CustomerID)
				if customerID == "" {
					return CustomerRecommendation{}, errors.New("customer_id is required")
				}

				windowDays := input.Days
				if windowDays <= 0 {
					windowDays = 30
				}
				if windowDays > 180 {
					windowDays = 180
				}

				customer, err := customers.GetByID(ctx.Context, customerdomain.GetCustomerRequest{
					ID: strings.TrimSpace(input.CustomerID),
				})
				if err != nil {
					return CustomerRecommendation{}, fmt.Errorf("failed to fetch customer %s: %w", customerID, err)
				}

				subscriptionResp, err := subscriptions.ListSubscriptions(ctx.Context, subscriptiondomain.ListSubscriptionRequest{
					PageSize:   maxToolPageSize,
					CustomerID: customer.ID,
				})
				if err != nil {
					return CustomerRecommendation{}, fmt.Errorf("failed to fetch customer %s: %w", customerID, err)
				}

				from, to := recentWindow(windowDays)
				invoiceResp, err := invoices.ListInvoices(ctx.Context, invoicedomain.ListInvoicesRequest{
					PageSize:    maxToolPageSize,
					CustomerID:  customer.ID,
					CreatedFrom: from,
					CreatedTo:   to,
				})
				if err != nil {
					return CustomerRecommendation{}, fmt.Errorf("failed to fetch customer %s: %w", customerID, err)
				}

				usageResp, err := usage.ListUsage(ctx.Context, usagedomain.ListUsageRequest{
					PageSize:     maxToolPageSize,
					CustomerID:   customer.ID,
					RecordedFrom: from,
					RecordedTo:   to,
				})
				if err != nil {
					return CustomerRecommendation{}, fmt.Errorf("failed to fetch customer %s: %w", customerID, err)
				}

				return buildCustomerRecommendation(customer, subscriptionResp.Subscriptions, invoiceResp.Invoices, usageResp.Events, windowDays), nil
			},
		),
	}
}

func pickPrimarySubscription(subscriptions []subscriptiondomain.SubscriptionResponse) (subscriptiondomain.SubscriptionResponse, bool) {
	for _, subscription := range subscriptions {
		if subscription.Status == subscriptiondomain.StatusActive || subscription.Status == subscriptiondomain.StatusTrialing {
			return subscription, true
		}
	}
	if len(subscriptions) == 0 {
		return subscriptiondomain.SubscriptionResponse{}, false
	}
	sort.Slice(subscriptions, func(i, j int) bool {
		return subscriptions[i].UpdatedAt.After(subscriptions[j].UpdatedAt)
	})
	return subscriptions[0], true
}

func sumUsageValue(events []usagedomain.UsageEventResponse) float64 {
	var total float64
	for _, event := range events {
		total += event.Value
	}
	return total
}

func buildPlanFitAnalysis(
	customerID string,
	subscription subscriptiondomain.SubscriptionResponse,
	events []usagedomain.UsageEventResponse,
) PlanFitAnalysis {
	periodDays := subscription.CurrentPeriodEnd.Sub(subscription.CurrentPeriodStart).Hours() / 24
	if periodDays <= 0 {
		periodDays = 30
	}

	var committedQuantity float64
	for _, item := range subscription.Items {
		committedQuantity += item.Quantity
	}

	usageValue := sumUsageValue(events)
	usagePerDay := usageValue / periodDays
	fit := "matched"
	confidence := "medium"
	headline := "Current plan broadly matches recent usage."
	signals := []string{
		fmt.Sprintf("Subscription status is %s.", subscription.Status),
		fmt.Sprintf("Observed %d usage events in the current period.", len(events)),
	}
	recommendations := []string{
		"Keep the current plan unless packaging, pricing, or invoice signals suggest a mismatch.",
		"Ground any plan change in invoice history and the current billing objective.",
	}

	switch {
	case len(events) == 0:
		fit = "under_utilized"
		confidence = "low"
		headline = "Current plan looks under-utilized based on recent activity."
		recommendations = []string{
			"Confirm whether onboarding or product adoption is still in progress before downgrading.",
			"Consider a simpler plan only if low activity persists across multiple billing windows.",
		}
	case committedQuantity > 0 && usageValue > committedQuantity*1.2:
		fit = "over_utilized"
		headline = "Current plan may be too small for observed usage."
		signals = append(signals, fmt.Sprintf("Observed usage value %.2f is materially above committed quantity %.2f.", usageValue, committedQuantity))
		recommendations = []string{
			"Review recent invoices and variable charges before recommending an upgrade.",
			"Consider a higher tier or an add-on if the current pattern is expected to continue.",
		}
	case committedQuantity > 0 && usageValue < math.Max(committedQuantity*0.35, 1):
		fit = "under_utilized"
		headline = "Current plan may be larger than the customer's recent usage pattern."
		signals = append(signals, fmt.Sprintf("Observed usage value %.2f is well below committed quantity %.2f.", usageValue, committedQuantity))
		recommendations = []string{
			"Check whether the customer bought headroom intentionally before recommending a downgrade.",
			"Use a longer observation window if the account is seasonal or enterprise-driven.",
		}
	}

	return PlanFitAnalysis{
		CustomerID:         customerID,
		SubscriptionID:     subscription.ID,
		PlanID:             subscription.PlanID,
		SubscriptionStatus: subscription.Status,
		CurrentPeriodStart: subscription.CurrentPeriodStart.Format(time.RFC3339),
		CurrentPeriodEnd:   subscription.CurrentPeriodEnd.Format(time.RFC3339),
		SubscriptionItems:  len(subscription.Items),
		CommittedQuantity:  committedQuantity,
		RecentUsageEvents:  len(events),
		RecentUsageValue:   usageValue,
		UsagePerDay:        usagePerDay,
		Fit:                fit,
		Confidence:         confidence,
		Headline:           headline,
		Signals:            signals,
		Recommendations:    recommendations,
	}
}

func buildUsageAnomalyReport(
	customerID string,
	recent []usagedomain.UsageEventResponse,
	baseline []usagedomain.UsageEventResponse,
	windowDays int,
	thresholdPct float64,
) UsageAnomalyReport {
	type aggregate struct {
		meterID   string
		meterCode string
		events    int
		value     float64
	}

	recentAgg := map[string]*aggregate{}
	baselineAgg := map[string]*aggregate{}
	merge := func(target map[string]*aggregate, events []usagedomain.UsageEventResponse) {
		for _, event := range events {
			key := strings.TrimSpace(event.MeterID)
			if key == "" {
				key = strings.TrimSpace(event.MeterCode)
			}
			if key == "" {
				key = "unknown"
			}
			item := target[key]
			if item == nil {
				item = &aggregate{meterID: event.MeterID, meterCode: event.MeterCode}
				target[key] = item
			}
			item.events++
			item.value += event.Value
		}
	}
	merge(recentAgg, recent)
	merge(baselineAgg, baseline)

	keys := map[string]struct{}{}
	for key := range recentAgg {
		keys[key] = struct{}{}
	}
	for key := range baselineAgg {
		keys[key] = struct{}{}
	}

	metrics := make([]UsageAnomalyMetric, 0, len(keys))
	flagged := false
	for key := range keys {
		recentItem := recentAgg[key]
		baselineItem := baselineAgg[key]
		var recentEvents, baselineEvents int
		var recentValue, baselineValue float64
		var meterID, meterCode string
		if recentItem != nil {
			recentEvents = recentItem.events
			recentValue = recentItem.value
			meterID = recentItem.meterID
			meterCode = recentItem.meterCode
		}
		if baselineItem != nil {
			baselineEvents = baselineItem.events
			baselineValue = baselineItem.value
			if meterID == "" {
				meterID = baselineItem.meterID
			}
			if meterCode == "" {
				meterCode = baselineItem.meterCode
			}
		}

		changePct := 0.0
		direction := "flat"
		switch {
		case baselineValue == 0 && recentValue > 0:
			changePct = 100
			direction = "up"
		case baselineValue > 0:
			changePct = ((recentValue - baselineValue) / baselineValue) * 100
			if changePct > 0 {
				direction = "up"
			} else if changePct < 0 {
				direction = "down"
			}
		}
		isFlagged := math.Abs(changePct) >= thresholdPct
		flagged = flagged || isFlagged
		metrics = append(metrics, UsageAnomalyMetric{
			MeterID:        meterID,
			MeterCode:      meterCode,
			RecentEvents:   recentEvents,
			BaselineEvents: baselineEvents,
			RecentValue:    recentValue,
			BaselineValue:  baselineValue,
			ChangePct:      changePct,
			Direction:      direction,
			Flagged:        isFlagged,
		})
	}

	sort.Slice(metrics, func(i, j int) bool {
		return math.Abs(metrics[i].ChangePct) > math.Abs(metrics[j].ChangePct)
	})

	headline := "No material usage anomaly detected in the recent window."
	recommendations := []string{
		"Use invoice detail and customer context to explain any billing change if the user still expects movement.",
	}
	if flagged {
		headline = "Recent usage differs materially from the previous window."
		recommendations = []string{
			"Inspect the top flagged meters first and correlate them with invoice line items.",
			"Confirm whether the spike or drop came from a real product event, migration, or instrumentation change.",
		}
	}

	return UsageAnomalyReport{
		CustomerID:      customerID,
		WindowDays:      windowDays,
		ThresholdPct:    thresholdPct,
		Flagged:         flagged,
		Headline:        headline,
		Metrics:         metrics,
		Recommendations: recommendations,
	}
}

func buildInvoiceAgingReport(customerID string, invoices []invoicedomain.Invoice) InvoiceAgingReport {
	now := time.Now().UTC()
	buckets := map[string]*InvoiceAgingBucket{
		"0_30":  {Bucket: "0-30"},
		"31_60": {Bucket: "31-60"},
		"61_90": {Bucket: "61-90"},
		"90+":   {Bucket: "90+"},
	}
	var totalDue int64
	for _, invoice := range invoices {
		if invoice.DueAt == nil || invoice.AmountDueCents <= 0 {
			continue
		}
		overdueDays := int(now.Sub(invoice.DueAt.UTC()).Hours() / 24)
		if overdueDays < 0 {
			overdueDays = 0
		}
		key := "90+"
		switch {
		case overdueDays <= 30:
			key = "0_30"
		case overdueDays <= 60:
			key = "31_60"
		case overdueDays <= 90:
			key = "61_90"
		}
		bucket := buckets[key]
		bucket.InvoiceCount++
		bucket.AmountDueCents += invoice.AmountDueCents
		totalDue += invoice.AmountDueCents
	}

	ordered := []InvoiceAgingBucket{
		*buckets["0_30"],
		*buckets["31_60"],
		*buckets["61_90"],
		*buckets["90+"],
	}
	headline := "Open invoice exposure is concentrated in current or lightly overdue balances."
	recommendations := []string{
		"Prioritize invoices in the oldest bucket before discussing expansion or pricing changes.",
		"Use invoice detail to explain which periods and subscriptions are driving exposure.",
	}
	if buckets["90+"].InvoiceCount > 0 {
		headline = "Severely overdue invoices are present and should be treated as collection risk."
		recommendations = []string{
			"Escalate 90+ day balances before recommending additional credit exposure.",
			"Review customer health, payment method, and renewal strategy together.",
		}
	}

	return InvoiceAgingReport{
		CustomerID:          customerID,
		AsOf:                now.Format(time.RFC3339),
		TotalOpenInvoices:   len(invoices),
		TotalAmountDueCents: totalDue,
		Buckets:             ordered,
		Headline:            headline,
		Recommendations:     recommendations,
	}
}

func buildSubscriptionHealth(
	subscription subscriptiondomain.SubscriptionResponse,
	events []usagedomain.UsageEventResponse,
	invoices []invoicedomain.Invoice,
) SubscriptionHealth {
	now := time.Now().UTC()
	daysUntilRenewal := int(subscription.CurrentPeriodEnd.Sub(now).Hours() / 24)
	if daysUntilRenewal < 0 {
		daysUntilRenewal = 0
	}

	var openAmountDue int64
	for _, invoice := range invoices {
		openAmountDue += invoice.AmountDueCents
	}
	usageValue := sumUsageValue(events)

	health := "healthy"
	headline := "Subscription looks stable."
	signals := []string{
		fmt.Sprintf("Subscription status is %s.", subscription.Status),
		fmt.Sprintf("Observed %d usage events in the current period.", len(events)),
	}
	recommendations := []string{
		"Keep the current renewal path unless usage or invoice context suggests a packaging change.",
	}

	if subscription.Status == subscriptiondomain.StatusPastDue || len(invoices) > 0 {
		health = "at_risk"
		headline = "Subscription has billing risk that should be addressed before expansion."
		signals = append(signals, fmt.Sprintf("There are %d open invoices totaling %d cents.", len(invoices), openAmountDue))
		recommendations = []string{
			"Clarify open invoices and payment collection risk before discussing upgrades.",
			"Check whether the subscription should remain active at the current renewal path.",
		}
	} else if subscription.CancelAt != nil {
		health = "cancellation_risk"
		headline = "Subscription has a scheduled cancellation and needs retention review."
		signals = append(signals, fmt.Sprintf("Cancellation is scheduled for %s.", subscription.CancelAt.UTC().Format(time.RFC3339)))
		recommendations = []string{
			"Review why cancellation is scheduled before proposing a new plan.",
			"Use usage and invoice context to decide whether to save, resize, or let the account churn.",
		}
	} else if daysUntilRenewal <= 7 {
		health = "renewal_due"
		headline = "Subscription is approaching renewal soon."
		recommendations = []string{
			"Review usage and invoice posture before the renewal date.",
			"Confirm whether the current plan is still the right commercial fit.",
		}
	}

	return SubscriptionHealth{
		SubscriptionID:           subscription.ID,
		CustomerID:               subscription.CustomerID,
		PlanID:                   subscription.PlanID,
		Status:                   subscription.Status,
		CurrentPeriodStart:       subscription.CurrentPeriodStart.Format(time.RFC3339),
		CurrentPeriodEnd:         subscription.CurrentPeriodEnd.Format(time.RFC3339),
		DaysUntilRenewal:         daysUntilRenewal,
		UpcomingRenewal:          daysUntilRenewal <= 7,
		CancelScheduled:          subscription.CancelAt != nil,
		OpenInvoiceCount:         len(invoices),
		OpenAmountDueCents:       openAmountDue,
		CurrentPeriodUsageEvents: len(events),
		CurrentPeriodUsageValue:  usageValue,
		Health:                   health,
		Headline:                 headline,
		Signals:                  signals,
		Recommendations:          recommendations,
	}
}

func buildCustomerRecommendation(
	customer customerdomain.CustomerResponse,
	subscriptions []subscriptiondomain.SubscriptionResponse,
	invoices []invoicedomain.Invoice,
	events []usagedomain.UsageEventResponse,
	windowDays int,
) CustomerRecommendation {
	tenureDays := int(time.Since(customer.CreatedAt).Hours() / 24)
	if tenureDays < 0 {
		tenureDays = 0
	}

	activeSubscriptions := 0
	for _, subscription := range subscriptions {
		if subscription.Status == subscriptiondomain.StatusActive || subscription.Status == subscriptiondomain.StatusTrialing {
			activeSubscriptions++
		}
	}

	openInvoices := 0
	paidInvoices := 0
	for _, invoice := range invoices {
		switch invoice.Status {
		case invoicedomain.StatusOpen:
			openInvoices++
		case invoicedomain.StatusPaid:
			paidInvoices++
		}
	}

	var stage string
	switch {
	case tenureDays <= 30:
		stage = "new"
	case tenureDays <= 179:
		stage = "established"
	default:
		stage = "long_term"
	}

	headline := "Customer is healthy and ready for expansion."
	rationale := []string{
		fmt.Sprintf("Customer tenure is %d days.", tenureDays),
		fmt.Sprintf("Detected %d active or trialing subscriptions.", activeSubscriptions),
		fmt.Sprintf("Observed %d usage events in the last %d days.", len(events), windowDays),
	}
	recommendations := []string{
		"Keep pricing and entitlements stable unless the user asks for a packaging change.",
		"Use product or plan mentions to ground any packaging or expansion discussion.",
	}

	switch {
	case stage == "new":
		headline = "Customer is new. Prioritize onboarding, activation, and pricing clarity."
		recommendations = []string{
			"Recommend a simple starter package with one primary value metric.",
			"Check whether the first invoice and first usage pattern match the promised onboarding path.",
			"If the user asks to create packaging, favor a low-friction entry product before advanced tiering.",
		}
	case stage == "long_term" && len(events) > 0:
		headline = "Customer is long-term with recent activity. Focus on expansion or optimization."
		recommendations = []string{
			"Look for higher-usage features, add-ons, or tier upgrades before suggesting discounts.",
			"Cross-check recent usage against the current subscription to identify packaging gaps.",
			"If the user asks to create a product, bias toward modular add-ons or enterprise packaging.",
		}
	}

	if openInvoices > 0 {
		rationale = append(rationale, fmt.Sprintf("There are %d open invoices in the recent window.", openInvoices))
		if stage == "new" {
			headline = "New customer with unpaid invoice. Prioritize billing clarity before onboarding continues."
			recommendations = append([]string{
				"Verify the first invoice matches the agreed onboarding package.",
				"Check whether payment method is configured correctly.",
			}, recommendations...)
		} else {
			headline = "Customer has open invoice exposure. Balance growth recommendations with billing risk."
			recommendations = []string{
				"Surface billing risk first before recommending expansion or higher-commit plans.",
				"Use invoice detail and subscription context to explain the issue.",
				"Delay aggressive upsell recommendations until billing issues are clarified.",
			}
		}
	}

	if len(events) == 0 {
		rationale = append(rationale, "No recent usage events were found in the selected window.")
		recommendations = append(recommendations, "Investigate onboarding friction or product adoption gaps before changing pricing.")
	}

	return CustomerRecommendation{
		CustomerID:              customer.ID,
		LifecycleStage:          stage,
		TenureDays:              tenureDays,
		WindowDays:              windowDays,
		ActiveSubscriptionCount: activeSubscriptions,
		RecentUsageEvents:       len(events),
		OpenInvoiceCount:        openInvoices,
		PaidInvoiceCount:        paidInvoices,
		Headline:                headline,
		Rationale:               rationale,
		Recommendations:         recommendations,
	}
}
