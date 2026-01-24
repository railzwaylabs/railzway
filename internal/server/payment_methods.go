package server

import (
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/smallbiznis/railzway/internal/orgcontext"
	"github.com/smallbiznis/railzway/internal/payment/domain"
	"gorm.io/datatypes"
)

// --- Request DTOs ---

type attachPaymentMethodRequest struct {
	Provider string `json:"provider" binding:"required"`
	Token    string `json:"token" binding:"required"` // Multi-use token from provider
}

type upsertPaymentMethodConfigRequest struct {
	ID                 *string        `json:"id,omitempty"`
	MethodType         string         `json:"method_type" binding:"required"`          // 'card', 'virtual_account', 'ewallet'
	MethodName         string         `json:"method_name" binding:"required"`          // e.g. 'card_global', 'va_bca'
	DisplayName        string         `json:"display_name" binding:"required"`         // User friendly name
	Description        *string        `json:"description,omitempty"`
	IconURL            *string        `json:"icon_url,omitempty"`
	Provider           string         `json:"provider" binding:"required"`             // 'stripe', 'xendit'
	ProviderMethodType *string        `json:"provider_method_type,omitempty"`          // e.g. 'BCA', 'GOPAY'
	Priority           *int           `json:"priority,omitempty"`
	AvailabilityRules  datatypes.JSON `json:"availability_rules" binding:"required"` // JSON rules { "countries": ["ID"], "currencies": ["IDR"] }
	IsActive           *bool          `json:"is_active,omitempty"`
}

type togglePaymentMethodConfigRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

// --- Customer API Handlers ---

// ListAvailablePaymentMethods returns available payment methods based on context (country, currency)
// GET /api/payment-methods/available?country=ID&currency=IDR
func (s *Server) ListAvailablePaymentMethods(c *gin.Context) {
	country := strings.ToUpper(c.Query("country"))
	currency := strings.ToUpper(c.Query("currency"))

	if country == "" || currency == "" {
		AbortWithError(c, newValidationError("query", "missing_params", "country and currency are required"))
		return
	}

	// Org from API key context
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	configs, err := s.paymentMethodConfigSvc.GetAvailablePaymentMethods(c.Request.Context(), orgID, country, currency)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": configs})
}

// AttachPaymentMethod attaches a tokenized payment method to a customer
// POST /api/customers/:id/payment-methods
func (s *Server) AttachPaymentMethod(c *gin.Context) {
	customerIDStr := c.Param("id")
	customerID, err := snowflake.ParseString(customerIDStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidCustomer)
		return
	}

	var req attachPaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	pm, err := s.paymentMethodSvc.AttachPaymentMethod(c.Request.Context(), customerID, req.Provider, req.Token)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_method": pm})
}

// ListCustomerPaymentMethods lists all payment methods for a customer
// GET /api/customers/:id/payment-methods
func (s *Server) ListCustomerPaymentMethods(c *gin.Context) {
	customerIDStr := c.Param("id")
	customerID, err := snowflake.ParseString(customerIDStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidCustomer)
		return
	}

	pms, err := s.paymentMethodSvc.ListPaymentMethods(c.Request.Context(), customerID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": pms})
}

// DetachPaymentMethod removes a payment method from a customer
// DELETE /api/customers/:id/payment-methods/:pm_id
func (s *Server) DetachPaymentMethod(c *gin.Context) {
	customerIDStr := c.Param("id")
	customerID, err := snowflake.ParseString(customerIDStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidCustomer)
		return
	}

	pmIDStr := c.Param("pm_id")
	pmID, err := snowflake.ParseString(pmIDStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidPaymentMethod)
		return
	}

	if err := s.paymentMethodSvc.DetachPaymentMethod(c.Request.Context(), customerID, pmID); err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// SetDefaultPaymentMethod sets a payment method as default for a customer
// POST /api/customers/:id/payment-methods/:pm_id/default
func (s *Server) SetDefaultPaymentMethod(c *gin.Context) {
	customerIDStr := c.Param("id")
	customerID, err := snowflake.ParseString(customerIDStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidCustomer)
		return
	}

	pmIDStr := c.Param("pm_id")
	pmID, err := snowflake.ParseString(pmIDStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidPaymentMethod)
		return
	}

	if err := s.paymentMethodSvc.SetDefaultPaymentMethod(c.Request.Context(), customerID, pmID); err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- Admin API Handlers ---

// ListPaymentMethodConfigs (Admin)
// GET /admin/payment-method-configs
func (s *Server) ListPaymentMethodConfigs(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	configs, err := s.paymentMethodConfigSvc.ListPaymentMethodConfigs(c.Request.Context(), orgID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

// UpsertPaymentMethodConfig (Admin)
// POST /admin/payment-method-configs
func (s *Server) UpsertPaymentMethodConfig(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	var req upsertPaymentMethodConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}
	icon := ""
	if req.IconURL != nil {
		icon = *req.IconURL
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}
	providerMethodType := ""
	if req.ProviderMethodType != nil {
		providerMethodType = *req.ProviderMethodType
	}

	config := &domain.PaymentMethodConfig{
		OrgID:              orgID,
		MethodType:         req.MethodType,
		MethodName:         req.MethodName,
		DisplayName:        req.DisplayName,
		Description:        desc,
		IconURL:            icon,
		Provider:           req.Provider,
		ProviderMethodType: providerMethodType,
		Priority:           priority,
		AvailabilityRules:  req.AvailabilityRules,
		IsActive:           active,
	}

	if req.ID != nil {
		id, err := snowflake.ParseString(*req.ID)
		if err == nil {
			config.ID = id
		}
	} else {
		config.ID = s.genID.Generate()
	}

	if err := s.paymentMethodConfigSvc.UpsertPaymentMethodConfig(c.Request.Context(), config); err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config})
}

// DeletePaymentMethodConfig (Admin)
// DELETE /admin/payment-method-configs/:id
func (s *Server) DeletePaymentMethodConfig(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := snowflake.ParseString(idStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidConfig)
		return
	}

	if err := s.paymentMethodConfigSvc.DeletePaymentMethodConfig(c.Request.Context(), orgID, id); err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// TogglePaymentMethodConfig (Admin)
// POST /admin/payment-method-configs/:id/toggle
func (s *Server) TogglePaymentMethodConfig(c *gin.Context) {
	orgID := s.orgIDFromContext(c)
	if orgID == 0 {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := snowflake.ParseString(idStr)
	if err != nil {
		AbortWithError(c, domain.ErrInvalidConfig)
		return
	}

	var req togglePaymentMethodConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	if err := s.paymentMethodConfigSvc.TogglePaymentMethodConfig(c.Request.Context(), orgID, id, *req.IsActive); err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Helper to get org ID from context (works for both API key and Admin session)
func (s *Server) orgIDFromContext(c *gin.Context) snowflake.ID {
	if orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context()); ok && orgID != 0 {
		return orgID
	}
	return 0
}
