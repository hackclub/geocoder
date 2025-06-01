package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"GOOGLE_GEOCODING_API_KEY":      os.Getenv("GOOGLE_GEOCODING_API_KEY"),
		"IPINFO_API_KEY":                os.Getenv("IPINFO_API_KEY"),
		"DATABASE_URL":                  os.Getenv("DATABASE_URL"),
		"PORT":                          os.Getenv("PORT"),
		"ADMIN_USERNAME":                os.Getenv("ADMIN_USERNAME"),
		"ADMIN_PASSWORD":                os.Getenv("ADMIN_PASSWORD"),
		"MAX_ADDRESS_CACHE_SIZE":        os.Getenv("MAX_ADDRESS_CACHE_SIZE"),
		"MAX_IP_CACHE_SIZE":             os.Getenv("MAX_IP_CACHE_SIZE"),
		"DEFAULT_RATE_LIMIT_PER_SECOND": os.Getenv("DEFAULT_RATE_LIMIT_PER_SECOND"),
		"LOG_LEVEL":                     os.Getenv("LOG_LEVEL"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Test with default values
	t.Run("Default values", func(t *testing.T) {
		// Clear all env vars
		for key := range originalVars {
			os.Unsetenv(key)
		}

		config := Load()

		if config.Port != "8080" {
			t.Errorf("Expected default port '8080', got '%s'", config.Port)
		}
		if config.AdminUsername != "admin" {
			t.Errorf("Expected default admin username 'admin', got '%s'", config.AdminUsername)
		}
		if config.MaxAddressCacheSize != 10000 {
			t.Errorf("Expected default address cache size 10000, got %d", config.MaxAddressCacheSize)
		}
		if config.DefaultRateLimitPerSecond != 10 {
			t.Errorf("Expected default rate limit 10, got %d", config.DefaultRateLimitPerSecond)
		}
	})

	// Test with custom values
	t.Run("Custom values", func(t *testing.T) {
		os.Setenv("GOOGLE_GEOCODING_API_KEY", "test-google-key")
		os.Setenv("IPINFO_API_KEY", "test-ipinfo-key")
		os.Setenv("DATABASE_URL", "postgres://test")
		os.Setenv("PORT", "9000")
		os.Setenv("ADMIN_USERNAME", "testadmin")
		os.Setenv("ADMIN_PASSWORD", "testpass")
		os.Setenv("MAX_ADDRESS_CACHE_SIZE", "5000")
		os.Setenv("MAX_IP_CACHE_SIZE", "2500")
		os.Setenv("DEFAULT_RATE_LIMIT_PER_SECOND", "20")
		os.Setenv("LOG_LEVEL", "debug")

		config := Load()

		if config.GoogleGeocodingAPIKey != "test-google-key" {
			t.Errorf("Expected Google API key 'test-google-key', got '%s'", config.GoogleGeocodingAPIKey)
		}
		if config.IPInfoAPIKey != "test-ipinfo-key" {
			t.Errorf("Expected IPInfo API key 'test-ipinfo-key', got '%s'", config.IPInfoAPIKey)
		}
		if config.DatabaseURL != "postgres://test" {
			t.Errorf("Expected database URL 'postgres://test', got '%s'", config.DatabaseURL)
		}
		if config.Port != "9000" {
			t.Errorf("Expected port '9000', got '%s'", config.Port)
		}
		if config.AdminUsername != "testadmin" {
			t.Errorf("Expected admin username 'testadmin', got '%s'", config.AdminUsername)
		}
		if config.AdminPassword != "testpass" {
			t.Errorf("Expected admin password 'testpass', got '%s'", config.AdminPassword)
		}
		if config.MaxAddressCacheSize != 5000 {
			t.Errorf("Expected address cache size 5000, got %d", config.MaxAddressCacheSize)
		}
		if config.MaxIPCacheSize != 2500 {
			t.Errorf("Expected IP cache size 2500, got %d", config.MaxIPCacheSize)
		}
		if config.DefaultRateLimitPerSecond != 20 {
			t.Errorf("Expected rate limit 20, got %d", config.DefaultRateLimitPerSecond)
		}
		if config.LogLevel != "debug" {
			t.Errorf("Expected log level 'debug', got '%s'", config.LogLevel)
		}
	})
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{"Use default", "TEST_KEY", "default", "", "default"},
		{"Use env value", "TEST_KEY", "default", "env-value", "env-value"},
		{"Empty env value", "TEST_KEY", "default", "", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			defer os.Unsetenv("TEST_KEY")

			if tt.envValue != "" {
				os.Setenv("TEST_KEY", tt.envValue)
			}

			result := getEnv("TEST_KEY", tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{"Use default", "TEST_INT", 42, "", 42},
		{"Use env value", "TEST_INT", 42, "100", 100},
		{"Invalid env value", "TEST_INT", 42, "invalid", 42},
		{"Empty env value", "TEST_INT", 42, "", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			defer os.Unsetenv("TEST_INT")

			if tt.envValue != "" {
				os.Setenv("TEST_INT", tt.envValue)
			}

			result := getEnvInt("TEST_INT", tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}
