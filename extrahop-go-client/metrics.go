package extrahop

import (
	"context"
	"fmt"
)

// MetricRequest specifies parameters for querying metrics.
type MetricRequest struct {
	Cycle       string          `json:"cycle,omitempty"`
	From        int64           `json:"from,omitempty"`
	Until       int64           `json:"until,omitempty"`
	MetricSpecs []MetricSpec    `json:"metric_specs,omitempty"`
	ObjectIDs   []int64         `json:"object_ids,omitempty"`
	ObjectType  string          `json:"object_type,omitempty"`
}

// MetricSpec specifies a metric to retrieve.
type MetricSpec struct {
	Name          string `json:"name,omitempty"`
	CalcType      string `json:"calc_type,omitempty"`
	Percentiles   []float64 `json:"percentiles,omitempty"`
}

// MetricResponse contains the results of a metric query.
type MetricResponse struct {
	Cycle  string          `json:"cycle,omitempty"`
	NodeID int64           `json:"node_id,omitempty"`
	Clock  int64           `json:"clock,omitempty"`
	From   int64           `json:"from,omitempty"`
	Until  int64           `json:"until,omitempty"`
	Stats  []MetricStat    `json:"stats,omitempty"`
	XID    int64           `json:"xid,omitempty"`
}

// MetricStat contains metric statistics.
type MetricStat struct {
	OID    int64                    `json:"oid,omitempty"`
	Time   int64                    `json:"time,omitempty"`
	Duration int64                  `json:"duration,omitempty"`
	Values []map[string]interface{} `json:"values,omitempty"`
}

// MetricTotalRequest specifies parameters for retrieving total metrics.
type MetricTotalRequest struct {
	Cycle       string       `json:"cycle,omitempty"`
	From        int64        `json:"from,omitempty"`
	Until       int64        `json:"until,omitempty"`
	MetricSpecs []MetricSpec `json:"metric_specs,omitempty"`
	ObjectIDs   []int64      `json:"object_ids,omitempty"`
	ObjectType  string       `json:"object_type,omitempty"`
}

// MetricService handles communication with metric-related endpoints.
type MetricService struct {
	client *Client
}

// Query performs a metric query.
func (s *MetricService) Query(ctx context.Context, req *MetricRequest) (*MetricResponse, error) {
	var resp MetricResponse
	_, err := s.client.post(ctx, "/metrics", req, &resp)
	return &resp, err
}

// QueryNext retrieves the next page of metric results.
func (s *MetricService) QueryNext(ctx context.Context, xid int64) (*MetricResponse, error) {
	var resp MetricResponse
	_, err := s.client.get(ctx, fmt.Sprintf("/metrics/next/%d", xid), &resp)
	return &resp, err
}

// QueryTotal performs a total metric query.
func (s *MetricService) QueryTotal(ctx context.Context, req *MetricTotalRequest) (*MetricResponse, error) {
	var resp MetricResponse
	_, err := s.client.post(ctx, "/metrics/total", req, &resp)
	return &resp, err
}

// QueryTotalByObject performs a total metric query grouped by object.
func (s *MetricService) QueryTotalByObject(ctx context.Context, req *MetricTotalRequest) (*MetricResponse, error) {
	var resp MetricResponse
	_, err := s.client.post(ctx, "/metrics/totalbyobject", req, &resp)
	return &resp, err
}
