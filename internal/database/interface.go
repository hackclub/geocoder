package database

import (
	"time"

	"github.com/hackclub/geocoder/internal/models"
)

// DatabaseInterface defines the database operations
type DatabaseInterface interface {
	Close() error
	Ping() error

	// API Key operations
	CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error)
	GetAPIKeyByHash(keyHash string) (*models.APIKey, error)
	UpdateAPIKeyUsage(keyID string) error
	GetAllAPIKeys() ([]models.APIKey, error)
	UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error
	DeactivateAPIKey(keyID string) error

	// Cache operations
	GetAddressCache(queryHash string) (*models.AddressCache, error)
	SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error
	GetIPCache(ipAddress string) (*models.IPCache, error)
	SetIPCache(ipAddress, responseData string, maxCacheSize int) error

	// Usage tracking
	LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error
	GetStats() (*models.Stats, error)
	UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error

	// Activity logging
	LogActivity(apiKeyName, endpoint, queryText string, resultCount, responseTimeMs int, apiSource string, cacheHit bool, ipAddress, userAgent string) error
	GetRecentActivity() ([]models.ActivityLog, error)

	// Usage summary
	GetAPIKeyUsageSummary(page, pageSize int) (*models.UsageSummaryResponse, error)
}

// Ensure DB implements DatabaseInterface
var _ DatabaseInterface = (*DB)(nil)
