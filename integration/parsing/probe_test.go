package parsing

import (
	"testing"
	"twist/integration/scripting"
	"twist/internal/api"
)

// TestProbeDataParsing demonstrates probe data parsing and database storage
func TestProbeDataParsing(t *testing.T) {
	// Use ConnectOptions with DatabasePath to enable database storage
	connectOpts := &api.ConnectOptions{DatabasePath: t.TempDir() + "/test.db"}

	// Execute test using script file
	result := scripting.ExecuteScriptFile(t, "probe_test.script", connectOpts)

	// Verify all sectors from the probe path were parsed and saved
	result.Assert.AssertSectorExists(274) // First sector probed
	result.Assert.AssertSectorExists(510) // Sector with Aachen port
	result.Assert.AssertSectorExists(493) // Final sector where probe self-destructed
	
	// Debug: Check what sectors actually exist in the database
	rows, err := result.Database.Query("SELECT sector_index, warp1, warp2, warp3 FROM sectors ORDER BY sector_index")
	if err != nil {
		t.Logf("Failed to query sectors: %v", err)
	} else {
		defer rows.Close()
		t.Logf("=== SECTORS IN DATABASE ===")
		for rows.Next() {
			var sector, w1, w2, w3 int
			rows.Scan(&sector, &w1, &w2, &w3)
			t.Logf("Sector %d: warps=[%d, %d, %d, ...]", sector, w1, w2, w3)
		}
		t.Logf("=== END SECTORS ===")
	}

	// Verify sector constellations
	result.Assert.AssertSectorConstellation(274, "uncharted space")
	result.Assert.AssertSectorConstellation(510, "uncharted space")
	result.Assert.AssertSectorConstellation(493, "uncharted space")

	// Verify ports were parsed correctly
	result.Assert.AssertPortExists(274, "Nerialt Annex", 7) // SSS port
	result.Assert.AssertPortExists(510, "Aachen", 2)        // BSB port

	// Verify probe movement created correct warp connections
	// Based on actual probe path: 190 -> 274 -> 510 -> 493
	result.Assert.AssertSectorWithWarps(190, []int{274}) // Should have warp to first probed sector
	result.Assert.AssertSectorWithWarps(274, []int{510}) // Should have warp to next sector in path
	result.Assert.AssertSectorWithWarps(510, []int{493}) // Should have warp to final sector
}
