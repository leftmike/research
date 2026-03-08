package extrahop

import "context"

// MetricRequest is the body for a targeted metric query (POST /metrics).
//
// TKG-driven usage: set ObjectType to "device" and ObjectIDs to the list of
// device IDs returned by a prior detection or device search. Then specify the
// MetricSpecs relevant to the protocol being investigated.
type MetricRequest struct {
	// Cycle is the metric roll-up period: "1sec", "30sec", "5min", "1hr", "24hr".
	Cycle string `json:"cycle,omitempty"`

	// From is the start of the query window in Unix milliseconds.
	// Negative values are relative to Until (e.g. -1800000 = last 30 minutes).
	From int64 `json:"from,omitempty"`

	// Until is the end of the query window in Unix milliseconds (0 = now).
	Until int64 `json:"until,omitempty"`

	// ObjectType scopes the query: "device", "device_group", "application",
	// "network", or "system".
	ObjectType string `json:"object_type,omitempty"`

	// ObjectIDs is the list of object IDs to query metrics for.
	ObjectIDs []int64 `json:"object_ids,omitempty"`

	// MetricSpecs identifies the specific metrics to retrieve.
	MetricSpecs []MetricSpec `json:"metric_specs,omitempty"`
}

// MetricSpec identifies a single metric to retrieve.
type MetricSpec struct {
	// Name is the fully-qualified metric name (e.g. "extrahop.device.net_detail").
	Name string `json:"name,omitempty"`

	// CalcType is the aggregation type: "sum", "mean", "percentile", "max", "min".
	CalcType string `json:"calc_type,omitempty"`

	// Percentiles lists percentile values to retrieve when CalcType is "percentile".
	Percentiles []float64 `json:"percentiles,omitempty"`

	// KeyPair targets a specific breakdown key (e.g. a peer device or port).
	KeyPair *MetricKeyPair `json:"key1,omitempty"`
}

// MetricKeyPair targets a specific key dimension in a detail metric.
type MetricKeyPair struct {
	// Key is the primary key value (e.g. an IP address string or device ID).
	Key interface{} `json:"key,omitempty"`

	// Key2 is the secondary key value for two-dimensional detail metrics.
	Key2 interface{} `json:"key2,omitempty"`
}

// MetricResponse contains the results of a POST /metrics query.
type MetricResponse struct {
	// Cycle is the roll-up period of the returned data.
	Cycle string `json:"cycle,omitempty"`

	// NodeID is the sensor that produced the data.
	NodeID int64 `json:"node_id,omitempty"`

	// Clock is the server clock at query time in Unix milliseconds.
	Clock int64 `json:"clock,omitempty"`

	// From is the actual start of the returned data window.
	From int64 `json:"from,omitempty"`

	// Until is the actual end of the returned data window.
	Until int64 `json:"until,omitempty"`

	// Stats contains per-object, per-time-interval metric values.
	Stats []MetricStat `json:"stats,omitempty"`

	// XID is a cursor for paginating large result sets via /metrics/next/{xid}.
	XID int64 `json:"xid,omitempty"`
}

// MetricStat is one row of metric data for a specific object and time interval.
type MetricStat struct {
	// OID is the object ID these stats belong to.
	OID int64 `json:"oid,omitempty"`

	// Time is the interval start time in Unix milliseconds.
	Time int64 `json:"time,omitempty"`

	// Duration is the interval length in milliseconds.
	Duration int64 `json:"duration,omitempty"`

	// Values contains the metric values, one entry per MetricSpec in the request.
	// Each entry may be a scalar, a count+sum map, or a percentile breakdown.
	Values []interface{} `json:"values,omitempty"`
}

// MetricService implements the live metric enrichment API.
type MetricService struct {
	client *Client
}

// Query performs a targeted metric query against one or more objects.
// Use ObjectType + ObjectIDs to scope the query to specific devices, then
// MetricSpecs to select the protocol metrics of interest.
//
// API: POST /metrics
func (s *MetricService) Query(ctx context.Context, req *MetricRequest) (*MetricResponse, error) {
	var out MetricResponse
	if err := s.client.post(ctx, "/metrics", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
