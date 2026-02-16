package server

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
)

const dateOnlyLayout = "2006-01-02"

// parsePaginationParams extracts and validates pagination parameters from query string.
// Returns page_token and validated page_size (defaults to 10, max 250).
func parsePaginationParams(pageToken string, pageSizeStr string) (string, int, error) {
	// Parse page size
	pageSize := pagination.DefaultPageSize
	if pageSizeStr != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(pageSizeStr))
		if err != nil {
			return "", 0, newValidationError("page_size", "invalid_page_size", "page size must be a number")
		}
		if parsed < pagination.MinPageSize || parsed > pagination.MaxPageSize {
			return "", 0, newValidationError("page_size", "invalid_page_size", "page size must be between 1 and 250")
		}
		pageSize = parsed
	}

	return strings.TrimSpace(pageToken), pageSize, nil
}

func parseOptionalBool(value string) (*bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalInt64(value string) (*int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalSnowflakeID(value string) (*snowflake.ID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := snowflake.ParseString(trimmed)
	if err != nil || parsed == 0 {
		return nil, errors.New("invalid_snowflake_id")
	}
	return &parsed, nil
}

func parseOptionalTime(value string, endOfDay bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse(dateOnlyLayout, trimmed); err == nil {
		if endOfDay {
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
		} else {
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		}
		return &parsed, nil
	}
	return nil, errors.New("invalid_time")
}
