package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestWarpProcessing(t *testing.T) {
	// Setup
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)

	tests := []struct {
		name           string
		warpData       string
		expectedWarps  []int
		expectedCount  int
		description    string
	}{
		{
			name:          "Basic dash-separated warps",
			warpData:      "2 - 3 - 4 - 5 - 6 - 7",
			expectedWarps: []int{2, 3, 4, 5, 6, 7},
			expectedCount: 6,
			description:   "Standard sector warp format with dashes",
		},
		{
			name:          "Comma-separated warps",
			warpData:      "100, 200, 300",
			expectedWarps: []int{100, 200, 300},
			expectedCount: 3,
			description:   "Alternative comma-separated format",
		},
		{
			name:          "Space-separated warps",
			warpData:      "1000 2000 3000 4000",
			expectedWarps: []int{1000, 2000, 3000, 4000},
			expectedCount: 4,
			description:   "Space-separated format",
		},
		{
			name:          "With parentheses",
			warpData:      "(150) - 250 - (350)",
			expectedWarps: []int{150, 250, 350},
			expectedCount: 3,
			description:   "Warps with parentheses should be cleaned",
		},
		{
			name:          "Unsorted warps",
			warpData:      "500 - 100 - 300 - 200",
			expectedWarps: []int{100, 200, 300, 500},
			expectedCount: 4,
			description:   "Warps should be sorted automatically",
		},
		{
			name:          "Duplicate warps",
			warpData:      "100 - 200 - 100 - 300",
			expectedWarps: []int{100, 200, 300},
			expectedCount: 3,
			description:   "Duplicate warps should be filtered out",
		},
		{
			name:          "Too many warps",
			warpData:      "1 - 2 - 3 - 4 - 5 - 6 - 7 - 8 - 9",
			expectedWarps: []int{1, 2, 3, 4, 5, 6},
			expectedCount: 6,
			description:   "Should limit to maximum 6 warps",
		},
		{
			name:          "Invalid sector numbers",
			warpData:      "0 - -100 - 150000 - 100",
			expectedWarps: []int{100},
			expectedCount: 1,
			description:   "Should filter out invalid sector numbers",
		},
		{
			name:          "Empty and whitespace",
			warpData:      "100 -  - 200 -   - 300",
			expectedWarps: []int{100, 200, 300},
			expectedCount: 3,
			description:   "Should handle empty strings and whitespace",
		},
		{
			name:          "Single warp",
			warpData:      "42",
			expectedWarps: []int{42},
			expectedCount: 1,
			description:   "Should handle single warp correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset parser state
			parser.currentSectorIndex = 1000
			parser.currentSectorWarps = [6]int{} // Reset warps array

			// Parse the warp data
			parser.parseWarpConnections(tt.warpData)

			// Verify the expected warps
			actualCount := 0
			for i, warp := range parser.currentSectorWarps {
				if warp != 0 {
					actualCount++
					if i < len(tt.expectedWarps) {
						if warp != tt.expectedWarps[i] {
							t.Errorf("Warp %d: expected %d, got %d", i, tt.expectedWarps[i], warp)
						}
					}
				}
			}

			if actualCount != tt.expectedCount {
				t.Errorf("Expected %d warps, got %d", tt.expectedCount, actualCount)
			}

			t.Logf("Test passed: %s", tt.description)
		})
	}
}

func TestWarpValidation(t *testing.T) {
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)

	tests := []struct {
		sector   int
		expected bool
		reason   string
	}{
		{0, false, "Zero sector should be invalid"},
		{-1, false, "Negative sector should be invalid"},
		{1, true, "Valid positive sector"},
		{5000, true, "Valid mid-range sector"},
		{20000, true, "Valid high-range sector"},
		{20001, false, "Sector above reasonable limit should be invalid"},
		{999999, false, "Very high sector should be invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			result := parser.validateWarpSector(tt.sector)
			if result != tt.expected {
				t.Errorf("validateWarpSector(%d): expected %t, got %t - %s", 
					tt.sector, tt.expected, result, tt.reason)
			}
		})
	}
}

