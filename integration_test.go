// +build integration

package main

import (
	"bytes"
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

// Test the health endpoint
func TestIntegration_HealthEndpoint(t *testing.T) {
	db := &mockIntegrationDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	router := mux.NewRouter()
	router.Use(middleware.CORS())
	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET")

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

// Test unsupported version endpoint
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

// Test geocoding endpoint with authentication
func TestIntegration_GeocodeEndpoint(t *testing.T) {
	db := &mockIntegrationDB{}
	db.init()
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	rateLimiter := middleware.NewRateLimiter()

	// Create API key for testing
	testAPIKey := "test_live_sk_123456789"
	keyHash := database.HashAPIKey(testAPIKey)
	testKey := &models.APIKey{
		ID:                 "test-key-id",
		KeyHash:            keyHash,
		Name:               "Test Key",
		IsActive:           true,
		RateLimitPerSecond: 10,
	}
	db.apiKeys[keyHash] = testKey

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.APIKeyAuth(db))
	v1.Use(rateLimiter.RateLimit())
	v1.HandleFunc("/geocode", handlers.HandleGeocode).Methods("GET")

	// Test missing API key
	req := httptest.NewRequest("GET", "/v1/geocode?address=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing API key, got %d", w.Code)
	}

	// Test invalid API key
	req = httptest.NewRequest("GET", "/v1/geocode?address=test&key=invalid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid API key, got %d", w.Code)
	}

	// Test missing address parameter
	req = httptest.NewRequest("GET", "/v1/geocode?key="+testAPIKey, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing address, got %d", w.Code)
	}

	// Test valid request (will fail because geocoding client not configured)
	req = httptest.NewRequest("GET", "/v1/geocode?address=test&key="+testAPIKey, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for unconfigured geocoding client, got %d", w.Code)
	}

	// Check rate limit headers are present
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Rate limit headers should be present")
	}
}

// Test the structured geocoding endpoint  
func TestIntegration_GeocodeStructuredEndpoint(t *testing.T) {
	db := &mockIntegrationDB{}
	db.init()
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	rateLimiter := middleware.NewRateLimiter()

	// Create API key for testing
	testAPIKey := "test_live_sk_123456789"
	keyHash := database.HashAPIKey(testAPIKey)
	testKey := &models.APIKey{
		ID:                 "test-key-id",
		KeyHash:            keyHash,
		Name:               "Test Key",
		IsActive:           true,
		RateLimitPerSecond: 10,
	}
	db.apiKeys[keyHash] = testKey

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.APIKeyAuth(db))
	v1.Use(rateLimiter.RateLimit())
	v1.HandleFunc("/geocode_structured", handlers.HandleGeocodeStructured).Methods("GET")

	// Test missing API key
	req := httptest.NewRequest("GET", "/v1/geocode_structured?address_line_1=123+Main+St&city=San+Francisco&state=CA&country=USA", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing API key, got %d", w.Code)
	}

	// Test empty address (all fields empty)
	req = httptest.NewRequest("GET", "/v1/geocode_structured?key="+testAPIKey, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty address, got %d", w.Code)
	}

	// Test valid structured address (will fail because geocoding client not configured)
	req = httptest.NewRequest("GET", "/v1/geocode_structured?address_line_1=123+Main+St&city=San+Francisco&state=CA&country=USA&key="+testAPIKey, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for unconfigured geocoding client, got %d", w.Code)
	}

	// Check rate limit headers are present
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Rate limit headers should be present")
	}
}

// Test IP geolocation endpoint
func TestIntegration_GeoIPEndpoint(t *testing.T) {
	db := &mockIntegrationDB{}
	db.init()
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	rateLimiter := middleware.NewRateLimiter()

	// Create API key for testing
	testAPIKey := "test_live_sk_123456789"
	keyHash := database.HashAPIKey(testAPIKey)
	testKey := &models.APIKey{
		ID:                 "test-key-id",
		KeyHash:            keyHash,
		Name:               "Test Key",
		IsActive:           true,
		RateLimitPerSecond: 10,
	}
	db.apiKeys[keyHash] = testKey

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.APIKeyAuth(db))
	v1.Use(rateLimiter.RateLimit())
	v1.HandleFunc("/geoip", handlers.HandleGeoIP).Methods("GET")

	// Test missing IP parameter
	req := httptest.NewRequest("GET", "/v1/geoip?key="+testAPIKey, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing IP, got %d", w.Code)
	}

	// Test invalid IP format
	req = httptest.NewRequest("GET", "/v1/geoip?ip=invalid&key="+testAPIKey, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid IP, got %d", w.Code)
	}

	// Test valid IP (will make external API call)
	req = httptest.NewRequest("GET", "/v1/geoip?ip=8.8.8.8&key="+testAPIKey, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return 200 because IPinfo works without API key
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for successful IP lookup, got %d", w.Code)
	}
	
	// Verify response contains expected fields
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse IP response: %v", err)
	}
	
	if response["ip"] != "8.8.8.8" {
		t.Errorf("Expected IP to be '8.8.8.8', got %v", response["ip"])
	}
}

