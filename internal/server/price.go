package server

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	pricedomain "github.com/railzwaylabs/railzway/internal/price/domain"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

type createPriceRequest struct {
	ProductID            string                      `json:"product_id"`
	Code                 string                      `json:"code"`
	LookupKey            string                      `json:"lookup_key"`
	Name                 string                      `json:"name"`
	Description          string                      `json:"description"`
	PricingModel         pricedomain.PricingModel    `json:"pricing_model"`
	BillingMode          pricedomain.BillingMode     `json:"billing_mode"`
	BillingInterval      pricedomain.BillingInterval `json:"billing_interval"`
	BillingIntervalCount int32                       `json:"billing_interval_count"`
	AggregateUsage       *pricedomain.AggregateUsage `json:"aggregate_usage"`
	BillingUnit          *pricedomain.BillingUnit    `json:"billing_unit"`
	BillingThreshold     *float64                    `json:"billing_threshold"`
	TaxBehavior          pricedomain.TaxBehavior     `json:"tax_behavior"`
	TaxCode              *string                     `json:"tax_code"`
	Version              *int32                      `json:"version"`
	IsDefault            *bool                       `json:"is_default"`
	Active               *bool                       `json:"active"`
	RetiredAt            *time.Time                  `json:"retired_at"`
	Metadata             map[string]any              `json:"metadata"`
}

// @Summary      Create Price
// @Description  Create a new price
// @Tags         prices
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Idempotency-Key  header  string  false  "Idempotency Key"
// @Param        request body createPriceRequest true "Create Price Request"
// @Success      200  {object}  DataResponse
// @Router       /prices [post]
func (s *Server) CreatePrice(c *gin.Context) {
	var req createPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.priceSvc.Create(c.Request.Context(), pricedomain.CreateRequest{
		ProductID:            strings.TrimSpace(req.ProductID),
		Code:                 strings.TrimSpace(req.Code),
		LookupKey:            req.LookupKey,
		Name:                 req.Name,
		Description:          req.Description,
		PricingModel:         req.PricingModel,
		BillingMode:          req.BillingMode,
		BillingInterval:      req.BillingInterval,
		BillingIntervalCount: req.BillingIntervalCount,
		AggregateUsage:       req.AggregateUsage,
		BillingUnit:          req.BillingUnit,
		BillingThreshold:     req.BillingThreshold,
		TaxBehavior:          req.TaxBehavior,
		TaxCode:              req.TaxCode,
		Version:              req.Version,
		IsDefault:            req.IsDefault,
		Active:               req.Active,
		RetiredAt:            req.RetiredAt,
		Metadata:             req.Metadata,
		IdempotencyKey:       idempotencyKeyFromHeader(c),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID.String()
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "price.create", "price", &targetID, map[string]any{
			"price_id":      resp.ID,
			"product_id":    resp.ProductID,
			"code":          resp.Code,
			"pricing_model": resp.PricingModel,
			"billing_mode":  resp.BillingMode,
			"active":        resp.Active,
		})
	}

	respondData(c, resp)
}

// @Summary      List Prices
// @Description  List available prices
// @Tags         prices
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        product_id  query  string  false  "Product ID"
// @Param        code        query  string  false  "Code"
// @Param        page_token  query  string  false  "Page Token"
// @Param        page_size   query  int     false  "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /prices [get]
func (s *Server) ListPrices(c *gin.Context) {
	var query struct {
		pagination.Pagination
		Code      string `form:"code"`
		ProductID string `form:"product_id"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	opts := pricedomain.ListOptions{
		ProductID: strings.TrimSpace(query.ProductID),
		PageToken: query.PageToken,
		PageSize:  int32(query.PageSize),
	}
	if code := strings.TrimSpace(query.Code); code != "" {
		opts.Code = &code
	}

	resp, err := s.priceSvc.List(c.Request.Context(), opts)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Prices, &resp.PageInfo)
}

// @Summary      Get Price
// @Description  Get price by ID
// @Tags         prices
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Price ID"
// @Success      200  {object}  DataResponse
// @Router       /prices/{id} [get]
func (s *Server) GetPriceByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	resp, err := s.priceSvc.Get(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, resp)
}

func isPriceValidationError(err error) bool {
	switch err {
	case pricedomain.ErrInvalidOrganization,
		pricedomain.ErrInvalidProduct,
		pricedomain.ErrInvalidCode,
		pricedomain.ErrInvalidPricingModel,
		pricedomain.ErrInvalidBillingMode,
		pricedomain.ErrInvalidBillingInterval,
		pricedomain.ErrInvalidBillingIntervalCount,
		pricedomain.ErrInvalidAggregateUsage,
		pricedomain.ErrInvalidBillingUnit,
		pricedomain.ErrInvalidBillingThreshold,
		pricedomain.ErrInvalidTaxBehavior,
		pricedomain.ErrInvalidVersion,
		pricedomain.ErrInvalidID:
		return true
	default:
		return false
	}
}
