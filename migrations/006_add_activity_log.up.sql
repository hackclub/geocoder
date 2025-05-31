CREATE TABLE activity_log (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    api_key_name VARCHAR(255) NOT NULL,
    endpoint VARCHAR(50) NOT NULL,
    query_text TEXT NOT NULL,
    result_count INTEGER NOT NULL,
    response_time_ms INTEGER NOT NULL,
    api_source VARCHAR(50) NOT NULL,
    cache_hit BOOLEAN NOT NULL DEFAULT FALSE,
    ip_address INET,
    user_agent TEXT
);

-- Index for efficient querying by timestamp
CREATE INDEX idx_activity_log_timestamp ON activity_log(timestamp DESC);

-- Function to maintain only the latest 100 records
CREATE OR REPLACE FUNCTION maintain_activity_log_limit()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM activity_log 
    WHERE id NOT IN (
        SELECT id FROM activity_log 
        ORDER BY timestamp DESC 
        LIMIT 100
    );
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically maintain the 100 record limit
CREATE TRIGGER trigger_maintain_activity_log_limit
    AFTER INSERT ON activity_log
    FOR EACH STATEMENT
    EXECUTE FUNCTION maintain_activity_log_limit();
