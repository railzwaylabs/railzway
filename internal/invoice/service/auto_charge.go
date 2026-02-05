package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type stripeAutoChargeIntent struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type stripeAutoChargeError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type stripeAutoChargeClient struct {
	apiKey    string
	accountID string
	client    *http.Client
}

func newStripeAutoChargeClient(apiKey string, accountID string) *stripeAutoChargeClient {
	return &stripeAutoChargeClient{
		apiKey:    strings.TrimSpace(apiKey),
		accountID: strings.TrimSpace(accountID),
		client:    &http.Client{Timeout: 12 * time.Second},
	}
}

type xenditInvoiceResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type xenditAutoChargeClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func newXenditAutoChargeClient(apiKey string, baseURL string) *xenditAutoChargeClient {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.xendit.co"
	}
	return &xenditAutoChargeClient{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: baseURL,
		client:  &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *xenditAutoChargeClient) createInvoice(
	ctx context.Context,
	invoice *invoicedomain.Invoice,
	amount float64,
	externalID string,
) (xenditInvoiceResponse, error) {
	if invoice == nil {
		return xenditInvoiceResponse{}, paymentdomain.ErrInvalidConfig
	}
	payload := map[string]any{
		"external_id": externalID,
		"amount":      amount,
		"currency":    strings.ToUpper(strings.TrimSpace(invoice.Currency)),
		"description": invoice.InvoiceNumber,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return xenditInvoiceResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/invoices", bytes.NewReader(body))
	if err != nil {
		return xenditInvoiceResponse{}, err
	}
	req.SetBasicAuth(c.apiKey, "")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return xenditInvoiceResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return xenditInvoiceResponse{}, errors.New("xendit_request_failed")
	}

	var out xenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return xenditInvoiceResponse{}, err
	}
	if strings.TrimSpace(out.ID) == "" {
		return xenditInvoiceResponse{}, errors.New("xendit_response_invalid")
	}
	return out, nil
}

func (c *stripeAutoChargeClient) createAndConfirmPaymentIntent(
	ctx context.Context,
	invoice *invoicedomain.Invoice,
	amount int64,
	paymentMethodID string,
	customerProviderID string,
) (stripeAutoChargeIntent, error) {
	if invoice == nil {
		return stripeAutoChargeIntent{}, paymentdomain.ErrInvalidConfig
	}
	values := url.Values{}
	values.Set("amount", strconv.FormatInt(amount, 10))
	values.Set("currency", strings.ToLower(invoice.Currency))
	values.Set("payment_method", strings.TrimSpace(paymentMethodID))
	values.Set("confirm", "true")
	values.Set("off_session", "true")
	values.Set("metadata[invoice_id]", invoice.ID.String())
	values.Set("metadata[invoice_number]", invoice.InvoiceNumber)
	values.Set("metadata[org_id]", invoice.OrgID.String())
	values.Set("metadata[customer_id]", invoice.CustomerID.String())
	if strings.TrimSpace(customerProviderID) != "" {
		values.Set("customer", strings.TrimSpace(customerProviderID))
	}

	return c.doRequest(ctx, http.MethodPost, "/v1/payment_intents", values, "auto_charge:"+invoice.ID.String())
}

