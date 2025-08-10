package parsing

import (
	"testing"
	"twist/internal/proxy/database"
)

// TestExplorationStatusPreservation tests that visited sectors maintain EtHolo status
// even when density scan information is processed afterward
func TestExplorationStatusPreservation(t *testing.T) {
	parser, _, db := CreateTestParser(t)
	defer db.CloseDatabase()

	sectorNum := 2921

	// Step 1: Simulate visiting sector 2921 (this should set it to EtHolo)
	sectorVisitText := `
Sector  : 2921 in uncharted space.
Warps to Sector(s) :  3212 - 7656

Command [TL=00:00:00]:[2921] (?=Help)? : `

	// Process the sector visit - this should trigger setting exploration status to EtHolo
	parser.ProcessInBound(sectorVisitText)
	parser.Finalize()

	// Verify sector was set to EtHolo after visit
	sector, err := db.LoadSector(sectorNum)
	if err != nil {
		t.Fatalf("Failed to load sector %d: %v", sectorNum, err)
	}

	if sector.Explored != database.EtHolo {
		t.Errorf("After sector visit, expected exploration status EtHolo (%d), got %d", database.EtHolo, sector.Explored)
	}

	// Step 2: Simulate density scan information being processed for the same sector
	// This mimics the bug where processDensityLine was downgrading visited sectors to EtDensity
	// First trigger density scan mode with the start trigger
	densityScanStart := "Sector 2921 (Uncharted Space) Density: 1500 NavHaz: 0% Warps: 2 Anomaly: No\r\n" + 
	                     "This is a Density Scanner report.\r\n\r\n"
	
	// Process the density scan - this should trigger density mode and then process the sector line
	parser.ProcessInBound(densityScanStart)
	parser.Finalize()

	// Step 3: Verify the sector maintains EtHolo status and was not downgraded to EtDensity
	sector, err = db.LoadSector(sectorNum)
	if err != nil {
		t.Fatalf("Failed to load sector %d after density scan: %v", sectorNum, err)
	}

	if sector.Explored != database.EtHolo {
		t.Errorf("After density scan, expected exploration status to remain EtHolo (%d), but got %d", database.EtHolo, sector.Explored)
	}

	// Additional verification: ensure the density information was still processed
	if sector.Density != 1500 {
		t.Errorf("Expected density to be updated to 1500, got %d", sector.Density)
	}

	if sector.Warps != 2 {
		t.Errorf("Expected warps count to be updated to 2, got %d", sector.Warps)
	}
}

// TestExplorationStatusTransitionFromNoToHolo tests that unvisited sectors
// can still be properly upgraded from EtNo to EtHolo when visited
func TestExplorationStatusTransitionFromNoToHolo(t *testing.T) {
	parser, _, db := CreateTestParser(t)
	defer db.CloseDatabase()

	sectorNum := 3212

	// Step 1: Ensure sector starts as EtNo (unvisited)
	sector, err := db.LoadSector(sectorNum)
	if err != nil {
		// Sector doesn't exist yet, which is equivalent to EtNo
		sector = database.NULLSector()
	}

	if sector.Explored != database.EtNo {
		// Reset to EtNo for this test
		sector.Explored = database.EtNo
		err := db.SaveSector(sector, sectorNum)
		if err != nil {
			t.Fatalf("Failed to reset sector %d to EtNo: %v", sectorNum, err)
		}
	}

	// Step 2: Process density scan first (should set to EtDensity)
	densityText := "Sector 3212 (Uncharted Space) Density: 2000 NavHaz: 0% Warps: 4 Anomaly: No"
	parser.ProcessInBound(densityText)
	parser.Finalize()

	// Verify it was set to EtDensity
	sector, err = db.LoadSector(sectorNum)
	if err != nil {
		t.Fatalf("Failed to load sector %d after density scan: %v", sectorNum, err)
	}

	if sector.Explored != database.EtDensity {
		t.Errorf("After density scan, expected EtDensity (%d), got %d", database.EtDensity, sector.Explored)
	}

	// Step 3: Visit the sector (should upgrade to EtHolo)
	sectorVisitText := `
Sector  : 3212 in uncharted space.
Warps to Sector(s) :  2921 - 10870 - (16983) - (17563)

Command [TL=00:00:00]:[3212] (?=Help)? : `

	parser.ProcessInBound(sectorVisitText)
	parser.Finalize()

	// Step 4: Verify it was upgraded to EtHolo
	sector, err = db.LoadSector(sectorNum)
	if err != nil {
		t.Fatalf("Failed to load sector %d after visit: %v", sectorNum, err)
	}

	if sector.Explored != database.EtHolo {
		t.Errorf("After sector visit, expected EtHolo (%d), got %d", database.EtHolo, sector.Explored)
	}

	// Step 5: Process another density scan (should NOT downgrade back to EtDensity)
	parser.ProcessInBound(densityText)
	parser.Finalize()

	sector, err = db.LoadSector(sectorNum)
	if err != nil {
		t.Fatalf("Failed to load sector %d after second density scan: %v", sectorNum, err)
	}

	if sector.Explored != database.EtHolo {
		t.Errorf("After second density scan, expected exploration status to remain EtHolo (%d), but got %d", database.EtHolo, sector.Explored)
	}
}

// TestExplorationStatusUnvisitedSectors tests that unvisited sectors are still
// properly handled by density scans
func TestExplorationStatusUnvisitedSectors(t *testing.T) {
	parser, _, db := CreateTestParser(t)
	defer db.CloseDatabase()

	sectorNum := 7656

	// Process density scan for unvisited sector (should set to EtDensity)
	densityText := "Sector 7656 (Uncharted Space) Density: 800 NavHaz: 0% Warps: 3 Anomaly: Yes"
	parser.ProcessInBound(densityText)
	parser.Finalize()

	// Verify it was set to EtDensity
	sector, err := db.LoadSector(sectorNum)
	if err != nil {
		t.Fatalf("Failed to load sector %d: %v", sectorNum, err)
	}

	if sector.Explored != database.EtDensity {
		t.Errorf("Expected unvisited sector to be set to EtDensity (%d), got %d", database.EtDensity, sector.Explored)
	}

	// Verify density and other data was properly set
	if sector.Density != 800 {
		t.Errorf("Expected density 800, got %d", sector.Density)
	}

	if sector.Warps != 3 {
		t.Errorf("Expected warps count 3, got %d", sector.Warps)
	}

	if !sector.Anomaly {
		t.Error("Expected anomaly to be true, got false")
	}

	if sector.Constellation != "??? (Density only)" {
		t.Errorf("Expected constellation to be '??? (Density only)', got '%s'", sector.Constellation)
	}
}