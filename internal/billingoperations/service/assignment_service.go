package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	auditcontext "github.com/smallbiznis/railzway/internal/auditcontext"
	"github.com/smallbiznis/railzway/internal/billingoperations/domain"
	"github.com/smallbiznis/railzway/internal/orgcontext"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Service) RecordAction(ctx context.Context, req domain.RecordActionRequest) (domain.RecordActionResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.RecordActionResponse{}, domain.ErrInvalidOrganization
	}

	entityType := strings.TrimSpace(req.EntityType)
	if entityType != domain.EntityTypeInvoice && entityType != domain.EntityTypeCustomer {
		return domain.RecordActionResponse{}, domain.ErrInvalidEntityType
	}

	actionType := strings.TrimSpace(req.ActionType)
	if actionType != domain.ActionTypeFollowUp &&
		actionType != domain.ActionTypeRetryPayment &&
		actionType != domain.ActionTypeMarkReviewed {
		return domain.RecordActionResponse{}, domain.ErrInvalidActionType
	}

	entityID, err := parseSnowflakeID(req.EntityID)
	if err != nil {
		return domain.RecordActionResponse{}, domain.ErrInvalidEntityID
	}

	idempotencyKey := normalizeIdempotencyKey(req.IdempotencyKey)
	if req.IdempotencyKey != "" && idempotencyKey == "" {
		return domain.RecordActionResponse{}, domain.ErrInvalidIdempotencyKey
	}

	now := s.clock.Now().UTC()
	bucket := now.Truncate(24 * time.Hour)
	actionID := s.genID.Generate()

	beforeSnapshot, err := s.repo.LoadEntitySnapshot(ctx, snowflake.ID(orgID), entityType, entityID)
	if err != nil {
		return domain.RecordActionResponse{}, err
	}
	afterSnapshot := beforeSnapshot

	metadata := datatypes.JSONMap{
		"entity_type":   entityType,
		"entity_id":     entityID.String(),
		"action_type":   actionType,
		"action_bucket": bucket.Format("2006-01-02"),
		"before":        beforeSnapshot,
		"after":         afterSnapshot,
	}
	for key, value := range req.Metadata {
		if strings.TrimSpace(key) == "" {
			continue
		}
		metadata[key] = value
	}

	actorType, actorID := auditcontext.ActorFromContext(ctx)

	inserted, err := s.repo.InsertBillingAction(ctx, domain.BillingActionRecord{
		ID:             actionID,
		OrgID:          snowflake.ID(orgID),
		EntityType:     entityType,
		EntityID:       entityID,
		ActionType:     actionType,
		ActionBucket:   bucket,
		IdempotencyKey: idempotencyKey,
		Metadata:       metadata,
		ActorType:      actorType,
		ActorID:        actorID,
		CreatedAt:      now,
	})
	if err != nil {
		return domain.RecordActionResponse{}, err
	}

	actionStatus := domain.ActionStatusRecorded
	resolvedActionID := actionID.String()
	if !inserted {
		actionStatus = domain.ActionStatusDuplicate
		resolvedActionID = ""
		if idempotencyKey != "" {
			existing, err := s.repo.FindActionByIdempotencyKey(ctx, snowflake.ID(orgID), idempotencyKey)
			if err == nil && existing != nil {
				resolvedActionID = existing.ID.String()
			}
		} else {
			existing, err := s.repo.FindActionByBucket(ctx, snowflake.ID(orgID), entityType, entityID, actionType, bucket)
			if err == nil && existing != nil {
				resolvedActionID = existing.ID.String()
			}
		}
	}

	// Update assignment status if needed
	if inserted && actionType != domain.ActionTypeClaim && actionType != domain.ActionTypeRelease {
		if err := s.repo.UpdateAssignmentStatus(
			ctx, snowflake.ID(orgID), entityType, entityID,
			domain.AssignmentStatusAssigned, domain.AssignmentStatusInProgress,
			now,
		); err != nil {
			s.log.Warn("failed to update assignment status on action", zap.Error(err))
		}
	}

	if s.auditSvc != nil {
		targetID := resolvedActionID
		if targetID == "" {
			targetID = actionID.String()
		}
		oid := snowflake.ID(orgID)
		if err := s.auditSvc.AuditLog(ctx, &oid, "", nil, buildAuditAction(actionType), "billing_operation_action", &targetID, map[string]any{
			"entity_type":   entityType,
			"entity_id":     entityID.String(),
			"action_type":   actionType,
			"action_bucket": bucket.Format("2006-01-02"),
			"status":        actionStatus,
			"before":        beforeSnapshot,
			"after":         afterSnapshot,
		}); err != nil {
			return domain.RecordActionResponse{}, err
		}
	}

	return domain.RecordActionResponse{
		ActionID:   resolvedActionID,
		Status:     actionStatus,
		RecordedAt: now,
	}, nil
}

