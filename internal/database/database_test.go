package database

import (
	"testing"
	"time"

	"github.com/hackclub/geocoder/internal/models"
)

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"Standard key", "test_abc123"},
		{"Empty key", ""},
		{"Unicode key", "test_🔑"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashAPIKey(tt.key)
			hash2 := HashAPIKey(tt.key)
			
			// Hash should be consistent
			if hash1 != hash2 {
				t.Errorf("Hash should be consistent for same input")
			}
			
			// Hash should be 64 characters (SHA-256 hex)
			if len(hash1) != 64 {
				t.Errorf("Expected hash length 64, got %d", len(hash1))
			}
		})
	}
}

func TestHashAPIKey_Different(t *testing.T) {
	key1 := "test_abc123"
	key2 := "test_def456"
	
	hash1 := HashAPIKey(key1)
	hash2 := HashAPIKey(key2)
	
	if hash1 == hash2 {
		t.Error("Different keys should produce different hashes")
	}
}

// Mock database implementation for testing
type mockDatabase struct {
	apiKeys     map[string]*models.APIKey
	addressCache map[string]*models.AddressCache
	ipCache     map[string]*models.IPCache
	stats       *models.Stats
}

func newMockDatabase() *mockDatabase {
	return &mockDatabase{
		apiKeys:      make(map[string]*models.APIKey),
		addressCache: make(map[string]*models.AddressCache),
		ipCache:      make(map[string]*models.IPCache),
		stats: &models.Stats{
			TotalRequests:       100,
			CacheHitRate:       75.5,
			AverageResponseTime: 150.0,
			ActiveAPIKeys:      3,
			TodaysRequests:     25,
			TodaysCacheHits:    20,
		},
	}
}

func (m *mockDatabase) Close() error { return nil }
func (m *mockDatabase) Ping() error  { return nil }

func (m *mockDatabase) CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error) {
	apiKey := &models.APIKey{
		ID:                 "test-id-123",
		KeyHash:            keyHash,
		Name:               name,
		Owner:              owner,
		AppName:            appName,
		Environment:        environment,
		IsActive:           true,
		RateLimitPerSecond: rateLimitPerSecond,
		CreatedAt:          time.Now(),
		RequestCount:       0,
	}
	m.apiKeys[keyHash] = apiKey
	return apiKey, nil
}

func (m *mockDatabase) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	if key, exists := m.apiKeys[keyHash]; exists {
		return key, nil
	}
	return nil, nil
}

func (m *mockDatabase) UpdateAPIKeyUsage(keyID string) error {
	// Find key by ID and increment usage
	for _, key := range m.apiKeys {
		if key.ID == keyID {
			key.RequestCount++
			now := time.Now()
			key.LastUsedAt = &now
			break
		}
	}
	return nil
}

func (m *mockDatabase) GetAllAPIKeys() ([]models.APIKey, error) {
	var keys []models.APIKey
	for _, key := range m.apiKeys {
		keys = append(keys, *key)
	}
	return keys, nil
}

func (m *mockDatabase) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error {
	for _, key := range m.apiKeys {
		if key.ID == keyID {
			key.RateLimitPerSecond = rateLimitPerSecond
			break
		}
	}
	return nil
}

func (m *mockDatabase) DeactivateAPIKey(keyID string) error {
	for _, key := range m.apiKeys {
		if key.ID == keyID {
			key.IsActive = false
			break
		}
	}
	return nil
}

func (m *mockDatabase) GetAddressCache(queryHash string) (*models.AddressCache, error) {
	if cache, exists := m.addressCache[queryHash]; exists {
		return cache, nil
	}
	return nil, nil
}

func (m *mockDatabase) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	m.addressCache[queryHash] = &models.AddressCache{
		ID:           len(m.addressCache) + 1,
		QueryHash:    queryHash,
		QueryText:    queryText,
		ResponseData: responseData,
		CreatedAt:    time.Now(),
	}
	return nil
}

func (m *mockDatabase) GetIPCache(ipAddress string) (*models.IPCache, error) {
	if cache, exists := m.ipCache[ipAddress]; exists {
		return cache, nil
	}
	return nil, nil
}

func (m *mockDatabase) SetIPCache(ipAddress, responseData string, maxCacheSize int) error {
	m.ipCache[ipAddress] = &models.IPCache{
		ID:           len(m.ipCache) + 1,
		IPAddress:    ipAddress,
		ResponseData: responseData,
		CreatedAt:    time.Now(),
	}
	return nil
}

func (m *mockDatabase) LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error {
	return nil
}

func (m *mockDatabase) GetStats() (*models.Stats, error) {
	return m.stats, nil
}

func (m *mockDatabase) UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error {
	return nil
}

func TestMockDatabase_APIKeyOperations(t *testing.T) {
	db := newMockDatabase()
	
	// Test creating API key
	keyHash := HashAPIKey("test_abc123")
	apiKey, err := db.CreateAPIKey(keyHash, "test-key", "test-owner", "test-app", "dev", 10)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}
	
	if apiKey.Name != "test-key" {
		t.Errorf("Expected name 'test-key', got '%s'", apiKey.Name)
	}
	
	// Test getting API key
	retrieved, err := db.GetAPIKeyByHash(keyHash)
	if err != nil {
		t.Fatalf("Failed to get API key: %v", err)
	}
	
	if retrieved.ID != apiKey.ID {
		t.Errorf("Expected ID '%s', got '%s'", apiKey.ID, retrieved.ID)
	}
	
	// Test updating usage
	err = db.UpdateAPIKeyUsage(apiKey.ID)
	if err != nil {
		t.Fatalf("Failed to update API key usage: %v", err)
	}
	
	// Test deactivating key
	err = db.DeactivateAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("Failed to deactivate API key: %v", err)
	}
	
	if apiKey.IsActive {
		t.Error("API key should be deactivated")
	}
}
