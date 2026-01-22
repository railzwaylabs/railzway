package server

import (
	"github.com/gin-gonic/gin"
	authdomain "github.com/smallbiznis/railzway/internal/auth/domain"
)

type SignupRequest struct {
	OrgName  string `json:"org_name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) Signup(c *gin.Context) {
	AbortWithError(c, ErrNotFound)
}

func (s *Server) enrichSessionFromToken(c *gin.Context, view *authdomain.SessionView, rawToken string) {
	if view == nil || rawToken == "" {
		return
	}

	session, err := s.authsvc.Authenticate(c.Request.Context(), rawToken)
	if err != nil || session == nil {
		return
	}

	orgIDs, err := s.loadUserOrgIDs(c.Request.Context(), session.UserID)
	if err != nil {
		return
	}

	if err := s.authsvc.UpdateSessionOrgContext(c.Request.Context(), session.ID, nil, orgIDs); err != nil {
		return
	}

	if view.Metadata == nil {
		view.Metadata = map[string]any{}
	}
	view.Metadata["org_ids"] = toOrgIDStrings(orgIDs)
}
