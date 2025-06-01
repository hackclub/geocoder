package geocoding

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeocodeClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{"With API key", "test-key", true},
		{"Without API key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey)
			if got := client.IsConfigured(); got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGeocodeClient_Geocode(t *testing.T) {
	// Mock Google Geocoding API response
	mockResponse := `{
		"results": [
			{
				"formatted_address": "1600 Amphitheatre Parkway, Mountain View, CA 94043, USA",
				"geometry": {
					"location": {
						"lat": 37.4224764,
						"lng": -122.0842499
					},
					"location_type": "ROOFTOP"
				},
				"place_id": "ChIJ2eUgeAK6j4ARbn5u_wAGqWA",
				"types": ["street_address"]
			}
		],
		"status": "OK"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-api-key")

	// We can't easily test the actual geocode method without modifying the URL
	// This test mainly checks that the client is properly initialized
	if !client.IsConfigured() {
		t.Error("Client should be configured with API key")
	}
}

func TestGeocodeClient_NoAPIKey(t *testing.T) {
	client := NewClient("")

	_, err := client.Geocode("test address")
	if err == nil {
		t.Error("Expected error when no API key is configured")
	}

	expectedError := "Google Geocoding API key not configured"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}
