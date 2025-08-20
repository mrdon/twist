package streaming

import (
	"database/sql"
	"twist/internal/debug"
	"github.com/Masterminds/squirrel"
)

// PlayerStatsTracker tracks discovered player stat fields during parsing
// Uses discovered field tracking - only updates fields that were actually parsed
type PlayerStatsTracker struct {
	updates map[string]interface{}
}

// NewPlayerStatsTracker creates a new player stats tracker
func NewPlayerStatsTracker() *PlayerStatsTracker {
	return &PlayerStatsTracker{
		updates: make(map[string]interface{}),
	}
}

// SetTurns records that turns field was discovered during parsing
func (p *PlayerStatsTracker) SetTurns(turns int) *PlayerStatsTracker {
	p.updates[ColPlayerTurns] = turns
	return p
}

// SetCredits records that credits field was discovered during parsing
func (p *PlayerStatsTracker) SetCredits(credits int) *PlayerStatsTracker {
	p.updates[ColPlayerCredits] = credits
	return p
}

// SetFighters records that fighters field was discovered during parsing
func (p *PlayerStatsTracker) SetFighters(fighters int) *PlayerStatsTracker {
	p.updates[ColPlayerFighters] = fighters
	return p
}

// SetShields records that shields field was discovered during parsing
func (p *PlayerStatsTracker) SetShields(shields int) *PlayerStatsTracker {
	p.updates[ColPlayerShields] = shields
	return p
}

// SetTotalHolds records that total_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetTotalHolds(totalHolds int) *PlayerStatsTracker {
	p.updates[ColPlayerTotalHolds] = totalHolds
	return p
}

// SetOreHolds records that ore_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetOreHolds(oreHolds int) *PlayerStatsTracker {
	p.updates[ColPlayerOreHolds] = oreHolds
	return p
}

// SetOrgHolds records that org_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetOrgHolds(orgHolds int) *PlayerStatsTracker {
	p.updates[ColPlayerOrgHolds] = orgHolds
	return p
}

// SetEquHolds records that equ_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetEquHolds(equHolds int) *PlayerStatsTracker {
	p.updates[ColPlayerEquHolds] = equHolds
	return p
}

// SetColHolds records that col_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetColHolds(colHolds int) *PlayerStatsTracker {
	p.updates[ColPlayerColHolds] = colHolds
	return p
}

// SetPhotons records that photons field was discovered during parsing
func (p *PlayerStatsTracker) SetPhotons(photons int) *PlayerStatsTracker {
	p.updates[ColPlayerPhotons] = photons
	return p
}

// SetArmids records that armids field was discovered during parsing
func (p *PlayerStatsTracker) SetArmids(armids int) *PlayerStatsTracker {
	p.updates[ColPlayerArmids] = armids
	return p
}

// SetLimpets records that limpets field was discovered during parsing
func (p *PlayerStatsTracker) SetLimpets(limpets int) *PlayerStatsTracker {
	p.updates[ColPlayerLimpets] = limpets
	return p
}

// SetGenTorps records that gen_torps field was discovered during parsing
func (p *PlayerStatsTracker) SetGenTorps(genTorps int) *PlayerStatsTracker {
	p.updates[ColPlayerGenTorps] = genTorps
	return p
}

// SetTwarpType records that twarp_type field was discovered during parsing
func (p *PlayerStatsTracker) SetTwarpType(twarpType int) *PlayerStatsTracker {
	p.updates[ColPlayerTwarpType] = twarpType
	return p
}

// SetCloaks records that cloaks field was discovered during parsing
func (p *PlayerStatsTracker) SetCloaks(cloaks int) *PlayerStatsTracker {
	p.updates[ColPlayerCloaks] = cloaks
	return p
}

// SetBeacons records that beacons field was discovered during parsing
func (p *PlayerStatsTracker) SetBeacons(beacons int) *PlayerStatsTracker {
	p.updates[ColPlayerBeacons] = beacons
	return p
}

// SetAtomics records that atomics field was discovered during parsing
func (p *PlayerStatsTracker) SetAtomics(atomics int) *PlayerStatsTracker {
	p.updates[ColPlayerAtomics] = atomics
	return p
}

// SetCorbomite records that corbomite field was discovered during parsing
func (p *PlayerStatsTracker) SetCorbomite(corbomite int) *PlayerStatsTracker {
	p.updates[ColPlayerCorbomite] = corbomite
	return p
}

