package streaming

import (
	"database/sql"
	"github.com/Masterminds/squirrel"
	"twist/internal/debug"
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
	// Validate and cap credits at reasonable maximum
	if credits < 0 {
		credits = 0
	}
	if credits > 2000000000 { // 2 billion credit cap
		credits = 2000000000
	}
	p.updates[ColPlayerCredits] = credits
	return p
}

// SetFighters records that fighters field was discovered during parsing
func (p *PlayerStatsTracker) SetFighters(fighters int) *PlayerStatsTracker {
	// Validate fighters must be non-negative
	if fighters < 0 {
		fighters = 0
	}
	p.updates[ColPlayerFighters] = fighters
	return p
}

// SetShields records that shields field was discovered during parsing
func (p *PlayerStatsTracker) SetShields(shields int) *PlayerStatsTracker {
	// Validate shields must be non-negative
	if shields < 0 {
		shields = 0
	}
	p.updates[ColPlayerShields] = shields
	return p
}

// SetTotalHolds records that total_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetTotalHolds(totalHolds int) *PlayerStatsTracker {
	// Validate total holds must be non-negative
	if totalHolds < 0 {
		totalHolds = 0
	}
	p.updates[ColPlayerTotalHolds] = totalHolds
	return p
}

// SetOreHolds records that ore_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetOreHolds(oreHolds int) *PlayerStatsTracker {
	// Validate ore holds must be non-negative
	if oreHolds < 0 {
		oreHolds = 0
	}
	p.updates[ColPlayerOreHolds] = oreHolds
	return p
}

// SetOrgHolds records that org_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetOrgHolds(orgHolds int) *PlayerStatsTracker {
	// Validate organics holds must be non-negative
	if orgHolds < 0 {
		orgHolds = 0
	}
	p.updates[ColPlayerOrgHolds] = orgHolds
	return p
}

// SetEquHolds records that equ_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetEquHolds(equHolds int) *PlayerStatsTracker {
	// Validate equipment holds must be non-negative
	if equHolds < 0 {
		equHolds = 0
	}
	p.updates[ColPlayerEquHolds] = equHolds
	return p
}

// SetColHolds records that col_holds field was discovered during parsing
func (p *PlayerStatsTracker) SetColHolds(colHolds int) *PlayerStatsTracker {
	// Validate colonists holds must be non-negative
	if colHolds < 0 {
		colHolds = 0
	}
	p.updates[ColPlayerColHolds] = colHolds
	return p
}

// SetPhotons records that photons field was discovered during parsing
func (p *PlayerStatsTracker) SetPhotons(photons int) *PlayerStatsTracker {
	// Validate photons must be non-negative
	if photons < 0 {
		photons = 0
	}
	p.updates[ColPlayerPhotons] = photons
	return p
}

// SetArmids records that armids field was discovered during parsing
func (p *PlayerStatsTracker) SetArmids(armids int) *PlayerStatsTracker {
	// Validate armids must be non-negative
	if armids < 0 {
		armids = 0
	}
	p.updates[ColPlayerArmids] = armids
	return p
}

// SetLimpets records that limpets field was discovered during parsing
func (p *PlayerStatsTracker) SetLimpets(limpets int) *PlayerStatsTracker {
	// Validate limpets must be non-negative
	if limpets < 0 {
		limpets = 0
	}
	p.updates[ColPlayerLimpets] = limpets
	return p
}

