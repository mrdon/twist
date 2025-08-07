package converter

import (
	"twist/internal/api"
)

// ConvertToPlayerInfo converts current sector and player name to API PlayerInfo
func ConvertToPlayerInfo(currentSector int, playerName string) api.PlayerInfo {
	
	playerInfo := api.PlayerInfo{
		Name:          playerName,
		CurrentSector: currentSector,
	}
	
	
	return playerInfo
}