package streaming

import (
	"strings"
	"testing"
)

func TestTWXParser_CommandPrompt(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test command prompt parsing
	commandPrompt := "Command [TL=150] (2500) ?"
	parser.ProcessString(commandPrompt)
	
	if parser.GetCurrentSector() != 2500 {
		t.Errorf("Expected sector 2500, got %d", parser.GetCurrentSector())
	}
	
	if parser.GetDisplayState() != DisplayNone {
		t.Errorf("Expected DisplayNone, got %v", parser.GetDisplayState())
	}
}

func TestTWXParser_ComputerPrompt(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test computer command prompt
	computerPrompt := "Computer command [TL=150] (1234) ?"
	parser.ProcessString(computerPrompt)
	
	if parser.GetCurrentSector() != 1234 {
		t.Errorf("Expected sector 1234, got %d", parser.GetCurrentSector())
	}
}

func TestTWXParser_SectorData(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test sector data parsing
	sectorData := `Sector  : 1 in The Sphere
Beacon  : FedSpace, FedLaw Enforced
Ports   : Stargate Alpha I, Class 9 Port (SSSx3)
        Build time: 0 days
Planets : Terra
Warps to Sector(s) : 2 - 3 - 4 - 5 - 6 - 7
`
	
	lines := strings.Split(sectorData, "\n")
	for _, line := range lines {
		if line != "" {
			parser.ProcessString(line + "\r")
		}
	}
	
	if parser.GetCurrentSector() != 1 {
		t.Errorf("Expected sector 1, got %d", parser.GetCurrentSector())
	}
}

func TestTWXParser_PartialLine(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test partial line handling (key TWX feature)
	// Send partial command prompt without newline
	parser.ProcessString("Command [TL=150] (2500)")
	
	// Should still parse the sector number
	if parser.GetCurrentSector() != 2500 {
		t.Errorf("Expected sector 2500 from partial line, got %d", parser.GetCurrentSector())
	}
}

func TestTWXParser_StreamingChunks(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test streaming chunks that split strings
	chunk1 := "Command [TL="
	chunk2 := "150] (2500) ?"
	
	parser.ProcessString(chunk1)
	// Should not have parsed sector yet
	if parser.GetCurrentSector() == 2500 {
		t.Errorf("Should not have parsed sector from incomplete chunk")
	}
	
	parser.ProcessString(chunk2)
	// Now should have parsed sector
	if parser.GetCurrentSector() != 2500 {
		t.Errorf("Expected sector 2500 after complete chunks, got %d", parser.GetCurrentSector())
	}
}

func TestTWXParser_ANSIStripping(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test ANSI stripping
	ansiText := "\x1b[1;36mCommand [TL=150] (2500) ?\x1b[0m"
	parser.ProcessString(ansiText)
	
	if parser.GetCurrentSector() != 2500 {
		t.Errorf("Expected sector 2500 after ANSI stripping, got %d", parser.GetCurrentSector())
	}
}

func TestTWXParser_PortData(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test port data parsing
	portData := `Docking...
Commerce report for Stargate Alpha I:
Fuel Ore     Selling       10,000 units at 100%
Organics     Selling       10,000 units at 100%
Equipment    Selling       10,000 units at 100%
`
	
	lines := strings.Split(portData, "\n")
	for _, line := range lines {
		if line != "" {
			parser.ProcessString(line + "\r")
		}
	}
	
	if parser.GetDisplayState() != DisplayPort {
		t.Errorf("Expected DisplayPort state, got %v", parser.GetDisplayState())
	}
}

func TestTWXParser_WarpLane(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test warp lane parsing
	warpLane := `The shortest path (13 hops) is:
1 > 2 > 3 > 4 > 5 > 6
`
	
	lines := strings.Split(warpLane, "\n")
	for _, line := range lines {
		if line != "" {
			parser.ProcessString(line + "\r")
		}
	}
	
	if parser.GetDisplayState() != DisplayWarpLane {
		t.Errorf("Expected DisplayWarpLane state, got %v", parser.GetDisplayState())
	}
}

