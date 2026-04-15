package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"gorm.io/gorm"
)

type dbOverviewMetricsProvider struct {
	db *gorm.DB
}

type currencyTotal struct {
	Currency string `gorm:"column:currency"`
	Total    int64  `gorm:"column:total"`
}

func NewOverviewMetricsProvider(db *gorm.DB) overviewMetricsProvider {
	return &dbOverviewMetricsProvider{db: db}
}

func (p *dbOverviewMetricsProvider) BuildSummaryCards(ctx context.Context) ([]domain.SummaryCard, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, domain.ErrInvalidOrganization
	}

	now := time.Now().UTC()
	currentStart := now.AddDate(0, 0, -30)
	previousStart := now.AddDate(0, 0, -60)

	currentTotals, err := p.sumInvoiceTotals(ctx, orgID, currentStart, now)
	if err != nil {
		return nil, err
	}
	previousTotals, err := p.sumInvoiceTotals(ctx, orgID, previousStart, currentStart)
	if err != nil {
		return nil, err
	}

	currentCurrency, currentValue := pickTopCurrency(currentTotals)
	previousValue := pickCurrencyTotal(previousTotals, currentCurrency)
	if currentCurrency == "" {
		currentCurrency = p.lookupOrgCurrency(ctx, orgID)
	}

	forecastValue := formatMoney(currentValue, currentCurrency)
	forecastDelta := formatSignedPercent(deltaPercent(currentValue, previousValue))

	overdueTotals, err := p.sumOverdue(ctx, orgID, now)
	if err != nil {
		return nil, err
	}
	overdueCurrency, overdueValue := pickTopCurrency(overdueTotals)
	if overdueCurrency == "" {
		overdueCurrency = currentCurrency
	}

	upgradeCount, err := p.countUsageCustomers(ctx, orgID, currentStart, now)
	if err != nil {
		return nil, err
	}

	churnCount, err := p.countChurnWatchlist(ctx, orgID, now.AddDate(0, 0, -30))
	if err != nil {
		return nil, err
	}

	return []domain.SummaryCard{
		{
			ID:    "forecast",
			Label: "30d Forecast",
			Value: forecastValue,
			Sub:   "Projected revenue",
			Tone:  "info",
			Delta: forecastDelta,
		},
		{
			ID:    "risk",
			Label: "Revenue at Risk",
			Value: formatMoney(overdueValue, overdueCurrency),
			Sub:   "Past-due + churn exposure",
			Tone:  "warning",
		},
		{
			ID:    "upgrade",
			Label: "Upgrade Potential",
			Value: formatCount(upgradeCount),
			Sub:   "High-fit customers",
			Tone:  "neutral",
		},
		{
			ID:    "churn",
			Label: "Churn Watchlist",
			Value: formatCount(churnCount),
			Sub:   "Needs attention",
			Tone:  "danger",
		},
	}, nil
}

func (p *dbOverviewMetricsProvider) BuildSignals(ctx context.Context) (domain.SignalPanel, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return domain.SignalPanel{}, domain.ErrInvalidOrganization
	}

	now := time.Now().UTC()
	overdueTotals, err := p.sumOverdue(ctx, orgID, now)
	if err != nil {
		return domain.SignalPanel{}, err
	}
	overdueCurrency, overdueValue := pickTopCurrency(overdueTotals)

	overdueCount, err := p.countOverdueInvoices(ctx, orgID, now)
	if err != nil {
		return domain.SignalPanel{}, err
	}

	recentStart := now.AddDate(0, 0, -14)
	previousStart := now.AddDate(0, 0, -28)
	recentTotals, err := p.sumInvoiceTotals(ctx, orgID, recentStart, now)
	if err != nil {
		return domain.SignalPanel{}, err
	}
	previousTotals, err := p.sumInvoiceTotals(ctx, orgID, previousStart, recentStart)
	if err != nil {
		return domain.SignalPanel{}, err
	}
	recentCurrency, recentValue := pickTopCurrency(recentTotals)
	previousValue := pickCurrencyTotal(previousTotals, recentCurrency)
	delta := deltaPercent(recentValue, previousValue)

	uncollectibleCount, err := p.countUncollectible(ctx, orgID, now.AddDate(0, 0, -30))
	if err != nil {
		return domain.SignalPanel{}, err
	}

	items := []domain.SignalItem{
		{
			ID:       "overdue_invoices",
			Title:    fmt.Sprintf("%s overdue invoices", formatCount(overdueCount)),
			Detail:   fmt.Sprintf("Past-due total %s", formatMoney(overdueValue, overdueCurrency)),
			Severity: severityForCount(overdueCount, "risk", "info"),
		},
		{
			ID:       "revenue_delta",
			Title:    fmt.Sprintf("Revenue %s", formatSignedPercent(delta)),
			Detail:   "Last 14 days vs previous 14 days",
			Severity: severityForDelta(delta),
		},
		{
			ID:       "uncollectible",
			Title:    fmt.Sprintf("%s uncollectible invoices", formatCount(uncollectibleCount)),
			Detail:   "Last 30 days",
			Severity: severityForCount(uncollectibleCount, "warning", "info"),
		},
	}

	return domain.SignalPanel{
		Title:       "Churn Signals",
		Description: "Behavioral and billing indicators",
		Items:       items,
	}, nil
}