func (s *Service) ClaimAssignment(ctx context.Context, req domain.ClaimAssignmentRequest) (domain.AssignmentResponse, error) {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.AssignmentResponse{}, domain.ErrInvalidOrganization
	}

	entityType := strings.TrimSpace(req.EntityType)
	if entityType != domain.EntityTypeInvoice && entityType != domain.EntityTypeCustomer {
		return domain.AssignmentResponse{}, domain.ErrInvalidEntityType
	}

	entityID, err := parseSnowflakeID(req.EntityID)
	if err != nil {
		return domain.AssignmentResponse{}, domain.ErrInvalidEntityID
	}

	assignedTo := strings.TrimSpace(req.AssignedTo)
	if assignedTo == "" {
		_, actorID := auditcontext.ActorFromContext(ctx)
		assignedTo = strings.TrimSpace(actorID)
	}
	if assignedTo == "" {
		return domain.AssignmentResponse{}, domain.ErrInvalidAssignee
	}

	ttlMinutes := req.AssignmentTTLMinutes
	if ttlMinutes == 0 {
		ttlMinutes = 120
	}
	if ttlMinutes < 0 {
		return domain.AssignmentResponse{}, domain.ErrInvalidAssignmentTTL
	}

	now := s.clock.Now().UTC()
	expiresAt := now.Add(time.Duration(ttlMinutes) * time.Minute)

	var result *domain.AssignmentResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)

		existing, err := repoTx.LoadAssignmentForUpdate(ctx, snowflake.ID(orgID), entityType, entityID)
		if err != nil {
			return err
		}

		if existing != nil && existing.Status != domain.AssignmentStatusReleased {
			if existing.AssignedTo != assignedTo {
				return domain.ErrAssignmentConflict
			}
			record := *existing
			record.AssignmentExpiresAt = expiresAt
			record.UpdatedAt = now

			if err := repoTx.UpsertAssignment(ctx, record); err != nil {
				return err
			}

			result = &domain.AssignmentResponse{
				Assignment: domain.Assignment{
					EntityType:          entityType,
					EntityID:            entityID.String(),
					Status:              existing.Status,
					AssignedTo:          assignedTo,
					AssignedAt:          existing.AssignedAt,
					AssignmentExpiresAt: expiresAt,
					LastActionAt:        timePtr(existing.LastActionAt),
				},
				Status: domain.AssignmentStatusAssigned,
			}
			return nil
		}

		snapshot, err := s.repo.LoadEntitySnapshot(ctx, snowflake.ID(orgID), req.EntityType, entityID)
		if err != nil {
			s.log.Warn("failed to load entity snapshot", zap.Error(err))
			snapshot = make(map[string]interface{})
		}

		snapshotJSON, err := json.Marshal(snapshot)
		if err != nil {
			s.log.Warn("failed to marshal snapshot", zap.Error(err))
			snapshotJSON = []byte("{}")
		}

		record := domain.BillingAssignmentRecord{
			ID:                  s.genID.Generate(),
			OrgID:               snowflake.ID(orgID),
			EntityType:          entityType,
			EntityID:            entityID,
			AssignedTo:          assignedTo,
			AssignedAt:          now,
			AssignmentExpiresAt: expiresAt,
			Status:              domain.AssignmentStatusAssigned,
			SnapshotMetadata:    datatypes.JSON(snapshotJSON),
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		if err := repoTx.UpsertAssignment(ctx, record); err != nil {
			return err
		}

		result = &domain.AssignmentResponse{
			Assignment: domain.Assignment{
				EntityType:          entityType,
				EntityID:            entityID.String(),
				Status:              domain.AssignmentStatusAssigned,
				AssignedTo:          assignedTo,
				AssignedAt:          now,
				AssignmentExpiresAt: expiresAt,
			},
			Status: domain.AssignmentStatusAssigned,
		}

		actionID := s.genID.Generate()
		bucket := now.Truncate(24 * time.Hour)

		if _, err := repoTx.InsertBillingAction(ctx, domain.BillingActionRecord{
			ID:           actionID,
			OrgID:        snowflake.ID(orgID),
			EntityType:   entityType,
			EntityID:     entityID,
			ActionType:   domain.ActionTypeClaim,
			ActionBucket: bucket,
			Metadata: datatypes.JSONMap{
				"assignment_id": record.ID.String(),
				"expires_at":    expiresAt,
			},
			ActorType: "user",
			ActorID:   assignedTo,
			CreatedAt: now,
		}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return domain.AssignmentResponse{}, err
	}
	if result == nil {
		return domain.AssignmentResponse{}, fmt.Errorf("internal error: result not set in transaction")
	}

	if s.auditSvc != nil && result.Status == domain.AssignmentStatusAssigned {
		targetID := result.Assignment.EntityID
		oid := snowflake.ID(orgID)
		_ = s.auditSvc.AuditLog(ctx, &oid, "", nil,
			"billing_operations.assignment.claimed",
			"billing_operation_assignment",
			&targetID,
			map[string]any{
				"entity_type": entityType,
				"entity_id":   entityID.String(),
				"assigned_to": assignedTo,
				"expires_at":  expiresAt.Format(time.RFC3339),
			},
		)
	}

	return *result, nil
}

