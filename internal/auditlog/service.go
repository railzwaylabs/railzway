package auditlog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/railzwaylabs/railzway/internal/db/pagination"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

var ErrInvalidCursor = errors.New("invalid_cursor")

type AuditLog struct {
	ID           uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID        uuid.UUID       `gorm:"not null" json:"org_id"`
	ActorType    string          `gorm:"not null" json:"actor_type"`
	ActorID      *uuid.UUID      `gorm:"" json:"actor_id"`
	Action       string          `gorm:"not null" json:"action"`
	ResourceType string          `gorm:"not null" json:"resource_type"`
	ResourceID   *string         `gorm:"" json:"resource_id"`
	BeforeData   json.RawMessage `gorm:"type:jsonb" json:"before_data"`
	AfterData    json.RawMessage `gorm:"type:jsonb" json:"after_data"`
	Metadata     json.RawMessage `gorm:"type:jsonb;not null" json:"metadata"`
	Reason       *string         `gorm:"" json:"reason"`
	RequestID    *string         `gorm:"" json:"request_id"`
	CreatedAt    time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type RecordInput struct {
	OrgID        uuid.UUID
	ActorType    string
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *string
	BeforeData   json.RawMessage
	AfterData    json.RawMessage
	Metadata     json.RawMessage
	Reason       *string
	RequestID    *string
}

func (s *Service) Record(ctx context.Context, input RecordInput) error {
	if s == nil || s.db == nil {
		return nil
	}
	entry := AuditLog{
		ID:           uuid.New(),
		OrgID:        input.OrgID,
		ActorType:    input.ActorType,
		ActorID:      input.ActorID,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		BeforeData:   input.BeforeData,
		AfterData:    input.AfterData,
		Metadata:     input.Metadata,
		Reason:       input.Reason,
		RequestID:    input.RequestID,
		CreatedAt:    time.Now().UTC(),
	}
	if entry.Metadata == nil {
		entry.Metadata = json.RawMessage(`{}`)
	}
	return s.db.WithContext(ctx).Create(&entry).Error
}

type ListRequest struct {
	PageToken    string
	PageSize     int
	Action       string
	ActorType    string
	ResourceType string
	ResourceID   string
	RequestID    string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}

type ListResponse struct {
	pagination.PageInfo
	Logs []AuditLog `json:"logs"`
}

type listCursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, req ListRequest) (ListResponse, error) {
	if s == nil || s.db == nil {
		return ListResponse{}, nil
	}
	if orgID == uuid.Nil {
		return ListResponse{}, nil
	}

	pageSize := pagination.ValidatePageSize(req.PageSize)
	filter := ListRequest{
		Action:       strings.TrimSpace(req.Action),
		ActorType:    strings.TrimSpace(req.ActorType),
		ResourceType: strings.TrimSpace(req.ResourceType),
		ResourceID:   strings.TrimSpace(req.ResourceID),
		RequestID:    strings.TrimSpace(req.RequestID),
		CreatedFrom:  req.CreatedFrom,
		CreatedTo:    req.CreatedTo,
	}

	cursor, err := decodeCursor(req.PageToken)
	if err != nil {
		return ListResponse{}, err
	}

	stmt := s.db.WithContext(ctx).Model(&AuditLog{}).Where("org_id = ?", orgID)
	if filter.Action != "" {
		stmt = stmt.Where("action = ?", filter.Action)
	}
	if filter.ActorType != "" {
		stmt = stmt.Where("actor_type = ?", filter.ActorType)
	}
	if filter.ResourceType != "" {
		stmt = stmt.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		stmt = stmt.Where("resource_id = ?", filter.ResourceID)
	}
	if filter.RequestID != "" {
		stmt = stmt.Where("request_id = ?", filter.RequestID)
	}
	if filter.CreatedFrom != nil {
		stmt = stmt.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		stmt = stmt.Where("created_at <= ?", *filter.CreatedTo)
	}
	if cursor != nil {
		stmt = stmt.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			cursor.CreatedAt,
			cursor.CreatedAt,
			cursor.ID,
		)
	}

	var logs []AuditLog
	if err := stmt.Order("created_at desc, id desc").Limit(pageSize + 1).Find(&logs).Error; err != nil {
		return ListResponse{}, err
	}

	resp := ListResponse{}
	items := make([]*AuditLog, len(logs))
	for i := range logs {
		items[i] = &logs[i]
	}
	pageInfo := pagination.BuildCursorPageInfo(items, int32(pageSize), func(item *AuditLog) string {
		token, err := pagination.EncodeCursor(pagination.Cursor{
			ID:        item.ID.String(),
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
		if err != nil {
			return ""
		}
		return token
	})
	if pageInfo != nil {
		resp.PageInfo = *pageInfo
		if pageInfo.HasMore && len(logs) > pageSize {
			logs = logs[:pageSize]
		}
	}
	resp.Logs = logs
	return resp, nil
}

func decodeCursor(token string) (*listCursor, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil
	}
	decoded, err := pagination.DecodeCursor(raw)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	if decoded == nil || decoded.ID == "" || decoded.CreatedAt == "" {
		return nil, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, decoded.CreatedAt)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	parsedID, err := uuid.Parse(decoded.ID)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	return &listCursor{
		ID:        parsedID,
		CreatedAt: parsedTime,
	}, nil
}
