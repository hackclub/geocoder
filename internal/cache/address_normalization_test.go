package cache

import (
	"testing"
)

func TestAddressNormalization(t *testing.T) {
	mockDB := newMockCacheDB()
	cacheService := NewService(mockDB, 1000, 1000)

	// Test cases for address normalization
	testCases := []struct {
		name    string
		addr1   string
		addr2   string
		shouldMatch bool
	}{
		{
			name:    "Case insensitive",
			addr1:   "15 Falls Road, Shelburne, VT",
			addr2:   "15 falls road, shelburne, vt",
			shouldMatch: true,
		},
		{
			name:    "Delimiter normalization",
			addr1:   "15 falls rd\\shelburne\\vt",
			addr2:   "15 falls rd, shelburne, vt",
			shouldMatch: true,
		},
		{
			name:    "Multiple spaces",
			addr1:   "15  falls   rd,  shelburne",
			addr2:   "15 falls rd, shelburne",
			shouldMatch: true,
		},
		{
			name:    "Abbreviation NOT normalized (conservative approach)",
			addr1:   "123 Main Street",
			addr2:   "123 main st",
			shouldMatch: false, // Conservative: don't risk changing meaning
		},
		{
			name:    "Road vs Rd NOT normalized",
			addr1:   "15 Falls Road",
			addr2:   "15 falls rd",
			shouldMatch: false, // Conservative: preserve original form
		},
		{
			name:    "Pipe delimiters normalized",
			addr1:   "15 falls rd|shelburne|vt",
			addr2:   "15 falls rd, shelburne, vt",
			shouldMatch: true,
		},
		{
			name:    "Semicolon preserved (conservative - might be meaningful)",
			addr1:   "15 falls rd;shelburne;vt",
			addr2:   "15 falls rd, shelburne, vt",
			shouldMatch: false, // Conservative: semicolons might have meaning
		},
		{
			name:    "Apartment NOT normalized (conservative)",
			addr1:   "123 Main St, Apartment 4B",
			addr2:   "123 main st, apt 4b",
			shouldMatch: false, // Conservative: preserve exact apartment designations
		},
		{
			name:    "Directional NOT normalized (conservative)",
			addr1:   "123 North Main Street",
			addr2:   "123 n main st",
			shouldMatch: false, // Conservative: "North Road" vs "N Road" could be different
		},
		{
			name:    "Completely different addresses",
			addr1:   "123 Main Street",
			addr2:   "456 Oak Avenue",
			shouldMatch: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash1 := cacheService.hashQuery(tc.addr1)
			hash2 := cacheService.hashQuery(tc.addr2)

			if tc.shouldMatch {
				if hash1 != hash2 {
					t.Errorf("Expected addresses to normalize to same hash:\n  '%s' -> %s\n  '%s' -> %s",
						tc.addr1, hash1[:16], tc.addr2, hash2[:16])
				}
			} else {
				if hash1 == hash2 {
					t.Errorf("Expected addresses to normalize to different hashes:\n  '%s' -> %s\n  '%s' -> %s",
						tc.addr1, hash1[:16], tc.addr2, hash2[:16])
				}
			}
		})
	}
}

func TestAddressNormalizationDetails(t *testing.T) {
	mockDB := newMockCacheDB()
	cacheService := NewService(mockDB, 1000, 1000)

	// Test specific normalization outputs (CONSERVATIVE approach)
	testCases := []struct {
		input    string
		expected string
	}{
		// SAFE transformations only
		{"15 Falls Road, Shelburne, VT", "15 falls road, shelburne, vt"}, // Case + trim only
		{"123 Main Street", "123 main street"},                           // Case only - no abbreviation
		{"456 Oak Avenue", "456 oak avenue"},                             // Case only - no abbreviation  
		{"789 Pine Boulevard", "789 pine boulevard"},                     // Case only - no abbreviation
		{"123 North Main Street", "123 north main street"},               // Case only - no directional abbrev
		{"456 Southwest Oak Avenue", "456 southwest oak avenue"},         // Case only - no directional abbrev
		{"15 falls rd\\shelburne\\vt", "15 falls rd, shelburne, vt"},     // Delimiter normalization ✅
		{"15 falls rd|shelburne|vt", "15 falls rd, shelburne, vt"},       // Delimiter normalization ✅
		{"123 Main St, Apartment 4B", "123 main st, apartment 4b"},       // Case only - no apt normalization
		{"123 Main St, Unit 4B", "123 main st, unit 4b"},                 // Case only - no unit normalization
		{"123 Main St, Suite 4B", "123 main st, suite 4b"},               // Case only - no suite normalization
		// Test spacing and delimiter safety
		{"  15  Falls  Road  ", "15 falls road"},                         // Space normalization ✅
		{"123|Main|Street", "123, main, street"},                         // Pipe delimiter ✅
		{"123\\Main\\Street", "123, main, street"},                       // Backslash delimiter ✅
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			normalized := cacheService.normalizeAddress(tc.input)
			if normalized != tc.expected {
				t.Errorf("Address normalization failed:\n  Input: '%s'\n  Expected: '%s'\n  Got: '%s'",
					tc.input, tc.expected, normalized)
			}
		})
	}
}
