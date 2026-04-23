package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
)

type createPlanRequest struct {
	Code           string                 `json:"code"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Active         *bool                  `json:"active"`
	ProductID      *string                `json:"product_id"`
	IdempotencyKey string                 `json:"idempotency_key"`
	Prices         []createPlanPriceInput `json:"prices"`
}

type createPlanPriceInput struct {
	Code                 string                    `json:"code"`
	Name                 string                    `json:"name"`
	Description          string                    `json:"description"`
	PriceType            string                    `json:"price_type"`
	BillingInterval      string                    `json:"billing_interval"`
	BillingIntervalCount int                       `json:"billing_interval_count"`
	AggregateUsage       string                    `json:"aggregate_usage"`
	BillingUnit          string                    `json:"billing_unit"`
	MeterID              *string                   `json:"meter_id"`
	MeterCode            string                    `json:"meter_code"`
	Active               *bool                     `json:"active"`
	IdempotencyKey       string                    `json:"idempotency_key"`
	Amounts              []createPlanAmountRequest `json:"amounts"`
	Tiers                []createPlanTierRequest   `json:"tiers"`
}

func (h *Handler) CreatePlan(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload createPlanRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	if len(payload.Prices) > 0 {
		planRequest := plandomain.CreatePlanRequest{
			Code:           strings.TrimSpace(payload.Code),
			Name:           strings.TrimSpace(payload.Name),
			Description:    strings.TrimSpace(payload.Description),
			Active:         payload.Active,
			ProductID:      payload.ProductID,
			IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
			Prices:         make([]plandomain.CreatePlanPriceInput, 0, len(payload.Prices)),
		}

		for _, price := range payload.Prices {
			priceReq := plandomain.CreatePlanPriceInput{
				Code:                 strings.TrimSpace(price.Code),
				Name:                 strings.TrimSpace(price.Name),
				Description:          strings.TrimSpace(price.Description),
				PriceType:            strings.TrimSpace(price.PriceType),
				BillingInterval:      strings.TrimSpace(price.BillingInterval),
				BillingIntervalCount: price.BillingIntervalCount,
				AggregateUsage:       strings.TrimSpace(price.AggregateUsage),
				BillingUnit:          strings.TrimSpace(price.BillingUnit),
				MeterID:              price.MeterID,
				MeterCode:            strings.TrimSpace(price.MeterCode),
				Active:               price.Active,
				IdempotencyKey:       strings.TrimSpace(price.IdempotencyKey),
				Amounts:              make([]plandomain.CreatePlanAmountInput, 0, len(price.Amounts)),
				Tiers:                make([]plandomain.CreatePlanTierInput, 0, len(price.Tiers)),
			}

			for _, amount := range price.Amounts {
				effectiveFrom, err := parseTimePtr(amount.EffectiveFrom)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_effective_from"})
					return
				}

				effectiveTo, err := parseTimePtr(amount.EffectiveTo)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_effective_to"})
					return
				}

				priceReq.Amounts = append(priceReq.Amounts, plandomain.CreatePlanAmountInput{
					Currency:           strings.TrimSpace(amount.Currency),
					UnitAmountCents:    amount.UnitAmountCents,
					MinimumAmountCents: amount.MinimumAmountCents,
					MaximumAmountCents: amount.MaximumAmountCents,
					EffectiveFrom:      effectiveFrom,
					EffectiveTo:        effectiveTo,
					IdempotencyKey:     strings.TrimSpace(amount.IdempotencyKey),
				})
			}

			for _, tier := range price.Tiers {
				priceReq.Tiers = append(priceReq.Tiers, plandomain.CreatePlanTierInput{
					TierMode:        strings.TrimSpace(tier.TierMode),
					StartQuantity:   tier.StartQuantity,
					EndQuantity:     tier.EndQuantity,
					UnitAmountCents: tier.UnitAmountCents,
					FlatAmountCents: tier.FlatAmountCents,
					Unit:            strings.TrimSpace(tier.Unit),
					IdempotencyKey:  strings.TrimSpace(tier.IdempotencyKey),
				})
			}

			planRequest.Prices = append(planRequest.Prices, priceReq)
		}

		resp, err := h.plans.CreatePlan(ctx, planRequest)
		if err != nil {
			writePlanError(c, err)
			return
		}

		c.JSON(http.StatusOK, resp)
		return
	}

	resp, err := h.plans.CreatePlan(ctx, plandomain.CreatePlanRequest{
		Code:           strings.TrimSpace(payload.Code),
		Name:           strings.TrimSpace(payload.Name),
		Description:    strings.TrimSpace(payload.Description),
		Active:         payload.Active,
		ProductID:      payload.ProductID,
		IdempotencyKey: strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writePlanError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

type createPlanPriceRequest struct {
	Code                 string  `json:"code"`
	Name                 string  `json:"name"`
	Description          string  `json:"description"`
	PriceType            string  `json:"price_type"`
	BillingInterval      string  `json:"billing_interval"`
	BillingIntervalCount int     `json:"billing_interval_count"`
	AggregateUsage       string  `json:"aggregate_usage"`
	BillingUnit          string  `json:"billing_unit"`
	MeterID              *string `json:"meter_id"`
	MeterCode            string  `json:"meter_code"`
	Active               *bool   `json:"active"`
	IdempotencyKey       string  `json:"idempotency_key"`
}

func (h *Handler) CreatePlanPrice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	planID := strings.TrimSpace(c.Param("plan_id"))
	var payload createPlanPriceRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.plans.CreatePlanPrice(ctx, plandomain.CreatePlanPriceRequest{
		PlanID:               planID,
		MeterID:              payload.MeterID,
		Code:                 strings.TrimSpace(payload.Code),
		Name:                 strings.TrimSpace(payload.Name),
		Description:          strings.TrimSpace(payload.Description),
		PriceType:            strings.TrimSpace(payload.PriceType),
		BillingInterval:      strings.TrimSpace(payload.BillingInterval),
		BillingIntervalCount: payload.BillingIntervalCount,
		AggregateUsage:       strings.TrimSpace(payload.AggregateUsage),
		BillingUnit:          strings.TrimSpace(payload.BillingUnit),
		MeterCode:            strings.TrimSpace(payload.MeterCode),
		Active:               payload.Active,
		IdempotencyKey:       strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type createPlanAmountRequest struct {
	Currency           string   `json:"currency"`
	UnitAmountCents    float64  `json:"unit_amount_cents"`
	MinimumAmountCents *float64 `json:"minimum_amount_cents"`
	MaximumAmountCents *float64 `json:"maximum_amount_cents"`
	EffectiveFrom      string   `json:"effective_from"`
	EffectiveTo        string   `json:"effective_to"`
	IdempotencyKey     string   `json:"idempotency_key"`
}

func (h *Handler) CreatePlanAmount(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	priceID := strings.TrimSpace(c.Param("price_id"))
	var payload createPlanAmountRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	effectiveFrom, err := parseTimePtr(payload.EffectiveFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_effective_from"})
		return
	}
	effectiveTo, err := parseTimePtr(payload.EffectiveTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_effective_to"})
		return
	}

	resp, err := h.plans.CreatePlanAmount(ctx, plandomain.CreatePlanAmountRequest{
		PlanPriceID:        priceID,
		Currency:           strings.TrimSpace(payload.Currency),
		UnitAmountCents:    payload.UnitAmountCents,
		MinimumAmountCents: payload.MinimumAmountCents,
		MaximumAmountCents: payload.MaximumAmountCents,
		EffectiveFrom:      effectiveFrom,
		EffectiveTo:        effectiveTo,
		IdempotencyKey:     strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type createPlanTierRequest struct {
	TierMode        string   `json:"tier_mode"`
	StartQuantity   float64  `json:"start_quantity"`
	EndQuantity     *float64 `json:"end_quantity"`
	UnitAmountCents *float64 `json:"unit_amount_cents"`
	FlatAmountCents *float64 `json:"flat_amount_cents"`
	Unit            string   `json:"unit"`
	IdempotencyKey  string   `json:"idempotency_key"`
}

func (h *Handler) CreatePlanTier(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	priceID := strings.TrimSpace(c.Param("price_id"))
	var payload createPlanTierRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.plans.CreatePlanTier(ctx, plandomain.CreatePlanTierRequest{
		PlanPriceID:     priceID,
		TierMode:        strings.TrimSpace(payload.TierMode),
		StartQuantity:   payload.StartQuantity,
		EndQuantity:     payload.EndQuantity,
		UnitAmountCents: payload.UnitAmountCents,
		FlatAmountCents: payload.FlatAmountCents,
		Unit:            strings.TrimSpace(payload.Unit),
		IdempotencyKey:  strings.TrimSpace(payload.IdempotencyKey),
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

type updatePlanRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
	ProductID   *string `json:"product_id"`
}

func (h *Handler) UpdatePlan(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	planID := strings.TrimSpace(c.Param("plan_id"))
	var payload updatePlanRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}

	resp, err := h.plans.UpdatePlan(ctx, planID, plandomain.UpdatePlanRequest{
		Name:        payload.Name,
		Description: payload.Description,
		Active:      payload.Active,
		ProductID:   payload.ProductID,
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPlan(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	planID := strings.TrimSpace(c.Param("plan_id"))
	resp, err := h.plans.GetPlanByID(ctx, plandomain.GetPlanRequest{ID: planID})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListPlans(c *gin.Context) {
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

	resp, err := h.plans.ListPlans(ctx, plandomain.ListPlanRequest{
		PageToken: c.Query("page_token"),
		PageSize:  parseInt32(c.Query("page_size")),
		Code:      c.Query("code"),
		Name:      c.Query("name"),
		Active:    active,
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPlanPrice(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	priceID := strings.TrimSpace(c.Param("price_id"))
	resp, err := h.plans.GetPlanPriceByID(ctx, plandomain.GetPlanPriceRequest{ID: priceID})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListPlanPrices(c *gin.Context) {
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

	resp, err := h.plans.ListPlanPrices(ctx, plandomain.ListPlanPriceRequest{
		PageToken:       c.Query("page_token"),
		PageSize:        parseInt32(c.Query("page_size")),
		PlanID:          strings.TrimSpace(c.Param("plan_id")),
		PriceType:       c.Query("price_type"),
		Active:          active,
		BillingInterval: c.Query("billing_interval"),
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPlanAmount(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	amountID := strings.TrimSpace(c.Param("amount_id"))
	resp, err := h.plans.GetPlanAmountByID(ctx, plandomain.GetPlanAmountRequest{ID: amountID})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListPlanAmounts(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.plans.ListPlanAmounts(ctx, plandomain.ListPlanAmountRequest{
		PageToken:   c.Query("page_token"),
		PageSize:    parseInt32(c.Query("page_size")),
		PlanPriceID: strings.TrimSpace(c.Param("price_id")),
		Currency:    c.Query("currency"),
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPlanTier(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	tierID := strings.TrimSpace(c.Param("tier_id"))
	resp, err := h.plans.GetPlanTierByID(ctx, plandomain.GetPlanTierRequest{ID: tierID})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListPlanTiers(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)

	resp, err := h.plans.ListPlanTiers(ctx, plandomain.ListPlanTierRequest{
		PageToken:   c.Query("page_token"),
		PageSize:    parseInt32(c.Query("page_size")),
		PlanPriceID: strings.TrimSpace(c.Param("price_id")),
		TierMode:    c.Query("tier_mode"),
	})
	if err != nil {
		writePlanError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
