package parsing

import (
	"testing"
	"twist/integration/scripting"
	"twist/internal/api"
	_ "github.com/mattn/go-sqlite3"
)


// TestExplorationStatusPreservation tests that visited sectors maintain EtHolo status
// even when density scan information is processed afterward
func TestExplorationStatusPreservation(t *testing.T) {
	// Server script sends sector data and density scan
	serverScript := `send "Sector  : 2921 in uncharted space.*"
send "Warps to Sector(s) :  3212 - 7656*"
send "*"
send "Command [TL=00:00:00]:[2921] (?=Help)? : "
send "                          Relative Density Scan*"
send "Sector  2921  ==>           1500  Warps : 2    NavHaz :     0%    Anom : No*"`

	// Client script expects sector data and density scan
	clientScript := `expect "Sector  : 2921"
expect "Warps to Sector"
expect "Command"
expect "Relative Density Scan"
expect "Sector  2921  ==>"` 

	// Use ConnectOptions with database path to enable game context
	dbPath := t.TempDir() + "/test.db"
	connectOpts := &api.ConnectOptions{DatabasePath: dbPath}
	
	result := scripting.Execute(t, serverScript, clientScript, connectOpts)

	// Verify game context was created
	if result.Database == nil {
		t.Fatal("Expected database instance")
	}
	defer result.Database.Close()

	// Query sector data
	var explored, density, warps int
	err := result.Database.QueryRow("SELECT explored, density, warps FROM sectors WHERE sector_index = ?", 2921).Scan(&explored, &density, &warps)
	if err != nil {
		t.Fatalf("Failed to query sector 2921: %v", err)
	}

	// Verify exploration status preserved (EtHolo = 3)
	if explored != 3 {
		t.Errorf("After density scan, expected exploration status to remain EtHolo (3), but got %d", explored)
	}

	// Verify sector basic parsing worked (warps should be parsed from sector display)
	if warps != 2 {
		t.Errorf("Expected warps count to be updated to 2, got %d", warps)
	}

	// Verify density information was processed (fixed parsing to match Pascal TWX implementation)
	if density != 1500 {
		t.Errorf("Expected density to be updated to 1500, got %d", density)
	}
}

// TestExplorationStatusTransitionFromNoToHolo tests that unvisited sectors
// can still be properly upgraded from EtNo to EtHolo when visited
func TestExplorationStatusTransitionFromNoToHolo(t *testing.T) {
	// Server script processes density scan first, then visits sector, then another density scan
	serverScript := `send "                          Relative Density Scan*"
send "Sector  3212  ==>           2000  Warps : 4    NavHaz :     0%    Anom : No*"
send "Sector  : 3212 in uncharted space.*"
send "Warps to Sector(s) :  2921 - 10870 - (16983) - (17563)*"
send "*"
send "Command [TL=00:00:00]:[3212] (?=Help)? : "
send "                          Relative Density Scan*"
send "Sector  3212  ==>           2000  Warps : 4    NavHaz :     0%    Anom : No*"`

	// Client script expects all the data
	clientScript := `expect "Relative Density Scan"
expect "Sector  3212  ==>"
expect "Sector  : 3212"
expect "Warps to Sector"
expect "Command"
expect "Relative Density Scan"
expect "Sector  3212  ==>"` 

	// Use ConnectOptions with database path to enable game context
	dbPath := t.TempDir() + "/test.db"
	connectOpts := &api.ConnectOptions{DatabasePath: dbPath}
	
	result := scripting.Execute(t, serverScript, clientScript, connectOpts)

	// Verify game context was created
	if result.Database == nil {
		t.Fatal("Expected database instance")
	}
	defer result.Database.Close()

	var explored int
	err := result.Database.QueryRow("SELECT explored FROM sectors WHERE sector_index = ?", 3212).Scan(&explored)
	if err != nil {
		t.Fatalf("Failed to query sector 3212: %v", err)
	}

	// Should remain EtHolo (3) after second density scan
	if explored != 3 {
		t.Errorf("After second density scan, expected exploration status to remain EtHolo (3), but got %d", explored)
	}
}

// TestExplorationStatusUnvisitedSectors tests that unvisited sectors are still
// properly handled by density scans
func TestExplorationStatusUnvisitedSectors(t *testing.T) {
	// Server script processes density scan for unvisited sector
	serverScript := `send "                          Relative Density Scan*"
send "Sector  7656  ==>            800  Warps : 3    NavHaz :     0%    Anom : Yes*"`

	// Client script expects density scan
	clientScript := `expect "Relative Density Scan"
expect "Sector  7656  ==>"` 

	// Use ConnectOptions with database path to enable game context
	dbPath := t.TempDir() + "/test.db"
	connectOpts := &api.ConnectOptions{DatabasePath: dbPath}
	
	result := scripting.Execute(t, serverScript, clientScript, connectOpts)

	// Verify game context was created
	if result.Database == nil {
		t.Fatal("Expected database instance")
	}
	defer result.Database.Close()

	var explored, density, warps int
	var anomaly bool
	var constellation string
	err := result.Database.QueryRow("SELECT explored, density, warps, anomaly, constellation FROM sectors WHERE sector_index = ?", 7656).Scan(&explored, &density, &warps, &anomaly, &constellation)
	if err != nil {
		t.Fatalf("Failed to query sector 7656: %v", err)
	}

	// Verify it was set to EtDensity (2)
	if explored != 2 {
		t.Errorf("Expected unvisited sector to be set to EtDensity (2), got %d", explored)
	}

	// Verify density and other data was properly set
	if density != 800 {
		t.Errorf("Expected density 800, got %d", density)
	}

	if warps != 3 {
		t.Errorf("Expected warps count 3, got %d", warps)
	}

	if !anomaly {
		t.Error("Expected anomaly to be true, got false")
	}

	if constellation != "??? (Density only)" {
		t.Errorf("Expected constellation to be '??? (Density only)', got '%s'", constellation)
	}
}