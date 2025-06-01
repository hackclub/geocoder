package main

import "fmt"

func calculateCacheSize(totalGB int, addressPercentage float64) {
	// From previous calculation:
	// Address cache: 2,276 bytes per record (with all overhead)
	// IP cache: 848 bytes per record (with all overhead)
	
	addressBytesPerRecord := 2276
	ipBytesPerRecord := 848
	
	// Total storage budget
	totalBytes := totalGB * 1024 * 1024 * 1024
	
	// Split the budget
	addressBytes := int(float64(totalBytes) * addressPercentage)
	ipBytes := totalBytes - addressBytes
	
	// Calculate max records
	maxAddressRecords := addressBytes / addressBytesPerRecord
	maxIPRecords := ipBytes / ipBytesPerRecord
	
	// Round down to nearest thousand for clean numbers
	recommendedAddresses := (maxAddressRecords / 1000) * 1000
	recommendedIPs := (maxIPRecords / 1000) * 1000
	
	// Calculate actual usage
	actualAddressBytes := recommendedAddresses * addressBytesPerRecord
	actualIPBytes := recommendedIPs * ipBytesPerRecord
	totalActualBytes := actualAddressBytes + actualIPBytes
	
	fmt.Printf("=== %d%% Addresses / %d%% IPs ===\n", 
		int(addressPercentage*100), int((1-addressPercentage)*100))
	fmt.Printf("Address cache: %d records → %.1f GB\n", 
		recommendedAddresses, float64(actualAddressBytes)/1024/1024/1024)
	fmt.Printf("IP cache: %d records → %.1f GB\n", 
		recommendedIPs, float64(actualIPBytes)/1024/1024/1024)
	fmt.Printf("Total usage: %.1f GB (%.1f%% of %d GB)\n\n", 
		float64(totalActualBytes)/1024/1024/1024, 
		float64(totalActualBytes)/float64(totalBytes)*100, totalGB)
}

func main() {
	totalGB := 20
	
	fmt.Printf("=== 20GB CACHE STORAGE SCENARIOS ===\n\n")
	
	// Different allocation strategies
	scenarios := []struct {
		name string
		addressPercent float64
	}{
		{"Address-heavy (80/20)", 0.80},
		{"Balanced (70/30)", 0.70},
		{"Moderate (60/40)", 0.60},
		{"Equal split (50/50)", 0.50},
		{"IP-heavy (40/60)", 0.40},
	}
	
	for _, scenario := range scenarios {
		fmt.Printf("** %s **\n", scenario.name)
		calculateCacheSize(totalGB, scenario.addressPercent)
	}
	
	// Show record size breakdown
	fmt.Printf("=== RECORD SIZE BREAKDOWN ===\n")
	fmt.Printf("Address records: 2,276 bytes each (includes all PostgreSQL overhead)\n")
	fmt.Printf("IP records: 848 bytes each (includes all PostgreSQL overhead)\n")
	fmt.Printf("Addresses are ~2.7x larger than IP records due to Google's detailed responses\n")
}
