package extrahop

import (
	"context"
	"fmt"
)

// Device represents an ExtraHop device.
type Device struct {
	ID              int64   `json:"id,omitempty"`
	ParentID        *int64  `json:"parent_id,omitempty"`
	NodeID          *int64  `json:"node_id,omitempty"`
	ExtrahopID      string  `json:"extrahop_id,omitempty"`
	Description     *string `json:"description,omitempty"`
	UserModTime     int64   `json:"user_mod_time,omitempty"`
	ModTime         int64   `json:"mod_time,omitempty"`
	DiscoverTime    int64   `json:"discover_time,omitempty"`
	VLANID          int     `json:"vlanid,omitempty"`
	MACAddr         string  `json:"macaddr,omitempty"`
	Vendor          string  `json:"vendor,omitempty"`
	IsL3            bool    `json:"is_l3,omitempty"`
	IPAddr4         *string `json:"ipaddr4,omitempty"`
	IPAddr6         *string `json:"ipaddr6,omitempty"`
	DeviceClass     string  `json:"device_class,omitempty"`
	DefaultName     string  `json:"default_name,omitempty"`
	CustomName      *string `json:"custom_name,omitempty"`
	CDPName         string  `json:"cdp_name,omitempty"`
	DHCPName        string  `json:"dhcp_name,omitempty"`
	NetBIOSName     string  `json:"netbios_name,omitempty"`
	DNSName         string  `json:"dns_name,omitempty"`
	CustomType      string  `json:"custom_type,omitempty"`
	AnalysisLevel   int     `json:"analysis_level,omitempty"`
	Analysis        string  `json:"analysis,omitempty"`
	AutoRole        string  `json:"auto_role,omitempty"`
	CustomMake      string  `json:"custom_make,omitempty"`
	CustomModel     string  `json:"custom_model,omitempty"`
	CriticalityLevel *int   `json:"criticality_level,omitempty"`
	DisplayName     string  `json:"display_name,omitempty"`
	CloudAccount    string  `json:"cloud_account,omitempty"`
	CloudInstanceID string  `json:"cloud_instance_id,omitempty"`
}

// DeviceSearchRequest specifies parameters for searching devices.
type DeviceSearchRequest struct {
	Filter *DeviceFilter `json:"filter,omitempty"`
	Limit  int           `json:"limit,omitempty"`
	Offset int           `json:"offset,omitempty"`
}

// DeviceFilter specifies filter criteria for device search.
type DeviceFilter struct {
	Field    string         `json:"field,omitempty"`
	Operand  interface{}    `json:"operand,omitempty"`
	Operator string         `json:"operator,omitempty"`
	Rules    []DeviceFilter `json:"rules,omitempty"`
}

// DeviceService handles communication with the device-related endpoints.
type DeviceService struct {
	client *Client
}

// List retrieves all devices. Use search for filtered results.
func (s *DeviceService) List(ctx context.Context, params *DeviceListParams) ([]*Device, error) {
	path := "/devices"
	if params != nil {
		path += params.encode()
	}
	var devices []*Device
	_, err := s.client.get(ctx, path, &devices)
	return devices, err
}

// DeviceListParams are optional parameters for List.
type DeviceListParams struct {
	ActiveFrom int64  `url:"active_from,omitempty"`
	ActiveUntil int64 `url:"active_until,omitempty"`
	Limit      int    `url:"limit,omitempty"`
	Offset     int    `url:"offset,omitempty"`
	SearchType string `url:"search_type,omitempty"`
	Value      string `url:"value,omitempty"`
}

func (p *DeviceListParams) encode() string {
	if p == nil {
		return ""
	}
	q := make([]string, 0)
	if p.ActiveFrom != 0 {
		q = append(q, fmt.Sprintf("active_from=%d", p.ActiveFrom))
	}
	if p.ActiveUntil != 0 {
		q = append(q, fmt.Sprintf("active_until=%d", p.ActiveUntil))
	}
	if p.Limit != 0 {
		q = append(q, fmt.Sprintf("limit=%d", p.Limit))
	}
	if p.Offset != 0 {
		q = append(q, fmt.Sprintf("offset=%d", p.Offset))
	}
	if p.SearchType != "" {
		q = append(q, "search_type="+p.SearchType)
	}
	if p.Value != "" {
		q = append(q, "value="+p.Value)
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + joinParams(q)
}

// Get retrieves a specific device by ID.
func (s *DeviceService) Get(ctx context.Context, id int64) (*Device, error) {
	var device Device
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d", id), &device)
	return &device, err
}

// Update modifies a specific device.
func (s *DeviceService) Update(ctx context.Context, id int64, device *Device) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/devices/%d", id), device)
	return err
}

// Search finds devices matching the given criteria.
func (s *DeviceService) Search(ctx context.Context, req *DeviceSearchRequest) ([]*Device, error) {
	var devices []*Device
	_, err := s.client.post(ctx, "/devices/search", req, &devices)
	return devices, err
}

// GetActivity retrieves activity for a device.
func (s *DeviceService) GetActivity(ctx context.Context, id int64) ([]map[string]interface{}, error) {
	var activity []map[string]interface{}
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d/activity", id), &activity)
	return activity, err
}

// ListAlerts retrieves alerts assigned to a device.
func (s *DeviceService) ListAlerts(ctx context.Context, id int64) ([]*Alert, error) {
	var alerts []*Alert
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d/alerts", id), &alerts)
	return alerts, err
}

// ListTags retrieves tags assigned to a device.
func (s *DeviceService) ListTags(ctx context.Context, id int64) ([]*Tag, error) {
	var tags []*Tag
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d/tags", id), &tags)
	return tags, err
}

// AssignTag assigns a tag to a device.
func (s *DeviceService) AssignTag(ctx context.Context, deviceID, tagID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/devices/%d/tags/%d", deviceID, tagID), nil, nil)
	return err
}

// UnassignTag removes a tag from a device.
func (s *DeviceService) UnassignTag(ctx context.Context, deviceID, tagID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/devices/%d/tags/%d", deviceID, tagID))
	return err
}

// ListDeviceGroups retrieves device groups for a device.
func (s *DeviceService) ListDeviceGroups(ctx context.Context, id int64) ([]*DeviceGroup, error) {
	var groups []*DeviceGroup
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d/devicegroups", id), &groups)
	return groups, err
}

// ListSoftware retrieves software for a device.
func (s *DeviceService) ListSoftware(ctx context.Context, id int64) ([]*Software, error) {
	var software []*Software
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d/software", id), &software)
	return software, err
}

// ListIPAddrs retrieves IP addresses for a device.
func (s *DeviceService) ListIPAddrs(ctx context.Context, id int64) ([]map[string]interface{}, error) {
	var addrs []map[string]interface{}
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d/ipaddrs", id), &addrs)
	return addrs, err
}

// ListDNSNames retrieves DNS names for a device.
func (s *DeviceService) ListDNSNames(ctx context.Context, id int64) ([]string, error) {
	var names []string
	_, err := s.client.get(ctx, fmt.Sprintf("/devices/%d/dnsnames", id), &names)
	return names, err
}

func joinParams(params []string) string {
	result := ""
	for i, p := range params {
		if i > 0 {
			result += "&"
		}
		result += p
	}
	return result
}
