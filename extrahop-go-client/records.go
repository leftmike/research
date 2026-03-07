package extrahop

import (
	"context"
	"fmt"
)

// RecordSearchRequest specifies parameters for searching records.
type RecordSearchRequest struct {
	From    int64         `json:"from,omitempty"`
	Until   int64         `json:"until,omitempty"`
	Limit   int           `json:"limit,omitempty"`
	Offset  int           `json:"offset,omitempty"`
	Filter  *RecordFilter `json:"filter,omitempty"`
	Types   []string      `json:"types,omitempty"`
	SortBy  []RecordSort  `json:"sort,omitempty"`
	Context string        `json:"context_ttl,omitempty"`
}

// RecordFilter specifies filter criteria for record search.
type RecordFilter struct {
	Field    string         `json:"field,omitempty"`
	Operand  interface{}    `json:"operand,omitempty"`
	Operator string         `json:"operator,omitempty"`
	Rules    []RecordFilter `json:"rules,omitempty"`
}

// RecordSort specifies sort criteria.
type RecordSort struct {
	Direction string `json:"direction,omitempty"`
	Field     string `json:"field,omitempty"`
}

// RecordSearchResponse contains the results of a record search.
type RecordSearchResponse struct {
	Records []map[string]interface{} `json:"records,omitempty"`
	Total   int64                    `json:"total,omitempty"`
	Cursor  string                   `json:"cursor,omitempty"`
	From    int64                    `json:"from,omitempty"`
	Until   int64                    `json:"until,omitempty"`
	Offset  int                      `json:"offset,omitempty"`
	Warnings map[string]interface{}  `json:"warnings,omitempty"`
}

// RecordService handles communication with record-related endpoints.
type RecordService struct {
	client *Client
}

// Search performs a record search.
func (s *RecordService) Search(ctx context.Context, req *RecordSearchRequest) (*RecordSearchResponse, error) {
	var resp RecordSearchResponse
	_, err := s.client.post(ctx, "/records/search", req, &resp)
	return &resp, err
}

// GetCursor retrieves the next page of record results.
func (s *RecordService) GetCursor(ctx context.Context, cursor string) (*RecordSearchResponse, error) {
	var resp RecordSearchResponse
	_, err := s.client.get(ctx, fmt.Sprintf("/records/cursor/%s", cursor), &resp)
	return &resp, err
}
