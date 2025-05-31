package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"

	"github.com/hackclub/geocoder/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/migrate/main.go <up|down>")
	}

	command := os.Args[1]
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create migrations table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	switch command {
	case "up":
		runMigrationsUp(db)
	case "down":
		runMigrationsDown(db)
	default:
		log.Fatalf("Unknown command: %s. Use 'up' or 'down'", command)
	}
}

func runMigrationsUp(db *sql.DB) {
	files, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {
		version := strings.TrimSuffix(file, ".sql")

		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if err != nil {
			log.Fatalf("Failed to check migration status: %v", err)
		}

		if count > 0 {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		// Read and execute migration
		content, err := os.ReadFile(filepath.Join("migrations", file))
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			log.Fatalf("Failed to execute migration %s: %v", version, err)
		}

		// Mark as applied
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			log.Fatalf("Failed to mark migration as applied: %v", err)
		}

		log.Printf("Applied migration: %s", version)
	}

	log.Println("All migrations applied successfully")
}

func runMigrationsDown(db *sql.DB) {
	// Get the latest applied migration
	var version string
	err := db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No migrations to rollback")
			return
		}
		log.Fatalf("Failed to get latest migration: %v", err)
	}

	// Remove from tracking table
	_, err = db.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
	if err != nil {
		log.Fatalf("Failed to remove migration record: %v", err)
	}

	log.Printf("Rolled back migration: %s", version)
	log.Println("Note: This tool only tracks migrations, manual schema changes may be required")
}
