package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/invoice/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/publiclink"
)

func (h *Handler) ListInvoices(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	periodStartFrom, err := parseTimePtr(c.Query("period_start_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_period_start_from"})
		return
	}
	periodStartTo, err := parseTimePtr(c.Query("period_start_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_period_start_to"})
		return
	}
	issuedFrom, err := parseTimePtr(c.Query("issued_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_issued_from"})
		return
	}
	issuedTo, err := parseTimePtr(c.Query("issued_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_issued_to"})
		return
	}
	createdFrom, err := parseTimePtr(c.Query("created_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_from"})
		return
	}
	createdTo, err := parseTimePtr(c.Query("created_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_created_to"})
		return
	}

	resp, err := h.invoices.ListInvoices(ctx, domain.ListInvoicesRequest{
		PageToken:       c.Query("page_token"),
		PageSize:        parseInt32(c.Query("page_size")),
		CustomerID:      strings.TrimSpace(c.Query("customer_id")),
		SubscriptionID:  strings.TrimSpace(c.Query("subscription_id")),
		Status:          strings.TrimSpace(c.Query("status")),
		Number:          strings.TrimSpace(c.Query("number")),
		PeriodStartFrom: periodStartFrom,
		PeriodStartTo:   periodStartTo,
		IssuedFrom:      issuedFrom,
		IssuedTo:        issuedTo,
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetInvoice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	resp, err := h.invoices.GetInvoice(ctx, domain.GetInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type generateInvoiceRequest struct {
	SubscriptionID string `json:"subscription_id"`
	PeriodStart    string `json:"period_start"`
	PeriodEnd      string `json:"period_end"`
	IssueAt        string `json:"issue_at"`
	DueAt          string `json:"due_at"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *Handler) GenerateInvoice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload generateInvoiceRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	periodStart, err := parseTime(payload.PeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_period_start"})
		return
	}
	periodEnd, err := parseTime(payload.PeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_period_end"})
		return
	}
	issueAt, err := parseTimePtr(payload.IssueAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_issue_at"})
		return
	}
	dueAt, err := parseTimePtr(payload.DueAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_due_at"})
		return
	}

	resp, err := h.invoices.GenerateInvoice(ctx, domain.GenerateInvoiceRequest{
		SubscriptionID: strings.TrimSpace(payload.SubscriptionID),
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		IssueAt:        issueAt,
		DueAt:          dueAt,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ResendInvoice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	resp, err := h.invoices.ResendInvoice(ctx, domain.ResendInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	link, linkErr := h.buildInvoicePublicLink(ctx, orgID, invoiceID)
	if linkErr != nil {
		c.JSON(http.StatusOK, invoiceResendResponse{Status: resp.Status})
		return
	}
	c.JSON(http.StatusOK, invoiceResendResponse{Status: resp.Status, PublicLink: link})
}

func (h *Handler) GetInvoicePublicLink(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	link, err := h.buildInvoicePublicLink(ctx, orgID, invoiceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "public_link_unavailable"})
		return
	}
	c.JSON(http.StatusOK, link)
}

type invoiceResendResponse struct {
	Status     string                  `json:"status"`
	PublicLink *invoicePublicLinkReply `json:"public_link,omitempty"`
}

type invoicePublicLinkReply struct {
	Token     string    `json:"token"`
	URL       string    `json:"url,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *Handler) buildInvoicePublicLink(ctx context.Context, orgID uuid.UUID, invoiceID string) (*invoicePublicLinkReply, error) {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return nil, fmt.Errorf("missing invoice id")
	}
	inv, err := h.invoices.GetInvoice(ctx, domain.GetInvoiceRequest{ID: invoiceID})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ttlHours := h.cfg.PublicLink.TTLHours
	if ttlHours <= 0 {
		ttlHours = 168
	}
	ttl := time.Duration(ttlHours) * time.Hour
	token, err := publiclink.BuildInvoiceToken(inv.Invoice.ID, orgID, h.cfg.PublicLink.Secret, ttl)
	if err != nil {
		return nil, err
	}
	expiresAt := now.Add(ttl)
	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicLink.BaseURL), "/")
	url := ""
	if base != "" {
		url = fmt.Sprintf("%s/checkout/%s", base, token)
	}
	return &invoicePublicLinkReply{Token: token, URL: url, ExpiresAt: expiresAt}, nil
}

func (h *Handler) OpenInvoice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	resp, err := h.invoices.OpenInvoice(ctx, domain.OpenInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) PayInvoice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	var payload invoiceActionRequest
	if !bindOptionalJSONOrAbort(c, &payload) {
		return
	}
	meta := payload.toMeta()
	if meta != nil {
		ctx = auditlog.WithMetadata(ctx, meta)
	}
	if strings.TrimSpace(payload.Reason) != "" {
		ctx = auditlog.WithReason(ctx, strings.TrimSpace(payload.Reason))
	}

	resp, err := h.invoices.PayInvoice(ctx, domain.PayInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) VoidInvoice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	var payload invoiceActionRequest
	if !bindOptionalJSONOrAbort(c, &payload) {
		return
	}
	meta := payload.toMeta()
	if meta != nil {
		ctx = auditlog.WithMetadata(ctx, meta)
	}
	if strings.TrimSpace(payload.Reason) != "" {
		ctx = auditlog.WithReason(ctx, strings.TrimSpace(payload.Reason))
	}

	resp, err := h.invoices.VoidInvoice(ctx, domain.VoidInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type invoiceActionRequest struct {
	Reason        string `json:"reason"`
	AttachmentURL string `json:"attachment_url"`
	Note          string `json:"note"`
}

func (p invoiceActionRequest) toMeta() map[string]interface{} {
	meta := map[string]interface{}{}
	if strings.TrimSpace(p.AttachmentURL) != "" {
		meta["attachment_url"] = strings.TrimSpace(p.AttachmentURL)
	}
	if strings.TrimSpace(p.Note) != "" {
		meta["note"] = strings.TrimSpace(p.Note)
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func (h *Handler) MarkInvoicePaid(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	invoiceID := strings.TrimSpace(c.Param("invoice_id"))
	var payload invoiceActionRequest
	if !bindOptionalJSONOrAbort(c, &payload) {
		return
	}
	meta := payload.toMeta()
	if meta == nil {
		meta = map[string]interface{}{}
	}
	meta["manual"] = true
	ctx = auditlog.WithMetadata(ctx, meta)
	if strings.TrimSpace(payload.Reason) != "" {
		ctx = auditlog.WithReason(ctx, strings.TrimSpace(payload.Reason))
	}

	resp, err := h.invoices.PayInvoice(ctx, domain.PayInvoiceRequest{ID: invoiceID})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
