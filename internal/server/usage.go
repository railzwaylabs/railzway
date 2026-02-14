package server

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/datatypes"
)

type usageEventResponse struct {
	ID                 string            `json:"id"`
	CustomerID         string            `json:"customer_id"`
	SubscriptionID     string            `json:"subscription_id,omitempty"`
	SubscriptionItemID string            `json:"subscription_item_id,omitempty"`
	MeterID            string            `json:"meter_id,omitempty"`
	MeterCode          string            `json:"meter_code"`
	Value              float64           `json:"value"`
	RecordedAt         time.Time         `json:"recorded_at"`
	Status             string            `json:"status"`
	Error              *string           `json:"error,omitempty"`
	IdempotencyKey     string            `json:"idempotency_key,omitempty"`
	Metadata           datatypes.JSONMap `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
}

// @Summary      Ingest Usage
// @Description  Ingest usage event
// @Tags         usage
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        request body usagedomain.CreateIngestRequest true "Ingest Usage Request"
// @Success      200  {object}  DataResponse
// @Router       /usage [post]
func (s *Server) IngestUsage(c *gin.Context) {

	var req usagedomain.CreateIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
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

	respondData(c, usage)
}

// @Summary      List Usage
// @Description  List usage events
// @Tags         usage
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        customer_id      query     string  false  "Customer ID"
// @Param        subscription_id  query     string  false  "Subscription ID"
// @Param        meter_id         query     string  false  "Meter ID"
// @Param        meter_code       query     string  false  "Meter Code"
// @Param        status           query     string  false  "Usage Status"
// @Param        recorded_from    query     string  false  "Recorded From (RFC3339 or YYYY-MM-DD)"
// @Param        recorded_to      query     string  false  "Recorded To (RFC3339 or YYYY-MM-DD)"
// @Param        page_token       query     string  false  "Page Token"
// @Param        page_size        query     int     false  "Page Size"
// @Success      200  {object}  ListResponse
// @Router       /usage [get]
func (s *Server) ListUsage(c *gin.Context) {
	var query struct {
		pagination.Pagination
		CustomerID     string `form:"customer_id"`
		SubscriptionID string `form:"subscription_id"`
		MeterID        string `form:"meter_id"`
		MeterCode      string `form:"meter_code"`
		Status         string `form:"status"`
		RecordedFrom   string `form:"recorded_from"`
		RecordedTo     string `form:"recorded_to"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	recordedFrom, err := parseOptionalTime(query.RecordedFrom, false)
	if err != nil {
		AbortWithError(c, newValidationError("recorded_from", "invalid_recorded_from", "invalid recorded_from"))
		return
	}

	recordedTo, err := parseOptionalTime(query.RecordedTo, true)
	if err != nil {
		AbortWithError(c, newValidationError("recorded_to", "invalid_recorded_to", "invalid recorded_to"))
		return
	}
	if recordedFrom != nil && recordedTo != nil && recordedFrom.After(*recordedTo) {
		AbortWithError(c, newValidationError("recorded_to", "invalid_recorded_range", "recorded_from must be before recorded_to"))
		return
	}

	status := strings.ToLower(strings.TrimSpace(query.Status))
	if status != "" && !isUsageStatusValid(status) {
		AbortWithError(c, newValidationError("status", "invalid_status", "invalid status"))
		return
	}

	resp, err := s.usagesvc.List(c.Request.Context(), usagedomain.ListUsageRequest{
		CustomerID:     strings.TrimSpace(query.CustomerID),
		SubscriptionID: strings.TrimSpace(query.SubscriptionID),
		MeterID:        strings.TrimSpace(query.MeterID),
		MeterCode:      strings.TrimSpace(query.MeterCode),
		Status:         status,
		PageToken:      query.PageToken,
		PageSize:       int32(query.PageSize),
		RecordedFrom:   recordedFrom,
		RecordedTo:     recordedTo,
	})
	if err != nil {
		AbortWithError(c, err)
		return
	}

	events := make([]usageEventResponse, 0, len(resp.UsageEvents))
	for _, item := range resp.UsageEvents {
		events = append(events, toUsageEventResponse(item))
	}

	respondList(c, events, &resp.PageInfo)
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
// @Success      200  {object}  DataResponse
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

	respondData(c, summary)
}

func toUsageEventResponse(item usagedomain.UsageEvent) usageEventResponse {
	resp := usageEventResponse{
		MeterCode:      item.MeterCode,
		Value:          item.Value,
		RecordedAt:     item.RecordedAt,
		Status:         item.Status,
		Error:          item.Error,
		IdempotencyKey: item.IdempotencyKey,
		Metadata:       item.Metadata,
		CreatedAt:      item.CreatedAt,
	}

	if item.ID != 0 {
		resp.ID = item.ID.String()
	}
	if item.CustomerID != 0 {
		resp.CustomerID = item.CustomerID.String()
	}
	if item.SubscriptionID != 0 {
		resp.SubscriptionID = item.SubscriptionID.String()
	}
	if item.SubscriptionItemID != 0 {
		resp.SubscriptionItemID = item.SubscriptionItemID.String()
	}
	if item.MeterID != 0 {
		resp.MeterID = item.MeterID.String()
	}

	return resp
}

func isUsageStatusValid(value string) bool {
	switch value {
	case usagedomain.UsageStatusAccepted,
		usagedomain.UsageStatusInvalid,
		usagedomain.UsageStatusEnriched,
		usagedomain.UsageStatusRated,
		usagedomain.UsageStatusUnmatchedMeter,
		usagedomain.UsageStatusUnmatchedSubscription:
		return true
	default:
		return false
	}
}
