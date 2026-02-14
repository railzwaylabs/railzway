package server

import (
	"strings"

	"github.com/gin-gonic/gin"
	pricetierdomain "github.com/railzwaylabs/railzway/internal/pricetier/domain"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

// @Summary      List Price Tiers
// @Description  List available price tiers
// @Tags         price_tiers
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        page_token query string false "Page Token"
// @Param        page_size query int false "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /price_tiers [get]
func (s *Server) ListPriceTiers(c *gin.Context) {
	var query struct {
		pagination.Pagination
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.priceTierSvc.List(c.Request.Context(), pricetierdomain.ListRequest{
		PageToken: query.PageToken,
		PageSize:  int32(query.PageSize),
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Tiers, &resp.PageInfo)
}

// @Summary      Get Price Tier
// @Description  Get price tier by ID
// @Tags         price_tiers
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Price Tier ID"
// @Success      200  {object}  DataResponse
// @Router       /price_tiers/{id} [get]
func (s *Server) GetPriceTierByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	resp, err := s.priceTierSvc.Get(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, resp)
}

// @Summary      Create Price Tier
// @Description  Create a new price tier
// @Tags         price_tiers
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Idempotency-Key  header  string  false  "Idempotency Key"
// @Param        request body pricetierdomain.CreateRequest true "Create Price Tier Request"
// @Success      200  {object}  DataResponse
// @Router       /price_tiers [post]
func (s *Server) CreatePriceTier(c *gin.Context) {
	var req pricetierdomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	req.PriceID = strings.TrimSpace(req.PriceID)
	req.Unit = strings.TrimSpace(req.Unit)
	req.IdempotencyKey = idempotencyKeyFromHeader(c)

	resp, err := s.priceTierSvc.Create(c.Request.Context(), req)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondData(c, resp)
}

func isPriceTierValidationError(err error) bool {
	switch err {
	case pricetierdomain.ErrInvalidOrganization,
		pricetierdomain.ErrInvalidPrice,
		pricetierdomain.ErrInvalidTierMode,
		pricetierdomain.ErrInvalidStartQty,
		pricetierdomain.ErrInvalidEndQty,
		pricetierdomain.ErrInvalidUnitAmount,
		pricetierdomain.ErrInvalidFlatAmount,
		pricetierdomain.ErrInvalidUnit,
		pricetierdomain.ErrInvalidID:
		return true
	default:
		return false
	}
}
