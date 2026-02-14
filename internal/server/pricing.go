package server

import (
	"strings"

	"github.com/gin-gonic/gin"
	pricedomain "github.com/railzwaylabs/railzway/internal/price/domain"
)

// @Summary      Create Pricing
// @Description  Create a new pricing
// @Tags         pricings
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Idempotency-Key  header  string  false  "Idempotency Key"
// @Param        request body pricedomain.CreateRequest true "Create Pricing Request"
// @Success      200  {object}  DataResponse
// @Router       /pricings [post]
func (s *Server) CreatePricing(c *gin.Context) {
	var req pricedomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.priceSvc.Create(c.Request.Context(), pricedomain.CreateRequest{
		Name:           strings.TrimSpace(req.Name),
		Description:    req.Description,
		Active:         req.Active,
		IdempotencyKey: idempotencyKeyFromHeader(c),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, resp)
}

// @Summary      List Pricings
// @Description  List available pricings
// @Tags         pricings
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        product_id  query  string  false  "Product ID"
// @Param        code        query  string  false  "Code"
// @Param        page_token  query  string  false  "Page Token"
// @Param        page_size   query  int     false  "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /pricings [get]
func (s *Server) ListPricings(c *gin.Context) {

	var query pricedomain.ListOptions
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.priceSvc.List(c.Request.Context(), query)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Prices, &resp.PageInfo)
}

// @Summary      Get Pricing
// @Description  Get pricing by ID
// @Tags         pricings
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Pricing ID"
// @Success      200  {object}  DataResponse
// @Router       /pricings/{id} [get]
func (s *Server) GetPricingByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	resp, err := s.priceSvc.Get(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, resp)
}

func isPricingValidationError(err error) bool {
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
