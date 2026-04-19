package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/clock"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
)

type upsertTestClockRequest struct {
	CurrentTime string `json:"current_time"`
	Status      string `json:"status"`
}

type advanceTestClockRequest struct {
	AdvanceBySeconds int64 `json:"advance_by_seconds"`
}

func (h *Handler) GetTestClock(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	clockInfo, err := h.testclocks.Get(ctx)
	if err != nil {
		if err == testclockdomain.ErrNotFound {
			c.JSON(http.StatusOK, gin.H{"clock": nil})
			return
		}
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"clock": clockInfo})
}

func (h *Handler) UpsertTestClock(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload upsertTestClockRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	currentTime, err := parseTimePtr(payload.CurrentTime)
	if err != nil || currentTime == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_current_time"})
		return
	}

	resp, err := h.testclocks.Upsert(ctx, testclockdomain.UpsertTestClockRequest{
		CurrentTime: currentTime.UTC(),
		Status:      strings.TrimSpace(payload.Status),
	})
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdvanceTestClock(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	var payload advanceTestClockRequest
	if !bindJSONOrAbort(c, &payload) {
		return
	}
	if payload.AdvanceBySeconds <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_advance_by"})
		return
	}

	resp, err := h.testclocks.Advance(ctx, testclockdomain.AdvanceTestClockRequest{
		AdvanceBy: time.Duration(payload.AdvanceBySeconds) * time.Second,
	})
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) PauseTestClock(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	resp, err := h.testclocks.Pause(ctx)
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ResumeTestClock(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	resp, err := h.testclocks.Resume(ctx)
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) InjectTestClock() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := orgIDFromContext(c)
		if !ok {
			c.Next()
			return
		}
		ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
		clockInfo, err := h.testclocks.Get(ctx)
		if err == nil && clockInfo != nil && clockInfo.Status == testclockdomain.StatusActive {
			ctx = clock.WithTestClock(ctx, clockInfo.ID, clockInfo.CurrentTime)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
