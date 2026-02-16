package server

import (
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

func (s *Server) ListBillingCustomers(c *gin.Context) {
	if s.billingDashboardSvc == nil {
		AbortWithError(c, ErrServiceUnavailable)
		return
	}

	var query pagination.Pagination
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.billingDashboardSvc.ListCustomerBalances(c.Request.Context(), query.PageToken, int32(query.PageSize))
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Customers, &resp.PageInfo)
}

func (s *Server) ListBillingCycles(c *gin.Context) {
	if s.billingDashboardSvc == nil {
		AbortWithError(c, ErrServiceUnavailable)
		return
	}

	var query pagination.Pagination
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.billingDashboardSvc.ListBillingCycles(c.Request.Context(), query.PageToken, int32(query.PageSize))
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Cycles, &resp.PageInfo)
}

func (s *Server) ListBillingActivity(c *gin.Context) {
	if s.billingDashboardSvc == nil {
		AbortWithError(c, ErrServiceUnavailable)
		return
	}

	var query pagination.Pagination
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	resp, err := s.billingDashboardSvc.ListBillingActivity(c.Request.Context(), query.PageToken, int32(query.PageSize))
	if err != nil {
		AbortWithError(c, err)
		return
	}

	respondList(c, resp.Activity, &resp.PageInfo)
}
