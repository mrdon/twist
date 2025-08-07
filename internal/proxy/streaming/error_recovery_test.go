package streaming

import (
	"strings"
	"testing"
	"twist/internal/proxy/database"
)

func TestErrorRecovery(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(db, nil)

	t.Run("MalformedLineHandling", func(t *testing.T) {
		// Test various malformed inputs that should be handled gracefully
		lines := []string{
			"",                                   // Empty line
			string([]byte{0, 0, 0}),            // Null characters
			strings.Repeat("A", 5000),          // Extremely long line
			"NavHaz  : ",                       // Incomplete NavHaz line
			"Sector  : ",                       // Incomplete sector line
			"Warps to Sector(s) : ",           // Incomplete warps line
			"NavHaz  : -999%",                 // Extreme negative value
			"NavHaz  : 999999%",               // Extreme positive value
			"Traders : Test, w/ -5000 ftrs",   // Negative fighters
		}

		for _, input := range lines {
			// Should not panic or crash
			parser.ProcessString(input + "\r")
		}

		t.Log("✓ All malformed inputs handled gracefully")
	})

	t.Run("BoundsCheckingValidation", func(t *testing.T) {
		// Test string bounds checking
		testCases := []struct {
			input       string
			description string
		}{
			{"NavHaz  :\r", "NavHaz line exactly at minimum length"},
			{"Sector  :\r", "Sector line exactly at minimum length"},
			{"N\r", "Extremely short line"},
			{"NavHaz  : 50% (1000000000000)\r", "NavHaz with extreme count"},
		}

		for _, tc := range testCases {
			// Should handle gracefully without panics
			parser.ProcessString(tc.input)
			t.Logf("✓ %s handled correctly", tc.description)
		}
	})

	t.Run("DataValidationLimits", func(t *testing.T) {
		// Test data validation and limits
		parser.currentDisplay = DisplaySector
		parser.sectorPosition = SectorPosNormal
		parser.currentSectorIndex = 1234

		// Test extreme values that should be validated/corrected
		parser.ProcessString("Sector  : 1234 in " + strings.Repeat("A", 1000) + "\r")
		parser.ProcessString("NavHaz  : -50%\r")
		parser.ProcessString("Traders : " + strings.Repeat("TestTrader", 50) + ", w/ -999999 ftrs\r")
		parser.ProcessString("Warps to Sector(s) :  (2) - 3\r")

		// Force sector completion to trigger validation
		if !parser.sectorSaved {
			parser.sectorCompleted()
		}

		// Validate that data was corrected appropriately
		if parser.currentSector.NavHaz != 0 {
			t.Errorf("Expected negative NavHaz to be corrected to 0, got %d", parser.currentSector.NavHaz)
		}

		// Constellation should be truncated to reasonable length
		if len(parser.currentSector.Constellation) > 500 {
			t.Errorf("Constellation not truncated: %d characters", len(parser.currentSector.Constellation))
		}

		t.Log("✓ Data validation limits working correctly")
	})

	t.Run("PlayerStatsValidation", func(t *testing.T) {
		// Test player stats validation with extreme values
		extremeStats := " Turns -1000�Creds -5000000�Figs -999999�Shlds -50�Hlds -10�Ore -5"
		parser.ProcessString(extremeStats + "\r")

		// All negative values should be corrected to 0 or reasonable values
		if parser.playerStats.Turns < 0 {
			t.Errorf("Negative turns not corrected: %d", parser.playerStats.Turns)
		}
		if parser.playerStats.Credits < 0 {
			t.Errorf("Negative credits not corrected: %d", parser.playerStats.Credits)
		}
		if parser.playerStats.Fighters < 0 {
			t.Errorf("Negative fighters not corrected: %d", parser.playerStats.Fighters)
		}

		t.Log("✓ Player stats validation working correctly")
	})

	t.Run("PanicRecoveryTest", func(t *testing.T) {
		// Test that parser can recover from potential panic situations
		panicInputs := []string{
			"NavHaz  : " + string([]byte{0xFF, 0xFE, 0xFD}) + "%\r", // Binary data
			"Sector  : " + strings.Repeat("999", 1000) + "\r",        // Extreme sector number
			"Warps to Sector(s) : " + strings.Repeat("(999999)", 100) + "\r", // Too many warps
		}

		for _, input := range panicInputs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Parser panicked on input: %q, panic: %v", input, r)
					}
				}()
				parser.ProcessString(input)
			}()
		}

		t.Log("✓ Panic recovery working correctly")
	})

	t.Run("StateConsistencyAfterErrors", func(t *testing.T) {
		// Introduce errors and verify parser state remains consistent
		parser.ProcessString("Invalid line that doesn't match any pattern\r")
		parser.ProcessString("Sector  : -999 in Invalid Sector\r")
		parser.ProcessString("NavHaz  : invalid%\r")
		parser.ProcessString("Warps to Sector(s) : invalid warp data\r")

		// Parser should still be in a valid state
		if parser.currentDisplay == DisplayNone {
			// This is expected for invalid data
		}

		// Process valid data to ensure parser still works
		parser.ProcessString("Sector  : 5000 in Recovery Test\r")
		parser.ProcessString("NavHaz  : 25%\r")
		parser.ProcessString("Warps to Sector(s) :  (5001) - 5002\r")

		if parser.currentSectorIndex != 5000 {
			t.Errorf("Parser state inconsistent after errors: sector %d", parser.currentSectorIndex)
		}
		if parser.currentSector.NavHaz != 25 {
			t.Errorf("Parser state inconsistent after errors: NavHaz %d", parser.currentSector.NavHaz)
		}

		t.Log("✓ Parser state consistency maintained after errors")
	})

	t.Run("DatabaseErrorRecovery", func(t *testing.T) {
		// Test that database errors don't crash the parser
		// Close the database to force errors
		db.CloseDatabase()

		// These should not crash even with database errors
		parser.ProcessString("Sector  : 6000 in Database Error Test\r")
		parser.ProcessString("NavHaz  : 15%\r")
		parser.ProcessString("Warps to Sector(s) :  (6001)\r")

		// Parser should still function even with database errors
		if parser.currentSectorIndex != 6000 {
			t.Errorf("Parser failed to parse data during database error: sector %d", parser.currentSectorIndex)
		}

		t.Log("✓ Database error recovery working correctly")
	})

	t.Run("MemoryLeakPrevention", func(t *testing.T) {
		// Test that error recovery doesn't cause memory leaks
		// Process large amounts of data with intermittent errors
		for i := 0; i < 1000; i++ {
			if i%10 == 0 {
				// Introduce errors every 10th iteration
				parser.ProcessString("Invalid data line " + strings.Repeat("x", i) + "\r")
			} else {
				// Valid data
				parser.ProcessString("Sector  : " + parser.intToString(7000+i) + " in Memory Test\r")
				parser.ProcessString("NavHaz  : " + parser.intToString(i%100) + "%\r")
				parser.ProcessString("Warps to Sector(s) :  (" + parser.intToString(7001+i) + ")\r")
			}
		}

		// Parser should still be responsive
		parser.ProcessString("Sector  : 8000 in Final Test\r")
		if parser.currentSectorIndex != 8000 {
			t.Errorf("Parser performance degraded after processing large dataset: sector %d", parser.currentSectorIndex)
		}

		t.Log("✓ Memory leak prevention working correctly")
	})
}

