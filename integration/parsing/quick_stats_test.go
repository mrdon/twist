package parsing

import (
	"testing"
	"twist/integration/scripting"
)

// TestQuickStats demonstrates quick stats parsing and database storage
func TestQuickStats(t *testing.T) {
	// Execute test using script file - no pre-setup needed, test what parser actually captures
	result := scripting.ExecuteScriptFile(t, "quick_stats.script", nil)

	// Expected values from quick stats display - what the parser should capture
	expectedCredits := 374916
	expectedExperience := 4
	expectedTurns := 20000
	expectedFighters := 2500
	expectedShields := 0
	expectedTotalHolds := 20
	expectedOreHolds := 2
	expectedOrgHolds := 3
	expectedEquHolds := 0
	expectedColHolds := 0  // col_holds
	expectedPhotons := 0
	expectedArmids := 0   // armids (armid mines)
	expectedLimpets := 0  // limpets (limpet mines) 
	expectedGenTorps := 0 // gen_torps (genesis torpedoes/devices)
	expectedCloaks := 0
	expectedBeacons := 0
	expectedAtomics := 0  // atomics (atomic detonators)
	expectedCorbomite := 0 // corbomite (carbonite)
	expectedEprobes := 14  // eprobes (ether probes)
	expectedMineDisr := 0  // mine_disr (mine disruptors)
	expectedAlignment := 16
	expectedShipNumber := 1 // ship_number (Ship 1 MerCru)

	// Verify sector 286 was parsed and saved
	result.Assert.AssertSectorExists(286)
	
	// Verify sector constellation
	result.Assert.AssertSectorConstellation(286, "uncharted space")

	// Verify sector 286 has correct warps: 39, 844
	result.Assert.AssertSectorWithWarps(286, []int{39, 844})

	// Verify the current sector is 286
	result.Assert.AssertCurrentSector(286)

	// Verify port "Grav" was parsed and saved with class 7 (SSS)
	result.Assert.AssertPortExists(286, "Grav", 7)

	// Verify player stats from quick stats display - these should be parsed from the quick stats
	result.Assert.AssertPlayerCredits(expectedCredits)
	result.Assert.AssertPlayerTurns(expectedTurns)
	result.Assert.AssertPlayerExperience(expectedExperience)
	result.Assert.AssertPlayerFighters(expectedFighters)
	result.Assert.AssertPlayerShields(expectedShields)
	
	// Verify player cargo hold information
	result.Assert.AssertPlayerCargo(expectedOreHolds, expectedOrgHolds, expectedEquHolds)
	result.Assert.AssertPlayerTotalHolds(expectedTotalHolds)
	result.Assert.AssertPlayerEmptyHolds(expectedTotalHolds - expectedOreHolds - expectedOrgHolds - expectedEquHolds) // 20 - 2 - 3 - 0 = 15
	
	// Verify player colonists
	result.Assert.AssertPlayerColonists(expectedColHolds)
	
	// Verify player weapons and equipment
	result.Assert.AssertPlayerPhotons(expectedPhotons)
	result.Assert.AssertPlayerArmidMines(expectedArmids)
	result.Assert.AssertPlayerLimpetMines(expectedLimpets)
	result.Assert.AssertPlayerGenesisDevices(expectedGenTorps)
	result.Assert.AssertPlayerCloaks(expectedCloaks)
	result.Assert.AssertPlayerBeacons(expectedBeacons)
	result.Assert.AssertPlayerAtomicDetonators(expectedAtomics)
	result.Assert.AssertPlayerCarbonite(expectedCorbomite)
	result.Assert.AssertPlayerEtherProbes(expectedEprobes)
	result.Assert.AssertPlayerMineDisruptors(expectedMineDisr)
	
	// Verify player alignment and ship number
	result.Assert.AssertPlayerAlignment(expectedAlignment)
	result.Assert.AssertPlayerShipNumber(expectedShipNumber) // 1 = MerCru

	// Verify that OnPlayerStatsUpdated was called during quick stats display
	// Quick stats should trigger player stats updates
	playerStatsCalls := result.TuiAPI.PlayerStatsCalls
	if len(playerStatsCalls) == 0 {
		t.Errorf("Expected OnPlayerStatsUpdated to be called during quick stats display, but got no calls")
	} else {
		t.Logf("OnPlayerStatsUpdated called %d times during quick stats display", len(playerStatsCalls))
		
		// Check that the final player stats call has the expected values
		finalStats := playerStatsCalls[len(playerStatsCalls)-1]
		if finalStats.Credits != expectedCredits {
			t.Errorf("Expected final stats credits to be %d, got %d", expectedCredits, finalStats.Credits)
		}
		if finalStats.Turns != expectedTurns {
			t.Errorf("Expected final stats turns to be %d, got %d", expectedTurns, finalStats.Turns)
		}
		if finalStats.Experience != expectedExperience {
			t.Errorf("Expected final stats experience to be %d, got %d", expectedExperience, finalStats.Experience)
		}
		if finalStats.Fighters != expectedFighters {
			t.Errorf("Expected final stats fighters to be %d, got %d", expectedFighters, finalStats.Fighters)
		}
		if finalStats.Shields != expectedShields {
			t.Errorf("Expected final stats shields to be %d, got %d", expectedShields, finalStats.Shields)
		}
		
		// Verify cargo in player stats events
		if finalStats.OreHolds != expectedOreHolds {
			t.Errorf("Expected final stats ore holds to be %d, got %d", expectedOreHolds, finalStats.OreHolds)
		}
		if finalStats.OrgHolds != expectedOrgHolds {
			t.Errorf("Expected final stats org holds to be %d, got %d", expectedOrgHolds, finalStats.OrgHolds)
		}
		if finalStats.EquHolds != expectedEquHolds {
			t.Errorf("Expected final stats equ holds to be %d, got %d", expectedEquHolds, finalStats.EquHolds)
		}
		if finalStats.TotalHolds != expectedTotalHolds {
			t.Errorf("Expected final stats total holds to be %d, got %d", expectedTotalHolds, finalStats.TotalHolds)
		}
		if finalStats.ColHolds != expectedColHolds {
			t.Errorf("Expected final stats colonists to be %d, got %d", expectedColHolds, finalStats.ColHolds)
		}
		
		// Verify weapons and equipment in player stats events
		if finalStats.Photons != expectedPhotons {
			t.Errorf("Expected final stats photons to be %d, got %d", expectedPhotons, finalStats.Photons)
		}
		if finalStats.Eprobes != expectedEprobes {
			t.Errorf("Expected final stats ether probes to be %d, got %d", expectedEprobes, finalStats.Eprobes)
		}
		if finalStats.Alignment != expectedAlignment {
			t.Errorf("Expected final stats alignment to be %d, got %d", expectedAlignment, finalStats.Alignment)
		}
	}
}