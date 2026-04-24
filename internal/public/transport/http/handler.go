package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appsdomain "github.com/railzwaylabs/railzway/internal/apps/domain"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	"github.com/railzwaylabs/railzway/internal/config"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	"github.com/railzwaylabs/railzway/internal/invoice/domain"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	"github.com/railzwaylabs/railzway/internal/publiclink"
)

// Handler handles unauthenticated invoice-viewer and checkout routes under /public/*.
type Handler struct {
	cfg         *config.Config
	invoices    domain.Service
	customers   customerdomain.Service
	orgs        organizationdomain.Service
	apps        appsdomain.Service
	audit       *auditlog.Service
	supportMu   sync.Mutex
	supportLast map[string]time.Time
	supportTTL  time.Duration
}

func NewHandler(
	cfg *config.Config,
	invoices domain.Service,
	customers customerdomain.Service,
	orgs organizationdomain.Service,
	apps appsdomain.Service,
	audit *auditlog.Service,
) *Handler {
	return &Handler{
		cfg:         cfg,
		invoices:    invoices,
		customers:   customers,
		orgs:        orgs,
		apps:        apps,
		audit:       audit,
		supportLast: map[string]time.Time{},
		supportTTL:  10 * time.Minute,
	}
}

// RegisterRoutes registers unauthenticated invoice-viewer routes under /public.
func RegisterRoutes(r *gin.Engine, h *Handler) {
	group := r.Group("/public")
	group.GET("/invoices/:token", h.GetInvoice)
	group.GET("/invoices/:token/payment-options", h.GetPaymentOptions)
	group.POST("/invoices/:token/checkout", h.CreateCheckoutSession)
	group.POST("/invoices/:token/support", h.RequestSupport)
}

type publicInvoiceResponse struct {
	Invoice            domain.Invoice                           `json:"invoice"`
	Items              []domain.InvoiceItem                     `json:"items"`
	Organization       *organizationdomain.OrganizationResponse `json:"organization,omitempty"`
	Customer           *customerdomain.CustomerResponse         `json:"customer,omitempty"`
	PaymentMethods     []string                                 `json:"payment_methods"`
	PaymentConfigured  bool                                     `json:"payment_configured"`
	BillingCountryCode string                                   `json:"billing_country"`
	ExpiresAt          time.Time                                `json:"expires_at"`
}

type paymentOptionsResponse struct {
	PaymentMethods     []string  `json:"payment_methods"`
	PaymentConfigured  bool      `json:"payment_configured"`
	BillingCountryCode string    `json:"billing_country"`
	ExpiresAt          time.Time `json:"expires_at"`
}

type supportResponse struct {
	Status string `json:"status"`
}

type checkoutResponse struct {
	Status      string `json:"status"`
	CheckoutURL string `json:"checkout_url,omitempty"`
}

func (h *Handler) GetInvoice(c *gin.Context) {
	invoiceID, orgID, exp, err := h.parseToken(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.invoices.GetInvoice(ctx, domain.GetInvoiceRequest{ID: invoiceID.String()})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}

	org, _ := h.orgs.GetByID(ctx, orgID.String())
	var custPtr *customerdomain.CustomerResponse
	if cust, err := h.customers.GetByID(ctx, customerdomain.GetCustomerRequest{ID: resp.Invoice.CustomerID.String()}); err == nil {
		custPtr = &cust
	}

	billingCountry := ""
	if org != nil && org.CountryCode != "" {
		billingCountry = org.CountryCode
	}

	methods, configured := h.resolvePaymentMethods(ctx, billingCountry)

	c.JSON(http.StatusOK, publicInvoiceResponse{
		Invoice:            resp.Invoice,
		Items:              resp.Items,
		Organization:       org,
		Customer:           custPtr,
		PaymentMethods:     methods,
		PaymentConfigured:  configured,
		BillingCountryCode: billingCountry,
		ExpiresAt:          exp,
	})
}

