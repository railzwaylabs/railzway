package server

import "github.com/railzwaylabs/railzway/pkg/db/pagination"

// Generic Swagger response envelopes to match API shape.
type DataResponse struct {
	Data any `json:"data"`
}

type ListResponse struct {
	Data     any                  `json:"data"`
	PageInfo *pagination.PageInfo `json:"page_info,omitempty"`
}
