package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	auditdomain "github.com/railzwaylabs/railzway/internal/audit/domain"
)

// ExportAuditLogs handles GET /api/v1/audit/export
func (s *Server) ExportAuditLogs(c *gin.Context) {
	// Parse query parameters
	startDateStr := strings.TrimSpace(c.Query("start_date"))
	endDateStr := strings.TrimSpace(c.Query("end_date"))
	formatStr := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv")))
	orgIDStr := strings.TrimSpace(c.Query("org_id"))
	actionsStr := strings.TrimSpace(c.Query("actions"))

	// Validate required parameters
	if startDateStr == "" || endDateStr == "" {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	// End date should be exclusive (end of day)
	endDate = endDate.Add(24 * time.Hour)

	// Validate date range
	if endDate.Before(startDate) {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	// Limit export to 90 days
	if endDate.Sub(startDate) > 90*24*time.Hour {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	// Parse format
	var format auditdomain.ExportFormat
	switch formatStr {
	case "csv":
		format = auditdomain.ExportFormatCSV
	case "json":
		format = auditdomain.ExportFormatJSON
	default:
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	// Parse org_id (optional)
	var orgID *snowflake.ID
	if orgIDStr != "" {
		id, err := snowflake.ParseString(orgIDStr)
		if err != nil {
			AbortWithError(c, ErrInvalidRequest)
			return
		}
		orgID = &id
	}

	// Parse actions filter (optional)
	var actions []string
	if actionsStr != "" {
		actions = strings.Split(actionsStr, ",")
		for i := range actions {
			actions[i] = strings.TrimSpace(actions[i])
		}
	}

	// Authorization check - skip for now as this is admin-only route
	// Authorization is handled by RequireRole middleware

	// Execute export
	result, err := s.auditExportSvc.Export(c.Request.Context(), auditdomain.ExportRequest{
		OrgID:     orgID,
		StartDate: startDate,
		EndDate:   endDate,
		Format:    format,
		Actions:   actions,
	})
	if err != nil {
		AbortWithError(c, ErrInternal)
		return
	}

	// Set response headers
	c.Header("X-Audit-Export-Checksum", result.Checksum)
	c.Header("X-Audit-Export-Count", strconv.Itoa(result.Count))

	// Set content type and filename
	var contentType, filename string
	switch result.Format {
	case auditdomain.ExportFormatCSV:
		contentType = "text/csv"
		filename = "audit_export_" + startDateStr + "_" + endDateStr + ".csv"
	case auditdomain.ExportFormatJSON:
		contentType = "application/json"
		filename = "audit_export_" + startDateStr + "_" + endDateStr + ".json"
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, contentType, result.Data)
}