func (c *stripeAutoChargeClient) doRequest(
	ctx context.Context,
	method string,
	path string,
	values url.Values,
	idempotencyKey string,
) (stripeAutoChargeIntent, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return stripeAutoChargeIntent{}, paymentdomain.ErrInvalidConfig
	}
	bodyReader := strings.NewReader(values.Encode())
	req, err := http.NewRequestWithContext(ctx, method, "https://api.stripe.com"+path, bodyReader)
	if err != nil {
		return stripeAutoChargeIntent{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if c.accountID != "" {
		req.Header.Set("Stripe-Account", c.accountID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return stripeAutoChargeIntent{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var stripeErr stripeAutoChargeError
		if err := json.NewDecoder(resp.Body).Decode(&stripeErr); err != nil {
			return stripeAutoChargeIntent{}, errors.New("stripe_request_failed")
		}
		message := strings.TrimSpace(stripeErr.Error.Message)
		if message == "" {
			message = "stripe_request_failed"
		}
		return stripeAutoChargeIntent{}, errors.New(message)
	}

	var intent stripeAutoChargeIntent
	if err := json.NewDecoder(resp.Body).Decode(&intent); err != nil {
		return stripeAutoChargeIntent{}, err
	}
	if intent.ID == "" {
		return stripeAutoChargeIntent{}, errors.New("stripe_response_invalid")
	}
	return intent, nil
}

func (s *Service) triggerAutoCharge(invoice *invoicedomain.Invoice) {
	if invoice == nil {
		return
	}
	if s.paymentMethodSvc == nil || s.paymentProviderSvc == nil {
		return
	}

	go func(inv *invoicedomain.Invoice) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.autoChargeInvoice(ctx, inv); err != nil {
			s.log.Warn("auto-charge failed", zap.Error(err), zap.String("invoice_id", inv.ID.String()))
		}
	}(invoice)
}

func (s *Service) autoChargeInvoice(ctx context.Context, invoice *invoicedomain.Invoice) error {
	if invoice == nil {
		return nil
	}
	if invoice.TotalAmount <= 0 || invoice.PaidAt != nil {
		return nil
	}
	if s.paymentMethodSvc == nil || s.paymentProviderSvc == nil {
		return nil
	}

	mode, err := s.loadSubscriptionCollectionMode(ctx, invoice.OrgID, invoice.SubscriptionID)
	if err != nil {
		return err
	}
	if mode != subscriptiondomain.SubscriptionCollectionModeChargeAutomatically {
		return nil
	}

	pm, err := s.paymentMethodSvc.GetDefaultPaymentMethod(ctx, invoice.CustomerID)
	if err != nil {
		if errors.Is(err, paymentdomain.ErrPaymentMethodNotFound) {
			s.recordAutoChargeFailure(ctx, invoice, "", "missing_payment_method", "")
			return nil
		}
		s.recordAutoChargeFailure(ctx, invoice, "", "payment_method_error", err.Error())
		return err
	}
	if pm == nil {
		s.recordAutoChargeFailure(ctx, invoice, "", "missing_payment_method", "")
		return nil
	}

	provider := strings.ToLower(strings.TrimSpace(pm.Provider))
	if provider == "" {
		s.recordAutoChargeFailure(ctx, invoice, "", "missing_provider", "")
		return nil
	}

	cfg, err := s.paymentProviderSvc.GetActiveProviderConfig(ctx, invoice.OrgID, provider)
	if err != nil {
		s.recordAutoChargeFailure(ctx, invoice, provider, "provider_config_error", err.Error())
		return err
	}

	var config map[string]any
	if err := json.Unmarshal(cfg.Config, &config); err != nil {
		s.recordAutoChargeFailure(ctx, invoice, provider, "provider_config_invalid", err.Error())
		return err
	}

	switch provider {
	case "stripe":
		return s.autoChargeStripe(ctx, invoice, pm.ProviderPaymentMethodID, config)
	case "xendit":
		return s.autoChargeXendit(ctx, invoice, config)
	default:
		s.recordAutoChargeFailure(ctx, invoice, provider, "provider_not_supported", "")
		return nil
	}
}

func (s *Service) autoChargeStripe(
	ctx context.Context,
	invoice *invoicedomain.Invoice,
	paymentMethodID string,
	config map[string]any,
) error {
	secret := readConfigString(config, "api_key", "secret_key")
	if secret == "" {
		err := paymentdomain.ErrInvalidConfig
		s.recordAutoChargeFailure(ctx, invoice, "stripe", "provider_config_invalid", err.Error())
		return err
	}
	paymentMethodID = strings.TrimSpace(paymentMethodID)
	if paymentMethodID == "" {
		s.recordAutoChargeFailure(ctx, invoice, "stripe", "missing_payment_method", "")
		return nil
	}

	customerProviderID := s.loadCustomerProviderID(ctx, invoice.CustomerID)
	accountID := readConfigString(config, "stripe_account_id")
	client := newStripeAutoChargeClient(secret, accountID)

	intent, err := client.createAndConfirmPaymentIntent(ctx, invoice, invoice.TotalAmount, paymentMethodID, customerProviderID)
	if err != nil {
		s.recordAutoChargeFailure(ctx, invoice, "stripe", "charge_failed", err.Error())
		return err
	}

	updates := map[string]any{
		"auto_charge_attempted_at":      time.Now().UTC().Format(time.RFC3339),
		"auto_charge_provider":          "stripe",
		"auto_charge_status":            intent.Status,
		"auto_charge_payment_intent_id": intent.ID,
		"auto_charge_payment_method_id": paymentMethodID,
		"payment_provider":              "stripe",
	}
	if accountID != "" {
		updates["stripe_account_id"] = accountID
	}
	return s.mergeInvoiceMetadata(ctx, invoice.OrgID, invoice.ID, updates)
}

func (s *Service) autoChargeXendit(
	ctx context.Context,
	invoice *invoicedomain.Invoice,
	config map[string]any,
) error {
	apiKey := readConfigString(config, "api_key")
	if apiKey == "" {
		err := paymentdomain.ErrInvalidConfig
		s.recordAutoChargeFailure(ctx, invoice, "xendit", "provider_config_invalid", err.Error())
		return err
	}

	amountMajor := amountToMajor(invoice.TotalAmount, invoice.Currency)
	if amountMajor <= 0 {
		return nil
	}

	externalID := buildXenditExternalID(invoice.CustomerID, invoice.ID)
	client := newXenditAutoChargeClient(apiKey, readConfigString(config, "base_url"))

	resp, err := client.createInvoice(ctx, invoice, amountMajor, externalID)
	if err != nil {
		s.recordAutoChargeFailure(ctx, invoice, "xendit", "charge_failed", err.Error())
		return err
	}

	updates := map[string]any{
		"auto_charge_attempted_at": time.Now().UTC().Format(time.RFC3339),
		"auto_charge_provider":     "xendit",
		"auto_charge_status":       resp.Status,
		"auto_charge_invoice_id":   resp.ID,
		"auto_charge_external_id":  externalID,
		"payment_provider":         "xendit",
	}
	return s.mergeInvoiceMetadata(ctx, invoice.OrgID, invoice.ID, updates)
}

func (s *Service) recordAutoChargeFailure(
	ctx context.Context,
	invoice *invoicedomain.Invoice,
	provider string,
	reason string,
	message string,
) {
	if invoice == nil {
		return
	}
	updates := map[string]any{
		"auto_charge_attempted_at": time.Now().UTC().Format(time.RFC3339),
		"auto_charge_status":       "failed",
	}
	if strings.TrimSpace(provider) != "" {
		updates["auto_charge_provider"] = strings.TrimSpace(provider)
	}
	if strings.TrimSpace(reason) != "" {
		updates["auto_charge_error_code"] = strings.TrimSpace(reason)
	}
	if strings.TrimSpace(message) != "" {
		updates["auto_charge_error_message"] = strings.TrimSpace(message)
	}
	if err := s.mergeInvoiceMetadata(ctx, invoice.OrgID, invoice.ID, updates); err != nil {
		s.log.Warn("failed to update invoice auto-charge metadata", zap.Error(err), zap.String("invoice_id", invoice.ID.String()))
	}
}

func (s *Service) mergeInvoiceMetadata(
	ctx context.Context,
	orgID snowflake.ID,
	invoiceID snowflake.ID,
	updates map[string]any,
) error {
	if orgID == 0 || invoiceID == 0 || len(updates) == 0 {
		return nil
	}

	var metadata datatypes.JSONMap
	if err := s.db.WithContext(ctx).Raw(
		`SELECT metadata FROM invoices WHERE org_id = ? AND id = ?`,
		orgID,
		invoiceID,
	).Scan(&metadata).Error; err != nil {
		return err
	}
	if metadata == nil {
		metadata = datatypes.JSONMap{}
	}
	for key, value := range updates {
		if strings.TrimSpace(key) == "" {
			continue
		}
		metadata[key] = value
	}

	return s.db.WithContext(ctx).Exec(
		`UPDATE invoices SET metadata = ?, updated_at = ? WHERE org_id = ? AND id = ?`,
		metadata,
		time.Now().UTC(),
		orgID,
		invoiceID,
	).Error
}

func (s *Service) loadSubscriptionCollectionMode(
	ctx context.Context,
	orgID snowflake.ID,
	subscriptionID snowflake.ID,
) (subscriptiondomain.SubscriptionCollectionMode, error) {
	var mode string
	if err := s.db.WithContext(ctx).Raw(
		`SELECT collection_mode FROM subscriptions WHERE org_id = ? AND id = ?`,
		orgID,
		subscriptionID,
	).Scan(&mode).Error; err != nil {
		return "", err
	}
	return subscriptiondomain.SubscriptionCollectionMode(strings.TrimSpace(mode)), nil
}

func (s *Service) loadCustomerProviderID(ctx context.Context, customerID snowflake.ID) string {
	var providerID string
	if err := s.db.WithContext(ctx).Raw(
		`SELECT provider_customer_id FROM customers WHERE id = ?`,
		customerID,
	).Scan(&providerID).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(providerID)
}

func readConfigString(config map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := config[key]
		if !ok {
			continue
		}
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

func amountToMajor(amount int64, currency string) float64 {
	c := strings.ToUpper(strings.TrimSpace(currency))
	decimals, ok := currencyDecimals[c]
	if !ok {
		decimals = 2
	}
	divisor := math.Pow10(decimals)
	if divisor == 0 {
		return float64(amount)
	}
	return float64(amount) / divisor
}

func buildXenditExternalID(customerID snowflake.ID, invoiceID snowflake.ID) string {
	if customerID == 0 {
		return ""
	}
	if invoiceID == 0 {
		return "customer_" + customerID.String()
	}
	return "customer_" + customerID.String() + "_invoice_" + invoiceID.String()
}
