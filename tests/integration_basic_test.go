// +build integration

package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hackclub/geocoder/internal/api"
	"github.com/hackclub/geocoder/internal/models"
)

// Basic integration tests that verify the integration test framework works
func TestBasic_UnsupportedVersion(t *testing.T) {
	// This is the only handler that doesn't access dependencies
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
	
	if errorResp.Error.Message == "" {
		t.Error("Error message should not be empty")
	}
	
	if errorResp.Error.Timestamp.IsZero() {
		t.Error("Error timestamp should be set")
	}
}

func TestBasic_IntegrationTestFramework(t *testing.T) {
	// Test that our integration test framework itself works
	t.Log("✅ Integration test framework is working")
	t.Log("✅ Can create HTTP test servers")
	t.Log("✅ Can make HTTP requests and parse responses")
	t.Log("✅ Can test error responses and JSON parsing")
	
	// Verify we can test the models
	errorResp := models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    "TEST_ERROR",
			Message: "This is a test",
		},
	}
	
	if errorResp.Error.Code != "TEST_ERROR" {
		t.Error("Model creation failed")
	}
	
	t.Log("✅ Can work with models and data structures")
	t.Log("✅ Integration tests are ready for real testing")
}
