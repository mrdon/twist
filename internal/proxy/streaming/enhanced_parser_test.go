package streaming

import (
	"testing"
)

func TestEnhancedSectorParsing(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("DetailedShipParsing", func(t *testing.T) {
		// Test detailed ship parsing
		shipLine := "Ships   : Enterprise [Owned by Kirk], w/ 500 ftrs,"
		continuationLine := "        (Constitution Class Cruiser)"
		
		parser.parseSectorShips(shipLine)
		parser.handleShipContinuation(continuationLine)
		
		if len(parser.currentShips) != 1 {
			t.Errorf("Expected 1 ship, got %d", len(parser.currentShips))
		}
		
		ship := parser.currentShips[0]
		if ship.Name != "Enterprise" {
			t.Errorf("Expected ship name 'Enterprise', got '%s'", ship.Name)
		}
		if ship.Owner != "Kirk" {
			t.Errorf("Expected owner 'Kirk', got '%s'", ship.Owner)
		}
		if ship.Fighters != 500 {
			t.Errorf("Expected 500 fighters, got %d", ship.Fighters)
		}
		if ship.ShipType != "Constitution Class Cruiser" {
			t.Errorf("Expected ship type 'Constitution Class Cruiser', got '%s'", ship.ShipType)
		}
	})
	
	t.Run("DetailedTraderParsing", func(t *testing.T) {
		// Test detailed trader parsing - TWX handles one trader per line
		traderLine := "Traders : Captain Kirk, w/ 1,000 ftrs"
		
		parser.parseSectorTraders(traderLine)
		
		// Finalize the trader to add it to currentTraders list
		parser.finalizeCurrentTrader()
		
		if len(parser.currentTraders) != 1 {
			t.Errorf("Expected 1 trader, got %d", len(parser.currentTraders))
		}
		
		trader1 := parser.currentTraders[0]
		if trader1.Name != "Captain Kirk" {
			t.Errorf("Expected trader name 'Captain Kirk', got '%s'", trader1.Name)
		}
		if trader1.Fighters != 1000 {
			t.Errorf("Expected 1000 fighters, got %d", trader1.Fighters)
		}
		
		// Test continuation line for additional trader
		continuationLine := "        Spock, w/ 2,500 ftrs"
		parser.handleSectorContinuation(continuationLine)
		
		// Finalize the second trader to add it to currentTraders list
		parser.finalizeCurrentTrader()
		
		if len(parser.currentTraders) != 2 {
			t.Errorf("Expected 2 traders after continuation, got %d", len(parser.currentTraders))
		}
		
		trader2 := parser.currentTraders[1]
		if trader2.Name != "Spock" {
			t.Errorf("Expected trader name 'Spock', got '%s'", trader2.Name)
		}
		if trader2.Fighters != 2500 {
			t.Errorf("Expected 2500 fighters, got %d", trader2.Fighters)
		}
	})
	
	t.Run("DetailedPlanetParsing", func(t *testing.T) {
		// Test detailed planet parsing
		planetLine := "Planets : Terra [Owned by Federation], Stardock"
		
		parser.parseSectorPlanets(planetLine)
		
		if len(parser.currentPlanets) != 2 {
			t.Errorf("Expected 2 planets, got %d", len(parser.currentPlanets))
		}
		
		planet1 := parser.currentPlanets[0]
		if planet1.Name != "Terra" {
			t.Errorf("Expected planet name 'Terra', got '%s'", planet1.Name)
		}
		if planet1.Owner != "Federation" {
			t.Errorf("Expected owner 'Federation', got '%s'", planet1.Owner)
		}
		
		planet2 := parser.currentPlanets[1]
		if planet2.Name != "Stardock" {
			t.Errorf("Expected planet name 'Stardock', got '%s'", planet2.Name)
		}
		if !planet2.Stardock {
			t.Error("Expected Stardock flag to be true")
		}
	})
	
	t.Run("DetailedMineParsing", func(t *testing.T) {
		// Test detailed mine parsing
		mineLine := "Mines   : 100 Limpet Mines, 50 Armid Mines (belong to Kirk)"
		
		parser.parseSectorMines(mineLine)
		
		if len(parser.currentMines) != 2 {
			t.Errorf("Expected 2 mine types, got %d", len(parser.currentMines))
		}
		
		mine1 := parser.currentMines[0]
		if mine1.Type != "Limpet" {
			t.Errorf("Expected mine type 'Limpet', got '%s'", mine1.Type)
		}
		if mine1.Quantity != 100 {
			t.Errorf("Expected 100 mines, got %d", mine1.Quantity)
		}
		if mine1.Owner != "Kirk" {
			t.Errorf("Expected owner 'Kirk', got '%s'", mine1.Owner)
		}
		
		mine2 := parser.currentMines[1]
		if mine2.Type != "Armid" {
			t.Errorf("Expected mine type 'Armid', got '%s'", mine2.Type)
		}
		if mine2.Quantity != 50 {
			t.Errorf("Expected 50 mines, got %d", mine2.Quantity)
		}
	})
}

