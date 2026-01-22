package server

import (
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	authdomain "github.com/smallbiznis/railzway/internal/auth/domain"
)

func (s *Server) ListUserOrgs(c *gin.Context) {
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

func (s *Server) UseOrg(c *gin.Context) {
	session, ok := s.sessionFromContext(c)
	if !ok {
		AbortWithError(c, ErrUnauthorized)
		return
	}

	rawOrgID := strings.TrimSpace(c.Param("orgId"))
	if rawOrgID == "" {
		AbortWithError(c, newValidationError("org_id", "invalid_org_id", "invalid org id"))
		return
	}

	parsed, err := snowflake.ParseString(rawOrgID)
	if err != nil {
		AbortWithError(c, newValidationError("org_id", "invalid_org_id", "invalid org id"))
		return
	}

	orgIDs, err := s.loadUserOrgIDs(c.Request.Context(), session.UserID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	resolvedOrgID := int64(parsed)
	if !containsOrgID(orgIDs, resolvedOrgID) {
		AbortWithError(c, ErrForbidden)
		return
	}

	activeOrgID := resolvedOrgID
	if err := s.authsvc.UpdateSessionOrgContext(c.Request.Context(), session.ID, &activeOrgID, orgIDs); err != nil {
		AbortWithError(c, err)
		return
	}

	session.ActiveOrgID = &activeOrgID
	session.OrgIDs = orgIDs

	c.JSON(http.StatusOK, sessionViewFromSession(session))
}

func sessionViewFromSession(session *authdomain.Session) *authdomain.SessionView {
	orgIDs := toOrgIDStrings(session.OrgIDs)

	metadata := map[string]any{
		"user_id": session.UserID.String(),
		"org_ids": orgIDs,
	}
	if session.ActiveOrgID != nil {
		metadata["active_org_id"] = snowflake.ID(*session.ActiveOrgID).String()
	}

	return &authdomain.SessionView{
		Metadata: metadata,
	}
}

func toOrgIDStrings(orgIDs []int64) []string {
	out := make([]string, 0, len(orgIDs))
	for _, orgID := range orgIDs {
		out = append(out, snowflake.ID(orgID).String())
	}
	return out
}
