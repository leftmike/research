package extrahop

import (
	"context"
	"fmt"
)

// Detection represents an ExtraHop detection.
type Detection struct {
	ID              int64                  `json:"id,omitempty"`
	Type            string                 `json:"type,omitempty"`
	Title           string                 `json:"title,omitempty"`
	Description     string                 `json:"description,omitempty"`
	RiskScore       *int                   `json:"risk_score,omitempty"`
	Assignee        string                 `json:"assignee,omitempty"`
	Status          string                 `json:"status,omitempty"`
	Resolution      string                 `json:"resolution,omitempty"`
	Participants    []DetectionParticipant `json:"participants,omitempty"`
	Categories      []string               `json:"categories,omitempty"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
	StartTime       int64                  `json:"start_time,omitempty"`
	EndTime         int64                  `json:"end_time,omitempty"`
	UpdateTime      int64                  `json:"update_time,omitempty"`
	ModTime         int64                  `json:"mod_time,omitempty"`
	UserModTime     int64                  `json:"user_mod_time,omitempty"`
	Ticket          *DetectionTicket       `json:"ticket,omitempty"`
	ApplianceID     int64                  `json:"appliance_id,omitempty"`
	MitreTechniques []MitreTechnique       `json:"mitre_techniques,omitempty"`
}

// DetectionParticipant is a participant in a detection.
type DetectionParticipant struct {
	ObjectType string `json:"object_type,omitempty"`
	ObjectID   int64  `json:"object_id,omitempty"`
	Role       string `json:"role,omitempty"`
}

// DetectionTicket represents ticket information for a detection.
type DetectionTicket struct {
	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

// MitreTechnique represents a MITRE ATT&CK technique.
type MitreTechnique struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// DetectionSearchRequest specifies parameters for searching detections.
type DetectionSearchRequest struct {
	Filter  *DetectionFilter `json:"filter,omitempty"`
	Limit   int              `json:"limit,omitempty"`
	Offset  int              `json:"offset,omitempty"`
	From    int64            `json:"from,omitempty"`
	Until   int64            `json:"until,omitempty"`
	SortBy  []DetectionSort  `json:"sort,omitempty"`
}

// DetectionFilter specifies filter criteria for detection search.
type DetectionFilter struct {
	Field    string            `json:"field,omitempty"`
	Operand  interface{}       `json:"operand,omitempty"`
	Operator string            `json:"operator,omitempty"`
	Rules    []DetectionFilter `json:"rules,omitempty"`
}

// DetectionSort specifies sort criteria.
type DetectionSort struct {
	Direction string `json:"direction,omitempty"`
	Field     string `json:"field,omitempty"`
}

// DetectionNote represents notes on a detection.
type DetectionNote struct {
	Author   string `json:"author,omitempty"`
	Note     string `json:"note,omitempty"`
	ModTime  int64  `json:"mod_time,omitempty"`
}

// DetectionHidingRule represents a detection hiding rule.
type DetectionHidingRule struct {
	ID          int64                  `json:"id,omitempty"`
	Author      string                 `json:"author,omitempty"`
	Enabled     bool                   `json:"enabled,omitempty"`
	Expiration  *int64                 `json:"expiration,omitempty"`
	Description string                 `json:"description,omitempty"`
	DetectionType string              `json:"detection_type,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// DetectionFormat represents a custom detection format.
type DetectionFormat struct {
	ID          string `json:"id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Type        string `json:"type,omitempty"`
}

// DetectionService handles communication with detection-related endpoints.
type DetectionService struct {
	client *Client
}

// List retrieves all detections.
func (s *DetectionService) List(ctx context.Context) ([]*Detection, error) {
	var detections []*Detection
	_, err := s.client.get(ctx, "/detections", &detections)
	return detections, err
}

// Get retrieves a specific detection by ID.
func (s *DetectionService) Get(ctx context.Context, id int64) (*Detection, error) {
	var detection Detection
	_, err := s.client.get(ctx, fmt.Sprintf("/detections/%d", id), &detection)
	return &detection, err
}

// Update modifies a specific detection.
func (s *DetectionService) Update(ctx context.Context, id int64, detection *Detection) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/detections/%d", id), detection)
	return err
}

// Search finds detections matching the given criteria.
func (s *DetectionService) Search(ctx context.Context, req *DetectionSearchRequest) ([]*Detection, error) {
	var detections []*Detection
	_, err := s.client.post(ctx, "/detections/search", req, &detections)
	return detections, err
}

// GetNotes retrieves notes for a detection.
func (s *DetectionService) GetNotes(ctx context.Context, id int64) (*DetectionNote, error) {
	var note DetectionNote
	_, err := s.client.get(ctx, fmt.Sprintf("/detections/%d/notes", id), &note)
	return &note, err
}

// SetNotes sets notes on a detection.
func (s *DetectionService) SetNotes(ctx context.Context, id int64, note *DetectionNote) error {
	_, err := s.client.put(ctx, fmt.Sprintf("/detections/%d/notes", id), note, nil)
	return err
}

// DeleteNotes deletes notes from a detection.
func (s *DetectionService) DeleteNotes(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/detections/%d/notes", id))
	return err
}

// GetRelated retrieves related detections.
func (s *DetectionService) GetRelated(ctx context.Context, id int64) ([]*Detection, error) {
	var detections []*Detection
	_, err := s.client.get(ctx, fmt.Sprintf("/detections/%d/related", id), &detections)
	return detections, err
}

// ListInvestigations retrieves investigations for a detection.
func (s *DetectionService) ListInvestigations(ctx context.Context, id int64) ([]*Investigation, error) {
	var investigations []*Investigation
	_, err := s.client.get(ctx, fmt.Sprintf("/detections/%d/investigations", id), &investigations)
	return investigations, err
}

// UpdateTickets bulk-updates ticket information for detections.
func (s *DetectionService) UpdateTickets(ctx context.Context, tickets interface{}) error {
	_, err := s.client.patch(ctx, "/detections/tickets", tickets)
	return err
}

// ListFormats retrieves detection formats.
func (s *DetectionService) ListFormats(ctx context.Context) ([]*DetectionFormat, error) {
	var formats []*DetectionFormat
	_, err := s.client.get(ctx, "/detections/formats", &formats)
	return formats, err
}

// CreateFormat creates a detection format.
func (s *DetectionService) CreateFormat(ctx context.Context, format *DetectionFormat) error {
	_, err := s.client.post(ctx, "/detections/formats", format, nil)
	return err
}

// GetFormat retrieves a specific detection format.
func (s *DetectionService) GetFormat(ctx context.Context, id string) (*DetectionFormat, error) {
	var format DetectionFormat
	_, err := s.client.get(ctx, fmt.Sprintf("/detections/formats/%s", id), &format)
	return &format, err
}

// UpdateFormat modifies a detection format.
func (s *DetectionService) UpdateFormat(ctx context.Context, id string, format *DetectionFormat) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/detections/formats/%s", id), format)
	return err
}

// DeleteFormat deletes a detection format.
func (s *DetectionService) DeleteFormat(ctx context.Context, id string) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/detections/formats/%s", id))
	return err
}

// ListHidingRules retrieves detection hiding rules.
func (s *DetectionService) ListHidingRules(ctx context.Context) ([]*DetectionHidingRule, error) {
	var rules []*DetectionHidingRule
	_, err := s.client.get(ctx, "/detections/rules/hiding", &rules)
	return rules, err
}

// CreateHidingRule creates a detection hiding rule.
func (s *DetectionService) CreateHidingRule(ctx context.Context, rule *DetectionHidingRule) error {
	_, err := s.client.post(ctx, "/detections/rules/hiding", rule, nil)
	return err
}

// GetHidingRule retrieves a detection hiding rule.
func (s *DetectionService) GetHidingRule(ctx context.Context, id int64) (*DetectionHidingRule, error) {
	var rule DetectionHidingRule
	_, err := s.client.get(ctx, fmt.Sprintf("/detections/rules/hiding/%d", id), &rule)
	return &rule, err
}

// UpdateHidingRule modifies a detection hiding rule.
func (s *DetectionService) UpdateHidingRule(ctx context.Context, id int64, rule *DetectionHidingRule) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/detections/rules/hiding/%d", id), rule)
	return err
}

// DeleteHidingRule deletes a detection hiding rule.
func (s *DetectionService) DeleteHidingRule(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/detections/rules/hiding/%d", id))
	return err
}
