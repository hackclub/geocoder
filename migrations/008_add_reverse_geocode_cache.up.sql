-- Reverse geocoding cache
CREATE TABLE reverse_geocode_cache (
    id SERIAL PRIMARY KEY,
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    query_text TEXT NOT NULL,
    response_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_reverse_geocode_cache_query_hash ON reverse_geocode_cache(query_hash);
CREATE INDEX idx_reverse_geocode_cache_created_at ON reverse_geocode_cache(created_at);