// SetGenTorps records that gen_torps field was discovered during parsing
func (p *PlayerStatsTracker) SetGenTorps(genTorps int) *PlayerStatsTracker {
	// Validate genesis torpedoes must be non-negative
	if genTorps < 0 {
		genTorps = 0
	}
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
		debug.Info("Failed to ensure player_stats record exists", "error", err)
		return err
	}

	// Build dynamic UPDATE query with only discovered fields
	query := psql.Update("player_stats").
		SetMap(p.updates).
		Set("updated_at", squirrel.Expr("CURRENT_TIMESTAMP")).
		Where(squirrel.Eq{"id": 1})

	sql, args, err := query.ToSql()
	if err != nil {
		debug.Info("Failed to build player stats update query", "error", err)
		return err
	}

	// debug.Info("Executing player stats update", "field_count", len(p.updates), "sql", sql)

	_, err = db.Exec(sql, args...)
	if err != nil {
		debug.Info("Failed to execute player stats update", "error", err)
		return err
	}

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
	// Validate NavHaz must be 0-100 (percentage)
	if navHaz < 0 {
		navHaz = 0
	}
	if navHaz > 100 {
		navHaz = 100
	}
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
	// Validate density must be non-negative (reasonable maximum ~50000)
	if density < 0 {
		density = 0
	}
	if density > 50000 {
		density = 50000
	}
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
		debug.Info("Failed to ensure sector record exists", "sector", s.sectorIndex, "error", err)
		return err
	}

	// Build dynamic UPDATE query with only discovered fields
	query := psql.Update("sectors").
		SetMap(s.updates).
		Set("update_time", squirrel.Expr("CURRENT_TIMESTAMP")).
		Where(squirrel.Eq{"sector_index": s.sectorIndex})

	sql, args, err := query.ToSql()
	if err != nil {
		debug.Info("Failed to build sector update query", "sector", s.sectorIndex, "error", err)
		return err
	}

	debug.Info("Executing sector update", "sector", s.sectorIndex, "field_count", len(s.updates), "sql", sql)

	_, err = db.Exec(sql, args...)
	if err != nil {
		debug.Info("Failed to execute sector update", "sector", s.sectorIndex, "error", err)
		return err
	}

	debug.Info("Successfully updated sector with discovered fields", "sector", s.sectorIndex, "fields", getFieldNames(s.updates))
	return nil
}

// PortTracker tracks discovered port field updates during parsing
// Uses discovered field tracking - only updates fields that were actually parsed
type PortTracker struct {
	sectorIndex int
	updates     map[string]interface{}
}

// NewPortTracker creates a new port tracker for the given sector
func NewPortTracker(sectorIndex int) *PortTracker {
	return &PortTracker{
		sectorIndex: sectorIndex,
		updates:     make(map[string]interface{}),
	}
}

// SetName records that name field was discovered during parsing
func (p *PortTracker) SetName(name string) *PortTracker {
	p.updates[ColPortName] = name
	return p
}

// SetDead records that dead field was discovered during parsing
func (p *PortTracker) SetDead(dead bool) *PortTracker {
	p.updates[ColPortDead] = dead
	return p
}

// SetBuildTime records that build_time field was discovered during parsing
func (p *PortTracker) SetBuildTime(buildTime int) *PortTracker {
	// Validate build time must be non-negative
	if buildTime < 0 {
		buildTime = 0
	}
	p.updates[ColPortBuildTime] = buildTime
	return p
}

// SetClassIndex records that class_index field was discovered during parsing
func (p *PortTracker) SetClassIndex(classIndex int) *PortTracker {
	p.updates[ColPortClassIndex] = classIndex
	return p
}

// SetBuyProducts records what products the port buys/sells
func (p *PortTracker) SetBuyProducts(buyFuelOre, buyOrganics, buyEquipment bool) *PortTracker {
	p.updates[ColPortBuyFuelOre] = buyFuelOre
	p.updates[ColPortBuyOrganics] = buyOrganics
	p.updates[ColPortBuyEquipment] = buyEquipment
	return p
}

// SetProductPercents records the percentages for each product
func (p *PortTracker) SetProductPercents(percentFuelOre, percentOrganics, percentEquipment int) *PortTracker {
	// Validate percentages must be 0-100
	if percentFuelOre < 0 {
		percentFuelOre = 0
	}
	if percentFuelOre > 100 {
		percentFuelOre = 100
	}
	if percentOrganics < 0 {
		percentOrganics = 0
	}
	if percentOrganics > 100 {
		percentOrganics = 100
	}
	if percentEquipment < 0 {
		percentEquipment = 0
	}
	if percentEquipment > 100 {
		percentEquipment = 100
	}
	p.updates[ColPortPercentFuelOre] = percentFuelOre
	p.updates[ColPortPercentOrganics] = percentOrganics
	p.updates[ColPortPercentEquipment] = percentEquipment
	return p
}

