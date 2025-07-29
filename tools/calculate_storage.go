package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	// Sample geocoding response (typical size)
	geocodeResponse := map[string]interface{}{
		"lat":               37.4223,
		"lng":               -122.0844,
		"formatted_address": "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
		"state_name":        "California",
		"state_code":        "CA",
		"country_name":      "United States",
		"country_code":      "US",
		"backend":           "google_maps_platform_geocoding",
		"raw_backend_response": map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"address_components": []map[string]interface{}{
						{"long_name": "1600", "short_name": "1600", "types": []string{"street_number"}},
						{"long_name": "Amphitheatre Pkwy", "short_name": "Amphitheatre Pkwy", "types": []string{"route"}},
						{"long_name": "Mountain View", "short_name": "Mountain View", "types": []string{"locality", "political"}},
						{"long_name": "Santa Clara County", "short_name": "Santa Clara County", "types": []string{"administrative_area_level_2", "political"}},
						{"long_name": "California", "short_name": "CA", "types": []string{"administrative_area_level_1", "political"}},
						{"long_name": "United States", "short_name": "US", "types": []string{"country", "political"}},
						{"long_name": "94043", "short_name": "94043", "types": []string{"postal_code"}},
					},
					"formatted_address": "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
					"geometry": map[string]interface{}{
						"location": map[string]interface{}{
							"lat": 37.4223878,
							"lng": -122.0844511,
						},
						"location_type": "ROOFTOP",
						"viewport": map[string]interface{}{
							"northeast": map[string]interface{}{"lat": 37.4237367802915, "lng": -122.0831021197085},
							"southwest": map[string]interface{}{"lat": 37.4210388197085, "lng": -122.0858000802915},
						},
					},
					"place_id": "ChIJtYuu0V25j4ARwu5e4wwRYgE",
					"types":    []string{"street_address"},
				},
			},
			"status": "OK",
		},
	}

	// Sample IP geolocation response
	ipResponse := map[string]interface{}{
		"lat":          37.4056,
		"lng":          -122.0775,
		"ip":           "8.8.8.8",
		"city":         "Mountain View",
		"region":       "California",
		"country_name": "United States",
		"country_code": "US",
		"postal_code":  "94043",
		"timezone":     "America/Los_Angeles",
		"org":          "Google LLC",
		"backend":      "ipinfo_api",
		"raw_backend_response": map[string]interface{}{
			"ip":       "8.8.8.8",
			"hostname": "dns.google",
			"city":     "Mountain View",
			"region":   "California",
			"country":  "US",
			"loc":      "37.4056,-122.0775",
			"postal":   "94043",
			"timezone": "America/Los_Angeles",
			"org":      "AS15169 Google LLC",
		},
	}

	// Convert to JSON to measure size
	geocodeJSON, _ := json.Marshal(geocodeResponse)
	ipJSON, _ := json.Marshal(ipResponse)

	fmt.Printf("Sample geocoding response JSON size: %d bytes\n", len(geocodeJSON))
	fmt.Printf("Sample IP response JSON size: %d bytes\n", len(ipJSON))

	// Calculate table overhead per row
	// PostgreSQL overhead per row: ~24 bytes minimum
	// SERIAL PRIMARY KEY (id): 4 bytes
	// TIMESTAMP (created_at): 8 bytes
	// Additional indexes and alignment: ~8 bytes
	rowOverhead := 24 + 4 + 8 + 8 // 44 bytes per row

	// Address cache calculations
	fmt.Printf("\n=== ADDRESS CACHE (10,000 records) ===\n")
	
	// Address cache fields:
	// - query_hash: VARCHAR(64) = ~64 bytes
	// - query_text: TEXT = ~50 bytes average (typical address length)
	// - response_data: JSONB = geocodeJSON size + JSONB overhead (~20%)
	avgAddressLength := 50
	geocodeJSONBSize := int(float64(len(geocodeJSON)) * 1.2) // JSONB overhead
	
	addressRecordSize := rowOverhead + 64 + avgAddressLength + geocodeJSONBSize
	totalAddressCache := addressRecordSize * 10000
	
	fmt.Printf("Average address query length: %d bytes\n", avgAddressLength)
	fmt.Printf("JSONB response data per record: %d bytes\n", geocodeJSONBSize)
	fmt.Printf("Total per address record: %d bytes\n", addressRecordSize)
	fmt.Printf("Total for 10,000 addresses: %d bytes (%.2f MB)\n", totalAddressCache, float64(totalAddressCache)/1024/1024)

	// IP cache calculations  
	fmt.Printf("\n=== IP CACHE (10,000 records) ===\n")
	
	// IP cache fields:
	// - ip_address: INET = ~16 bytes (IPv4/IPv6)
	// - response_data: JSONB = ipJSON size + JSONB overhead
	ipJSONBSize := int(float64(len(ipJSON)) * 1.2) // JSONB overhead
	
	ipRecordSize := rowOverhead + 16 + ipJSONBSize
	totalIPCache := ipRecordSize * 10000
	
	fmt.Printf("IP address field size: 16 bytes\n")
	fmt.Printf("JSONB response data per record: %d bytes\n", ipJSONBSize)
	fmt.Printf("Total per IP record: %d bytes\n", ipRecordSize)
	fmt.Printf("Total for 10,000 IP records: %d bytes (%.2f MB)\n", totalIPCache, float64(totalIPCache)/1024/1024)

	// Combined totals
	fmt.Printf("\n=== COMBINED TOTALS ===\n")
	totalSize := totalAddressCache + totalIPCache
	fmt.Printf("Total cache storage: %d bytes (%.2f MB)\n", totalSize, float64(totalSize)/1024/1024)
	
	// Add index overhead (rough estimate: 20-30% of data size)
	indexOverhead := int(float64(totalSize) * 0.25)
	totalWithIndexes := totalSize + indexOverhead
	fmt.Printf("With indexes (~25%% overhead): %d bytes (%.2f MB)\n", totalWithIndexes, float64(totalWithIndexes)/1024/1024)
	
	// PostgreSQL page overhead and WAL
	postgresOverhead := int(float64(totalWithIndexes) * 0.15)
	totalWithPostgresOverhead := totalWithIndexes + postgresOverhead
	fmt.Printf("With PostgreSQL overhead (~15%%): %d bytes (%.2f MB)\n", totalWithPostgresOverhead, float64(totalWithPostgresOverhead)/1024/1024)
}