func (p *dbOverviewMetricsProvider) sumInvoiceTotals(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]currencyTotal, error) {
	var rows []currencyTotal
	statuses := []string{invoicedomain.StatusOpen, invoicedomain.StatusPaid}
	dateExpr := "COALESCE(issued_at, created_at)"
	if err := p.db.WithContext(ctx).
		Table("invoices").
		Select("currency, SUM(total_cents) as total").
		Where("org_id = ? AND status IN ? AND "+dateExpr+" >= ? AND "+dateExpr+" < ?", orgID, statuses, start, end).
		Group("currency").
		Order("total desc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (p *dbOverviewMetricsProvider) sumOverdue(ctx context.Context, orgID uuid.UUID, asOf time.Time) ([]currencyTotal, error) {
	var rows []currencyTotal
	if err := p.db.WithContext(ctx).
		Table("invoices").
		Select("currency, SUM(CASE WHEN amount_due_cents > amount_paid_cents THEN amount_due_cents - amount_paid_cents ELSE 0 END) as total").
		Where("org_id = ? AND status = ? AND due_at IS NOT NULL AND due_at < ?", orgID, invoicedomain.StatusOpen, asOf).
		Group("currency").
		Order("total desc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (p *dbOverviewMetricsProvider) countOverdueInvoices(ctx context.Context, orgID uuid.UUID, asOf time.Time) (int64, error) {
	var count int64
	if err := p.db.WithContext(ctx).
		Table("invoices").
		Where("org_id = ? AND status = ? AND due_at IS NOT NULL AND due_at < ? AND amount_due_cents > amount_paid_cents", orgID, invoicedomain.StatusOpen, asOf).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (p *dbOverviewMetricsProvider) countUsageCustomers(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int64, error) {
	var count int64
	if err := p.db.WithContext(ctx).
		Table("invoice_items").
		Select("COUNT(DISTINCT customer_id)").
		Where("org_id = ? AND line_type = ? AND created_at >= ? AND created_at < ?", orgID, invoicedomain.LineTypeUsage, start, end).
		Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (p *dbOverviewMetricsProvider) countChurnWatchlist(ctx context.Context, orgID uuid.UUID, overdueBefore time.Time) (int64, error) {
	var count int64
	if err := p.db.WithContext(ctx).
		Table("invoices").
		Select("COUNT(DISTINCT customer_id)").
		Where("org_id = ? AND status = ? AND due_at IS NOT NULL AND due_at < ? AND amount_due_cents > amount_paid_cents", orgID, invoicedomain.StatusOpen, overdueBefore).
		Scan(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (p *dbOverviewMetricsProvider) countUncollectible(ctx context.Context, orgID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	if err := p.db.WithContext(ctx).
		Table("invoices").
		Where("org_id = ? AND status = ? AND created_at >= ?", orgID, invoicedomain.StatusUncollectible, since).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (p *dbOverviewMetricsProvider) lookupOrgCurrency(ctx context.Context, orgID uuid.UUID) string {
	var currency string
	err := p.db.WithContext(ctx).
		Table("organization_billing_preferences").
		Select("currency").
		Where("org_id = ?", orgID).
		Limit(1).
		Scan(&currency).Error
	if err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(currency))
}

func pickTopCurrency(rows []currencyTotal) (string, int64) {
	if len(rows) == 0 {
		return "", 0
	}
	return strings.ToUpper(rows[0].Currency), rows[0].Total
}

func pickCurrencyTotal(rows []currencyTotal, currency string) int64 {
	if currency == "" {
		return 0
	}
	for _, row := range rows {
		if strings.EqualFold(row.Currency, currency) {
			return row.Total
		}
	}
	return 0
}

func deltaPercent(current, previous int64) float64 {
	if previous == 0 {
		return 0
	}
	return (float64(current-previous) / float64(previous)) * 100
}

func formatMoney(cents int64, currency string) string {
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

func formatSignedPercent(value float64) string {
	if value == 0 {
		return "0%"
	}
	rounded := math.Round(value)
	if rounded > 0 {
		return fmt.Sprintf("+%.0f%%", rounded)
	}
	return fmt.Sprintf("%.0f%%", rounded)
}

func formatCount(value int64) string {
	return fmt.Sprintf("%d", value)
}

func severityForCount(value int64, high, low string) string {
	if value > 0 {
		return high
	}
	return low
}

func severityForDelta(delta float64) string {
	if delta < -5 {
		return "warning"
	}
	if delta > 5 {
		return "info"
	}
	return "info"
}
