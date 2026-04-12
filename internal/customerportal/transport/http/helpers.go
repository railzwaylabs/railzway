package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
)

func parseInt32(value string) int32 {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return int32(parsed)
}

func parseTimePtr(value string) (*time.Time, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func writeSubscriptionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subscriptiondomain.ErrInvalidOrganization),
		errors.Is(err, subscriptiondomain.ErrInvalidID),
		errors.Is(err, subscriptiondomain.ErrInvalidCustomer),
		errors.Is(err, subscriptiondomain.ErrInvalidPlan),
		errors.Is(err, subscriptiondomain.ErrInvalidPlanPrice),
		errors.Is(err, subscriptiondomain.ErrInvalidStatus),
		errors.Is(err, subscriptiondomain.ErrInvalidCurrency),
		errors.Is(err, subscriptiondomain.ErrInvalidPeriod),
		errors.Is(err, subscriptiondomain.ErrInvalidQuantity),
		errors.Is(err, subscriptiondomain.ErrMissingItems):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, subscriptiondomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

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
