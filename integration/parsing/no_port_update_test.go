package parsing

import (
	"testing"
)

// TestNoPortDetectionUpdatesDatabase verifies that when we visit a sector
// and no port is found, the database is properly updated to remove any
// existing port data for that sector.
func TestNoPortDetectionUpdatesDatabase(t *testing.T) {
	// Create test parser with database
	parser, _, db := CreateTestParser(t)
	defer db.CloseDatabase()

	// First, simulate a sector with a port (to establish initial state)
	sectorWithPort := "Sector  : 1234 in uncharted space\r\n" +
		"Warps to Sector(s) : 5678\r\n" +
		"Ports   : Alpha Station, Class 1 BBS (Ore-61% Organics-80% Equipment-90%)\r\n" +
		"\r\n"

	// Parse sector with port
	parser.ProcessInBound(sectorWithPort)
	parser.Finalize()

	// Verify port was saved
	port, err := db.LoadPort(1234)
	if err != nil {
		t.Fatalf("Expected port to be saved, but got error: %v", err)
	}
	if port.Name != "Alpha Station" {
		t.Errorf("Expected port name 'Alpha Station', got '%s'", port.Name)
	}
	if port.ClassIndex != 1 {
		t.Errorf("Expected port class 1, got %d", port.ClassIndex)
	}

	// Now simulate visiting the same sector but with no port
	sectorWithoutPort := "Sector  : 1234 in uncharted space\r\n" +
		"Warps to Sector(s) : 5678\r\n" +
		"\r\n" // No port line - just end the sector

	// Parse sector without port  
	parser.ProcessInBound(sectorWithoutPort)
	parser.Finalize()

	// Verify port data was cleared
	port, err = db.LoadPort(1234)
	if err == nil && port.ClassIndex > 0 {
		t.Errorf("Expected port data to be cleared, but port still exists: %+v", port)
	}
}

// TestNoPortDetectionWithMultipleSectors verifies the fix works correctly
// across multiple sector visits
func TestNoPortDetectionWithMultipleSectors(t *testing.T) {
	// Create test parser with database
	parser, _, db := CreateTestParser(t)
	defer db.CloseDatabase()

	// Sector 1000 with port
	sector1000WithPort := "Sector  : 1000 in uncharted space\r\n" +
		"Warps to Sector(s) : 1001\r\n" +
		"Ports   : Trading Post, Class 2 BSS (Ore-50% Organics-70% Equipment-85%)\r\n" +
		"\r\n"

	parser.ProcessInBound(sector1000WithPort)

	// Sector 1001 without port
	sector1001WithoutPort := "Sector  : 1001 in uncharted space\r\n" +
		"Warps to Sector(s) : 1000 1002\r\n" +
		"\r\n" // No port

	parser.ProcessInBound(sector1001WithoutPort)

	// Sector 1002 with port
	sector1002WithPort := "Sector  : 1002 in uncharted space\r\n" +
		"Warps to Sector(s) : 1001\r\n" +
		"Ports   : Fuel Depot, Class 3 SBS (Ore-40% Organics-60% Equipment-75%)\r\n" +
		"\r\n"

	parser.ProcessInBound(sector1002WithPort)
	parser.Finalize()

	// Verify sector 1000 still has port
	port1000, err := db.LoadPort(1000)
	if err != nil || port1000.ClassIndex <= 0 {
		t.Errorf("Expected sector 1000 to have port, got error: %v, port: %+v", err, port1000)
	}

	// Verify sector 1001 has no port
	port1001, err := db.LoadPort(1001)
	if err == nil && port1001.ClassIndex > 0 {
		t.Errorf("Expected sector 1001 to have no port, but found: %+v", port1001)
	}

	// Verify sector 1002 has port
	port1002, err := db.LoadPort(1002)
	if err != nil || port1002.ClassIndex <= 0 {
		t.Errorf("Expected sector 1002 to have port, got error: %v, port: %+v", err, port1002)
	}
}

// TestPortUpdateWhenVisitingSectorWithDifferentPort verifies that visiting
// a sector updates the port data correctly when the port has changed
func TestPortUpdateWhenVisitingSectorWithDifferentPort(t *testing.T) {
	// Create test parser with database
	parser, _, db := CreateTestParser(t)
	defer db.CloseDatabase()

	// First visit - sector with Class 1 port
	firstVisit := "Sector  : 7000 in uncharted space\r\n" +
		"Warps to Sector(s) : 7001\r\n" +
		"Ports   : Original Port, Class 1 BBS (Ore-50% Organics-60% Equipment-70%)\r\n" +
		"\r\n"

	parser.ProcessInBound(firstVisit)
	parser.Finalize()

	// Verify first port
	port, err := db.LoadPort(7000)
	if err != nil {
		t.Fatalf("Expected port to be saved, got error: %v", err)
	}
	if port.Name != "Original Port" || port.ClassIndex != 1 {
		t.Errorf("Expected 'Original Port' class 1, got '%s' class %d", port.Name, port.ClassIndex)
	}

	// Second visit - same sector but different port (port upgraded)
	secondVisit := "Sector  : 7000 in uncharted space\r\n" +
		"Warps to Sector(s) : 7001\r\n" +
		"Ports   : Upgraded Station, Class 5 SSB (Ore-80% Organics-85% Equipment-90%)\r\n" +
		"\r\n"

	parser.ProcessInBound(secondVisit)
	parser.Finalize()

	// Verify port was updated
	port, err = db.LoadPort(7000)
	if err != nil {
		t.Fatalf("Expected updated port, got error: %v", err)
	}
	if port.Name != "Upgraded Station" || port.ClassIndex != 5 {
		t.Errorf("Expected 'Upgraded Station' class 5, got '%s' class %d", port.Name, port.ClassIndex)
	}

	// Third visit - same sector but no port (port destroyed)
	thirdVisit := "Sector  : 7000 in uncharted space\r\n" +
		"Warps to Sector(s) : 7001\r\n" +
		"\r\n" // No port

	parser.ProcessInBound(thirdVisit)
	parser.Finalize()

	// Verify port was cleared
	port, err = db.LoadPort(7000)
	if err == nil && port.ClassIndex > 0 {
		t.Errorf("Expected port to be cleared, but still exists: %+v", port)
	}
}