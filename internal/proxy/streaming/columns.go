package streaming

// Player stats column constants - single source of truth for column names
// These constants MUST match the database schema in schema.go
const (
	ColPlayerTurns         = "turns"
	ColPlayerCredits       = "credits"
	ColPlayerFighters      = "fighters"
	ColPlayerShields       = "shields"
	ColPlayerTotalHolds    = "total_holds"
	ColPlayerOreHolds      = "ore_holds"
	ColPlayerOrgHolds      = "org_holds"
	ColPlayerEquHolds      = "equ_holds"
	ColPlayerColHolds      = "col_holds"
	ColPlayerPhotons       = "photons"
	ColPlayerArmids        = "armids"
	ColPlayerLimpets       = "limpets"
	ColPlayerGenTorps      = "gen_torps"
	ColPlayerTwarpType     = "twarp_type"
	ColPlayerCloaks        = "cloaks"
	ColPlayerBeacons       = "beacons"
	ColPlayerAtomics       = "atomics"
	ColPlayerCorbomite     = "corbomite"
	ColPlayerEprobes       = "eprobes"
	ColPlayerMineDisr      = "mine_disr"
	ColPlayerAlignment     = "alignment"
	ColPlayerExperience    = "experience"
	ColPlayerCorp          = "corp"
	ColPlayerShipNumber    = "ship_number"
	ColPlayerPsychicProbe  = "psychic_probe"
	ColPlayerPlanetScanner = "planet_scanner"
	ColPlayerScanType      = "scan_type"
	ColPlayerShipClass     = "ship_class"
	ColPlayerCurrentSector = "current_sector"
	ColPlayerPlayerName    = "player_name"
)

// Future: Sector column constants for Phase 2
const (
	ColSectorConstellation = "constellation"
	ColSectorBeacon        = "beacon"
	ColSectorNavHaz        = "nav_haz"
	ColSectorWarp1         = "warp1"
	ColSectorWarp2         = "warp2"
	ColSectorWarp3         = "warp3"
	ColSectorWarp4         = "warp4"
	ColSectorWarp5         = "warp5"
	ColSectorWarp6         = "warp6"
	ColSectorWarps         = "warps"
	ColSectorDensity       = "density"
	ColSectorAnomaly       = "anomaly"
	ColSectorExplored      = "explored"
)