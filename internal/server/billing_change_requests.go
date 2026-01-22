package server

import (
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	approvalservice "github.com/smallbiznis/railzway/internal/billingoperations/service"
)

// RequestBillingCycleReRating handles POST /admin/billing/cycles/:id/request-rerating
func (s *Server) RequestBillingCycleReRating(c *gin.Context) {
	cycleIDStr := strings.TrimSpace(c.Param("id"))
	cycleID, err := snowflake.ParseString(cycleIDStr)
	if err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	orgID := snowflake.ID(0)  // TODO: get from context
	userID := snowflake.ID(0) // TODO: get from context
	userName := "system"      // TODO: get from context

	approvalSvc := approvalservice.NewApprovalService(s.db, s.genID, s.auditSvc)
	changeRequest, err := approvalSvc.RequestReRating(c.Request.Context(), approvalservice.RequestReRatingInput{
		OrgID:           orgID,
		BillingCycleID:  cycleID,
		RequestedBy:     userID,
		RequestedByName: userName,
		Reason:          req.Reason,
	})
	if err != nil {
		AbortWithError(c, ErrInternal)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     changeRequest.ID.String(),
		"status": string(changeRequest.Status),
	})
}

// ApproveBillingCycleReRating handles POST /admin/billing/change-requests/:id/approve
func (s *Server) ApproveBillingCycleReRating(c *gin.Context) {
	requestIDStr := strings.TrimSpace(c.Param("id"))
	requestID, err := snowflake.ParseString(requestIDStr)
	if err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	userID := snowflake.ID(0) // TODO: get from context
	userName := "system"      // TODO: get from context

	approvalSvc := approvalservice.NewApprovalService(s.db, s.genID, s.auditSvc)
	if err := approvalSvc.ApproveReRating(c.Request.Context(), approvalservice.ApproveReRatingInput{
		RequestID:      requestID,
		ApprovedBy:     userID,
		ApprovedByName: userName,
	}); err != nil {
		if strings.Contains(err.Error(), "cannot approve their own request") {
			AbortWithError(c, ErrForbidden)
			return
		}
		AbortWithError(c, ErrInternal)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

// RejectBillingCycleReRating handles POST /admin/billing/change-requests/:id/reject
func (s *Server) RejectBillingCycleReRating(c *gin.Context) {
	requestIDStr := strings.TrimSpace(c.Param("id"))
	requestID, err := snowflake.ParseString(requestIDStr)
	if err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	userID := snowflake.ID(0) // TODO: get from context
	userName := "system"      // TODO: get from context

	approvalSvc := approvalservice.NewApprovalService(s.db, s.genID, s.auditSvc)
	if err := approvalSvc.RejectReRating(c.Request.Context(), approvalservice.RejectReRatingInput{
		RequestID:      requestID,
		RejectedBy:     userID,
		RejectedByName: userName,
		Reason:         req.Reason,
	}); err != nil {
		AbortWithError(c, ErrInternal)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

// ListBillingChangeRequests handles GET /admin/billing/change-requests
func (s *Server) ListBillingChangeRequests(c *gin.Context) {
	orgID := snowflake.ID(0) // TODO: get from context

	var requests []struct {
		ID              string  `json:"id"`
		BillingCycleID  string  `json:"billing_cycle_id"`
		Type            string  `json:"type"`
		Status          string  `json:"status"`
		RequestedBy     string  `json:"requested_by"`
		RequestedByName string  `json:"requested_by_name"`
		ApprovedBy      *string `json:"approved_by,omitempty"`
		ApprovedByName  *string `json:"approved_by_name,omitempty"`
		Reason          string  `json:"reason"`
		RejectionReason *string `json:"rejection_reason,omitempty"`
		CreatedAt       string  `json:"created_at"`
		ApprovedAt      *string `json:"approved_at,omitempty"`
	}

	if err := s.db.WithContext(c.Request.Context()).
		Table("billing_change_requests").
		Where("org_id = ?", orgID).
		Order("created_at DESC").
		Scan(&requests).Error; err != nil {
		AbortWithError(c, ErrInternal)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": requests})
}
