package extrahop

import (
	"context"
	"fmt"
	"io"
)

// ExtraHopInfo represents system information about the ExtraHop appliance.
type ExtraHopInfo struct {
	Hostname   string `json:"hostname,omitempty"`
	MgmtIPAddr string `json:"mgmt_ipaddr,omitempty"`
	UUID       string `json:"uuid,omitempty"`
	Version    string `json:"version,omitempty"`
	Platform   string `json:"platform,omitempty"`
	DisplayHost string `json:"display_host,omitempty"`
}

// ExtraHopVersion represents the firmware version.
type ExtraHopVersion struct {
	Version string `json:"version,omitempty"`
}

// ExtraHopPlatform represents platform information.
type ExtraHopPlatform struct {
	Platform string `json:"platform,omitempty"`
}

// ExtraHopEdition represents edition information.
type ExtraHopEdition struct {
	Edition string `json:"edition,omitempty"`
}

// ExtraHopServices represents enabled services.
type ExtraHopServices struct {
	Services map[string]interface{} `json:"services,omitempty"`
}

// License represents the ExtraHop license.
type License struct {
	Dossier     string                 `json:"dossier,omitempty"`
	Modules     map[string]interface{} `json:"modules,omitempty"`
	ExpiryDate  string                 `json:"expiry_date,omitempty"`
	Platform    string                 `json:"platform,omitempty"`
}

// Job represents a background job.
type Job struct {
	ID         string `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	StatusMsg  string `json:"status_msg,omitempty"`
	Type       string `json:"type,omitempty"`
	Step       int    `json:"step,omitempty"`
	TotalSteps int    `json:"total_steps,omitempty"`
}

// Software represents a software entry.
type Software struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

// VLAN represents a VLAN.
type VLAN struct {
	ID          int64  `json:"id,omitempty"`
	VlanID      int    `json:"vlanid,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	NetworkID   int64  `json:"network_id,omitempty"`
	ModTime     int64  `json:"mod_time,omitempty"`
}

// Node represents a console node.
type Node struct {
	ID          int64  `json:"id,omitempty"`
	UUID        string `json:"uuid,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	Firmware    string `json:"firmware,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	ModTime     int64  `json:"mod_time,omitempty"`
}

// ExtraHopService handles communication with system-level endpoints.
type ExtraHopService struct {
	client *Client
}

// Get retrieves system information.
func (s *ExtraHopService) Get(ctx context.Context) (*ExtraHopInfo, error) {
	var info ExtraHopInfo
	_, err := s.client.get(ctx, "/extrahop", &info)
	return &info, err
}

// GetVersion retrieves firmware version.
func (s *ExtraHopService) GetVersion(ctx context.Context) (*ExtraHopVersion, error) {
	var version ExtraHopVersion
	_, err := s.client.get(ctx, "/extrahop/version", &version)
	return &version, err
}

// GetPlatform retrieves platform information.
func (s *ExtraHopService) GetPlatform(ctx context.Context) (*ExtraHopPlatform, error) {
	var platform ExtraHopPlatform
	_, err := s.client.get(ctx, "/extrahop/platform", &platform)
	return &platform, err
}

// GetEdition retrieves the edition.
func (s *ExtraHopService) GetEdition(ctx context.Context) (*ExtraHopEdition, error) {
	var edition ExtraHopEdition
	_, err := s.client.get(ctx, "/extrahop/edition", &edition)
	return &edition, err
}

// GetServices retrieves service status.
func (s *ExtraHopService) GetServices(ctx context.Context) (map[string]interface{}, error) {
	var services map[string]interface{}
	_, err := s.client.get(ctx, "/extrahop/services", &services)
	return services, err
}

// UpdateServices updates service configuration.
func (s *ExtraHopService) UpdateServices(ctx context.Context, services interface{}) error {
	_, err := s.client.patch(ctx, "/extrahop/services", services)
	return err
}

// GetProcesses retrieves running processes.
func (s *ExtraHopService) GetProcesses(ctx context.Context) ([]map[string]interface{}, error) {
	var processes []map[string]interface{}
	_, err := s.client.get(ctx, "/extrahop/processes", &processes)
	return processes, err
}

// RestartProcess restarts a process.
func (s *ExtraHopService) RestartProcess(ctx context.Context, process string) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/extrahop/processes/%s/restart", process), nil, nil)
	return err
}

// Restart restarts the ExtraHop system.
func (s *ExtraHopService) Restart(ctx context.Context) error {
	_, err := s.client.post(ctx, "/extrahop/restart", nil, nil)
	return err
}

// Shutdown shuts down the ExtraHop system.
func (s *ExtraHopService) Shutdown(ctx context.Context) error {
	_, err := s.client.post(ctx, "/extrahop/shutdown", nil, nil)
	return err
}

// SetSSLCert uploads an SSL certificate.
func (s *ExtraHopService) SetSSLCert(ctx context.Context, cert interface{}) error {
	_, err := s.client.put(ctx, "/extrahop/sslcert", cert, nil)
	return err
}

// GetTicketing retrieves ticketing integration configuration.
func (s *ExtraHopService) GetTicketing(ctx context.Context) (map[string]interface{}, error) {
	var config map[string]interface{}
	_, err := s.client.get(ctx, "/extrahop/ticketing", &config)
	return config, err
}

// UpdateTicketing updates ticketing integration configuration.
func (s *ExtraHopService) UpdateTicketing(ctx context.Context, config interface{}) error {
	_, err := s.client.patch(ctx, "/extrahop/ticketing", config)
	return err
}

// LicenseService handles license-related endpoints.
type LicenseService struct {
	client *Client
}

