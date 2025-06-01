package models

import (
	"strings"
	"time"
)

// APIKey represents an API key in the database
type APIKey struct {
	ID                 string     `json:"id" db:"id"`
	KeyHash            string     `json:"-" db:"key_hash"`
	Name               string     `json:"name" db:"name"`
	Owner              string     `json:"owner" db:"owner"`
	AppName            string     `json:"app_name" db:"app_name"`
	Environment        string     `json:"environment" db:"environment"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	RateLimitPerSecond int        `json:"rate_limit_per_second" db:"rate_limit_per_second"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	RequestCount       int        `json:"request_count" db:"request_count"`
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
	Date             time.Time `json:"date" db:"date"`
	GeocodeRequests  int       `json:"geocode_requests" db:"geocode_requests"`
	GeocodeCacheHits int       `json:"geocode_cache_hits" db:"geocode_cache_hits"`
	GeoipRequests    int       `json:"geoip_requests" db:"geoip_requests"`
	GeoipCacheHits   int       `json:"geoip_cache_hits" db:"geoip_cache_hits"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd" db:"estimated_cost_usd"`
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
	Status            string            `json:"status"`
	Services          map[string]string `json:"services"`
	Timestamp         time.Time         `json:"timestamp"`
	DatabaseConnected bool              `json:"database_connected"`
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
	TotalRequests       int64   `json:"total_requests"`
	CacheHitRate        float64 `json:"cache_hit_rate"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	ActiveAPIKeys       int     `json:"active_api_keys"`
	TodaysRequests      int64   `json:"todays_requests"`
	TodaysCacheHits     int64   `json:"todays_cache_hits"`
}

// APIKeyUsageSummary represents usage analytics for an API key
type APIKeyUsageSummary struct {
	APIKey           APIKey            `json:"api_key"`
	TotalRequests    int64             `json:"total_requests"`
	GeocodeRequests  int64             `json:"geocode_requests"`
	GeoipRequests    int64             `json:"geoip_requests"`
	CacheHits        int64             `json:"cache_hits"`
	CacheHitRate     float64           `json:"cache_hit_rate"`
	EstimatedCostUSD float64           `json:"estimated_cost_usd"`
	DailyUsage       []DailyUsageStats `json:"daily_usage"`
	LastUsedAt       *time.Time        `json:"last_used_at"`
}

// DailyUsageStats represents daily usage for an API key
type DailyUsageStats struct {
	Date             time.Time `json:"date" db:"date"`
	GeocodeRequests  int       `json:"geocode_requests" db:"geocode_requests"`
	GeocodeCacheHits int       `json:"geocode_cache_hits" db:"geocode_cache_hits"`
	GeoipRequests    int       `json:"geoip_requests" db:"geoip_requests"`
	GeoipCacheHits   int       `json:"geoip_cache_hits" db:"geoip_cache_hits"`
	TotalRequests    int       `json:"total_requests" db:"total_requests"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd" db:"estimated_cost_usd"`
}

// UsageSummaryRequest represents pagination request for usage summary
type UsageSummaryRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// UsageSummaryResponse represents paginated usage summary response
type UsageSummaryResponse struct {
	APIKeys    []APIKeyUsageSummary `json:"api_keys"`
	TotalCount int                  `json:"total_count"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

// GeocodeAPIResponse represents our standardized geocoding API response
type GeocodeAPIResponse struct {
	Lat                  float64     `json:"lat"`
	Lng                  float64     `json:"lng"`
	FormattedAddress     string      `json:"formatted_address"`
	CountryName          string      `json:"country_name"`
	CountryCode          string      `json:"country_code"`
	Backend              string      `json:"backend"`
	RawBackendResponse   interface{} `json:"raw_backend_response"`
}

// GeoIPAPIResponse represents our standardized IP geolocation API response
type GeoIPAPIResponse struct {
	Lat                float64     `json:"lat"`
	Lng                float64     `json:"lng"`
	IP                 string      `json:"ip"`
	City               string      `json:"city"`
	Region             string      `json:"region"`
	CountryName        string      `json:"country_name"`
	CountryCode        string      `json:"country_code"`
	PostalCode         string      `json:"postal_code"`
	Timezone           string      `json:"timezone"`
	Org                string      `json:"org"`
	Backend            string      `json:"backend"`
	RawBackendResponse interface{} `json:"raw_backend_response"`
}

// StructuredAddress represents a structured address for geocoding
type StructuredAddress struct {
	AddressLine1 string `json:"address_line_1"`
	AddressLine2 string `json:"address_line_2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postal_code"`
	Country      string `json:"country"`
}

// ToFormattedString converts the structured address to a single formatted string for geocoding
func (sa *StructuredAddress) ToFormattedString() string {
	var parts []string
	
	if sa.AddressLine1 != "" {
		parts = append(parts, sa.AddressLine1)
	}
	if sa.AddressLine2 != "" {
		parts = append(parts, sa.AddressLine2)
	}
	if sa.City != "" {
		parts = append(parts, sa.City)
	}
	if sa.State != "" {
		parts = append(parts, sa.State)
	}
	if sa.PostalCode != "" {
		parts = append(parts, sa.PostalCode)
	}
	if sa.Country != "" {
		parts = append(parts, sa.Country)
	}
	
	return strings.Join(parts, ", ")
}
