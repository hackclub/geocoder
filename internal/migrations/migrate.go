package migrations

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations automatically runs all pending migrations
func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	// Get list of migration files
	files, err := os.ReadDir("migrations")
	if err != nil {
		// If migrations directory doesn't exist, create tables directly
		log.Println("Migrations directory not found, creating tables...")
		return createTablesDirectly(db)
	}

	var migrationFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		log.Println("No migration files found, creating tables...")
		return createTablesDirectly(db)
	}

	// Apply each migration
	for _, file := range migrationFiles {
		version := strings.TrimSuffix(file, ".sql")

		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		// Read and execute migration
		content, err := os.ReadFile(filepath.Join("migrations", file))
		if err != nil {
			return err
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return err
		}

		// Mark as applied
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			return err
		}

		log.Printf("Applied migration: %s", version)
	}

	log.Println("All migrations applied successfully")
	return nil
}

// createTablesDirectly creates all tables when migration files are not available
func createTablesDirectly(db *sql.DB) error {
	schema := `
-- Address geocoding cache
CREATE TABLE IF NOT EXISTS address_cache (
    id SERIAL PRIMARY KEY,
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    query_text TEXT NOT NULL,
    response_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_address_cache_query_hash ON address_cache(query_hash);
CREATE INDEX IF NOT EXISTS idx_address_cache_created_at ON address_cache(created_at);

-- IP geolocation cache  
CREATE TABLE IF NOT EXISTS ip_cache (
    id SERIAL PRIMARY KEY,
    ip_address INET UNIQUE NOT NULL,
    response_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ip_cache_ip_address ON ip_cache(ip_address);
CREATE INDEX IF NOT EXISTS idx_ip_cache_created_at ON ip_cache(created_at);

-- API key management
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    rate_limit_per_second INTEGER DEFAULT 10,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP,
    request_count INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);

-- Usage tracking
CREATE TABLE IF NOT EXISTS usage_logs (
    id BIGSERIAL PRIMARY KEY,
    api_key_id UUID REFERENCES api_keys(id),
    endpoint VARCHAR(20) NOT NULL,
    cache_hit BOOLEAN NOT NULL,
    response_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_id ON usage_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_logs_endpoint ON usage_logs(endpoint);

-- Cost tracking aggregates
CREATE TABLE IF NOT EXISTS cost_tracking (
    date DATE PRIMARY KEY,
    geocode_requests INTEGER DEFAULT 0,
    geocode_cache_hits INTEGER DEFAULT 0,
    geoip_requests INTEGER DEFAULT 0,
    geoip_cache_hits INTEGER DEFAULT 0,
    estimated_cost_usd DECIMAL(10,4) DEFAULT 0
);
`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Mark as applied
	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ('001_initial_schema') ON CONFLICT (version) DO NOTHING")
	if err != nil {
		return err
	}

	log.Println("Database schema created successfully")
	return nil
}