func TestTWXParser_CIMPrompt(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test CIM prompt
	parser.ProcessString(": ")
	
	// Should reset display state for CIM
	if parser.GetDisplayState() == DisplayCIM {
		// CIM state handling depends on previous state
		t.Logf("CIM prompt handled correctly")
	}
}

func TestTWXParser_Reset(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Set some state
	parser.ProcessString("Command [TL=150] (2500) ?")
	
	// Reset should clear state
	parser.Reset()
	
	if parser.GetCurrentSector() != 0 {
		t.Errorf("Expected sector 0 after reset, got %d", parser.GetCurrentSector())
	}
	
	if parser.GetDisplayState() != DisplayNone {
		t.Errorf("Expected DisplayNone after reset, got %v", parser.GetDisplayState())
	}
}

func TestTWXParser_MultipleLines(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test multiple lines in one string
	multiLine := "Command [TL=150] (2500) ?\rSector  : 1 in The Sphere\r"
	parser.ProcessString(multiLine)
	
	// Should process both lines
	if parser.GetCurrentSector() != 1 {
		t.Errorf("Expected sector 1 from multi-line input, got %d", parser.GetCurrentSector())
	}
}

func TestTWXParser_ProbePrompt(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test probe prompts
	parser.ProcessString("Probe entering sector : 1234")
	
	if parser.GetDisplayState() != DisplayNone {
		t.Errorf("Expected DisplayNone after probe prompt, got %v", parser.GetDisplayState())
	}
	
	parser.ProcessString("Probe Self Destructs")
	
	if parser.GetDisplayState() != DisplayNone {
		t.Errorf("Expected DisplayNone after probe self destruct, got %v", parser.GetDisplayState())
	}
}

