package extrahop

import (
	"context"
	"fmt"
)

// Alert represents an ExtraHop alert.
type Alert struct {
	ID                int64    `json:"id,omitempty"`
	Name              string   `json:"name,omitempty"`
	Description       string   `json:"description,omitempty"`
	Author            string   `json:"author,omitempty"`
	Disabled          bool     `json:"disabled,omitempty"`
	Type              string   `json:"type,omitempty"`
	Severity          int      `json:"severity,omitempty"`
	IntervalLength    int64    `json:"interval_length,omitempty"`
	FireCount         int64    `json:"fire_count,omitempty"`
	ModTime           int64    `json:"mod_time,omitempty"`
	RefiringInterval  int64    `json:"refiring_interval,omitempty"`
	StatName          string   `json:"stat_name,omitempty"`
	FieldName         string   `json:"field_name,omitempty"`
	FieldName2        string   `json:"field_name2,omitempty"`
	Units             string   `json:"units,omitempty"`
	IntervalType      string   `json:"interval_type,omitempty"`
	OperandType       string   `json:"operand_type,omitempty"`
	Operand           string   `json:"operand,omitempty"`
	Operator          string   `json:"operator,omitempty"`
	Param             string   `json:"param,omitempty"`
	Param2            string   `json:"param2,omitempty"`
	ObjectType        string   `json:"object_type,omitempty"`
	Protocols         []string `json:"protocols,omitempty"`
	NotifySnmp        bool     `json:"notify_snmp,omitempty"`
}

// AlertStats represents alert statistics.
type AlertStats struct {
	AlertID   int64 `json:"alert_id,omitempty"`
	FireCount int64 `json:"fire_count,omitempty"`
}

// AlertService handles communication with alert-related endpoints.
type AlertService struct {
	client *Client
}

// List retrieves all alerts.
func (s *AlertService) List(ctx context.Context) ([]*Alert, error) {
	var alerts []*Alert
	_, err := s.client.get(ctx, "/alerts", &alerts)
	return alerts, err
}

// Create creates a new alert.
func (s *AlertService) Create(ctx context.Context, alert *Alert) error {
	_, err := s.client.post(ctx, "/alerts", alert, nil)
	return err
}

// Get retrieves a specific alert.
func (s *AlertService) Get(ctx context.Context, id int64) (*Alert, error) {
	var alert Alert
	_, err := s.client.get(ctx, fmt.Sprintf("/alerts/%d", id), &alert)
	return &alert, err
}

// Update modifies a specific alert.
func (s *AlertService) Update(ctx context.Context, id int64, alert *Alert) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/alerts/%d", id), alert)
	return err
}

// Delete deletes a specific alert.
func (s *AlertService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/alerts/%d", id))
	return err
}

// GetStats retrieves alert statistics.
func (s *AlertService) GetStats(ctx context.Context, id int64) (*AlertStats, error) {
	var stats AlertStats
	_, err := s.client.get(ctx, fmt.Sprintf("/alerts/%d/stats", id), &stats)
	return &stats, err
}

// ListDevices retrieves devices assigned to an alert.
func (s *AlertService) ListDevices(ctx context.Context, id int64) ([]*Device, error) {
	var devices []*Device
	_, err := s.client.get(ctx, fmt.Sprintf("/alerts/%d/devices", id), &devices)
	return devices, err
}

// AssignDevice assigns a device to an alert.
func (s *AlertService) AssignDevice(ctx context.Context, alertID, deviceID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/alerts/%d/devices/%d", alertID, deviceID), nil, nil)
	return err
}

// UnassignDevice removes a device from an alert.
func (s *AlertService) UnassignDevice(ctx context.Context, alertID, deviceID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/alerts/%d/devices/%d", alertID, deviceID))
	return err
}

// ListDeviceGroups retrieves device groups assigned to an alert.
func (s *AlertService) ListDeviceGroups(ctx context.Context, id int64) ([]*DeviceGroup, error) {
	var groups []*DeviceGroup
	_, err := s.client.get(ctx, fmt.Sprintf("/alerts/%d/devicegroups", id), &groups)
	return groups, err
}

// AssignDeviceGroup assigns a device group to an alert.
func (s *AlertService) AssignDeviceGroup(ctx context.Context, alertID, groupID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/alerts/%d/devicegroups/%d", alertID, groupID), nil, nil)
	return err
}

// UnassignDeviceGroup removes a device group from an alert.
func (s *AlertService) UnassignDeviceGroup(ctx context.Context, alertID, groupID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/alerts/%d/devicegroups/%d", alertID, groupID))
	return err
}

// ListNetworks retrieves networks assigned to an alert.
func (s *AlertService) ListNetworks(ctx context.Context, id int64) ([]*Network, error) {
	var networks []*Network
	_, err := s.client.get(ctx, fmt.Sprintf("/alerts/%d/networks", id), &networks)
	return networks, err
}

// AssignNetwork assigns a network to an alert.
func (s *AlertService) AssignNetwork(ctx context.Context, alertID, networkID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/alerts/%d/networks/%d", alertID, networkID), nil, nil)
	return err
}

// UnassignNetwork removes a network from an alert.
func (s *AlertService) UnassignNetwork(ctx context.Context, alertID, networkID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/alerts/%d/networks/%d", alertID, networkID))
	return err
}
