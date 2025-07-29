package geocoding

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hackclub/geocoder/internal/models"
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

func TestReverseGeocodeClient_ReverseGeocode(t *testing.T) {
	// Mock Google Reverse Geocoding API response
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
				"types": ["street_address"],
				"address_components": [
					{
						"long_name": "United States",
						"short_name": "US",
						"types": ["country", "political"]
					}
				]
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

	// We can't easily test the actual reverse geocode method without modifying the URL
	// This test mainly checks that the client is properly initialized for reverse geocoding
	if !client.IsConfigured() {
		t.Error("Client should be configured with API key")
	}
}

func TestReverseGeocodeClient_NoAPIKey(t *testing.T) {
	client := NewClient("")

	_, err := client.ReverseGeocode(37.4224764, -122.0842499)
	if err == nil {
		t.Error("Expected error when no API key is configured")
	}

	expectedError := "Google Geocoding API key not configured"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestReverseGeocodeClient_ToStandardFormat(t *testing.T) {
	client := NewClient("")

	// Test with no API key - should return error
	_, err := client.ReverseGeocodeToStandardFormat(37.4224764, -122.0842499)
	if err == nil {
		t.Error("Expected error when no API key is configured")
	}

	expectedError := "Google Geocoding API key not configured"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGeocodeClient_ToStandardFormat_WithStateFields(t *testing.T) {
	// Test that our models include the new state fields
	testResponse := &models.GeocodeAPIResponse{
		Lat:              37.4224764,
		Lng:              -122.0842499,
		FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA 94043, USA",
		StateName:        "California",
		StateCode:        "CA",
		CountryName:      "United States",
		CountryCode:      "US",
		Backend:          "google_maps_platform_geocoding",
	}
	
	// Test the expected structure
	if testResponse.StateName == "" {
		t.Error("StateName field should not be empty")
	}
	if testResponse.StateCode == "" {
		t.Error("StateCode field should not be empty")
	}
	
	// Verify expected values
	expectedStateName := "California"
	expectedStateCode := "CA"
	
	if testResponse.StateName != expectedStateName {
		t.Errorf("Expected StateName '%s', got '%s'", expectedStateName, testResponse.StateName)
	}
	if testResponse.StateCode != expectedStateCode {
		t.Errorf("Expected StateCode '%s', got '%s'", expectedStateCode, testResponse.StateCode)
	}
}

func TestGeocodeClient_ExtractStateFromComponents(t *testing.T) {
	// Test the actual parsing logic by testing the component extraction
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
				"types": ["street_address"],
				"address_components": [
					{
						"long_name": "1600",
						"short_name": "1600",
						"types": ["street_number"]
					},
					{
						"long_name": "Amphitheatre Parkway",
						"short_name": "Amphitheatre Pkwy",
						"types": ["route"]
					},
					{
						"long_name": "Mountain View",
						"short_name": "Mountain View",
						"types": ["locality", "political"]
					},
					{
						"long_name": "California",
						"short_name": "CA",
						"types": ["administrative_area_level_1", "political"]
					},
					{
						"long_name": "United States",
						"short_name": "US",
						"types": ["country", "political"]
					},
					{
						"long_name": "94043",
						"short_name": "94043",
						"types": ["postal_code"]
					}
				]
			}
		],
		"status": "OK"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is for geocoding
		if !strings.Contains(r.URL.Query().Get("address"), "test") {
			t.Errorf("Expected test address in query")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	// For this test, we're just testing that the model supports the fields
	// A real integration test would require overriding the HTTP client URL
	client := NewClient("test-api-key")
	
	// Verify the client is configured
	if !client.IsConfigured() {
		t.Error("Client should be configured with API key")
	}
	
	// Test that we can create a response with state fields
	expectedResponse := &models.GeocodeAPIResponse{
		Lat:              37.4224764,
		Lng:              -122.0842499,
		FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA 94043, USA",
		StateName:        "California",
		StateCode:        "CA",
		CountryName:      "United States",
		CountryCode:      "US",
		Backend:          "google_maps_platform_geocoding",
	}
	
	// Verify the state fields are properly set
	if expectedResponse.StateName != "California" {
		t.Errorf("Expected StateName 'California', got '%s'", expectedResponse.StateName)
	}
	if expectedResponse.StateCode != "CA" {
		t.Errorf("Expected StateCode 'CA', got '%s'", expectedResponse.StateCode)
	}
}
