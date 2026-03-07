package extrahop

import (
	"context"
	"fmt"
)

// Appliance represents an ExtraHop appliance.
type Appliance struct {
	ID                 int64  `json:"id,omitempty"`
	UUID               string `json:"uuid,omitempty"`
	Hostname           string `json:"hostname,omitempty"`
	DisplayName        string `json:"display_name,omitempty"`
	Platform           string `json:"platform,omitempty"`
	Firmware           string `json:"firmware,omitempty"`
	NicknameOverride   string `json:"nickname,omitempty"`
	ManagedByRemote    bool   `json:"managed_by_remote,omitempty"`
	ConnectionType     string `json:"connection_type,omitempty"`
	StatusMessage      string `json:"status_message,omitempty"`
	LicenseStatus      string `json:"license_status,omitempty"`
	ModTime            int64  `json:"mod_time,omitempty"`
}

// FirmwareImage represents firmware information.
type FirmwareImage struct {
	Version  string `json:"version,omitempty"`
	Release  string `json:"release,omitempty"`
	Current  bool   `json:"current,omitempty"`
}

// ApplianceService handles communication with appliance-related endpoints.
type ApplianceService struct {
	client *Client
}

// List retrieves all appliances.
func (s *ApplianceService) List(ctx context.Context) ([]*Appliance, error) {
	var appliances []*Appliance
	_, err := s.client.get(ctx, "/appliances", &appliances)
	return appliances, err
}

// Get retrieves a specific appliance.
func (s *ApplianceService) Get(ctx context.Context, id int64) (*Appliance, error) {
	var appliance Appliance
	_, err := s.client.get(ctx, fmt.Sprintf("/appliances/%d", id), &appliance)
	return &appliance, err
}

// GetFirmwareNext retrieves the next available firmware version.
func (s *ApplianceService) GetFirmwareNext(ctx context.Context) ([]*FirmwareImage, error) {
	var images []*FirmwareImage
	_, err := s.client.get(ctx, "/appliances/firmware/next", &images)
	return images, err
}

// GetCloudServices retrieves cloud services configuration for an appliance.
func (s *ApplianceService) GetCloudServices(ctx context.Context, id int64) (map[string]interface{}, error) {
	var config map[string]interface{}
	_, err := s.client.get(ctx, fmt.Sprintf("/appliances/%d/cloudservices", id), &config)
	return config, err
}

// GetProductKey retrieves the product key for an appliance.
func (s *ApplianceService) GetProductKey(ctx context.Context, id int64) (map[string]interface{}, error) {
	var key map[string]interface{}
	_, err := s.client.get(ctx, fmt.Sprintf("/appliances/%d/productkey", id), &key)
	return key, err
}

// Delete removes an appliance (sensor) from the console.
func (s *ApplianceService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/appliances/%d", id))
	return err
}
