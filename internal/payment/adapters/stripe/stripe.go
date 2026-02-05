package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	disputedomain "github.com/railzwaylabs/railzway/internal/payment/dispute/domain"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Provider() string {
	return "stripe"
}

func (f *Factory) NewAdapter(cfg paymentdomain.AdapterConfig) (paymentdomain.PaymentAdapter, error) {
	secret, ok := readString(cfg.Config, "webhook_secret")
	if !ok {
		return nil, paymentdomain.ErrInvalidConfig
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, paymentdomain.ErrInvalidConfig
	}

	// Extract API key for payment method operations
	apiKey, ok := readString(cfg.Config, "api_key")
	if !ok {
		apiKey = "" // API key is optional for webhook-only usage
	}

	return &Adapter{
		orgID:         cfg.OrgID,
		webhookSecret: secret,
		apiKey:        strings.TrimSpace(apiKey),
	}, nil
}

type Adapter struct {
	orgID         snowflake.ID
	webhookSecret string
	apiKey        string
}

func (a *Adapter) Verify(ctx context.Context, payload []byte, headers http.Header) error {
	sigHeader := strings.TrimSpace(headers.Get("Stripe-Signature"))
	if sigHeader == "" {
		return paymentdomain.ErrInvalidSignature
	}

	timestamp, signatures, err := parseStripeSignature(sigHeader)
	if err != nil {
		return paymentdomain.ErrInvalidSignature
	}

	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(a.webhookSecret))
	_, _ = mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, signature := range signatures {
		if hmac.Equal([]byte(signature), []byte(expected)) {
			return nil
		}
	}

	return paymentdomain.ErrInvalidSignature
}

func (a *Adapter) Parse(ctx context.Context, payload []byte) (*paymentdomain.PaymentEvent, error) {
	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}
	if strings.TrimSpace(event.ID) == "" {
		return nil, paymentdomain.ErrInvalidEvent
	}

	// Log event type for debugging
	fmt.Printf("[STRIPE] Processing event: type=%s, id=%s\n", event.Type, event.ID)

	switch strings.TrimSpace(event.Type) {
	case "payment_intent.succeeded":
		return a.parsePaymentIntent(event, payload)
	case "payment_intent.payment_failed":
		return a.parsePaymentIntentFailed(event, payload)
	case "charge.succeeded":
		return a.parseCharge(event, payload, paymentdomain.EventTypePaymentSucceeded)
	case "checkout.session.completed":
		return a.parseCheckoutSessionCompleted(event, payload)
	case "charge.refunded":
		return a.parseCharge(event, payload, paymentdomain.EventTypeRefunded)
	default:
		fmt.Printf("[STRIPE] Unhandled event type: %s (event_id=%s)\n", event.Type, event.ID)
		return nil, paymentdomain.ErrEventIgnored
	}
}

func (a *Adapter) ParseDispute(ctx context.Context, payload []byte) (*disputedomain.DisputeEvent, error) {
	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}
	if strings.TrimSpace(event.ID) == "" {
		return nil, paymentdomain.ErrInvalidEvent
	}

	var disputeType string
	switch strings.TrimSpace(event.Type) {
	case "charge.dispute.created":
		disputeType = disputedomain.EventTypeDisputeCreated
	case "charge.dispute.funds_withdrawn":
		disputeType = disputedomain.EventTypeDisputeFundsWithdrawn
	case "charge.dispute.funds_reinstated":
		disputeType = disputedomain.EventTypeDisputeFundsReinstated
	case "charge.dispute.closed":
		disputeType = disputedomain.EventTypeDisputeClosed
	default:
		return nil, paymentdomain.ErrEventIgnored
	}

	var dispute stripeDispute
	if err := json.Unmarshal(event.Data.Object, &dispute); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}
	if strings.TrimSpace(dispute.ID) == "" {
		return nil, paymentdomain.ErrInvalidEvent
	}

	customerID, _, err := parseMetadataIDs(dispute.Metadata)
	if err != nil {
		return nil, err
	}

	occurredAt := timestamp(dispute.Created, event.Created)
	return &disputedomain.DisputeEvent{
		Provider:          "stripe",
		ProviderEventID:   event.ID,
		ProviderDisputeID: dispute.ID,
		Type:              disputeType,
		OrgID:             a.orgID,
		CustomerID:        customerID,
		Amount:            dispute.Amount,
		Currency:          strings.ToUpper(strings.TrimSpace(dispute.Currency)),
		Reason:            strings.TrimSpace(dispute.Reason),
		OccurredAt:        occurredAt,
		RawPayload:        payload,
	}, nil
}

type stripeEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Created int64           `json:"created"`
	Data    stripeEventData `json:"data"`
}

type stripeCheckoutSession struct {
	ID                string         `json:"id"`
	ClientReferenceID string         `json:"client_reference_id"`
	PaymentIntent     any            `json:"payment_intent"` // Can be string ID or expanded object
	Status            string         `json:"status"`
	PaymentStatus     string         `json:"payment_status"`
	AmountTotal       int64          `json:"amount_total"`
	Currency          string         `json:"currency"`
	Created           int64          `json:"created"`
	ExpiresAt         int64          `json:"expires_at"`
	Metadata          map[string]any `json:"metadata"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripePaymentIntent struct {
	ID             string         `json:"id"`
	Amount         int64          `json:"amount"`
	AmountReceived int64          `json:"amount_received"`
	Currency       string         `json:"currency"`
	Created        int64          `json:"created"`
	Metadata       map[string]any `json:"metadata"`
}

type stripeCharge struct {
	ID             string         `json:"id"`
	Amount         int64          `json:"amount"`
	AmountRefunded int64          `json:"amount_refunded"`
	Currency       string         `json:"currency"`
	Created        int64          `json:"created"`
	Metadata       map[string]any `json:"metadata"`
}

type stripeDispute struct {
	ID       string         `json:"id"`
	Amount   int64          `json:"amount"`
	Currency string         `json:"currency"`
	Reason   string         `json:"reason"`
	Created  int64          `json:"created"`
	Metadata map[string]any `json:"metadata"`
}

func (a *Adapter) parsePaymentIntent(event stripeEvent, payload []byte) (*paymentdomain.PaymentEvent, error) {
	var intent stripePaymentIntent
	if err := json.Unmarshal(event.Data.Object, &intent); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}

	amount := intent.AmountReceived
	if amount <= 0 {
		amount = intent.Amount
	}
	customerID, invoiceID, err := parseMetadataIDs(intent.Metadata)
	if err != nil {
		return nil, err
	}

	occurredAt := timestamp(intent.Created, event.Created)
	return &paymentdomain.PaymentEvent{
		Provider:            "stripe",
		ProviderEventID:     event.ID,
		ProviderPaymentID:   intent.ID,
		ProviderPaymentType: "payment_intent",
		Type:                paymentdomain.EventTypePaymentSucceeded,
		OrgID:               a.orgID,
		CustomerID:          customerID,
		Amount:              amount,
		Currency:            strings.ToUpper(strings.TrimSpace(intent.Currency)),
		OccurredAt:          occurredAt,
		RawPayload:          payload,
		InvoiceID:           invoiceID,
	}, nil
}

func (a *Adapter) parsePaymentIntentFailed(event stripeEvent, payload []byte) (*paymentdomain.PaymentEvent, error) {
	var intent stripePaymentIntent
	if err := json.Unmarshal(event.Data.Object, &intent); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}

	customerID, invoiceID, err := parseMetadataIDs(intent.Metadata)
	if err != nil {
		return nil, err
	}

	occurredAt := timestamp(intent.Created, event.Created)
	return &paymentdomain.PaymentEvent{
		Provider:            "stripe",
		ProviderEventID:     event.ID,
		ProviderPaymentID:   intent.ID,
		ProviderPaymentType: "payment_intent",
		Type:                paymentdomain.EventTypePaymentFailed,
		OrgID:               a.orgID,
		CustomerID:          customerID,
		Amount:              intent.Amount,
		Currency:            strings.ToUpper(strings.TrimSpace(intent.Currency)),
		OccurredAt:          occurredAt,
		RawPayload:          payload,
		InvoiceID:           invoiceID,
	}, nil
}

func (a *Adapter) parseCharge(event stripeEvent, payload []byte, eventType string) (*paymentdomain.PaymentEvent, error) {
	var charge stripeCharge
	if err := json.Unmarshal(event.Data.Object, &charge); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}

	amount := charge.Amount
	if eventType == paymentdomain.EventTypeRefunded && charge.AmountRefunded > 0 {
		amount = charge.AmountRefunded
	}
	customerID, invoiceID, err := parseMetadataIDs(charge.Metadata)
	if err != nil {
		return nil, err
	}

	occurredAt := timestamp(charge.Created, event.Created)
	return &paymentdomain.PaymentEvent{
		Provider:            "stripe",
		ProviderEventID:     event.ID,
		ProviderPaymentID:   charge.ID,
		ProviderPaymentType: "charge",
		Type:                eventType,
		OrgID:               a.orgID,
		CustomerID:          customerID,
		Amount:              amount,
		Currency:            strings.ToUpper(strings.TrimSpace(charge.Currency)),
		OccurredAt:          occurredAt,
		RawPayload:          payload,
		InvoiceID:           invoiceID,
	}, nil
}

func (a *Adapter) parseCheckoutSessionCompleted(event stripeEvent, payload []byte) (*paymentdomain.PaymentEvent, error) {
	var session stripeCheckoutSession
	if err := json.Unmarshal(event.Data.Object, &session); err != nil {
		return nil, paymentdomain.ErrInvalidPayload
	}

	customerIDStr := session.ClientReferenceID
	if customerIDStr == "" {
		// Fallback to metadata if available
		customerIDStr = readMetadataValue(session.Metadata, "customer_id")
	}

	var customerID snowflake.ID
	var err error
	if customerIDStr != "" {
		customerID, err = snowflake.ParseString(customerIDStr)
		if err != nil {
			fmt.Printf("[STRIPE] Invalid customer_id format in checkout session: %s\n", customerIDStr)
			return nil, paymentdomain.ErrInvalidCustomer
		}
	} else {
		// Log session details for debugging
		fmt.Printf("[STRIPE] Missing customer_id in checkout session: client_reference_id=%s, metadata=%+v, session_id=%s\n",
			session.ClientReferenceID, session.Metadata, session.ID)
		return nil, paymentdomain.ErrInvalidCustomer
	}

	occurredAt := timestamp(session.Created, event.Created)

	return &paymentdomain.PaymentEvent{
		Provider:            "stripe",
		ProviderEventID:     event.ID,
		ProviderPaymentID:   session.ID, // Use Session ID as ProviderPaymentID for this event type
		ProviderPaymentType: "checkout_session",
		Type:                paymentdomain.EventTypeCheckoutSessionCompleted,
		OrgID:               a.orgID,
		CustomerID:          customerID,
		Amount:              session.AmountTotal,
		Currency:            strings.ToUpper(strings.TrimSpace(session.Currency)),
		OccurredAt:          occurredAt,
		RawPayload:          payload,
		InvoiceID:           nil, // Not an invoice event
	}, nil
}

func parseStripeSignature(header string) (string, []string, error) {
	parts := strings.Split(header, ",")
	var timestamp string
	signatures := []string{}
	for _, part := range parts {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		keyValue := strings.SplitN(piece, "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])
		if key == "t" {
			timestamp = value
		}
		if key == "v1" {
			signatures = append(signatures, value)
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return "", nil, errors.New("invalid_signature")
	}
	return timestamp, signatures, nil
}

func timestamp(primary int64, fallback int64) time.Time {
	value := primary
	if value == 0 {
		value = fallback
	}
	if value == 0 {
		return time.Now().UTC()
	}
	return time.Unix(value, 0).UTC()
}

func parseMetadataIDs(metadata map[string]any) (snowflake.ID, *snowflake.ID, error) {
	customerRaw := readMetadataValue(metadata, "customer_id")
	if customerRaw == "" {
		// Log metadata for debugging
		fmt.Printf("[STRIPE] Missing customer_id in metadata: %+v\n", metadata)
		return 0, nil, paymentdomain.ErrInvalidCustomer
	}
	customerID, err := snowflake.ParseString(customerRaw)
	if err != nil {
		fmt.Printf("[STRIPE] Invalid customer_id format: %s\n", customerRaw)
		return 0, nil, paymentdomain.ErrInvalidCustomer
	}

	invoiceRaw := readMetadataValue(metadata, "invoice_id")
	if invoiceRaw == "" {
		return customerID, nil, nil
	}
	invoiceID, err := snowflake.ParseString(invoiceRaw)
	if err != nil {
		return customerID, nil, nil
	}
	return customerID, &invoiceID, nil
}

func readMetadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	switch cast := value.(type) {
	case string:
		return strings.TrimSpace(cast)
	case float64:
		if cast == 0 {
			return ""
		}
		return strconv.FormatInt(int64(cast), 10)
	case json.Number:
		return cast.String()
	case int64:
		return strconv.FormatInt(cast, 10)
	case int:
		return strconv.Itoa(cast)
	}
	return ""
}

// AttachPaymentMethod attaches a payment method to a customer
func (a *Adapter) AttachPaymentMethod(ctx context.Context, customerProviderID, token string) (*paymentdomain.PaymentMethodDetails, error) {
	if a.apiKey == "" {
		return nil, errors.New("stripe api key not configured")
	}

	// Call Stripe API: POST /v1/payment_methods/{token}/attach
	endpoint := fmt.Sprintf("https://api.stripe.com/v1/payment_methods/%s/attach", token)

	data := url.Values{}
	data.Set("customer", customerProviderID)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		// If already attached, just retrieve it
		if strings.Contains(bodyStr, "already attached to a customer") {
			return a.retrievePaymentMethod(ctx, token)
		}

		return nil, fmt.Errorf("stripe api error: %d body: %s", resp.StatusCode, bodyStr)
	}

	var pm stripePaymentMethod
	if err := json.NewDecoder(resp.Body).Decode(&pm); err != nil {
		return nil, err
	}

	return &paymentdomain.PaymentMethodDetails{
		ID:       pm.ID,
		Type:     pm.Type,
		Last4:    pm.Card.Last4,
		Brand:    pm.Card.Brand,
		ExpMonth: pm.Card.ExpMonth,
		ExpYear:  pm.Card.ExpYear,
	}, nil
}

func (a *Adapter) retrievePaymentMethod(ctx context.Context, id string) (*paymentdomain.PaymentMethodDetails, error) {
	if a.apiKey == "" {
		return nil, errors.New("stripe api key not configured")
	}
	endpoint := fmt.Sprintf("https://api.stripe.com/v1/payment_methods/%s", id)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stripe api error retrieving pm: %d", resp.StatusCode)
	}
	var pm stripePaymentMethod
	if err := json.NewDecoder(resp.Body).Decode(&pm); err != nil {
		return nil, err
	}
	return &paymentdomain.PaymentMethodDetails{
		ID:       pm.ID,
		Type:     pm.Type,
		Last4:    pm.Card.Last4,
		Brand:    pm.Card.Brand,
		ExpMonth: pm.Card.ExpMonth,
		ExpYear:  pm.Card.ExpYear,
	}, nil
}

// DetachPaymentMethod detaches a payment method
func (a *Adapter) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	if a.apiKey == "" {
		return errors.New("stripe api key not configured")
	}

	// Call Stripe API: POST /v1/payment_methods/{id}/detach
	url := fmt.Sprintf("https://api.stripe.com/v1/payment_methods/%s/detach", paymentMethodID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stripe api error: %d", resp.StatusCode)
	}

	return nil
}

// CreateCheckoutSession creates a new checkout session
func (a *Adapter) CreateCheckoutSession(ctx context.Context, input paymentdomain.CheckoutSessionInput) (*paymentdomain.ProviderCheckoutSession, error) {
	if a.apiKey == "" {
		return nil, errors.New("stripe api key not configured")
	}

	// Call Stripe API: POST /v1/checkout/sessions
	endpoint := "https://api.stripe.com/v1/checkout/sessions"

	data := url.Values{}
	data.Set("mode", "payment")
	data.Set("success_url", input.SuccessURL)
	data.Set("cancel_url", input.CancelURL)
	data.Set("line_items[0][price_data][currency]", strings.ToLower(input.Currency))
	data.Set("line_items[0][price_data][product_data][name]", "Payment") // Generic name for now
	data.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(input.Amount, 10))
	data.Set("line_items[0][quantity]", "1")
	data.Set("payment_intent_data[setup_future_usage]", "off_session")

	if input.CustomerID != 0 {
		data.Set("client_reference_id", input.CustomerID.String())
	}

	if input.ProviderCustomerID != "" {
		data.Set("customer", input.ProviderCustomerID)
	}

	if input.AllowPromotionCodes {
		data.Set("allow_promotion_codes", "true")
	}

	// Force add internal customer ID to metadata for robust webhook handling
	if input.CustomerID != 0 {
		data.Set("metadata[customer_id]", input.CustomerID.String())
		// Also set in payment_intent_data so payment_intent.succeeded webhook has customer_id
		data.Set("payment_intent_data[metadata][customer_id]", input.CustomerID.String())
	}

	for k, v := range input.Metadata {
		data.Set("metadata["+k+"]", v)
		// Also copy to payment_intent_data metadata
		data.Set("payment_intent_data[metadata]["+k+"]", v)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stripe api error: %d body: %s", resp.StatusCode, string(bodyBytes))
	}

	var session struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Status    string `json:"status"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	status := paymentdomain.CheckoutSessionStatusOpen
	switch session.Status {
	case "complete":
		status = paymentdomain.CheckoutSessionStatusComplete
	case "expired":
		status = paymentdomain.CheckoutSessionStatusExpired
	}

	return &paymentdomain.ProviderCheckoutSession{
		ID:        session.ID,
		Provider:  "stripe",
		URL:       session.URL,
		Status:    status,
		ExpiresAt: time.Unix(session.ExpiresAt, 0),
	}, nil
}

