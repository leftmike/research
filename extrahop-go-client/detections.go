package extrahop

import (
	"context"
	"fmt"
	"strings"
)

// Detection represents an ExtraHop detection event with full participant detail.
type Detection struct {
	// ID is the unique detection identifier.
	ID int64 `json:"id,omitempty"`

	// Type is the detection type identifier (e.g. "NETWORK_SCAN").
	Type string `json:"type,omitempty"`

	// Title is the human-readable detection title.
	Title string `json:"title,omitempty"`

	// Description provides narrative detail about the detection.
	Description string `json:"description,omitempty"`

	// RiskScore is 0–99; higher values indicate greater risk.
	RiskScore *int `json:"risk_score,omitempty"`

	// Status is "new", "in_progress", or "closed".
	Status string `json:"status,omitempty"`

	// Resolution is "acknowledged", "action_taken", or "no_action_taken".
	Resolution string `json:"resolution,omitempty"`

	// Assignee is the username of the assigned analyst.
	Assignee string `json:"assignee,omitempty"`

	// Participants lists every device or user involved in the detection.
	Participants []DetectionParticipant `json:"participants,omitempty"`

	// Categories contains MITRE or custom category labels.
	Categories []string `json:"categories,omitempty"`

	// MitreTechniques lists the MITRE ATT&CK techniques observed.
	MitreTechniques []MitreTechnique `json:"mitre_techniques,omitempty"`

	// Properties contains detection-type-specific key-value pairs.
	Properties map[string]interface{} `json:"properties,omitempty"`

	// Ticket contains external ticketing system metadata.
	Ticket *DetectionTicket `json:"ticket,omitempty"`

	// StartTime is the detection start time in Unix milliseconds.
	StartTime int64 `json:"start_time,omitempty"`

	// EndTime is the detection end time in Unix milliseconds (0 = ongoing).
	EndTime int64 `json:"end_time,omitempty"`

	// UpdateTime is when the detection was last updated, in Unix milliseconds.
	UpdateTime int64 `json:"update_time,omitempty"`

	// ApplianceID identifies which sensor observed the detection.
	ApplianceID int64 `json:"appliance_id,omitempty"`
}

// DetectionParticipant is a device or user involved in a detection.
type DetectionParticipant struct {
	// ObjectType is "device" or "user".
	ObjectType string `json:"object_type,omitempty"`

	// ObjectID is the device or user ID.
	ObjectID int64 `json:"object_id,omitempty"`

	// Role is "offender" or "victim".
	Role string `json:"role,omitempty"`

	// Hostname is the resolved hostname at detection time (may be empty).
	Hostname string `json:"hostname,omitempty"`
}

// DetectionTicket holds external ticket information linked to a detection.
type DetectionTicket struct {
	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

// MitreTechnique represents a MITRE ATT&CK technique observed in a detection.
type MitreTechnique struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// DetectionListParams filters the active detections returned by List.
// All fields are optional; zero values are omitted from the query string.
type DetectionListParams struct {
	// Limit caps the number of detections returned (default 100, max 1000).
	Limit int

	// Offset is the number of detections to skip for pagination.
	Offset int

	// From is the earliest detection start time to include, in Unix milliseconds.
	// Use negative values for relative time (e.g. -3600000 = last hour).
	From int64

	// Until is the latest detection start time to include, in Unix milliseconds.
	// 0 means "now".
	Until int64

	// Statuses filters by detection status. Valid values: "new", "in_progress", "closed".
	Statuses []string
}

func (p *DetectionListParams) queryString() string {
	if p == nil {
		return ""
	}
	var parts []string
	if p.Limit > 0 {
		parts = append(parts, fmt.Sprintf("limit=%d", p.Limit))
	}
	if p.Offset > 0 {
		parts = append(parts, fmt.Sprintf("offset=%d", p.Offset))
	}
	if p.From != 0 {
		parts = append(parts, fmt.Sprintf("from=%d", p.From))
	}
	if p.Until != 0 {
		parts = append(parts, fmt.Sprintf("until=%d", p.Until))
	}
	for _, s := range p.Statuses {
		parts = append(parts, "filter="+s)
	}
	if len(parts) == 0 {
		return ""
	}
	return "?" + strings.Join(parts, "&")
}

// DetectionService implements the detection ingestion APIs.
type DetectionService struct {
	client *Client
}

// List returns active detections. Pass nil params to use server defaults.
//
// API: GET /detections
func (s *DetectionService) List(ctx context.Context, params *DetectionListParams) ([]*Detection, error) {
	path := "/detections" + params.queryString()
	var out []*Detection
	if err := s.client.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns full detail for a single detection, including all participants.
//
// API: GET /detections/{id}
func (s *DetectionService) Get(ctx context.Context, id int64) (*Detection, error) {
	var out Detection
	if err := s.client.get(ctx, fmt.Sprintf("/detections/%d", id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
