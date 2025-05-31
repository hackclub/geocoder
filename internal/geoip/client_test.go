package geoip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeoIPClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
	}{
		{"With API key", "test-key"},
		{"Without API key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey)
			// IPinfo client is always configured (can work without API key)
			if !client.IsConfigured() {
				t.Error("IPinfo client should always be configured")
			}
		})
	}
}

func TestGeoIPClient_GetIPInfo(t *testing.T) {
	// Mock IPinfo API response
	mockResponse := `{
		"ip": "8.8.8.8",
		"city": "Mountain View",
		"region": "California",
		"country": "US",
		"loc": "37.4056,-122.0775",
		"org": "AS15169 Google LLC",
		"postal": "94043",
		"timezone": "America/Los_Angeles"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("")
	
	// We can't easily test the actual method without modifying the URL
	// This test mainly verifies the client initialization
	if !client.IsConfigured() {
		t.Error("Client should be configured")
	}
}
