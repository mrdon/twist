package parsing

import (
	"database/sql"
	"testing"
	"twist/integration/scripting"
	"twist/internal/api"
	
	_ "github.com/mattn/go-sqlite3"
)

// TestPortBuy demonstrates port trading parsing and database storage
func TestPortBuy(t *testing.T) {
	// Use ConnectOptions with DatabasePath to enable database storage
	dbPath := t.TempDir() + "/test.db"
	connectOpts := &api.ConnectOptions{DatabasePath: dbPath}

	// Set up test initial values BEFORE running the script
	testCredits := 374999
	testExperience := 0
	testTurns := 19994  // Will be decremented to 19993 after docking
	
	// Pre-setup the database with test values
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database for pre-setup: %v", err)
	}
	defer db.Close()
	
	// Create the database schema first by attempting to connect
	tempResult := scripting.ExecuteScriptFile(t, "port-buy.script", connectOpts)
	if tempResult.Database != nil {
		tempResult.Database.Close()
	}
	
	// Now set up the test initial values
	_, err = db.Exec("INSERT OR REPLACE INTO player_stats (id, credits, experience, turns) VALUES (1, ?, ?, ?)", testCredits, testExperience, testTurns)
	if err != nil {
		t.Fatalf("Failed to set up test initial values: %v", err)
	}

	// Execute test using script file
	result := scripting.ExecuteScriptFile(t, "port-buy.script", connectOpts)

	// Verify sector 286 was parsed and saved
	result.Assert.AssertSectorExists(286)
	
	// Verify sector constellation
	result.Assert.AssertSectorConstellation(286, "uncharted space")

	// Verify sector 286 has correct warps: 39, 844
	result.Assert.AssertSectorWithWarps(286, []int{39, 844})

	// Verify sector was visited and saved as etHolo (explored=3)
	result.Assert.AssertSectorExplorationStatus(286, 3) // etHolo

	// Verify that player credits were updated after trading
	// Started with 374,999, bought 2 Fuel Ore for 25 credits and 3 Organics for 58 credits
	// Final credits should be 374,916
	result.Assert.AssertPlayerCredits(testCredits - 83) // 374,999 - 83 = 374,916

	// Verify that OnCurrentSectorChanged was called for sector 286
	sectorChangeCalls := result.TuiAPI.SectorChangeCalls
	found286 := false
	for _, call := range sectorChangeCalls {
		if call.Number == 286 {
			found286 = true
			break
		}
	}
	
	if !found286 {
		t.Errorf("Expected OnCurrentSectorChanged call for sector 286")
	}

	// Verify the current sector is 286
	result.Assert.AssertCurrentSector(286)

	// Verify port "Grav" was parsed and saved with class 7 (SSS)
	result.Assert.AssertPortExists(286, "Grav", 7)

	// Verify port commodity information from the script
	// Fuel Ore: Selling 2500 at 100%, Organics: Selling 1180 at 100%, Equipment: Selling 1180 at 100%
	result.Assert.AssertPortCommodity(286, "fuel_ore", 2500, 100, false)   // Port is selling (not buying)
	result.Assert.AssertPortCommodity(286, "organics", 1180, 100, false)   // Port is selling (not buying)  
	result.Assert.AssertPortCommodity(286, "equipment", 1180, 100, false)  // Port is selling (not buying)

	// Verify player cargo hold contents after trading
	// Bought 2 Fuel Ore and 3 Organics, 0 Equipment
	result.Assert.AssertPlayerCargo(2, 3, 0)
	
	// Verify player has 20 total cargo holds (as shown in port screen)
	result.Assert.AssertPlayerTotalHolds(20)
	
	// Verify player has 15 empty cargo holds after trading (started with 20, bought 5 total)
	result.Assert.AssertPlayerEmptyHolds(15)
	
	// Verify player turns were decremented by 1 for docking (19994 -> 19993)
	result.Assert.AssertPlayerTurns(testTurns - 1)
	
	// Verify port status: not dead, build time 0 (active port)
	result.Assert.AssertPortStatus(286, false, 0)
	
	// Verify player experience increased by 4 points total:
	// 1 (finding unused port) + 2 (fuel ore trade) + 1 (organics trade) = 4 experience points
	result.Assert.AssertPlayerExperience(testExperience + 4)
	
	// Verify that OnPlayerStatsUpdated was called during port trading
	// Port trading should trigger player stats updates for credits, experience, cargo, and turns
	playerStatsCalls := result.TuiAPI.PlayerStatsCalls
	if len(playerStatsCalls) == 0 {
		t.Errorf("Expected OnPlayerStatsUpdated to be called during port trading, but got no calls")
	} else {
		t.Logf("OnPlayerStatsUpdated called %d times during port trading", len(playerStatsCalls))
		
		// Check that the final player stats call has the expected values
		finalStats := playerStatsCalls[len(playerStatsCalls)-1]
		if finalStats.Credits != testCredits-83 {
			t.Errorf("Expected final stats credits to be %d, got %d", testCredits-83, finalStats.Credits)
		}
		if finalStats.Turns != testTurns-1 {
			t.Errorf("Expected final stats turns to be %d, got %d", testTurns-1, finalStats.Turns)
		}
		// Verify cargo in player stats events
		if finalStats.OreHolds != 2 {
			t.Errorf("Expected final stats ore holds to be 2, got %d", finalStats.OreHolds)
		}
		if finalStats.OrgHolds != 3 {
			t.Errorf("Expected final stats org holds to be 3, got %d", finalStats.OrgHolds)
		}
		if finalStats.EquHolds != 0 {
			t.Errorf("Expected final stats equ holds to be 0, got %d", finalStats.EquHolds)
		}
		if finalStats.TotalHolds != 20 {
			t.Errorf("Expected final stats total holds to be 20, got %d", finalStats.TotalHolds)
		}
	}
}