package extrahop

import (
	"context"
	"fmt"
)

// Bundle represents an ExtraHop bundle.
type Bundle struct {
	ID          int64                  `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Author      string                 `json:"author,omitempty"`
	Description string                 `json:"description,omitempty"`
	ModTime     int64                  `json:"mod_time,omitempty"`
	Built       map[string]interface{} `json:"built,omitempty"`
}

// Application represents an ExtraHop application.
type Application struct {
	ID           int64  `json:"id,omitempty"`
	NodeID       *int64 `json:"node_id,omitempty"`
	ExtrahopID   string `json:"extrahop_id,omitempty"`
	DiscoveryID  string `json:"discovery_id,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	Description  string `json:"description,omitempty"`
	UserModTime  int64  `json:"user_mod_time,omitempty"`
	ModTime      int64  `json:"mod_time,omitempty"`
}

// ActivityMap represents an ExtraHop activity map.
type ActivityMap struct {
	ID              int64    `json:"id,omitempty"`
	Name            string   `json:"name,omitempty"`
	Description     string   `json:"description,omitempty"`
	Owner           string   `json:"owner,omitempty"`
	Mode            string   `json:"mode,omitempty"`
	Weighting       string   `json:"weighting,omitempty"`
	ShowAlertStatus bool     `json:"show_alert_status,omitempty"`
	ShortCode       string   `json:"short_code,omitempty"`
	Walks           []interface{} `json:"walks,omitempty"`
	Rights          []string `json:"rights,omitempty"`
	ModTime         int64    `json:"mod_time,omitempty"`
}

// CustomDevice represents an ExtraHop custom device.
type CustomDevice struct {
	ID          int64                  `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Author      string                 `json:"author,omitempty"`
	Description string                 `json:"description,omitempty"`
	ExtrahopID  string                 `json:"extrahop_id,omitempty"`
	Criteria    []map[string]interface{} `json:"criteria,omitempty"`
	Disabled    bool                   `json:"disabled,omitempty"`
	ModTime     int64                  `json:"mod_time,omitempty"`
}

// EmailGroup represents an email notification group.
type EmailGroup struct {
	ID          int64    `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	EmailAddresses []string `json:"email_addresses,omitempty"`
	ModTime     int64    `json:"mod_time,omitempty"`
}

// ExclusionInterval represents an alert exclusion interval.
type ExclusionInterval struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Interval    *TimeInterval `json:"interval,omitempty"`
	ModTime     int64  `json:"mod_time,omitempty"`
}

// TimeInterval represents a recurring time interval.
type TimeInterval struct {
	DaysOfWeek []int  `json:"days_of_week,omitempty"`
	StartTime  string `json:"start,omitempty"`
	EndTime    string `json:"end,omitempty"`
}

// ThreatCollection represents a threat intelligence collection.
type ThreatCollection struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	UserKey     string `json:"user_key,omitempty"`
	SourceType  string `json:"source_type,omitempty"`
	ModTime     int64  `json:"mod_time,omitempty"`
}