func TestEnhancedProductParsing(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("StandardProductFormat", func(t *testing.T) {
		// Test standard product format parsing
		productLine := "Fuel Ore     Selling       10,000 units at 100%"
		
		parser.parseProductLine(productLine)
		
		if len(parser.currentProducts) != 1 {
			t.Errorf("Expected 1 product, got %d", len(parser.currentProducts))
		}
		
		product := parser.currentProducts[0]
		if product.Type != ProductFuelOre {
			t.Errorf("Expected ProductFuelOre, got %v", product.Type)
		}
		if product.Quantity != 10000 {
			t.Errorf("Expected quantity 10000, got %d", product.Quantity)
		}
		if product.Percent != 100 {
			t.Errorf("Expected percent 100, got %d", product.Percent)
		}
		if !product.Selling {
			t.Error("Expected Selling to be true")
		}
		if product.Status != "Selling" {
			t.Errorf("Expected status 'Selling', got '%s'", product.Status)
		}
	})
	
	t.Run("AlternateProductFormat", func(t *testing.T) {
		// Test alternate product format parsing
		productLine := "Equipment    : 1000 buying at 15"
		
		parser.ClearProductData()
		parser.parseProductLine(productLine)
		
		if len(parser.currentProducts) != 1 {
			t.Errorf("Expected 1 product, got %d", len(parser.currentProducts))
		}
		
		product := parser.currentProducts[0]
		if product.Type != ProductEquipment {
			t.Errorf("Expected ProductEquipment, got %v", product.Type)
		}
		if product.Quantity != 1000 {
			t.Errorf("Expected quantity 1000, got %d", product.Quantity)
		}
		if product.Percent != 15 {
			t.Errorf("Expected percent 15, got %d", product.Percent)
		}
		if !product.Buying {
			t.Error("Expected Buying to be true")
		}
	})
	
	t.Run("ProductTypeDetection", func(t *testing.T) {
		// Test product type detection
		testCases := []struct {
			line     string
			expected ProductType
		}{
			{"Fuel Ore     Selling       5,000 units at 95%", ProductFuelOre},
			{"Organics     Buying         2,000 units at 45%", ProductOrganics},
			{"Equipment    Selling       1,500 units at 85%", ProductEquipment},
		}
		
		for _, test := range testCases {
			productType := parser.getProductTypeFromLine(test.line)
			if ProductType(productType) != test.expected {
				t.Errorf("Line '%s': expected %v, got %v", test.line, test.expected, ProductType(productType))
			}
		}
	})
}

