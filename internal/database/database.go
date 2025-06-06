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

func (db *DB) GetReverseGeocodeCache(queryHash string) (*models.ReverseGeocodeCache, error) {
	var cache models.ReverseGeocodeCache
	query := `
		SELECT id, query_hash, query_text, response_data, created_at
		FROM reverse_geocode_cache
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

func (db *DB) SetReverseGeocodeCache(queryHash, queryText, responseData string, maxCacheSize int) error {
	// Insert new cache entry
	insertQuery := `
		INSERT INTO reverse_geocode_cache (query_hash, query_text, response_data)
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
	countQuery := `SELECT COUNT(*) FROM reverse_geocode_cache`
	err = db.conn.QueryRow(countQuery).Scan(&count)
	if err != nil {
		return err
	}

	if count >= maxCacheSize {
		// Delete oldest 10% of entries
		deleteCount := maxCacheSize / 10
		deleteQuery := `
			DELETE FROM reverse_geocode_cache
			WHERE id IN (
				SELECT id FROM reverse_geocode_cache
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

// GetAPIKeyUsageSummary returns usage analytics for all API keys with pagination
func (db *DB) GetAPIKeyUsageSummary(page, pageSize int) (*models.UsageSummaryResponse, error) {
	offset := (page - 1) * pageSize

	// Get total count first
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM api_keys WHERE is_active = true"
	err := db.conn.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	// Main query to get API keys with usage stats
	query := `
		SELECT 
			ak.id, ak.key_hash, ak.name, ak.owner, ak.app_name, ak.environment, 
			ak.is_active, ak.rate_limit_per_second, ak.created_at, ak.last_used_at, ak.request_count,
			COALESCE(SUM(CASE WHEN ul.endpoint = 'v1/geocode' THEN 1 ELSE 0 END), 0) as geocode_requests,
			COALESCE(SUM(CASE WHEN ul.endpoint = 'v1/geoip' THEN 1 ELSE 0 END), 0) as geoip_requests,
			COALESCE(SUM(CASE WHEN ul.cache_hit = true THEN 1 ELSE 0 END), 0) as cache_hits,
			COALESCE(COUNT(ul.id), 0) as total_requests,
			COALESCE(SUM(CASE 
				WHEN ul.endpoint = 'v1/geocode' AND ul.cache_hit = false THEN 0.005 
				ELSE 0 
			END), 0) as estimated_cost_usd
		FROM api_keys ak
		LEFT JOIN usage_logs ul ON ak.id = ul.api_key_id
		WHERE ak.is_active = true
		GROUP BY ak.id, ak.key_hash, ak.name, ak.owner, ak.app_name, ak.environment, 
				 ak.is_active, ak.rate_limit_per_second, ak.created_at, ak.last_used_at, ak.request_count
		ORDER BY ak.last_used_at DESC NULLS LAST, total_requests DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.conn.Query(query, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.APIKeyUsageSummary
	for rows.Next() {
		var summary models.APIKeyUsageSummary
		var totalRequests, geocodeRequests, geoipRequests, cacheHits int64
		var estimatedCost float64

		err := rows.Scan(
			&summary.APIKey.ID, &summary.APIKey.KeyHash, &summary.APIKey.Name,
			&summary.APIKey.Owner, &summary.APIKey.AppName, &summary.APIKey.Environment,
			&summary.APIKey.IsActive, &summary.APIKey.RateLimitPerSecond,
			&summary.APIKey.CreatedAt, &summary.APIKey.LastUsedAt, &summary.APIKey.RequestCount,
			&geocodeRequests, &geoipRequests, &cacheHits, &totalRequests, &estimatedCost,
		)
		if err != nil {
			return nil, err
		}

		summary.TotalRequests = totalRequests
		summary.GeocodeRequests = geocodeRequests
		summary.GeoipRequests = geoipRequests
		summary.CacheHits = cacheHits
		summary.EstimatedCostUSD = estimatedCost
		summary.LastUsedAt = summary.APIKey.LastUsedAt

		// Calculate cache hit rate
		if totalRequests > 0 {
			summary.CacheHitRate = float64(cacheHits) / float64(totalRequests) * 100
		}

		// Get daily usage for last 30 days
		dailyUsage, err := db.getAPIKeyDailyUsage(summary.APIKey.ID)
		if err != nil {
			return nil, err
		}
		summary.DailyUsage = dailyUsage

		summaries = append(summaries, summary)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	return &models.UsageSummaryResponse{
		APIKeys:    summaries,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// getAPIKeyDailyUsage gets daily usage stats for an API key over the last 30 days
func (db *DB) getAPIKeyDailyUsage(apiKeyID string) ([]models.DailyUsageStats, error) {
	query := `
		SELECT 
			DATE(ul.created_at) as date,
			SUM(CASE WHEN ul.endpoint = 'v1/geocode' THEN 1 ELSE 0 END) as geocode_requests,
			SUM(CASE WHEN ul.endpoint = 'v1/geocode' AND ul.cache_hit = true THEN 1 ELSE 0 END) as geocode_cache_hits,
			SUM(CASE WHEN ul.endpoint = 'v1/geoip' THEN 1 ELSE 0 END) as geoip_requests,
			SUM(CASE WHEN ul.endpoint = 'v1/geoip' AND ul.cache_hit = true THEN 1 ELSE 0 END) as geoip_cache_hits,
			COUNT(*) as total_requests,
			SUM(CASE 
				WHEN ul.endpoint = 'v1/geocode' AND ul.cache_hit = false THEN 0.005 
				ELSE 0 
			END) as estimated_cost_usd
		FROM usage_logs ul
		WHERE ul.api_key_id = $1 
		  AND ul.created_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(ul.created_at)
		ORDER BY date DESC
	`

	rows, err := db.conn.Query(query, apiKeyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyStats []models.DailyUsageStats
	for rows.Next() {
		var stats models.DailyUsageStats
		err := rows.Scan(
			&stats.Date,
			&stats.GeocodeRequests,
			&stats.GeocodeCacheHits,
			&stats.GeoipRequests,
			&stats.GeoipCacheHits,
			&stats.TotalRequests,
			&stats.EstimatedCostUSD,
		)
		if err != nil {
			return nil, err
		}
		dailyStats = append(dailyStats, stats)
	}

	return dailyStats, nil
}

// Helper function to hash API keys
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}
