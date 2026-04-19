package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
)

type createProductRequest struct {
	Code           string                   `json:"code" binding:"required"`
	Name           string                   `json:"name" binding:"required"`
	Description    *string                  `json:"description,omitempty"`
	Active         *bool                    `json:"active,omitempty"`
	IdempotencyKey string                   `json:"idempotency_key" binding:"required"`
	FeatureIDs     []string                 `json:"feature_ids,omitempty"`
	Plans          []createProductPlanInput `json:"plans,omitempty"`
}

type createProductPlanInput struct {
	Code        string                        `json:"code" binding:"required"`
	Name        string                        `json:"name" binding:"required"`
	Description *string                       `json:"description,omitempty"`
	Active      *bool                         `json:"active,omitempty"`
	Prices      []createProductPlanPriceInput `json:"prices,omitempty"`
}

type createProductPlanPriceInput struct {
	Code                 string                         `json:"code" binding:"required"`
	Name                 string                         `json:"name"`
	Description          *string                        `json:"description,omitempty"`
	PriceType            string                         `json:"price_type" binding:"required,oneof=flat usage tiered"`
	BillingInterval      string                         `json:"billing_interval" binding:"required"`
	BillingIntervalCount int                            `json:"billing_interval_count"`
	AggregateUsage       *string                        `json:"aggregate_usage,omitempty"`
	BillingUnit          *string                        `json:"billing_unit,omitempty"`
	MeterID              *string                        `json:"meter_id,omitempty"`
	MeterCode            *string                        `json:"meter_code,omitempty"`
	Active               *bool                          `json:"active,omitempty"`
	Amounts              []createProductPlanAmountInput `json:"amounts,omitempty"`
	Tiers                []createProductPlanTierInput   `json:"tiers,omitempty"`
}

type createProductPlanAmountInput struct {
	Currency           string `json:"currency" binding:"required"`
	UnitAmountCents    int64  `json:"unit_amount_cents"`
	MinimumAmountCents *int64 `json:"minimum_amount_cents,omitempty"`
	MaximumAmountCents *int64 `json:"maximum_amount_cents,omitempty"`
	EffectiveFrom      string `json:"effective_from,omitempty"`
	EffectiveTo        string `json:"effective_to,omitempty"`
}

type createProductPlanTierInput struct {
	TierMode        string   `json:"tier_mode" binding:"required"`
	StartQuantity   float64  `json:"start_quantity"`
	EndQuantity     *float64 `json:"end_quantity,omitempty"`
	UnitAmountCents *int64   `json:"unit_amount_cents,omitempty"`
	FlatAmountCents *int64   `json:"flat_amount_cents,omitempty"`
	Unit            string   `json:"unit" binding:"required"`
}

func (h *Handler) CreateProduct(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createProductRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	plans, err := toCreateProductPlanInputs(payload.Plans)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.products.Create(ctx, productdomain.CreateProductRequest{
		Code:           strings.TrimSpace(payload.Code),
		Name:           strings.TrimSpace(payload.Name),
		Description:    payload.Description,
		Active:         payload.Active,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
		FeatureIDs:     payload.FeatureIDs,
		Plans:          plans,
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func toCreateProductPlanInputs(inputs []createProductPlanInput) ([]productdomain.CreateProductPlanInput, error) {
	plans := make([]productdomain.CreateProductPlanInput, 0, len(inputs))
	for _, plan := range inputs {
		prices := make([]productdomain.CreateProductPlanPriceInput, 0, len(plan.Prices))
		for _, price := range plan.Prices {
			amounts := make([]productdomain.CreateProductPlanAmountInput, 0, len(price.Amounts))
			for _, amount := range price.Amounts {
				effectiveFrom, err := parseTimePtr(amount.EffectiveFrom)
				if err != nil {
					return nil, errors.New("invalid_effective_from")
				}
				effectiveTo, err := parseTimePtr(amount.EffectiveTo)
				if err != nil {
					return nil, errors.New("invalid_effective_to")
				}
				amounts = append(amounts, productdomain.CreateProductPlanAmountInput{
					Currency:           strings.TrimSpace(amount.Currency),
					UnitAmountCents:    amount.UnitAmountCents,
					MinimumAmountCents: amount.MinimumAmountCents,
					MaximumAmountCents: amount.MaximumAmountCents,
					EffectiveFrom:      effectiveFrom,
					EffectiveTo:        effectiveTo,
				})
			}

			tiers := make([]productdomain.CreateProductPlanTierInput, 0, len(price.Tiers))
			for _, tier := range price.Tiers {
				tiers = append(tiers, productdomain.CreateProductPlanTierInput{
					TierMode:        strings.TrimSpace(tier.TierMode),
					StartQuantity:   tier.StartQuantity,
					EndQuantity:     tier.EndQuantity,
					UnitAmountCents: tier.UnitAmountCents,
					FlatAmountCents: tier.FlatAmountCents,
					Unit:            strings.TrimSpace(tier.Unit),
				})
			}

			prices = append(prices, productdomain.CreateProductPlanPriceInput{
				Code:                 strings.TrimSpace(price.Code),
				Name:                 strings.TrimSpace(price.Name),
				Description:          trimOptionalStringPtr(price.Description),
				PriceType:            strings.TrimSpace(price.PriceType),
				BillingInterval:      strings.TrimSpace(price.BillingInterval),
				BillingIntervalCount: price.BillingIntervalCount,
				AggregateUsage:       trimOptionalStringPtr(price.AggregateUsage),
				BillingUnit:          trimOptionalStringPtr(price.BillingUnit),
				MeterID:              trimOptionalStringPtr(price.MeterID),
				MeterCode:            trimOptionalStringPtr(price.MeterCode),
				Active:               price.Active,
				Amounts:              amounts,
				Tiers:                tiers,
			})
		}

		plans = append(plans, productdomain.CreateProductPlanInput{
			Code:        strings.TrimSpace(plan.Code),
			Name:        strings.TrimSpace(plan.Name),
			Description: trimOptionalStringPtr(plan.Description),
			Active:      plan.Active,
			Prices:      prices,
		})
	}

	return plans, nil
}

func trimOptionalStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

type updateProductRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Active      *bool     `json:"active,omitempty"`
	FeatureIDs  *[]string `json:"feature_ids,omitempty"`
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	productID := strings.TrimSpace(c.Param("product_id"))
	var payload updateProductRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.products.Update(ctx, productID, productdomain.UpdateProductRequest{
		Name:        payload.Name,
		Description: payload.Description,
		Active:      payload.Active,
		FeatureIDs:  payload.FeatureIDs,
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetProduct(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	productID := strings.TrimSpace(c.Param("product_id"))
	expand := strings.TrimSpace(c.Query("expand"))
	resp, err := h.products.GetByID(ctx, productdomain.GetProductRequest{
		ID:             productID,
		ExpandPlans:    strings.Contains(expand, "plans"),
		ExpandFeatures: strings.Contains(expand, "features"),
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListProducts(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	active, err := parseBoolPtr(c.Query("active"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_active"})
		return
	}

	expand := strings.TrimSpace(c.Query("expand"))
	resp, err := h.products.List(ctx, productdomain.ListProductRequest{
		PageToken:      c.Query("page_token"),
		PageSize:       parseInt32(c.Query("page_size")),
		Code:           c.Query("code"),
		Name:           c.Query("name"),
		Active:         active,
		ExpandPlans:    strings.Contains(expand, "plans"),
		ExpandFeatures: strings.Contains(expand, "features"),
	})
	if err != nil {
		writeProductError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
