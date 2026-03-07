package extrahop

import (
	"context"
	"fmt"
)

// Trigger represents an ExtraHop trigger.
type Trigger struct {
	ID          int64    `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Author      string   `json:"author,omitempty"`
	Description string   `json:"description,omitempty"`
	Disabled    bool     `json:"disabled,omitempty"`
	Debug       bool     `json:"debug,omitempty"`
	Events      []string `json:"events,omitempty"`
	Script      string   `json:"script,omitempty"`
	Hints       map[string]interface{} `json:"hints,omitempty"`
	ModTime     int64    `json:"mod_time,omitempty"`
}

// TriggerService handles communication with trigger-related endpoints.
type TriggerService struct {
	client *Client
}

// List retrieves all triggers.
func (s *TriggerService) List(ctx context.Context) ([]*Trigger, error) {
	var triggers []*Trigger
	_, err := s.client.get(ctx, "/triggers", &triggers)
	return triggers, err
}

// Create creates a new trigger.
func (s *TriggerService) Create(ctx context.Context, trigger *Trigger) error {
	_, err := s.client.post(ctx, "/triggers", trigger, nil)
	return err
}

// Get retrieves a specific trigger.
func (s *TriggerService) Get(ctx context.Context, id int64) (*Trigger, error) {
	var trigger Trigger
	_, err := s.client.get(ctx, fmt.Sprintf("/triggers/%d", id), &trigger)
	return &trigger, err
}

// Update modifies a trigger.
func (s *TriggerService) Update(ctx context.Context, id int64, trigger *Trigger) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/triggers/%d", id), trigger)
	return err
}

// Delete deletes a trigger.
func (s *TriggerService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/triggers/%d", id))
	return err
}

// ListDevices retrieves devices assigned to a trigger.
func (s *TriggerService) ListDevices(ctx context.Context, id int64) ([]*Device, error) {
	var devices []*Device
	_, err := s.client.get(ctx, fmt.Sprintf("/triggers/%d/devices", id), &devices)
	return devices, err
}

// AssignDevice assigns a device to a trigger.
func (s *TriggerService) AssignDevice(ctx context.Context, triggerID, deviceID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/triggers/%d/devices/%d", triggerID, deviceID), nil, nil)
	return err
}

// UnassignDevice removes a device from a trigger.
func (s *TriggerService) UnassignDevice(ctx context.Context, triggerID, deviceID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/triggers/%d/devices/%d", triggerID, deviceID))
	return err
}

// ListDeviceGroups retrieves device groups assigned to a trigger.
func (s *TriggerService) ListDeviceGroups(ctx context.Context, id int64) ([]*DeviceGroup, error) {
	var groups []*DeviceGroup
	_, err := s.client.get(ctx, fmt.Sprintf("/triggers/%d/devicegroups", id), &groups)
	return groups, err
}

// AssignDeviceGroup assigns a device group to a trigger.
func (s *TriggerService) AssignDeviceGroup(ctx context.Context, triggerID, groupID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/triggers/%d/devicegroups/%d", triggerID, groupID), nil, nil)
	return err
}

// UnassignDeviceGroup removes a device group from a trigger.
func (s *TriggerService) UnassignDeviceGroup(ctx context.Context, triggerID, groupID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/triggers/%d/devicegroups/%d", triggerID, groupID))
	return err
}

// SendExternalData sends external data to triggers.
func (s *TriggerService) SendExternalData(ctx context.Context, data interface{}) error {
	_, err := s.client.post(ctx, "/triggers/externaldata", data, nil)
	return err
}