func TestWarpSorting(t *testing.T) {
	parser := NewTWXParser(nil, nil)

	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "Already sorted",
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "Reverse sorted",
			input:    []int{5, 4, 3, 2, 1},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "Random order",
			input:    []int{3, 1, 4, 2, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "With duplicates",
			input:    []int{3, 1, 3, 2, 1},
			expected: []int{1, 1, 2, 3, 3},
		},
		{
			name:     "Single element",
			input:    []int{42},
			expected: []int{42},
		},
		{
			name:     "Empty array",
			input:    []int{},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test data
			result := make([]int, len(tt.input))
			copy(result, tt.input)
			
			parser.sortWarps(result)

			if len(result) != len(tt.expected) {
				t.Fatalf("Length mismatch: expected %d, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Index %d: expected %d, got %d", i, expected, result[i])
				}
			}
		})
	}
}

func TestContainsWarp(t *testing.T) {
	parser := NewTWXParser(nil, nil)

	tests := []struct {
		warps    []int
		warp     int
		expected bool
		name     string
	}{
		{[]int{1, 2, 3, 4, 5}, 3, true, "Contains existing warp"},
		{[]int{1, 2, 3, 4, 5}, 6, false, "Does not contain non-existing warp"},
		{[]int{}, 1, false, "Empty array does not contain any warp"},
		{[]int{42}, 42, true, "Single element array contains the element"},
		{[]int{42}, 43, false, "Single element array does not contain different element"},
		{[]int{100, 200, 300}, 200, true, "Contains middle element"},
		{[]int{100, 200, 300}, 100, true, "Contains first element"},
		{[]int{100, 200, 300}, 300, true, "Contains last element"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.containsWarp(tt.warps, tt.warp)
			if result != tt.expected {
				t.Errorf("containsWarp(%v, %d): expected %t, got %t", 
					tt.warps, tt.warp, tt.expected, result)
			}
		})
	}
}

func TestWarpProcessingIntegration(t *testing.T) {
	// Create an in-memory test database
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	// Test complete warp processing flow
	t.Run("Complete warp processing flow", func(t *testing.T) {
		// Set up a sector
		parser.currentSectorIndex = 1000

		// Process a sector with warps
		lines := []string{
			"Sector  : 1000 in Test Space",
			"Beacon  : Test Beacon",
			"Warps to Sector(s) : 1001 - 1002 - 1003",
		}

		for _, line := range lines {
			parser.ProcessInBound(line + "\r")
		}

		// Force sector completion to ensure database save
		parser.sectorCompleted()

		// Verify the warps were processed correctly
		expectedWarps := []int{1001, 1002, 1003}
		for i, expected := range expectedWarps {
			if parser.currentSectorWarps[i] != expected {
				t.Errorf("Warp %d: expected %d, got %d", i, expected, parser.currentSectorWarps[i])
			}
		}

		// Verify sector was saved to database
		savedSector, err := db.LoadSector(1000)
		if err != nil {
			t.Fatalf("Failed to load saved sector: %v", err)
		}

		// Check that warps were saved
		for i, expected := range expectedWarps {
			if savedSector.Warp[i] != expected {
				t.Errorf("Saved warp %d: expected %d, got %d", i, expected, savedSector.Warp[i])
			}
		}
	})

	t.Run("Reverse warp connections", func(t *testing.T) {
		// Set up initial sectors in database
		for i := 2000; i <= 2003; i++ {
			sector := database.NULLSector()
			sector.Constellation = "Test Space"
			if err := db.SaveSector(sector, i); err != nil {
				t.Fatalf("Failed to save test sector %d: %v", i, err)
			}
		}

		// Process a sector with warps to other sectors
		parser.currentSectorIndex = 2000
		parser.parseWarpConnections("2001 - 2002 - 2003")

		// Check that reverse warps were added to destination sectors
		for _, destSector := range []int{2001, 2002, 2003} {
			sector, err := db.LoadSector(destSector)
			if err != nil {
				t.Fatalf("Failed to load sector %d: %v", destSector, err)
			}

			// Check if reverse warp to 2000 was added
			found := false
			for _, warp := range sector.Warp {
				if warp == 2000 {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Reverse warp from %d to 2000 was not added", destSector)
			}
		}
	})
}

func TestRealWorldWarpData(t *testing.T) {
	// Test cases based on actual data from raw.log
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)

	realWorldTests := []struct {
		name           string
		warpData       string
		expectedWarps  []int
		expectedCount  int
		description    string
	}{
		{
			name:          "Real data format 1 - parentheses mixed",
			warpData:      "  (8247) - 18964",
			expectedWarps: []int{8247, 18964},
			expectedCount: 2,
			description:   "From raw.log line 130: mixed parentheses format",
		},
		{
			name:          "Real data format 2 - multiple parentheses", 
			warpData:      " 2142 - (13975) - (16563) - (16589)",
			expectedWarps: []int{2142, 13975, 16563, 16589},
			expectedCount: 4,
			description:   "From raw.log line 148: multiple sectors with parentheses",
		},
		{
			name:          "Real data format 3 - single warp",
			warpData:      " 8247",
			expectedWarps: []int{8247},
			expectedCount: 1,
			description:   "From raw.log line 170: single warp connection",
		},
		{
			name:          "Real data with extra spaces",
			warpData:      "  (1234)  -  5678  - (9012) ",
			expectedWarps: []int{1234, 5678, 9012},
			expectedCount: 3,
			description:   "Real format with various spacing",
		},
		{
			name:          "High sector numbers",
			warpData:      "18964 - 16563 - 16589",
			expectedWarps: []int{16563, 16589, 18964},
			expectedCount: 3,
			description:   "High sector numbers like in real game",
		},
	}

	for _, tt := range realWorldTests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset parser state
			parser.currentSectorIndex = 2142 // Use real sector from log
			parser.currentSectorWarps = [6]int{} // Reset warps array

			// Parse the warp data
			parser.parseWarpConnections(tt.warpData)

			// Verify the expected warps
			actualCount := 0
			for i, warp := range parser.currentSectorWarps {
				if warp != 0 {
					actualCount++
					if i < len(tt.expectedWarps) {
						if warp != tt.expectedWarps[i] {
							t.Errorf("Warp %d: expected %d, got %d", i, tt.expectedWarps[i], warp)
						}
					}
				}
			}

			if actualCount != tt.expectedCount {
				t.Errorf("Expected %d warps, got %d", tt.expectedCount, actualCount)
			}

			t.Logf("✓ Real data test passed: %s", tt.description)
		})
	}
}