func (h *Handler) GetPaymentOptions(c *gin.Context) {
	_, orgID, exp, err := h.parseToken(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	billingCountry := strings.TrimSpace(c.Query("country"))
	if billingCountry == "" {
		org, _ := h.orgs.GetByID(ctx, orgID.String())
		if org != nil {
			billingCountry = org.CountryCode
		}
	}

	methods, configured := h.resolvePaymentMethods(ctx, billingCountry)
	c.JSON(http.StatusOK, paymentOptionsResponse{
		PaymentMethods:     methods,
		PaymentConfigured:  configured,
		BillingCountryCode: billingCountry,
		ExpiresAt:          exp,
	})
}

func (h *Handler) RequestSupport(c *gin.Context) {
	invoiceID, orgID, _, err := h.parseToken(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	now := time.Now().UTC()
	key := invoiceID.String()
	h.supportMu.Lock()
	if last, ok := h.supportLast[key]; ok && now.Sub(last) < h.supportTTL {
		h.supportMu.Unlock()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "support_request_throttled"})
		return
	}
	h.supportLast[key] = now
	h.supportMu.Unlock()

	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	inv, err := h.invoices.GetInvoice(ctx, domain.GetInvoiceRequest{ID: invoiceID.String()})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}

	meta := map[string]interface{}{
		"channel": "email",
		"reason":  "payment_unavailable",
	}
	recordAudit(h.audit, ctx, orgID, inv.Invoice.ID.String(), meta)

	c.JSON(http.StatusOK, supportResponse{Status: "queued"})
}

func (h *Handler) CreateCheckoutSession(c *gin.Context) {
	invoiceID, orgID, _, err := h.parseToken(c.Param("token"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	if !h.hasPaymentProvider(ctx) {
		c.JSON(http.StatusConflict, gin.H{"error": "payment_unavailable"})
		return
	}
	inv, err := h.invoices.GetInvoice(ctx, domain.GetInvoiceRequest{ID: invoiceID.String()})
	if err != nil {
		writeInvoiceError(c, err)
		return
	}
	checkoutURL := extractCheckoutURL(inv.Invoice.Metadata)
	if checkoutURL == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "checkout_unavailable"})
		return
	}
	c.JSON(http.StatusOK, checkoutResponse{Status: "ready", CheckoutURL: checkoutURL})
}

func extractCheckoutURL(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	if v, ok := meta["checkout_url"]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func recordAudit(audit *auditlog.Service, ctx context.Context, orgID uuid.UUID, invoiceID string, meta map[string]interface{}) {
	if audit == nil {
		return
	}
	_ = audit.Record(ctx, auditlog.RecordInput{
		OrgID:        orgID,
		ActorType:    "customer",
		Action:       "invoice.support_request",
		ResourceType: "invoice",
		ResourceID:   &invoiceID,
		Metadata:     defaultJSON(meta),
	})
}

func defaultJSON(meta map[string]interface{}) []byte {
	if len(meta) == 0 {
		return []byte(`{}`)
	}
	b, _ := json.Marshal(meta)
	return b
}

func (h *Handler) parseToken(token string) (uuid.UUID, uuid.UUID, time.Time, error) {
	secret := strings.TrimSpace(h.cfg.PublicLink.Secret)
	now := time.Now().UTC()
	invoiceID, orgID, exp, err := publiclink.ParseInvoiceToken(token, secret, now)
	return invoiceID, orgID, exp, err
}

func (h *Handler) resolvePaymentMethods(ctx context.Context, country string) ([]string, bool) {
	configured := h.hasPaymentProvider(ctx)
	if !configured {
		return []string{}, false
	}
	code := strings.ToUpper(strings.TrimSpace(country))
	switch code {
	case "ID":
		return []string{"card", "qris", "va", "gopay", "ovo"}, true
	case "SG":
		return []string{"card", "grabpay"}, true
	case "MY":
		return []string{"card", "fpx"}, true
	case "US":
		return []string{"card", "ach"}, true
	case "GB", "DE", "FR", "ES", "NL", "EU":
		return []string{"card", "sepa"}, true
	default:
		return []string{"card"}, true
	}
}

func (h *Handler) hasPaymentProvider(ctx context.Context) bool {
	if h.apps == nil {
		return false
	}
	catalog, err := h.apps.ListCatalog(ctx)
	if err != nil {
		return false
	}
	catalogByID := map[string]appsdomain.AppDefinition{}
	for _, app := range catalog.Apps {
		catalogByID[app.ID] = app
	}
	installations, err := h.apps.ListInstallations(ctx)
	if err != nil {
		return false
	}
	for _, inst := range installations.Installations {
		app, ok := catalogByID[inst.AppID]
		if !ok {
			continue
		}
		if app.Category != appsdomain.CategoryPayment {
			continue
		}
		if inst.Status == appsdomain.InstallStatusActive {
			return true
		}
	}
	return false
}