// Test reverse geocoding endpoint  
func TestIntegration_ReverseGeocodeEndpoint(t *testing.T) {
	db := &mockIntegrationDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	// Initialize the mock database
	db.init()
	
	// Create API key for testing
	testAPIKey := "test-key"
	keyHash := database.HashAPIKey(testAPIKey)
	apiKey := &models.APIKey{
		ID:                 "test-key-id",
		KeyHash:            keyHash,
		Name:               "Test Key",
		IsActive:           true,
		RateLimitPerSecond: 10,
	}
	db.apiKeys[keyHash] = apiKey

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.APIKeyAuth(db))
	v1.HandleFunc("/reverse_geocode", handlers.HandleReverseGeocode).Methods("GET")

	// Test valid coordinates
	req := httptest.NewRequest("GET", "/v1/reverse_geocode?lat=37.422476&lng=-122.084250&key=test-key", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Test valid request (will fail because geocoding client not configured)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for unconfigured geocoding client, got %d", w.Code)
	}

	// Test missing coordinates
	req = httptest.NewRequest("GET", "/v1/reverse_geocode?key=test-key", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing coordinates, got %d", w.Code)
	}

	// Test invalid latitude
	req = httptest.NewRequest("GET", "/v1/reverse_geocode?lat=invalid&lng=-122.084250&key=test-key", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid latitude, got %d", w.Code)
	}

	// Test coordinate range validation
	req = httptest.NewRequest("GET", "/v1/reverse_geocode?lat=91&lng=-122.084250&key=test-key", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for latitude out of range, got %d", w.Code)
	}

	// Test without API key
	req = httptest.NewRequest("GET", "/v1/reverse_geocode?lat=37.422476&lng=-122.084250", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing API key, got %d", w.Code)
	}

	// Cleanup
	_ = apiKey
}

// Test rate limiting functionality
func TestIntegration_RateLimit(t *testing.T) {
	db := &mockIntegrationDB{}
	db.init()
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
	rateLimiter := middleware.NewRateLimiter()

	// Create API key with low rate limit
	testAPIKey := "test_live_sk_123456789"
	keyHash := database.HashAPIKey(testAPIKey)
	testKey := &models.APIKey{
		ID:                 "test-key-id",
		KeyHash:            keyHash,
		Name:               "Test Key",
		IsActive:           true,
		RateLimitPerSecond: 2, // Very low limit for testing
	}
	db.apiKeys[keyHash] = testKey

	router := mux.NewRouter()
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.APIKeyAuth(db))
	v1.Use(rateLimiter.RateLimit())
	v1.HandleFunc("/geocode", handlers.HandleGeocode).Methods("GET")

	// Make multiple requests to trigger rate limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/v1/geocode?address=test&key="+testAPIKey, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if i < 2 {
			// First 2 requests should pass auth but fail on service unavailable
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("Request %d: Expected status 503, got %d", i, w.Code)
			}
		} else {
			// 3rd request should be rate limited
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: Expected status 429, got %d", i, w.Code)
			}
			
			var errorResp models.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
				t.Errorf("Failed to parse error response: %v", err)
			}
			
			if errorResp.Error.Code != "RATE_LIMIT_EXCEEDED" {
				t.Errorf("Expected error code 'RATE_LIMIT_EXCEEDED', got '%s'", errorResp.Error.Code)
			}
		}
	}
}

