package setup

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

// DBAsserts provides database assertion helpers for integration tests
type DBAsserts struct {
	t  *testing.T
	db *sql.DB
}

// NewDBAsserts creates a new DBAsserts instance using an existing database connection
func NewDBAsserts(t *testing.T, db *sql.DB) *DBAsserts {
	if db == nil {
		t.Fatal("Database connection is nil")
	}

	return &DBAsserts{
		t:  t,
		db: db,
	}
}

// Note: Assert does not close the database connection since it doesn't own it

// AssertSectorWithWarps verifies that a sector exists with the specified warp connections
func (a *DBAsserts) AssertSectorWithWarps(sectorNum int, expectedWarps []int) {
	// Check if sector exists
	var exists int
	err := a.db.QueryRow("SELECT COUNT(*) FROM sectors WHERE sector_index = ?", sectorNum).Scan(&exists)
	if err != nil {
		a.t.Fatalf("Failed to check if sector %d exists: %v", sectorNum, err)
	}
	if exists != 1 {
		a.t.Errorf("Expected sector %d to exist, but found %d records", sectorNum, exists)
		return
	}

	// Get actual warps for the sector
	var actualWarps [6]int
	err = a.db.QueryRow("SELECT warp1, warp2, warp3, warp4, warp5, warp6 FROM sectors WHERE sector_index = ?", sectorNum).Scan(
		&actualWarps[0], &actualWarps[1], &actualWarps[2], &actualWarps[3], &actualWarps[4], &actualWarps[5])
	if err != nil {
		a.t.Fatalf("Failed to query warps for sector %d: %v", sectorNum, err)
		return
	}

	// Convert actual warps to slice, filtering out zeros
	var nonZeroWarps []int
	for _, warp := range actualWarps {
		if warp != 0 {
			nonZeroWarps = append(nonZeroWarps, warp)
		}
	}

	// Check that all expected warps are present
	for _, expectedWarp := range expectedWarps {
		found := false
		for _, actualWarp := range nonZeroWarps {
			if actualWarp == expectedWarp {
				found = true
				break
			}
		}
		if !found {
			a.t.Errorf("Expected warp from sector %d to %d, but not found. Actual warps: %v",
				sectorNum, expectedWarp, nonZeroWarps)
		}
	}

	// Check that no unexpected warps are present (optional - can be strict or lenient)
	for _, actualWarp := range nonZeroWarps {
		found := false
		for _, expectedWarp := range expectedWarps {
			if expectedWarp == actualWarp {
				found = true
				break
			}
		}
		if !found {
			a.t.Logf("Found unexpected warp from sector %d to %d. Actual warps: %v",
				sectorNum, actualWarp, nonZeroWarps)
		}
	}

	a.t.Logf("Sector %d warps verified: %v", sectorNum, nonZeroWarps)
}

// AssertSectorExists verifies that a sector exists in the database
func (a *DBAsserts) AssertSectorExists(sectorNum int) {
	var exists int
	err := a.db.QueryRow("SELECT COUNT(*) FROM sectors WHERE sector_index = ?", sectorNum).Scan(&exists)
	if err != nil {
		a.t.Fatalf("Failed to check if sector %d exists: %v", sectorNum, err)
	}
	if exists != 1 {
		a.t.Errorf("Expected sector %d to exist, but found %d records", sectorNum, exists)
	}
}

// AssertPortExists verifies that a port exists in the specified sector
func (a *DBAsserts) AssertPortExists(sectorNum int, portName string, classIndex int) {
	var exists int
	err := a.db.QueryRow("SELECT COUNT(*) FROM ports WHERE sector_index = ? AND name = ? AND class_index = ?",
		sectorNum, portName, classIndex).Scan(&exists)
	if err != nil {
		a.t.Fatalf("Failed to check if port '%s' exists in sector %d: %v", portName, sectorNum, err)
	}
	if exists != 1 {
		a.t.Errorf("Expected port '%s' Class %d in sector %d, but found %d records",
			portName, classIndex, sectorNum, exists)
	}
}

// AssertSectorConstellation verifies that a sector has the expected constellation name
func (a *DBAsserts) AssertSectorConstellation(sectorNum int, expectedConstellation string) {
	var actualConstellation string
	err := a.db.QueryRow("SELECT constellation FROM sectors WHERE sector_index = ?", sectorNum).Scan(&actualConstellation)
	if err != nil {
		a.t.Fatalf("Failed to get constellation for sector %d: %v", sectorNum, err)
	}
	if actualConstellation != expectedConstellation {
		a.t.Errorf("Expected sector %d constellation to be %q, got %q",
			sectorNum, expectedConstellation, actualConstellation)
	}
}

