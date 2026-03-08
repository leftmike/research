package extrahop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setupTestServer creates a test HTTP server and a client pointed at it.
// The server handles the OAuth2 token endpoint automatically, returning a
// dummy Bearer token so individual test handlers can focus on the API call.
func setupTestServer(handler http.HandlerFunc) (*Client, *httptest.Server) {
	mux := http.NewServeMux()

	// Token endpoint: return a dummy access token.
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   600,
		})
	})

	// API endpoints: delegate to the test-supplied handler.
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", handler))

	server := httptest.NewServer(mux)
	client, _ := NewClient(server.URL+"/api/v1", "test-client-id", "test-secret", nil)
	return client, server
}

// assertBearer verifies the request carries a Bearer token.
func assertBearer(t *testing.T, r *http.Request) {
	t.Helper()
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer token", auth)
	}
}

// ---------- Client construction ----------

func TestNewClientDefaults(t *testing.T) {
	c, err := NewClient("", "id", "secret", nil)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if c.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
}

func TestNewClientCustomBaseURL(t *testing.T) {
	c, err := NewClient("https://custom.example.com/api/v1", "id", "secret", nil)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if c.baseURL != "https://custom.example.com/api/v1" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	if c.tokenURL != "https://custom.example.com/oauth2/token" {
		t.Errorf("tokenURL = %q", c.tokenURL)
	}
}

func TestNewClientTrailingSlashStripped(t *testing.T) {
	c, err := NewClient("https://example.com/api/v1/", "id", "secret", nil)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if strings.HasSuffix(c.baseURL, "/") {
		t.Errorf("baseURL has trailing slash: %q", c.baseURL)
	}
}

// ---------- OAuth2 token refresh ----------

func TestTokenFetchedOnFirstRequest(t *testing.T) {
	var gotAuth string
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode([]*Detection{})
	})
	defer server.Close()

	client.Detections.List(context.Background(), nil) //nolint:errcheck
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer token", gotAuth)
	}
}

// ---------- Stage 1: Ingestion – Detections ----------

func TestDetectionsList(t *testing.T) {
	want := []*Detection{
		{ID: 1, Title: "Lateral Movement", RiskScore: intPtr(85), Status: "new"},
		{ID: 2, Title: "Data Exfiltration", RiskScore: intPtr(92), Status: "in_progress"},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/detections" {
			t.Errorf("path = %q, want /detections", r.URL.Path)
		}
		assertBearer(t, r)
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Detections.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d detections, want 2", len(got))
	}
	if got[0].Title != "Lateral Movement" {
		t.Errorf("title = %q, want %q", got[0].Title, "Lateral Movement")
	}
}

func TestDetectionsListWithParams(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "50" {
			t.Errorf("limit = %q, want 50", q.Get("limit"))
		}
		json.NewEncoder(w).Encode([]*Detection{})
	})
	defer server.Close()

	client.Detections.List(context.Background(), &DetectionListParams{Limit: 50}) //nolint:errcheck
}

