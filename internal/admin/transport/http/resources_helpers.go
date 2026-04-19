package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	adminauth "github.com/railzwaylabs/railzway/internal/admin/auth"
	appsdomain "github.com/railzwaylabs/railzway/internal/apps/domain"
	"github.com/railzwaylabs/railzway/internal/auditlog"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	"github.com/railzwaylabs/railzway/internal/httpmiddleware"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	ledgerdomain "github.com/railzwaylabs/railzway/internal/ledger/domain"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	paymentdomain "github.com/railzwaylabs/railzway/internal/payment/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	productfeaturedomain "github.com/railzwaylabs/railzway/internal/productfeature/domain"
	ratingdomain "github.com/railzwaylabs/railzway/internal/rating/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	taxdomain "github.com/railzwaylabs/railzway/internal/tax/domain"
	testclockdomain "github.com/railzwaylabs/railzway/internal/testclock/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
)

type validationErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func parseBoolPtr(value string) (*bool, error) {
	raw := strings.TrimSpace(strings.ToLower(value))
	if raw == "" {
		return nil, nil
	}
	switch raw {
	case "true", "1", "yes", "y", "on":
		val := true
		return &val, nil
	case "false", "0", "no", "n", "off":
		val := false
		return &val, nil
	default:
		return nil, errors.New("invalid_bool")
	}
}

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

func bindOptionalJSON(c *gin.Context, dst interface{}) error {
	if c.Request.Body == nil {
		return nil
	}
	if c.Request.ContentLength == 0 {
		return nil
	}
	if err := c.ShouldBindJSON(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func bindJSONOrAbort(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		writeBindJSONError(c, err)
		return false
	}
	return true
}

func bindOptionalJSONOrAbort(c *gin.Context, dst interface{}) bool {
	if err := bindOptionalJSON(c, dst); err != nil {
		writeBindJSONError(c, err)
		return false
	}
	return true
}

func writeBindJSONError(c *gin.Context, err error) {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_json",
			"message": "Request body contains invalid JSON syntax.",
		})
		return
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := strings.TrimSpace(typeErr.Field)
		if field == "" {
			field = "request_body"
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation_failed",
			"details": []validationErrorDetail{
				{
					Field:   field,
					Message: "must be a " + typeErr.Type.String(),
				},
			},
		})
		return
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make([]validationErrorDetail, 0, len(validationErrs))
		for _, verr := range validationErrs {
			field := toSnakeCase(verr.Field())
			message := validationMessage(verr)
			details = append(details, validationErrorDetail{
				Field:   field,
				Message: message,
			})
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_failed",
			"details": details,
		})
		return
	}

	if errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_json",
			"message": "Request body is empty.",
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "invalid_json",
		"message": "Request body could not be parsed.",
	})
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "oneof":
		return "must be one of: " + err.Param()
	case "min":
		return "must be at least " + err.Param()
	case "max":
		return "must be at most " + err.Param()
	case "gt":
		return "must be greater than " + err.Param()
	case "gte":
		return "must be greater than or equal to " + err.Param()
	case "lt":
		return "must be less than " + err.Param()
	case "lte":
		return "must be less than or equal to " + err.Param()
	default:
		return "is invalid"
	}
}

