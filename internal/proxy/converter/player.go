package converter

import (
	"twist/internal/api"
	"twist/internal/proxy/database"
	"twist/internal/proxy/types"
)

// ConvertToPlayerInfo converts current sector and player name to API PlayerInfo
func ConvertToPlayerInfo(currentSector int, playerName string) api.PlayerInfo {
	
	playerInfo := api.PlayerInfo{
		Name:          playerName,
		CurrentSector: currentSector,
	}
	
	
	return playerInfo
}

// ConvertTPlayerStatsToPlayerStatsInfo converts database TPlayerStats to API PlayerStatsInfo
func ConvertTPlayerStatsToPlayerStatsInfo(playerStats database.TPlayerStats) api.PlayerStatsInfo {
	return api.PlayerStatsInfo{
		Turns:         playerStats.Turns,
		Credits:       playerStats.Credits,
		Fighters:      playerStats.Fighters,
		Shields:       playerStats.Shields,
		TotalHolds:    playerStats.TotalHolds,
		OreHolds:      playerStats.OreHolds,
		OrgHolds:      playerStats.OrgHolds,
		EquHolds:      playerStats.EquHolds,
		ColHolds:      playerStats.ColHolds,
		Photons:       playerStats.Photons,
		Armids:        playerStats.Armids,
		Limpets:       playerStats.Limpets,
		GenTorps:      playerStats.GenTorps,
		TwarpType:     playerStats.TwarpType,
		Cloaks:        playerStats.Cloaks,
		Beacons:       playerStats.Beacons,
		Atomics:       playerStats.Atomics,
		Corbomite:     playerStats.Corbomite,
		Eprobes:       playerStats.Eprobes,
		MineDisr:      playerStats.MineDisr,
		Alignment:     playerStats.Alignment,
		Experience:    playerStats.Experience,
		Corp:          playerStats.Corp,
		ShipNumber:    playerStats.ShipNumber,
		ShipClass:     playerStats.ShipClass,
		PsychicProbe:  playerStats.PsychicProbe,
		PlanetScanner: playerStats.PlanetScanner,
		ScanType:      playerStats.ScanType,
		CurrentSector: playerStats.CurrentSector,
		PlayerName:    playerStats.PlayerName,
	}
}

// ConvertPlayerStatsToPlayerStatsInfo converts shared types.PlayerStats to API PlayerStatsInfo
func ConvertPlayerStatsToPlayerStatsInfo(playerStats types.PlayerStats) api.PlayerStatsInfo {
	return api.PlayerStatsInfo{
		Turns:         playerStats.Turns,
		Credits:       playerStats.Credits,
		Fighters:      playerStats.Fighters,
		Shields:       playerStats.Shields,
		TotalHolds:    playerStats.TotalHolds,
		OreHolds:      playerStats.OreHolds,
		OrgHolds:      playerStats.OrgHolds,
		EquHolds:      playerStats.EquHolds,
		ColHolds:      playerStats.ColHolds,
		Photons:       playerStats.Photons,
		Armids:        playerStats.Armids,
		Limpets:       playerStats.Limpets,
		GenTorps:      playerStats.GenTorps,
		TwarpType:     playerStats.TwarpType,
		Cloaks:        playerStats.Cloaks,
		Beacons:       playerStats.Beacons,
		Atomics:       playerStats.Atomics,
		Corbomite:     playerStats.Corbomite,
		Eprobes:       playerStats.Eprobes,
		MineDisr:      playerStats.MineDisr,
		Alignment:     playerStats.Alignment,
		Experience:    playerStats.Experience,
		Corp:          playerStats.Corp,
		ShipNumber:    playerStats.ShipNumber,
		ShipClass:     playerStats.ShipClass,
		PsychicProbe:  playerStats.PsychicProbe,
		PlanetScanner: playerStats.PlanetScanner,
		ScanType:      playerStats.ScanType,
		CurrentSector: playerStats.CurrentSector,
		PlayerName:    playerStats.PlayerName,
	}
}