func (s *Service) ReleaseAssignment(ctx context.Context, req domain.ReleaseAssignmentRequest) error {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.ErrInvalidOrganization
	}

	entityType := strings.TrimSpace(req.EntityType)
	if entityType != domain.EntityTypeInvoice && entityType != domain.EntityTypeCustomer {
		return domain.ErrInvalidEntityType
	}

	entityID, err := parseSnowflakeID(req.EntityID)
	if err != nil {
		return domain.ErrInvalidEntityID
	}

	releasedBy := strings.TrimSpace(req.ReleasedBy)
	if releasedBy == "" {
		_, actorID := auditcontext.ActorFromContext(ctx)
		releasedBy = strings.TrimSpace(actorID)
	}
	if releasedBy == "" {
		return domain.ErrInvalidAssignee
	}

	now := s.clock.Now().UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)

		existing, err := repoTx.LoadAssignmentForUpdate(ctx, snowflake.ID(orgID), entityType, entityID)
		if err != nil {
			return err
		}
		if existing == nil || existing.Status == domain.AssignmentStatusReleased {
			return nil
		}

		existing.Status = domain.AssignmentStatusReleased
		existing.ReleasedAt = sql.NullTime{Time: now, Valid: true}
		existing.ReleasedBy = sql.NullString{String: releasedBy, Valid: true}
		existing.ReleaseReason = sql.NullString{String: req.Reason, Valid: true}
		existing.ResolvedAt = sql.NullTime{Time: now, Valid: true}
		existing.ResolvedBy = sql.NullString{String: releasedBy, Valid: true}
		existing.UpdatedAt = now

		if err := repoTx.UpsertAssignment(ctx, *existing); err != nil {
			return err
		}

		actionID := s.genID.Generate()
		bucket := now.Truncate(24 * time.Hour)

		snapshot, err := s.repo.LoadEntitySnapshot(ctx, snowflake.ID(orgID), entityType, entityID)
		if err != nil {
			s.log.Warn("failed to load entity snapshot", zap.Error(err))
			snapshot = make(map[string]interface{})
		}

		if _, err := repoTx.InsertBillingAction(ctx, domain.BillingActionRecord{
			ID:           actionID,
			OrgID:        snowflake.ID(orgID),
			EntityType:   entityType,
			EntityID:     entityID,
			ActionType:   domain.ActionTypeRelease,
			ActionBucket: bucket,
			Metadata: datatypes.JSONMap{
				"assignment_id": existing.ID.String(),
				"released_by":   releasedBy,
				"reason":        req.Reason,
				"snapshot":      snapshot,
			},
			ActorType: "user",
			ActorID:   releasedBy,
			CreatedAt: now,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if s.auditSvc != nil {
		targetID := entityID.String()
		oid := snowflake.ID(orgID)
		_ = s.auditSvc.AuditLog(ctx, &oid, "", nil,
			"billing_operations.assignment.released",
			"billing_operation_assignment",
			&targetID,
			map[string]any{
				"entity_type": entityType,
				"entity_id":   entityID.String(),
				"released_by": releasedBy,
				"reason":      req.Reason,
			},
		)
	}

	return nil
}

func (s *Service) ResolveAssignment(ctx context.Context, req domain.ResolveAssignmentRequest) error {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.ErrInvalidOrganization
	}

	entityType := strings.TrimSpace(req.EntityType)
	if entityType != domain.EntityTypeInvoice && entityType != domain.EntityTypeCustomer {
		return domain.ErrInvalidEntityType
	}

	entityID, err := parseSnowflakeID(req.EntityID)
	if err != nil {
		return domain.ErrInvalidEntityID
	}

	resolvedBy := strings.TrimSpace(req.ResolvedBy)
	if resolvedBy == "" {
		_, actorID := auditcontext.ActorFromContext(ctx)
		resolvedBy = strings.TrimSpace(actorID)
	}
	if resolvedBy == "" {
		return domain.ErrInvalidAssignee
	}

	now := s.clock.Now().UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)

		existing, err := repoTx.LoadAssignmentForUpdate(ctx, snowflake.ID(orgID), entityType, entityID)
		if err != nil {
			return err
		}
		if existing == nil {
			return nil
		}
		if existing.Status == domain.AssignmentStatusResolved {
			return nil
		}

		existing.Status = domain.AssignmentStatusResolved
		existing.ResolvedAt = sql.NullTime{Time: now, Valid: true}
		existing.ResolvedBy = sql.NullString{String: resolvedBy, Valid: true}
		existing.ReleaseReason = sql.NullString{String: req.Resolution, Valid: true}
		existing.UpdatedAt = now

		if err := repoTx.UpsertAssignment(ctx, *existing); err != nil {
			return err
		}

		actionID := s.genID.Generate()
		bucket := now.Truncate(24 * time.Hour)

		if _, err := repoTx.InsertBillingAction(ctx, domain.BillingActionRecord{
			ID:           actionID,
			OrgID:        snowflake.ID(orgID),
			EntityType:   entityType,
			EntityID:     entityID,
			ActionType:   domain.ActionTypeResolve,
			ActionBucket: bucket,
			Metadata: datatypes.JSONMap{
				"assignment_id": existing.ID.String(),
				"resolution":    req.Resolution,
				"resolved_by":   resolvedBy,
			},
			ActorType: "user",
			ActorID:   resolvedBy,
			CreatedAt: now,
		}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	if s.auditSvc != nil {
		targetID := entityID.String()
		oid := snowflake.ID(orgID)
		_ = s.auditSvc.AuditLog(ctx, &oid, "", nil,
			"billing_operations.assignment.resolved",
			"billing_operation_assignment",
			&targetID,
			map[string]any{
				"entity_type": entityType,
				"entity_id":   entityID.String(),
				"resolution":  req.Resolution,
				"resolved_by": resolvedBy,
			},
		)
	}

	return nil
}

