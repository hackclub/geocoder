// +build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hackclub/geocoder/internal/api"
	"github.com/hackclub/geocoder/internal/cache"
	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/geocoding"
	"github.com/hackclub/geocoder/internal/geoip"
	"github.com/hackclub/geocoder/internal/middleware"
	"github.com/hackclub/geocoder/internal/models"
)

var (
	testDB       database.DatabaseInterface
	testAPIKey   string
	testServer   *httptest.Server
	testHandlers *api.Handlers
)

func TestMain(m *testing.M) {
	// Setup test database
	setupTestDB()
	
	// Setup test server
	setupTestServer()
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	cleanup()
	
	os.Exit(code)
}

func setupTestDB() {
	// Always use mock database for integration tests to avoid external dependencies
	// Real database integration can be tested manually
	testDB = &mockFullIntegrationDB{
		apiKeys: make(map[string]*models.APIKey),
		usageLogs: make([]models.UsageLog, 0),
	}
}

func setupTestServer() {
	geocodeClient := geocoding.NewClient(os.Getenv("GOOGLE_GEOCODING_API_KEY"))
	geoipClient := geoip.NewClient(os.Getenv("IPINFO_API_KEY"))
	cacheService := cache.NewService(testDB, 1000, 500)
	testHandlers = api.NewHandlers(testDB, geocodeClient, geoipClient, cacheService)
	
	// Create test API key
	keyPlain := "test-integration-key-12345"
	keyHash := database.HashAPIKey(keyPlain)
	testAPIKey = keyPlain
	
	testDB.CreateAPIKey(keyHash, "Integration Test Key", "test", "geocoder", "test", 100)
	
	// Setup router
	router := mux.NewRouter()
	
	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter()
	
	// Add middleware
	router.Use(middleware.CORS())
	
	// API routes
	apiRouter := router.PathPrefix("/v1").Subrouter()
	apiRouter.Use(middleware.APIKeyAuth(testDB))
	apiRouter.Use(rateLimiter.RateLimit())
	
	apiRouter.HandleFunc("/geocode", testHandlers.HandleGeocode).Methods("GET")
	apiRouter.HandleFunc("/geoip", testHandlers.HandleGeoIP).Methods("GET")
	
	// Other routes
	router.HandleFunc("/health", testHandlers.HandleHealth).Methods("GET")
	router.HandleFunc("/", testHandlers.HandleDocs).Methods("GET")
	
	// Admin routes
	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.BasicAuth("admin", "admin"))
	
	adminRouter.HandleFunc("/ws", testHandlers.HandleWebSocket).Methods("GET")
	adminRouter.HandleFunc("/keys", testHandlers.HandleAdminKeys).Methods("POST", "GET")
	adminRouter.HandleFunc("/stats", testHandlers.HandleAdminStats).Methods("GET")
	adminRouter.HandleFunc("/dashboard", testHandlers.HandleAdminDashboard).Methods("GET")
	
	// Catch unsupported versions
	router.PathPrefix("/").HandlerFunc(testHandlers.HandleUnsupportedVersion)
	
	testServer = httptest.NewServer(router)
}

func cleanup() {
	if testServer != nil {
		testServer.Close()
	}
	if testDB != nil {
		testDB.Close()
	}
}

// Test complete geocoding flow
func TestIntegration_GeocodingFlow(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expectError bool
		errorCode   string
	}{
		{
			name:        "Valid address",
			address:     "1600 Amphitheatre Parkway, Mountain View, CA",
			expectError: false,
		},
		{
			name:        "Valid international address",
			address:     "10 Downing Street, London, UK",
			expectError: false,
		},
		{
			name:        "Empty address",
			address:     "",
			expectError: true,
			errorCode:   "INVALID_ADDRESS",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/v1/geocode?address=%s&key=%s", 
				testServer.URL, tt.address, testAPIKey)
			
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()
			
			if tt.expectError {
				if resp.StatusCode == http.StatusOK {
					t.Errorf("Expected error but got success")
				}
				
				var errorResp models.ErrorResponse
				if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				
				if errorResp.Error.Code != tt.errorCode {
					t.Errorf("Expected error code %s, got %s", tt.errorCode, errorResp.Error.Code)
				}
			} else {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected success but got status %d", resp.StatusCode)
				}
				
				// Check rate limit headers
				if resp.Header.Get("X-RateLimit-Limit") == "" {
					t.Error("Missing rate limit headers")
				}
			}
		})
	}
}

