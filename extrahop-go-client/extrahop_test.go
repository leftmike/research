package extrahop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client, _ := NewClient(server.URL, "test-api-key", nil)
	return client, server
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("https://extrahop.example.com", "test-key", nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client.baseURL.String() != "https://extrahop.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL.String(), "https://extrahop.example.com")
	}
	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", client.apiKey, "test-key")
	}
}

func TestNewClientWithoutScheme(t *testing.T) {
	client, err := NewClient("extrahop.example.com", "test-key", nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client.baseURL.Scheme != "https" {
		t.Errorf("scheme = %q, want %q", client.baseURL.Scheme, "https")
	}
}

func TestNewClientTrailingSlash(t *testing.T) {
	client, err := NewClient("https://extrahop.example.com/", "test-key", nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client.baseURL.String() != "https://extrahop.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL.String(), "https://extrahop.example.com")
	}
}

func TestAuthHeader(t *testing.T) {
	var gotAuth string
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{})
	})
	defer server.Close()

	client.ExtraHop.Get(context.Background())
	want := "ExtraHop apikey=test-api-key"
	if gotAuth != want {
		t.Errorf("Authorization header = %q, want %q", gotAuth, want)
	}
}

func TestDevicesList(t *testing.T) {
	devices := []*Device{
		{ID: 1, DefaultName: "Device1", IPAddr4: strPtr("10.0.0.1")},
		{ID: 2, DefaultName: "Device2", IPAddr4: strPtr("10.0.0.2")},
	}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/v1/devices" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/devices")
		}
		json.NewEncoder(w).Encode(devices)
	})
	defer server.Close()

	result, err := client.Devices.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d devices, want 2", len(result))
	}
	if result[0].DefaultName != "Device1" {
		t.Errorf("device name = %q, want %q", result[0].DefaultName, "Device1")
	}
}

func TestDeviceGet(t *testing.T) {
	device := &Device{ID: 42, DefaultName: "TestDevice", IPAddr4: strPtr("192.168.1.1")}

	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/devices/42" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/devices/42")
		}
		json.NewEncoder(w).Encode(device)
	})
	defer server.Close()

	result, err := client.Devices.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.ID != 42 {
		t.Errorf("device ID = %d, want 42", result.ID)
	}
}

func TestDeviceSearch(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/v1/devices/search" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/devices/search")
		}
		json.NewEncoder(w).Encode([]*Device{{ID: 1}})
	})
	defer server.Close()

	result, err := client.Devices.Search(context.Background(), &DeviceSearchRequest{
		Filter: &DeviceFilter{
			Field:    "ipaddr",
			Operand:  "10.0.0.1",
			Operator: "=",
		},
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d devices, want 1", len(result))
	}
}

func TestDetectionsList(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/detections" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/detections")
		}
		json.NewEncoder(w).Encode([]*Detection{{ID: 1, Title: "Test Detection"}})
	})
	defer server.Close()

	result, err := client.Detections.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d detections, want 1", len(result))
	}
}

func TestAlertsList(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/alerts" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/alerts")
		}
		json.NewEncoder(w).Encode([]*Alert{{ID: 1, Name: "Test Alert"}})
	})
	defer server.Close()

	result, err := client.Alerts.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d alerts, want 1", len(result))
	}
}

func TestErrorResponse(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error_message": "not found"})
	})
	defer server.Close()

	_, err := client.Devices.Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("status code = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "not found" {
		t.Errorf("message = %q, want %q", apiErr.Message, "not found")
	}
}

func TestMetricsQuery(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/api/v1/metrics" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/metrics")
		}
		json.NewEncoder(w).Encode(&MetricResponse{Cycle: "30sec"})
	})
	defer server.Close()

	result, err := client.Metrics.Query(context.Background(), &MetricRequest{
		Cycle:      "30sec",
		ObjectType: "device",
		ObjectIDs:  []int64{1},
		MetricSpecs: []MetricSpec{{Name: "extrahop.device.net_detail"}},
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if result.Cycle != "30sec" {
		t.Errorf("cycle = %q, want %q", result.Cycle, "30sec")
	}
}

func TestTagsList(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]*Tag{{ID: 1, Name: "important"}})
	})
	defer server.Close()

	result, err := client.Tags.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("got %d tags, want 1", len(result))
	}
}

func TestExtraHopGet(t *testing.T) {
	client, server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/extrahop" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/api/v1/extrahop")
		}
		json.NewEncoder(w).Encode(&ExtraHopInfo{
			Hostname: "extrahop.example.com",
			Version:  "9.5.0",
		})
	})
	defer server.Close()

	result, err := client.ExtraHop.Get(context.Background())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.Hostname != "extrahop.example.com" {
		t.Errorf("hostname = %q, want %q", result.Hostname, "extrahop.example.com")
	}
}

func strPtr(s string) *string { return &s }
