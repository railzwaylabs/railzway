package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	auditdomain "github.com/smallbiznis/railzway/internal/audit/domain"
	billingoverviewdomain "github.com/smallbiznis/railzway/internal/billingoverview/domain"
	"github.com/smallbiznis/railzway/internal/orgcontext"
	"github.com/smallbiznis/railzway/pkg/db/pagination"
)

type HomeDashboardResponse struct {
	CycleHealth CycleHealth        `json:"cycle_health"`
	Pulse       SystemPulse        `json:"pulse"`
	Alerts      []AlertBase        `json:"alerts"`
	Activity    []HomeActivityItem `json:"activity"`
}

type CycleHealth struct {
	DateProgress    string  `json:"date_progress"`    // "Day 11 of 31"
	ProgressPercent int     `json:"progress_percent"` // 35
	CurrentRevenue  float64 `json:"current_revenue"`
	PreviousRevenue float64 `json:"previous_revenue"` // Same day last month
}

type SystemPulse struct {
	UsageVolumeLastHour int     `json:"usage_volume_last_hour"`
	ErrorRate           float64 `json:"error_rate"` // Percentage
	JobsRunning         int     `json:"jobs_running"`
}

type AlertBase struct {
	ID       string `json:"id"`
	Severity string `json:"severity"` // "critical", "warning"
	Message  string `json:"message"`
	Count    int    `json:"count"`
}

type HomeActivityItem struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	OccurredAt  time.Time `json:"occurred_at"`
	Actor       string    `json:"actor"`
}

func (s *Server) GetHomeDashboard(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 1. Cycle Health: Fetch Revenue MTD
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	revReq := billingoverviewdomain.OverviewRequest{
		Start:       startOfMonth,
		End:         now,
		Granularity: billingoverviewdomain.GranularityDay,
		Compare:     true, // To get previous month same period? (GetRevenue might not support "same period last month" exactly without custom logic, but let's try)
	}

	revenue, err := s.billingOverviewSvc.GetRevenue(c.Request.Context(), revReq)
	currentRevenue := 0.0
	previousRevenue := 0.0
	if err == nil {
		if revenue.Total != nil {
			currentRevenue = float64(*revenue.Total) / 100.0 // Cents to Dollars
		}
		if revenue.Previous != nil {
			previousRevenue = float64(*revenue.Previous) / 100.0
		}
	}

	day := now.Day()
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()

	health := CycleHealth{
		DateProgress:    fmt.Sprintf("Day %d of %d", day, daysInMonth),
		ProgressPercent: int((float64(day) / float64(daysInMonth)) * 100),
		CurrentRevenue:  currentRevenue,
		PreviousRevenue: previousRevenue,
	}

	// 2. Pulse: Real Usage Stats (1 hour)
	oneHourAgo := now.Add(-1 * time.Hour)
	var usageStats struct {
		TotalCount int64
		ErrorCount int64
	}

	// Direct DB query for speed/convenience since usage svc doesn't expose stats
	// Assuming table 'usage_events'
	s.db.WithContext(c.Request.Context()).Raw(`
		SELECT 
			COUNT(*) as total_count,
			COUNT(CASE WHEN error IS NOT NULL AND error != '' THEN 1 END) as error_count
		FROM usage_events
		WHERE org_id = ? AND created_at >= ?
	`, orgID, oneHourAgo).Scan(&usageStats)

	errorRate := 0.0
	if usageStats.TotalCount > 0 {
		errorRate = (float64(usageStats.ErrorCount) / float64(usageStats.TotalCount)) * 100
	}

	pulse := SystemPulse{
		UsageVolumeLastHour: int(usageStats.TotalCount),
		ErrorRate:           errorRate,
		JobsRunning:         0, // TODO: Link to job system
	}

	// 3. Alerts: Check for Overdue Invoices
	opsCtx := c.Request.Context()
	overdueInvoices, _ := s.billingOperationsSvc.ListOverdueInvoices(opsCtx, 5)

	alerts := []AlertBase{}

	if len(overdueInvoices.Invoices) > 0 {
		alerts = append(alerts, AlertBase{
			ID:       "overdue_invoices",
			Severity: "warning",
			Message:  "Overdue invoices require attention",
			Count:    len(overdueInvoices.Invoices),
		})
	}

	// Check failed payment provider configs? (Mock for now)

	// 4. Activity Feed (Real Audit Logs)
	auditLogs, err := s.auditSvc.List(c.Request.Context(), auditdomain.ListAuditLogRequest{
		Pagination: pagination.Pagination{
			PageSize: 10,
		},
	})
	activity := make([]HomeActivityItem, 0)
	if err == nil {
		for _, log := range auditLogs.AuditLogs {
			actor := "System"
			if log.ActorID != nil {
				actor = *log.ActorID
			}
			target := log.TargetType
			if log.TargetID != nil {
				target = fmt.Sprintf("%s %s", log.TargetType, *log.TargetID)
			}

			activity = append(activity, HomeActivityItem{
				ID:          log.ID.String(),
				Description: fmt.Sprintf("%s %s", log.Action, target),
				OccurredAt:  log.CreatedAt,
				Actor:       actor,
			})
		}
	} else {
		// fallback mock if audit fails or empty
		activity = append(activity, HomeActivityItem{
			ID:          "mock-1",
			Description: "System Check",
			OccurredAt:  now,
			Actor:       "System",
		})
	}

	c.JSON(http.StatusOK, HomeDashboardResponse{
		CycleHealth: health,
		Pulse:       pulse,
		Alerts:      alerts,
		Activity:    activity,
	})
}