func toSnakeCase(value string) string {
	if value == "" {
		return value
	}
	var b strings.Builder
	for i, r := range value {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (h *Handler) withAuditContext(c *gin.Context, ctx context.Context) context.Context {
	var actorID *uuid.UUID
	if val, ok := c.Get(adminauth.CtxUserID); ok {
		switch v := val.(type) {
		case uuid.UUID:
			if v != uuid.Nil {
				actorID = &v
			}
		case string:
			if parsed, err := uuid.Parse(v); err == nil && parsed != uuid.Nil {
				actorID = &parsed
			}
		}
	}
	ctx = auditlog.WithActor(ctx, "user", actorID)

	requestID := httpmiddleware.RequestIDFromContext(c.Request.Context())
	if requestID == "" {
		requestID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
	}
	if requestID == "" {
		requestID = httpmiddleware.TraceIDFromContext(c.Request.Context())
	}
	if requestID == "" {
		requestID = strings.TrimSpace(c.GetHeader("X-Trace-ID"))
	}
	if requestID != "" {
		ctx = auditlog.WithRequestID(ctx, requestID)
	}

	traceID := httpmiddleware.TraceIDFromContext(c.Request.Context())
	correlationID := httpmiddleware.CorrelationIDFromContext(c.Request.Context())
	if traceID != "" || correlationID != "" {
		metadata := auditlog.MetadataFromContext(ctx)
		nextMetadata := make(map[string]interface{}, len(metadata)+2)
		for key, value := range metadata {
			nextMetadata[key] = value
		}
		if traceID != "" {
			nextMetadata["trace_id"] = traceID
		}
		if correlationID != "" {
			nextMetadata["correlation_id"] = correlationID
		}
		ctx = auditlog.WithMetadata(ctx, nextMetadata)
	}

	reason := strings.TrimSpace(c.GetHeader("X-Reason"))
	if reason != "" {
		ctx = auditlog.WithReason(ctx, reason)
	}

	return ctx
}

func parseTime(value string) (time.Time, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return time.Time{}, errors.New("missing_time")
	}
	return time.Parse(time.RFC3339, raw)
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

func writePlanError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, plandomain.ErrInvalidOrganization),
		errors.Is(err, plandomain.ErrInvalidID),
		errors.Is(err, plandomain.ErrInvalidCode),
		errors.Is(err, plandomain.ErrInvalidName),
		errors.Is(err, plandomain.ErrInvalidPriceType),
		errors.Is(err, plandomain.ErrInvalidInterval),
		errors.Is(err, plandomain.ErrInvalidIntervalCount),
		errors.Is(err, plandomain.ErrInvalidCurrency),
		errors.Is(err, plandomain.ErrInvalidTierMode),
		errors.Is(err, plandomain.ErrInvalidQuantity),
		errors.Is(err, plandomain.ErrInvalidAmount),
		errors.Is(err, plandomain.ErrInvalidMeter):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, plandomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeCustomerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, customerdomain.ErrInvalidOrganization),
		errors.Is(err, customerdomain.ErrInvalidID),
		errors.Is(err, customerdomain.ErrInvalidName),
		errors.Is(err, customerdomain.ErrInvalidEmail),
		errors.Is(err, customerdomain.ErrInvalidCurrency):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, customerdomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeProductError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, productdomain.ErrInvalidOrganization),
		errors.Is(err, productdomain.ErrInvalidID),
		errors.Is(err, productdomain.ErrInvalidCode),
		errors.Is(err, productdomain.ErrInvalidName):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, productdomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeFeatureError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, featuredomain.ErrInvalidOrganization),
		errors.Is(err, featuredomain.ErrInvalidID),
		errors.Is(err, featuredomain.ErrInvalidCode),
		errors.Is(err, featuredomain.ErrInvalidName),
		errors.Is(err, featuredomain.ErrInvalidType),
		errors.Is(err, featuredomain.ErrInvalidMeter):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, featuredomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeProductFeatureError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, productfeaturedomain.ErrInvalidOrganization),
		errors.Is(err, productfeaturedomain.ErrInvalidProductID),
		errors.Is(err, productfeaturedomain.ErrInvalidFeatureID),
		errors.Is(err, productfeaturedomain.ErrInvalidMeterID),
		errors.Is(err, productfeaturedomain.ErrFeatureInactive):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, productfeaturedomain.ErrProductNotFound),
		errors.Is(err, productfeaturedomain.ErrFeatureNotFound),
		errors.Is(err, productfeaturedomain.ErrMeterNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
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

func writeUsageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usagedomain.ErrInvalidOrganization),
		errors.Is(err, usagedomain.ErrInvalidID),
		errors.Is(err, usagedomain.ErrInvalidCode),
		errors.Is(err, usagedomain.ErrInvalidName),
		errors.Is(err, usagedomain.ErrInvalidAggregation),
		errors.Is(err, usagedomain.ErrInvalidUnit),
		errors.Is(err, usagedomain.ErrInvalidMeter),
		errors.Is(err, usagedomain.ErrInvalidCustomer),
		errors.Is(err, usagedomain.ErrInvalidValue),
		errors.Is(err, usagedomain.ErrInvalidRecordedAt),
		errors.Is(err, usagedomain.ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, usagedomain.ErrNotFound):
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

func writeOrganizationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, organizationdomain.ErrInvalidName),
		errors.Is(err, organizationdomain.ErrInvalidCountry),
		errors.Is(err, organizationdomain.ErrInvalidTimezone),
		errors.Is(err, organizationdomain.ErrInvalidCurrency),
		errors.Is(err, organizationdomain.ErrInvalidUser),
		errors.Is(err, organizationdomain.ErrInvalidOrganization),
		errors.Is(err, organizationdomain.ErrInvalidEmail),
		errors.Is(err, organizationdomain.ErrInvalidRole),
		errors.Is(err, organizationdomain.ErrInvalidInvite),
		errors.Is(err, organizationdomain.ErrInvalidLink),
		errors.Is(err, organizationdomain.ErrInvalidFormat),
		errors.Is(err, organizationdomain.ErrInvalidFormatID),
		errors.Is(err, organizationdomain.ErrInvalidSequenceScope),
		errors.Is(err, organizationdomain.ErrInvalidEffectiveRange):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, organizationdomain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, organizationdomain.ErrOverlappingFormat):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, organizationdomain.ErrFormatNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeAppsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appsdomain.ErrInvalidOrganization),
		errors.Is(err, appsdomain.ErrInvalidApp),
		errors.Is(err, appsdomain.ErrInvalidStatus),
		errors.Is(err, appsdomain.ErrInvalidAuthMethod),
		errors.Is(err, appsdomain.ErrInvalidID),
		errors.Is(err, appsdomain.ErrMissingCredentials):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, appsdomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, appsdomain.ErrCredentialsKey):
		c.JSON(http.StatusInternalServerError, gin.H{"error": "credentials_key_missing"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeRatingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ratingdomain.ErrInvalidOrganization),
		errors.Is(err, ratingdomain.ErrInvalidUsageEvent),
		errors.Is(err, ratingdomain.ErrInvalidAmount),
		errors.Is(err, ratingdomain.ErrPricingNotFound),
		errors.Is(err, ratingdomain.ErrInvalidCustomer),
		errors.Is(err, ratingdomain.ErrInvalidSubscription),
		errors.Is(err, ratingdomain.ErrInvalidPlanPrice),
		errors.Is(err, ratingdomain.ErrInvalidMeter),
		errors.Is(err, ratingdomain.ErrInvalidPeriod),
		errors.Is(err, ratingdomain.ErrInvalidCursor):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ratingdomain.ErrUsageNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writePaymentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, paymentdomain.ErrInvalidOrganization),
		errors.Is(err, paymentdomain.ErrInvalidCustomer),
		errors.Is(err, paymentdomain.ErrInvalidInvoice),
		errors.Is(err, paymentdomain.ErrInvalidStatus),
		errors.Is(err, paymentdomain.ErrInvalidCursor):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeTaxError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, taxdomain.ErrInvalidOrganization),
		errors.Is(err, taxdomain.ErrInvalidCursor),
		errors.Is(err, taxdomain.ErrInvalidCode),
		errors.Is(err, taxdomain.ErrInvalidName),
		errors.Is(err, taxdomain.ErrInvalidPercentage):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, taxdomain.ErrTaxCodeExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeLedgerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ledgerdomain.ErrInvalidOrganization),
		errors.Is(err, ledgerdomain.ErrInvalidCode),
		errors.Is(err, ledgerdomain.ErrInvalidName),
		errors.Is(err, ledgerdomain.ErrInvalidType),
		errors.Is(err, ledgerdomain.ErrInvalidCurrency),
		errors.Is(err, ledgerdomain.ErrInvalidEntry),
		errors.Is(err, ledgerdomain.ErrInvalidAmount),
		errors.Is(err, ledgerdomain.ErrInvalidSource),
		errors.Is(err, ledgerdomain.ErrInvalidCursor),
		errors.Is(err, ledgerdomain.ErrUnbalancedEntry):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ledgerdomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func writeTestClockError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, testclockdomain.ErrInvalidOrganization),
		errors.Is(err, testclockdomain.ErrInvalidTime),
		errors.Is(err, testclockdomain.ErrInvalidAdvance),
		errors.Is(err, testclockdomain.ErrInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, testclockdomain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
