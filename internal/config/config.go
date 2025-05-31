package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	GoogleGeocodingAPIKey     string
	IPInfoAPIKey              string
	DatabaseURL               string
	Port                      string
	AdminUsername             string
	AdminPassword             string
	MaxAddressCacheSize       int
	MaxIPCacheSize            int
	DefaultRateLimitPerSecond int
	LogLevel                  string
}

func Load() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		GoogleGeocodingAPIKey:     getEnv("GOOGLE_GEOCODING_API_KEY", ""),
		IPInfoAPIKey:              getEnv("IPINFO_API_KEY", ""),
		DatabaseURL:               getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/geocoder?sslmode=disable"),
		Port:                      getEnv("PORT", "8080"),
		AdminUsername:             getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:             getEnv("ADMIN_PASSWORD", "admin"),
		MaxAddressCacheSize:       getEnvInt("MAX_ADDRESS_CACHE_SIZE", 10000),
		MaxIPCacheSize:            getEnvInt("MAX_IP_CACHE_SIZE", 5000),
		DefaultRateLimitPerSecond: getEnvInt("DEFAULT_RATE_LIMIT_PER_SECOND", 10),
		LogLevel:                  getEnv("LOG_LEVEL", "info"),
	}

	if config.GoogleGeocodingAPIKey == "" {
		log.Println("Warning: GOOGLE_GEOCODING_API_KEY not set")
	}
	if config.IPInfoAPIKey == "" {
		log.Println("Warning: IPINFO_API_KEY not set")
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
