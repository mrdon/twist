package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestVersionDetection(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParserWithAPI(db, nil)

	t.Run("TWGS Version Detection", func(t *testing.T) {
		// Test TWGS server detection (Pascal: Copy(Line, 1, 14) = 'TradeWars Game')
		testCases := []struct {
			name           string
			input          string
			expectedType   int    // 0=unknown, 1=TW2002, 2=TWGS
			expectedTWGS   string
			expectedTW2002 string
			description    string
		}{
			{
				name:           "TWGS Server Detection",
				input:          "TradeWars Game Server v2.20b",
				expectedType:   2,
				expectedTWGS:   "2.20b",
				expectedTW2002: "3.34",
				description:    "Pascal: FTWGSType := 2; FTWGSVer := '2.20b'; FTW2002Ver := '3.34'",
			},
			{
				name:           "TWGS with different format",
				input:          "TradeWars Game System Online",
				expectedType:   2,
				expectedTWGS:   "2.20b",
				expectedTW2002: "3.34",
				description:    "Any line starting with 'TradeWars Game' should trigger TWGS detection",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state
				parser.twgsType = 0
				parser.twgsVer = ""
				parser.tw2002Ver = ""
				
				// Process the version detection line
				parser.ProcessString(tc.input + "\r")
				
				// Verify version detection results
				if parser.GetTWGSType() != tc.expectedType {
					t.Errorf("Expected TWGS type %d, got %d", tc.expectedType, parser.GetTWGSType())
				}
				
				if parser.GetTWGSVersion() != tc.expectedTWGS {
					t.Errorf("Expected TWGS version '%s', got '%s'", tc.expectedTWGS, parser.GetTWGSVersion())
				}
				
				if parser.GetTW2002Version() != tc.expectedTW2002 {
					t.Errorf("Expected TW2002 version '%s', got '%s'", tc.expectedTW2002, parser.GetTW2002Version())
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("TW2002 Version Detection", func(t *testing.T) {
		// Test TW2002 server detection (Pascal: Copy(Line, 1, 20) = 'Trade Wars 2002 Game')
		testCases := []struct {
			name           string
			input          string
			expectedType   int
			expectedTWGS   string
			expectedTW2002 string
			description    string
		}{
			{
				name:           "TW2002 Server Detection",
				input:          "Trade Wars 2002 Game Server v1.03",
				expectedType:   1,
				expectedTWGS:   "1.03",
				expectedTW2002: "3.13",
				description:    "Pascal: FTWGSType := 1; FTWGSVer := '1.03'; FTW2002Ver := '3.13'",
			},
			{
				name:           "TW2002 with copyright line",
				input:          "Trade Wars 2002 Game Server v1.03                          Copyright (C) 1998",
				expectedType:   1,
				expectedTWGS:   "1.03",
				expectedTW2002: "3.13",
				description:    "Full copyright line should still trigger TW2002 detection",
			},
			{
				name:           "TW2002 Epic Interactive",
				input:          "Trade Wars 2002 Game Server - Epic Interactive Strategy",
				expectedType:   1,
				expectedTWGS:   "1.03",
				expectedTW2002: "3.13",
				description:    "Any line starting with 'Trade Wars 2002 Game' should trigger TW2002 detection",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state
				parser.twgsType = 0
				parser.twgsVer = ""
				parser.tw2002Ver = ""
				
				// Process the version detection line
				parser.ProcessString(tc.input + "\r")
				
				// Verify version detection results
				if parser.GetTWGSType() != tc.expectedType {
					t.Errorf("Expected TWGS type %d, got %d", tc.expectedType, parser.GetTWGSType())
				}
				
				if parser.GetTWGSVersion() != tc.expectedTWGS {
					t.Errorf("Expected TWGS version '%s', got '%s'", tc.expectedTWGS, parser.GetTWGSVersion())
				}
				
				if parser.GetTW2002Version() != tc.expectedTW2002 {
					t.Errorf("Expected TW2002 version '%s', got '%s'", tc.expectedTW2002, parser.GetTW2002Version())
				}
				
				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("No Version Detection", func(t *testing.T) {
		// Test that other lines don't trigger version detection
		nonVersionLines := []string{
			"Welcome to the game",
			"TradeWars - but not the game server line", // Doesn't start with exact pattern
			"Trade Wars but not 2002 Game",              // Doesn't match exact pattern
			"Some other server announcement",
			"Command [TL=30] (? for help): ",
		}

		for _, line := range nonVersionLines {
			// Reset parser state
			parser.twgsType = 0
			parser.twgsVer = ""
			parser.tw2002Ver = ""
			
			// Process the line
			parser.ProcessString(line + "\r")
			
			// Verify no version detection occurred
			if parser.GetTWGSType() != 0 {
				t.Errorf("Line '%s' incorrectly triggered version detection (type %d)", line, parser.GetTWGSType())
			}
			
			if parser.GetTWGSVersion() != "" {
				t.Errorf("Line '%s' incorrectly set TWGS version '%s'", line, parser.GetTWGSVersion())
			}
			
			if parser.GetTW2002Version() != "" {
				t.Errorf("Line '%s' incorrectly set TW2002 version '%s'", line, parser.GetTW2002Version())
			}
		}
		
		t.Log("✓ Non-version lines correctly ignored")
	})

	t.Run("Version Persistence", func(t *testing.T) {
		// Test that version information persists across multiple lines
		
		// First detect a TWGS server
		parser.ProcessString("TradeWars Game Server v2.20b\r")
		
		// Verify detection
		if parser.GetTWGSType() != 2 {
			t.Fatalf("Expected TWGS type 2, got %d", parser.GetTWGSType())
		}
		
		// Process other game lines
		parser.ProcessString("Welcome to the game\r")
		parser.ProcessString("Command [TL=30] (? for help): \r")
		parser.ProcessString("Sector  : 1 in Sol\r")
		
		// Verify version information persists
		if parser.GetTWGSType() != 2 {
			t.Errorf("TWGS type should persist, expected 2, got %d", parser.GetTWGSType())
		}
		
		if parser.GetTWGSVersion() != "2.20b" {
			t.Errorf("TWGS version should persist, expected '2.20b', got '%s'", parser.GetTWGSVersion())
		}
		
		if parser.GetTW2002Version() != "3.34" {
			t.Errorf("TW2002 version should persist, expected '3.34', got '%s'", parser.GetTW2002Version())
		}
		
		t.Log("✓ Version information persists across game session")
	})

	t.Run("Server Type Switching", func(t *testing.T) {
		// Test switching between different server types
		
		// Start with TWGS
		parser.ProcessString("TradeWars Game Server v2.20b\r")
		if parser.GetTWGSType() != 2 {
			t.Fatalf("Expected TWGS type 2, got %d", parser.GetTWGSType())
		}
		
		// Switch to TW2002
		parser.ProcessString("Trade Wars 2002 Game Server v1.03\r")
		
		// Verify the switch
		if parser.GetTWGSType() != 1 {
			t.Errorf("Expected switch to TW2002 (type 1), got %d", parser.GetTWGSType())
		}
		
		if parser.GetTWGSVersion() != "1.03" {
			t.Errorf("Expected TWGS version '1.03', got '%s'", parser.GetTWGSVersion())
		}
		
		if parser.GetTW2002Version() != "3.13" {
			t.Errorf("Expected TW2002 version '3.13', got '%s'", parser.GetTW2002Version())
		}
		
		t.Log("✓ Server type switching works correctly")
	})
}

func TestVersionDetectionIntegration(t *testing.T) {
	// Test version detection in real game session contexts
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParserWithAPI(db, nil)

	t.Run("Game Session with Version Detection", func(t *testing.T) {
		// Simulate a real game session with version detection
		sessionLines := []string{
			"TradeWars Game Server v2.20b                   Copyright (C) 1998-2024",
			"www.tradewars.com                               Epic Interactive Strategy",
			"",
			"Welcome to TradeWars 2002!",
			"",
			"Command [TL=30] (? for help): ",
		}

		for i, line := range sessionLines {
			parser.ProcessString(line + "\r")
			t.Logf("Processed line %d: %s", i+1, line)
		}

		// Verify version was detected from the first line
		if parser.GetTWGSType() != 2 {
			t.Errorf("Expected TWGS type 2, got %d", parser.GetTWGSType())
		}

		if parser.GetTWGSVersion() != "2.20b" {
			t.Errorf("Expected TWGS version '2.20b', got '%s'", parser.GetTWGSVersion())
		}

		if parser.GetTW2002Version() != "3.34" {
			t.Errorf("Expected TW2002 version '3.34', got '%s'", parser.GetTW2002Version())
		}

		t.Log("✓ Game session with TWGS version detection completed successfully")
	})

	t.Run("TW2002 Game Session", func(t *testing.T) {
		// Reset parser for TW2002 session
		parser.twgsType = 0
		parser.twgsVer = ""
		parser.tw2002Ver = ""

		// Simulate TW2002 game session
		sessionLines := []string{
			"Trade Wars 2002 Game Server v1.03                          Copyright (C) 1998",
			"www.tradewars.com                                   Epic Interactive Strategy",
			"",
			"Welcome to your adventure!",
			"",
			"Command [TL=30] (? for help): ",
		}

		for i, line := range sessionLines {
			parser.ProcessString(line + "\r")
			t.Logf("Processed line %d: %s", i+1, line)
		}

		// Verify TW2002 version was detected
		if parser.GetTWGSType() != 1 {
			t.Errorf("Expected TW2002 type 1, got %d", parser.GetTWGSType())
		}

		if parser.GetTWGSVersion() != "1.03" {
			t.Errorf("Expected TWGS version '1.03', got '%s'", parser.GetTWGSVersion())
		}

		if parser.GetTW2002Version() != "3.13" {
			t.Errorf("Expected TW2002 version '3.13', got '%s'", parser.GetTW2002Version())
		}

		t.Log("✓ TW2002 game session version detection completed successfully")
	})
}

func TestVersionDetectionEdgeCases(t *testing.T) {
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParserWithAPI(db, nil)

	t.Run("Case Sensitivity", func(t *testing.T) {
		// Test that detection is case-sensitive (as per Pascal string comparison)
		caseSensitiveTests := []string{
			"tradewars game server", // lowercase - should not match
			"TRADEWARS GAME SERVER", // uppercase - should not match
			"TradeWars game server", // mixed case - should not match (game is lowercase)
		}

		for _, line := range caseSensitiveTests {
			parser.twgsType = 0
			parser.ProcessString(line + "\r")
			
			if parser.GetTWGSType() != 0 {
				t.Errorf("Case-sensitive test failed: '%s' should not trigger detection", line)
			}
		}

		t.Log("✓ Case sensitivity working correctly")
	})

	t.Run("Partial Matches", func(t *testing.T) {
		// Test partial matches that should not trigger detection
		partialMatches := []string{
			"Trade Wars Game",          // Missing "2002"
			"TradeWars",               // Too short
			"Wars Game Server",        // Missing "Trade"
			"Trade Wars 2002",         // Missing "Game"
		}

		for _, line := range partialMatches {
			parser.twgsType = 0
			parser.ProcessString(line + "\r")
			
			if parser.GetTWGSType() != 0 {
				t.Errorf("Partial match test failed: '%s' should not trigger detection", line)
			}
		}

		t.Log("✓ Partial matches correctly rejected")
	})

	t.Run("Multiple Detections", func(t *testing.T) {
		// Test multiple version lines in same session
		parser.twgsType = 0
		
		// First detection
		parser.ProcessString("TradeWars Game Server v2.20b\r")
		firstType := parser.GetTWGSType()
		
		// Second detection (should override)
		parser.ProcessString("Trade Wars 2002 Game Server v1.03\r")
		secondType := parser.GetTWGSType()
		
		if firstType != 2 {
			t.Errorf("First detection failed, expected type 2, got %d", firstType)
		}
		
		if secondType != 1 {
			t.Errorf("Second detection failed, expected type 1, got %d", secondType)
		}
		
		t.Log("✓ Multiple detections handled correctly")
	})
}