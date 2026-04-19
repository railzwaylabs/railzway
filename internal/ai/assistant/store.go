package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var ErrThreadNotFound = errors.New("ai_assistant_thread_not_found")

type StoreParams struct {
	fx.In

	DB *gorm.DB
}

type ThreadStore struct {
	db *gorm.DB
}

type Token struct {
	ID             string                 `json:"id"`
	Kind           string                 `json:"kind"`
	Type           string                 `json:"type"`
	Key            string                 `json:"key"`
	Label          string                 `json:"label"`
	SecondaryLabel string                 `json:"secondary_label,omitempty"`
	ResourceID     string                 `json:"resource_id,omitempty"`
	Value          string                 `json:"value,omitempty"`
	From           string                 `json:"from,omitempty"`
	To             string                 `json:"to,omitempty"`
	Timezone       string                 `json:"timezone,omitempty"`
	Preset         string                 `json:"preset,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type MessageBlock struct {
	Type  string      `json:"type"`
	Tone  string      `json:"tone,omitempty"`
	Title string      `json:"title,omitempty"`
	Text  string      `json:"text,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type Usage struct {
	CachedContentTokens int                `json:"cached_content_tokens,omitempty"`
	Custom              map[string]float64 `json:"custom,omitempty"`
	InputAudioFiles     int                `json:"input_audio_files,omitempty"`
	InputCharacters     int                `json:"input_characters,omitempty"`
	InputImages         int                `json:"input_images,omitempty"`
	InputTokens         int                `json:"input_tokens,omitempty"`
	InputVideos         int                `json:"input_videos,omitempty"`
	OutputAudioFiles    int                `json:"output_audio_files,omitempty"`
	OutputCharacters    int                `json:"output_characters,omitempty"`
	OutputImages        int                `json:"output_images,omitempty"`
	OutputTokens        int                `json:"output_tokens,omitempty"`
	OutputVideos        int                `json:"output_videos,omitempty"`
	ThoughtsTokens      int                `json:"thoughts_tokens,omitempty"`
	TotalTokens         int                `json:"total_tokens,omitempty"`
}

