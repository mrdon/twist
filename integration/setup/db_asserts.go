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
