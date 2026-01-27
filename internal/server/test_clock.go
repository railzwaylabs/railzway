package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
)

func (s *Server) CreateTestClock(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		InitialTime string `json:"initial_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	initialTime := time.Now()
	if req.InitialTime != "" {
		var err error
		initialTime, err = time.Parse(time.RFC3339, req.InitialTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid initial_time format"})
			return
		}
	}

	clock, err := s.testClockSvc.Create(c.Request.Context(), testclockdomain.CreateTestClockRequest{
		Name:        req.Name,
		InitialTime: initialTime,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": clock})
}

func (s *Server) GetTestClock(c *gin.Context) {
	id := c.Param("id")
	clock, err := s.testClockSvc.Get(c.Request.Context(), id)
	if err != nil {
		if err == testclockdomain.ErrTestClockNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Test clock not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": clock})
}

func (s *Server) AdvanceTestClock(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Seconds int `json:"seconds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assuming the service takes options or we just advance by duration
	// In phase 1 we integrated Advance(ctx, id, duration)
	clock, err := s.testClockSvc.Advance(c.Request.Context(), id, time.Duration(req.Seconds)*time.Second)
	if err != nil {
		if err == testclockdomain.ErrTestClockNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Test clock not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": clock})
}

func (s *Server) ListTestClocks(c *gin.Context) {
	clocks, err := s.testClockSvc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": clocks})
}

func (s *Server) DeleteTestClock(c *gin.Context) {
	id := c.Param("id")
	if err := s.testClockSvc.Delete(c.Request.Context(), id); err != nil {
		if err == testclockdomain.ErrTestClockNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Test clock not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) UpdateTestClock(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clock, err := s.testClockSvc.Update(c.Request.Context(), id, testclockdomain.UpdateTestClockRequest{
		Name: req.Name,
	})
	if err != nil {
		if err == testclockdomain.ErrTestClockNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Test clock not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": clock})
}
