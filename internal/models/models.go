package models

import (
	"time"
)

// APIKey represents an API key in the database
type APIKey struct {
	ID                 string    `json:"id" db:"id"`
	KeyHash            string    `json:"-" db:"key_hash"`
	Name               string    `json:"name" db:"name"`
	Owner              string    `json:"owner" db:"owner"`
	AppName            string    `json:"app_name" db:"app_name"`
	Environment        string    `json:"environment" db:"environment"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	RateLimitPerSecond int       `json:"rate_limit_per_second" db:"rate_limit_per_second"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	RequestCount       int       `json:"request_count" db:"request_count"`
}

// AddressCache represents a cached geocoding result
type AddressCache struct {
	ID           int       `json:"id" db:"id"`
	QueryHash    string    `json:"query_hash" db:"query_hash"`
	QueryText    string    `json:"query_text" db:"query_text"`
	ResponseData string    `json:"response_data" db:"response_data"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// IPCache represents a cached IP geolocation result
type IPCache struct {
	ID           int       `json:"id" db:"id"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	ResponseData string    `json:"response_data" db:"response_data"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// UsageLog represents a usage log entry
type UsageLog struct {
	ID             int64     `json:"id" db:"id"`
	APIKeyID       string    `json:"api_key_id" db:"api_key_id"`
	Endpoint       string    `json:"endpoint" db:"endpoint"`
	CacheHit       bool      `json:"cache_hit" db:"cache_hit"`
	ResponseTimeMs int       `json:"response_time_ms" db:"response_time_ms"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// CostTracking represents daily cost tracking
type CostTracking struct {
	Date               time.Time `json:"date" db:"date"`
	GeocodeRequests    int       `json:"geocode_requests" db:"geocode_requests"`
	GeocodeCacheHits   int       `json:"geocode_cache_hits" db:"geocode_cache_hits"`
	GeoipRequests      int       `json:"geoip_requests" db:"geoip_requests"`
	GeoipCacheHits     int       `json:"geoip_cache_hits" db:"geoip_cache_hits"`
	EstimatedCostUSD   float64   `json:"estimated_cost_usd" db:"estimated_cost_usd"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateAPIKeyRequest represents the request to create a new API key
type CreateAPIKeyRequest struct {
	Name               string `json:"name"`
	Owner              string `json:"owner"`
	AppName            string `json:"app_name"`
	Environment        string `json:"environment"`
	Prefix             string `json:"prefix"`
	RateLimitPerSecond int    `json:"rate_limit_per_second"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Owner              string    `json:"owner"`
	AppName            string    `json:"app_name"`
	Environment        string    `json:"environment"`
	Key                string    `json:"key"`
	RateLimitPerSecond int       `json:"rate_limit_per_second"`
	CreatedAt          time.Time `json:"created_at"`
}

// UpdateRateLimitRequest represents a rate limit update request
type UpdateRateLimitRequest struct {
	RateLimitPerSecond int `json:"rate_limit_per_second"`
}

// WebSocketMessage represents a real-time update message
type WebSocketMessage struct {
	Type      string       `json:"type"`
	Lat       float64      `json:"lat,omitempty"`
	Lng       float64      `json:"lng,omitempty"`
	CacheHit  bool         `json:"cache_hit,omitempty"`
	Endpoint  string       `json:"endpoint,omitempty"`
	Address   string       `json:"address,omitempty"`
	IP        string       `json:"ip,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
	Stats     *Stats       `json:"stats,omitempty"`
	Activity  *ActivityLog `json:"activity,omitempty"`
}

// HealthStatus represents the health check response
type HealthStatus struct {
	Status           string            `json:"status"`
	Services         map[string]string `json:"services"`
	Timestamp        time.Time         `json:"timestamp"`
	DatabaseConnected bool             `json:"database_connected"`
}

// ActivityLog represents a recent geocoding activity entry
type ActivityLog struct {
	ID             int64     `json:"id" db:"id"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
	APIKeyName     string    `json:"api_key_name" db:"api_key_name"`
	Endpoint       string    `json:"endpoint" db:"endpoint"`
	QueryText      string    `json:"query_text" db:"query_text"`
	ResultCount    int       `json:"result_count" db:"result_count"`
	ResponseTimeMs int       `json:"response_time_ms" db:"response_time_ms"`
	APISource      string    `json:"api_source" db:"api_source"`
	CacheHit       bool      `json:"cache_hit" db:"cache_hit"`
	IPAddress      string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      string    `json:"user_agent,omitempty" db:"user_agent"`
}

// Stats represents usage statistics
type Stats struct {
	TotalRequests      int64   `json:"total_requests"`
	CacheHitRate       float64 `json:"cache_hit_rate"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	ActiveAPIKeys      int     `json:"active_api_keys"`
	TodaysRequests     int64   `json:"todays_requests"`
	TodaysCacheHits    int64   `json:"todays_cache_hits"`
}