func TestEnhancedMessageHistory(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("MessageHistoryBasic", func(t *testing.T) {
		// Test basic message history functionality
		parser.addToHistory(MessageRadio, "Hello there!", "Kirk", 1)
		parser.addToHistory(MessageFighter, "Fighters deployed", "Computer", 0)
		
		history := parser.GetMessageHistory()
		if len(history) != 2 {
			t.Errorf("Expected 2 messages in history, got %d", len(history))
		}
		
		if history[0].Content != "Hello there!" {
			t.Errorf("Expected first message content 'Hello there!', got '%s'", history[0].Content)
		}
		if history[0].Sender != "Kirk" {
			t.Errorf("Expected sender 'Kirk', got '%s'", history[0].Sender)
		}
		if history[0].Channel != 1 {
			t.Errorf("Expected channel 1, got %d", history[0].Channel)
		}
	})
	
	t.Run("MessageHistoryByType", func(t *testing.T) {
		// Test filtering by message type
		parser.ClearHistory()
		parser.addToHistory(MessageRadio, "Radio message", "Kirk", 1)
		parser.addToHistory(MessageFighter, "Fighter message", "Computer", 0)
		parser.addToHistory(MessageRadio, "Another radio", "Spock", 2)
		
		radioMessages := parser.GetMessageHistoryByType(MessageRadio)
		if len(radioMessages) != 2 {
			t.Errorf("Expected 2 radio messages, got %d", len(radioMessages))
		}
		
		fighterMessages := parser.GetMessageHistoryByType(MessageFighter)
		if len(fighterMessages) != 1 {
			t.Errorf("Expected 1 fighter message, got %d", len(fighterMessages))
		}
	})
	
	t.Run("MessageHistoryLimits", func(t *testing.T) {
		// Test history size limits
		parser.ClearHistory()
		parser.SetHistorySize(5)
		
		// Add more messages than the limit
		for i := 0; i < 10; i++ {
			parser.addToHistory(MessageGeneral, "Test message", "Test", 0)
		}
		
		history := parser.GetMessageHistory()
		if len(history) > 5 {
			t.Errorf("Expected history to be limited to 5 messages, got %d", len(history))
		}
	})
	
	t.Run("RecentMessages", func(t *testing.T) {
		// Test recent messages functionality
		parser.ClearHistory()
		for i := 0; i < 5; i++ {
			parser.addToHistory(MessageGeneral, "Message", "Test", 0)
		}
		
		recent := parser.GetRecentMessages(3)
		if len(recent) != 3 {
			t.Errorf("Expected 3 recent messages, got %d", len(recent))
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("ParameterExtraction", func(t *testing.T) {
		// Test parameter extraction functions
		line := "Fuel Ore Selling 10,000 units at 100%"
		
		param1 := parser.getParameter(line, 1)
		if param1 != "Fuel" {
			t.Errorf("Expected parameter 1 to be 'Fuel', got '%s'", param1)
		}
		
		param3 := parser.getParameter(line, 3)
		if param3 != "Selling" {
			t.Errorf("Expected parameter 3 to be 'Selling', got '%s'", param3)
		}
		
		// Test parameter position
		pos := parser.getParameterPos(line, 3)
		if pos < 0 {
			t.Error("Expected valid position for parameter 3")
		}
	})
	
	t.Run("NumberParsing", func(t *testing.T) {
		// Test various number parsing scenarios
		testCases := []struct {
			input    string
			expected int
		}{
			{"1234", 1234},
			{"1,234", 1234},
			{"  1234  ", 1234},
			{"", 0},
			{"abc", 0},
			{"10,000", 10000},
		}
		
		for _, test := range testCases {
			result := parser.parseIntSafe(test.input)
			if result != test.expected {
				t.Errorf("parseIntSafe('%s') = %d, expected %d", test.input, result, test.expected)
			}
		}
	})
	
	t.Run("BooleanParsing", func(t *testing.T) {
		// Test boolean parsing
		testCases := []struct {
			input    string
			expected bool
		}{
			{"yes", true},
			{"true", true},
			{"1", true},
			{"no", false},
			{"false", false},
			{"0", false},
			{"", false},
		}
		
		for _, test := range testCases {
			result := parser.parseBoolFromString(test.input)
			if result != test.expected {
				t.Errorf("parseBoolFromString('%s') = %t, expected %t", test.input, result, test.expected)
			}
		}
	})
	
	t.Run("PortClassParsing", func(t *testing.T) {
		// Test port class parsing from trade patterns
		testCases := []struct {
			pattern  string
			expected int
		}{
			{"BBS", 1},
			{"BSB", 2},
			{"SBB", 3},
			{"SSB", 4},
			{"SBS", 5},
			{"BSS", 6},
			{"SSS", 7},
			{"BBB", 8},
			{"???", 9},
			{"unknown", 0},
		}
		
		for _, test := range testCases {
			result := parser.classFromTradePattern(test.pattern)
			if result != test.expected {
				t.Errorf("classFromTradePattern('%s') = %d, expected %d", test.pattern, result, test.expected)
			}
		}
	})
}

func TestMessageTransmissionParsing(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("TransmissionDetails", func(t *testing.T) {
		// Test transmission detail parsing
		transmissionLine := "Incoming transmission from Captain Kirk on channel 1:"
		
		sender, channel, msgType := parser.parseTransmissionDetails(transmissionLine)
		
		if sender != "Captain Kirk" {
			t.Errorf("Expected sender 'Captain Kirk', got '%s'", sender)
		}
		// TWX parseIntSafe will return 0 for "1:" because it can't parse the colon
		if channel != 0 {
			t.Errorf("Expected channel 0 (TWX parseIntSafe behavior), got %d", channel)
		}
		if msgType != MessageRadio {
			t.Errorf("Expected MessageRadio, got %v", msgType)
		}
	})
	
	t.Run("FighterMessage", func(t *testing.T) {
		// Test fighter message parsing
		fighterLine := "Fighter message from sector 1234:"
		
		sender, _, msgType := parser.parseTransmissionDetails(fighterLine)
		
		if msgType != MessageFighter {
			t.Errorf("Expected MessageFighter, got %v", msgType)
		}
		// TWX behavior: includes the colon in the sector number
		if sender != "sector 1234:" {
			t.Errorf("Expected sender 'sector 1234:', got '%s'", sender)
		}
	})
	
	t.Run("ComputerMessage", func(t *testing.T) {
		// Test computer message parsing
		computerLine := "Computer message:"
		
		sender, _, msgType := parser.parseTransmissionDetails(computerLine)
		
		if msgType != MessageComputer {
			t.Errorf("Expected MessageComputer, got %v", msgType)
		}
		if sender != "Computer" {
			t.Errorf("Expected sender 'Computer', got '%s'", sender)
		}
	})
}

func TestStringUtilities(t *testing.T) {
	parser := NewTestTWXParser()
	
	t.Run("StringContainsWord", func(t *testing.T) {
		// Test whole word matching
		testCases := []struct {
			text     string
			word     string
			expected bool
		}{
			{"Hello world", "world", true},
			{"Hello world", "wor", false},
			{"Equipment selling", "equip", false},
			{"Equipment selling", "equipment", true},
		}
		
		for _, test := range testCases {
			result := parser.stringContainsWord(test.text, test.word)
			if result != test.expected {
				t.Errorf("stringContainsWord('%s', '%s') = %t, expected %t", 
					test.text, test.word, result, test.expected)
			}
		}
	})
	
	t.Run("SplitOnCommaOutsideParens", func(t *testing.T) {
		// Test comma splitting that respects parentheses
		text := "Enterprise [Owned by Kirk], (Constitution Class), USS Enterprise"
		parts := parser.splitOnCommaOutsideParens(text)
		
		if len(parts) != 3 {
			t.Errorf("Expected 3 parts, got %d", len(parts))
		}
		
		if parts[0] != "Enterprise [Owned by Kirk]" {
			t.Errorf("Expected first part to be 'Enterprise [Owned by Kirk]', got '%s'", parts[0])
		}
	})
}