// Test IP geolocation flow
func TestIntegration_GeoIPFlow(t *testing.T) {
	tests := []struct {
		name        string
		ip          string
		expectError bool
		errorCode   string
	}{
		{
			name:        "Valid IPv4",
			ip:          "8.8.8.8",
			expectError: false,
		},
		{
			name:        "Valid IPv6",
			ip:          "2001:4860:4860::8888",
			expectError: false,
		},
		{
			name:        "Invalid IP",
			ip:          "invalid-ip",
			expectError: true,
			errorCode:   "INVALID_IP",
		},
		{
			name:        "Empty IP",
			ip:          "",
			expectError: true,
			errorCode:   "INVALID_IP",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/v1/geoip?ip=%s&key=%s", 
				testServer.URL, tt.ip, testAPIKey)
			
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()
			
			if tt.expectError {
				if resp.StatusCode == http.StatusOK {
					t.Errorf("Expected error but got success")
				}
				
				var errorResp models.ErrorResponse
				if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				
				if errorResp.Error.Code != tt.errorCode {
					t.Errorf("Expected error code %s, got %s", tt.errorCode, errorResp.Error.Code)
				}
			} else {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected success but got status %d", resp.StatusCode)
				}
			}
		})
	}
}

// Test authentication and authorization
func TestIntegration_Authentication(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		expectErr bool
		errorCode string
	}{
		{
			name:      "Valid API key",
			apiKey:    testAPIKey,
			expectErr: false,
		},
		{
			name:      "Invalid API key",
			apiKey:    "invalid-key",
			expectErr: true,
			errorCode: "INVALID_API_KEY",
		},
		{
			name:      "Missing API key",
			apiKey:    "",
			expectErr: true,
			errorCode: "INVALID_API_KEY",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/v1/geocode?address=test&key=%s", 
				testServer.URL, tt.apiKey)
			
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()
			
			if tt.expectErr {
				if resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("Expected 401, got %d", resp.StatusCode)
				}
			} else {
				if resp.StatusCode == http.StatusUnauthorized {
					t.Error("Expected success but got unauthorized")
				}
			}
		})
	}
}

// Test rate limiting
func TestIntegration_RateLimit(t *testing.T) {
	// Create API key with low rate limit for testing
	lowLimitKey := "test-rate-limit-key"
	keyHash := database.HashAPIKey(lowLimitKey)
	testDB.CreateAPIKey(keyHash, "Rate Limit Test", "test", "geocoder", "test", 2)
	
	url := fmt.Sprintf("%s/v1/geocode?address=test&key=%s", 
		testServer.URL, lowLimitKey)
	
	// Make requests up to the limit
	for i := 0; i < 2; i++ {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		resp.Body.Close()
		
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Errorf("Got rate limited on request %d", i+1)
		}
	}
	
	// Next request should be rate limited
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Rate limit test request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", resp.StatusCode)
	}
	
	// Wait and try again
	time.Sleep(1100 * time.Millisecond)
	
	resp, err = http.Get(url)
	if err != nil {
		t.Fatalf("Post-wait request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Error("Still rate limited after waiting")
	}
}

// Test admin endpoints
func TestIntegration_AdminEndpoints(t *testing.T) {
	client := &http.Client{}
	
	// Test admin stats
	req, _ := http.NewRequest("GET", testServer.URL+"/admin/stats", nil)
	req.SetBasicAuth("admin", "admin")
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Admin stats request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	
	var stats models.Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode stats: %v", err)
	}
	
	// Test admin key creation
	keyData := map[string]interface{}{
		"name":   "Integration Test Key 2",
		"prefix": "test",
	}
	
	keyJSON, _ := json.Marshal(keyData)
	req, _ = http.NewRequest("POST", testServer.URL+"/admin/keys", bytes.NewBuffer(keyJSON))
	req.SetBasicAuth("admin", "admin")
	req.Header.Set("Content-Type", "application/json")
	
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Admin key creation failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