// Test admin API key management endpoints
func TestIntegration_AdminEndpoints(t *testing.T) {
	db := &mockIntegrationDB{}
	db.init()
	
	// Pre-create a test API key for admin operations
	testKeyHash := database.HashAPIKey("admin_test_key")
	testKey := &models.APIKey{
		ID:                 "test-key-id",
		KeyHash:            testKeyHash,
		Name:               "Admin Test Key",
		IsActive:           true,
		RateLimitPerSecond: 10,
		CreatedAt:          time.Now(),
	}
	db.apiKeys[testKeyHash] = testKey
	
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	router := mux.NewRouter()
	admin := router.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.BasicAuth("admin", "password"))
	admin.HandleFunc("/keys", handlers.HandleAdminKeys).Methods("GET", "POST")
	admin.HandleFunc("/keys/{key_id}/rate-limit", handlers.HandleUpdateAPIKeyRateLimit).Methods("PUT")
	admin.HandleFunc("/keys/{key_id}", handlers.HandleDeactivateAPIKey).Methods("DELETE")
	admin.HandleFunc("/stats", handlers.HandleAdminStats).Methods("GET")

	// Test unauthorized access
	req := httptest.NewRequest("GET", "/admin/keys", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unauthorized access, got %d", w.Code)
	}

	// Helper function to add basic auth
	addAuth := func(req *http.Request) {
		req.SetBasicAuth("admin", "password")
	}

	// Test getting all keys
	req = httptest.NewRequest("GET", "/admin/keys", nil)
	addAuth(req)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for admin keys, got %d", w.Code)
	}

	// Test creating a key
	createKeyReq := map[string]interface{}{
		"name":                     "Test API Key",
		"owner":                    "test-owner",
		"app_name":                "test-app",
		"environment":             "test",
		"rate_limit_per_second":   15,
	}
	reqBody, _ := json.Marshal(createKeyReq)
	req = httptest.NewRequest("POST", "/admin/keys", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for key creation, got %d", w.Code)
	}

	// Test updating rate limit
	updateReq := map[string]interface{}{
		"rate_limit_per_second": 20,
	}
	reqBody, _ = json.Marshal(updateReq)
	req = httptest.NewRequest("PUT", "/admin/keys/test-key-id/rate-limit", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	addAuth(req)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for rate limit update, got %d. Response: %s", w.Code, w.Body.String())
	}

	// Test deactivating key
	req = httptest.NewRequest("DELETE", "/admin/keys/test-key-id", nil)
	addAuth(req)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for key deactivation, got %d. Response: %s", w.Code, w.Body.String())
	}

	// Test getting stats
	req = httptest.NewRequest("GET", "/admin/stats", nil)
	addAuth(req)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for admin stats, got %d", w.Code)
	}

	var stats models.Stats
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Errorf("Failed to parse stats response: %v", err)
	}
}

// Test cache eviction functionality
func TestIntegration_CacheEviction(t *testing.T) {
	// Create a database with small cache size to trigger eviction
	db := &mockIntegrationDB{}
	cacheService := cache.NewService(db, 3, 2) // Very small cache sizes

	// Test address cache eviction
	for i := 0; i < 5; i++ {
		address := fmt.Sprintf("test address %d", i)
		result := &geocoding.GeocodeResponse{
			Results: []geocoding.GeocodeResult{
				{FormattedAddress: address},
			},
		}
		err := cacheService.SetGeocodeResult(address, result)
		if err != nil {
			t.Errorf("Failed to set geocode cache: %v", err)
		}
	}

	// Check that eviction was called
	if !db.addressEvictionCalled {
		t.Error("Address cache eviction should have been called")
	}

	// Test IP cache eviction
	for i := 0; i < 4; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i)
		result := &geoip.IPInfoResponse{
			IP:      ip,
			City:    "Test City",
			Country: "US",
		}
		err := cacheService.SetIPResult(ip, result)
		if err != nil {
			t.Errorf("Failed to set IP cache: %v", err)
		}
	}

	// Check that eviction was called
	if !db.ipEvictionCalled {
		t.Error("IP cache eviction should have been called")
	}
}

// Test cache hit and miss functionality
func TestIntegration_CacheHitMiss(t *testing.T) {
	db := &mockIntegrationDB{}
	cacheService := cache.NewService(db, 1000, 1000)

	// Test cache miss
	_, hit := cacheService.GetGeocodeResult("nonexistent address")
	if hit {
		t.Error("Should be cache miss for nonexistent address")
	}

	// Test cache hit after setting
	testAddress := "123 Test Street"
	testResult := &geocoding.GeocodeResponse{
		Results: []geocoding.GeocodeResult{
			{FormattedAddress: testAddress},
		},
	}
	
	err := cacheService.SetGeocodeResult(testAddress, testResult)
	if err != nil {
		t.Errorf("Failed to set cache: %v", err)
	}

	cached, hit := cacheService.GetGeocodeResult(testAddress)
	if !hit {
		t.Error("Should be cache hit for existing address")
	}
	if cached == nil || len(cached.Results) == 0 || cached.Results[0].FormattedAddress != testAddress {
		t.Error("Cached result should match original")
	}
}

