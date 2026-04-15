package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"gorm.io/gorm"
)

type dbInsightReadModel struct {
	db *gorm.DB
}

func NewInsightReadModel(db *gorm.DB) InsightReadModel {
	return &dbInsightReadModel{db: db}
}

type planInvoiceRow struct {
	SubscriptionID uuid.UUID `gorm:"column:subscription_id"`
	CustomerID     uuid.UUID `gorm:"column:customer_id"`
	PlanID         uuid.UUID `gorm:"column:plan_id"`
	PlanName       string    `gorm:"column:plan_name"`
	Currency       string    `gorm:"column:currency"`
	PeriodEnd      time.Time `gorm:"column:period_end"`
	TotalCents     int64     `gorm:"column:total_cents"`
	UsageCents     int64     `gorm:"column:usage_cents"`
	BaseCents      int64     `gorm:"column:base_cents"`
}

func (m *dbInsightReadModel) ListPlanFitSnapshots(ctx context.Context, limit int) ([]domain.PlanFitSnapshot, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, domain.ErrInvalidOrganization
	}

	if limit <= 0 {
		limit = 20
	}

	windowDays := 90
	since := time.Now().UTC().AddDate(0, 0, -windowDays)
	invoiceLimit := limit * 6

	var rows []planInvoiceRow
	err := m.db.WithContext(ctx).
		Table("invoices i").
		Select(
			"i.subscription_id, i.customer_id, s.plan_id, COALESCE(p.name, '') as plan_name, "+
				"i.currency, i.period_end, i.total_cents, "+
				"SUM(CASE WHEN ii.line_type = ? THEN ii.amount_cents ELSE 0 END) as usage_cents, "+
				"SUM(CASE WHEN ii.line_type != ? THEN ii.amount_cents ELSE 0 END) as base_cents",
			invoicedomain.LineTypeUsage,
			invoicedomain.LineTypeUsage,
		).
		Joins("JOIN invoice_items ii ON ii.invoice_id = i.id").
		Joins("LEFT JOIN subscriptions s ON s.id = i.subscription_id").
		Joins("LEFT JOIN plans p ON p.id = s.plan_id").
		Where("i.org_id = ? AND i.subscription_id IS NOT NULL AND i.status IN ? AND i.period_end >= ?", orgID, []string{invoicedomain.StatusOpen, invoicedomain.StatusPaid}, since).
		Group("i.id, i.subscription_id, i.customer_id, s.plan_id, p.name, i.currency, i.period_end, i.total_cents").
		Order("i.period_end desc").
		Limit(invoiceLimit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	bySubscription := map[uuid.UUID][]planInvoiceRow{}
	for _, row := range rows {
		if row.SubscriptionID == uuid.Nil {
			continue
		}
		list := bySubscription[row.SubscriptionID]
		if len(list) >= 6 {
			continue
		}
		bySubscription[row.SubscriptionID] = append(list, row)
	}

	snapshots := make([]domain.PlanFitSnapshot, 0, len(bySubscription))
	for _, invoices := range bySubscription {
		if len(invoices) == 0 {
			continue
		}
		sort.Slice(invoices, func(i, j int) bool {
			return invoices[i].PeriodEnd.After(invoices[j].PeriodEnd)
		})

		latest := invoices[0]
		var sumTotal int64
		var overagePeriods int
		consecutiveHigh := 0
		consecutiveLow := 0
		usagePctLatest := 0.0

		for i, inv := range invoices {
			total := inv.BaseCents + inv.UsageCents
			usagePct := 0.0
			if total > 0 {
				usagePct = float64(inv.UsageCents) / float64(total)
			}
			if i == 0 {
				usagePctLatest = usagePct
			}

			if inv.UsageCents > 0 && inv.UsageCents > inv.BaseCents {
				overagePeriods++
			}

			if i == consecutiveHigh && usagePct >= 0.85 {
				consecutiveHigh++
			}
			if i == consecutiveLow && usagePct <= 0.3 {
				consecutiveLow++
			}

			sumTotal += inv.TotalCents
		}

		avg := int64(0)
		if len(invoices) > 0 {
			avg = sumTotal / int64(len(invoices))
		}

		snapshots = append(snapshots, domain.PlanFitSnapshot{
			CustomerID:           latest.CustomerID.String(),
			SubscriptionID:       latest.SubscriptionID.String(),
			PlanID:               latest.PlanID.String(),
			PlanName:             latest.PlanName,
			WindowDays:           windowDays,
			UsagePctOfIncluded:   usagePctLatest,
			ConsecutiveHighUsage: consecutiveHigh,
			ConsecutiveLowUsage:  consecutiveLow,
			OveragePeriods:       overagePeriods,
			AvgInvoiceAmount:     avg,
			LastInvoiceAmount:    latest.TotalCents,
			Currency:             latest.Currency,
			LastObservedAt:       latest.PeriodEnd,
		})
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].LastObservedAt.After(snapshots[j].LastObservedAt)
	})

	if len(snapshots) > limit {
		snapshots = snapshots[:limit]
	}

	return snapshots, nil
}

type churnAggRow struct {
	SubscriptionID uuid.UUID `gorm:"column:subscription_id"`
	CustomerID     uuid.UUID `gorm:"column:customer_id"`
	PlanID         uuid.UUID `gorm:"column:plan_id"`
	PlanName       string    `gorm:"column:plan_name"`
	LastObservedAt time.Time `gorm:"column:last_observed_at"`
	RecentUsage    int64     `gorm:"column:recent_usage"`
	PrevUsage      int64     `gorm:"column:prev_usage"`
	RecentTotal    int64     `gorm:"column:recent_total"`
	PrevTotal      int64     `gorm:"column:prev_total"`
}

