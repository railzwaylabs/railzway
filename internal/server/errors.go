package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	apikeydomain "github.com/smallbiznis/railzway/internal/apikey/domain"
	auditdomain "github.com/smallbiznis/railzway/internal/audit/domain"
	authdomain "github.com/smallbiznis/railzway/internal/auth/domain"
	authscope "github.com/smallbiznis/railzway/internal/auth/scope"
	"github.com/smallbiznis/railzway/internal/authorization"
	billingdashboarddomain "github.com/smallbiznis/railzway/internal/billingdashboard/domain"
	billingoperationsdomain "github.com/smallbiznis/railzway/internal/billingoperations/domain"
	billingoverviewdomain "github.com/smallbiznis/railzway/internal/billingoverview/domain"
	customerdomain "github.com/smallbiznis/railzway/internal/customer/domain"
	featuredomain "github.com/smallbiznis/railzway/internal/feature/domain"
	invoicedomain "github.com/smallbiznis/railzway/internal/invoice/domain"
	invoicetemplatedomain "github.com/smallbiznis/railzway/internal/invoicetemplate/domain"
	meterdomain "github.com/smallbiznis/railzway/internal/meter/domain"
	organizationdomain "github.com/smallbiznis/railzway/internal/organization/domain"
	paymentdomain "github.com/smallbiznis/railzway/internal/payment/domain"
	pricedomain "github.com/smallbiznis/railzway/internal/price/domain"
	priceamountdomain "github.com/smallbiznis/railzway/internal/priceamount/domain"
	pricetierdomain "github.com/smallbiznis/railzway/internal/pricetier/domain"
	productdomain "github.com/smallbiznis/railzway/internal/product/domain"
	productfeaturedomain "github.com/smallbiznis/railzway/internal/productfeature/domain"
	paymentproviderdomain "github.com/smallbiznis/railzway/internal/providers/payment/domain"
	ratingdomain "github.com/smallbiznis/railzway/internal/rating/domain"
	signupdomain "github.com/smallbiznis/railzway/internal/signup/domain"
	subscriptiondomain "github.com/smallbiznis/railzway/internal/subscription/domain"
	taxdomain "github.com/smallbiznis/railzway/internal/tax/domain"
	usagedomain "github.com/smallbiznis/railzway/internal/usage/domain"
	"gorm.io/gorm"
)

type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (v ValidationErrors) Error() string {
	return "validation error"
}

type errorPayload struct {
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

type errorResponse struct {
	Error errorPayload `json:"error"`
}

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrConflict           = errors.New("conflict")
	ErrInternal           = errors.New("internal_error")
	ErrNotFound           = errors.New("not_found")
	ErrInvalidRequest     = errors.New("invalid_request")
	ErrServiceUnavailable = errors.New("service_unavailable")
	ErrOrgRequired        = errors.New("org_required")
	ErrRateLimited        = errors.New("rate_limited")
	ErrInvoiceUnavailable = errors.New("invoice_unavailable")
)

func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() {
			return
		}

		lastErr := c.Errors.Last()
		if lastErr == nil {
			return
		}

		status, payload := mapError(lastErr.Err)
		c.Header("Content-Type", "application/json")
		c.AbortWithStatusJSON(status, errorResponse{Error: payload})
	}
}

func AbortWithError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	_ = c.Error(err)
	c.Abort()
}

func invalidRequestError() error {
	return newValidationError("request", "invalid_request", "invalid request")
}

func newValidationError(field, code, message string) error {
	return &ValidationErrors{
		Errors: []ValidationError{
			{
				Field:   field,
				Code:    code,
				Message: message,
			},
		},
	}
}

