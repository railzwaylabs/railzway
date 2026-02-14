package server

import (
	"strings"

	"github.com/gin-gonic/gin"
	priceamountdomain "github.com/railzwaylabs/railzway/internal/priceamount/domain"
)

// @Summary      Create Price Amount
// @Description  Create a new price amount
// @Tags         price_amounts
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Idempotency-Key  header  string  false  "Idempotency Key"
// @Param        request body priceamountdomain.CreateRequest true "Create Price Amount Request"
// @Success      200  {object}  DataResponse
// @Router       /price_amounts [post]
func (s *Server) CreatePriceAmount(c *gin.Context) {
	var req priceamountdomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}
	req.IdempotencyKey = idempotencyKeyFromHeader(c)

	resp, err := s.priceAmountSvc.Create(c.Request.Context(), req)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if s.auditSvc != nil {
		targetID := resp.ID.String()
		metadata := map[string]any{
			"price_amount_id":   resp.ID,
			"price_id":          resp.PriceID,
			"currency":          resp.Currency,
			"unit_amount_cents": resp.UnitAmountCents,
		}
		if resp.MinimumAmountCents != nil {
			metadata["minimum_amount_cents"] = *resp.MinimumAmountCents
		}
		if resp.MaximumAmountCents != nil {
			metadata["maximum_amount_cents"] = *resp.MaximumAmountCents
		}
		_ = s.auditSvc.AuditLog(c.Request.Context(), nil, "", nil, "price_amount.create", "price_amount", &targetID, metadata)
	}

	respondData(c, resp)
}

// @Summary      List Price Amounts
// @Description  List available price amounts
// @Tags         price_amounts
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        price_id query string false "Price ID"
// @Param        currency query string false "Currency (ISO-4217)"
// @Param        effective_from query string false "Effective From (RFC3339)"
// @Param        effective_to query string false "Effective To (RFC3339)"
// @Param        page_token query string false "Page Token"
// @Param        page_size query int false "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /price_amounts [get]
func (s *Server) ListPriceAmounts(c *gin.Context) {

	var req priceamountdomain.ListPriceAmountRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.priceAmountSvc.List(c.Request.Context(), req)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Amounts, &resp.PageInfo)
}

// @Summary      Get Price Amount
// @Description  Get price amount by ID
// @Tags         price_amounts
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Price Amount ID"
// @Success      200  {object}  DataResponse
// @Router       /price_amounts/{id} [get]
func (s *Server) GetPriceAmountByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	resp, err := s.priceAmountSvc.Get(c.Request.Context(), priceamountdomain.GetPriceAmountByID{
		ID: id,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, resp)
}

func isPriceAmountValidationError(err error) bool {
	switch err {
	case priceamountdomain.ErrInvalidOrganization,
		priceamountdomain.ErrInvalidPrice,
		priceamountdomain.ErrInvalidCurrency,
		priceamountdomain.ErrInvalidUnitAmount,
		priceamountdomain.ErrInvalidMinAmount,
		priceamountdomain.ErrInvalidMaxAmount,
		priceamountdomain.ErrInvalidMeterID,
		priceamountdomain.ErrInvalidEffectiveFrom,
		priceamountdomain.ErrInvalidEffectiveTo,
		priceamountdomain.ErrEffectiveOverlap,
		priceamountdomain.ErrEffectiveGap,
		priceamountdomain.ErrInvalidID:
		return true
	default:
		return false
	}
}