// SetProductAmounts records the amounts for each product
func (p *PortTracker) SetProductAmounts(amountFuelOre, amountOrganics, amountEquipment int) *PortTracker {
	// Validate amounts must be non-negative
	if amountFuelOre < 0 {
		amountFuelOre = 0
	}
	if amountOrganics < 0 {
		amountOrganics = 0
	}
	if amountEquipment < 0 {
		amountEquipment = 0
	}
	p.updates[ColPortAmountFuelOre] = amountFuelOre
	p.updates[ColPortAmountOrganics] = amountOrganics
	p.updates[ColPortAmountEquipment] = amountEquipment
	return p
}

// HasUpdates returns true if any fields were discovered during parsing
func (p *PortTracker) HasUpdates() bool {
	return len(p.updates) > 0
}

// GetUpdates returns a copy of the updates map for debugging
func (p *PortTracker) GetUpdates() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range p.updates {
		result[k] = v
	}
	return result
}

// Individual product field setters for precise updates
func (p *PortTracker) SetFuelOreAmount(amount int) *PortTracker {
	// Validate amount must be non-negative
	if amount < 0 {
		amount = 0
	}
	p.updates[ColPortAmountFuelOre] = amount
	return p
}

func (p *PortTracker) SetFuelOrePercent(percent int) *PortTracker {
	// Validate percent must be 0-100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	p.updates[ColPortPercentFuelOre] = percent
	return p
}

func (p *PortTracker) SetFuelOreBuying(buying bool) *PortTracker {
	p.updates[ColPortBuyFuelOre] = buying
	return p
}

func (p *PortTracker) SetOrganicsAmount(amount int) *PortTracker {
	// Validate amount must be non-negative
	if amount < 0 {
		amount = 0
	}
	p.updates[ColPortAmountOrganics] = amount
	return p
}

func (p *PortTracker) SetOrganicsPercent(percent int) *PortTracker {
	// Validate percent must be 0-100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	p.updates[ColPortPercentOrganics] = percent
	return p
}

func (p *PortTracker) SetOrganicsBuying(buying bool) *PortTracker {
	p.updates[ColPortBuyOrganics] = buying
	return p
}

func (p *PortTracker) SetEquipmentAmount(amount int) *PortTracker {
	// Validate amount must be non-negative
	if amount < 0 {
		amount = 0
	}
	p.updates[ColPortAmountEquipment] = amount
	return p
}

func (p *PortTracker) SetEquipmentPercent(percent int) *PortTracker {
	// Validate percent must be 0-100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	p.updates[ColPortPercentEquipment] = percent
	return p
}

func (p *PortTracker) SetEquipmentBuying(buying bool) *PortTracker {
	p.updates[ColPortBuyEquipment] = buying
	return p
}

// Execute writes discovered fields to database using Squirrel query builder
// Only fields that were actually parsed/discovered are updated
func (p *PortTracker) Execute(db *sql.DB) error {
	if len(p.updates) == 0 {
		return nil // No updates to perform
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Ensure port record exists (UPSERT pattern)
	_, err := db.Exec("INSERT OR IGNORE INTO ports (sector_index) VALUES (?)", p.sectorIndex)
	if err != nil {
		debug.Info("Failed to ensure port record exists", "sector", p.sectorIndex, "error", err)
		return err
	}

	// Build dynamic UPDATE query with only discovered fields
	query := psql.Update("ports").
		SetMap(p.updates).
		Set("updated_at", squirrel.Expr("CURRENT_TIMESTAMP")).
		Where(squirrel.Eq{"sector_index": p.sectorIndex})

	sql, args, err := query.ToSql()
	if err != nil {
		debug.Info("Failed to build port update query", "sector", p.sectorIndex, "error", err)
		return err
	}

	debug.Info("Executing port update", "sector", p.sectorIndex, "field_count", len(p.updates), "sql", sql)

	_, err = db.Exec(sql, args...)
	if err != nil {
		debug.Info("Failed to execute port update", "sector", p.sectorIndex, "error", err)
		return err
	}

	debug.Info("Successfully updated port with discovered fields", "sector", p.sectorIndex, "fields", getFieldNames(p.updates))
	return nil
}
