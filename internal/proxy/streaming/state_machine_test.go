package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

func TestComprehensiveStateMachine(t *testing.T) {
	// Create test database and parser
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(func() database.Database { return db }, nil)

	t.Run("Display State Transitions", func(t *testing.T) {
		// Test state transitions match Pascal FCurrentDisplay logic
		testCases := []struct {
			name            string
			input           string
			expectedDisplay DisplayType
			description     string
		}{
			{
				name:            "Sector detection",
				input:           "Sector  : 1234 in Sol",
				expectedDisplay: DisplaySector,
				description:     "Pascal: FCurrentDisplay := dSector",
			},
			{
				name:            "Density scanner start",
				input:           "                          Relative Density Scan",
				expectedDisplay: DisplayDensity,
				description:     "Pascal: FCurrentDisplay := dDensity",
			},
			{
				name:            "CIM download start",
				input:           ": Starting download...",
				expectedDisplay: DisplayCIM,
				description:     "Pascal: FCurrentDisplay := dCIM",
			},
			{
				name:            "Normal port docking",
				input:           "Docking...",
				expectedDisplay: DisplayPort,
				description:     "Pascal: FCurrentDisplay := dPort",
			},
			{
				name:            "Computer port report",
				input:           "What sector is the port in? [1234] 5678",
				expectedDisplay: DisplayPortCR,
				description:     "Pascal: FCurrentDisplay := dPortCR",
			},
			{
				name:            "Fighter scan detection",
				input:           "                 Deployed  Fighter  Scan",
				expectedDisplay: DisplayFigScan,
				description:     "Pascal: FCurrentDisplay := dFigScan",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset parser state
				parser.currentDisplay = DisplayNone
				parser.sectorPosition = SectorPosNormal

				// Process input
				parser.ProcessString(tc.input + "\r")

				// Verify state transition
				if parser.currentDisplay != tc.expectedDisplay {
					t.Errorf("Expected display %d, got %d for input: %s", tc.expectedDisplay, parser.currentDisplay, tc.input)
				}

				t.Logf("✓ %s: %s", tc.name, tc.description)
			})
		}
	})

	t.Run("Sector Position State Machine", func(t *testing.T) {
		// Test FSectorPosition logic matching Pascal implementation
		parser.currentDisplay = DisplaySector
		parser.sectorPosition = SectorPosNormal

		testCases := []struct {
			input            string
			expectedPosition SectorPosition
			description      string
		}{
			{
				input:            "Ports   : Terra Port, Class 1 Port BSS",
				expectedPosition: SectorPosPorts,
				description:      "Pascal: FSectorPosition := spPorts",
			},
			{
				input:            "Planets : Terra (L)",
				expectedPosition: SectorPosPlanets,
				description:      "Pascal: FSectorPosition := spPlanets",
			},
			{
				input:            "Traders : Captain Kirk",
				expectedPosition: SectorPosTraders,
				description:      "Pascal: FSectorPosition := spTraders",
			},
			{
				input:            "Ships   : USS Enterprise",
				expectedPosition: SectorPosShips,
				description:      "Pascal: FSectorPosition := spShips",
			},
			{
				input:            "Mines   : 500 Armid Mines",
				expectedPosition: SectorPosMines,
				description:      "Pascal: FSectorPosition := spMines",
			},
			{
				input:            "NavHaz  : 5%",
				expectedPosition: SectorPosNormal,
				description:      "Pascal: FSectorPosition := spNormal (9th char is ':')",
			},
		}

		for _, step := range testCases {
			t.Logf("Processing input: %q", step.input)
			parser.ProcessString(step.input + "\r")
			t.Logf("After processing, sectorPosition = %d, expected = %d", parser.sectorPosition, step.expectedPosition)

			if parser.sectorPosition != step.expectedPosition {
				t.Errorf("Expected sector position %d, got %d for input: %s",
					step.expectedPosition, parser.sectorPosition, step.input)
			} else {
				t.Logf("✓ %s", step.description)
			}
		}
	})

	t.Run("Continuation Line Processing", func(t *testing.T) {
		// Test 8-space continuation line logic (Pascal: Copy(Line, 1, 8) = '        ')
		parser.currentDisplay = DisplaySector

		testCases := []struct {
			sectorPos   SectorPosition
			input       string
			expected    bool
			description string
		}{
			{
				sectorPos:   SectorPosPorts,
				input:       "        Build Time: 24 hours",
				expected:    true,
				description: "Pascal: FSectorPosition = spPorts continuation",
			},
			{
				sectorPos:   SectorPosPlanets,
				input:       "        Additional Planet Name",
				expected:    true,
				description: "Pascal: FSectorPosition = spPlanets continuation",
			},
			{
				sectorPos:   SectorPosTraders,
				input:       "        in USS Defiant (Dreadnought)",
				expected:    true,
				description: "Pascal: FSectorPosition = spTraders continuation",
			},
			{
				sectorPos:   SectorPosShips,
				input:       "        (Constitution Class)",
				expected:    true,
				description: "Pascal: FSectorPosition = spShips continuation",
			},
			{
				sectorPos:   SectorPosMines,
				input:       "        500 Limpet Mines owned by Captain Kirk",
				expected:    true,
				description: "Pascal: FSectorPosition = spMines continuation",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				parser.sectorPosition = tc.sectorPos

				// Process continuation line
				parser.ProcessString(tc.input + "\r")

				// Verify continuation was processed (check that position didn't change to Normal)
				if parser.sectorPosition != tc.sectorPos {
					t.Errorf("Continuation line processing failed - sector position changed from %d to %d",
						tc.sectorPos, parser.sectorPosition)
				} else {
					t.Logf("✓ %s", tc.description)
				}
			})
		}
	})

	t.Run("State Reset Logic", func(t *testing.T) {
		// Test Pascal logic for resetting display states
		testCases := []struct {
			initialDisplay  DisplayType
			input           string
			expectedDisplay DisplayType
			description     string
		}{
			{
				initialDisplay:  DisplaySector,
				input:           "Command [TL=150] (1234) ?",
				expectedDisplay: DisplayNone,
				description:     "Pascal: FCurrentDisplay := dNone after sector completion",
			},
			{
				initialDisplay:  DisplayCIM,
				input:           "Invalid CIM data",
				expectedDisplay: DisplayNone,
				description:     "Pascal: FCurrentDisplay := dNone on CIM error",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				parser.currentDisplay = tc.initialDisplay
				parser.ProcessString(tc.input + "\r")

				if parser.currentDisplay != tc.expectedDisplay {
					t.Errorf("Expected display %d after reset, got %d", tc.expectedDisplay, parser.currentDisplay)
				} else {
					t.Logf("✓ %s", tc.description)
				}
			})
		}
	})

	t.Run("CIM State Machine", func(t *testing.T) {
		// Test CIM-specific state transitions
		parser.currentDisplay = DisplayCIM

		testCases := []struct {
			input           string
			expectedDisplay DisplayType
			description     string
		}{
			{
				input:           "1234 5000 60% 3000 80% 2000 90%",
				expectedDisplay: DisplayPortCIM,
				description:     "Pascal: FCurrentDisplay := dPortCIM (contains %)",
			},
			{
				input:           "1234 5678 9012 3456 7890 1234 4567",
				expectedDisplay: DisplayWarpCIM,
				description:     "Pascal: FCurrentDisplay := dWarpCIM (no %)",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				parser.currentDisplay = DisplayCIM
				parser.ProcessString(tc.input + "\r")

				if parser.currentDisplay != tc.expectedDisplay {
					t.Errorf("Expected display %d, got %d for CIM input: %s",
						tc.expectedDisplay, parser.currentDisplay, tc.input)
				} else {
					t.Logf("✓ %s", tc.description)
				}
			})
		}
	})
}

