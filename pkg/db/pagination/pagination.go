package pagination

import (
	"encoding/base64"
	"encoding/json"
)

type Pagination struct {
	PageToken string `form:"page_token"`
	PageSize  int    `form:"page_size,default=10" validate:"gte=1,lte=250"` // Min 1, Max 250
}

type Cursor struct {
	ID        string `json:"id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type PageInfo struct {
	NextPageToken      string `json:"next_page_token"`
	PreviousPageTooken string `json:"previous_page_token"`
	HasMore            bool   `json:"has_more"`
}

func EncodeCursor(data Cursor) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func DecodeCursor(data string) (*Cursor, error) {
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	var cursor Cursor
	if err := json.Unmarshal(b, &cursor); err != nil {
		return nil, err
	}

	return &cursor, nil
}

func BuildCursorPageInfo[T any](data []*T, limit int32, extractCursor func(*T) string) *PageInfo {
	if len(data) == 0 {
		return &PageInfo{HasMore: false}
	}

	hasMore := false
	if len(data) > int(limit) {
		hasMore = true
		data = data[:limit]
	}

	pageInfo := &PageInfo{
		HasMore:       hasMore,
		NextPageToken: extractCursor(data[len(data)-1]),
	}

	return pageInfo
}