// AssertCurrentSector verifies that the current sector is set to the expected value
func (a *DBAsserts) AssertCurrentSector(expectedSector int) {
	var currentSector int
	err := a.db.QueryRow("SELECT COALESCE(current_sector, 0) FROM player_stats WHERE id = 1").Scan(&currentSector)
	if err != nil {
		a.t.Fatalf("Failed to get current sector: %v", err)
	}
	if currentSector != expectedSector {
		a.t.Errorf("Expected current sector to be %d, got %d", expectedSector, currentSector)
	}
}

// AssertPlayerCredits verifies that player credits match the expected value
func (a *DBAsserts) AssertPlayerCredits(expectedCredits int) {
	var actualCredits int
	err := a.db.QueryRow("SELECT COALESCE(credits, 0) FROM player_stats WHERE id = 1").Scan(&actualCredits)
	if err != nil {
		a.t.Fatalf("Failed to get player credits: %v", err)
	}
	if actualCredits != expectedCredits {
		a.t.Errorf("Expected player credits to be %d, got %d", expectedCredits, actualCredits)
	}
}

// SetPlayerCredits sets the player credits to a specific value for testing
func (a *DBAsserts) SetPlayerCredits(credits int) {
	_, err := a.db.Exec("INSERT OR REPLACE INTO player_stats (id, credits) VALUES (1, ?)", credits)
	if err != nil {
		a.t.Fatalf("Failed to set player credits: %v", err)
	}
}

// AssertSectorExplorationStatus verifies that a sector has the expected exploration status
func (a *DBAsserts) AssertSectorExplorationStatus(sectorNum int, expectedExplored int) {
	var actualExplored int
	err := a.db.QueryRow("SELECT explored FROM sectors WHERE sector_index = ?", sectorNum).Scan(&actualExplored)
	if err != nil {
		a.t.Fatalf("Failed to get exploration status for sector %d: %v", sectorNum, err)
	}
	if actualExplored != expectedExplored {
		a.t.Errorf("Expected sector %d exploration status to be %d, got %d", sectorNum, expectedExplored, actualExplored)
	}
}

// AssertPortCommodity verifies that a port has the expected commodity information
func (a *DBAsserts) AssertPortCommodity(sectorNum int, commodityType string, expectedAmount int, expectedPercent int, expectedBuying bool) {
	var actualAmount, actualPercent int
	var actualBuying bool
	
	var query string
	switch commodityType {
	case "fuel_ore":
		query = "SELECT amount_fuel_ore, percent_fuel_ore, buy_fuel_ore FROM ports WHERE sector_index = ?"
	case "organics":
		query = "SELECT amount_organics, percent_organics, buy_organics FROM ports WHERE sector_index = ?"
	case "equipment":
		query = "SELECT amount_equipment, percent_equipment, buy_equipment FROM ports WHERE sector_index = ?"
	default:
		a.t.Fatalf("Invalid commodity type: %s", commodityType)
		return
	}
	
	err := a.db.QueryRow(query, sectorNum).Scan(&actualAmount, &actualPercent, &actualBuying)
	if err != nil {
		a.t.Fatalf("Failed to get %s commodity info for port in sector %d: %v", commodityType, sectorNum, err)
	}
	
	if actualAmount != expectedAmount {
		a.t.Errorf("Expected port %s amount to be %d, got %d", commodityType, expectedAmount, actualAmount)
	}
	if actualPercent != expectedPercent {
		a.t.Errorf("Expected port %s percent to be %d, got %d", commodityType, expectedPercent, actualPercent)
	}
	if actualBuying != expectedBuying {
		a.t.Errorf("Expected port %s buying status to be %t, got %t", commodityType, expectedBuying, actualBuying)
	}
}

