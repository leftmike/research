// Package extrahop provides a Go client for the ExtraHop REST API.
//
// The ExtraHop REST API enables automation of administration and configuration
// tasks on ExtraHop systems. Authentication is performed via API keys.
//
// Usage:
//
//	client, err := extrahop.NewClient("https://extrahop.example.com", "your-api-key", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	devices, err := client.Devices.List(ctx, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
package extrahop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	apiBasePath = "/api/v1"
	userAgent   = "extrahop-go-client/0.1.0"
)

// Client manages communication with the ExtraHop REST API.
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	apiKey     string
	userAgent  string

	ActivityMaps       *ActivityMapService
	Alerts             *AlertService
	APIKeys            *APIKeyService
	Appliances         *ApplianceService
	Applications       *ApplicationService
	AuditLog           *AuditLogService
	Bundles            *BundleService
	CustomDevices      *CustomDeviceService
	Dashboards         *DashboardService
	Detections         *DetectionService
	DeviceGroups       *DeviceGroupService
	Devices            *DeviceService
	EmailGroups        *EmailGroupService
	ExclusionIntervals *ExclusionIntervalService
	ExtraHop           *ExtraHopService
	Investigations     *InvestigationService
	Jobs               *JobService
	License            *LicenseService
	Metrics            *MetricService
	Networks           *NetworkService
	NetworkLocalities  *NetworkLocalityService
	Nodes              *NodeService
	PacketSearch       *PacketSearchService
	Records            *RecordService
	Reports            *ReportService
	RunningConfig      *RunningConfigService
	Software           *SoftwareService
	SupportPacks       *SupportPackService
	Tags               *TagService
	ThreatCollections  *ThreatCollectionService
	Triggers           *TriggerService
	Users              *UserService
	UserGroups         *UserGroupService
	VLANs              *VLANService
	Watchlist          *WatchlistService
}

// NewClient creates a new ExtraHop API client. If httpClient is nil,
// http.DefaultClient is used. The host should be the base URL of the
// ExtraHop system (e.g., "https://extrahop.example.com").
func NewClient(host, apiKey string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	if !strings.HasPrefix(host, "http") {
		host = "https://" + host
	}
	host = strings.TrimRight(host, "/")

	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid host URL: %w", err)
	}

	c := &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKey:     apiKey,
		userAgent:  userAgent,
	}

	c.ActivityMaps = &ActivityMapService{client: c}
	c.Alerts = &AlertService{client: c}
	c.APIKeys = &APIKeyService{client: c}
	c.Appliances = &ApplianceService{client: c}
	c.Applications = &ApplicationService{client: c}
	c.AuditLog = &AuditLogService{client: c}
	c.Bundles = &BundleService{client: c}
	c.CustomDevices = &CustomDeviceService{client: c}
	c.Dashboards = &DashboardService{client: c}
	c.Detections = &DetectionService{client: c}
	c.DeviceGroups = &DeviceGroupService{client: c}
	c.Devices = &DeviceService{client: c}
	c.EmailGroups = &EmailGroupService{client: c}
	c.ExclusionIntervals = &ExclusionIntervalService{client: c}
	c.ExtraHop = &ExtraHopService{client: c}
	c.Investigations = &InvestigationService{client: c}
	c.Jobs = &JobService{client: c}
	c.License = &LicenseService{client: c}
	c.Metrics = &MetricService{client: c}
	c.Networks = &NetworkService{client: c}
	c.NetworkLocalities = &NetworkLocalityService{client: c}
	c.Nodes = &NodeService{client: c}
	c.PacketSearch = &PacketSearchService{client: c}
	c.Records = &RecordService{client: c}
	c.Reports = &ReportService{client: c}
	c.RunningConfig = &RunningConfigService{client: c}
	c.Software = &SoftwareService{client: c}
	c.SupportPacks = &SupportPackService{client: c}
	c.Tags = &TagService{client: c}
	c.ThreatCollections = &ThreatCollectionService{client: c}
	c.Triggers = &TriggerService{client: c}
	c.Users = &UserService{client: c}
	c.UserGroups = &UserGroupService{client: c}
	c.VLANs = &VLANService{client: c}
	c.Watchlist = &WatchlistService{client: c}

	return c, nil
}

// Error represents an error response from the ExtraHop API.
type Error struct {
	StatusCode int    `json:"-"`
	Message    string `json:"error_message,omitempty"`
	Type       string `json:"type,omitempty"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("extrahop: %d %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("extrahop: %d", e.StatusCode)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(apiBasePath + path)
	if err != nil {
		return nil, err
	}
	u := c.baseURL.ResolveReference(rel)

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "ExtraHop apikey="+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &Error{StatusCode: resp.StatusCode}
		data, _ := io.ReadAll(resp.Body)
		if len(data) > 0 {
			_ = json.Unmarshal(data, apiErr)
		}
		return resp, apiErr
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}

	return resp, err
}

// get performs a GET request.
func (c *Client) get(ctx context.Context, path string, v interface{}) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req, v)
}

// post performs a POST request.
func (c *Client) post(ctx context.Context, path string, body, v interface{}) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	return c.do(req, v)
}

// put performs a PUT request.
func (c *Client) put(ctx context.Context, path string, body, v interface{}) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	return c.do(req, v)
}

// patch performs a PATCH request.
func (c *Client) patch(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}
	return c.do(req, nil)
}

// delete performs a DELETE request.
func (c *Client) delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req, nil)
}
