package game

import (
	"sync"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/proxy/database"
)

// StateManager handles game state changes and coordinates with the TuiAPI
type StateManager struct {
	mu           sync.RWMutex
	currentSector int
	playerName    string
	tuiAPI        api.TuiAPI
	db            database.Database
}

// NewStateManager creates a new game state manager
func NewStateManager(tuiAPI api.TuiAPI, db database.Database) *StateManager {
	return &StateManager{
		tuiAPI: tuiAPI,
		db:     db,
	}
}

// convertSectorToSectorInfo converts database TSector to API SectorInfo
func (sm *StateManager) convertSectorToSectorInfo(sector database.TSector, sectorNum int) api.SectorInfo {
	// Convert warp array to slice, filtering out zero values
	var warps []int
	for _, warp := range sector.Warp {
		if warp != 0 {
			warps = append(warps, warp)
		}
	}
	
	// Count traders
	hasTraders := len(sector.Traders)
	
	return api.SectorInfo{
		Number:        sectorNum,
		NavHaz:        sector.NavHaz,
		HasTraders:    hasTraders,
		Constellation: sector.Constellation,
		Beacon:        sector.Beacon,
		Warps:         warps,
	}
}

// SetCurrentSector updates the current sector and notifies the TuiAPI
func (sm *StateManager) SetCurrentSector(sectorNum int) {
	sm.mu.Lock()
	oldSector := sm.currentSector
	sm.currentSector = sectorNum
	sm.mu.Unlock()
	
	// Only notify if sector actually changed
	if oldSector != sectorNum {
		
		// Load complete sector information from database
		var sectorInfo api.SectorInfo
		if sm.db != nil && sm.db.GetDatabaseOpen() {
			if sector, err := sm.db.LoadSector(sectorNum); err == nil {
				sectorInfo = sm.convertSectorToSectorInfo(sector, sectorNum)
			} else {
				// Fallback to basic sector info with just the number
				sectorInfo = api.SectorInfo{Number: sectorNum}
			}
		} else {
			// No database available, provide basic sector info
			sectorInfo = api.SectorInfo{Number: sectorNum}
		}
		
		debug.Log("STATE_MANAGER: Firing OnCurrentSectorChanged for sector %d (oldSector=%d) [SOURCE: SetCurrentSector]", sectorNum, oldSector)
		sm.tuiAPI.OnCurrentSectorChanged(sectorInfo)
	}
}

// SetPlayerName updates the player name
func (sm *StateManager) SetPlayerName(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.playerName = name
}

// GetCurrentSector returns the current sector (thread-safe)
func (sm *StateManager) GetCurrentSector() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentSector
}

// GetPlayerName returns the player name (thread-safe)
func (sm *StateManager) GetPlayerName() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.playerName
}