// Get retrieves the license.
func (s *LicenseService) Get(ctx context.Context) (*License, error) {
	var license License
	_, err := s.client.get(ctx, "/license", &license)
	return &license, err
}

// Set applies a new license.
func (s *LicenseService) Set(ctx context.Context, license interface{}) error {
	_, err := s.client.put(ctx, "/license", license, nil)
	return err
}

// JobService handles job-related endpoints.
type JobService struct {
	client *Client
}

// List retrieves all jobs.
func (s *JobService) List(ctx context.Context) ([]*Job, error) {
	var jobs []*Job
	_, err := s.client.get(ctx, "/jobs", &jobs)
	return jobs, err
}

// Get retrieves a specific job.
func (s *JobService) Get(ctx context.Context, id string) (*Job, error) {
	var job Job
	_, err := s.client.get(ctx, fmt.Sprintf("/jobs/%s", id), &job)
	return &job, err
}

// SoftwareService handles software-related endpoints.
type SoftwareService struct {
	client *Client
}

// List retrieves all software.
func (s *SoftwareService) List(ctx context.Context) ([]*Software, error) {
	var software []*Software
	_, err := s.client.get(ctx, "/software", &software)
	return software, err
}

// Get retrieves a specific software entry.
func (s *SoftwareService) Get(ctx context.Context, id int64) (*Software, error) {
	var sw Software
	_, err := s.client.get(ctx, fmt.Sprintf("/software/%d", id), &sw)
	return &sw, err
}

// VLANService handles VLAN-related endpoints.
type VLANService struct {
	client *Client
}

// List retrieves all VLANs.
func (s *VLANService) List(ctx context.Context) ([]*VLAN, error) {
	var vlans []*VLAN
	_, err := s.client.get(ctx, "/vlans", &vlans)
	return vlans, err
}

// Get retrieves a specific VLAN.
func (s *VLANService) Get(ctx context.Context, id int64) (*VLAN, error) {
	var vlan VLAN
	_, err := s.client.get(ctx, fmt.Sprintf("/vlans/%d", id), &vlan)
	return &vlan, err
}

// Update modifies a VLAN.
func (s *VLANService) Update(ctx context.Context, id int64, vlan *VLAN) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/vlans/%d", id), vlan)
	return err
}

// NodeService handles node-related endpoints.
type NodeService struct {
	client *Client
}

// List retrieves all nodes.
func (s *NodeService) List(ctx context.Context) ([]*Node, error) {
	var nodes []*Node
	_, err := s.client.get(ctx, "/nodes", &nodes)
	return nodes, err
}

// Get retrieves a specific node.
func (s *NodeService) Get(ctx context.Context, id int64) (*Node, error) {
	var node Node
	_, err := s.client.get(ctx, fmt.Sprintf("/nodes/%d", id), &node)
	return &node, err
}

// Update modifies a node.
func (s *NodeService) Update(ctx context.Context, id int64, node *Node) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/nodes/%d", id), node)
	return err
}

// RunningConfigService handles running configuration endpoints.
type RunningConfigService struct {
	client *Client
}

// Get retrieves the running configuration.
func (s *RunningConfigService) Get(ctx context.Context) (map[string]interface{}, error) {
	var config map[string]interface{}
	_, err := s.client.get(ctx, "/runningconfig", &config)
	return config, err
}

// Set replaces the running configuration.
func (s *RunningConfigService) Set(ctx context.Context, config interface{}) error {
	_, err := s.client.put(ctx, "/runningconfig", config, nil)
	return err
}

// Save saves the running configuration.
func (s *RunningConfigService) Save(ctx context.Context) error {
	_, err := s.client.post(ctx, "/runningconfig/save", nil, nil)
	return err
}

// GetSaved retrieves the saved configuration.
func (s *RunningConfigService) GetSaved(ctx context.Context) (map[string]interface{}, error) {
	var config map[string]interface{}
	_, err := s.client.get(ctx, "/runningconfig/saved", &config)
	return config, err
}

// WatchlistService handles watchlist endpoints.
type WatchlistService struct {
	client *Client
}

// ListDevices retrieves all devices on the watchlist.
func (s *WatchlistService) ListDevices(ctx context.Context) ([]*Device, error) {
	var devices []*Device
	_, err := s.client.get(ctx, "/watchlist/devices", &devices)
	return devices, err
}

// AddDevice adds a device to the watchlist.
func (s *WatchlistService) AddDevice(ctx context.Context, deviceID int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/watchlist/device/%d", deviceID), nil, nil)
	return err
}

// RemoveDevice removes a device from the watchlist.
func (s *WatchlistService) RemoveDevice(ctx context.Context, deviceID int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/watchlist/device/%d", deviceID))
	return err
}

// PacketSearchService handles packet search endpoints.
type PacketSearchService struct {
	client *Client
}

// Search performs a packet search and writes PCAP data to w.
func (s *PacketSearchService) Search(ctx context.Context, params map[string]interface{}, w io.Writer) error {
	_, err := s.client.post(ctx, "/packets/search", params, w)
	return err
}

// SupportPackService handles support pack endpoints.
type SupportPackService struct {
	client *Client
}

// List retrieves all support packs.
func (s *SupportPackService) List(ctx context.Context) ([]map[string]interface{}, error) {
	var packs []map[string]interface{}
	_, err := s.client.get(ctx, "/supportpacks", &packs)
	return packs, err
}

// Execute generates a support pack.
func (s *SupportPackService) Execute(ctx context.Context) error {
	_, err := s.client.post(ctx, "/supportpacks/execute", nil, nil)
	return err
}