// RetrieveCheckoutSession retrieves a checkout session from Stripe
func (a *Adapter) RetrieveCheckoutSession(ctx context.Context, providerSessionID string) (*paymentdomain.ProviderCheckoutSession, error) {
	if a.apiKey == "" {
		return nil, errors.New("stripe api key not configured")
	}

	// Call Stripe API: GET /v1/checkout/sessions/{id}?expand[]=payment_intent
	url := fmt.Sprintf("https://api.stripe.com/v1/checkout/sessions/%s?expand[]=payment_intent", providerSessionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stripe api error: %d", resp.StatusCode)
	}

	var session stripeCheckoutSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	status := paymentdomain.CheckoutSessionStatusOpen
	switch session.Status {
	case "complete":
		status = paymentdomain.CheckoutSessionStatusComplete
	case "expired":
		status = paymentdomain.CheckoutSessionStatusExpired
	}

	paymentMethodID := ""
	paymentIntentID := ""

	// Extract PaymentIntent ID and PaymentMethod ID from expanded object
	if session.PaymentIntent != nil {
		switch pi := session.PaymentIntent.(type) {
		case string:
			paymentIntentID = pi
		case map[string]any:
			if id, ok := pi["id"].(string); ok {
				paymentIntentID = id
			}
			if pm, ok := pi["payment_method"].(string); ok {
				paymentMethodID = pm
			}
		}
	}

	return &paymentdomain.ProviderCheckoutSession{
		ID:              session.ID,
		Provider:        "stripe",
		Status:          status,
		ExpiresAt:       time.Unix(session.ExpiresAt, 0),
		PaymentMethodID: paymentMethodID,
		PaymentIntentID: paymentIntentID,
	}, nil
}

