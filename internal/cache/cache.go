package cache

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/geocoding"
	"github.com/hackclub/geocoder/internal/geoip"
	"github.com/hackclub/geocoder/internal/models"
)

type CacheService struct {
	db                  database.DatabaseInterface
	maxAddressCacheSize int
	maxIPCacheSize      int
}

func NewService(db database.DatabaseInterface, maxAddressCacheSize, maxIPCacheSize int) *CacheService {
	return &CacheService{
		db:                  db,
		maxAddressCacheSize: maxAddressCacheSize,
		maxIPCacheSize:      maxIPCacheSize,
	}
}

func (c *CacheService) GetGeocodeResult(address string) (*geocoding.GeocodeResponse, bool) {
	queryHash := c.hashQuery(address)

	cached, err := c.db.GetAddressCache(queryHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false // Cache miss
		}
		return nil, false // Error, treat as cache miss
	}

	var result geocoding.GeocodeResponse
	if err := json.Unmarshal([]byte(cached.ResponseData), &result); err != nil {
		return nil, false // Invalid cached data, treat as cache miss
	}

	return &result, true // Cache hit
}

func (c *CacheService) SetGeocodeResult(address string, result *geocoding.GeocodeResponse) error {
	queryHash := c.hashQuery(address)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal geocode result: %w", err)
	}

	return c.db.SetAddressCache(queryHash, address, string(resultJSON), c.maxAddressCacheSize)
}

func (c *CacheService) GetIPResult(ip string) (*geoip.IPInfoResponse, bool) {
	cached, err := c.db.GetIPCache(ip)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false // Cache miss
		}
		return nil, false // Error, treat as cache miss
	}

	var result geoip.IPInfoResponse
	if err := json.Unmarshal([]byte(cached.ResponseData), &result); err != nil {
		return nil, false // Invalid cached data, treat as cache miss
	}

	return &result, true // Cache hit
}

func (c *CacheService) SetIPResult(ip string, result *geoip.IPInfoResponse) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal IP result: %w", err)
	}

	return c.db.SetIPCache(ip, string(resultJSON), c.maxIPCacheSize)
}

// GetStandardGeocodeResult retrieves a cached standard geocoding response
func (c *CacheService) GetStandardGeocodeResult(address string) (*models.GeocodeAPIResponse, bool) {
	queryHash := c.hashQuery(address)

	cached, err := c.db.GetAddressCache(queryHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false // Cache miss
		}
		return nil, false // Error, treat as cache miss
	}

	var result models.GeocodeAPIResponse
	if err := json.Unmarshal([]byte(cached.ResponseData), &result); err != nil {
		return nil, false // Invalid cached data, treat as cache miss
	}

	return &result, true // Cache hit
}

// SetStandardGeocodeResult caches a standard geocoding response
func (c *CacheService) SetStandardGeocodeResult(address string, result *models.GeocodeAPIResponse) error {
	queryHash := c.hashQuery(address)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal standard geocode result: %w", err)
	}

	return c.db.SetAddressCache(queryHash, address, string(resultJSON), c.maxAddressCacheSize)
}

// GetStandardIPResult retrieves a cached standard IP geolocation response
func (c *CacheService) GetStandardIPResult(ip string) (*models.GeoIPAPIResponse, bool) {
	cached, err := c.db.GetIPCache(ip)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false // Cache miss
		}
		return nil, false // Error, treat as cache miss
	}

	var result models.GeoIPAPIResponse
	if err := json.Unmarshal([]byte(cached.ResponseData), &result); err != nil {
		return nil, false // Invalid cached data, treat as cache miss
	}

	return &result, true // Cache hit
}

// SetStandardIPResult caches a standard IP geolocation response
func (c *CacheService) SetStandardIPResult(ip string, result *models.GeoIPAPIResponse) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal standard IP result: %w", err)
	}

	return c.db.SetIPCache(ip, string(resultJSON), c.maxIPCacheSize)
}

// GetStandardReverseGeocodeResult retrieves a cached standard reverse geocoding response
func (c *CacheService) GetStandardReverseGeocodeResult(lat, lng float64) (*models.ReverseGeocodeAPIResponse, bool) {
	queryHash := c.hashCoordinates(lat, lng)

	cached, err := c.db.GetReverseGeocodeCache(queryHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false // Cache miss
		}
		return nil, false // Error, treat as cache miss
	}

	var result models.ReverseGeocodeAPIResponse
	if err := json.Unmarshal([]byte(cached.ResponseData), &result); err != nil {
		return nil, false // Invalid cached data, treat as cache miss
	}

	return &result, true // Cache hit
}

// SetStandardReverseGeocodeResult caches a standard reverse geocoding response
func (c *CacheService) SetStandardReverseGeocodeResult(lat, lng float64, result *models.ReverseGeocodeAPIResponse) error {
	queryHash := c.hashCoordinates(lat, lng)
	queryText := fmt.Sprintf("%f,%f", lat, lng)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal standard reverse geocode result: %w", err)
	}

	return c.db.SetReverseGeocodeCache(queryHash, queryText, string(resultJSON), c.maxAddressCacheSize)
}

// normalizeAddress performs conservative address normalization for geocoding cache
// Only applies transformations that are guaranteed safe for geocoding accuracy
func (c *CacheService) normalizeAddress(address string) string {
	// Start with basic trimming and lowercase (SAFE: Google is case-insensitive)
	normalized := strings.ToLower(strings.TrimSpace(address))
	
	// Replace non-standard delimiters with standard comma-space
	// SAFE: These are clearly delimiter characters, not meaningful address content
	// Handle: backslashes, pipes, tabs, newlines (but NOT semicolons - they might be meaningful)
	delimitersRegex := regexp.MustCompile(`[\\|\t\n]+`)
	normalized = delimitersRegex.ReplaceAllString(normalized, ", ")
	
	// Normalize multiple commas to single comma with consistent spacing
	// SAFE: Improves structure without losing information
	multiCommaRegex := regexp.MustCompile(`,\s*,+`)
	normalized = multiCommaRegex.ReplaceAllString(normalized, ", ")
	
	// Ensure consistent comma spacing (but preserve commas)
	// SAFE: Only affects whitespace around commas
	commaSpaceRegex := regexp.MustCompile(`,\s*`)
	normalized = commaSpaceRegex.ReplaceAllString(normalized, ", ")
	
	// Remove commas at start/end (structural cleanup)
	// SAFE: Leading/trailing commas don't add meaningful information
	normalized = strings.Trim(normalized, ", ")
	
	// Collapse multiple spaces into single spaces
	// SAFE: Google handles multiple spaces fine, this is just normalization
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	// Final whitespace cleanup
	// SAFE: Ensures no leading/trailing spaces remain
	normalized = strings.TrimSpace(normalized)
	
	return normalized
}

func (c *CacheService) hashQuery(query string) string {
	// Use address-specific normalization for geocoding queries
	normalized := c.normalizeAddress(query)
	
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

func (c *CacheService) hashCoordinates(lat, lng float64) string {
	// Round coordinates to 5 decimal places for consistent caching
	// This provides ~1.1m precision which is reasonable for caching
	normalized := fmt.Sprintf("%.5f,%.5f", lat, lng)
	
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}