// Report represents a scheduled report.
type Report struct {
	ID          int64    `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Type        string   `json:"type,omitempty"`
	Enabled     bool     `json:"enabled,omitempty"`
	ModTime     int64    `json:"mod_time,omitempty"`
}

// APIKey represents an API key.
type APIKey struct {
	ID          int64  `json:"id,omitempty"`
	KeyID       string `json:"keyid,omitempty"`
	Description string `json:"description,omitempty"`
	Time        string `json:"time,omitempty"`
}

// BundleService handles bundle-related endpoints.
type BundleService struct {
	client *Client
}

func (s *BundleService) List(ctx context.Context) ([]*Bundle, error) {
	var bundles []*Bundle
	_, err := s.client.get(ctx, "/bundles", &bundles)
	return bundles, err
}

func (s *BundleService) Create(ctx context.Context, bundle *Bundle) error {
	_, err := s.client.post(ctx, "/bundles", bundle, nil)
	return err
}

func (s *BundleService) Get(ctx context.Context, id int64) (*Bundle, error) {
	var bundle Bundle
	_, err := s.client.get(ctx, fmt.Sprintf("/bundles/%d", id), &bundle)
	return &bundle, err
}

func (s *BundleService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/bundles/%d", id))
	return err
}

func (s *BundleService) Apply(ctx context.Context, id int64) error {
	_, err := s.client.post(ctx, fmt.Sprintf("/bundles/%d/apply", id), nil, nil)
	return err
}

// ApplicationService handles application-related endpoints.
type ApplicationService struct {
	client *Client
}

func (s *ApplicationService) List(ctx context.Context) ([]*Application, error) {
	var apps []*Application
	_, err := s.client.get(ctx, "/applications", &apps)
	return apps, err
}

func (s *ApplicationService) Get(ctx context.Context, id int64) (*Application, error) {
	var app Application
	_, err := s.client.get(ctx, fmt.Sprintf("/applications/%d", id), &app)
	return &app, err
}

func (s *ApplicationService) Update(ctx context.Context, id int64, app *Application) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/applications/%d", id), app)
	return err
}

// ActivityMapService handles activity map endpoints.
type ActivityMapService struct {
	client *Client
}

func (s *ActivityMapService) List(ctx context.Context) ([]*ActivityMap, error) {
	var maps []*ActivityMap
	_, err := s.client.get(ctx, "/activitymaps", &maps)
	return maps, err
}

func (s *ActivityMapService) Create(ctx context.Context, am *ActivityMap) error {
	_, err := s.client.post(ctx, "/activitymaps", am, nil)
	return err
}

func (s *ActivityMapService) Get(ctx context.Context, id int64) (*ActivityMap, error) {
	var am ActivityMap
	_, err := s.client.get(ctx, fmt.Sprintf("/activitymaps/%d", id), &am)
	return &am, err
}

func (s *ActivityMapService) Update(ctx context.Context, id int64, am *ActivityMap) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/activitymaps/%d", id), am)
	return err
}

func (s *ActivityMapService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/activitymaps/%d", id))
	return err
}

// CustomDeviceService handles custom device endpoints.
type CustomDeviceService struct {
	client *Client
}

func (s *CustomDeviceService) List(ctx context.Context) ([]*CustomDevice, error) {
	var devices []*CustomDevice
	_, err := s.client.get(ctx, "/customdevices", &devices)
	return devices, err
}

func (s *CustomDeviceService) Create(ctx context.Context, device *CustomDevice) error {
	_, err := s.client.post(ctx, "/customdevices", device, nil)
	return err
}

func (s *CustomDeviceService) Get(ctx context.Context, id int64) (*CustomDevice, error) {
	var device CustomDevice
	_, err := s.client.get(ctx, fmt.Sprintf("/customdevices/%d", id), &device)
	return &device, err
}

func (s *CustomDeviceService) Update(ctx context.Context, id int64, device *CustomDevice) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/customdevices/%d", id), device)
	return err
}

func (s *CustomDeviceService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/customdevices/%d", id))
	return err
}

// EmailGroupService handles email group endpoints.
type EmailGroupService struct {
	client *Client
}

func (s *EmailGroupService) List(ctx context.Context) ([]*EmailGroup, error) {
	var groups []*EmailGroup
	_, err := s.client.get(ctx, "/emailgroups", &groups)
	return groups, err
}

func (s *EmailGroupService) Create(ctx context.Context, group *EmailGroup) error {
	_, err := s.client.post(ctx, "/emailgroups", group, nil)
	return err
}

func (s *EmailGroupService) Get(ctx context.Context, id int64) (*EmailGroup, error) {
	var group EmailGroup
	_, err := s.client.get(ctx, fmt.Sprintf("/emailgroups/%d", id), &group)
	return &group, err
}

func (s *EmailGroupService) Update(ctx context.Context, id int64, group *EmailGroup) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/emailgroups/%d", id), group)
	return err
}

func (s *EmailGroupService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/emailgroups/%d", id))
	return err
}

// ExclusionIntervalService handles exclusion interval endpoints.
type ExclusionIntervalService struct {
	client *Client
}

func (s *ExclusionIntervalService) List(ctx context.Context) ([]*ExclusionInterval, error) {
	var intervals []*ExclusionInterval
	_, err := s.client.get(ctx, "/exclusionintervals", &intervals)
	return intervals, err
}

func (s *ExclusionIntervalService) Create(ctx context.Context, interval *ExclusionInterval) error {
	_, err := s.client.post(ctx, "/exclusionintervals", interval, nil)
	return err
}

func (s *ExclusionIntervalService) Get(ctx context.Context, id int64) (*ExclusionInterval, error) {
	var interval ExclusionInterval
	_, err := s.client.get(ctx, fmt.Sprintf("/exclusionintervals/%d", id), &interval)
	return &interval, err
}

func (s *ExclusionIntervalService) Update(ctx context.Context, id int64, interval *ExclusionInterval) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/exclusionintervals/%d", id), interval)
	return err
}

func (s *ExclusionIntervalService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/exclusionintervals/%d", id))
	return err
}

// ThreatCollectionService handles threat collection endpoints.
type ThreatCollectionService struct {
	client *Client
}

func (s *ThreatCollectionService) List(ctx context.Context) ([]*ThreatCollection, error) {
	var collections []*ThreatCollection
	_, err := s.client.get(ctx, "/threatcollections", &collections)
	return collections, err
}

func (s *ThreatCollectionService) Create(ctx context.Context, collection *ThreatCollection) error {
	_, err := s.client.post(ctx, "/threatcollections", collection, nil)
	return err
}

func (s *ThreatCollectionService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/threatcollections/%d", id))
	return err
}

// ReportService handles report endpoints.
type ReportService struct {
	client *Client
}

func (s *ReportService) List(ctx context.Context) ([]*Report, error) {
	var reports []*Report
	_, err := s.client.get(ctx, "/reports", &reports)
	return reports, err
}

func (s *ReportService) Create(ctx context.Context, report *Report) error {
	_, err := s.client.post(ctx, "/reports", report, nil)
	return err
}

func (s *ReportService) Get(ctx context.Context, id int64) (*Report, error) {
	var report Report
	_, err := s.client.get(ctx, fmt.Sprintf("/reports/%d", id), &report)
	return &report, err
}

func (s *ReportService) Update(ctx context.Context, id int64, report *Report) error {
	_, err := s.client.patch(ctx, fmt.Sprintf("/reports/%d", id), report)
	return err
}

func (s *ReportService) Delete(ctx context.Context, id int64) error {
	_, err := s.client.delete(ctx, fmt.Sprintf("/reports/%d", id))
	return err
}

// APIKeyService handles API key endpoints.
type APIKeyService struct {
	client *Client
}

func (s *APIKeyService) List(ctx context.Context) ([]*APIKey, error) {
	var keys []*APIKey
	_, err := s.client.get(ctx, "/apikeys", &keys)
	return keys, err
}

func (s *APIKeyService) Create(ctx context.Context, key *APIKey) error {
	_, err := s.client.post(ctx, "/apikeys", key, nil)
	return err
}

func (s *APIKeyService) Get(ctx context.Context, keyID string) (*APIKey, error) {
	var key APIKey
	_, err := s.client.get(ctx, fmt.Sprintf("/apikeys/%s", keyID), &key)
	return &key, err
}

// AuditLogService handles audit log endpoints.
type AuditLogService struct {
	client *Client
}

func (s *AuditLogService) List(ctx context.Context) ([]map[string]interface{}, error) {
	var entries []map[string]interface{}
	_, err := s.client.get(ctx, "/auditlog", &entries)
	return entries, err
}

// CustomizationService handles customization endpoints.
type CustomizationService struct {
	client *Client
}

// Customization represents a backup customization.
type Customization struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	ModTime int64  `json:"mod_time,omitempty"`
}