type ThreadSummary struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        uuid.UUID      `json:"id"`
	ThreadID  uuid.UUID      `json:"thread_id"`
	OrgID     uuid.UUID      `json:"org_id"`
	UserID    *uuid.UUID     `json:"user_id,omitempty"`
	Role      string         `json:"role"`
	Prompt    string         `json:"prompt,omitempty"`
	Tokens    []Token        `json:"tokens,omitempty"`
	Blocks    []MessageBlock `json:"blocks,omitempty"`
	Usage     *Usage         `json:"usage,omitempty"`
	LatencyMs float64        `json:"latency_ms,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type ThreadDetail struct {
	Thread   ThreadSummary `json:"thread"`
	Messages []Message     `json:"messages"`
}

type CreateThreadInput struct {
	OrgID  uuid.UUID
	UserID uuid.UUID
	Title  string
}

type CreateMessageInput struct {
	ID        uuid.UUID
	ThreadID  uuid.UUID
	OrgID     uuid.UUID
	UserID    *uuid.UUID
	Role      string
	Prompt    string
	Tokens    []Token
	Blocks    []MessageBlock
	Usage     *Usage
	LatencyMs float64
}

type threadRow struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	UserID    uuid.UUID
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type messageRow struct {
	ID        uuid.UUID
	ThreadID  uuid.UUID
	OrgID     uuid.UUID
	UserID    *uuid.UUID
	Role      string
	Prompt    string
	Tokens    []byte
	Blocks    []byte
	Usage     []byte
	LatencyMs float64
	CreatedAt time.Time
}

func NewThreadStore(p StoreParams) *ThreadStore {
	if p.DB == nil {
		return nil
	}
	return &ThreadStore{db: p.DB}
}

func (s *ThreadStore) CreateThread(ctx context.Context, input CreateThreadInput) (ThreadSummary, error) {
	if s == nil || s.db == nil {
		return ThreadSummary{}, gorm.ErrInvalidDB
	}
	now := time.Now().UTC()
	id := uuid.New()
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "New conversation"
	}
	if err := s.db.WithContext(ctx).Exec(
		`INSERT INTO ai_assistant_threads (id, org_id, user_id, title, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, input.OrgID, input.UserID, title, now, now,
	).Error; err != nil {
		return ThreadSummary{}, err
	}
	return ThreadSummary{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *ThreadStore) ListThreads(ctx context.Context, orgID, userID uuid.UUID, limit int) ([]ThreadSummary, error) {
	if s == nil || s.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows := make([]threadRow, 0, limit)
	if err := s.db.WithContext(ctx).Raw(
		`SELECT id, org_id, user_id, title, created_at, updated_at
		 FROM ai_assistant_threads
		 WHERE org_id = ? AND user_id = ? AND archived_at IS NULL
		 ORDER BY updated_at DESC
		 LIMIT ?`,
		orgID, userID, limit,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ThreadSummary, 0, len(rows))
	for _, row := range rows {
		if row.ID == uuid.Nil {
			continue
		}
		out = append(out, ThreadSummary{
			ID:        row.ID,
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (s *ThreadStore) GetThread(ctx context.Context, orgID, userID, threadID uuid.UUID) (ThreadSummary, error) {
	if s == nil || s.db == nil {
		return ThreadSummary{}, gorm.ErrInvalidDB
	}
	var row threadRow
	if err := s.db.WithContext(ctx).Raw(
		`SELECT id, org_id, user_id, title, created_at, updated_at
		 FROM ai_assistant_threads
		 WHERE id = ? AND org_id = ? AND user_id = ? AND archived_at IS NULL
		 LIMIT 1`,
		threadID, orgID, userID,
	).Scan(&row).Error; err != nil {
		return ThreadSummary{}, err
	}
	if row.ID == uuid.Nil {
		return ThreadSummary{}, ErrThreadNotFound
	}
	return ThreadSummary{
		ID:        row.ID,
		Title:     row.Title,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *ThreadStore) GetThreadDetail(ctx context.Context, orgID, userID, threadID uuid.UUID) (ThreadDetail, error) {
	thread, err := s.GetThread(ctx, orgID, userID, threadID)
	if err != nil {
		return ThreadDetail{}, err
	}

	rows := make([]messageRow, 0, 16)
	if err := s.db.WithContext(ctx).Raw(
		`SELECT id, thread_id, org_id, user_id, role, prompt, tokens, blocks, usage, latency_ms, created_at
		 FROM ai_assistant_messages
		 WHERE thread_id = ? AND org_id = ?
		 ORDER BY created_at ASC`,
		threadID, orgID,
	).Scan(&rows).Error; err != nil {
		return ThreadDetail{}, err
	}

	messages := make([]Message, 0, len(rows))
	for _, row := range rows {
		if row.ID == uuid.Nil {
			continue
		}
		messages = append(messages, decodeMessageRow(row))
	}

	return ThreadDetail{
		Thread:   thread,
		Messages: messages,
	}, nil
}

func (s *ThreadStore) DeleteThread(ctx context.Context, orgID, userID, threadID uuid.UUID) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	result := s.db.WithContext(ctx).Exec(
		`UPDATE ai_assistant_threads
		 SET archived_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND org_id = ? AND user_id = ? AND archived_at IS NULL`,
		threadID, orgID, userID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrThreadNotFound
	}
	return nil
}

func (s *ThreadStore) CreateMessage(ctx context.Context, input CreateMessageInput) (Message, error) {
	if s == nil || s.db == nil {
		return Message{}, gorm.ErrInvalidDB
	}
	if input.ID == uuid.Nil {
		input.ID = uuid.New()
	}

	tokensJSON, err := json.Marshal(input.Tokens)
	if err != nil {
		return Message{}, err
	}
	blocksJSON, err := json.Marshal(input.Blocks)
	if err != nil {
		return Message{}, err
	}
	usageJSON, err := json.Marshal(input.Usage)
	if err != nil {
		return Message{}, err
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Exec(
		`INSERT INTO ai_assistant_messages (id, thread_id, org_id, user_id, role, prompt, tokens, blocks, usage, latency_ms, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?::jsonb, ?::jsonb, ?::jsonb, ?, ?)`,
		input.ID, input.ThreadID, input.OrgID, input.UserID, strings.TrimSpace(input.Role), strings.TrimSpace(input.Prompt), string(tokensJSON), string(blocksJSON), string(usageJSON), input.LatencyMs, now,
	).Error; err != nil {
		return Message{}, err
	}
	if err := s.db.WithContext(ctx).Exec(
		`UPDATE ai_assistant_threads
		 SET updated_at = ?
		 WHERE id = ?`,
		now, input.ThreadID,
	).Error; err != nil {
		return Message{}, err
	}

	return Message{
		ID:        input.ID,
		ThreadID:  input.ThreadID,
		OrgID:     input.OrgID,
		UserID:    input.UserID,
		Role:      strings.TrimSpace(input.Role),
		Prompt:    strings.TrimSpace(input.Prompt),
		Tokens:    input.Tokens,
		Blocks:    input.Blocks,
		Usage:     input.Usage,
		LatencyMs: input.LatencyMs,
		CreatedAt: now,
	}, nil
}

func (s *ThreadStore) UpdateThreadTitle(ctx context.Context, orgID, userID, threadID uuid.UUID, title string) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}
	result := s.db.WithContext(ctx).Exec(
		`UPDATE ai_assistant_threads
		 SET title = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND org_id = ? AND user_id = ? AND archived_at IS NULL`,
		title, threadID, orgID, userID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrThreadNotFound
	}
	return nil
}

func (s *ThreadStore) GetRecentMessages(ctx context.Context, orgID, threadID uuid.UUID, limit int) ([]Message, error) {
	if s == nil || s.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	rows := make([]messageRow, 0, limit)
	if err := s.db.WithContext(ctx).Raw(
		`SELECT id, thread_id, org_id, user_id, role, prompt, tokens, blocks, usage, latency_ms, created_at
		 FROM ai_assistant_messages
		 WHERE thread_id = ? AND org_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		threadID, orgID, limit,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Reverse to get ASC order (oldest first) for context
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	messages := make([]Message, 0, len(rows))
	for _, row := range rows {
		if row.ID != uuid.Nil {
			messages = append(messages, decodeMessageRow(row))
		}
	}
	return messages, nil
}

func decodeMessageRow(row messageRow) Message {
	message := Message{
		ID:        row.ID,
		ThreadID:  row.ThreadID,
		OrgID:     row.OrgID,
		UserID:    row.UserID,
		Role:      row.Role,
		Prompt:    row.Prompt,
		CreatedAt: row.CreatedAt,
	}
	if len(row.Tokens) > 0 {
		_ = json.Unmarshal(row.Tokens, &message.Tokens)
	}
	if len(row.Blocks) > 0 {
		_ = json.Unmarshal(row.Blocks, &message.Blocks)
	}
	if len(row.Usage) > 0 && string(row.Usage) != "null" {
		var usage Usage
		if err := json.Unmarshal(row.Usage, &usage); err == nil {
			message.Usage = &usage
		}
	}
	message.LatencyMs = row.LatencyMs
	return message
}