func TestCompleteRealWorldSectorParsing(t *testing.T) {
	// Test complete sector parsing with real data from raw.log
	db := database.NewDatabase() 
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	// Real sector data from raw.log
	realSectorData := []string{
		"Sector  : 2142 in uncharted space.",
		"Ports   : Ranger Annex, Class 1 (BBS)",
		"Warps to Sector(s) :  (8247) - 18964",
	}

	t.Run("Complete real sector parsing", func(t *testing.T) {
		for _, line := range realSectorData {
			parser.ProcessInBound(line + "\r")
		}

		// Force sector completion to ensure database save
		parser.sectorCompleted()

		// Verify the sector was parsed correctly
		if parser.currentSectorIndex != 2142 {
			t.Errorf("Expected sector 2142, got %d", parser.currentSectorIndex)
		}

		// Verify warps were parsed
		expectedWarps := []int{8247, 18964}
		for i, expected := range expectedWarps {
			if parser.currentSectorWarps[i] != expected {
				t.Errorf("Warp %d: expected %d, got %d", i, expected, parser.currentSectorWarps[i])
			}
		}

		// Verify sector was saved to database
		savedSector, err := db.LoadSector(2142)
		if err != nil {
			t.Fatalf("Failed to load saved sector: %v", err)
		}

		if savedSector.Constellation != "uncharted space" {
			t.Errorf("Expected constellation 'uncharted space', got '%s'", savedSector.Constellation)
		}

		for i, expected := range expectedWarps {
			if savedSector.Warp[i] != expected {
				t.Errorf("Saved warp %d: expected %d, got %d", i, expected, savedSector.Warp[i])
			}
		}

		t.Log("✓ Complete real world sector parsing test passed")
	})
}

// Benchmark warp processing performance
func BenchmarkWarpProcessing(b *testing.B) {
	parser := NewTWXParser(nil, nil)
	testData := "1000 - 2000 - 3000 - 4000 - 5000 - 6000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.parseWarpConnections(testData)
	}
}

func BenchmarkWarpSorting(b *testing.B) {
	parser := NewTWXParser(nil, nil)
	testWarps := []int{5000, 1000, 3000, 2000, 4000, 6000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy for each iteration
		warps := make([]int, len(testWarps))
		copy(warps, testWarps)
		parser.sortWarps(warps)
	}
}

func BenchmarkRealWorldWarpProcessing(b *testing.B) {
	parser := NewTWXParser(nil, nil)
	// Real data from raw.log
	testData := " 2142 - (13975) - (16563) - (16589)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.parseWarpConnections(testData)
	}
}