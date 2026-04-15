package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"gorm.io/gorm"
)

type planRankingProvider interface {
	Recommend(ctx context.Context, insight *domain.Insight, snap *domain.PlanFitSnapshot) (*domain.PlanRecommendation, error)
}

type dbPlanRankingProvider struct {
	db *gorm.DB
}

type rankedPlanRow struct {
	PlanID            uuid.UUID `gorm:"column:plan_id"`
	PlanName          string    `gorm:"column:plan_name"`
	RankedAmountCents int64     `gorm:"column:ranked_amount_cents"`
	AmountCount       int64     `gorm:"column:amount_count"`
}

func NewPlanRankingProvider(db *gorm.DB) planRankingProvider {
	return &dbPlanRankingProvider{db: db}
}

func (p *dbPlanRankingProvider) Recommend(ctx context.Context, insight *domain.Insight, snap *domain.PlanFitSnapshot) (*domain.PlanRecommendation, error) {
	if insight == nil {
		return nil, nil
	}

	currentPlan := "Current plan"
	currency := ""
	currentPlanID := uuid.Nil
	usagePct := 0.0

	if snap != nil {
		if strings.TrimSpace(snap.PlanName) != "" {
			currentPlan = snap.PlanName
		}
		if parsed, err := uuid.Parse(strings.TrimSpace(snap.PlanID)); err == nil {
			currentPlanID = parsed
		}
		currency = strings.ToUpper(strings.TrimSpace(snap.Currency))
		usagePct = snap.UsagePctOfIncluded
	}

	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == uuid.Nil {
		return fallbackPlanRecommendation(insight, currentPlan, usagePct), nil
	}

	if currency == "" {
		currency = p.lookupOrgCurrency(ctx, orgID)
	}

	rankedPlans, err := p.listRankedPlans(ctx, orgID, currency)
	if err != nil {
		return nil, err
	}
	if len(rankedPlans) == 0 {
		return fallbackPlanRecommendation(insight, currentPlan, usagePct), nil
	}

	currentIndex := indexCurrentPlan(rankedPlans, currentPlanID, currentPlan)
	if currentIndex == -1 {
		return fallbackPlanRecommendation(insight, currentPlan, usagePct), nil
	}

	recommendedIndex := currentIndex
	label := normalizePlanRecommendationLabel(insight.Title)
	switch label {
	case "Upgrade":
		if currentIndex < len(rankedPlans)-1 {
			recommendedIndex = currentIndex + 1
		}
	case "Downsize":
		if currentIndex > 0 {
			recommendedIndex = currentIndex - 1
		}
	}

	recommended := rankedPlans[recommendedIndex]
	current := rankedPlans[currentIndex]
	if recommended.PlanID == current.PlanID {
		return fallbackPlanRecommendation(insight, currentPlan, usagePct), nil
	}

	return &domain.PlanRecommendation{
		CurrentPlan:     current.PlanName,
		RecommendedPlan: recommended.PlanName,
		SavingsEstimate: planSavingsFromRanking(label, current.RankedAmountCents, recommended.RankedAmountCents, currency, usagePct),
		BillingImpact:   planBillingImpact(current.RankedAmountCents, recommended.RankedAmountCents, currency),
		ReasonSummary:   insight.Summary,
	}, nil
}

func (p *dbPlanRankingProvider) listRankedPlans(ctx context.Context, orgID uuid.UUID, currency string) ([]rankedPlanRow, error) {
	now := time.Now().UTC()
	stmt := p.db.WithContext(ctx).
		Table("plans AS p").
		Select(
			"p.id AS plan_id, p.name AS plan_name, "+
				"SUM(COALESCE(pa.minimum_amount_cents, pa.unit_amount_cents, 0)) AS ranked_amount_cents, "+
				"COUNT(pa.id) AS amount_count",
		).
		Joins("JOIN plan_prices pp ON pp.plan_id = p.id AND pp.org_id = p.org_id AND pp.active = ? AND pp.price_type = ?", true, plandomain.PriceTypeFlat).
		Joins("JOIN plan_amounts pa ON pa.plan_price_id = pp.id AND pa.org_id = p.org_id AND pa.effective_from <= ? AND (pa.effective_to IS NULL OR pa.effective_to > ?)", now, now).
		Where("p.org_id = ? AND p.active = ?", orgID, true).
		Group("p.id, p.name").
		Order("ranked_amount_cents asc, p.name asc")

	if currency != "" {
		stmt = stmt.Where("UPPER(pa.currency) = ?", currency)
	}

	var rows []rankedPlanRow
	if err := stmt.Scan(&rows).Error; err != nil {
		return nil, err
	}

	filtered := make([]rankedPlanRow, 0, len(rows))
	for _, row := range rows {
		if row.AmountCount == 0 {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered, nil
}

func (p *dbPlanRankingProvider) lookupOrgCurrency(ctx context.Context, orgID uuid.UUID) string {
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

func indexCurrentPlan(rows []rankedPlanRow, planID uuid.UUID, planName string) int {
	for i, row := range rows {
		if planID != uuid.Nil && row.PlanID == planID {
			return i
		}
	}
	for i, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.PlanName), strings.TrimSpace(planName)) {
			return i
		}
	}
	return -1
}

func fallbackPlanRecommendation(insight *domain.Insight, currentPlan string, usagePct float64) *domain.PlanRecommendation {
	if insight == nil {
		return nil
	}

	recommendedPlan := "Review plan tiers"
	switch normalizePlanRecommendationLabel(insight.Title) {
	case "Upgrade":
		recommendedPlan = "Higher priced plan"
	case "Downsize":
		recommendedPlan = "Lower priced plan"
	}

	return &domain.PlanRecommendation{
		CurrentPlan:     currentPlan,
		RecommendedPlan: recommendedPlan,
		SavingsEstimate: planSavingsEstimate(usagePct),
		BillingImpact:   "Review pricing catalog",
		ReasonSummary:   insight.Summary,
	}
}

func planSavingsFromRanking(label string, currentCents, recommendedCents int64, currency string, usagePct float64) string {
	diff := recommendedCents - currentCents
	switch label {
	case "Downsize":
		if diff < 0 {
			return fmt.Sprintf("Save %s", formatMoneyLabel(-diff, currency))
		}
	case "Upgrade":
		if diff > 0 {
			return fmt.Sprintf("Add %s base price", formatMoneyLabel(diff, currency))
		}
	}
	return planSavingsEstimate(usagePct)
}

func planBillingImpact(currentCents, recommendedCents int64, currency string) string {
	return fmt.Sprintf("%s -> %s", formatMoneyLabel(currentCents, currency), formatMoneyLabel(recommendedCents, currency))
}
