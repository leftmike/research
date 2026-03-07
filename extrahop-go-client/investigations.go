package extrahop

import (
	"context"
	"fmt"
)

// Investigation represents an ExtraHop investigation.
type Investigation struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Assignee    string `json:"assignee,omitempty"`
	CreatedTime int64  `json:"created_time,omitempty"`
	UpdateTime  int64  `json:"update_time,omitempty"`
	ModTime     int64  `json:"mod_time,omitempty"`
}

// InvestigationSearchRequest specifies parameters for searching investigations.
type InvestigationSearchRequest struct {
	Filter interface{} `json:"filter,omitempty"`
	Limit  int         `json:"limit,omitempty"`
	Offset int         `json:"offset,omitempty"`
}

// InvestigationService handles communication with investigation-related endpoints.
type InvestigationService struct {
	client *Client
}

// List retrieves all investigations.
func (s *InvestigationService) List(ctx context.Context) ([]*Investigation, error) {
	var investigations []*Investigation
	_, err := s.client.get(ctx, "/investigations", &investigations)
	return investigations, err
}

// Create creates a new investigation.
func (s *InvestigationService) Create(ctx context.Context, inv *Investigation) error {
	_, err := s.client.post(ctx, "/investigations", inv, nil)
	return err
}

// Get retrieves a specific investigation.
func (s *InvestigationService) Get(ctx context.Context, id int64) (*Investigation, error) {
	var inv Investigation
	_, err := s.client.get(ctx, fmt.Sprintf("/investigations/%d", id), &inv)
	return &inv, err
}

// Update modifies an investigation.
func (s *InvestigationService) Update(ctx context.Context, id int64, inv *Investigation) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/investigations/%d", id), inv)
	return err
}

// Delete deletes an investigation.
func (s *InvestigationService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/investigations/%d", id))
	return err
}

// Search finds investigations matching the given criteria.
func (s *InvestigationService) Search(ctx context.Context, req *InvestigationSearchRequest) ([]*Investigation, error) {
	var investigations []*Investigation
	_, err := s.client.post(ctx, "/investigations/search", req, &investigations)
	return investigations, err
}
