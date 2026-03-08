package extrahop

import "context"

// ActivityMapQueryRequest is the body for POST /activitymaps/query.
//
// An activity map query expresses a communication topology walk: starting from
// one or more origin devices or groups, follow relationships (protocol roles)
// for a given number of steps to discover connected peers.
//
// Typical agent patterns:
//
//	// 1. Walk: single origin, follow all protocols one hop out
//	req := &ActivityMapQueryRequest{
//	    From:  -1800000, // last 30 minutes
//	    Walks: []Walk{{
//	        Origins: []Origin{{ObjectType: "device", ObjectID: victimID}},
//	        Steps:   []Step{{Direction: "any"}},
//	    }},
//	}
//
//	// 2. Origins: list multiple starting points in one query
//	req := &ActivityMapQueryRequest{
//	    From:  -3600000,
//	    Walks: []Walk{{
//	        Origins: []Origin{
//	            {ObjectType: "device", ObjectID: deviceA},
//	            {ObjectType: "device", ObjectID: deviceB},
//	        },
//	        Steps: []Step{{Direction: "any"}},
//	    }},
//	}
//
//	// 3. Steps: multi-hop traversal to trace lateral movement
//	req := &ActivityMapQueryRequest{
//	    From:  -3600000,
//	    Walks: []Walk{{
//	        Origins: []Origin{{ObjectType: "device", ObjectID: pivotID}},
//	        Steps: []Step{
//	            {Relationships: []Relationship{{Protocol: "SMB", Role: "client"}}},
//	            {Relationships: []Relationship{{Protocol: "HTTP", Role: "any"}}},
//	        },
//	    }},
//	}
type ActivityMapQueryRequest struct {
	// From is the start of the observation window in Unix milliseconds.
	// Negative values are relative to Until (e.g. -1800000 = last 30 minutes).
	From int64 `json:"from,omitempty"`

	// Until is the end of the observation window in Unix milliseconds (0 = now).
	Until int64 `json:"until,omitempty"`

	// Walks describes one or more origin-step traversals to execute.
	// Results from all walks are merged into a single topology graph.
	Walks []Walk `json:"walks"`

	// Weighting determines how edge weights are calculated.
	// "bytes" (default) or "turns" (connection count).
	Weighting string `json:"weighting,omitempty"`

	// Edge controls which device pairs generate edges.
	// "peer2peer" shows direct connections; "any" includes transit hops.
	Edge string `json:"edge,omitempty"`
}

// Walk is one traversal within an activity map query, defining origins and steps.
type Walk struct {
	// Origins are the starting devices or groups for this walk.
	Origins []Origin `json:"origins"`

	// Steps defines the hops to follow from the origins.
	// An empty Steps slice returns only the origins' direct peers.
	Steps []Step `json:"steps,omitempty"`
}

// Origin is a starting point for an activity map walk.
type Origin struct {
	// ObjectType is "device", "device_group", "application", or "network".
	ObjectType string `json:"object_type"`

	// ObjectID is the numeric ID of the object.
	ObjectID int64 `json:"object_id"`
}

// Step defines the filter for one hop in a walk traversal.
// An empty Step follows all protocol relationships in both directions.
type Step struct {
	// Relationships restricts this hop to specific protocols and roles.
	// An empty slice means "follow all relationships".
	Relationships []Relationship `json:"relationships,omitempty"`

	// Direction filters connections by flow direction relative to the current node.
	// "any" (default), "in" (inbound), or "out" (outbound).
	Direction string `json:"direction,omitempty"`
}

// Relationship filters a step to a specific protocol and participation role.
type Relationship struct {
	// Protocol is the application protocol name (e.g. "HTTP", "DNS", "SMB", "SSL").
	// Use an empty string to match all protocols.
	Protocol string `json:"proto,omitempty"`

	// Role is the participation role: "client", "server", or "any".
	Role string `json:"role,omitempty"`
}

// ActivityMapQueryResponse is the result of POST /activitymaps/query.
// It describes the communication topology as a directed graph of edges.
type ActivityMapQueryResponse struct {
	// Edges is the list of observed communication relationships.
	Edges []ActivityMapEdge `json:"edges,omitempty"`
}

// ActivityMapEdge represents a single directed communication relationship
// between two endpoints observed during the query window.
type ActivityMapEdge struct {
	// From is the originating endpoint (typically the client).
	From ActivityMapEndpoint `json:"from"`

	// To is the destination endpoint (typically the server).
	To ActivityMapEndpoint `json:"to"`

	// Annotations contains observed protocol names and traffic counts for this edge.
	Annotations ActivityMapAnnotations `json:"annotations,omitempty"`
}

// ActivityMapEndpoint identifies one side of a communication edge.
type ActivityMapEndpoint struct {
	// ObjectType is "device", "device_group", "application", "network", or "external".
	ObjectType string `json:"object_type,omitempty"`

	// ObjectID is the numeric ID. 0 indicates an external/unknown endpoint.
	ObjectID int64 `json:"object_id,omitempty"`
}

// ActivityMapAnnotations describes the traffic observed on an edge.
type ActivityMapAnnotations struct {
	// Protocols is the list of application protocols observed on this edge.
	Protocols []string `json:"protocols,omitempty"`

	// Counts contains traffic volume metrics keyed by metric name (e.g. "bytes", "turns").
	Counts map[string]int64 `json:"counts,omitempty"`
}

// ActivityMapService implements the communication topology query API.
type ActivityMapService struct {
	client *Client
}

// Query executes an activity map topology query and returns the communication
// graph for the specified origins, steps, and time window.
//
// This single method covers all three topology patterns:
//   - Walks: traverse the graph hop-by-hop from one or more origins
//   - Origins: query multiple starting devices in a single request
//   - Steps: filter each hop by protocol and direction for lateral-movement tracing
//
// API: POST /activitymaps/query
func (s *ActivityMapService) Query(ctx context.Context, req *ActivityMapQueryRequest) (*ActivityMapQueryResponse, error) {
	var out ActivityMapQueryResponse
	if err := s.client.post(ctx, "/activitymaps/query", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
