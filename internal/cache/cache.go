package cache

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/geocoding"
	"github.com/hackclub/geocoder/internal/geoip"
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

func (c *CacheService) hashQuery(query string) string {
	// Normalize query: lowercase, trim, collapse spaces
	normalized := strings.ToLower(strings.TrimSpace(query))
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}
