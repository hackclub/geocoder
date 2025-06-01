// +build integration

package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hackclub/geocoder/internal/api"
	"github.com/hackclub/geocoder/internal/middleware"
	"github.com/hackclub/geocoder/internal/models"
)

// Minimal integration tests focusing only on HTTP routing and basic responses
func TestMinimal_HealthEndpoint(t *testing.T) {
	// Create handlers with nil dependencies (only for health check)
	handlers := api.NewHandlers(nil, nil, nil, nil)
	
	router := mux.NewRouter()
	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET")
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	
	var health models.HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}
	
	// Health check should work even with nil dependencies
	if health.Status == "" {
		t.Error("Health status should not be empty")
	}
}

func TestMinimal_DocsEndpoint(t *testing.T) {
	// Docs endpoint should work without any dependencies
	handlers := api.NewHandlers(nil, nil, nil, nil)
	
	router := mux.NewRouter()
	router.HandleFunc("/", handlers.HandleDocs).Methods("GET")
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Docs request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Type") != "text/html" {
		t.Error("Expected HTML content type")
	}
}

func TestMinimal_UnsupportedVersion(t *testing.T) {
	handlers := api.NewHandlers(nil, nil, nil, nil)
	
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(handlers.HandleUnsupportedVersion)
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	resp, err := http.Get(server.URL + "/geocode")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
	
	var errorResp models.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	
	if errorResp.Error.Code != "UNSUPPORTED_VERSION" {
		t.Errorf("Expected error code 'UNSUPPORTED_VERSION', got '%s'", errorResp.Error.Code)
	}
}

func TestMinimal_CORS(t *testing.T) {
	handlers := api.NewHandlers(nil, nil, nil, nil)
	
	router := mux.NewRouter()
	router.Use(middleware.CORS())
	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET")
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	// Check CORS headers
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers not set correctly")
	}
	
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS methods header not set")
	}
}

func TestMinimal_TestMapEndpoint(t *testing.T) {
	// Test the new test map endpoint
	handlers := api.NewHandlers(nil, nil, nil, nil)
	
	router := mux.NewRouter()
	router.HandleFunc("/test", handlers.HandleTestMap).Methods("GET")
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	resp, err := http.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("Test map request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Type") != "text/html" {
		t.Error("Expected HTML content type")
	}
}
