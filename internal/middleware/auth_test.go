package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/models"
)

func TestBasicAuth(t *testing.T) {
	username := "admin"
	password := "secret"

	middleware := BasicAuth(username, password)

	// Create a simple handler to test
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware(handler)

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{"Valid credentials", "admin", "secret", http.StatusOK},
		{"Invalid username", "wrong", "secret", http.StatusUnauthorized},
		{"Invalid password", "admin", "wrong", http.StatusUnauthorized},
		{"Empty credentials", "", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin", nil)
			if tt.username != "" || tt.password != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusUnauthorized {
				authHeader := w.Header().Get("WWW-Authenticate")
				if authHeader == "" {
					t.Error("Expected WWW-Authenticate header for unauthorized request")
				}
			}
		})
	}
}

func TestBasicAuth_NoAuthHeader(t *testing.T) {
	middleware := BasicAuth("admin", "secret")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// Mock database for middleware testing
type mockAuthDB struct {
	apiKeys map[string]*models.APIKey
}

func newMockAuthDB() *mockAuthDB {
	return &mockAuthDB{
		apiKeys: make(map[string]*models.APIKey),
	}
}

func (m *mockAuthDB) Close() error { return nil }
func (m *mockAuthDB) Ping() error  { return nil }

func (m *mockAuthDB) CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error) {
	return nil, nil
}

func (m *mockAuthDB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	if key, exists := m.apiKeys[keyHash]; exists {
		return key, nil
	}
	return nil, fmt.Errorf("api key not found") // Simulate not found error
}

func (m *mockAuthDB) UpdateAPIKeyUsage(keyID string) error {
	return nil
}

func (m *mockAuthDB) GetAllAPIKeys() ([]models.APIKey, error)                          { return nil, nil }
func (m *mockAuthDB) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error { return nil }
func (m *mockAuthDB) DeactivateAPIKey(keyID string) error                              { return nil }
func (m *mockAuthDB) GetAddressCache(queryHash string) (*models.AddressCache, error)   { return nil, nil }
func (m *mockAuthDB) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	return nil
}
func (m *mockAuthDB) GetIPCache(ipAddress string) (*models.IPCache, error)              { return nil, nil }
func (m *mockAuthDB) SetIPCache(ipAddress, responseData string, maxCacheSize int) error { return nil }
func (m *mockAuthDB) LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error {
	return nil
}
func (m *mockAuthDB) GetStats() (*models.Stats, error) { return nil, nil }
func (m *mockAuthDB) UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error {
	return nil
}
func (m *mockAuthDB) GetRecentActivity() ([]models.ActivityLog, error) { return nil, nil }
func (m *mockAuthDB) LogActivity(apiKeyName, endpoint, queryText string, resultCount, responseTimeMs int, apiSource string, cacheHit bool, ipAddress, userAgent string) error {
	return nil
}
func (m *mockAuthDB) GetAPIKeyUsageSummary(page, pageSize int) (*models.UsageSummaryResponse, error) {
	return nil, nil
}
func (m *mockAuthDB) GetReverseGeocodeCache(queryHash string) (*models.ReverseGeocodeCache, error) {
	return nil, nil
}
func (m *mockAuthDB) SetReverseGeocodeCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	return nil
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	db := newMockAuthDB()
	middleware := APIKeyAuth(db)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/v1/geocode", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	db := newMockAuthDB()
	middleware := APIKeyAuth(db)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/v1/geocode?key=invalid-key", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	db := newMockAuthDB()

	// Add a valid API key to the mock database
	apiKey := &models.APIKey{
		ID:                 "test-id",
		KeyHash:            database.HashAPIKey("valid-key"),
		Name:               "test-key",
		IsActive:           true,
		RateLimitPerSecond: 10,
	}
	db.apiKeys[apiKey.KeyHash] = apiKey

	middleware := APIKeyAuth(db)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if API key is in context
		contextKey := r.Context().Value(APIKeyContextKey)
		if contextKey == nil {
			t.Error("Expected API key in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/v1/geocode?key=valid-key", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyAuth_InactiveKey(t *testing.T) {
	db := newMockAuthDB()

	// Add an inactive API key to the mock database
	apiKey := &models.APIKey{
		ID:                 "test-id",
		KeyHash:            database.HashAPIKey("inactive-key"),
		Name:               "test-key",
		IsActive:           false, // Inactive
		RateLimitPerSecond: 10,
	}
	db.apiKeys[apiKey.KeyHash] = apiKey

	middleware := APIKeyAuth(db)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/v1/geocode?key=inactive-key", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
