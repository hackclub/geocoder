package cache

import (
	"database/sql"
	"testing"
	"time"

	"github.com/hackclub/geocoder/internal/geocoding"
	"github.com/hackclub/geocoder/internal/geoip"
	"github.com/hackclub/geocoder/internal/models"
)

// Mock database for cache testing
type mockCacheDB struct {
	addressCache map[string]*models.AddressCache
	ipCache      map[string]*models.IPCache
}

func newMockCacheDB() *mockCacheDB {
	return &mockCacheDB{
		addressCache: make(map[string]*models.AddressCache),
		ipCache:      make(map[string]*models.IPCache),
	}
}

func (m *mockCacheDB) Close() error { return nil }
func (m *mockCacheDB) Ping() error  { return nil }

func (m *mockCacheDB) CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error) {
	return nil, nil
}
func (m *mockCacheDB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) { return nil, nil }
func (m *mockCacheDB) UpdateAPIKeyUsage(keyID string) error                   { return nil }
func (m *mockCacheDB) GetAllAPIKeys() ([]models.APIKey, error)                { return nil, nil }
func (m *mockCacheDB) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error {
	return nil
}
func (m *mockCacheDB) DeactivateAPIKey(keyID string) error { return nil }

func (m *mockCacheDB) GetAddressCache(queryHash string) (*models.AddressCache, error) {
	if cache, exists := m.addressCache[queryHash]; exists {
		return cache, nil
	}
	return nil, sql.ErrNoRows // Simulate sql.ErrNoRows
}

func (m *mockCacheDB) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	m.addressCache[queryHash] = &models.AddressCache{
		ID:           len(m.addressCache) + 1,
		QueryHash:    queryHash,
		QueryText:    queryText,
		ResponseData: responseData,
		CreatedAt:    time.Now(),
	}
	return nil
}

func (m *mockCacheDB) GetIPCache(ipAddress string) (*models.IPCache, error) {
	if cache, exists := m.ipCache[ipAddress]; exists {
		return cache, nil
	}
	return nil, sql.ErrNoRows // Simulate sql.ErrNoRows
}

func (m *mockCacheDB) SetIPCache(ipAddress, responseData string, maxCacheSize int) error {
	m.ipCache[ipAddress] = &models.IPCache{
		ID:           len(m.ipCache) + 1,
		IPAddress:    ipAddress,
		ResponseData: responseData,
		CreatedAt:    time.Now(),
	}
	return nil
}

func (m *mockCacheDB) LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error {
	return nil
}
func (m *mockCacheDB) GetStats() (*models.Stats, error) { return nil, nil }
func (m *mockCacheDB) UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error {
	return nil
}
func (m *mockCacheDB) GetRecentActivity() ([]models.ActivityLog, error) {
	return nil, nil
}
func (m *mockCacheDB) LogActivity(apiKeyName, endpoint, queryText string, resultCount, responseTimeMs int, apiSource string, cacheHit bool, ipAddress, userAgent string) error {
	return nil
}
func (m *mockCacheDB) GetAPIKeyUsageSummary(page, pageSize int) (*models.UsageSummaryResponse, error) {
	return nil, nil
}

func TestCacheService_GeocodeCache(t *testing.T) {
	db := newMockCacheDB()
	cache := NewService(db, 1000, 1000)

	address := "1600 Amphitheatre Parkway, Mountain View, CA"

	// Test cache miss
	result, hit := cache.GetGeocodeResult(address)
	if hit {
		t.Error("Expected cache miss for new address")
	}
	if result != nil {
		t.Error("Expected nil result for cache miss")
	}

	// Create mock geocode result
	geocodeResult := &geocoding.GeocodeResponse{
		Results: []geocoding.GeocodeResult{
			{
				FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA 94043, USA",
				Geometry: geocoding.GeocodeGeometry{
					Location: geocoding.GeocodeLocation{
						Lat: 37.4224764,
						Lng: -122.0842499,
					},
				},
			},
		},
		Status: "OK",
	}

	// Set cache
	err := cache.SetGeocodeResult(address, geocodeResult)
	if err != nil {
		t.Fatalf("Failed to set geocode cache: %v", err)
	}

	// Test cache hit
	result, hit = cache.GetGeocodeResult(address)
	if !hit {
		t.Error("Expected cache hit for cached address")
	}
	if result == nil {
		t.Fatal("Expected non-nil result for cache hit")
	}
	if result.Status != "OK" {
		t.Errorf("Expected status 'OK', got '%s'", result.Status)
	}
	if len(result.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result.Results))
	}
}

func TestCacheService_IPCache(t *testing.T) {
	db := newMockCacheDB()
	cache := NewService(db, 1000, 1000)

	ip := "8.8.8.8"

	// Test cache miss
	result, hit := cache.GetIPResult(ip)
	if hit {
		t.Error("Expected cache miss for new IP")
	}
	if result != nil {
		t.Error("Expected nil result for cache miss")
	}

	// Create mock IP result
	ipResult := &geoip.IPInfoResponse{
		IP:       "8.8.8.8",
		City:     "Mountain View",
		Region:   "California",
		Country:  "US",
		Loc:      "37.4056,-122.0775",
		Org:      "AS15169 Google LLC",
		Postal:   "94043",
		Timezone: "America/Los_Angeles",
	}

	// Set cache
	err := cache.SetIPResult(ip, ipResult)
	if err != nil {
		t.Fatalf("Failed to set IP cache: %v", err)
	}

	// Test cache hit
	result, hit = cache.GetIPResult(ip)
	if !hit {
		t.Error("Expected cache hit for cached IP")
	}
	if result == nil {
		t.Fatal("Expected non-nil result for cache hit")
	}
	if result.IP != "8.8.8.8" {
		t.Errorf("Expected IP '8.8.8.8', got '%s'", result.IP)
	}
}

func TestCacheService_QueryNormalization(t *testing.T) {
	db := newMockCacheDB()
	cache := NewService(db, 1000, 1000)

	// These should all produce the same hash
	addresses := []string{
		"1600 Amphitheatre Parkway",
		" 1600 Amphitheatre Parkway ",
		"1600  Amphitheatre  Parkway",
		"1600 AMPHITHEATRE PARKWAY",
		"1600 amphitheatre parkway",
	}

	// Create mock result
	geocodeResult := &geocoding.GeocodeResponse{
		Results: []geocoding.GeocodeResult{
			{FormattedAddress: "Test Address"},
		},
		Status: "OK",
	}

	// Cache the first address
	err := cache.SetGeocodeResult(addresses[0], geocodeResult)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// All variations should hit the cache
	for i, addr := range addresses {
		result, hit := cache.GetGeocodeResult(addr)
		if !hit {
			t.Errorf("Address variation %d should hit cache: '%s'", i, addr)
		}
		if result == nil || result.Status != "OK" {
			t.Errorf("Address variation %d should return valid result", i)
		}
	}
}

func TestCacheService_InvalidCachedData(t *testing.T) {
	db := newMockCacheDB()
	cache := NewService(db, 1000, 1000)

	// Manually insert invalid JSON into the mock cache
	queryHash := cache.hashQuery("test address")
	db.addressCache[queryHash] = &models.AddressCache{
		QueryHash:    queryHash,
		QueryText:    "test address",
		ResponseData: "invalid json{",
		CreatedAt:    time.Now(),
	}

	// Should treat as cache miss due to invalid JSON
	result, hit := cache.GetGeocodeResult("test address")
	if hit {
		t.Error("Expected cache miss for invalid cached data")
	}
	if result != nil {
		t.Error("Expected nil result for invalid cached data")
	}
}
