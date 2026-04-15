package service

import (
	"context"
	"sort"

	"github.com/railzwaylabs/railzway/internal/aiassistant/domain"
	"github.com/railzwaylabs/railzway/internal/clock"
)

type InsightReadModel interface {
	ListPlanFitSnapshots(ctx context.Context, limit int) ([]domain.PlanFitSnapshot, error)
	ListChurnRiskSnapshots(ctx context.Context, limit int) ([]domain.ChurnRiskSnapshot, error)
}

type InsightService struct {
	readModel InsightReadModel
	clock     clock.Clock
}

func NewInsightService(readModel InsightReadModel, clock clock.Clock) *InsightService {
	return &InsightService{readModel: readModel, clock: clock}
}

func (s *InsightService) ListInsights(ctx context.Context) ([]domain.Insight, error) {
	var out []domain.Insight

	planSnapshots, err := s.readModel.ListPlanFitSnapshots(ctx, 50)
	if err != nil {
		return nil, err
	}
	for _, snap := range planSnapshots {
		if insight := EvaluatePlanRecommendation(snap); insight != nil {
			out = append(out, *insight)
		}
	}

	churnSnapshots, err := s.readModel.ListChurnRiskSnapshots(ctx, 50)
	if err != nil {
		return nil, err
	}
	for _, snap := range churnSnapshots {
		if insight := EvaluateChurnRisk(snap); insight != nil {
			out = append(out, *insight)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ObservedAt.After(out[j].ObservedAt)
	})

	return out, nil
}

func (s *InsightService) ListPlanFitSnapshots(ctx context.Context, limit int) ([]domain.PlanFitSnapshot, error) {
	return s.readModel.ListPlanFitSnapshots(ctx, limit)
}

func (s *InsightService) ListChurnRiskSnapshots(ctx context.Context, limit int) ([]domain.ChurnRiskSnapshot, error) {
	return s.readModel.ListChurnRiskSnapshots(ctx, limit)
}
