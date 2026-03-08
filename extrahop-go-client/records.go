package extrahop

import "context"

// RecordSearchRequest is the body for POST /records/search.
//
// Records are transaction-level data (flows, DNS lookups, HTTP requests, etc.)
// stored in the ExtraHop recordstore. Use time window + filter to narrow the
// result to the transactions relevant to a detection or device investigation.
type RecordSearchRequest struct {
	// From is the start of the time window in Unix milliseconds.
	// Negative values are relative to Until (e.g. -3600000 = last hour).
	From int64 `json:"from,omitempty"`

	// Until is the end of the time window in Unix milliseconds (0 = now).
	Until int64 `json:"until,omitempty"`

	// Limit caps the number of records returned (default 100, max 1000).
	Limit int `json:"limit,omitempty"`

	// Offset supports cursor-based pagination when combined with a cursor.
	Offset int `json:"offset,omitempty"`

	// Filter narrows results to matching transactions.
	Filter *RecordFilter `json:"filter,omitempty"`

	// Types restricts results to specific record types
	// (e.g. "~HTTP", "~DNS", "~SSL", "~TCP").
	Types []string `json:"types,omitempty"`

	// SortBy controls result ordering.
	SortBy []RecordSort `json:"sort,omitempty"`

	// ContextTTL is an opaque cursor returned by a previous search response.
	// Pass it to continue iterating over a large result set.
	ContextTTL string `json:"context_ttl,omitempty"`
}

// RecordFilter is a filter node in the record search expression tree.
//
// Leaf node example (filter by sender IP):
//
//	RecordFilter{Field: "senderAddr", Operator: "=", Operand: "10.0.0.5"}
//
// Compound node (AND two conditions):
//
//	RecordFilter{Operator: "and", Rules: []RecordFilter{...}}
type RecordFilter struct {
	// Field is the record field to filter on (e.g. "senderAddr", "receiverPort",
	// "proto", "method", "statusCode").
	Field string `json:"field,omitempty"`

	// Operand is the comparison value.
	Operand interface{} `json:"operand,omitempty"`

	// Operator is "=", "!=", "<", ">", "startswith", "and", "or", "not".
	Operator string `json:"operator,omitempty"`

	// Rules contains sub-expressions for compound operators.
	Rules []RecordFilter `json:"rules,omitempty"`
}

// RecordSort specifies ordering for search results.
type RecordSort struct {
	// Direction is "asc" or "desc".
	Direction string `json:"direction,omitempty"`

	// Field is the record field to sort by (e.g. "timestamp").
	Field string `json:"field,omitempty"`
}

// RecordSearchResponse contains the results of POST /records/search.
type RecordSearchResponse struct {
	// Records is the list of matching transaction records.
	// Each record is a free-form map whose keys depend on the record type.
	Records []map[string]interface{} `json:"records,omitempty"`

	// Total is the total number of matching records (may exceed Limit).
	Total int64 `json:"total,omitempty"`

	// Cursor is an opaque token for fetching the next page.
	// Pass it as ContextTTL in a follow-up RecordSearchRequest.
	Cursor string `json:"cursor,omitempty"`

	// From is the actual start of the query window used by the server.
	From int64 `json:"from,omitempty"`

	// Until is the actual end of the query window used by the server.
	Until int64 `json:"until,omitempty"`
}

// RecordService implements the transaction record search API.
type RecordService struct {
	client *Client
}

// Search queries transaction records for matching flows, DNS lookups,
// HTTP requests, and other protocol data stored in the recordstore.
//
// API: POST /records/search
func (s *RecordService) Search(ctx context.Context, req *RecordSearchRequest) (*RecordSearchResponse, error) {
	var out RecordSearchResponse
	if err := s.client.post(ctx, "/records/search", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
