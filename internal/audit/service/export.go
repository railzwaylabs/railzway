package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	auditdomain "github.com/railzwaylabs/railzway/internal/audit/domain"
	"gorm.io/gorm"
)

type ExportService struct {
	db *gorm.DB
}

func NewExportService(db *gorm.DB) auditdomain.ExportService {
	return &ExportService{db: db}
}

func (s *ExportService) Export(ctx context.Context, req auditdomain.ExportRequest) (*auditdomain.ExportResult, error) {
	// Fetch audit logs
	query := s.db.WithContext(ctx).Model(&auditdomain.AuditLog{}).
		Where("created_at >= ? AND created_at < ?", req.StartDate, req.EndDate)
	
	if req.OrgID != nil {
		query = query.Where("org_id = ?", *req.OrgID)
	}
	
	if len(req.Actions) > 0 {
		query = query.Where("action IN ?", req.Actions)
	}
	
	var logs []auditdomain.AuditLog
	if err := query.Order("created_at ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	
	var data []byte
	var err error
	
	switch req.Format {
	case auditdomain.ExportFormatCSV:
		data, err = s.formatCSV(logs)
	case auditdomain.ExportFormatJSON:
		data, err = s.formatJSON(logs)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", req.Format)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Calculate checksum for integrity verification
	checksum := calculateChecksum(data)
	
	return &auditdomain.ExportResult{
		Data:     data,
		Checksum: checksum,
		Format:   req.Format,
		Count:    len(logs),
	}, nil
}

func (s *ExportService) formatCSV(logs []auditdomain.AuditLog) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	
	// Write header
	header := []string{
		"timestamp",
		"org_id",
		"actor_type",
		"actor_id",
		"action",
		"target_type",
		"target_id",
		"ip_address",
		"user_agent",
		"metadata",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	
	// Write rows
	for _, log := range logs {
		metadataJSON, _ := json.Marshal(log.Metadata)
		
		row := []string{
			log.CreatedAt.Format(time.RFC3339),
			formatSnowflakeID(log.OrgID),
			log.ActorType,
			formatStringPtr(log.ActorID),
			log.Action,
			log.TargetType,
			formatStringPtr(log.TargetID),
			formatStringPtr(log.IPAddress),
			formatStringPtr(log.UserAgent),
			string(metadataJSON),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func (s *ExportService) formatJSON(logs []auditdomain.AuditLog) ([]byte, error) {
	// Create export-friendly structure
	type ExportRecord struct {
		Timestamp  string                 `json:"timestamp"`
		OrgID      string                 `json:"org_id,omitempty"`
		ActorType  string                 `json:"actor_type"`
		ActorID    string                 `json:"actor_id,omitempty"`
		Action     string                 `json:"action"`
		TargetType string                 `json:"target_type"`
		TargetID   string                 `json:"target_id,omitempty"`
		IPAddress  string                 `json:"ip_address,omitempty"`
		UserAgent  string                 `json:"user_agent,omitempty"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
	}
	
	records := make([]ExportRecord, 0, len(logs))
	for _, log := range logs {
		records = append(records, ExportRecord{
			Timestamp:  log.CreatedAt.Format(time.RFC3339),
			OrgID:      formatSnowflakeID(log.OrgID),
			ActorType:  log.ActorType,
			ActorID:    formatStringPtr(log.ActorID),
			Action:     log.Action,
			TargetType: log.TargetType,
			TargetID:   formatStringPtr(log.TargetID),
			IPAddress:  formatStringPtr(log.IPAddress),
			UserAgent:  formatStringPtr(log.UserAgent),
			Metadata:   log.Metadata,
		})
	}
	
	return json.MarshalIndent(records, "", "  ")
}

func formatSnowflakeID(id *snowflake.ID) string {
	if id == nil || *id == 0 {
		return ""
	}
	return id.String()
}

func formatStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