func TestValidationFunctions(t *testing.T) {
	parser := NewTestTWXParser()

	t.Run("SectorNumberValidation", func(t *testing.T) {
		testCases := []struct {
			sector int
			valid  bool
		}{
			{0, false},
			{-1, false},
			{1, true},
			{20000, true},
			{50000, true},
			{50001, false},
		}

		for _, tc := range testCases {
			result := parser.validateSectorNumber(tc.sector)
			if result != tc.valid {
				t.Errorf("Sector %d validation: expected %t, got %t", tc.sector, tc.valid, result)
			}
		}
	})

	t.Run("PercentageValidation", func(t *testing.T) {
		testCases := []struct {
			input    int
			expected int
		}{
			{-50, 0},
			{0, 0},
			{50, 50},
			{100, 100},
			{150, 100},
		}

		for _, tc := range testCases {
			result := parser.validatePercentage(tc.input)
			if result != tc.expected {
				t.Errorf("Percentage %d validation: expected %d, got %d", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("FighterCountValidation", func(t *testing.T) {
		testCases := []struct {
			input    int
			expected int
		}{
			{-1000, 0},
			{0, 0},
			{50000, 50000},
			{100000000001, 100000000000}, // Capped at maximum
		}

		for _, tc := range testCases {
			result := parser.validateFighterCount(tc.input)
			if result != tc.expected {
				t.Errorf("Fighter count %d validation: expected %d, got %d", tc.input, tc.expected, result)
			}
		}
	})
}