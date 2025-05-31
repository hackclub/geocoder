package database

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/hackclub/geocoder/internal/models"
)

type DB struct {
	conn *sql.DB
}

func New(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Ping() error {
	return db.conn.Ping()
}

func (db *DB) GetDB() *sql.DB {
	return db.conn
}

// API Key operations
func (db *DB) CreateAPIKey(keyHash, name, owner, appName, environment string, rateLimitPerSecond int) (*models.APIKey, error) {
	var apiKey models.APIKey
	query := `
		INSERT INTO api_keys (key_hash, name, owner, app_name, environment, rate_limit_per_second)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, key_hash, name, owner, app_name, environment, is_active, rate_limit_per_second, created_at, last_used_at, request_count
	`
	err := db.conn.QueryRow(query, keyHash, name, owner, appName, environment, rateLimitPerSecond).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.Name, &apiKey.Owner, &apiKey.AppName, &apiKey.Environment, &apiKey.IsActive,
		&apiKey.RateLimitPerSecond, &apiKey.CreatedAt, &apiKey.LastUsedAt, &apiKey.RequestCount,
	)
	return &apiKey, err
}

func (db *DB) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	query := `
		SELECT id, key_hash, name, owner, app_name, environment, is_active, rate_limit_per_second, created_at, last_used_at, request_count
		FROM api_keys
		WHERE key_hash = $1 AND is_active = true
	`
	err := db.conn.QueryRow(query, keyHash).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.Name, &apiKey.Owner, &apiKey.AppName, &apiKey.Environment, &apiKey.IsActive,
		&apiKey.RateLimitPerSecond, &apiKey.CreatedAt, &apiKey.LastUsedAt, &apiKey.RequestCount,
	)
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (db *DB) UpdateAPIKeyUsage(keyID string) error {
	query := `
		UPDATE api_keys 
		SET last_used_at = NOW(), request_count = request_count + 1
		WHERE id = $1
	`
	_, err := db.conn.Exec(query, keyID)
	return err
}

func (db *DB) GetAllAPIKeys() ([]models.APIKey, error) {
	query := `
		SELECT id, key_hash, name, owner, app_name, environment, is_active, rate_limit_per_second, created_at, last_used_at, request_count
		FROM api_keys
		ORDER BY created_at DESC
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var key models.APIKey
		err := rows.Scan(&key.ID, &key.KeyHash, &key.Name, &key.Owner, &key.AppName, &key.Environment, &key.IsActive,
			&key.RateLimitPerSecond, &key.CreatedAt, &key.LastUsedAt, &key.RequestCount)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (db *DB) UpdateAPIKeyRateLimit(keyID string, rateLimitPerSecond int) error {
	query := `UPDATE api_keys SET rate_limit_per_second = $1 WHERE id = $2`
	_, err := db.conn.Exec(query, rateLimitPerSecond, keyID)
	return err
}

func (db *DB) DeactivateAPIKey(keyID string) error {
	query := `UPDATE api_keys SET is_active = false WHERE id = $1`
	_, err := db.conn.Exec(query, keyID)
	return err
}

// Cache operations
func (db *DB) GetAddressCache(queryHash string) (*models.AddressCache, error) {
	var cache models.AddressCache
	query := `
		SELECT id, query_hash, query_text, response_data, created_at
		FROM address_cache
		WHERE query_hash = $1
	`
	err := db.conn.QueryRow(query, queryHash).Scan(
		&cache.ID, &cache.QueryHash, &cache.QueryText, &cache.ResponseData, &cache.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

func (db *DB) SetAddressCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	// Insert new cache entry
	insertQuery := `
		INSERT INTO address_cache (query_hash, query_text, response_data)
		VALUES ($1, $2, $3)
		ON CONFLICT (query_hash) DO UPDATE SET
			response_data = EXCLUDED.response_data,
			created_at = NOW()
	`
	_, err := db.conn.Exec(insertQuery, queryHash, queryText, responseData)
	if err != nil {
		return err
	}

	// Check if we need to evict old entries
	var count int
	countQuery := `SELECT COUNT(*) FROM address_cache`
	err = db.conn.QueryRow(countQuery).Scan(&count)
	if err != nil {
		return err
	}

	if count >= maxCacheSize {
		// Delete oldest 10% of entries
		deleteCount := maxCacheSize / 10
		deleteQuery := `
			DELETE FROM address_cache
			WHERE id IN (
				SELECT id FROM address_cache
				ORDER BY created_at ASC
				LIMIT $1
			)
		`
		_, err = db.conn.Exec(deleteQuery, deleteCount)
	}

	return err
}

func (db *DB) GetIPCache(ipAddress string) (*models.IPCache, error) {
	var cache models.IPCache
	query := `
		SELECT id, ip_address, response_data, created_at
		FROM ip_cache
		WHERE ip_address = $1
	`
	err := db.conn.QueryRow(query, ipAddress).Scan(
		&cache.ID, &cache.IPAddress, &cache.ResponseData, &cache.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

func (db *DB) SetIPCache(ipAddress, responseData string, maxCacheSize int) error {
	// Insert new cache entry
	insertQuery := `
		INSERT INTO ip_cache (ip_address, response_data)
		VALUES ($1, $2)
		ON CONFLICT (ip_address) DO UPDATE SET
			response_data = EXCLUDED.response_data,
			created_at = NOW()
	`
	_, err := db.conn.Exec(insertQuery, ipAddress, responseData)
	if err != nil {
		return err
	}

	// Check if we need to evict old entries
	var count int
	countQuery := `SELECT COUNT(*) FROM ip_cache`
	err = db.conn.QueryRow(countQuery).Scan(&count)
	if err != nil {
		return err
	}

	if count >= maxCacheSize {
		// Delete oldest 10% of entries
		deleteCount := maxCacheSize / 10
		deleteQuery := `
			DELETE FROM ip_cache
			WHERE id IN (
				SELECT id FROM ip_cache
				ORDER BY created_at ASC
				LIMIT $1
			)
		`
		_, err = db.conn.Exec(deleteQuery, deleteCount)
	}

	return err
}

// Usage tracking
func (db *DB) LogUsage(apiKeyID, endpoint string, cacheHit bool, responseTimeMs int) error {
	query := `
		INSERT INTO usage_logs (api_key_id, endpoint, cache_hit, response_time_ms)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.conn.Exec(query, apiKeyID, endpoint, cacheHit, responseTimeMs)
	return err
}