func mapError(err error) (int, errorPayload) {
	if err == nil {
		return http.StatusInternalServerError, errorPayload{
			Type:    "internal_error",
			Message: "internal server error",
		}
	}

	if vErr := asValidationErrors(err); vErr != nil {
		return http.StatusBadRequest, errorPayload{
			Type:    "validation_error",
			Message: "validation error",
			Errors:  vErr.Errors,
		}
	}

	if isValidationError(err) {
		code := validationErrorCode(err)
		return http.StatusBadRequest, errorPayload{
			Type:    "validation_error",
			Message: "validation error",
			Errors: []ValidationError{
				{
					Field:   validationErrorField(code),
					Code:    code,
					Message: validationErrorMessage(code),
				},
			},
		}
	}

	switch {
	case errors.Is(err, ErrUnauthorized),
		errors.Is(err, authdomain.ErrInvalidCredentials),
		errors.Is(err, authdomain.ErrInvalidSession),
		errors.Is(err, authdomain.ErrSessionExpired),
		errors.Is(err, authdomain.ErrSessionRevoked):
		return http.StatusUnauthorized, errorPayload{
			Type:    "unauthorized",
			Message: "unauthorized",
		}
	case errors.Is(err, ErrForbidden),
		errors.Is(err, authorization.ErrForbidden):
		return http.StatusForbidden, errorPayload{
			Type:    "forbidden",
			Message: "forbidden",
		}
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests, errorPayload{
			Type:    "rate_limited",
			Message: "rate limited",
		}
	case errors.Is(err, organizationdomain.ErrForbidden):
		return http.StatusForbidden, errorPayload{
			Type:    "forbidden",
			Message: "forbidden",
		}
	case errors.Is(err, ErrConflict),
		errors.Is(err, authdomain.ErrUserExists):
		return http.StatusConflict, errorPayload{
			Type:    "conflict",
			Message: "conflict",
		}
	case isNotFoundError(err):
		return http.StatusNotFound, errorPayload{
			Type:    "not_found",
			Message: "not found",
		}
	case errors.Is(err, ErrInvoiceUnavailable):
		return http.StatusNotFound, errorPayload{
			Type:    "not_found",
			Message: "invoice not available",
		}
	case errors.Is(err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable, errorPayload{
			Type:    "service_unavailable",
			Message: "service unavailable",
		}
	case errors.Is(err, paymentproviderdomain.ErrEncryptionKeyMissing):
		return http.StatusServiceUnavailable, errorPayload{
			Type:    "service_unavailable",
			Message: "service unavailable",
		}
	case errors.Is(err, ErrOrgRequired):
		return http.StatusPreconditionRequired, errorPayload{
			Type:    "precondition_required",
			Message: "organization required",
		}
	case errors.Is(err, ErrInternal):
		return http.StatusInternalServerError, errorPayload{
			Type:    "internal_error",
			Message: "internal server error",
		}
	default:
		return http.StatusInternalServerError, errorPayload{
			Type:    "internal_error",
			Message: "internal server error",
		}
	}
}

func classifyErrorForLog(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	_, payload := mapError(err)
	code := ""
	if len(payload.Errors) > 0 {
		code = payload.Errors[0].Code
	}
	return payload.Type, code
}

func asValidationErrors(err error) *ValidationErrors {
	var vErr *ValidationErrors
	if errors.As(err, &vErr) && vErr != nil {
		return vErr
	}
	return nil
}

func isValidationError(err error) bool {
	switch {
	case errors.Is(err, ErrInvalidRequest),
		errors.Is(err, signupdomain.ErrInvalidRequest):
		return true
	case isOrganizationValidationError(err),
		isCustomerValidationError(err),
		isBillingDashboardValidationError(err),
		isBillingOperationsValidationError(err),
		isBillingOverviewValidationError(err),
		isInvoiceValidationError(err),
		isInvoiceTemplateValidationError(err),
		isRatingValidationError(err),
		isUsageValidationError(err),
		isPaymentValidationError(err),
		isProductValidationError(err),
		isFeatureValidationError(err),
		isPriceValidationError(err),
		isPricingValidationError(err),
		isPriceAmountValidationError(err),
		isPriceTierValidationError(err),
		isMeterValidationError(err),
		isSubscriptionValidationError(err),
		isAPIKeyValidationError(err),
		isAuditValidationError(err),
		isAuthorizationValidationError(err),
		isPaymentProviderValidationError(err),
		isTaxValidationError(err),
		isProductFeatureValidationError(err),
		isScopeValidationError(err):
		return true
	default:
		return false
	}
}

func isUsageValidationError(err error) bool {
	switch err {
	case usagedomain.ErrInvalidOrganization,
		usagedomain.ErrInvalidCustomer,
		usagedomain.ErrInvalidSubscription,
		usagedomain.ErrInvalidSubscriptionItem,
		usagedomain.ErrInvalidMeter,
		usagedomain.ErrInvalidMeterCode,
		usagedomain.ErrInvalidValue,
		usagedomain.ErrInvalidRecordedAt,
		usagedomain.ErrInvalidIdempotencyKey:
		return true
	default:
		return false
	}
}

func isFeatureValidationError(err error) bool {
	switch err {
	case featuredomain.ErrInvalidOrganization,
		featuredomain.ErrInvalidCode,
		featuredomain.ErrInvalidName,
		featuredomain.ErrInvalidType,
		featuredomain.ErrInvalidMeterID,
		featuredomain.ErrInvalidID:
		return true
	default:
		return false
	}
}

func isProductFeatureValidationError(err error) bool {
	switch err {
	case productfeaturedomain.ErrInvalidOrganization,
		productfeaturedomain.ErrInvalidProductID,
		productfeaturedomain.ErrInvalidFeatureID,
		productfeaturedomain.ErrInvalidMeterID,
		productfeaturedomain.ErrFeatureInactive:
		return true
	default:
		return false
	}
}

func isBillingDashboardValidationError(err error) bool {
	switch err {
	case billingdashboarddomain.ErrInvalidOrganization:
		return true
	default:
		return false
	}
}

func isBillingOperationsValidationError(err error) bool {
	switch err {
	case billingoperationsdomain.ErrInvalidOrganization,
		billingoperationsdomain.ErrInvalidEntityType,
		billingoperationsdomain.ErrInvalidEntityID,
		billingoperationsdomain.ErrInvalidActionType,
		billingoperationsdomain.ErrInvalidAssignee,
		billingoperationsdomain.ErrInvalidIdempotencyKey,
		billingoperationsdomain.ErrInvalidAssignmentTTL:
		return true
	default:
		return false
	}
}

func isBillingOverviewValidationError(err error) bool {
	switch err {
	case billingoverviewdomain.ErrInvalidOrganization:
		return true
	default:
		return false
	}
}

func isTaxValidationError(err error) bool {
	switch err {
	case taxdomain.ErrInvalidOrganization,
		taxdomain.ErrInvalidName,
		taxdomain.ErrInvalidID,
		taxdomain.ErrInvalidTaxCode,
		taxdomain.ErrInvalidTaxMode,
		taxdomain.ErrInvalidTaxRate:
		return true
	default:
		return false
	}
}

func isNotFoundError(err error) bool {
	switch {
	case errors.Is(err, ErrNotFound),
		errors.Is(err, customerdomain.ErrNotFound),
		errors.Is(err, invoicetemplatedomain.ErrNotFound),
		errors.Is(err, invoicedomain.ErrInvoiceTemplateNotFound),
		errors.Is(err, productdomain.ErrNotFound),
		errors.Is(err, productfeaturedomain.ErrProductNotFound),
		errors.Is(err, featuredomain.ErrNotFound),
		errors.Is(err, productfeaturedomain.ErrFeatureNotFound),
		errors.Is(err, pricedomain.ErrNotFound),
		errors.Is(err, apikeydomain.ErrNotFound),
		errors.Is(err, meterdomain.ErrMeterNotFound),
		errors.Is(err, productfeaturedomain.ErrMeterNotFound),
		errors.Is(err, priceamountdomain.ErrNotFound),
		errors.Is(err, pricetierdomain.ErrNotFound),
		errors.Is(err, invoicedomain.ErrBillingCycleNotFound),
		errors.Is(err, invoicedomain.ErrInvoiceNotFound),
		errors.Is(err, ratingdomain.ErrBillingCycleNotFound),
		errors.Is(err, subscriptiondomain.ErrSubscriptionNotFound),
		errors.Is(err, subscriptiondomain.ErrSubscriptionItemNotFound),
		errors.Is(err, paymentdomain.ErrProviderNotFound),
		errors.Is(err, paymentproviderdomain.ErrNotFound),
		errors.Is(err, taxdomain.ErrNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return true
	default:
		return false
	}
}

func isInvoiceValidationError(err error) bool {
	switch err {
	case invoicedomain.ErrInvalidOrganization,
		invoicedomain.ErrInvalidBillingCycle,
		invoicedomain.ErrBillingCycleNotClosed,
		invoicedomain.ErrMissingLedgerEntry,
		invoicedomain.ErrMissingRatingResults,
		invoicedomain.ErrCurrencyMismatch,
		invoicedomain.ErrInvalidInvoiceID,
		invoicedomain.ErrInvoiceNotDraft,
		invoicedomain.ErrInvoiceNotFinalized:
		return true
	default:
		return false
	}
}

func isInvoiceTemplateValidationError(err error) bool {
	switch err {
	case invoicetemplatedomain.ErrInvalidOrganization,
		invoicetemplatedomain.ErrInvalidID,
		invoicetemplatedomain.ErrInvalidName,
		invoicetemplatedomain.ErrInvalidCurrency,
		invoicetemplatedomain.ErrInvalidLocale:
		return true
	default:
		return false
	}
}

func isAPIKeyValidationError(err error) bool {
	switch err {
	case apikeydomain.ErrInvalidOrganization,
		apikeydomain.ErrInvalidName,
		apikeydomain.ErrInvalidKeyID:
		return true
	default:
		return false
	}
}

func isAuditValidationError(err error) bool {
	switch err {
	case auditdomain.ErrInvalidOrganization,
		auditdomain.ErrInvalidPageToken,
		auditdomain.ErrInvalidTimeRange,
		auditdomain.ErrInvalidAction:
		return true
	default:
		return false
	}
}

func isAuthorizationValidationError(err error) bool {
	switch err {
	case authorization.ErrInvalidActor,
		authorization.ErrInvalidOrganization,
		authorization.ErrInvalidObject,
		authorization.ErrInvalidAction:
		return true
	default:
		return false
	}
}

func isPaymentProviderValidationError(err error) bool {
	switch err {
	case paymentproviderdomain.ErrInvalidOrganization,
		paymentproviderdomain.ErrInvalidProvider,
		paymentproviderdomain.ErrInvalidConfig:
		return true
	default:
		return false
	}
}

func isPaymentValidationError(err error) bool {
	switch err {
	case paymentdomain.ErrInvalidProvider,
		paymentdomain.ErrInvalidSignature,
		paymentdomain.ErrInvalidPayload,
		paymentdomain.ErrInvalidEvent,
		paymentdomain.ErrInvalidCustomer,
		paymentdomain.ErrInvalidAmount,
		paymentdomain.ErrInvalidCurrency:
		return true
	default:
		return false
	}
}

func isScopeValidationError(err error) bool {
	switch err {
	case authscope.ErrInvalidScope:
		return true
	default:
		return false
	}
}

func isRatingValidationError(err error) bool {
	switch err {
	case ratingdomain.ErrInvalidBillingCycle,
		ratingdomain.ErrBillingCycleNotClosing,
		ratingdomain.ErrMissingUsage,
		ratingdomain.ErrMissingPriceAmount,
		ratingdomain.ErrMissingMeter,
		ratingdomain.ErrInvalidQuantity:
		return true
	default:
		return false
	}
}

func validationErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidRequest),
		errors.Is(err, signupdomain.ErrInvalidRequest):
		return "invalid_request"
	default:
		return err.Error()
	}
}

func validationErrorField(code string) string {
	if code == "invalid_scope" {
		return "scopes"
	}
	if strings.HasPrefix(code, "invalid_") {
		return strings.TrimPrefix(code, "invalid_")
	}
	if code == "invalid_request" {
		return "request"
	}
	return ""
}

func validationErrorMessage(code string) string {
	switch code {
	case "invalid_request":
		return "invalid request"
	default:
		return "invalid value"
	}
}
