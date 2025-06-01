package cache

import (
	"testing"

	"github.com/hackclub/geocoder/internal/models"
)

// TestCacheHitsForSimilarAddresses verifies that similar addresses result in cache hits
func TestCacheHitsForSimilarAddresses(t *testing.T) {
	mockDB := newMockCacheDB()
	cacheService := NewService(mockDB, 1000, 1000)

	// Create a sample geocoding response
	originalResponse := &models.GeocodeAPIResponse{
		Lat:              37.4223,
		Lng:              -122.0844,
		FormattedAddress: "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
		CountryName:      "United States",
		CountryCode:      "US",
		Backend:          "google_maps_platform_geocoding",
		RawBackendResponse: map[string]interface{}{
			"status": "OK",
		},
	}

	// Test that similar addresses share the same cache
	testCases := []struct {
		name              string
		addresses         []string
		shouldShareCache  bool
	}{
		{
			name: "Case variations should share cache",
			addresses: []string{
				"1600 Amphitheatre Parkway, Mountain View, CA",
				"1600 AMPHITHEATRE PARKWAY, MOUNTAIN VIEW, CA",
				"1600 amphitheatre parkway, mountain view, ca",
			},
			shouldShareCache: true,
		},
		{
			name: "Whitespace variations should share cache",
			addresses: []string{
				"1600 Amphitheatre Parkway, Mountain View, CA",
				"  1600  Amphitheatre  Parkway,  Mountain  View,  CA  ",
				"1600   Amphitheatre   Parkway,   Mountain   View,   CA",
			},
			shouldShareCache: true,
		},
		{
			name: "Delimiter variations should share cache",
			addresses: []string{
				"1600 Amphitheatre Parkway, Mountain View, CA",
				"1600 Amphitheatre Parkway\\Mountain View\\CA",
				"1600 Amphitheatre Parkway|Mountain View|CA",
			},
			shouldShareCache: true,
		},
		{
			name: "Mixed variations should share cache",
			addresses: []string{
				"1600 Amphitheatre Parkway, Mountain View, CA",
				"  1600  AMPHITHEATRE  PARKWAY,  MOUNTAIN  VIEW,  CA  ",
				"1600 amphitheatre parkway\\mountain view\\ca",
			},
			shouldShareCache: true,
		},
		{
			name: "Conservative: abbreviations should NOT share cache (safe)",
			addresses: []string{
				"1600 Amphitheatre Parkway, Mountain View, CA",
				"1600 Amphitheatre Pkwy, Mountain View, CA",
			},
			shouldShareCache: false, // Conservative approach preserves exact terms
		},
		{
			name: "Conservative: periods should NOT share cache (safe)",
			addresses: []string{
				"1600 Amphitheatre Parkway, Mountain View, CA",
				"1600 Amphitheatre Parkway. Mountain View. CA",
			},
			shouldShareCache: false, // Periods might be meaningful
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear the mock database for each test
			mockDB.addressCache = make(map[string]*models.AddressCache)

			if len(tc.addresses) < 2 {
				t.Fatal("Test case must have at least 2 addresses")
			}

			baseAddress := tc.addresses[0]

			// Cache the response for the first address
			err := cacheService.SetStandardGeocodeResult(baseAddress, originalResponse)
			if err != nil {
				t.Fatalf("Failed to cache result: %v", err)
			}

			// Test that other addresses either hit or miss the cache as expected
			for i, addr := range tc.addresses[1:] {
				cached, hit := cacheService.GetStandardGeocodeResult(addr)

				if tc.shouldShareCache {
					if !hit {
						t.Errorf("Address %d should have been a cache hit but was a miss:\n  Base: '%s'\n  Test: '%s'", 
							i+1, baseAddress, addr)
						continue
					}
					
					// Verify the cached data is correct
					if cached.Lat != originalResponse.Lat || cached.Lng != originalResponse.Lng {
						t.Errorf("Cached response data doesn't match original for address %d:\n  Expected: lat=%f, lng=%f\n  Got: lat=%f, lng=%f",
							i+1, originalResponse.Lat, originalResponse.Lng, cached.Lat, cached.Lng)
					}
				} else {
					if hit {
						t.Errorf("Address %d should have been a cache miss but was a hit (conservative approach):\n  Base: '%s'\n  Test: '%s'", 
							i+1, baseAddress, addr)
					}
				}
			}
		})
	}
}

// TestCacheEfficiencyImprovement demonstrates the cache efficiency improvement
func TestCacheEfficiencyImprovement(t *testing.T) {
	mockDB := newMockCacheDB()
	cacheService := NewService(mockDB, 1000, 1000)

	// Sample response
	response := &models.GeocodeAPIResponse{
		Lat:              40.7128,
		Lng:              -74.0060,
		FormattedAddress: "New York, NY, USA",
		CountryName:      "United States",
		CountryCode:      "US",
		Backend:          "google_maps_platform_geocoding",
		RawBackendResponse: map[string]interface{}{"status": "OK"},
	}

	// Addresses that should normalize to the same cache entry
	similarAddresses := []string{
		"123 Main Street, New York, NY",
		"123 MAIN STREET, NEW YORK, NY",
		"  123  Main  Street,  New  York,  NY  ",
		"123 Main Street\\New York\\NY",
		"123|Main|Street,|New|York,|NY",
		"123 main street, new york, ny",
	}

	// Cache the first address
	err := cacheService.SetStandardGeocodeResult(similarAddresses[0], response)
	if err != nil {
		t.Fatalf("Failed to cache initial address: %v", err)
	}

	// Count cache hits
	cacheHits := 0
	totalLookups := len(similarAddresses) - 1 // Exclude the first one we cached

	for _, addr := range similarAddresses[1:] {
		_, hit := cacheService.GetStandardGeocodeResult(addr)
		if hit {
			cacheHits++
		}
	}

	// Calculate cache hit rate
	hitRate := float64(cacheHits) / float64(totalLookups) * 100

	t.Logf("Cache efficiency test results:")
	t.Logf("  Total lookups: %d", totalLookups)
	t.Logf("  Cache hits: %d", cacheHits)
	t.Logf("  Cache hit rate: %.1f%%", hitRate)

	// We expect a significant improvement (at least 80% hit rate for these similar addresses)
	expectedMinHitRate := 80.0
	if hitRate < expectedMinHitRate {
		t.Errorf("Cache hit rate %.1f%% is below expected minimum %.1f%%", hitRate, expectedMinHitRate)
	}

	t.Logf("✅ Conservative normalization achieved %.1f%% cache hit rate for similar addresses", hitRate)
}