// Stats and cost tracking
func (db *DB) GetStats() (*models.Stats, error) {
	stats := &models.Stats{}

	// Total requests
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM usage_logs`).Scan(&stats.TotalRequests)
	if err != nil {
		return nil, err
	}

	// Cache hit rate
	var cacheHits int64
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM usage_logs WHERE cache_hit = true`).Scan(&cacheHits)
	if err != nil {
		return nil, err
	}

	if stats.TotalRequests > 0 {
		stats.CacheHitRate = float64(cacheHits) / float64(stats.TotalRequests) * 100
	}

	// Average response time
	err = db.conn.QueryRow(`SELECT COALESCE(AVG(response_time_ms), 0) FROM usage_logs`).Scan(&stats.AverageResponseTime)
	if err != nil {
		return nil, err
	}

	// Active API keys
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE is_active = true`).Scan(&stats.ActiveAPIKeys)
	if err != nil {
		return nil, err
	}

	// Today's requests
	today := time.Now().Format("2006-01-02")
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM usage_logs WHERE DATE(created_at) = $1`, today).Scan(&stats.TodaysRequests)
	if err != nil {
		return nil, err
	}

	// Today's cache hits
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM usage_logs WHERE DATE(created_at) = $1 AND cache_hit = true`, today).Scan(&stats.TodaysCacheHits)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (db *DB) UpdateCostTracking(date time.Time, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits int, estimatedCost float64) error {
	query := `
		INSERT INTO cost_tracking (date, geocode_requests, geocode_cache_hits, geoip_requests, geoip_cache_hits, estimated_cost_usd)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (date) DO UPDATE SET
			geocode_requests = cost_tracking.geocode_requests + EXCLUDED.geocode_requests,
			geocode_cache_hits = cost_tracking.geocode_cache_hits + EXCLUDED.geocode_cache_hits,
			geoip_requests = cost_tracking.geoip_requests + EXCLUDED.geoip_requests,
			geoip_cache_hits = cost_tracking.geoip_cache_hits + EXCLUDED.geoip_cache_hits,
			estimated_cost_usd = cost_tracking.estimated_cost_usd + EXCLUDED.estimated_cost_usd
	`
	_, err := db.conn.Exec(query, date, geocodeRequests, geocodeCacheHits, geoipRequests, geoipCacheHits, estimatedCost)
	return err
}

// Activity logging
func (db *DB) LogActivity(apiKeyName, endpoint, queryText string, resultCount, responseTimeMs int, apiSource string, cacheHit bool, ipAddress, userAgent string) error {
	query := `
		INSERT INTO activity_log (api_key_name, endpoint, query_text, result_count, response_time_ms, api_source, cache_hit, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := db.conn.Exec(query, apiKeyName, endpoint, queryText, resultCount, responseTimeMs, apiSource, cacheHit, ipAddress, userAgent)
	return err
}

func (db *DB) GetRecentActivity() ([]models.ActivityLog, error) {
	query := `
		SELECT id, timestamp, api_key_name, endpoint, query_text, result_count, response_time_ms, api_source, cache_hit, 
		       COALESCE(ip_address::text, '') as ip_address, COALESCE(user_agent, '') as user_agent
		FROM activity_log
		ORDER BY timestamp DESC
		LIMIT 100
	`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var activities []models.ActivityLog
	for rows.Next() {
		var activity models.ActivityLog
		err := rows.Scan(
			&activity.ID,
			&activity.Timestamp,
			&activity.APIKeyName,
			&activity.Endpoint,
			&activity.QueryText,
			&activity.ResultCount,
			&activity.ResponseTimeMs,
			&activity.APISource,
			&activity.CacheHit,
			&activity.IPAddress,
			&activity.UserAgent,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	
	return activities, rows.Err()
}

// Helper function to hash API keys
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}