// SetEprobes records that eprobes field was discovered during parsing
func (p *PlayerStatsTracker) SetEprobes(eprobes int) *PlayerStatsTracker {
	p.updates[ColPlayerEprobes] = eprobes
	return p
}

// SetMineDisr records that mine_disr field was discovered during parsing
func (p *PlayerStatsTracker) SetMineDisr(mineDisr int) *PlayerStatsTracker {
	p.updates[ColPlayerMineDisr] = mineDisr
	return p
}

// SetAlignment records that alignment field was discovered during parsing
func (p *PlayerStatsTracker) SetAlignment(alignment int) *PlayerStatsTracker {
	p.updates[ColPlayerAlignment] = alignment
	return p
}

// SetExperience records that experience field was discovered during parsing
func (p *PlayerStatsTracker) SetExperience(experience int) *PlayerStatsTracker {
	p.updates[ColPlayerExperience] = experience
	return p
}

// SetCorp records that corp field was discovered during parsing
func (p *PlayerStatsTracker) SetCorp(corp int) *PlayerStatsTracker {
	p.updates[ColPlayerCorp] = corp
	return p
}

// SetShipNumber records that ship_number field was discovered during parsing
func (p *PlayerStatsTracker) SetShipNumber(shipNumber int) *PlayerStatsTracker {
	p.updates[ColPlayerShipNumber] = shipNumber
	return p
}

// SetPsychicProbe records that psychic_probe field was discovered during parsing
func (p *PlayerStatsTracker) SetPsychicProbe(psychicProbe bool) *PlayerStatsTracker {
	p.updates[ColPlayerPsychicProbe] = psychicProbe
	return p
}

// SetPlanetScanner records that planet_scanner field was discovered during parsing
func (p *PlayerStatsTracker) SetPlanetScanner(planetScanner bool) *PlayerStatsTracker {
	p.updates[ColPlayerPlanetScanner] = planetScanner
	return p
}

// SetScanType records that scan_type field was discovered during parsing
func (p *PlayerStatsTracker) SetScanType(scanType int) *PlayerStatsTracker {
	p.updates[ColPlayerScanType] = scanType
	return p
}

// SetShipClass records that ship_class field was discovered during parsing
func (p *PlayerStatsTracker) SetShipClass(shipClass string) *PlayerStatsTracker {
	p.updates[ColPlayerShipClass] = shipClass
	return p
}

// SetCurrentSector records that current_sector field was discovered during parsing
func (p *PlayerStatsTracker) SetCurrentSector(currentSector int) *PlayerStatsTracker {
	p.updates[ColPlayerCurrentSector] = currentSector
	return p
}

// SetPlayerName records that player_name field was discovered during parsing
func (p *PlayerStatsTracker) SetPlayerName(playerName string) *PlayerStatsTracker {
	p.updates[ColPlayerPlayerName] = playerName
	return p
}

// HasUpdates returns true if any fields were discovered during parsing
func (p *PlayerStatsTracker) HasUpdates() bool {
	return len(p.updates) > 0
}

// Execute writes discovered fields to database using Squirrel query builder
// Only fields that were actually parsed/discovered are updated
func (p *PlayerStatsTracker) Execute(db *sql.DB) error {
	if len(p.updates) == 0 {
		return nil // No updates to perform
	}
	
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	
	// Ensure player_stats record exists (single row table with id=1)
	_, err := db.Exec("INSERT OR IGNORE INTO player_stats (id) VALUES (1)")
	if err != nil {
		debug.Log("Failed to ensure player_stats record exists: %v", err)
		return err
	}
	
	// Build dynamic UPDATE query with only discovered fields
	query := psql.Update("player_stats").
		SetMap(p.updates).
		Set("updated_at", "CURRENT_TIMESTAMP").
		Where(squirrel.Eq{"id": 1})
	
	sql, args, err := query.ToSql()
	if err != nil {
		debug.Log("Failed to build player stats update query: %v", err)
		return err
	}
	
	debug.Log("Executing player stats update with %d discovered fields: %s", len(p.updates), sql)
	
	_, err = db.Exec(sql, args...)
	if err != nil {
		debug.Log("Failed to execute player stats update: %v", err)
		return err
	}
	
	debug.Log("Successfully updated player stats with discovered fields: %v", getFieldNames(p.updates))
	return nil
}

