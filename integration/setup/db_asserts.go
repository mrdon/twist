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
