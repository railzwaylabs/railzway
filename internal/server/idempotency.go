package server

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func idempotencyKeyFromHeader(c *gin.Context) string {
	return strings.TrimSpace(c.GetHeader("Idempotency-Key"))
}
