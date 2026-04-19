package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ListAIJobs(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	if h.aiScheduler == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ai_scheduler_not_configured"})
		return
	}

	status := c.Query("status")
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	jobs, total, err := h.aiScheduler.ListJobs(c.Request.Context(), orgID, status, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"total": total,
		"page":  page,
		"limit": pageSize,
	})
}

func (h *Handler) RetryAIJob(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_job_id"})
		return
	}

	if err := h.aiScheduler.RetryJob(c.Request.Context(), orgID, jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) CancelAIJob(c *gin.Context) {
	orgID, ok := orgIDFromContext(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_job_id"})
		return
	}

	if err := h.aiScheduler.CancelJob(c.Request.Context(), orgID, jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