// Test CORS functionality
func TestIntegration_CORS(t *testing.T) {
	db := &mockIntegrationDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	router := mux.NewRouter()
	router.Use(middleware.CORS())
	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET", "OPTIONS")

	// Test OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", w.Code)
	}

	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	for header, expected := range expectedHeaders {
		if w.Header().Get(header) != expected {
			t.Errorf("Expected %s header to be '%s', got '%s'", header, expected, w.Header().Get(header))
		}
	}
}

// Test API key validation and lifecycle
func TestIntegration_APIKeyLifecycle(t *testing.T) {
	db := &mockIntegrationDB{}

	// Test key creation
	keyHash := database.HashAPIKey("test_key")
	key, err := db.CreateAPIKey(keyHash, "Test Key", "test-owner", "test-app", "test", 10)
	if err != nil {
		t.Errorf("Failed to create API key: %v", err)
	}
	if key.Name != "Test Key" || key.RateLimitPerSecond != 10 {
		t.Error("API key properties not set correctly")
	}

	// Test key retrieval
	retrieved, err := db.GetAPIKeyByHash(keyHash)
	if err != nil {
		t.Errorf("Failed to retrieve API key: %v", err)
	}
	if retrieved.Name != "Test Key" {
		t.Error("Retrieved key doesn't match created key")
	}

	// Test key deactivation
	err = db.DeactivateAPIKey(retrieved.ID)
	if err != nil {
		t.Errorf("Failed to deactivate API key: %v", err)
	}

	// Verify key is deactivated
	deactivated, _ := db.GetAPIKeyByHash(keyHash)
	if deactivated.IsActive {
		t.Error("Key should be deactivated")
	}
}

// Test error response formats
func TestIntegration_ErrorFormats(t *testing.T) {
	db := &mockIntegrationDB{}
	geocodeClient := geocoding.NewClient("")
	geoipClient := geoip.NewClient("")
	cacheService := cache.NewService(db, 1000, 1000)
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(handlers.HandleUnsupportedVersion)

	req := httptest.NewRequest("GET", "/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var errorResp models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
		t.Errorf("Failed to parse error response: %v", err)
	}

	// Check error format
	if errorResp.Error.Code == "" {
		t.Error("Error code should not be empty")
	}
	if errorResp.Error.Message == "" {
		t.Error("Error message should not be empty")
	}
	if errorResp.Error.Timestamp.IsZero() {
		t.Error("Error timestamp should be set")
	}
}

// Comprehensive mock database for integration testing
type mockIntegrationDB struct {
	apiKeys                     map[string]*models.APIKey
	addressCache                map[string]*models.AddressCache
	ipCache                     map[string]*models.IPCache
	reverseGeocodeCache         map[string]*models.ReverseGeocodeCache
	addressEvictionCalled       bool
	ipEvictionCalled            bool
	reverseGeocodeEvictionCalled bool
}

func (m *mockIntegrationDB) init() {
	if m.apiKeys == nil {
		m.apiKeys = make(map[string]*models.APIKey)
	}
	if m.addressCache == nil {
		m.addressCache = make(map[string]*models.AddressCache)
	}
	if m.ipCache == nil {
		m.ipCache = make(map[string]*models.IPCache)
	}
	if m.reverseGeocodeCache == nil {
		m.reverseGeocodeCache = make(map[string]*models.ReverseGeocodeCache)
	}
}

func (m *mockIntegrationDB) Close() error { return nil }
func (m *mockIntegrationDB) Ping() error  { return nil }

func (m *mockIntegrationDB) CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error) {
	m.init()
	key := &models.APIKey{
		ID:                 "integration-test-id",
		KeyHash:            keyHash,
		Name:               name,
		IsActive:           true,
		RateLimitPerSecond: rateLimitPerSecond,
		CreatedAt:          time.Now(),
	}
	m.apiKeys[keyHash] = key
	return key, nil
}