type overdueRow struct {
	SubscriptionID uuid.UUID `gorm:"column:subscription_id"`
	OverdueDays    int64     `gorm:"column:overdue_days"`
	OverdueCount   int64     `gorm:"column:overdue_count"`
}

type latePaymentRow struct {
	SubscriptionID uuid.UUID `gorm:"column:subscription_id"`
	LateCount      int64     `gorm:"column:late_count"`
}

func (m *dbInsightReadModel) ListChurnRiskSnapshots(ctx context.Context, limit int) ([]domain.ChurnRiskSnapshot, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return nil, domain.ErrInvalidOrganization
	}

	if limit <= 0 {
		limit = 20
	}

	now := time.Now().UTC()
	recentStart := now.AddDate(0, 0, -14)
	previousStart := now.AddDate(0, 0, -28)
	lateSince := now.AddDate(0, 0, -90)

	var rows []churnAggRow
	err := m.db.WithContext(ctx).
		Table("invoices i").
		Select(
			"i.subscription_id, i.customer_id, s.plan_id, COALESCE(p.name, '') as plan_name, "+
				"MAX(i.period_end) as last_observed_at, "+
				"SUM(CASE WHEN i.period_end >= ? THEN CASE WHEN ii.line_type = ? THEN ii.amount_cents ELSE 0 END ELSE 0 END) as recent_usage, "+
				"SUM(CASE WHEN i.period_end >= ? THEN ii.amount_cents ELSE 0 END) as recent_total, "+
				"SUM(CASE WHEN i.period_end >= ? AND i.period_end < ? THEN CASE WHEN ii.line_type = ? THEN ii.amount_cents ELSE 0 END ELSE 0 END) as prev_usage, "+
				"SUM(CASE WHEN i.period_end >= ? AND i.period_end < ? THEN ii.amount_cents ELSE 0 END) as prev_total",
			recentStart,
			invoicedomain.LineTypeUsage,
			recentStart,
			previousStart,
			recentStart,
			invoicedomain.LineTypeUsage,
			previousStart,
			recentStart,
		).
		Joins("JOIN invoice_items ii ON ii.invoice_id = i.id").
		Joins("LEFT JOIN subscriptions s ON s.id = i.subscription_id").
		Joins("LEFT JOIN plans p ON p.id = s.plan_id").
		Where("i.org_id = ? AND i.subscription_id IS NOT NULL AND i.status IN ? AND i.period_end >= ?", orgID, []string{invoicedomain.StatusOpen, invoicedomain.StatusPaid}, previousStart).
		Group("i.subscription_id, i.customer_id, s.plan_id, p.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	var overdueRows []overdueRow
	err = m.db.WithContext(ctx).
		Table("invoices").
		Select("subscription_id, MAX(EXTRACT(EPOCH FROM (? - due_at)) / 86400) as overdue_days, COUNT(*) as overdue_count", now).
		Where("org_id = ? AND subscription_id IS NOT NULL AND status = ? AND due_at IS NOT NULL AND due_at < ? AND amount_due_cents > amount_paid_cents", orgID, invoicedomain.StatusOpen, now).
		Group("subscription_id").
		Scan(&overdueRows).Error
	if err != nil {
		return nil, err
	}

	overdueBySub := map[uuid.UUID]overdueRow{}
	for _, row := range overdueRows {
		overdueBySub[row.SubscriptionID] = row
	}

	var lateRows []latePaymentRow
	err = m.db.WithContext(ctx).
		Table("invoices").
		Select("subscription_id, COUNT(*) as late_count").
		Where("org_id = ? AND subscription_id IS NOT NULL AND paid_at IS NOT NULL AND due_at IS NOT NULL AND paid_at > due_at AND paid_at >= ?", orgID, lateSince).
		Group("subscription_id").
		Scan(&lateRows).Error
	if err != nil {
		return nil, err
	}

	lateBySub := map[uuid.UUID]int64{}
	for _, row := range lateRows {
		lateBySub[row.SubscriptionID] = row.LateCount
	}

	snapshots := make([]domain.ChurnRiskSnapshot, 0, len(rows))
	for _, row := range rows {
		if row.SubscriptionID == uuid.Nil {
			continue
		}

		usageDrop := percentDrop(row.PrevUsage, row.RecentUsage)
		invoiceDrop := percentDrop(row.PrevTotal, row.RecentTotal)

		overdue := overdueBySub[row.SubscriptionID]
		hasOverdue := overdue.OverdueCount > 0

		snapshots = append(snapshots, domain.ChurnRiskSnapshot{
			CustomerID:           row.CustomerID.String(),
			SubscriptionID:       row.SubscriptionID.String(),
			PlanID:               row.PlanID.String(),
			WindowDays:           28,
			UsageDropPct:         usageDrop,
			HasOverdueInvoice:    hasOverdue,
			OverdueDays:          int(overdue.OverdueDays),
			LatePaymentCount90d:  int(lateBySub[row.SubscriptionID]),
			InvoiceAmountDropPct: invoiceDrop,
			LastObservedAt:       row.LastObservedAt,
		})
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].LastObservedAt.After(snapshots[j].LastObservedAt)
	})

	if len(snapshots) > limit {
		snapshots = snapshots[:limit]
	}

	return snapshots, nil
}

func percentDrop(previous, current int64) float64 {
	if previous <= 0 {
		return 0
	}
	delta := float64(previous-current) / float64(previous)
	if delta < 0 {
		return 0
	}
	return delta * 100
}