// Test WebSocket connection
func TestIntegration_WebSocket(t *testing.T) {
	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(testServer.URL, "http://", "ws://", 1) + "/admin/ws"
	
	// Set up WebSocket connection with basic auth
	header := http.Header{}
	header.Set("Authorization", "Basic YWRtaW46YWRtaW4=") // admin:admin
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()
	
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	// Make a geocoding request to trigger WebSocket update
	go func() {
		time.Sleep(100 * time.Millisecond)
		url := fmt.Sprintf("%s/v1/geocode?address=test&key=%s", 
			testServer.URL, testAPIKey)
		http.Get(url)
	}()
	
	// Read WebSocket message
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read WebSocket message: %v", err)
	}
	
	// Parse message
	var wsMessage map[string]interface{}
	if err := json.Unmarshal(message, &wsMessage); err != nil {
		t.Fatalf("Failed to parse WebSocket message: %v", err)
	}
	
	// Verify message structure
	if wsMessage["type"] == nil {
		t.Error("WebSocket message missing type field")
	}
}

// Test caching behavior
func TestIntegration_Caching(t *testing.T) {
	address := "1600 Amphitheatre Parkway, Mountain View, CA"
	url := fmt.Sprintf("%s/v1/geocode?address=%s&key=%s", 
		testServer.URL, address, testAPIKey)
	
	// First request (cache miss)
	start := time.Now()
	resp1, err := http.Get(url)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()
	firstDuration := time.Since(start)
	
	// Second request (cache hit)
	start = time.Now()
	resp2, err := http.Get(url)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	resp2.Body.Close()
	secondDuration := time.Since(start)
	
	// Cache hit should be faster
	if secondDuration >= firstDuration {
		t.Logf("First request: %v, Second request: %v", firstDuration, secondDuration)
		// Note: This might not always be true in tests due to timing variations
		// So we'll just log it instead of failing
	}
	
	if resp1.StatusCode != http.StatusOK || resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected both requests to succeed")
	}
}

// Test health endpoint
func TestIntegration_Health(t *testing.T) {
	resp, err := http.Get(testServer.URL + "/health")
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
	
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}
}

// Mock database for integration testing
type mockFullIntegrationDB struct {
	apiKeys   map[string]*models.APIKey
	usageLogs []models.UsageLog
}

func (m *mockFullIntegrationDB) Close() error { return nil }
func (m *mockFullIntegrationDB) Ping() error  { return nil }

func (m *mockFullIntegrationDB) CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error) {
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

func (m *mockFullIntegrationDB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	if key, exists := m.apiKeys[keyHash]; exists {
		return key, nil
	}
	return nil, fmt.Errorf("API key not found")
}

func (m *mockFullIntegrationDB) UpdateAPIKeyUsage(keyID string) error {
	return nil
}

func (m *mockFullIntegrationDB) GetAllAPIKeys() ([]models.APIKey, error) {
	keys := make([]models.APIKey, 0, len(m.apiKeys))
	for _, key := range m.apiKeys {
		keys = append(keys, *key)
	}
	return keys, nil
}

func (m *mockFullIntegrationDB) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error {
	return nil
}

func (m *mockFullIntegrationDB) DeactivateAPIKey(keyID string) error {
	return nil
}

func (m *mockFullIntegrationDB) GetAddressCache(queryHash string) (*models.AddressCache, error) {
	return nil, nil // Always cache miss for testing
}

func (m *mockFullIntegrationDB) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	return nil
}

func (m *mockFullIntegrationDB) GetIPCache(ipAddress string) (*models.IPCache, error) {
	return nil, nil // Always cache miss for testing
}

func (m *mockFullIntegrationDB) SetIPCache(ipAddress, responseData string, maxCacheSize int) error {
	return nil
}

func (m *mockFullIntegrationDB) LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error {
	log := models.UsageLog{
		APIKeyID:       apiKeyID,
		Endpoint:       endpoint,
		CacheHit:       cacheHit,
		ResponseTimeMs: responseTimeMs,
		CreatedAt:      time.Now(),
	}
	m.usageLogs = append(m.usageLogs, log)
	return nil
}

func (m *mockFullIntegrationDB) GetStats() (*models.Stats, error) {
	return &models.Stats{
		TotalRequests:       int64(len(m.usageLogs)),
		CacheHitRate:       75.0,
		AverageResponseTime: 120.0,
		ActiveAPIKeys:      len(m.apiKeys),
		TodaysRequests:     10,
		TodaysCacheHits:    7,
	}, nil
}

func (m *mockFullIntegrationDB) UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error {
	return nil
}

func (m *mockFullIntegrationDB) LogActivity(apiKeyName, endpoint, queryText string, resultCount, responseTimeMs int, apiSource string, cacheHit bool, ipAddress, userAgent string) error {
	return nil
}

func (m *mockFullIntegrationDB) GetRecentActivity() ([]models.ActivityLog, error) {
	return []models.ActivityLog{}, nil
}