// AssertPlayerCargo verifies that player has the expected cargo amounts
func (a *DBAsserts) AssertPlayerCargo(expectedOre, expectedOrganics, expectedEquipment int) {
	var actualOre, actualOrganics, actualEquipment int
	err := a.db.QueryRow("SELECT COALESCE(ore_holds, 0), COALESCE(org_holds, 0), COALESCE(equ_holds, 0) FROM player_stats WHERE id = 1").Scan(&actualOre, &actualOrganics, &actualEquipment)
	if err != nil {
		a.t.Fatalf("Failed to get player cargo: %v", err)
	}
	
	if actualOre != expectedOre {
		a.t.Errorf("Expected player ore holds to be %d, got %d", expectedOre, actualOre)
	}
	if actualOrganics != expectedOrganics {
		a.t.Errorf("Expected player organics holds to be %d, got %d", expectedOrganics, actualOrganics)
	}
	if actualEquipment != expectedEquipment {
		a.t.Errorf("Expected player equipment holds to be %d, got %d", expectedEquipment, actualEquipment)
	}
}

// AssertPlayerEmptyHolds verifies that player has the expected number of empty cargo holds
func (a *DBAsserts) AssertPlayerEmptyHolds(expectedEmpty int) {
	var totalHolds, oreHolds, orgHolds, equHolds int
	err := a.db.QueryRow("SELECT COALESCE(total_holds, 0), COALESCE(ore_holds, 0), COALESCE(org_holds, 0), COALESCE(equ_holds, 0) FROM player_stats WHERE id = 1").Scan(&totalHolds, &oreHolds, &orgHolds, &equHolds)
	if err != nil {
		a.t.Fatalf("Failed to get player cargo holds: %v", err)
	}
	
	actualEmpty := totalHolds - oreHolds - orgHolds - equHolds
	if actualEmpty != expectedEmpty {
		a.t.Errorf("Expected player empty holds to be %d, got %d (total: %d, ore: %d, org: %d, equ: %d)", 
			expectedEmpty, actualEmpty, totalHolds, oreHolds, orgHolds, equHolds)
	}
}

// AssertPlayerTotalHolds verifies that player has the expected total number of cargo holds
func (a *DBAsserts) AssertPlayerTotalHolds(expectedTotal int) {
	var actualTotal int
	err := a.db.QueryRow("SELECT COALESCE(total_holds, 0) FROM player_stats WHERE id = 1").Scan(&actualTotal)
	if err != nil {
		a.t.Fatalf("Failed to get player total holds: %v", err)
	}
	if actualTotal != expectedTotal {
		a.t.Errorf("Expected player total holds to be %d, got %d", expectedTotal, actualTotal)
	}
}

// AssertPlayerExperience verifies that player has the expected experience points
func (a *DBAsserts) AssertPlayerExperience(expectedExperience int) {
	var actualExperience int
	err := a.db.QueryRow("SELECT COALESCE(experience, 0) FROM player_stats WHERE id = 1").Scan(&actualExperience)
	if err != nil {
		a.t.Fatalf("Failed to get player experience: %v", err)
	}
	if actualExperience != expectedExperience {
		a.t.Errorf("Expected player experience to be %d, got %d", expectedExperience, actualExperience)
	}
}

// AssertPlayerTurns verifies that player has the expected number of turns
func (a *DBAsserts) AssertPlayerTurns(expectedTurns int) {
	var actualTurns int
	err := a.db.QueryRow("SELECT COALESCE(turns, 0) FROM player_stats WHERE id = 1").Scan(&actualTurns)
	if err != nil {
		a.t.Fatalf("Failed to get player turns: %v", err)
	}
	if actualTurns != expectedTurns {
		a.t.Errorf("Expected player turns to be %d, got %d", expectedTurns, actualTurns)
	}
}

// AssertPortStatus verifies port status information (dead status, build time)
func (a *DBAsserts) AssertPortStatus(sectorNum int, expectedDead bool, expectedBuildTime int) {
	var actualDead bool
	var actualBuildTime int
	err := a.db.QueryRow("SELECT dead, build_time FROM ports WHERE sector_index = ?", sectorNum).Scan(&actualDead, &actualBuildTime)
	if err != nil {
		a.t.Fatalf("Failed to get port status for sector %d: %v", sectorNum, err)
	}
	
	if actualDead != expectedDead {
		a.t.Errorf("Expected port dead status to be %t, got %t", expectedDead, actualDead)
	}
	if actualBuildTime != expectedBuildTime {
		a.t.Errorf("Expected port build time to be %d, got %d", expectedBuildTime, actualBuildTime)
	}
}

