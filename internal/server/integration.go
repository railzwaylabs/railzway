package server

import (
	"net/http"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/internal/integration/domain"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

func (s *Server) ListIntegrationCatalog(c *gin.Context) {
	items, err := s.integrationSvc.ListCatalog(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	// Ensure we always return an array, never null
	if items == nil {
		items = []domain.CatalogItem{}
	}

	c.JSON(http.StatusOK, items)
}

func (s *Server) ListIntegrationConnections(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	conns, err := s.integrationSvc.ListConnections(c.Request.Context(), orgID)
	if err != nil {
		c.Error(err)
		return
	}

	// Ensure we always return an array, never null
	if conns == nil {
		conns = []domain.Connection{}
	}

	c.JSON(http.StatusOK, conns)
}

type ConnectIntegrationRequest struct {
	IntegrationID string         `json:"integration_id" binding:"required"`
	Name          string         `json:"name" binding:"required"`
	Config        map[string]any `json:"config"`
	Credentials   map[string]any `json:"credentials"`
}

func (s *Server) ConnectIntegration(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var req ConnectIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conn, err := s.integrationSvc.Connect(c.Request.Context(), domain.ConnectInput{
		OrgID:         orgID,
		IntegrationID: req.IntegrationID,
		Name:          req.Name,
		Config:        req.Config,
		Credentials:   req.Credentials,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, conn)
}

func (s *Server) DisconnectIntegration(c *gin.Context) {
	orgID, ok := orgcontext.OrgIDFromContext(c.Request.Context())
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := snowflake.ParseString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Verify ownership: fetch the connection and check if it belongs to the current org
	conn, err := s.integrationSvc.GetConnection(c.Request.Context(), id)
	if err != nil {
		// If connection not found or any other error, return appropriate error
		if err == domain.ErrConnectionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
			return
		}
		c.Error(err)
		return
	}

	// Verify that the connection belongs to the current organization
	if conn.OrgID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := s.integrationSvc.Disconnect(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
}
