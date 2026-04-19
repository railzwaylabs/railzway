package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
)

type createOrganizationRequest struct {
	Name         string `json:"name"`
	CountryCode  string `json:"country_code"`
	TimezoneName string `json:"timezone_name"`
}

func (h *Handler) CreateOrganization(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	var payload createOrganizationRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.organizations.Create(ctx, userID, organizationdomain.CreateOrganizationRequest{
		Name:         strings.TrimSpace(payload.Name),
		CountryCode:  strings.TrimSpace(payload.CountryCode),
		TimezoneName: strings.TrimSpace(payload.TimezoneName),
	})
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListOrganizations(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	resp, err := h.organizations.ListOrganizationsByUser(c.Request.Context(), userID)
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetOrganization(c *gin.Context) {
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	orgID := strings.TrimSpace(c.Param("org_id"))
	resp, err := h.organizations.GetByID(c.Request.Context(), orgID)
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type updateOrganizationRequest struct {
	Name         *string `json:"name"`
	CountryCode  *string `json:"country_code"`
	TimezoneName *string `json:"timezone_name"`
}

func (h *Handler) UpdateOrganization(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	orgID := strings.TrimSpace(c.Param("org_id"))
	var payload updateOrganizationRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.organizations.Update(ctx, userID, orgID, organizationdomain.UpdateOrganizationRequest{
		Name:         payload.Name,
		CountryCode:  payload.CountryCode,
		TimezoneName: payload.TimezoneName,
	})
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListOrganizationMembers(c *gin.Context) {
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	orgID := strings.TrimSpace(c.Param("org_id"))
	resp, err := h.organizations.ListMembers(c.Request.Context(), orgID)
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type inviteMembersRequest struct {
	Invites []organizationdomain.InviteRequest `json:"invites"`
}

func (h *Handler) InviteOrganizationMembers(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	orgID := strings.TrimSpace(c.Param("org_id"))
	var payload inviteMembersRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	if err := h.organizations.InviteMembers(ctx, userID, orgID, payload.Invites); err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) AcceptOrganizationInvite(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	inviteID := strings.TrimSpace(c.Param("invite_id"))
	if err := h.organizations.AcceptInvite(ctx, userID, inviteID); err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type billingPreferencesRequest struct {
	Currency             string `json:"currency"`
	Timezone             string `json:"timezone"`
	InvoicePrefix        string `json:"invoice_prefix"`
	InvoiceNumberFormat  string `json:"invoice_number_format"`
	InvoiceSequenceScope string `json:"invoice_sequence_scope"`
}

func (h *Handler) UpsertOrganizationBillingPreferences(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	orgID := strings.TrimSpace(c.Param("org_id"))
	var payload billingPreferencesRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	if err := h.organizations.SetBillingPreferences(ctx, userID, orgID, organizationdomain.BillingPreferencesRequest{
		Currency:             strings.TrimSpace(payload.Currency),
		Timezone:             strings.TrimSpace(payload.Timezone),
		InvoicePrefix:        strings.TrimSpace(payload.InvoicePrefix),
		InvoiceNumberFormat:  strings.TrimSpace(payload.InvoiceNumberFormat),
		InvoiceSequenceScope: strings.TrimSpace(payload.InvoiceSequenceScope),
	}); err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type createInvoiceFormatRequest struct {
	Format        string `json:"format"`
	SequenceScope string `json:"sequence_scope"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
}

func (h *Handler) CreateOrganizationInvoiceFormat(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	orgID := strings.TrimSpace(c.Param("org_id"))
	var payload createInvoiceFormatRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	effectiveFrom, err := parseTime(payload.EffectiveFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_effective_from"})
		return
	}
	effectiveTo, err := parseTimePtr(payload.EffectiveTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_effective_to"})
		return
	}

	resp, err := h.organizations.CreateInvoiceNumberFormat(ctx, userID, orgID, organizationdomain.InvoiceNumberFormatRequest{
		Format:        strings.TrimSpace(payload.Format),
		SequenceScope: strings.TrimSpace(payload.SequenceScope),
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   effectiveTo,
	})
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListOrganizationInvoiceFormats(c *gin.Context) {
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	orgID := strings.TrimSpace(c.Param("org_id"))
	resp, err := h.organizations.ListInvoiceNumberFormats(c.Request.Context(), orgID)
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type closeInvoiceFormatRequest struct {
	EffectiveTo string `json:"effective_to"`
}

func (h *Handler) CloseOrganizationInvoiceFormat(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	orgID := strings.TrimSpace(c.Param("org_id"))
	formatID := strings.TrimSpace(c.Param("format_id"))

	var payload closeInvoiceFormatRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	effectiveTo, err := parseTime(payload.EffectiveTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_effective_to"})
		return
	}

	resp, err := h.organizations.CloseInvoiceNumberFormat(ctx, userID, orgID, formatID, effectiveTo)
	if err != nil {
		writeOrganizationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type linkOrganizationRequest struct {
	ChildOrgID string `json:"child_org_id"`
	Mode       string `json:"mode"`
	Role       string `json:"role"`
}

func (h *Handler) LinkChildOrganization(c *gin.Context) {
	if _, ok := userIDFromContext(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, ok := requireOrgParam(c, "org_id"); !ok {
		return
	}
	ctx := h.withAuditContext(c, c.Request.Context())

	parentOrgID := strings.TrimSpace(c.Param("org_id"))
	var payload linkOrganizationRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	parentID, err := uuid.Parse(parentOrgID)
	if err != nil {
		writeOrganizationError(c, organizationdomain.ErrInvalidOrganization)
		return
	}

	childID, err := uuid.Parse(strings.TrimSpace(payload.ChildOrgID))
	if err != nil {
		writeOrganizationError(c, organizationdomain.ErrInvalidOrganization)
		return
	}

	if err := h.organizations.LinkChildOrganization(ctx, parentID, childID, payload.Mode, payload.Role); err != nil {
		writeOrganizationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
