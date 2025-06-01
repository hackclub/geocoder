// +build integration

package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// Simple integration tests that focus on HTTP layer and basic functionality
func TestIntegrationSimple_HealthEndpoint(t *testing.T) {
	// Create minimal mock database
	db := &simpleMockDB{}
	
	// Create handlers with minimal dependencies
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 100, 100)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	
	// Setup router
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
	
	if health.Status == "" {
		t.Error("Health status should not be empty")
	}
}

func TestIntegrationSimple_DocsEndpoint(t *testing.T) {
	// Create minimal setup
	db := &simpleMockDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 100, 100)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	
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

func TestIntegrationSimple_Authentication(t *testing.T) {
	// Create mock database with a test API key
	db := &simpleMockDB{
		apiKeys: map[string]*models.APIKey{
			database.HashAPIKey("test-key"): {
				ID:                 "test-id",
				KeyHash:            database.HashAPIKey("test-key"),
				Name:               "Test Key",
				IsActive:           true,
				RateLimitPerSecond: 10,
			},
		},
	}
	
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 100, 100)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	
	router := mux.NewRouter()
	rateLimiter := middleware.NewRateLimiter()
	
	apiRouter := router.PathPrefix("/v1").Subrouter()
	apiRouter.Use(middleware.APIKeyAuth(db))
	apiRouter.Use(rateLimiter.RateLimit())
	apiRouter.HandleFunc("/geocode", handlers.HandleGeocode).Methods("GET")
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	tests := []struct {
		name           string
		apiKey         string
		expectStatus   int
		expectErrorCode string
	}{
		{
			name:         "Valid API key",
			apiKey:       "test-key",
			expectStatus: 400, // Will fail with INVALID_ADDRESS but pass auth
		},
		{
			name:            "Invalid API key",
			apiKey:          "invalid-key",
			expectStatus:    401,
			expectErrorCode: "INVALID_API_KEY",
		},
		{
			name:            "Missing API key",
			apiKey:          "",
			expectStatus:    401,
			expectErrorCode: "INVALID_API_KEY",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/v1/geocode?address=test&key=%s", server.URL, tt.apiKey)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, resp.StatusCode)
			}
			
			if tt.expectErrorCode != "" {
				var errorResp models.ErrorResponse
				if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				
				if errorResp.Error.Code != tt.expectErrorCode {
					t.Errorf("Expected error code %s, got %s", tt.expectErrorCode, errorResp.Error.Code)
				}
			}
		})
	}
}

func TestIntegrationSimple_UnsupportedVersion(t *testing.T) {
	db := &simpleMockDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 100, 100)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	
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

func TestIntegrationSimple_CORS(t *testing.T) {
	db := &simpleMockDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 100, 100)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	
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
}

func TestIntegrationSimple_AdminBasicAuth(t *testing.T) {
	db := &simpleMockDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 100, 100)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	
	router := mux.NewRouter()
	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.BasicAuth("admin", "admin"))
	adminRouter.HandleFunc("/stats", handlers.HandleAdminStats).Methods("GET")
	
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test without auth
	resp, err := http.Get(server.URL + "/admin/stats")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()
	
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 without auth, got %d", resp.StatusCode)
	}
	
	// Test with valid auth
	client := &http.Client{}
	req, _ := http.NewRequest("GET", server.URL+"/admin/stats", nil)
	req.SetBasicAuth("admin", "admin")
	
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Authenticated request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 with valid auth, got %d", resp.StatusCode)
	}
}

// Minimal mock database for testing HTTP layer only
type simpleMockDB struct {
	apiKeys map[string]*models.APIKey
}

func (m *simpleMockDB) Close() error { return nil }
func (m *simpleMockDB) Ping() error  { return nil }

func (m *simpleMockDB) CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error) {
	if m.apiKeys == nil {
		m.apiKeys = make(map[string]*models.APIKey)
	}
	
	key := &models.APIKey{
		ID:                 fmt.Sprintf("key-%d", len(m.apiKeys)),
		KeyHash:            keyHash,
		Name:               name,
		Owner:              owner,
		AppName:            appName,
		Environment:        environment,
		IsActive:           true,
		RateLimitPerSecond: rateLimitPerSecond,
		CreatedAt:          time.Now(),
	}
	m.apiKeys[keyHash] = key
	return key, nil
}

func (m *simpleMockDB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	if m.apiKeys == nil {
		return nil, fmt.Errorf("API key not found")
	}
	if key, exists := m.apiKeys[keyHash]; exists {
		return key, nil
	}
	return nil, fmt.Errorf("API key not found")
}

func (m *simpleMockDB) UpdateAPIKeyUsage(keyID string) error { return nil }

func (m *simpleMockDB) GetAllAPIKeys() ([]models.APIKey, error) {
	keys := make([]models.APIKey, 0)
	if m.apiKeys != nil {
		for _, key := range m.apiKeys {
			keys = append(keys, *key)
		}
	}
	return keys, nil
}

func (m *simpleMockDB) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error { return nil }
func (m *simpleMockDB) DeactivateAPIKey(keyID string) error                               { return nil }

// Cache operations - return nil to simulate cache miss
func (m *simpleMockDB) GetAddressCache(queryHash string) (*models.AddressCache, error) { return nil, nil }
func (m *simpleMockDB) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	return nil
}
func (m *simpleMockDB) GetIPCache(ipAddress string) (*models.IPCache, error) { return nil, nil }
func (m *simpleMockDB) SetIPCache(ipAddress, responseData string, maxCacheSize int) error {
	return nil
}

// Usage tracking
func (m *simpleMockDB) LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error {
	return nil
}

func (m *simpleMockDB) GetStats() (*models.Stats, error) {
	return &models.Stats{
		TotalRequests:       100,
		CacheHitRate:       75.0,
		AverageResponseTime: 120.0,
		ActiveAPIKeys:      2,
		TodaysRequests:     50,
		TodaysCacheHits:    37,
	}, nil
}

func (m *simpleMockDB) UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error {
	return nil
}

func (m *simpleMockDB) LogActivity(apiKeyName, endpoint, queryText string, resultCount, responseTimeMs int, apiSource string, cacheHit bool, ipAddress, userAgent string) error {
	return nil
}

func (m *simpleMockDB) GetRecentActivity() ([]models.ActivityLog, error) {
	return []models.ActivityLog{}, nil
}
