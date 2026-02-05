package server

import (
	"net/http"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/payment/domain"
)

type createCheckoutSessionRequest struct {
	Provider            string                       `json:"provider"`
	Customer            string                       `json:"customer" binding:"required"`
	Currency            string                       `json:"currency" binding:"required,len=3"`
	LineItems           []domain.LineItemInput       `json:"line_items" binding:"required,min=1,dive"`
	SuccessURL          string                       `json:"success_url" binding:"required"`
	CancelURL           string                       `json:"cancel_url" binding:"required"`
	ClientReferenceID   string                       `json:"client_reference_id,omitempty"`
	Metadata            map[string]string            `json:"metadata"`
	AllowPromotionCodes bool                         `json:"allow_promotion_codes"`
}

// CreateCheckoutSession
// POST /api/checkout/sessions
func (s *Server) CreateCheckoutSession(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	var req createCheckoutSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	customerID, err := snowflake.ParseString(req.Customer)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidCustomer)
		return
	}

	input := domain.CheckoutSessionInput{
		OrgID:               orgID,
		Provider:            req.Provider,
		CustomerID:          customerID,
		Currency:            req.Currency,
		LineItems:           req.LineItems,
		SuccessURL:          req.SuccessURL,
		CancelURL:           req.CancelURL,
		ClientReferenceID:   req.ClientReferenceID,
		Metadata:            req.Metadata,
		AllowPromotionCodes: req.AllowPromotionCodes,
	}

	session, err := s.checkoutSvc.CreateSession(c.Request.Context(), input)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// GetCheckoutSession
// GET /api/checkout/sessions/:id
func (s *Server) GetCheckoutSession(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := snowflake.ParseString(sessionIDStr)
	if err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	session, err := s.checkoutSvc.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// GetCheckoutSessionLineItems
// GET /api/checkout/sessions/:id/line_items
func (s *Server) GetCheckoutSessionLineItems(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := snowflake.ParseString(sessionIDStr)
	if err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	lineItems, err := s.checkoutSvc.GetLineItems(c.Request.Context(), sessionID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object":   "list",
		"data":     lineItems,
		"has_more": false,
	})
}

// ExpireCheckoutSession
// POST /api/checkout/sessions/:id/expire
func (s *Server) ExpireCheckoutSession(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := snowflake.ParseString(sessionIDStr)
	if err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	session, err := s.checkoutSvc.ExpireSession(c.Request.Context(), sessionID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// VerifyCheckoutSession
// GET /api/checkout/sessions/:session_id/verify
func (s *Server) VerifyCheckoutSession(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		AbortWithError(c, invalidRequestError())
		return
	}

	session, err := s.checkoutSvc.VerifyAndComplete(c.Request.Context(), sessionID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}
