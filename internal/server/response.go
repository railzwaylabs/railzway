package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

func respondData(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func respondList(c *gin.Context, data any, pageInfo *pagination.PageInfo) {
	if pageInfo == nil {
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "page_info": pageInfo})
}
