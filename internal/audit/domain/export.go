package domain

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
)

// ExportFormat represents the output format for audit exports.
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
)

// ExportRequest defines parameters for audit trail export.
type ExportRequest struct {
	OrgID     *snowflake.ID
	StartDate time.Time
	EndDate   time.Time
	Format    ExportFormat
	Actions   []string // Optional filter by action types
}

// ExportResult contains the exported data and metadata.
type ExportResult struct {
	Data     []byte
	Checksum string
	Format   ExportFormat
	Count    int
}

// ExportService defines the interface for audit export operations.
type ExportService interface {
	Export(ctx context.Context, req ExportRequest) (*ExportResult, error)
}