func (m *mockIntegrationDB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	m.init()
	if key, exists := m.apiKeys[keyHash]; exists {
		return key, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (m *mockIntegrationDB) UpdateAPIKeyUsage(keyID string) error {
	return nil
}

func (m *mockIntegrationDB) GetAllAPIKeys() ([]models.APIKey, error) {
	m.init()
	var keys []models.APIKey
	for _, key := range m.apiKeys {
		keys = append(keys, *key)
	}
	return keys, nil
}

func (m *mockIntegrationDB) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error {
	m.init()
	for _, key := range m.apiKeys {
		if key.ID == keyID {
			key.RateLimitPerSecond = rateLimitPerSecond
			return nil
		}
	}
	return fmt.Errorf("key not found")
}

func (m *mockIntegrationDB) DeactivateAPIKey(keyID string) error {
	m.init()
	for _, key := range m.apiKeys {
		if key.ID == keyID {
			key.IsActive = false
			return nil
		}
	}
	return fmt.Errorf("key not found")
}

func (m *mockIntegrationDB) GetAddressCache(queryHash string) (*models.AddressCache, error) {
	m.init()
	if cache, exists := m.addressCache[queryHash]; exists {
		return cache, nil
	}
	return nil, fmt.Errorf("cache not found")
}

func (m *mockIntegrationDB) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	m.init()
	
	// Simulate cache size check and eviction
	if len(m.addressCache) >= maxCacheSize {
		m.addressEvictionCalled = true
		// Simulate deletion of oldest entries
		deleteCount := max(1, maxCacheSize/10)
		deleted := 0
		for hash := range m.addressCache {
			delete(m.addressCache, hash)
			deleted++
			if deleted >= deleteCount {
				break
			}
		}
	}
	
	m.addressCache[queryHash] = &models.AddressCache{
		ID:           1,
		QueryHash:    queryHash,
		QueryText:    queryText,
		ResponseData: responseData,
		CreatedAt:    time.Now(),
	}
	return nil
}

func (m *mockIntegrationDB) GetIPCache(ipAddress string) (*models.IPCache, error) {
	m.init()
	if cache, exists := m.ipCache[ipAddress]; exists {
		return cache, nil
	}
	return nil, fmt.Errorf("cache not found")
}

func (m *mockIntegrationDB) SetIPCache(ipAddress, responseData string, maxCacheSize int) error {
	m.init()
	
	// Simulate cache size check and eviction
	if len(m.ipCache) >= maxCacheSize {
		m.ipEvictionCalled = true
		// Simulate deletion of oldest entries
		deleteCount := max(1, maxCacheSize/10)
		deleted := 0
		for ip := range m.ipCache {
			delete(m.ipCache, ip)
			deleted++
			if deleted >= deleteCount {
				break
			}
		}
	}
	
	m.ipCache[ipAddress] = &models.IPCache{
		ID:           1,
		IPAddress:    ipAddress,
		ResponseData: responseData,
		CreatedAt:    time.Now(),
	}
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

func (m *mockIntegrationDB) LogActivity(apiKeyName, endpoint, queryText string, resultCount, responseTimeMs int, apiSource string, cacheHit bool, ipAddress, userAgent string) error {
	return nil
}

func (m *mockIntegrationDB) GetRecentActivity() ([]models.ActivityLog, error) {
	return []models.ActivityLog{}, nil
}

func (m *mockIntegrationDB) GetAPIKeyUsageSummary(page, pageSize int) (*models.UsageSummaryResponse, error) {
	return &models.UsageSummaryResponse{
		APIKeys:    []models.APIKeyUsageSummary{},
		TotalCount: 0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
	}, nil
}

func (m *mockIntegrationDB) GetReverseGeocodeCache(queryHash string) (*models.ReverseGeocodeCache, error) {
	m.init()
	if cache, exists := m.reverseGeocodeCache[queryHash]; exists {
		return cache, nil
	}
	return nil, fmt.Errorf("no rows")
}

func (m *mockIntegrationDB) SetReverseGeocodeCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	m.init()
	
	// Simulate eviction when hitting max size
	if len(m.reverseGeocodeCache) >= maxCacheSize {
		m.reverseGeocodeEvictionCalled = true
		// Remove one entry to simulate eviction
		for k := range m.reverseGeocodeCache {
			delete(m.reverseGeocodeCache, k)
			break
		}
	}
	
	m.reverseGeocodeCache[queryHash] = &models.ReverseGeocodeCache{
		ID:           len(m.reverseGeocodeCache) + 1,
		QueryHash:    queryHash,
		QueryText:    queryText,
		ResponseData: responseData,
		CreatedAt:    time.Now(),
	}
	return nil
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