func TestStateMachineWorkflow(t *testing.T) {
	// Test complete workflows through the state machine
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()

	parser := NewTWXParser(func() database.Database { return db }, nil)

	t.Run("Complete Sector Scan Workflow", func(t *testing.T) {
		// Simulate complete sector scanning session
		testCases := []struct {
			input            string
			expectedDisplay  DisplayType
			expectedPosition SectorPosition
			description      string
		}{
			{
				input:            "Sector  : 1001 in Unknown Space",
				expectedDisplay:  DisplaySector,
				expectedPosition: SectorPosNormal,
				description:      "Start sector scan",
			},
			{
				input:            "Ports   : Trading Post, Class 2 Port SBB",
				expectedDisplay:  DisplaySector,
				expectedPosition: SectorPosPorts,
				description:      "Detect ports section",
			},
			{
				input:            "        Build Time: 48 hours",
				expectedDisplay:  DisplaySector,
				expectedPosition: SectorPosPorts,
				description:      "Process port continuation",
			},
			{
				input:            "Planets : Terra (L)",
				expectedDisplay:  DisplaySector,
				expectedPosition: SectorPosPlanets,
				description:      "Switch to planets section",
			},
			{
				input:            "        Additional Planet",
				expectedDisplay:  DisplaySector,
				expectedPosition: SectorPosPlanets,
				description:      "Process planet continuation",
			},
			{
				input:            "NavHaz  : 5%",
				expectedDisplay:  DisplaySector,
				expectedPosition: SectorPosNormal,
				description:      "Reset to normal on 9th char colon",
			},
			{
				input:            "Warps to Sector(s) :  (1002) - 1003",
				expectedDisplay:  DisplaySector,
				expectedPosition: SectorPosNormal,
				description:      "Parse sector warps (sector still active)",
			},
		}

		for i, step := range testCases {
			parser.ProcessString(step.input + "\r")

			if parser.currentDisplay != step.expectedDisplay {
				t.Errorf("Step %d: Expected display %d, got %d for: %s",
					i+1, step.expectedDisplay, parser.currentDisplay, step.input)
			}

			if parser.sectorPosition != step.expectedPosition {
				t.Errorf("Step %d: Expected position %d, got %d for: %s",
					i+1, step.expectedPosition, parser.sectorPosition, step.input)
			}

			t.Logf("✓ Step %d: %s", i+1, step.description)
		}
	})

	t.Run("Mixed Display Mode Workflow", func(t *testing.T) {
		// Test transitions between different display modes
		mixedWorkflow := []struct {
			input           string
			expectedDisplay DisplayType
			description     string
		}{
			{
				input:           "Sector  : 2001 in Alpha Sector",
				expectedDisplay: DisplaySector,
				description:     "Start with sector",
			},
			{
				input:           "Command [TL=150] (2002) ?",
				expectedDisplay: DisplayNone,
				description:     "Complete sector with command prompt",
			},
			{
				input:           "                          Relative Density Scan",
				expectedDisplay: DisplayDensity,
				description:     "Switch to density scan",
			},
			{
				input:           "Sector 2004 (Test) Density: 1500 NavHaz: 0% Warps: 6 Anomaly: No",
				expectedDisplay: DisplayDensity,
				description:     "Process density data",
			},
			{
				input:           ": Starting CIM download",
				expectedDisplay: DisplayCIM,
				description:     "Switch to CIM mode",
			},
			{
				input:           "2005 3000 70% 2000 80% 1000 90%",
				expectedDisplay: DisplayPortCIM,
				description:     "Process port CIM data",
			},
			{
				input:           "Docking...",
				expectedDisplay: DisplayPort,
				description:     "Switch to port mode",
			},
		}

		for i, step := range mixedWorkflow {
			parser.ProcessString(step.input + "\r")

			if parser.currentDisplay != step.expectedDisplay {
				t.Errorf("Mixed workflow step %d: Expected display %d, got %d for: %s",
					i+1, step.expectedDisplay, parser.currentDisplay, step.input)
			} else {
				t.Logf("✓ Mixed step %d: %s", i+1, step.description)
			}
		}
	})
}
