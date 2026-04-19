package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

func parseTimePtrValue(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	return parseTimePtr(*value)
}

type validationErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func bindJSONOrAbort(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
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
				{Field: field, Message: "must be a " + typeErr.Type.String()},
			},
		})
		return
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		details := make([]validationErrorDetail, 0, len(validationErrs))
		for _, verr := range validationErrs {
			details = append(details, validationErrorDetail{
				Field:   toSnakeCase(verr.Field()),
				Message: validationMessage(verr),
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
