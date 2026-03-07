package extrahop

import (
	"context"
	"fmt"
)

// DeviceGroup represents an ExtraHop device group.
type DeviceGroup struct {
	ID          int64          `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Dynamic     bool           `json:"dynamic,omitempty"`
	Field       string         `json:"field,omitempty"`
	Value       string         `json:"value,omitempty"`
	Filter      *DeviceFilter  `json:"filter,omitempty"`
	ModTime     int64          `json:"mod_time,omitempty"`
}

// DeviceGroupService handles communication with device group endpoints.
type DeviceGroupService struct {
	client *Client
}

// List retrieves all device groups.
func (s *DeviceGroupService) List(ctx context.Context) ([]*DeviceGroup, error) {
	var groups []*DeviceGroup
	_, err := s.client.get(ctx, "/devicegroups", &groups)
	return groups, err
}

// Create creates a new device group.
func (s *DeviceGroupService) Create(ctx context.Context, group *DeviceGroup) error {
	_, err := s.client.post(ctx, "/devicegroups", group, nil)
	return err
}

// Get retrieves a specific device group.
func (s *DeviceGroupService) Get(ctx context.Context, id int64) (*DeviceGroup, error) {
	var group DeviceGroup
	_, err := s.client.get(ctx, fmt.Sprintf("/devicegroups/%d", id), &group)
	return &group, err
}

// Update modifies a device group.
func (s *DeviceGroupService) Update(ctx context.Context, id int64, group *DeviceGroup) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/devicegroups/%d", id), group)
	return err
}

// Delete deletes a device group.
func (s *DeviceGroupService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/devicegroups/%d", id))
	return err
}

// ListDevices retrieves devices in a device group.
func (s *DeviceGroupService) ListDevices(ctx context.Context, id int64) ([]*Device, error) {
	var devices []*Device
	_, err := s.client.get(ctx, fmt.Sprintf("/devicegroups/%d/devices", id), &devices)
	return devices, err
}

// AddDevice adds a device to a device group.
func (s *DeviceGroupService) AddDevice(ctx context.Context, groupID, deviceID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/devicegroups/%d/devices/%d", groupID, deviceID), nil, nil)
	return err
}

// RemoveDevice removes a device from a device group.
func (s *DeviceGroupService) RemoveDevice(ctx context.Context, groupID, deviceID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/devicegroups/%d/devices/%d", groupID, deviceID))
	return err
}

// ListAlerts retrieves alerts assigned to a device group.
func (s *DeviceGroupService) ListAlerts(ctx context.Context, id int64) ([]*Alert, error) {
	var alerts []*Alert
	_, err := s.client.get(ctx, fmt.Sprintf("/devicegroups/%d/alerts", id), &alerts)
	return alerts, err
}

// ListDashboards retrieves dashboards for a device group.
func (s *DeviceGroupService) ListDashboards(ctx context.Context, id int64) ([]*Dashboard, error) {
	var dashboards []*Dashboard
	_, err := s.client.get(ctx, fmt.Sprintf("/devicegroups/%d/dashboards", id), &dashboards)
	return dashboards, err
}

// ListTriggers retrieves triggers assigned to a device group.
func (s *DeviceGroupService) ListTriggers(ctx context.Context, id int64) ([]*Trigger, error) {
	var triggers []*Trigger
	_, err := s.client.get(ctx, fmt.Sprintf("/devicegroups/%d/triggers", id), &triggers)
	return triggers, err
}
