package types

// PlayerStats holds current player statistics (mirrors TWX Pascal state)
// This is shared between streaming and API packages to avoid circular dependencies
type PlayerStats struct {
	Turns         int
	Credits       int
	Fighters      int
	Shields       int
	TotalHolds    int
	OreHolds      int
	OrgHolds      int
	EquHolds      int
	ColHolds      int
	Photons       int
	Armids        int
	Limpets       int
	GenTorps      int
	TwarpType     int
	Cloaks        int
	Beacons       int
	Atomics       int
	Corbomite     int
	Eprobes       int
	MineDisr      int
	Alignment     int
	Experience    int
	Corp          int
	ShipNumber    int
	ShipClass     string
	PsychicProbe  bool
	PlanetScanner bool
	ScanType      int
	CurrentSector int
	PlayerName    string
}