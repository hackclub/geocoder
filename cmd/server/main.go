package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/hackclub/geocoder/internal/api"
	"github.com/hackclub/geocoder/internal/cache"
	"github.com/hackclub/geocoder/internal/config"
	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/geocoding"
	"github.com/hackclub/geocoder/internal/geoip"
	"github.com/hackclub/geocoder/internal/middleware"
	"github.com/hackclub/geocoder/internal/migrations"
)

func main() {
	// Set timezone to Eastern Time
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatalf("Failed to load Eastern timezone: %v", err)
	}
	time.Local = loc
	log.Println("Timezone set to Eastern Time (America/New_York)")

	log.Println("Starting Hack Club Geocoder...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations automatically on startup
	log.Println("Running database migrations...")
	if err := migrations.RunMigrations(db.GetDB()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize external API clients
	geocodeClient := geocoding.NewClient(cfg.GoogleGeocodingAPIKey)
	geoipClient := geoip.NewClient(cfg.IPInfoAPIKey)

	// Initialize cache service
	cacheService := cache.NewService(db, cfg.MaxAddressCacheSize, cfg.MaxIPCacheSize)

	// Initialize handlers
	handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter()
	rateLimiter.Cleanup() // Start cleanup goroutine

	// Set up routes
	router := mux.NewRouter()

	// Apply CORS middleware to all routes
	router.Use(middleware.CORS())

	// API v1 routes (with authentication and rate limiting)
	v1 := router.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.APIKeyAuth(db))
	v1.Use(rateLimiter.RateLimit())
	v1.HandleFunc("/geocode", handlers.HandleGeocode).Methods("GET")
	v1.HandleFunc("/geocode_structured", handlers.HandleGeocodeStructured).Methods("GET")
	v1.HandleFunc("/geoip", handlers.HandleGeoIP).Methods("GET")

	// Admin routes (with basic auth)
	admin := router.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.BasicAuth(cfg.AdminUsername, cfg.AdminPassword))
	admin.HandleFunc("/dashboard", handlers.HandleAdminDashboard).Methods("GET")
	admin.HandleFunc("/keys", handlers.HandleAdminKeys).Methods("GET", "POST")
	admin.HandleFunc("/keys/{key_id}/rate-limit", handlers.HandleUpdateAPIKeyRateLimit).Methods("PUT")
	admin.HandleFunc("/keys/{key_id}", handlers.HandleDeactivateAPIKey).Methods("DELETE")
	admin.HandleFunc("/stats", handlers.HandleAdminStats).Methods("GET")
	admin.HandleFunc("/activity", handlers.HandleAdminActivity).Methods("GET")
	admin.HandleFunc("/usage-summary", handlers.HandleUsageSummary).Methods("GET")
	admin.HandleFunc("/ws", handlers.HandleWebSocket)

	// Redirect /admin to /admin/dashboard
	router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	}).Methods("GET")

	// Health check (no auth required)
	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET")

	// Documentation (no auth required)
	router.HandleFunc("/", handlers.HandleDocs).Methods("GET")

	// Test interface for map functionality
	router.HandleFunc("/test", handlers.HandleTestMap).Methods("GET")

	// Catch-all for API version routes only
	router.PathPrefix("/v").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers := api.NewHandlers(db, geocodeClient, geoipClient, cacheService)
		handlers.HandleUnsupportedVersion(w, r)
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