// AssertPlayerFighters verifies that player has the expected number of fighters
func (a *DBAsserts) AssertPlayerFighters(expectedFighters int) {
	var actualFighters int
	err := a.db.QueryRow("SELECT COALESCE(fighters, 0) FROM player_stats WHERE id = 1").Scan(&actualFighters)
	if err != nil {
		a.t.Fatalf("Failed to get player fighters: %v", err)
	}
	if actualFighters != expectedFighters {
		a.t.Errorf("Expected player fighters to be %d, got %d", expectedFighters, actualFighters)
	}
}

// AssertPlayerShields verifies that player has the expected number of shields
func (a *DBAsserts) AssertPlayerShields(expectedShields int) {
	var actualShields int
	err := a.db.QueryRow("SELECT COALESCE(shields, 0) FROM player_stats WHERE id = 1").Scan(&actualShields)
	if err != nil {
		a.t.Fatalf("Failed to get player shields: %v", err)
	}
	if actualShields != expectedShields {
		a.t.Errorf("Expected player shields to be %d, got %d", expectedShields, actualShields)
	}
}

// AssertPlayerColonists verifies that player has the expected number of colonists
func (a *DBAsserts) AssertPlayerColonists(expectedColonists int) {
	var actualColonists int
	err := a.db.QueryRow("SELECT COALESCE(col_holds, 0) FROM player_stats WHERE id = 1").Scan(&actualColonists)
	if err != nil {
		a.t.Fatalf("Failed to get player colonists: %v", err)
	}
	if actualColonists != expectedColonists {
		a.t.Errorf("Expected player colonists to be %d, got %d", expectedColonists, actualColonists)
	}
}

// AssertPlayerPhotons verifies that player has the expected number of photons
func (a *DBAsserts) AssertPlayerPhotons(expectedPhotons int) {
	var actualPhotons int
	err := a.db.QueryRow("SELECT COALESCE(photons, 0) FROM player_stats WHERE id = 1").Scan(&actualPhotons)
	if err != nil {
		a.t.Fatalf("Failed to get player photons: %v", err)
	}
	if actualPhotons != expectedPhotons {
		a.t.Errorf("Expected player photons to be %d, got %d", expectedPhotons, actualPhotons)
	}
}

// AssertPlayerArmidMines verifies that player has the expected number of armid mines
func (a *DBAsserts) AssertPlayerArmidMines(expectedArmids int) {
	var actualArmids int
	err := a.db.QueryRow("SELECT COALESCE(armids, 0) FROM player_stats WHERE id = 1").Scan(&actualArmids)
	if err != nil {
		a.t.Fatalf("Failed to get player armid mines: %v", err)
	}
	if actualArmids != expectedArmids {
		a.t.Errorf("Expected player armid mines to be %d, got %d", expectedArmids, actualArmids)
	}
}

// AssertPlayerLimpetMines verifies that player has the expected number of limpet mines
func (a *DBAsserts) AssertPlayerLimpetMines(expectedLimpets int) {
	var actualLimpets int
	err := a.db.QueryRow("SELECT COALESCE(limpets, 0) FROM player_stats WHERE id = 1").Scan(&actualLimpets)
	if err != nil {
		a.t.Fatalf("Failed to get player limpet mines: %v", err)
	}
	if actualLimpets != expectedLimpets {
		a.t.Errorf("Expected player limpet mines to be %d, got %d", expectedLimpets, actualLimpets)
	}
}

// AssertPlayerGenesisDevices verifies that player has the expected number of genesis devices
func (a *DBAsserts) AssertPlayerGenesisDevices(expectedGenTorps int) {
	var actualGenTorps int
	err := a.db.QueryRow("SELECT COALESCE(gen_torps, 0) FROM player_stats WHERE id = 1").Scan(&actualGenTorps)
	if err != nil {
		a.t.Fatalf("Failed to get player genesis devices: %v", err)
	}
	if actualGenTorps != expectedGenTorps {
		a.t.Errorf("Expected player genesis devices to be %d, got %d", expectedGenTorps, actualGenTorps)
	}
}

// AssertPlayerCloaks verifies that player has the expected number of cloaks
func (a *DBAsserts) AssertPlayerCloaks(expectedCloaks int) {
	var actualCloaks int
	err := a.db.QueryRow("SELECT COALESCE(cloaks, 0) FROM player_stats WHERE id = 1").Scan(&actualCloaks)
	if err != nil {
		a.t.Fatalf("Failed to get player cloaks: %v", err)
	}
	if actualCloaks != expectedCloaks {
		a.t.Errorf("Expected player cloaks to be %d, got %d", expectedCloaks, actualCloaks)
	}
}

