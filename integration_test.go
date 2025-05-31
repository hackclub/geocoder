// +build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/hackclub/geocoder/internal/api"
	"github.com/hackclub/geocoder/internal/cache"
	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/geocoding"
	"github.com/hackclub/geocoder/internal/geoip"
	"github.com/hackclub/geocoder/internal/middleware"
	"github.com/hackclub/geocoder/internal/models"
)

// This is a simple integration test to verify the overall system works
func TestIntegration_HealthEndpoint(t *testing.T) {
	// Create mock dependencies
	db := &mockIntegrationDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	// Set up router similar to main.go
	router := mux.NewRouter()
	router.Use(middleware.CORS())
	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET")

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var health models.HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Errorf("Failed to parse health response: %v", err)
	}

	if health.Status == "" {
		t.Error("Health status should not be empty")
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers not set correctly")
	}
}

func TestIntegration_UnsupportedVersion(t *testing.T) {
	db := &mockIntegrationDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(handlers.HandleUnsupportedVersion)

	req := httptest.NewRequest("GET", "/geocode", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var errorResp models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
		t.Errorf("Failed to parse error response: %v", err)
	}

	if errorResp.Error.Code != "UNSUPPORTED_VERSION" {
		t.Errorf("Expected error code 'UNSUPPORTED_VERSION', got '%s'", errorResp.Error.Code)
	}
}

// Comprehensive mock database for integration testing
type mockIntegrationDB struct{}

func (m *mockIntegrationDB) Close() error { return nil }
func (m *mockIntegrationDB) Ping() error  { return nil }

func (m *mockIntegrationDB) CreateAPIKey(keyHash, name string, rateLimitPerSecond int) (*models.APIKey, error) {
	return &models.APIKey{
		ID:                 "integration-test-id",
		KeyHash:            keyHash,
		Name:               name,
		IsActive:           true,
		RateLimitPerSecond: rateLimitPerSecond,
	}, nil
}

func (m *mockIntegrationDB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	return nil, database.HashAPIKey("not found")
}

func (m *mockIntegrationDB) UpdateAPIKeyUsage(keyID string) error {
	return nil
}

func (m *mockIntegrationDB) GetAllAPIKeys() ([]models.APIKey, error) {
	return []models.APIKey{
		{
			ID:                 "test-1",
			Name:               "Test Key 1",
			IsActive:           true,
			RateLimitPerSecond: 10,
		},
	}, nil
}

func (m *mockIntegrationDB) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error {
	return nil
}

func (m *mockIntegrationDB) DeactivateAPIKey(keyID string) error {
	return nil
}

func (m *mockIntegrationDB) GetAddressCache(queryHash string) (*models.AddressCache, error) {
	return nil, nil
}

func (m *mockIntegrationDB) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	return nil
}

func (m *mockIntegrationDB) GetIPCache(ipAddress string) (*models.IPCache, error) {
	return nil, nil
}

func (m *mockIntegrationDB) SetIPCache(ipAddress, responseData string, maxCacheSize int) error {
	return nil
}

func (m *mockIntegrationDB) LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error {
	return nil
}

func (m *mockIntegrationDB) GetStats() (*models.Stats, error) {
	return &models.Stats{
		TotalRequests:       1000,
		CacheHitRate:       85.5,
		AverageResponseTime: 120.0,
		ActiveAPIKeys:      3,
		TodaysRequests:     50,
		TodaysCacheHits:    42,
	}, nil
}

func (m *mockIntegrationDB) UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error {
	return nil
}
