package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/internal/billingoperations/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Service) CalculatePerformance(ctx context.Context, userID string, start, end time.Time) (domain.FinOpsScoreSnapshot, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.FinOpsScoreSnapshot{}, domain.ErrInvalidOrganization
	}

	assignments, err := s.repo.ListBillingAssignmentsForPerformance(ctx, snowflake.ID(orgID), userID, start, end)
	if err != nil {
		return domain.FinOpsScoreSnapshot{}, err
	}

	metrics := domain.PerformanceMetrics{
		TotalAssigned: len(assignments),
	}

	if len(assignments) == 0 {
		return domain.FinOpsScoreSnapshot{
			OrgID:          snowflake.ID(orgID).String(),
			UserID:         userID,
			PeriodType:     domain.PeriodTypeDaily,
			PeriodStart:    start,
			PeriodEnd:      end,
			ScoringVersion: domain.ScoringVersionV1EqualWeight,
			Metrics:        metrics,
			Scores:         domain.PerformanceScores{},
		}, nil
	}

	var totalResponseTime time.Duration
	var responseCount int64

	for _, a := range assignments {
		if a.Status.String == domain.AssignmentStatusReleased {
			metrics.TotalResolved++
		}
		if a.Status.String == domain.AssignmentStatusEscalated || a.BreachedAt.Valid {
			metrics.TotalEscalated++
		}

		var firstAction domain.BillingActionRecord
		err := s.db.WithContext(ctx).Table("billing_operation_actions").
			Where("org_id = ? AND entity_id = ? AND created_at >= ? AND created_at < ?", orgID, a.EntityID, a.AssignedAt.Time, end).
			Order("created_at ASC").
			Limit(1).
			Scan(&firstAction).Error

		if err == nil && firstAction.ID != 0 {
			diff := firstAction.CreatedAt.Sub(a.AssignedAt.Time)
			if diff < 0 {
				diff = 0
			}
			totalResponseTime += diff
			responseCount++
		}

		if a.Status.String == domain.AssignmentStatusReleased {
			var releaseAction domain.BillingActionRecord
			err := s.db.WithContext(ctx).Table("billing_operation_actions").
				Where("org_id = ? AND entity_id = ? AND action_type = ? AND created_at >= ?", orgID, a.EntityID, domain.ActionTypeRelease, a.AssignedAt.Time).
				Order("created_at DESC").
				Limit(1).
				Scan(&releaseAction).Error

			if err == nil && releaseAction.ID != 0 {
				if snap, ok := releaseAction.Metadata["snapshot"].(map[string]any); ok {
					var amt int64
					if val, ok := snap["amount_due"].(float64); ok {
						amt = int64(val)
					} else if val, ok := snap["amount_due"].(int64); ok {
						amt = val
					} else if val, ok := snap["amount_due"].(int); ok {
						amt = int64(val)
					} else if val, ok := snap["amount_due"].(json.Number); ok {
						if v, err := val.Float64(); err == nil {
							amt = int64(v)
						}
					}

					metrics.ExposureHandled += amt
				}
			}
		}
	}

	if responseCount > 0 {
		metrics.AvgResponseMS = int64(totalResponseTime.Milliseconds()) / responseCount
	}
	metrics.CompletionRatio = float64(metrics.TotalResolved) / float64(metrics.TotalAssigned)
	metrics.EscalationRate = float64(metrics.TotalEscalated) / float64(metrics.TotalAssigned)

	scores := domain.PerformanceScores{}

	if metrics.AvgResponseMS > 0 {
		avgHours := float64(metrics.AvgResponseMS) / 3600000.0
		if avgHours <= 1 {
			scores.Responsiveness = 100
		} else if avgHours >= 24 {
			scores.Responsiveness = 0
		} else {
			scores.Responsiveness = int(100 - ((avgHours-1)/23)*100)
		}
	} else {
		if metrics.TotalAssigned > 0 {
			scores.Responsiveness = 0
		}
	}

	if metrics.TotalAssigned > 0 {
		scores.Completion = int(metrics.CompletionRatio * 100)
	}

	if metrics.TotalAssigned > 0 {
		scores.Risk = int((1.0 - metrics.EscalationRate) * 100)
	}

	if metrics.ExposureHandled > 100000 {
		scores.Effectiveness = 100
	} else if metrics.ExposureHandled > 10000 {
		scores.Effectiveness = 75
	} else if metrics.ExposureHandled > 0 {
		scores.Effectiveness = 50
	} else {
		scores.Effectiveness = 0
	}

	scores.Total = (scores.Responsiveness + scores.Completion + scores.Risk + scores.Effectiveness) / 4

	return domain.FinOpsScoreSnapshot{
		OrgID:          snowflake.ID(orgID).String(),
		UserID:         userID,
		PeriodType:     domain.PeriodTypeDaily,
		PeriodStart:    start,
		PeriodEnd:      end,
		ScoringVersion: domain.ScoringVersionV1EqualWeight,
		Metrics:        metrics,
		Scores:         scores,
	}, nil
}

