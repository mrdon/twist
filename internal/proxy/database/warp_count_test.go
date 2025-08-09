package database

import (
	"testing"
)

func TestWarpCountInterceptor(t *testing.T) {
	// Test automatic warp count updates when saving/loading sectors
	
	t.Run("SaveSector automatically updates warp count", func(t *testing.T) {
		db := NewDatabase()
		if err := db.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.CloseDatabase()

		// Create sector with warp connections
		sector := NULLSector()
		sector.Warp[0] = 123
		sector.Warp[2] = 456
		sector.Warp[4] = 789
		sector.Constellation = "Test Space"
		
		// Warp count should be automatically calculated when saved
		if err := db.SaveSector(sector, 100); err != nil {
			t.Fatalf("Failed to save sector: %v", err)
		}
		
		// Load sector and verify warp count
		loadedSector, err := db.LoadSector(100)
		if err != nil {
			t.Fatalf("Failed to load sector: %v", err)
		}
		
		if loadedSector.Warps != 3 {
			t.Errorf("Expected warp count 3, got %d", loadedSector.Warps)
		}
		
		// Verify warp array is preserved
		expectedWarps := [6]int{123, 0, 456, 0, 789, 0}
		if loadedSector.Warp != expectedWarps {
			t.Errorf("Warp array mismatch. Expected %v, got %v", expectedWarps, loadedSector.Warp)
		}
	})
	
	t.Run("LoadSector calculates warp count from array", func(t *testing.T) {
		db := NewDatabase()
		if err := db.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.CloseDatabase()

		// Create sector with warp connections
		sector := NULLSector()
		sector.Warp[1] = 111
		sector.Warp[3] = 333
		sector.Constellation = "Auto Count Test"
		
		if err := db.SaveSector(sector, 200); err != nil {
			t.Fatalf("Failed to save sector: %v", err)
		}
		
		// Load sector - should auto-calculate count
		loadedSector, err := db.LoadSector(200)
		if err != nil {
			t.Fatalf("Failed to load sector: %v", err)
		}
		
		if loadedSector.Warps != 2 {
			t.Errorf("Expected auto-calculated warp count 2, got %d", loadedSector.Warps)
		}
	})
	
	t.Run("Density scan counts preserved when no warp array", func(t *testing.T) {
		db := NewDatabase()
		if err := db.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.CloseDatabase()

		// Create sector with density scan count but no actual warps
		sector := NULLSector()
		sector.Warp = [6]int{0, 0, 0, 0, 0, 0} // No connections
		sector.Warps = 5 // From density scan
		sector.Constellation = "Density Space"
		sector.Explored = EtDensity
		
		if err := db.SaveSector(sector, 300); err != nil {
			t.Fatalf("Failed to save sector: %v", err)
		}
		
		loadedSector, err := db.LoadSector(300)
		if err != nil {
			t.Fatalf("Failed to load sector: %v", err)
		}
		
		// Should preserve density scan count since no actual warp connections
		if loadedSector.Warps != 5 {
			t.Errorf("Expected preserved density count 5, got %d", loadedSector.Warps)
		}
	})
	
	t.Run("Warp array overrides stored count", func(t *testing.T) {
		db := NewDatabase()
		if err := db.CreateDatabase(":memory:"); err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.CloseDatabase()

		// Create sector with both warp connections AND a stored count
		sector := NULLSector()
		sector.Warp[0] = 111
		sector.Warp[1] = 222
		sector.Warps = 99 // Old/wrong count
		sector.Constellation = "Override Test"
		
		if err := db.SaveSector(sector, 400); err != nil {
			t.Fatalf("Failed to save sector: %v", err)
		}
		
		loadedSector, err := db.LoadSector(400)
		if err != nil {
			t.Fatalf("Failed to load sector: %v", err)
		}
		
		// Warp array should be authoritative, overriding stored count
		if loadedSector.Warps != 2 {
			t.Errorf("Expected warp array to override stored count, got %d instead of 2", loadedSector.Warps)
		}
	})
}

func TestWarpHelperMethods(t *testing.T) {
	db := NewDatabase()
	
	t.Run("SetSectorWarp updates count automatically", func(t *testing.T) {
		sector := NULLSector()
		
		// Add warps using helper
		db.SetSectorWarp(&sector, 0, 100)
		db.SetSectorWarp(&sector, 2, 200)
		
		if sector.Warps != 2 {
			t.Errorf("Expected helper to set count to 2, got %d", sector.Warps)
		}
		
		expectedWarps := [6]int{100, 0, 200, 0, 0, 0}
		if sector.Warp != expectedWarps {
			t.Errorf("Warp array mismatch. Expected %v, got %v", expectedWarps, sector.Warp)
		}
	})
	
	t.Run("ClearSectorWarp updates count automatically", func(t *testing.T) {
		sector := NULLSector()
		sector.Warp = [6]int{100, 200, 300, 0, 0, 0}
		sector.Warps = 3
		
		// Clear one warp
		db.ClearSectorWarp(&sector, 1)
		
		if sector.Warps != 2 {
			t.Errorf("Expected count to decrease to 2, got %d", sector.Warps)
		}
		
		expectedWarps := [6]int{100, 0, 300, 0, 0, 0}
		if sector.Warp != expectedWarps {
			t.Errorf("Warp array mismatch after clear. Expected %v, got %v", expectedWarps, sector.Warp)
		}
	})
	
	t.Run("SetSectorWarps updates entire array and count", func(t *testing.T) {
		sector := NULLSector()
		newWarps := [6]int{10, 20, 30, 40, 0, 0}
		
		db.SetSectorWarps(&sector, newWarps)
		
		if sector.Warps != 4 {
			t.Errorf("Expected count to be set to 4, got %d", sector.Warps)
		}
		
		if sector.Warp != newWarps {
			t.Errorf("Warp array not set correctly. Expected %v, got %v", newWarps, sector.Warp)
		}
	})
	
	t.Run("UpdateWarpCount utility function", func(t *testing.T) {
		sector := NULLSector()
		sector.Warp = [6]int{1, 2, 0, 4, 5, 0}
		sector.Warps = 999 // Wrong count
		
		UpdateWarpCount(&sector)
		
		if sector.Warps != 4 {
			t.Errorf("UpdateWarpCount should set count to 4, got %d", sector.Warps)
		}
	})
	
	t.Run("UpdateWarpCount preserves density scan counts", func(t *testing.T) {
		sector := NULLSector()
		sector.Warp = [6]int{0, 0, 0, 0, 0, 0} // No connections
		sector.Warps = 3 // From density scan
		sector.Explored = EtDensity
		
		UpdateWarpCount(&sector)
		
		// Should preserve existing count when no warp connections
		if sector.Warps != 3 {
			t.Errorf("UpdateWarpCount should preserve density count 3, got %d", sector.Warps)
		}
	})
}