// getFieldNames extracts field names from updates map for logging
func getFieldNames(updates map[string]interface{}) []string {
	fields := make([]string, 0, len(updates))
	for field := range updates {
		fields = append(fields, field)
	}
	return fields
}

// SectorTracker tracks discovered sector field updates during parsing
// Uses discovered field tracking - only updates fields that were actually parsed
type SectorTracker struct {
	sectorIndex int
	updates     map[string]interface{}
}

// NewSectorTracker creates a new sector tracker for the given sector
func NewSectorTracker(sectorIndex int) *SectorTracker {
	return &SectorTracker{
		sectorIndex: sectorIndex,
		updates:     make(map[string]interface{}),
	}
}

// SetConstellation records that constellation field was discovered during parsing
func (s *SectorTracker) SetConstellation(constellation string) *SectorTracker {
	s.updates[ColSectorConstellation] = constellation
	return s
}

// SetBeacon records that beacon field was discovered during parsing
func (s *SectorTracker) SetBeacon(beacon string) *SectorTracker {
	s.updates[ColSectorBeacon] = beacon
	return s
}

// SetNavHaz records that nav_haz field was discovered during parsing
func (s *SectorTracker) SetNavHaz(navHaz int) *SectorTracker {
	s.updates[ColSectorNavHaz] = navHaz
	return s
}

// SetWarps records that warp fields were discovered during parsing
func (s *SectorTracker) SetWarps(warps [6]int) *SectorTracker {
	s.updates[ColSectorWarp1] = warps[0]
	s.updates[ColSectorWarp2] = warps[1]
	s.updates[ColSectorWarp3] = warps[2]
	s.updates[ColSectorWarp4] = warps[3]
	s.updates[ColSectorWarp5] = warps[4]
	s.updates[ColSectorWarp6] = warps[5]
	
	// Count non-zero warps for the warps field
	warpCount := 0
	for _, warp := range warps {
		if warp > 0 {
			warpCount++
		}
	}
	s.updates[ColSectorWarps] = warpCount
	
	return s
}

// SetDensity records that density field was discovered during parsing
func (s *SectorTracker) SetDensity(density int) *SectorTracker {
	s.updates[ColSectorDensity] = density
	return s
}

// SetAnomaly records that anomaly field was discovered during parsing
func (s *SectorTracker) SetAnomaly(anomaly bool) *SectorTracker {
	s.updates[ColSectorAnomaly] = anomaly
	return s
}

// SetExplored records that explored field was discovered during parsing
func (s *SectorTracker) SetExplored(explored int) *SectorTracker {
	s.updates[ColSectorExplored] = explored
	return s
}

// HasUpdates returns true if any fields were discovered during parsing
func (s *SectorTracker) HasUpdates() bool {
	return len(s.updates) > 0
}

// Execute writes discovered fields to database using Squirrel query builder
// Only fields that were actually parsed/discovered are updated
func (s *SectorTracker) Execute(db *sql.DB) error {
	if len(s.updates) == 0 {
		return nil // No updates to perform
	}
	
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	
	// Ensure sector record exists (UPSERT pattern)
	_, err := db.Exec("INSERT OR IGNORE INTO sectors (sector_index) VALUES (?)", s.sectorIndex)
	if err != nil {
		debug.Log("Failed to ensure sector record exists for sector %d: %v", s.sectorIndex, err)
		return err
	}
	
	// Build dynamic UPDATE query with only discovered fields
	query := psql.Update("sectors").
		SetMap(s.updates).
		Set("update_time", "CURRENT_TIMESTAMP").
		Where(squirrel.Eq{"sector_index": s.sectorIndex})
	
	sql, args, err := query.ToSql()
	if err != nil {
		debug.Log("Failed to build sector update query for sector %d: %v", s.sectorIndex, err)
		return err
	}
	
	debug.Log("Executing sector update for sector %d with %d discovered fields: %s", s.sectorIndex, len(s.updates), sql)
	
	_, err = db.Exec(sql, args...)
	if err != nil {
		debug.Log("Failed to execute sector update for sector %d: %v", s.sectorIndex, err)
		return err
	}
	
	debug.Log("Successfully updated sector %d with discovered fields: %v", s.sectorIndex, getFieldNames(s.updates))
	return nil
}