func (s *Service) GetPerformanceHistory(ctx context.Context, userID string, limit int) ([]domain.FinOpsScoreSnapshot, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	if limit <= 0 {
		limit = 30
	}

	type outputRow struct {
		OrgID       snowflake.ID
		UserID      string
		PeriodStart time.Time
		PeriodEnd   time.Time
		Metrics     datatypes.JSON
		Scores      datatypes.JSON
	}

	var rows []outputRow
	if err := s.db.WithContext(ctx).Table("finops_performance_snapshots").
		Where("org_id = ? AND user_id = ?", orgID, userID).
		Order("period_start DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	snapshots := make([]domain.FinOpsScoreSnapshot, len(rows))
	for i, r := range rows {
		var m domain.PerformanceMetrics
		var sc domain.PerformanceScores
		_ = json.Unmarshal(r.Metrics, &m)
		_ = json.Unmarshal(r.Scores, &sc)

		snapshots[i] = domain.FinOpsScoreSnapshot{
			OrgID:       r.OrgID.String(),
			UserID:      r.UserID,
			PeriodStart: r.PeriodStart,
			PeriodEnd:   r.PeriodEnd,
			Metrics:     m,
			Scores:      sc,
		}
	}
	return snapshots, nil
}

func (s *Service) AggregateDailyPerformance(ctx context.Context) error {
	now := s.clock.Now(ctx).UTC()
	start := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	type UserOrg struct {
		OrgID      snowflake.ID
		AssignedTo string
	}
	var userOrgs []UserOrg
	if err := s.db.WithContext(ctx).Table("billing_operation_assignments").
		Select("DISTINCT org_id, assigned_to").
		Where("assigned_at >= ? AND assigned_at < ?", start, end).
		Scan(&userOrgs).Error; err != nil {
		return err
	}

	for _, uo := range userOrgs {
		if uo.AssignedTo == "" {
			continue
		}

		orgCtx := orgcontext.WithOrgID(ctx, uo.OrgID.Int64())

		snapshot, err := s.CalculatePerformance(orgCtx, uo.AssignedTo, start, end)
		if err != nil {
			s.log.Error("failed to calc performance", zap.Error(err), zap.String("user", uo.AssignedTo))
			continue
		}

		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`
					DELETE FROM finops_performance_snapshots 
					WHERE org_id = ? AND user_id = ? AND period_type = ? AND period_start = ?
				`, uo.OrgID, uo.AssignedTo, snapshot.PeriodType, start).Error; err != nil {
				return err
			}

			return tx.Exec(`
					INSERT INTO finops_performance_snapshots 
					(id, org_id, user_id, period_type, period_start, period_end, scoring_version, metrics, scores, total_score, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, s.genID.Generate(), uo.OrgID, uo.AssignedTo, snapshot.PeriodType, start, end, snapshot.ScoringVersion,
				datatypes.JSON(toJson(snapshot.Metrics)),
				datatypes.JSON(toJson(snapshot.Scores)),
				snapshot.Scores.Total,
				now, now).Error
		})

		if err != nil {
			s.log.Error("failed to persist snapshot", zap.Error(err))
		}
	}
	return nil
}

func (s *Service) GetMyPerformance(ctx context.Context, userID string, req domain.GetPerformanceRequest) (*domain.PerformanceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}
	if userID == "" {
		return nil, domain.ErrInvalidAssignee
	}

	start := req.From
	end := req.To
	now := s.clock.Now(ctx).UTC()

	if end.IsZero() {
		end = now
	}
	if start.IsZero() {
		start = end.AddDate(0, 0, -30)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 30
	}

	snapshots, err := s.repo.FindSnapshotsByUserWithLimit(ctx, snowflake.ID(orgID), userID, req.PeriodType, start, end, limit)
	if err != nil {
		return nil, err
	}

	apiSnapshots := make([]domain.APISnapshot, len(snapshots))
	for i, s := range snapshots {
		apiSnapshots[i] = domain.APISnapshot{
			PeriodStart: s.PeriodStart,
			PeriodEnd:   s.PeriodEnd,
			TotalScore:  s.Scores.Total,
			Scores:      s.Scores,
			Metrics: domain.APIMetrics{
				AvgResponseMinutes: float64(s.Metrics.AvgResponseMS) / 60000.0,
				CompletionRatio:    s.Metrics.CompletionRatio,
				EscalationRatio:    s.Metrics.EscalationRate,
				ExposureHandled:    s.Metrics.ExposureHandled,
			},
		}
	}

	return &domain.PerformanceResponse{
		UserID:         userID,
		PeriodType:     req.PeriodType,
		ScoringVersion: domain.ScoringVersionV1EqualWeight,
		Snapshots:      apiSnapshots,
	}, nil
}

func (s *Service) GetTeamPerformance(ctx context.Context, req domain.GetPerformanceRequest) (*domain.TeamPerformanceResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return nil, domain.ErrInvalidOrganization
	}

	start := req.From
	end := req.To
	now := s.clock.Now(ctx).UTC()

	if end.IsZero() {
		end = now
	}
	if start.IsZero() {
		start = end.AddDate(0, 0, -30)
	}

	snapshots, err := s.repo.FindSnapshotsByOrg(ctx, snowflake.ID(orgID), req.PeriodType, start, end)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]domain.FinOpsScoreSnapshot)
	for _, snap := range snapshots {
		grouped[snap.UserID] = append(grouped[snap.UserID], snap)
	}

	teamSummaries := make([]domain.TeamMemberSummary, 0, len(grouped))
	for uid, snaps := range grouped {
		var totalScore int
		var totalAssigned, totalResolved, totalEscalated int
		var totalExposure int64
		var weightedResponseMS float64
		var count int

		for _, s := range snaps {
			totalScore += s.Scores.Total
			count++
			totalAssigned += s.Metrics.TotalAssigned
			totalResolved += s.Metrics.TotalResolved
			totalEscalated += s.Metrics.TotalEscalated
			totalExposure += s.Metrics.ExposureHandled
			weightedResponseMS += float64(s.Metrics.AvgResponseMS) * float64(s.Metrics.TotalResolved)
		}

		avgScore := 0
		if count > 0 {
			avgScore = totalScore / count
		}

		var avgResponseMS int64
		if totalResolved > 0 {
			avgResponseMS = int64(weightedResponseMS / float64(totalResolved))
		}

		var completionRatio, escalationRate float64
		if totalAssigned > 0 {
			completionRatio = float64(totalResolved) / float64(totalAssigned)
			escalationRate = float64(totalEscalated) / float64(totalAssigned)
		}

		teamSummaries = append(teamSummaries, domain.TeamMemberSummary{
			UserID:   uid,
			AvgScore: avgScore,
			MetricsSummary: domain.APIMetrics{
				AvgResponseMinutes: float64(avgResponseMS) / 60000.0,
				CompletionRatio:    completionRatio,
				EscalationRatio:    escalationRate,
				ExposureHandled:    totalExposure,
			},
		})
	}

	return &domain.TeamPerformanceResponse{
		PeriodType: req.PeriodType,
		TeamSize:   len(teamSummaries),
		Snapshots:  teamSummaries,
	}, nil
}
