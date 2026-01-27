package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	invoicetemplatedomain "github.com/railzwaylabs/railzway/internal/invoicetemplate/domain"
	meterdomain "github.com/railzwaylabs/railzway/internal/meter/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	pricedomain "github.com/railzwaylabs/railzway/internal/price/domain"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	taxdomain "github.com/railzwaylabs/railzway/internal/tax/domain"
)

type ReadinessState string

const (
	ReadinessStateReady    ReadinessState = "ready"
	ReadinessStateNotReady ReadinessState = "not_ready"
	ReadinessStateOptional ReadinessState = "optional"
)

type ReadinessIssue struct {
	ID             string            `json:"id"`
	Status         ReadinessState    `json:"status"`
	DependencyHint *string           `json:"dependency_hint,omitempty"`
	ActionHref     string            `json:"action_href"`
	Evidence       map[string]string `json:"evidence,omitempty"`
}

type ReadinessResponse struct {
	SystemState ReadinessState   `json:"system_state"`
	Issues      []ReadinessIssue `json:"issues"`
}

func (s *Server) GetOrganizationReadiness(c *gin.Context) {
	ctx := c.Request.Context()
	orgID, ok := orgcontext.OrgIDFromContext(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "organization context missing"})
		return
	}
	orgIDStr := orgID.String()

	issues := []ReadinessIssue{}
	isSystemReady := true

	// --- REQUIRED CHECKS ---

	// 1. Check Product Exists
	products, err := s.productSvc.List(ctx, productdomain.ListRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	activeProducts := 0
	for _, p := range products {
		if p.Active {
			activeProducts++
		}
	}

	if activeProducts == 0 {
		isSystemReady = false
		issues = append(issues, ReadinessIssue{
			ID:         "product_exists",
			Status:     ReadinessStateNotReady,
			ActionHref: "/products",
			Evidence:   map[string]string{"active_products": "0"},
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "product_exists",
			Status:     ReadinessStateReady,
			ActionHref: "/products",
		})
	}

	// 2. Check Price Exists
	prices, err := s.priceSvc.List(ctx, pricedomain.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list prices"})
		return
	}

	activePrices := 0
	hasUsagePrice := false
	for _, p := range prices {
		if p.Active {
			activePrices++
			if p.BillingMode == pricedomain.Metered {
				hasUsagePrice = true
			}
		}
	}

	if activePrices == 0 {
		isSystemReady = false
		issues = append(issues, ReadinessIssue{
			ID:             "price_exists_for_product",
			Status:         ReadinessStateNotReady,
			ActionHref:     "/prices",
			DependencyHint: ptr("product_exists"),
			Evidence:       map[string]string{"active_prices": "0"},
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "price_exists_for_product",
			Status:     ReadinessStateReady,
			ActionHref: "/prices",
		})
	}

	// 3. Check Meter Exists (Condition: If Usage Price Exists)
	if hasUsagePrice {
		meters, err := s.meterSvc.List(ctx, meterdomain.ListRequest{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list meters"})
			return
		}

		activeMeters := 0
		for _, m := range meters {
			if m.Active {
				activeMeters++
			}
		}

		if activeMeters == 0 {
			isSystemReady = false
			issues = append(issues, ReadinessIssue{
				ID:         "meter_exists_if_usage_price",
				Status:     ReadinessStateNotReady,
				ActionHref: "/meters",
				Evidence:   map[string]string{"active_meters": "0"},
			})
		} else {
			issues = append(issues, ReadinessIssue{
				ID:         "meter_exists_if_usage_price",
				Status:     ReadinessStateReady,
				ActionHref: "/meters",
			})
		}
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "meter_exists_if_usage_price",
			Status:     ReadinessStateReady,
			ActionHref: "/meters",
		})
	}

	// 4. Check Payment Provider Connected
	providers, err := s.paymentProviderSvc.ListConfigs(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list payment providers"})
		return
	}

	hasActiveProvider := false
	for _, p := range providers {
		if p.IsActive && p.Configured {
			hasActiveProvider = true
			break
		}
	}

	if !hasActiveProvider {
		isSystemReady = false
		issues = append(issues, ReadinessIssue{
			ID:         "payment_provider_connected",
			Status:     ReadinessStateNotReady,
			ActionHref: "/payment-providers",
			Evidence:   map[string]string{"active_providers": "0"},
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "payment_provider_connected",
			Status:     ReadinessStateReady,
			ActionHref: "/payment-providers",
		})
	}

	// 5. Check Payment Configuration (Methods)
	pmConfigs, err := s.paymentMethodConfigSvc.ListPaymentMethodConfigs(ctx, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list payment method configs"})
		return
	}

	hasActiveMethod := false
	for _, pm := range pmConfigs {
		if pm.IsActive {
			hasActiveMethod = true
			break
		}
	}

	if !hasActiveMethod {
		isSystemReady = false
		issues = append(issues, ReadinessIssue{
			ID:             "payment_configuration_complete",
			Status:         ReadinessStateNotReady,
			ActionHref:     "/payment-method-configs",
			DependencyHint: ptr("payment_provider_connected"),
			Evidence:       map[string]string{"active_methods": "0"},
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "payment_configuration_complete",
			Status:     ReadinessStateReady,
			ActionHref: "/payment-method-configs",
		})
	}

	// --- RECOMMENDED CHECKS (Does not affect SystemState) ---

	// 6. Invoice Template Customized (Optional)
	templates, err := s.invoiceTemplateSvc.List(ctx, invoicetemplatedomain.ListRequest{})
	if err == nil && len(templates) > 0 {
		issues = append(issues, ReadinessIssue{
			ID:         "invoice_template_customized",
			Status:     ReadinessStateReady,
			ActionHref: "/invoice-templates",
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "invoice_template_customized",
			Status:     ReadinessStateOptional,
			ActionHref: "/invoice-templates",
		})
	}

	// 7. Tax Configuration (Optional)
	taxDefs, err := s.taxSvc.List(ctx, taxdomain.ListRequest{IsEnabled: ptrBool(true)})
	if err == nil && len(taxDefs) > 0 {
		issues = append(issues, ReadinessIssue{
			ID:         "tax_configuration_explicit",
			Status:     ReadinessStateReady,
			ActionHref: "/tax-definitions",
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "tax_configuration_explicit",
			Status:     ReadinessStateOptional,
			ActionHref: "/tax-definitions",
		})
	}

	// 8. Secondary Admin Present (Optional)
	members, err := s.organizationSvc.ListMembers(ctx, orgIDStr)
	if err == nil && len(members) > 1 {
		issues = append(issues, ReadinessIssue{
			ID:         "secondary_admin_present",
			Status:     ReadinessStateReady,
			ActionHref: "/settings/team",
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "secondary_admin_present",
			Status:     ReadinessStateOptional,
			ActionHref: "/settings/team",
		})
	}

	// 9. API Keys Created (Optional/Recommended)
	// API keys are needed to integrate the backend.
	apiKeys, err := s.apiKeySvc.List(ctx)
	if err == nil && len(apiKeys) > 0 {
		issues = append(issues, ReadinessIssue{
			ID:         "api_key_created",
			Status:     ReadinessStateReady,
			ActionHref: "/api-keys",
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "api_key_created",
			Status:     ReadinessStateOptional,
			ActionHref: "/api-keys",
		})
	}

	// 10. Webhooks (Optional)
	issues = append(issues, ReadinessIssue{
		ID:         "webhooks_configured",
		Status:     ReadinessStateOptional,
		ActionHref: "/settings/webhooks",
	})

	state := ReadinessStateReady
	if !isSystemReady {
		state = ReadinessStateNotReady
	}

	c.JSON(http.StatusOK, ReadinessResponse{
		SystemState: state,
		Issues:      issues,
	})
}

func ptr(s string) *string {
	return &s
}

func ptrBool(b bool) *bool {
	return &b
}