func TestDetectionsGet(t *testing.T) {
	participant := DetectionParticipant{ObjectType: "device", ObjectID: 99, Role: "offender"}
	want := &Detection{
		ID:           42,
		Title:        "Port Scan",
		Status:       "new",
		RiskScore:    intPtr(60),
		Participants: []DetectionParticipant{participant},
		MitreTechniques: []MitreTechnique{
			{ID: "T1046", Name: "Network Service Scanning"},
		},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/detections/42" {
			t.Errorf("path = %q, want /detections/42", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Detections.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if len(got.Participants) != 1 {
		t.Fatalf("participants = %d, want 1", len(got.Participants))
	}
	if got.Participants[0].Role != "offender" {
		t.Errorf("role = %q, want offender", got.Participants[0].Role)
	}
}

// ---------- Stage 1: Ingestion – Devices ----------

func TestDevicesSearch(t *testing.T) {
	want := []*Device{{ID: 7, IPAddr4: strPtr("10.0.0.7"), DefaultName: "web-server"}}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/devices/search" {
			t.Errorf("path = %q, want /devices/search", r.URL.Path)
		}
		var req DeviceSearchRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		if req.Filter == nil || req.Filter.Field != "ipaddr" {
			t.Errorf("expected ipaddr filter, got %+v", req.Filter)
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Devices.Search(context.Background(), &DeviceSearchRequest{
		Filter: &DeviceFilter{Field: "ipaddr", Operator: "=", Operand: "10.0.0.7"},
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 7 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestDevicesSearchByMAC(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req DeviceSearchRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		if req.Filter.Field != "macaddr" {
			t.Errorf("field = %q, want macaddr", req.Filter.Field)
		}
		json.NewEncoder(w).Encode([]*Device{{ID: 3}})
	})
	defer server.Close()

	client.Devices.Search(context.Background(), &DeviceSearchRequest{ //nolint:errcheck
		Filter: &DeviceFilter{Field: "macaddr", Operator: "=", Operand: "aa:bb:cc:dd:ee:ff"},
	})
}

func TestDevicesGet(t *testing.T) {
	want := &Device{
		ID:          100,
		DisplayName: "db-server",
		IPAddr4:     strPtr("192.168.1.10"),
		DeviceClass: "server",
		Analysis:    "full",
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/devices/100" {
			t.Errorf("path = %q, want /devices/100", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Devices.Get(context.Background(), 100)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.DisplayName != "db-server" {
		t.Errorf("name = %q, want db-server", got.DisplayName)
	}
}

// ---------- Stage 2: Live Enrichment – Device Activity ----------

func TestDevicesGetActivity(t *testing.T) {
	want := []*DeviceActivity{
		{Protocol: "HTTP", InBytes: 1024000, OutBytes: 512000, Requests: 500},
		{Protocol: "DNS", InBytes: 2048, OutBytes: 1024, Requests: 100},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/devices/55/activity" {
			t.Errorf("path = %q, want /devices/55/activity", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Devices.GetActivity(context.Background(), 55)
	if err != nil {
		t.Fatalf("GetActivity error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d activity records, want 2", len(got))
	}
	if got[0].Protocol != "HTTP" {
		t.Errorf("protocol = %q, want HTTP", got[0].Protocol)
	}
}

// ---------- Stage 2: Live Enrichment – Device Groups ----------

func TestDevicesListDeviceGroups(t *testing.T) {
	want := []*DeviceGroup{
		{ID: 10, Name: "Web Servers", Type: "dynamic"},
		{ID: 20, Name: "Critical Assets", Type: "static"},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/devices/55/devicegroups" {
			t.Errorf("path = %q, want /devices/55/devicegroups", r.URL.Path)
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Devices.ListDeviceGroups(context.Background(), 55)
	if err != nil {
		t.Fatalf("ListDeviceGroups error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d groups, want 2", len(got))
	}
	if got[0].Name != "Web Servers" {
		t.Errorf("name = %q, want Web Servers", got[0].Name)
	}
}

// ---------- Stage 2: Live Enrichment – Metrics ----------

func TestMetricsQuery(t *testing.T) {
	want := &MetricResponse{
		Cycle: "30sec",
		From:  1700000000000,
		Stats: []MetricStat{
			{OID: 100, Time: 1700000000000, Values: []interface{}{float64(5000)}},
		},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/metrics" {
			t.Errorf("path = %q, want /metrics", r.URL.Path)
		}
		var req MetricRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		if req.ObjectType != "device" {
			t.Errorf("object_type = %q, want device", req.ObjectType)
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Metrics.Query(context.Background(), &MetricRequest{
		Cycle:       "30sec",
		From:        -1800000,
		ObjectType:  "device",
		ObjectIDs:   []int64{100},
		MetricSpecs: []MetricSpec{{Name: "extrahop.device.net_detail", CalcType: "sum"}},
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if got.Cycle != "30sec" {
		t.Errorf("cycle = %q, want 30sec", got.Cycle)
	}
	if len(got.Stats) != 1 {
		t.Fatalf("stats count = %d, want 1", len(got.Stats))
	}
}

// ---------- Stage 2: Live Enrichment – Records ----------

func TestRecordsSearch(t *testing.T) {
	want := &RecordSearchResponse{
		Total:  3,
		Cursor: "abc123",
		Records: []map[string]interface{}{
			{"type": "~HTTP", "senderAddr": "10.0.0.1", "receiverAddr": "10.0.0.2"},
		},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/records/search" {
			t.Errorf("path = %q, want /records/search", r.URL.Path)
		}
		var req RecordSearchRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		if len(req.Types) == 0 || req.Types[0] != "~HTTP" {
			t.Errorf("types = %v, want [~HTTP]", req.Types)
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.Records.Search(context.Background(), &RecordSearchRequest{
		From:  -3600000,
		Limit: 100,
		Types: []string{"~HTTP"},
		Filter: &RecordFilter{
			Field: "senderAddr", Operator: "=", Operand: "10.0.0.1",
		},
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if got.Total != 3 {
		t.Errorf("total = %d, want 3", got.Total)
	}
	if got.Cursor != "abc123" {
		t.Errorf("cursor = %q, want abc123", got.Cursor)
	}
}

// ---------- Stage 2: Live Enrichment – Activity Maps ----------

func TestActivityMapsQuery(t *testing.T) {
	want := &ActivityMapQueryResponse{
		Edges: []ActivityMapEdge{
			{
				From: ActivityMapEndpoint{ObjectType: "device", ObjectID: 10},
				To:   ActivityMapEndpoint{ObjectType: "device", ObjectID: 20},
				Annotations: ActivityMapAnnotations{
					Protocols: []string{"HTTP"},
					Counts:    map[string]int64{"bytes": 1048576},
				},
			},
		},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/activitymaps/query" {
			t.Errorf("path = %q, want /activitymaps/query", r.URL.Path)
		}
		var req ActivityMapQueryRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		if len(req.Walks) == 0 {
			t.Error("expected at least one walk")
		}
		json.NewEncoder(w).Encode(want)
	})
	defer server.Close()

	got, err := client.ActivityMaps.Query(context.Background(), &ActivityMapQueryRequest{
		From: -1800000,
		Walks: []Walk{
			{
				Origins: []Origin{{ObjectType: "device", ObjectID: 10}},
				Steps:   []Step{{Direction: "any"}},
			},
		},
		Weighting: "bytes",
	})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(got.Edges) != 1 {
		t.Fatalf("edges = %d, want 1", len(got.Edges))
	}
	if got.Edges[0].From.ObjectID != 10 {
		t.Errorf("from ID = %d, want 10", got.Edges[0].From.ObjectID)
	}
}

func TestActivityMapsQueryMultiHop(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req ActivityMapQueryRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		walk := req.Walks[0]
		if len(walk.Steps) != 2 {
			t.Errorf("steps = %d, want 2 (multi-hop)", len(walk.Steps))
		}
		json.NewEncoder(w).Encode(&ActivityMapQueryResponse{})
	})
	defer server.Close()

	client.ActivityMaps.Query(context.Background(), &ActivityMapQueryRequest{ //nolint:errcheck
		From: -3600000,
		Walks: []Walk{
			{
				Origins: []Origin{{ObjectType: "device", ObjectID: 5}},
				Steps: []Step{
					{Relationships: []Relationship{{Protocol: "SMB", Role: "client"}}},
					{Relationships: []Relationship{{Protocol: "HTTP", Role: "any"}}},
				},
			},
		},
	})
}

// ---------- Error handling ----------

func TestAPIError(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error_message": "detection not found"})
	})
	defer server.Close()

	_, err := client.Detections.Get(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "detection not found" {
		t.Errorf("Message = %q, want 'detection not found'", apiErr.Message)
	}
}

// ---------- helpers ----------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