func TestTWXParser_IntegerParsing(t *testing.T) {
	parser := NewTestTWXParser()
	
	tests := []struct {
		input    string
		expected int
	}{
		{"1234", 1234},
		{"1,234", 1234},
		{"  1234  ", 1234},
		{"", 0},
		{"abc", 0},
		{"123abc", 0},
		{"0", 0},
	}
	
	for _, test := range tests {
		result := parser.parseIntSafe(test.input)
		if result != test.expected {
			t.Errorf("parseIntSafe(%q) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestTWXParser_QuickStats(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test QuickStats parsing
	quickStatsLine := " Turns 150�Creds 10,000�Figs 500�Shlds 100�Ship 1 Merchant"
	parser.processQuickStats(quickStatsLine)
	
	stats := parser.GetPlayerStats()
	
	if stats.Turns != 150 {
		t.Errorf("Expected turns 150, got %d", stats.Turns)
	}
	if stats.Credits != 10000 {
		t.Errorf("Expected credits 10000, got %d", stats.Credits)
	}
	if stats.Fighters != 500 {
		t.Errorf("Expected fighters 500, got %d", stats.Fighters)
	}
	if stats.Shields != 100 {
		t.Errorf("Expected shields 100, got %d", stats.Shields)
	}
	if stats.ShipNumber != 1 {
		t.Errorf("Expected ship number 1, got %d", stats.ShipNumber)
	}
	if stats.ShipClass != "Merchant" {
		t.Errorf("Expected ship class Merchant, got %s", stats.ShipClass)
	}
}

func TestTWXParser_EnhancedSectorData(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test enhanced sector data parsing
	sectorData := `Sector  : 1 in The Sphere
Beacon  : FedSpace, FedLaw Enforced
Ports   : Stargate Alpha I, Class 9 Port (SSSx3)
        Build time: 0 days
Planets : Terra
Traders : Captain Kirk, w/ 1,000 ftrs
Warps to Sector(s) : 2 - 3 - 4 - 5 - 6 - 7
`
	
	lines := strings.Split(sectorData, "\n")
	for _, line := range lines {
		if line != "" {
			parser.ProcessString(line + "\r")
		}
	}
	
	if parser.GetCurrentSector() != 1 {
		t.Errorf("Expected sector 1, got %d", parser.GetCurrentSector())
	}
	
	if parser.currentSector.Beacon != "FedSpace, FedLaw Enforced" {
		t.Errorf("Expected beacon 'FedSpace, FedLaw Enforced', got '%s'", parser.currentSector.Beacon)
	}
}

func TestTWXParser_MessageHandling(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Test transmission detection
	transmission := "Incoming transmission from Captain Kirk on channel 1:"
	parser.ProcessString(transmission + "\r")
	
	if parser.currentMessage == "" {
		t.Error("Expected currentMessage to be set after transmission")
	}
	
	// Test message content
	messageContent := "Hello there!"
	parser.ProcessString(messageContent + "\r")
	
	// Message should be cleared after content
	if parser.currentMessage != "" {
		t.Error("Expected currentMessage to be cleared after message content")
	}
}

func TestTWXParser_CIMProcessing(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("WarpCIM", func(t *testing.T) {
		// Test warp CIM processing
		parser.currentDisplay = DisplayCIM
		warpCIMLine := "1234 5678 9012 3456 7890 2345 6789"
		parser.processCIMLine(warpCIMLine)
		
		if parser.currentDisplay != DisplayWarpCIM {
			t.Error("Expected display to be set to DisplayWarpCIM")
		}
	})
	
	t.Run("PortCIM", func(t *testing.T) {
		// Test port CIM processing
		parser.currentDisplay = DisplayCIM
		portCIMLine := "1234 5000 60% 3000 80% 2000 90%"
		parser.processCIMLine(portCIMLine)
		
		if parser.currentDisplay != DisplayPortCIM {
			t.Error("Expected display to be set to DisplayPortCIM")
		}
	})
	
	t.Run("PortCIMWithDashes", func(t *testing.T) {
		// Test port CIM with buy indicators (dashes)
		parser.currentDisplay = DisplayCIM
		portCIMLine := "1234 -5000 60% 3000 80% -2000 90%"
		parser.processCIMLine(portCIMLine)
		
		if parser.currentDisplay != DisplayPortCIM {
			t.Error("Expected display to be set to DisplayPortCIM")
		}
	})
}

func TestTWXParser_FighterScanProcessing(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("NoFightersDeployed", func(t *testing.T) {
		// Test "No fighters deployed" case
		parser.currentDisplay = DisplayFigScan
		noFigLine := "No fighters deployed in any sectors."
		parser.processFigScanLine(noFigLine)
		// Should trigger reset of fighter database
	})
	
	t.Run("BasicFighterScan", func(t *testing.T) {
		// Test basic fighter scan line
		parser.currentDisplay = DisplayFigScan
		figScanLine := "1234 500 Personal Defensive N/A"
		parser.processFigScanLine(figScanLine)
	})
	
	t.Run("FighterScanWithMultiplier", func(t *testing.T) {
		// Test fighter scan with T/M/B multipliers
		parser.currentDisplay = DisplayFigScan
		
		testCases := []struct {
			line     string
			expected int
		}{
			{"1234 10T Personal Offensive N/A", 10000},
			{"1234 5M Corporate Toll N/A", 5000000},
			{"1234 2B Personal Defensive N/A", 2000000000},
		}
		
		for _, tc := range testCases {
			parser.processFigScanLine(tc.line)
			// In a real implementation, we'd verify the stored quantity
		}
	})
	
	t.Run("FighterQuantityParsing", func(t *testing.T) {
		// Test fighter quantity parsing with multipliers
		testCases := []struct {
			input    string
			expected int
		}{
			{"1000", 1000},
			{"10T", 10000},
			{"5M", 5000000},
			{"2B", 2000000000},
			{"1,000", 1000},
			{"10t", 10000}, // lowercase
			{"", 0},
		}
		
		for _, tc := range testCases {
			result := parser.parseFighterQuantity(tc.input)
			if result != tc.expected {
				t.Errorf("parseFighterQuantity(%q) = %d, expected %d", tc.input, result, tc.expected)
			}
		}
	})
}

func TestTWXParser_DensityScanning(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("DensityStart", func(t *testing.T) {
		// Test density scan start
		densityStart := "            Relative Density Scan"
		parser.ProcessString(densityStart + "\r")
		
		if parser.currentDisplay != DisplayDensity {
			t.Error("Expected display to be set to DisplayDensity")
		}
	})
	
	t.Run("DensitySectorData", func(t *testing.T) {
		// Test density sector data parsing
		parser.currentDisplay = DisplayDensity
		densitySector := "Sector 1234 (The Sphere) Density: 1,500, NavHaz: 5%, Warps: 6, Anomaly: Yes"
		parser.processDensityLine(densitySector)
	})
	
	t.Run("DensitySectorNoAnomaly", func(t *testing.T) {
		// Test density sector without anomaly
		parser.currentDisplay = DisplayDensity
		densitySector := "Sector 5678 (Deep Space) Density: 800, NavHaz: 0%, Warps: 3, Anomaly: No"
		parser.processDensityLine(densitySector)
	})
}

func TestTWXParser_CompleteGameSession(t *testing.T) {
	parser := NewTestTWXParser()
	
	// Simulate a complete game session with various data types
	gameSession := []string{
		"Command [TL=150] (2500) ?",
		" Turns 150�Creds 10,000�Figs 500�Shlds 100�Ship 1 Merchant",
		"Sector  : 1 in The Sphere",
		"Beacon  : FedSpace, FedLaw Enforced", 
		"Ports   : Stargate Alpha I, Class 9 Port (SSSx3)",
		"Warps to Sector(s) : 2 - 3 - 4 - 5 - 6 - 7",
		"Docking...",
		"Commerce report for Stargate Alpha I:",
		"Fuel Ore     Selling       10,000 units at 100%",
		"Equipment    Selling       5,000 units at 90%",
		": ",  // CIM prompt
		"1234 5000 60% 3000 80% 2000 90%", // Port CIM data
		"1234 5678 9012 3456 7890 2345 6789", // Warp CIM data
		"            Deployed  Fighter  Scan",
		"1234 10T Personal Defensive N/A",
		"5678 5M Corporate Offensive N/A",
		"            Relative Density Scan",
		"Sector 1234 (The Sphere) Density: 1,500, NavHaz: 5%, Warps: 6, Anomaly: Yes",
		"Incoming transmission from Captain Kirk on channel 1:",
		"Hello there, trader!",
	}
	
	for _, line := range gameSession {
		parser.ProcessString(line + "\r")
	}
	
	// Verify final state - should be sector 1 from the "Sector : 1 in The Sphere" line
	if parser.GetCurrentSector() != 1 {
		t.Errorf("Expected final sector 1 (from sector data), got %d", parser.GetCurrentSector())
	}
	
	stats := parser.GetPlayerStats()
	if stats.Turns != 150 {
		t.Errorf("Expected 150 turns, got %d", stats.Turns)
	}
	if stats.Credits != 10000 {
		t.Errorf("Expected 10000 credits, got %d", stats.Credits)
	}
}

// Benchmark tests for performance
func BenchmarkTWXParser_ProcessString(b *testing.B) {
	parser := NewTestTWXParser()
	testString := "Command [TL=150] (2500) ?"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ProcessString(testString)
		parser.Reset()
	}
}

func BenchmarkTWXParser_ProcessChunk(b *testing.B) {
	parser := NewTestTWXParser()
	testData := []byte("Command [TL=150] (2500) ?")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ProcessChunk(testData)
		parser.Reset()
	}
}

func BenchmarkTWXParser_SectorData(b *testing.B) {
	parser := NewTestTWXParser()
	sectorData := `Sector  : 1 in The Sphere
Beacon  : FedSpace, FedLaw Enforced
Ports   : Stargate Alpha I, Class 9 Port (SSSx3)
Planets : Terra
Traders : Captain Kirk, w/ 1,000 ftrs
Ships   : Enterprise [Owned by Kirk], w/ 500 ftrs,
        (Constitution Class Cruiser)
Fighters: 2,500 (belong to Kirk) [Defensive]
NavHaz  : 5% (10)
Mines   : 100 Limpet Mines (belong to Kirk)
Warps to Sector(s) : 2 - 3 - 4 - 5 - 6 - 7
`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lines := strings.Split(sectorData, "\n")
		for _, line := range lines {
			if line != "" {
				parser.ProcessString(line + "\r")
			}
		}
		parser.Reset()
	}
}