package xendit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
)

// Factory creates Xendit adapters
type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Provider() string {
	return "xendit"
}

func (f *Factory) NewAdapter(cfg paymentdomain.AdapterConfig) (paymentdomain.PaymentAdapter, error) {
	webhookSecret, ok := readString(cfg.Config, "webhook_secret")
	if !ok || strings.TrimSpace(webhookSecret) == "" {
		return nil, paymentdomain.ErrInvalidConfig
	}

	apiKey, ok := readString(cfg.Config, "api_key")
	if !ok || strings.TrimSpace(apiKey) == "" {
		return nil, paymentdomain.ErrInvalidConfig
	}

	return &Adapter{
		orgID:         cfg.OrgID,
		webhookSecret: webhookSecret,
		apiKey:        apiKey,
	}, nil
}

// Adapter implements PaymentAdapter for Xendit
type Adapter struct {
	orgID         snowflake.ID
	webhookSecret string
	apiKey        string
}

// Verify verifies Xendit webhook signature
func (a *Adapter) Verify(ctx context.Context, payload []byte, headers http.Header) error {
	// Xendit uses X-Callback-Token header for webhook verification
	callbackToken := strings.TrimSpace(headers.Get("X-Callback-Token"))
	if callbackToken == "" {
		return paymentdomain.ErrInvalidSignature
	}

	if callbackToken != a.webhookSecret {
		return paymentdomain.ErrInvalidSignature
	}

	return nil
}

// Parse parses Xendit webhook payload
func (a *Adapter) Parse(ctx context.Context, payload []byte) (*paymentdomain.PaymentEvent, error) {
	var event xenditEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}

	if strings.TrimSpace(event.ID) == "" {
		return nil, paymentdomain.ErrInvalidEvent
	}

	// Parse based on event type
	switch strings.ToLower(strings.TrimSpace(event.Event)) {
	case "payment.succeeded", "invoice.paid":
		return a.parsePaymentSucceeded(event, payload)
	case "payment.failed", "invoice.expired":
		return a.parsePaymentFailed(event, payload)
	default:
		return nil, paymentdomain.ErrEventIgnored
	}
}

// AttachPaymentMethod creates a multi-use token in Xendit
func (a *Adapter) AttachPaymentMethod(ctx context.Context, customerProviderID, token string) (*paymentdomain.PaymentMethodDetails, error) {
	// TODO: Implement Xendit multi-use token creation
	// POST /credit_card_tokens with is_multiple_use: true
	// For now, return placeholder
	return &paymentdomain.PaymentMethodDetails{
		ID:    token,
		Type:  "card",
		Last4: "****", // Will be populated from Xendit API
		Brand: "unknown",
	}, nil
}

// DetachPaymentMethod removes a payment method
func (a *Adapter) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	// Xendit doesn't require explicit detachment
	// Just stop using the token
	return nil
}

// GetPaymentMethod retrieves payment method details
func (a *Adapter) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*paymentdomain.PaymentMethodDetails, error) {
	// TODO: Implement Xendit token retrieval
	// GET /credit_card_tokens/{token_id}
	return nil, errors.New("not implemented")
}