// AssertPlayerBeacons verifies that player has the expected number of beacons
func (a *DBAsserts) AssertPlayerBeacons(expectedBeacons int) {
	var actualBeacons int
	err := a.db.QueryRow("SELECT COALESCE(beacons, 0) FROM player_stats WHERE id = 1").Scan(&actualBeacons)
	if err != nil {
		a.t.Fatalf("Failed to get player beacons: %v", err)
	}
	if actualBeacons != expectedBeacons {
		a.t.Errorf("Expected player beacons to be %d, got %d", expectedBeacons, actualBeacons)
	}
}

// AssertPlayerAtomicDetonators verifies that player has the expected number of atomic detonators
func (a *DBAsserts) AssertPlayerAtomicDetonators(expectedAtomics int) {
	var actualAtomics int
	err := a.db.QueryRow("SELECT COALESCE(atomics, 0) FROM player_stats WHERE id = 1").Scan(&actualAtomics)
	if err != nil {
		a.t.Fatalf("Failed to get player atomic detonators: %v", err)
	}
	if actualAtomics != expectedAtomics {
		a.t.Errorf("Expected player atomic detonators to be %d, got %d", expectedAtomics, actualAtomics)
	}
}

// AssertPlayerCarbonite verifies that player has the expected amount of carbonite
func (a *DBAsserts) AssertPlayerCarbonite(expectedCorbomite int) {
	var actualCorbomite int
	err := a.db.QueryRow("SELECT COALESCE(corbomite, 0) FROM player_stats WHERE id = 1").Scan(&actualCorbomite)
	if err != nil {
		a.t.Fatalf("Failed to get player carbonite: %v", err)
	}
	if actualCorbomite != expectedCorbomite {
		a.t.Errorf("Expected player carbonite to be %d, got %d", expectedCorbomite, actualCorbomite)
	}
}

// AssertPlayerEtherProbes verifies that player has the expected number of ether probes
func (a *DBAsserts) AssertPlayerEtherProbes(expectedEprobes int) {
	var actualEprobes int
	err := a.db.QueryRow("SELECT COALESCE(eprobes, 0) FROM player_stats WHERE id = 1").Scan(&actualEprobes)
	if err != nil {
		a.t.Fatalf("Failed to get player ether probes: %v", err)
	}
	if actualEprobes != expectedEprobes {
		a.t.Errorf("Expected player ether probes to be %d, got %d", expectedEprobes, actualEprobes)
	}
}

// AssertPlayerMineDisruptors verifies that player has the expected number of mine disruptors
func (a *DBAsserts) AssertPlayerMineDisruptors(expectedMineDisr int) {
	var actualMineDisr int
	err := a.db.QueryRow("SELECT COALESCE(mine_disr, 0) FROM player_stats WHERE id = 1").Scan(&actualMineDisr)
	if err != nil {
		a.t.Fatalf("Failed to get player mine disruptors: %v", err)
	}
	if actualMineDisr != expectedMineDisr {
		a.t.Errorf("Expected player mine disruptors to be %d, got %d", expectedMineDisr, actualMineDisr)
	}
}

// AssertPlayerAlignment verifies that player has the expected alignment
func (a *DBAsserts) AssertPlayerAlignment(expectedAlignment int) {
	var actualAlignment int
	err := a.db.QueryRow("SELECT COALESCE(alignment, 0) FROM player_stats WHERE id = 1").Scan(&actualAlignment)
	if err != nil {
		a.t.Fatalf("Failed to get player alignment: %v", err)
	}
	if actualAlignment != expectedAlignment {
		a.t.Errorf("Expected player alignment to be %d, got %d", expectedAlignment, actualAlignment)
	}
}

// AssertPlayerShipNumber verifies that player has the expected ship number
func (a *DBAsserts) AssertPlayerShipNumber(expectedShipNumber int) {
	var actualShipNumber int
	err := a.db.QueryRow("SELECT COALESCE(ship_number, 0) FROM player_stats WHERE id = 1").Scan(&actualShipNumber)
	if err != nil {
		a.t.Fatalf("Failed to get player ship number: %v", err)
	}
	if actualShipNumber != expectedShipNumber {
		a.t.Errorf("Expected player ship number to be %d, got %d", expectedShipNumber, actualShipNumber)
	}
}