// GetPaymentMethod retrieves payment method details
func (a *Adapter) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*paymentdomain.PaymentMethodDetails, error) {
	if a.apiKey == "" {
		return nil, errors.New("stripe api key not configured")
	}

	// Call Stripe API: GET /v1/payment_methods/{id}
	url := fmt.Sprintf("https://api.stripe.com/v1/payment_methods/%s", paymentMethodID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stripe api error: %d", resp.StatusCode)
	}

	var pm stripePaymentMethod
	if err := json.NewDecoder(resp.Body).Decode(&pm); err != nil {
		return nil, err
	}

	return &paymentdomain.PaymentMethodDetails{
		ID:       pm.ID,
		Type:     pm.Type,
		Last4:    pm.Card.Last4,
		Brand:    pm.Card.Brand,
		ExpMonth: pm.Card.ExpMonth,
		ExpYear:  pm.Card.ExpYear,
	}, nil
}

// ListPaymentMethods lists customer payment methods
func (a *Adapter) ListPaymentMethods(ctx context.Context, customerProviderID string) ([]*paymentdomain.PaymentMethodDetails, error) {
	if a.apiKey == "" {
		return nil, errors.New("stripe api key not configured")
	}

	// Call Stripe API: GET /v1/payment_methods?customer={id}&type=card
	url := fmt.Sprintf("https://api.stripe.com/v1/payment_methods?customer=%s&type=card", customerProviderID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stripe api error: %d", resp.StatusCode)
	}

	var listResp stripePaymentMethodList
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	result := make([]*paymentdomain.PaymentMethodDetails, len(listResp.Data))
	for i, pm := range listResp.Data {
		result[i] = &paymentdomain.PaymentMethodDetails{
			ID:       pm.ID,
			Type:     pm.Type,
			Last4:    pm.Card.Last4,
			Brand:    pm.Card.Brand,
			ExpMonth: pm.Card.ExpMonth,
			ExpYear:  pm.Card.ExpYear,
		}
	}

	return result, nil
}

// Stripe API response structures
type stripePaymentMethod struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Card struct {
		Last4    string `json:"last4"`
		Brand    string `json:"brand"`
		ExpMonth int    `json:"exp_month"`
		ExpYear  int    `json:"exp_year"`
	} `json:"card"`
}

type stripePaymentMethodList struct {
	Data []stripePaymentMethod `json:"data"`
}

func readString(config map[string]any, key string) (string, bool) {
	value, ok := config[key]
	if !ok {
		return "", false
	}
	switch cast := value.(type) {
	case string:
		return cast, true
	default:
		return "", false
	}
}