// CreateCheckoutSession creates a new checkout session (invoice)
func (a *Adapter) CreateCheckoutSession(ctx context.Context, input paymentdomain.CheckoutSessionInput) (*paymentdomain.ProviderCheckoutSession, error) {
	if a.apiKey == "" {
		return nil, errors.New("xendit api key not configured")
	}

	// Call Xendit API: POST /v2/invoices
	url := "https://api.xendit.co/v2/invoices"

	// Create request body
	reqBody := map[string]interface{}{
		"external_id":          fmt.Sprintf("checkout-%s-%d", input.CustomerID, time.Now().UnixNano()),
		"amount":               input.Amount,
		"currency":             input.Currency,
		"success_redirect_url": input.SuccessURL,
		"failure_redirect_url": input.CancelURL,
		"description":          "Payment", // Generic description
	}

	// Use metadata or specific input if available for email later
	if input.Metadata != nil {
		if email, ok := input.Metadata["email"]; ok {
			reqBody["payer_email"] = email
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	// Basic Auth with API Key as username
	req.SetBasicAuth(a.apiKey, "")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xendit api error: %d", resp.StatusCode)
	}

	var invoice struct {
		ID         string `json:"id"`
		InvoiceURL string `json:"invoice_url"`
		Status     string `json:"status"`
		ExpiryDate string `json:"expiry_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invoice); err != nil {
		return nil, err
	}

	status := paymentdomain.CheckoutSessionStatusOpen
	if invoice.Status == "PAID" || invoice.Status == "SETTLED" {
		status = paymentdomain.CheckoutSessionStatusComplete
	} else if invoice.Status == "EXPIRED" {
		status = paymentdomain.CheckoutSessionStatusExpired
	}

	expiresAt, _ := time.Parse(time.RFC3339, invoice.ExpiryDate)

	return &paymentdomain.ProviderCheckoutSession{
		ID:        invoice.ID,
		Provider:  "xendit",
		URL:       invoice.InvoiceURL,
		Status:    status,
		ExpiresAt: expiresAt,
	}, nil
}

// RetrieveCheckoutSession retrieves a checkout session (invoice) from Xendit
func (a *Adapter) RetrieveCheckoutSession(ctx context.Context, providerSessionID string) (*paymentdomain.ProviderCheckoutSession, error) {
	if a.apiKey == "" {
		return nil, errors.New("xendit api key not configured")
	}

	// Call Xendit API: GET /v2/invoices/{id}
	url := fmt.Sprintf("https://api.xendit.co/v2/invoices/%s", providerSessionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(a.apiKey, "")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xendit api error: %d", resp.StatusCode)
	}

	var invoice struct {
		ID         string `json:"id"`
		InvoiceURL string `json:"invoice_url"`
		Status     string `json:"status"`
		ExpiryDate string `json:"expiry_date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invoice); err != nil {
		return nil, err
	}

	status := paymentdomain.CheckoutSessionStatusOpen
	if invoice.Status == "PAID" || invoice.Status == "SETTLED" {
		status = paymentdomain.CheckoutSessionStatusComplete
	} else if invoice.Status == "EXPIRED" {
		status = paymentdomain.CheckoutSessionStatusExpired
	}

	expiresAt, _ := time.Parse(time.RFC3339, invoice.ExpiryDate)

	return &paymentdomain.ProviderCheckoutSession{
		ID:        invoice.ID,
		Provider:  "xendit",
		URL:       invoice.InvoiceURL,
		Status:    status,
		ExpiresAt: expiresAt,
	}, nil
}

// ListPaymentMethods lists customer payment methods
func (a *Adapter) ListPaymentMethods(ctx context.Context, customerProviderID string) ([]*paymentdomain.PaymentMethodDetails, error) {
	// Xendit doesn't have a native list API for tokens
	// Return empty list
	return []*paymentdomain.PaymentMethodDetails{}, nil
}

// xenditEvent represents a Xendit webhook event
type xenditEvent struct {
	ID         string          `json:"id"`
	Event      string          `json:"event"`
	Created    string          `json:"created"`
	Data       json.RawMessage `json:"data"`
	ExternalID string          `json:"external_id"`
	Amount     float64         `json:"amount"`
	Currency   string          `json:"currency"`
	Status     string          `json:"status"`
}

func (a *Adapter) parsePaymentSucceeded(event xenditEvent, payload []byte) (*paymentdomain.PaymentEvent, error) {
	// Parse customer_id and invoice_id from external_id or metadata
	// Format: "customer_{customerID}_invoice_{invoiceID}"
	customerID, invoiceID, err := parseExternalID(event.ExternalID)
	if err != nil {
		return nil, paymentdomain.ErrInvalidCustomer
	}

	// Convert amount (Xendit uses float, we use int64 cents)
	amount := int64(event.Amount * 100)

	// Parse timestamp
	occurredAt, _ := time.Parse(time.RFC3339, event.Created)
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return &paymentdomain.PaymentEvent{
		Provider:            "xendit",
		ProviderEventID:     event.ID,
		ProviderPaymentID:   event.ID,
		ProviderPaymentType: "xendit_payment",
		Type:                paymentdomain.EventTypePaymentSucceeded,
		OrgID:               a.orgID,
		CustomerID:          customerID,
		Amount:              amount,
		Currency:            strings.ToUpper(strings.TrimSpace(event.Currency)),
		OccurredAt:          occurredAt,
		RawPayload:          payload,
		InvoiceID:           invoiceID,
	}, nil
}

func (a *Adapter) parsePaymentFailed(event xenditEvent, payload []byte) (*paymentdomain.PaymentEvent, error) {
	customerID, invoiceID, err := parseExternalID(event.ExternalID)
	if err != nil {
		return nil, paymentdomain.ErrInvalidCustomer
	}

	amount := int64(event.Amount * 100)
	occurredAt, _ := time.Parse(time.RFC3339, event.Created)
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return &paymentdomain.PaymentEvent{
		Provider:            "xendit",
		ProviderEventID:     event.ID,
		ProviderPaymentID:   event.ID,
		ProviderPaymentType: "xendit_payment",
		Type:                paymentdomain.EventTypePaymentFailed,
		OrgID:               a.orgID,
		CustomerID:          customerID,
		Amount:              amount,
		Currency:            strings.ToUpper(strings.TrimSpace(event.Currency)),
		OccurredAt:          occurredAt,
		RawPayload:          payload,
		InvoiceID:           invoiceID,
	}, nil
}

// parseExternalID extracts customer_id and invoice_id from external_id
// Format: "customer_{customerID}_invoice_{invoiceID}"
func parseExternalID(externalID string) (snowflake.ID, *snowflake.ID, error) {
	parts := strings.Split(externalID, "_")
	if len(parts) < 2 {
		return 0, nil, errors.New("invalid external_id format")
	}

	// Find customer_id
	var customerIDStr string
	for i, part := range parts {
		if part == "customer" && i+1 < len(parts) {
			customerIDStr = parts[i+1]
			break
		}
	}

	if customerIDStr == "" {
		return 0, nil, errors.New("customer_id not found in external_id")
	}

	customerID, err := snowflake.ParseString(customerIDStr)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid customer_id: %w", err)
	}

	// Find invoice_id (optional)
	var invoiceIDStr string
	for i, part := range parts {
		if part == "invoice" && i+1 < len(parts) {
			invoiceIDStr = parts[i+1]
			break
		}
	}

	if invoiceIDStr == "" {
		return customerID, nil, nil
	}

	invoiceID, err := snowflake.ParseString(invoiceIDStr)
	if err != nil {
		return customerID, nil, nil // Invoice ID is optional
	}

	return customerID, &invoiceID, nil
}

func readString(config map[string]any, key string) (string, bool) {
	value, ok := config[key]
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}
