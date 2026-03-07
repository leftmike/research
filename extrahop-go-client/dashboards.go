package extrahop

import (
	"context"
	"fmt"
)

// Dashboard represents an ExtraHop dashboard.
type Dashboard struct {
	ID        int64    `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	Author    string   `json:"author,omitempty"`
	Comment   string   `json:"comment,omitempty"`
	Owner     string   `json:"owner,omitempty"`
	ModTime   int64    `json:"mod_time,omitempty"`
	Rights    []string `json:"rights,omitempty"`
	ShortCode string   `json:"short_code,omitempty"`
}

// SharingPolicy represents sharing permissions for a resource.
type SharingPolicy struct {
	Anyone string            `json:"anyone,omitempty"`
	Users  map[string]string `json:"users,omitempty"`
}

// DashboardService handles communication with dashboard-related endpoints.
type DashboardService struct {
	client *Client
}

// List retrieves all dashboards.
func (s *DashboardService) List(ctx context.Context) ([]*Dashboard, error) {
	var dashboards []*Dashboard
	_, err := s.client.get(ctx, "/dashboards", &dashboards)
	return dashboards, err
}

// Get retrieves a specific dashboard.
func (s *DashboardService) Get(ctx context.Context, id int64) (*Dashboard, error) {
	var dashboard Dashboard
	_, err := s.client.get(ctx, fmt.Sprintf("/dashboards/%d", id), &dashboard)
	return &dashboard, err
}

// Update modifies a dashboard.
func (s *DashboardService) Update(ctx context.Context, id int64, dashboard *Dashboard) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/dashboards/%d", id), dashboard)
	return err
}

// Delete deletes a dashboard.
func (s *DashboardService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/dashboards/%d", id))
	return err
}

// GetSharing retrieves sharing permissions for a dashboard.
func (s *DashboardService) GetSharing(ctx context.Context, id int64) (*SharingPolicy, error) {
	var policy SharingPolicy
	_, err := s.client.get(ctx, fmt.Sprintf("/dashboards/%d/sharing", id), &policy)
	return &policy, err
}

// UpdateSharing updates sharing permissions for a dashboard.
func (s *DashboardService) UpdateSharing(ctx context.Context, id int64, policy *SharingPolicy) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/dashboards/%d/sharing", id), policy)
	return err
}

// ReplaceSharing replaces sharing permissions for a dashboard.
func (s *DashboardService) ReplaceSharing(ctx context.Context, id int64, policy *SharingPolicy) error {
	_, err := s.client.put(ctx, fmt.Sprintf("/dashboards/%d/sharing", id), policy, nil)
	return err
}
