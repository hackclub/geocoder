-- Address geocoding cache
CREATE TABLE address_cache (
    id SERIAL PRIMARY KEY,
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    query_text TEXT NOT NULL,
    response_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_address_cache_query_hash ON address_cache(query_hash);
CREATE INDEX idx_address_cache_created_at ON address_cache(created_at);

-- IP geolocation cache  
CREATE TABLE ip_cache (
    id SERIAL PRIMARY KEY,
    ip_address INET UNIQUE NOT NULL,
    response_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_ip_cache_ip_address ON ip_cache(ip_address);
CREATE INDEX idx_ip_cache_created_at ON ip_cache(created_at);

-- API key management
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    rate_limit_per_second INTEGER DEFAULT 10,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    request_count INTEGER DEFAULT 0
);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);

-- Usage tracking
CREATE TABLE usage_logs (
    id BIGSERIAL PRIMARY KEY,
    api_key_id UUID REFERENCES api_keys(id),
    endpoint VARCHAR(20) NOT NULL,
    cache_hit BOOLEAN NOT NULL,
    response_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_usage_logs_api_key_id ON usage_logs(api_key_id);
CREATE INDEX idx_usage_logs_created_at ON usage_logs(created_at);
CREATE INDEX idx_usage_logs_endpoint ON usage_logs(endpoint);

-- Cost tracking aggregates
CREATE TABLE cost_tracking (
    date DATE PRIMARY KEY,
    geocode_requests INTEGER DEFAULT 0,
    geocode_cache_hits INTEGER DEFAULT 0,
    geoip_requests INTEGER DEFAULT 0,
    geoip_cache_hits INTEGER DEFAULT 0,
    estimated_cost_usd DECIMAL(10,4) DEFAULT 0
);
