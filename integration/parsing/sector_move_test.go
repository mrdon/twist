package parsing

import (
	"database/sql"
	"testing"
	"twist/integration/scripting"
	"twist/internal/api"
	
	_ "modernc.org/sqlite"
)

// TestSectorMove demonstrates sector movement parsing and database storage
func TestSectorMove(t *testing.T) {
	// Use ConnectOptions with DatabasePath to enable database storage
	dbPath := t.TempDir() + "/test.db"
	connectOpts := &api.ConnectOptions{DatabasePath: dbPath}

	// Set up test credits BEFORE running the script
	testCredits := 374999
	
	// Pre-setup the database with test credits
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database for pre-setup: %v", err)
	}
	defer db.Close()
	
	// Create the database schema first by attempting to connect
	tempResult := scripting.ExecuteScriptFile(t, "sector_move.script", connectOpts)
	if tempResult.Database != nil {
		tempResult.Database.Close()
	}
	
	// Now set up the test credits
	_, err = db.Exec("INSERT OR REPLACE INTO player_stats (id, credits) VALUES (1, ?)", testCredits)
	if err != nil {
		t.Fatalf("Failed to set up test credits: %v", err)
	}

	// Execute test using script file
	result := scripting.ExecuteScriptFile(t, "sector_move.script", connectOpts)

	// Verify sectors were parsed and saved
	result.Assert.AssertSectorExists(705) // Starting sector
	result.Assert.AssertSectorExists(279) // Destination sector
	
	// Verify sector constellations
	result.Assert.AssertSectorConstellation(705, "uncharted space")
	result.Assert.AssertSectorConstellation(279, "uncharted space")

	// Verify sector 705 has correct warps: 279, 903, 927
	// (903) just means sector 903 is undiscovered, but still a valid warp
	result.Assert.AssertSectorWithWarps(705, []int{279, 903, 927})
	
	// Verify sector 279 has correct warps: 578, 705, 810, 844, 877
	// (578) just means sector 578 is undiscovered, but still a valid warp  
	result.Assert.AssertSectorWithWarps(279, []int{578, 705, 810, 844, 877})

	// Verify both sectors were visited and saved as etHolo (explored=3)
	result.Assert.AssertSectorExplorationStatus(705, 3) // etHolo
	result.Assert.AssertSectorExplorationStatus(279, 3) // etHolo

	// Verify that player credits weren't affected by the parsing
	result.Assert.AssertPlayerCredits(testCredits)

	// Verify that OnCurrentSectorChanged was called correctly for player movement
	// Should be called first for sector 705 (starting sector) and then for sector 279 (after move)
	sectorChangeCalls := result.TuiAPI.SectorChangeCalls
	if len(sectorChangeCalls) < 2 {
		t.Errorf("Expected at least 2 OnCurrentSectorChanged calls, got %d", len(sectorChangeCalls))
	}
	
	// Check that we got calls for both sectors in the right order
	found705 := false
	found279 := false
	lastCallFor705 := -1
	firstCallFor279 := -1
	
	for i, call := range sectorChangeCalls {
		if call.Number == 705 {
			found705 = true
			lastCallFor705 = i
		}
		if call.Number == 279 && !found279 {
			found279 = true
			firstCallFor279 = i
		}
	}
	
	if !found705 {
		t.Errorf("Expected OnCurrentSectorChanged call for sector 705")
	}
	if !found279 {
		t.Errorf("Expected OnCurrentSectorChanged call for sector 279")
	}
	
	// Verify that sector 705 calls came before sector 279 calls
	if found705 && found279 && lastCallFor705 >= firstCallFor279 {
		t.Errorf("Expected sector 705 events to come before sector 279 events, but last 705 call at %d >= first 279 call at %d", lastCallFor705, firstCallFor279)
	}

	// Verify the final current sector is 279 (where the player moved to)
	result.Assert.AssertCurrentSector(279)
}