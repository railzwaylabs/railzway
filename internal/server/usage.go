package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

// @Summary      Ingest Usage
// @Description  Ingest usage event
// @Tags         usage
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        request body usagedomain.CreateIngestRequest true "Ingest Usage Request"
// @Success      200  {object}  usagedomain.UsageEvent
// @Router       /usage [post]
func (s *Server) IngestUsage(c *gin.Context) {

	var req usagedomain.CreateIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, err)
		return
	}
	if meterCode := strings.TrimSpace(req.MeterCode); meterCode != "" {
		c.Set("meter_code", meterCode)
	}

	usage, err := s.usagesvc.Ingest(c.Request.Context(), req)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, usage)
}

// @Summary      Get Usage Summary
// @Description  Get aggregated usage summary for a customer
// @Tags         usage
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        customer_id   query     string  true  "Customer ID"
// @Param        start         query     string  true  "Start Time (RFC3339)"
// @Param        end           query     string  true  "End Time (RFC3339)"
// @Success      200  {object}  map[string]float64
// @Router       /usage/summary [get]
func (s *Server) GetUsageSummary(c *gin.Context) {
	customerID := strings.TrimSpace(c.Query("customer_id"))
	if customerID == "" {
		AbortWithError(c, newValidationError("customer_id", "required", "customer_id is required"))
		return
	}

	startStr := c.Query("start")
	endStr := c.Query("end")

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		AbortWithError(c, newValidationError("start", "invalid_time", "start time must be RFC3339"))
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		AbortWithError(c, newValidationError("end", "invalid_time", "end time must be RFC3339"))
		return
	}

	summary, err := s.usagesvc.GetUsageSummary(c.Request.Context(), usagedomain.UsageSummaryRequest{
		CustomerID: customerID,
		Start:      start,
		End:        end,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}