func (s *Service) RecordFollowUp(ctx context.Context, req domain.RecordFollowUpRequest) error {
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok || orgID == 0 {
		return domain.ErrInvalidOrganization
	}

	assignmentID, err := parseSnowflakeID(req.AssignmentID)
	if err != nil {
		return domain.ErrInvalidEntityID // Or a specific error for assignment ID
	}

	now := s.clock.Now().UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// This is a simplified version of what follow_up.go might have done
		// Usually it records an action and potentially updates the assignment's last action time
		
		var assignment domain.BillingAssignmentRecord
		if err := tx.Where("org_id = ? AND id = ?", orgID, assignmentID).First(&assignment).Error; err != nil {
			return err
		}

		actionID := s.genID.Generate()
		bucket := now.Truncate(24 * time.Hour)
		_, actorID := auditcontext.ActorFromContext(ctx)

		if _, err := s.repo.WithTx(tx).InsertBillingAction(ctx, domain.BillingActionRecord{
			ID:           actionID,
			OrgID:        snowflake.ID(orgID),
			EntityType:   assignment.EntityType,
			EntityID:     assignment.EntityID,
			ActionType:   domain.ActionTypeFollowUp,
			ActionBucket: bucket,
			Metadata: datatypes.JSONMap{
				"assignment_id":  assignmentID.String(),
				"email_provider": req.EmailProvider,
			},
			ActorType: "user",
			ActorID:   actorID,
			CreatedAt: now,
		}); err != nil {
			return err
		}

		// Update last action at
		return tx.Model(&domain.BillingAssignmentRecord{}).
			Where("org_id = ? AND id = ?", orgID, assignmentID).
			Updates(map[string]interface{}{
				"last_action_at": sql.NullTime{Time: now, Valid: true},
				"updated_at":     now,
			}).Error
	})

	return err
}
