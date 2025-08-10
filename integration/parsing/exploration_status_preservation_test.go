package parsing

import (
	"testing"
	"twist/integration/scripting"
)


// TestExplorationStatusPreservation tests that visited sectors maintain EtHolo status
// even when density scan information is processed afterward
func TestExplorationStatusPreservation(t *testing.T) {
	// Create expect-based test framework
	bridge := scripting.NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	// Server script sends sector data and density scan
	serverScript := `
log "Server: Starting exploration status preservation test"
timeout "10s"

send "Trade Wars 2002*"
send "Enter your login name: "
expect "testuser"

# Step 1: Send sector visit data (should set to EtHolo)
send "Sector  : 2921 in uncharted space.*"
send "Warps to Sector(s) :  3212 - 7656*"
send "*"
send "Command [TL=00:00:00]:[2921] (?=Help)? : "

# Step 2: Send density scan data for same sector
send "                          Relative Density Scan*"
send "Sector  2921  ==>           1500  Warps : 2    NavHaz :     0%    Anom : No*"
`

	// Client script just sends username
	clientScript := `
log "Client: Starting client script"
timeout "10s"
send "testuser"
`

	// Run scripts with automatic synchronization and get opened database
	db, err := bridge.RunSyncedScripts(serverScript, clientScript)
	if err != nil {
		t.Fatalf("Failed to run synced scripts: %v", err)
	}
	defer db.Close()

	// Query sector data
	var explored, density, warps int
	err = db.QueryRow("SELECT explored, density, warps FROM sectors WHERE sector_index = ?", 2921).Scan(&explored, &density, &warps)
	if err != nil {
		t.Fatalf("Failed to query sector 2921: %v", err)
	}

	// Verify exploration status preserved (EtHolo = 3)
	if explored != 3 {
		t.Errorf("After density scan, expected exploration status to remain EtHolo (3), but got %d", explored)
	}

	// Verify density information was processed
	if density != 1500 {
		t.Errorf("Expected density to be updated to 1500, got %d", density)
	}

	if warps != 2 {
		t.Errorf("Expected warps count to be updated to 2, got %d", warps)
	}
}

// TestExplorationStatusTransitionFromNoToHolo tests that unvisited sectors
// can still be properly upgraded from EtNo to EtHolo when visited
func TestExplorationStatusTransitionFromNoToHolo(t *testing.T) {
	bridge := scripting.NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	serverScript := `
log "Server: Testing EtNo to EtHolo transition"
timeout "10s"

send "Trade Wars 2002*"
send "Enter your login name: "
expect "testuser"

# Step 1: Process density scan first (should set to EtDensity)
send "                          Relative Density Scan*"
send "Sector  3212  ==>           2000  Warps : 4    NavHaz :     0%    Anom : No*"

# Step 2: Visit the sector (should upgrade to EtHolo)
send "Sector  : 3212 in uncharted space.*"
send "Warps to Sector(s) :  2921 - 10870 - (16983) - (17563)*"
send "*"
send "Command [TL=00:00:00]:[3212] (?=Help)? : "

# Step 3: Process another density scan (should NOT downgrade back to EtDensity)
send "                          Relative Density Scan*"
send "Sector  3212  ==>           2000  Warps : 4    NavHaz :     0%    Anom : No*"
`

	clientScript := `
log "Client: Starting transition test"
timeout "10s"
send "testuser"
`

	db, err := bridge.RunSyncedScripts(serverScript, clientScript)
	if err != nil {
		t.Fatalf("Failed to run synced scripts: %v", err)
	}
	defer db.Close()

	var explored int
	err = db.QueryRow("SELECT explored FROM sectors WHERE sector_index = ?", 3212).Scan(&explored)
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
	bridge := scripting.NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	serverScript := `
log "Server: Testing unvisited sector density scan"
timeout "10s"

send "Trade Wars 2002*"
send "Enter your login name: "
expect "testuser"

# Process density scan for unvisited sector (should set to EtDensity)
send "                          Relative Density Scan*"
send "Sector  7656  ==>            800  Warps : 3    NavHaz :     0%    Anom : Yes*"
`

	clientScript := `
log "Client: Starting unvisited sector test"
timeout "10s"
send "testuser"
`

	db, err := bridge.RunSyncedScripts(serverScript, clientScript)
	if err != nil {
		t.Fatalf("Failed to run synced scripts: %v", err)
	}
	defer db.Close()

	var explored, density, warps int
	var anomaly bool
	var constellation string
	err = db.QueryRow("SELECT explored, density, warps, anomaly, constellation FROM sectors WHERE sector_index = ?", 7656).Scan(&explored, &density, &warps, &anomaly, &constellation)
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