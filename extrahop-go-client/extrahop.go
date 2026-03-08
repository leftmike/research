// Package extrahop provides a Go client for the ExtraHop RevealX 360 Cloud API.
//
// Authentication uses OAuth2 Client Credentials with automatic token refresh.
// Tokens are refreshed proactively before expiry (defaulting to 10-minute TTL
// when the server does not advertise an expiry).
//
// Usage:
//
//	client, err := extrahop.NewClient("", "my-client-id", "my-client-secret", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Stage 1 – Ingestion
//	detections, err := client.Detections.List(ctx, nil)
//	device, err := client.Devices.Get(ctx, id)
//
//	// Stage 2 – Live Enrichment
//	metrics, err := client.Metrics.Query(ctx, req)
//	topology, err := client.ActivityMaps.Query(ctx, req)
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
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the ExtraHop RevealX 360 Cloud API base URL (REST API v26.1).
	DefaultBaseURL = "https://extrahop-corp.api.cloud.extrahop.com/api/v1"

	// defaultTokenTTL is used when the token endpoint does not return expires_in.
	defaultTokenTTL = 10 * time.Minute

	// tokenRefreshBuffer refreshes the token this long before it expires.
	tokenRefreshBuffer = 30 * time.Second

	userAgent = "extrahop-go-client/2.0.0"
)

// Client manages communication with the ExtraHop RevealX 360 Cloud API.
// All methods are safe for concurrent use.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	tokenURL     string
	clientID     string
	clientSecret string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time

	// Stage 1 – Ingestion
	Detections *DetectionService
	Devices    *DeviceService

	// Stage 2 – Live Enrichment
	Metrics      *MetricService
	Records      *RecordService
	ActivityMaps *ActivityMapService
}

// NewClient creates a new ExtraHop RevealX 360 API client.
//
// baseURL defaults to DefaultBaseURL when empty. clientID and clientSecret are
// the OAuth2 client credentials. httpClient defaults to http.DefaultClient.
func NewClient(baseURL, clientID, clientSecret string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("extrahop: invalid base URL: %w", err)
	}
	// Derive the OAuth2 token endpoint from the API base URL host.
	tokenURL := fmt.Sprintf("%s://%s/oauth2/token", u.Scheme, u.Host)

	c := &Client{
		httpClient:   httpClient,
		baseURL:      baseURL,
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
	}

	c.Detections = &DetectionService{client: c}
	c.Devices = &DeviceService{client: c}
	c.Metrics = &MetricService{client: c}
	c.Records = &RecordService{client: c}
	c.ActivityMaps = &ActivityMapService{client: c}

	return c, nil
}

// tokenResponse is the OAuth2 token endpoint response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// refreshToken fetches a new OAuth2 access token using client credentials.
// The caller must not hold c.mu.
func (c *Client) refreshToken(ctx context.Context) error {
	body := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s",
		url.QueryEscape(c.clientID),
		url.QueryEscape(c.clientSecret),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL,
		strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("extrahop: building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("extrahop: fetching token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("extrahop: token endpoint %d: %s", resp.StatusCode, string(data))
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return fmt.Errorf("extrahop: decoding token response: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("extrahop: empty access_token in response")
	}

	ttl := defaultTokenTTL
	if tr.ExpiresIn > 0 {
		ttl = time.Duration(tr.ExpiresIn)*time.Second - tokenRefreshBuffer
	}

	c.mu.Lock()
	c.accessToken = tr.AccessToken
	c.tokenExpiry = time.Now().Add(ttl)
	c.mu.Unlock()

	return nil
}

// token returns a valid access token, refreshing it if expired or missing.
func (c *Client) token(ctx context.Context) (string, error) {
	c.mu.Lock()
	tok := c.accessToken
	expiry := c.tokenExpiry
	c.mu.Unlock()

	if tok == "" || time.Now().After(expiry) {
		if err := c.refreshToken(ctx); err != nil {
			return "", err
		}
		c.mu.Lock()
		tok = c.accessToken
		c.mu.Unlock()
	}
	return tok, nil
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
	return fmt.Sprintf("extrahop: HTTP %d", e.StatusCode)
}

// get performs an authenticated GET request and decodes the JSON response into v.
func (c *Client) get(ctx context.Context, path string, v interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, v)
}

// post performs an authenticated POST request with JSON body and decodes the response into v.
func (c *Client) post(ctx context.Context, path string, body, v interface{}) error {
	return c.doRequest(ctx, http.MethodPost, path, body, v)
}

// doRequest executes an authenticated HTTP request.
func (c *Client) doRequest(ctx context.Context, method, path string, body, v interface{}) error {
	tok, err := c.token(ctx)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("extrahop: marshaling request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("extrahop: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("extrahop: executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &Error{StatusCode: resp.StatusCode}
		data, _ := io.ReadAll(resp.Body)
		if len(data) > 0 {
			_ = json.Unmarshal(data, apiErr)
			if apiErr.Message == "" {
				apiErr.Message = string(data)
			}
		}
		return apiErr
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("extrahop: decoding response: %w", err)
		}
	}
	return nil
}
