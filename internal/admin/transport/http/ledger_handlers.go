package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	ledgerdomain "github.com/railzwaylabs/railzway/internal/ledger/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type createLedgerAccountRequest struct {
	Code string `json:"code"`
	Type string `json:"type"`
	Name string `json:"name"`
}

func (h *Handler) CreateLedgerAccount(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	if h.flags != nil && !h.flags.IsEnabled(c.Request.Context(), orgID.String(), "ledger_custom_accounts") {
		c.JSON(http.StatusForbidden, gin.H{"error": "feature_disabled"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createLedgerAccountRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.ledger.CreateAccount(ctx, ledgerdomain.CreateAccountRequest{
		Code: strings.TrimSpace(payload.Code),
		Type: strings.TrimSpace(payload.Type),
		Name: strings.TrimSpace(payload.Name),
	})
	if err != nil {
		writeLedgerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListLedgerAccounts(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	resp, err := h.ledger.ListAccounts(ctx)
	if err != nil {
		writeLedgerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListLedgerTransactions(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	occurredFrom, err := parseTimePtr(c.Query("occurred_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_occurred_from"})
		return
	}
	occurredTo, err := parseTimePtr(c.Query("occurred_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_occurred_to"})
		return
	}
	postedFrom, err := parseTimePtr(c.Query("posted_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_posted_from"})
		return
	}
	postedTo, err := parseTimePtr(c.Query("posted_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_posted_to"})
		return
	}

	resp, err := h.ledger.ListTransactions(ctx, ledgerdomain.ListTransactionsRequest{
		PageToken:      c.Query("page_token"),
		PageSize:       parseInt32(c.Query("page_size")),
		SourceType:     strings.TrimSpace(c.Query("source_type")),
		SourceID:       strings.TrimSpace(c.Query("source_id")),
		CustomerID:     strings.TrimSpace(c.Query("customer_id")),
		SubscriptionID: strings.TrimSpace(c.Query("subscription_id")),
		InvoiceID:      strings.TrimSpace(c.Query("invoice_id")),
		PlanPriceID:    strings.TrimSpace(c.Query("plan_price_id")),
		MeterID:        strings.TrimSpace(c.Query("meter_id")),
		OccurredFrom:   occurredFrom,
		OccurredTo:     occurredTo,
		PostedFrom:     postedFrom,
		PostedTo:       postedTo,
	})
	if err != nil {
		writeLedgerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type ledgerEntryInput struct {
	AccountCode string          `json:"account_code"`
	EntryType   string          `json:"entry_type"`
	AmountCents int64           `json:"amount_cents"`
	Currency    string          `json:"currency"`
	Description *string         `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
}

type createLedgerTransactionRequest struct {
	Currency       string             `json:"currency"`
	SourceType     string             `json:"source_type"`
	SourceID       string             `json:"source_id"`
	ReferenceType  *string            `json:"reference_type"`
	ReferenceID    *string            `json:"reference_id"`
	CustomerID     *string            `json:"customer_id"`
	SubscriptionID *string            `json:"subscription_id"`
	InvoiceID      *string            `json:"invoice_id"`
	OccurredAt     string             `json:"occurred_at"`
	IdempotencyKey string             `json:"idempotency_key"`
	Entries        []ledgerEntryInput `json:"entries"`
}

func (h *Handler) CreateLedgerTransaction(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createLedgerTransactionRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	occurredAt, err := parseTimePtr(payload.OccurredAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_occurred_at"})
		return
	}

	entries := make([]ledgerdomain.LedgerEntryInput, 0, len(payload.Entries))
	for _, entry := range payload.Entries {
		entries = append(entries, ledgerdomain.LedgerEntryInput{
			AccountCode: strings.TrimSpace(entry.AccountCode),
			EntryType:   ledgerdomain.LedgerEntryType(strings.ToLower(strings.TrimSpace(entry.EntryType))),
			AmountCents: entry.AmountCents,
			Currency:    strings.TrimSpace(entry.Currency),
			Description: entry.Description,
			Metadata:    entry.Metadata,
		})
	}

	resp, err := h.ledger.CreateTransaction(ctx, ledgerdomain.CreateTransactionRequest{
		Currency:       strings.TrimSpace(payload.Currency),
		SourceType:     strings.TrimSpace(payload.SourceType),
		SourceID:       strings.TrimSpace(payload.SourceID),
		ReferenceType:  payload.ReferenceType,
		ReferenceID:    payload.ReferenceID,
		CustomerID:     payload.CustomerID,
		SubscriptionID: payload.SubscriptionID,
		InvoiceID:      payload.InvoiceID,
		OccurredAt:     occurredAt,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
		Entries:        entries,
	})
	if err != nil {
		writeLedgerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetLedgerTransaction(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	txID := strings.TrimSpace(c.Param("transaction_id"))
	resp, err := h.ledger.GetTransaction(ctx, ledgerdomain.GetTransactionRequest{ID: txID})
	if err != nil {
		writeLedgerError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
