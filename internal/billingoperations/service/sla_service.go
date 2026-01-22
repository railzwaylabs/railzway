package service

import (
	"context"
	"time"

	"github.com/smallbiznis/railzway/internal/billingoperations/domain"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Service) EvaluateSLAs(ctx context.Context) error {
	const (
		initialResponseSLA = 30 * time.Minute
		idleActionSLA      = 60 * time.Minute
	)
	now := s.clock.Now().UTC()

	var records []domain.BillingAssignmentRecord
	records, err := s.repo.ListActiveAssignments(ctx)
	if err != nil {
		return err
	}

	for _, rec := range records {
		isBreached := false
		breachType := ""

		if rec.Status == domain.AssignmentStatusAssigned {
			if now.Sub(rec.AssignedAt) > initialResponseSLA {
				isBreached = true
				breachType = "initial_response"
			}
		}

		if !isBreached && rec.LastActionAt.Valid {
			if now.Sub(rec.LastActionAt.Time) > idleActionSLA {
				isBreached = true
				breachType = "idle_action"
			}
		}

		if isBreached {
			err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				repoTx := s.repo.WithTx(tx)

				if err := repoTx.EscalateAssignment(ctx, rec.OrgID, rec.EntityType, rec.EntityID, breachType, now); err != nil {
					return err
				}

				actionID := s.genID.Generate()
				bucket := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

				metadata := datatypes.JSONMap{
					"assignment_id": rec.ID.String(),
					"breach_type":   breachType,
					"minutes_idle":  0,
				}
				if rec.LastActionAt.Valid {
					metadata["minutes_idle"] = int(now.Sub(rec.LastActionAt.Time).Minutes())
				} else {
					metadata["minutes_since_assigned"] = int(now.Sub(rec.AssignedAt).Minutes())
				}

				_, err := repoTx.InsertBillingAction(ctx, domain.BillingActionRecord{
					ID:           actionID,
					OrgID:        rec.OrgID,
					EntityType:   rec.EntityType,
					EntityID:     rec.EntityID,
					ActionType:   "sla_breached",
					ActionBucket: bucket,
					Metadata:     metadata,
					ActorType:    "system",
					ActorID:      "sla_monitor",
					CreatedAt:    now,
				})
				return err
			})

			if err != nil {
				s.log.Error("failed to escalate assignment",
					zap.String("assignment_id", rec.ID.String()),
					zap.Error(err))
				continue
			}

			if s.auditSvc != nil {
				targetID := rec.EntityID.String()
				_ = s.auditSvc.AuditLog(ctx, &rec.OrgID, "system", nil,
					"billing_operations.assignment.escalated",
					"billing_operation_assignment",
					&targetID,
					map[string]any{
						"breach_type":   breachType,
						"assignment_id": rec.ID.String(),
					})
			}
		}
	}
	return nil
}
