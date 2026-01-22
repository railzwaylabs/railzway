package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	auditdomain "github.com/smallbiznis/railzway/internal/audit/domain"
	billingopsdomain "github.com/smallbiznis/railzway/internal/billingoperations/domain"
	"gorm.io/gorm"
)

type ApprovalService struct {
	db       *gorm.DB
	genID    *snowflake.Node
	auditSvc auditdomain.Service
}

func NewApprovalService(db *gorm.DB, genID *snowflake.Node, auditSvc auditdomain.Service) *ApprovalService {
	return &ApprovalService{
		db:       db,
		genID:    genID,
		auditSvc: auditSvc,
	}
}

// RequestReRating creates a pending approval request for re-rating a billing cycle.
func (s *ApprovalService) RequestReRating(ctx context.Context, req RequestReRatingInput) (*billingopsdomain.BillingChangeRequest, error) {
	if req.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	now := time.Now().UTC()
	changeRequest := &billingopsdomain.BillingChangeRequest{
		ID:              s.genID.Generate(),
		OrgID:           req.OrgID,
		BillingCycleID:  req.BillingCycleID,
		Type:            billingopsdomain.ChangeRequestTypeReRating,
		Status:          billingopsdomain.ChangeRequestStatusPending,
		RequestedBy:     req.RequestedBy,
		RequestedByName: req.RequestedByName,
		Reason:          req.Reason,
		CreatedAt:       now,
	}

	if err := s.db.WithContext(ctx).Create(changeRequest).Error; err != nil {
		return nil, err
	}

	// Audit log
	s.auditSvc.AuditLog(ctx, &req.OrgID, string(auditdomain.ActorTypeUser), stringPtr(req.RequestedBy.String()), "billing.change_request.created", "billing_cycle", stringPtr(req.BillingCycleID.String()), map[string]interface{}{
		"request_id": changeRequest.ID.String(),
		"type":       string(changeRequest.Type),
		"reason":     req.Reason,
	})

	return changeRequest, nil
}

// ApproveReRating approves a pending request and resets the billing cycle for re-rating.
func (s *ApprovalService) ApproveReRating(ctx context.Context, req ApproveReRatingInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock and fetch the change request
		var changeRequest billingopsdomain.BillingChangeRequest
		if err := tx.Where("id = ?", req.RequestID).First(&changeRequest).Error; err != nil {
			return err
		}

		// Validate status
		if changeRequest.Status != billingopsdomain.ChangeRequestStatusPending {
			return fmt.Errorf("request is not pending (status: %s)", changeRequest.Status)
		}

		// Four-eyes principle: requester cannot approve their own request
		if !changeRequest.CanApprove(req.ApprovedBy) {
			return fmt.Errorf("requester cannot approve their own request")
		}

		// SoD validation: Check if approver has billing_approver or owner/admin role
		var membership struct {
			Role string `gorm:"column:role"`
		}
		if err := tx.Raw(`
			SELECT role
			FROM organization_members
			WHERE org_id = ? AND user_id = ?
		`, changeRequest.OrgID, req.ApprovedBy).Scan(&membership).Error; err != nil {
			return fmt.Errorf("failed to verify approver role: %w", err)
		}

		approverRole := strings.ToLower(strings.TrimSpace(membership.Role))
		if approverRole != "billing_approver" && approverRole != "owner" && approverRole != "admin" {
			return fmt.Errorf("user does not have approval permissions (role: %s)", approverRole)
		}

		// Update change request status
		now := time.Now().UTC()
		if err := tx.Model(&changeRequest).Updates(map[string]interface{}{
			"status":           billingopsdomain.ChangeRequestStatusApproved,
			"approved_by":      req.ApprovedBy,
			"approved_by_name": req.ApprovedByName,
			"approved_at":      now,
		}).Error; err != nil {
			return err
		}

		// Reset billing cycle for re-rating
		if err := tx.Exec(`
			UPDATE billing_cycles
			SET rating_completed_at = NULL,
			    closed_at = NULL,
			    status = 2,
			    last_error = NULL
			WHERE id = ?
		`, changeRequest.BillingCycleID).Error; err != nil {
			return err
		}

		// Mark as executed
		if err := tx.Model(&changeRequest).Update("executed_at", now).Error; err != nil {
			return err
		}

		// Audit log
		s.auditSvc.AuditLog(ctx, &changeRequest.OrgID, string(auditdomain.ActorTypeUser), stringPtr(req.ApprovedBy.String()), "billing.change_request.approved", "billing_cycle", stringPtr(changeRequest.BillingCycleID.String()), map[string]interface{}{
			"request_id":   changeRequest.ID.String(),
			"requested_by": changeRequest.RequestedBy.String(),
			"reason":       changeRequest.Reason,
		})

		return nil
	})
}

// RejectReRating rejects a pending request.
func (s *ApprovalService) RejectReRating(ctx context.Context, req RejectReRatingInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var changeRequest billingopsdomain.BillingChangeRequest
		if err := tx.Where("id = ?", req.RequestID).First(&changeRequest).Error; err != nil {
			return err
		}

		if changeRequest.Status != billingopsdomain.ChangeRequestStatusPending {
			return fmt.Errorf("request is not pending")
		}

		now := time.Now().UTC()
		if err := tx.Model(&changeRequest).Updates(map[string]interface{}{
			"status":           billingopsdomain.ChangeRequestStatusRejected,
			"approved_by":      req.RejectedBy,
			"approved_by_name": req.RejectedByName,
			"approved_at":      now,
			"rejection_reason": req.Reason,
		}).Error; err != nil {
			return err
		}

		// Audit log
		s.auditSvc.AuditLog(ctx, &changeRequest.OrgID, string(auditdomain.ActorTypeUser), stringPtr(req.RejectedBy.String()), "billing.change_request.rejected", "billing_cycle", stringPtr(changeRequest.BillingCycleID.String()), map[string]interface{}{
			"request_id": changeRequest.ID.String(),
			"reason":     req.Reason,
		})

		return nil
	})
}

type RequestReRatingInput struct {
	OrgID           snowflake.ID
	BillingCycleID  snowflake.ID
	RequestedBy     snowflake.ID
	RequestedByName string
	Reason          string
}

type ApproveReRatingInput struct {
	RequestID      snowflake.ID
	ApprovedBy     snowflake.ID
	ApprovedByName string
}

type RejectReRatingInput struct {
	RequestID      snowflake.ID
	RejectedBy     snowflake.ID
	RejectedByName string
	Reason         string
}

func stringPtr(s string) *string {
	return &s
}
