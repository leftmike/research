package extrahop

import (
	"context"
	"fmt"
)

// Network represents an ExtraHop network.
type Network struct {
	ID          int64  `json:"id,omitempty"`
	NodeID      *int64 `json:"node_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Idle        bool   `json:"idle,omitempty"`
	ModTime     int64  `json:"mod_time,omitempty"`
}

// NetworkLocality represents a network locality entry.
type NetworkLocality struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Network     string `json:"network,omitempty"`
	Description string `json:"description,omitempty"`
	External    bool   `json:"external,omitempty"`
	ModTime     int64  `json:"mod_time,omitempty"`
}

// NetworkService handles communication with network-related endpoints.
type NetworkService struct {
	client *Client
}

// List retrieves all networks.
func (s *NetworkService) List(ctx context.Context) ([]*Network, error) {
	var networks []*Network
	_, err := s.client.get(ctx, "/networks", &networks)
	return networks, err
}

// Get retrieves a specific network.
func (s *NetworkService) Get(ctx context.Context, id int64) (*Network, error) {
	var network Network
	_, err := s.client.get(ctx, fmt.Sprintf("/networks/%d", id), &network)
	return &network, err
}

// Update modifies a specific network.
func (s *NetworkService) Update(ctx context.Context, id int64, network *Network) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/networks/%d", id), network)
	return err
}

// ListAlerts retrieves alerts assigned to a network.
func (s *NetworkService) ListAlerts(ctx context.Context, id int64) ([]*Alert, error) {
	var alerts []*Alert
	_, err := s.client.get(ctx, fmt.Sprintf("/networks/%d/alerts", id), &alerts)
	return alerts, err
}

// ListVLANs retrieves VLANs for a network.
func (s *NetworkService) ListVLANs(ctx context.Context, id int64) ([]*VLAN, error) {
	var vlans []*VLAN
	_, err := s.client.get(ctx, fmt.Sprintf("/networks/%d/vlans", id), &vlans)
	return vlans, err
}

// NetworkLocalityService handles communication with network locality endpoints.
type NetworkLocalityService struct {
	client *Client
}

// List retrieves all network localities.
func (s *NetworkLocalityService) List(ctx context.Context) ([]*NetworkLocality, error) {
	var localities []*NetworkLocality
	_, err := s.client.get(ctx, "/networklocalities", &localities)
	return localities, err
}

// Create creates a new network locality.
func (s *NetworkLocalityService) Create(ctx context.Context, locality *NetworkLocality) error {
	_, err := s.client.post(ctx, "/networklocalities", locality, nil)
	return err
}

// Get retrieves a specific network locality.
func (s *NetworkLocalityService) Get(ctx context.Context, id int64) (*NetworkLocality, error) {
	var locality NetworkLocality
	_, err := s.client.get(ctx, fmt.Sprintf("/networklocalities/%d", id), &locality)
	return &locality, err
}

// Update modifies a network locality.
func (s *NetworkLocalityService) Update(ctx context.Context, id int64, locality *NetworkLocality) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/networklocalities/%d", id), locality)
	return err
}

// Delete deletes a network locality.
func (s *NetworkLocalityService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/networklocalities/%d", id))
	return err
}
