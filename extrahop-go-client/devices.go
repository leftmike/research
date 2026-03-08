package extrahop

import (
	"context"
	"fmt"
)

// Device represents an ExtraHop network device.
type Device struct {
	// ID is the unique device identifier.
	ID int64 `json:"id,omitempty"`

	// ExtrahopID is the stable ExtraHop-generated identifier.
	ExtrahopID string `json:"extrahop_id,omitempty"`

	// DisplayName is the best available name for the device (resolved at query time).
	DisplayName string `json:"display_name,omitempty"`

	// DefaultName is the system-assigned name.
	DefaultName string `json:"default_name,omitempty"`

	// CustomName is an analyst-assigned name override.
	CustomName *string `json:"custom_name,omitempty"`

	// DHCPName is the hostname observed via DHCP.
	DHCPName string `json:"dhcp_name,omitempty"`

	// DNSName is the primary DNS name for the device.
	DNSName string `json:"dns_name,omitempty"`

	// NetBIOSName is the NetBIOS hostname.
	NetBIOSName string `json:"netbios_name,omitempty"`

	// CDPName is the Cisco Discovery Protocol name.
	CDPName string `json:"cdp_name,omitempty"`

	// IPAddr4 is the primary IPv4 address.
	IPAddr4 *string `json:"ipaddr4,omitempty"`

	// IPAddr6 is the primary IPv6 address.
	IPAddr6 *string `json:"ipaddr6,omitempty"`

	// MACAddr is the MAC address (colon-separated hex).
	MACAddr string `json:"macaddr,omitempty"`

	// Vendor is the NIC/device vendor derived from the MAC OUI.
	Vendor string `json:"vendor,omitempty"`

	// DeviceClass is the detected device class (e.g. "node", "server", "remote").
	DeviceClass string `json:"device_class,omitempty"`

	// Analysis is the current analysis level ("full", "discovery", "standard").
	Analysis string `json:"analysis,omitempty"`

	// AutoRole is the system-inferred device role.
	AutoRole string `json:"auto_role,omitempty"`

	// IsL3 is true when the device is tracked by IP rather than MAC.
	IsL3 bool `json:"is_l3,omitempty"`

	// VLANID is the VLAN the device belongs to (0 = untagged).
	VLANID int `json:"vlanid,omitempty"`

	// DiscoverTime is when the device was first seen, in Unix milliseconds.
	DiscoverTime int64 `json:"discover_time,omitempty"`

	// ModTime is when the device record was last modified, in Unix milliseconds.
	ModTime int64 `json:"mod_time,omitempty"`

	// CriticalityLevel is the analyst-assigned criticality (0–3).
	CriticalityLevel *int `json:"criticality_level,omitempty"`

	// CloudAccount is the cloud provider account ID (for cloud-based devices).
	CloudAccount string `json:"cloud_account,omitempty"`

	// CloudInstanceID is the cloud instance identifier.
	CloudInstanceID string `json:"cloud_instance_id,omitempty"`

	// ParentID is the ID of the parent device (e.g. hypervisor).
	ParentID *int64 `json:"parent_id,omitempty"`
}

// DeviceSearchRequest searches for devices by IP address, MAC address, or name.
// At least one filter rule should be provided for meaningful results.
type DeviceSearchRequest struct {
	// Filter is the top-level filter expression. Combine multiple criteria
	// with operator "and" or "or" and nested Rules.
	Filter *DeviceFilter `json:"filter,omitempty"`

	// Limit caps the number of results (default 100, max 1000).
	Limit int `json:"limit,omitempty"`

	// Offset supports pagination.
	Offset int `json:"offset,omitempty"`
}

// DeviceFilter is a filter node in the device search expression tree.
//
// Leaf node example (match by IP):
//
//	DeviceFilter{Field: "ipaddr", Operator: "=", Operand: "10.0.0.1"}
//
// Compound node example (match by IP OR name):
//
//	DeviceFilter{Operator: "or", Rules: []DeviceFilter{...}}
type DeviceFilter struct {
	// Field is the device attribute to filter on.
	// Common values: "ipaddr", "macaddr", "name", "dhcp_name", "dns_name".
	Field string `json:"field,omitempty"`

	// Operand is the value to compare against (string, int, etc.).
	Operand interface{} `json:"operand,omitempty"`

	// Operator is "=", "!=", "startswith", "and", "or".
	Operator string `json:"operator,omitempty"`

	// Rules are sub-expressions for compound operators ("and", "or").
	Rules []DeviceFilter `json:"rules,omitempty"`
}

// DeviceActivity represents protocol-level activity observed on a device.
// Used for protocol validation during live enrichment.
type DeviceActivity struct {
	// Protocol is the application protocol name (e.g. "HTTP", "DNS", "SMB").
	Protocol string `json:"proto,omitempty"`

	// InBytes is the total ingress byte count for this protocol.
	InBytes int64 `json:"bytes_in,omitempty"`

	// OutBytes is the total egress byte count for this protocol.
	OutBytes int64 `json:"bytes_out,omitempty"`

	// Responses is the number of responses observed.
	Responses int64 `json:"responses,omitempty"`

	// Requests is the number of requests observed.
	Requests int64 `json:"requests,omitempty"`

	// ModTime is when this activity record was last updated, in Unix milliseconds.
	ModTime int64 `json:"mod_time,omitempty"`
}

// DeviceGroup represents a logical grouping of devices used for BKG clustering.
type DeviceGroup struct {
	// ID is the unique device group identifier.
	ID int64 `json:"id,omitempty"`

	// Name is the human-readable group name.
	Name string `json:"name,omitempty"`

	// Description is an optional description of the group's purpose.
	Description string `json:"description,omitempty"`

	// Type is "static" (manually managed) or "dynamic" (rule-based).
	Type string `json:"type,omitempty"`

	// ModTime is when the group was last modified, in Unix milliseconds.
	ModTime int64 `json:"mod_time,omitempty"`
}

// DeviceService implements device-related enrichment and search APIs.
type DeviceService struct {
	client *Client
}

// Search finds devices matching the given criteria (IP, MAC address, or name).
//
// API: POST /devices/search
func (s *DeviceService) Search(ctx context.Context, req *DeviceSearchRequest) ([]*Device, error) {
	var out []*Device
	if err := s.client.post(ctx, "/devices/search", req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns full detail for a single device by its numeric ID.
//
// API: GET /devices/{id}
func (s *DeviceService) Get(ctx context.Context, id int64) (*Device, error) {
	var out Device
	if err := s.client.get(ctx, fmt.Sprintf("/devices/%d", id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetActivity returns the protocol-level activity for a device.
// Use this to validate which protocols are active on a device (live enrichment).
//
// API: GET /devices/{id}/activity
func (s *DeviceService) GetActivity(ctx context.Context, id int64) ([]*DeviceActivity, error) {
	var out []*DeviceActivity
	if err := s.client.get(ctx, fmt.Sprintf("/devices/%d/activity", id), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListDeviceGroups returns the device groups that contain the specified device.
// Use this to determine BKG cluster membership during live enrichment.
//
// API: GET /devices/{id}/devicegroups
func (s *DeviceService) ListDeviceGroups(ctx context.Context, id int64) ([]*DeviceGroup, error) {
	var out []*DeviceGroup
	if err := s.client.get(ctx, fmt.Sprintf("/devices/%d/devicegroups", id), &out); err != nil {
		return nil, err
	}
	return out, nil
}
