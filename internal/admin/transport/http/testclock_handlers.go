package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
)

type upsertTestClockRequest struct {
	FrozenTime  *int64 `json:"frozen_time"`
	CurrentTime string `json:"current_time"`
	Name        string `json:"name"`
	Status      string `json:"status"`
}

type advanceTestClockRequest struct {
	FrozenTime       *int64 `json:"frozen_time"`
	AdvanceBySeconds int64  `json:"advance_by_seconds"`
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

func (h *Handler) ListTestClocks(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	clocks, err := h.testclocks.List(ctx)
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"test_clocks": clocks})
}

func (h *Handler) GetTestClockByID(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := orgcontext.WithOrgID(c.Request.Context(), orgID)
	clockInfo, err := h.testclocks.GetByID(ctx, strings.TrimSpace(c.Param("test_clock_id")))
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, clockInfo)
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
	frozenTime, err := parseTestClockFrozenTime(payload.FrozenTime, payload.CurrentTime)
	if err != nil || frozenTime.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_frozen_time"})
		return
	}

	resp, err := h.testclocks.Upsert(ctx, testclockdomain.UpsertTestClockRequest{
		FrozenTime: frozenTime.UTC(),
		Name:       strings.TrimSpace(payload.Name),
		Status:     strings.TrimSpace(payload.Status),
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
	clockID := strings.TrimSpace(c.Param("test_clock_id"))
	frozenTime, err := h.parseTestClockAdvanceTime(ctx, clockID, payload)
	if err != nil || frozenTime.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_frozen_time"})
		return
	}

	resp, err := h.testclocks.Advance(ctx, testclockdomain.AdvanceTestClockRequest{
		ID:         clockID,
		FrozenTime: frozenTime.UTC(),
	})
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) PauseTestClockByID(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	resp, err := h.testclocks.PauseByID(ctx, strings.TrimSpace(c.Param("test_clock_id")))
	if err != nil {
		writeTestClockError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ResumeTestClockByID(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	ctx := h.withAuditContext(c, orgcontext.WithOrgID(c.Request.Context(), orgID))

	resp, err := h.testclocks.ResumeByID(ctx, strings.TrimSpace(c.Param("test_clock_id")))
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

func parseTestClockFrozenTime(epoch *int64, fallbackRFC3339 string) (time.Time, error) {
	if epoch != nil {
		if *epoch <= 0 {
			return time.Time{}, testclockdomain.ErrInvalidTime
		}
		return time.Unix(*epoch, 0).UTC(), nil
	}
	currentTime, err := parseTimePtr(fallbackRFC3339)
	if err != nil || currentTime == nil {
		return time.Time{}, testclockdomain.ErrInvalidTime
	}
	return currentTime.UTC(), nil
}

func (h *Handler) parseTestClockAdvanceTime(ctx context.Context, id string, payload advanceTestClockRequest) (time.Time, error) {
	if payload.FrozenTime != nil {
		if *payload.FrozenTime <= 0 {
			return time.Time{}, testclockdomain.ErrInvalidTime
		}
		return time.Unix(*payload.FrozenTime, 0).UTC(), nil
	}
	if payload.AdvanceBySeconds <= 0 {
		return time.Time{}, testclockdomain.ErrInvalidAdvance
	}
	var (
		clockInfo *testclockdomain.TestClock
		err       error
	)
	if strings.TrimSpace(id) == "" {
		clockInfo, err = h.testclocks.Get(ctx)
	} else {
		clockInfo, err = h.testclocks.GetByID(ctx, id)
	}
	if err != nil {
		return time.Time{}, err
	}
	return clockInfo.CurrentTime.Add(time.Duration(payload.AdvanceBySeconds) * time.Second).UTC(), nil
}
