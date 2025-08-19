package parsing

import (
	"testing"
	"twist/integration/scripting"
)

// TestInfo demonstrates parsing of the 'i' (info) command display and database storage
func TestInfo(t *testing.T) {
	// Execute test using script file - no pre-setup needed, test what parser actually captures
	result := scripting.ExecuteScriptFile(t, "info_test.script", nil)

	// Expected values from the info command output in the script
	expectedCredits := 140585
	expectedExperience := 4
	expectedTurns := 19993
	expectedFighters := 2500
	expectedShields := 0
	expectedTotalHolds := 20
	expectedOreHolds := 2
	expectedOrgHolds := 3
	expectedEquHolds := 0
	expectedColHolds := 0
	expectedPhotons := 0
	expectedArmids := 0
	expectedLimpets := 0
	expectedGenTorps := 0
	expectedCloaks := 0
	expectedBeacons := 0
	expectedAtomics := 0
	expectedCorbomite := 0
	expectedEprobes := 25
	expectedMineDisr := 0
	expectedAlignment := 28
	expectedShipNumber := 1

	// Verify player stats from the info display were parsed correctly
	// From the info display:
	// Trader Name: Private 1st Class mrdon
	// Rank and Exp: 4 points, Alignment=28 Tolerant
	// Ship Info: Le Richelieu Merchant Cruiser Ported=3 Kills=0
	// Date Built: 12:21:54 PM Sun Aug 17, 2053
	// Turns to Warp: 3
	// Current Sector: 190
	// Turns left: 19993
	// Total Holds: 20 - Fuel Ore=2 Organics=3 Empty=15
	// Fighters: 2,500
	// Ether Probes: 25
	// LongRange Scan: Holographic Scanner
	// Credits: 140,585

	result.Assert.AssertPlayerCredits(expectedCredits)
	result.Assert.AssertPlayerExperience(expectedExperience)
	result.Assert.AssertPlayerTurns(expectedTurns)
	result.Assert.AssertPlayerFighters(expectedFighters)
	result.Assert.AssertPlayerShields(expectedShields)
	result.Assert.AssertPlayerTotalHolds(expectedTotalHolds)
	result.Assert.AssertPlayerCargo(expectedOreHolds, expectedOrgHolds, expectedEquHolds)
	result.Assert.AssertPlayerEmptyHolds(15) // 20 total - 2 ore - 3 organics - 0 equipment = 15
	result.Assert.AssertPlayerColonists(expectedColHolds)
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
	result.Assert.AssertPlayerAlignment(expectedAlignment)
	result.Assert.AssertPlayerShipNumber(expectedShipNumber)
	
	// Verify that OnPlayerStatsUpdated was called during info command
	// The info command should trigger player stats updates
	playerStatsCalls := result.TuiAPI.PlayerStatsCalls
	if len(playerStatsCalls) == 0 {
		t.Errorf("Expected OnPlayerStatsUpdated to be called during info command, but got no calls")
	} else {
		t.Logf("OnPlayerStatsUpdated called %d times during info command", len(playerStatsCalls))
		
		// Check that the player stats call has the expected values
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
		if finalStats.Alignment != expectedAlignment {
			t.Errorf("Expected final stats alignment to be %d, got %d", expectedAlignment, finalStats.Alignment)
		}
		if finalStats.Eprobes != expectedEprobes {
			t.Errorf("Expected final stats ether probes to be %d, got %d", expectedEprobes, finalStats.Eprobes)
		}
	}
}