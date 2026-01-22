package server

import (
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/smallbiznis/railzway/internal/organization/domain"
)

func (s *Server) CreateOrg(c *gin.Context) {
	userID, ok := s.userIDFromSession(c)
	if !ok {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	var req domain.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, invalidRequestError())
		return
	}

	org, err := s.organizationSvc.Create(c.Request.Context(), userID, req)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"org": org})
}

func (s *Server) GetOrg(c *gin.Context) {

	orgID := strings.TrimSpace(c.Param("id"))
	if orgID == "" {
		AbortWithError(c, newValidationError("id", "invalid_id", "invalid id"))
		return
	}

	if _, err := snowflake.ParseString(orgID); err != nil {
		AbortWithError(c, newValidationError("id", "invalid_id", "invalid id"))
		return
	}

	org, err := s.organizationSvc.GetByID(c.Request.Context(), orgID)
	if err != nil {
		AbortWithError(c, err)
		return
	}
	if org == nil {
		AbortWithError(c, ErrNotFound)
		return
	}

	c.JSON(http.StatusOK, gin.H{"org": org})
}

func (s *Server) MeOrg(c *gin.Context) {
	userID, ok := s.userIDFromSession(c)
	if !ok {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	orgs, err := s.organizationSvc.ListOrganizationsByUser(c.Request.Context(), userID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"orgs": orgs})
}
