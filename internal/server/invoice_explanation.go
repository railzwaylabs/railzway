package server

import (
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	explanationservice "github.com/smallbiznis/railzway/internal/invoice/service"
)

// ExplainInvoice handles GET /admin/invoices/:id/explanation
func (s *Server) ExplainInvoice(c *gin.Context) {
	invoiceIDStr := strings.TrimSpace(c.Param("id"))
	invoiceID, err := snowflake.ParseString(invoiceIDStr)
	if err != nil {
		AbortWithError(c, ErrInvalidRequest)
		return
	}

	explanationSvc := explanationservice.NewExplanationService(s.db)
	explanation, err := explanationSvc.ExplainInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		AbortWithError(c, ErrInternal)
		return
	}

	c.JSON(http.StatusOK, explanation)
}
