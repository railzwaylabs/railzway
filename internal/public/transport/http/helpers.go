package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
)

func writeInvoiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, invoicedomain.ErrInvalidOrganization),
		errors.Is(err, invoicedomain.ErrInvalidSubscription),
		errors.Is(err, invoicedomain.ErrInvalidCustomer),
		errors.Is(err, invoicedomain.ErrInvalidPeriod),
		errors.Is(err, invoicedomain.ErrInvalidStatus),
		errors.Is(err, invoicedomain.ErrInvalidCursor):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, invoicedomain.ErrUsageNotReady):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, invoicedomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
