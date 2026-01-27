package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterSystemRoutes() {
	s.engine.GET("/ready", s.GetSystemReadiness)
}

// GetSystemReadiness exposes a system-level readiness endpoint for control-plane orchestration.
func (s *Server) GetSystemReadiness(c *gin.Context) {
	ctx := c.Request.Context()

	issues := make([]ReadinessIssue, 0, 4)
	isReady := true

	// Schema gate
	if s.schemaGate == nil {
		isReady = false
		issues = append(issues, ReadinessIssue{
			ID:         "schema_gate",
			Status:     ReadinessStateNotReady,
			ActionHref: "",
			Evidence:   map[string]string{"error": "schema gate not configured"},
		})
	} else if err := s.schemaGate.MustBeActive(ctx); err != nil {
		isReady = false
		issues = append(issues, ReadinessIssue{
			ID:         "schema_gate",
			Status:     ReadinessStateNotReady,
			ActionHref: "",
			Evidence:   map[string]string{"error": err.Error()},
		})
	} else {
		issues = append(issues, ReadinessIssue{
			ID:         "schema_gate",
			Status:     ReadinessStateReady,
			ActionHref: "",
		})
	}

	// TestClock should not be active in production.
	if s.db == nil {
		isReady = false
		issues = append(issues, ReadinessIssue{
			ID:         "test_clock_disabled",
			Status:     ReadinessStateNotReady,
			ActionHref: "",
			Evidence:   map[string]string{"error": "db not configured"},
		})
	} else {
		activeCount, err := s.countActiveTestClocks(ctx)
		if err != nil {
			isReady = false
			issues = append(issues, ReadinessIssue{
				ID:         "test_clock_disabled",
				Status:     ReadinessStateNotReady,
				ActionHref: "",
				Evidence:   map[string]string{"error": err.Error()},
			})
		} else if activeCount > 0 {
			isReady = false
			issues = append(issues, ReadinessIssue{
				ID:         "test_clock_disabled",
				Status:     ReadinessStateNotReady,
				ActionHref: "",
				Evidence:   map[string]string{"active_test_clocks": fmt.Sprintf("%d", activeCount)},
			})
		} else {
			issues = append(issues, ReadinessIssue{
				ID:         "test_clock_disabled",
				Status:     ReadinessStateReady,
				ActionHref: "",
			})
		}
	}

	// Scheduler lease safety is enforced by the control-plane; mark optional.
	issues = append(issues, ReadinessIssue{
		ID:         "scheduler_single_lease",
		Status:     ReadinessStateOptional,
		ActionHref: "",
		Evidence:   map[string]string{"note": "lease enforced by control-plane"},
	})

	state := ReadinessStateReady
	if !isReady {
		state = ReadinessStateNotReady
	}

	c.JSON(http.StatusOK, gin.H{
		"ready":        isReady,
		"system_state": state,
		"issues":       issues,
	})
}

func (s *Server) countActiveTestClocks(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(1)
		 FROM test_clocks
		 WHERE status IN ('active', 'advancing')`,
	).Scan(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
