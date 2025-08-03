package components

import (
	"fmt"
	"strings"
	"time"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/theme"
	
	"github.com/rivo/tview"
)

// PanelComponent manages the side panel components
type PanelComponent struct {
	leftView     *tview.TextView
	leftWrapper  *tview.Flex
	rightWrapper *tview.Flex
	sectorMap    *SectorMapComponent  // New sector map component
	proxyAPI     api.ProxyAPI  // API access for game data
	retryCount   int           // Track retry attempts
}

// NewPanelComponent creates new panel components
func NewPanelComponent() *PanelComponent {
	// Left panel for trader info using theme
	leftPanel := theme.NewPanelView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	leftPanel.SetBorder(true).SetTitle("Trader Info")
	leftPanel.SetText("[yellow]Player Info[-]\n\n[cyan]Connect and load database to see player info[-]")
	
	leftWrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(leftPanel, 0, 1, false)
	
	// Create sector map component for right panel
	sectorMap := NewSectorMapComponent()
	
	// Right panel is just the sector map
	rightWrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(sectorMap.GetView(), 0, 1, false)
	
	return &PanelComponent{
		leftView:     leftPanel,
		leftWrapper:  leftWrapper,
		rightWrapper: rightWrapper,
		sectorMap:    sectorMap,
	}
}

// GetLeftWrapper returns the left panel wrapper
func (pc *PanelComponent) GetLeftWrapper() *tview.Flex {
	return pc.leftWrapper
}

// GetRightWrapper returns the right panel wrapper
func (pc *PanelComponent) GetRightWrapper() *tview.Flex {
	return pc.rightWrapper
}

// SetProxyAPI sets the API reference for accessing game data
func (pc *PanelComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
	pc.proxyAPI = proxyAPI
	if pc.sectorMap != nil {
		pc.sectorMap.SetProxyAPI(proxyAPI)
	}
	
	// Don't load data immediately - wait for database to be ready
	// Data will be loaded when panels become visible via animation completion
	if proxyAPI != nil {
		pc.setWaitingMessage()
	}
}

// LoadRealData loads real player and sector data from the API
func (pc *PanelComponent) LoadRealData() {
	debug.Log("PanelComponent: LoadRealData called (retry count: %d)", pc.retryCount)
	
	if pc.proxyAPI == nil {
		debug.Log("PanelComponent: No proxyAPI available")
		pc.setWaitingMessage()
		return
	}
	
	// Get player info
	playerInfo, err := pc.proxyAPI.GetPlayerInfo()
	if err != nil {
		debug.Log("PanelComponent: GetPlayerInfo failed: %v", err)
		pc.setWaitingMessage()
		return
	}
	
	debug.Log("PanelComponent: Got player info - Name: %s, CurrentSector: %d", playerInfo.Name, playerInfo.CurrentSector)
	
	// Check if we have valid sector data
	if playerInfo.CurrentSector <= 0 {
		// Limit retries to prevent infinite loops
		if pc.retryCount >= 5 {
			debug.Log("PanelComponent: Max retries reached, giving up")
			pc.setWaitingMessage()
			return
		}
		
		debug.Log("PanelComponent: Invalid sector number: %d - will retry (attempt %d/5)", playerInfo.CurrentSector, pc.retryCount+1)
		pc.setWaitingMessage()
		pc.retryCount++
		
		// Retry after a short delay - player data might not be ready yet
		go func() {
			time.Sleep(2 * time.Second)
			debug.Log("PanelComponent: Retrying LoadRealData after sector data delay")
			pc.LoadRealData()
		}()
		return
	}
	
	// Reset retry count on success
	pc.retryCount = 0
	
	// Get current sector info
	sectorInfo, err := pc.proxyAPI.GetSectorInfo(playerInfo.CurrentSector)
	if err != nil {
		debug.Log("PanelComponent: GetSectorInfo failed for sector %d: %v", playerInfo.CurrentSector, err)
		// Show player info even if sector info fails
		pc.UpdateTraderInfo(playerInfo)
		return
	}
	
	debug.Log("PanelComponent: Got sector info for sector %d", sectorInfo.Number)
	
	// Update displays with real data
	pc.UpdateTraderInfo(playerInfo)
	pc.UpdateSectorInfo(sectorInfo)
	
	// Also trigger sector map data loading
	if pc.sectorMap != nil {
		debug.Log("PanelComponent: Triggering sector map data load")
		pc.sectorMap.LoadRealMapData()
	}
}

// setWaitingMessage displays a waiting message in the left panel
func (pc *PanelComponent) setWaitingMessage() {
	// Use theme colors for the waiting message
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	// Create blinking gray text using the themed waiting color
	waitingText := fmt.Sprintf("[yellow]Player Info[-]\n\n[%s::bl]Waiting...[-]", 
		defaultColors.Waiting.String())
	pc.leftView.SetText(waitingText)
}

// setErrorMessage displays an error message in the left panel
func (pc *PanelComponent) setErrorMessage(message string) {
	errorText := fmt.Sprintf("[red]Error[-]\n\n%s", message)
	pc.leftView.SetText(errorText)
}

// showTestTraderInfo displays test trader information (kept for fallback)
func (pc *PanelComponent) showTestTraderInfo() {
	var info strings.Builder
	info.WriteString("[yellow]Player Info[-]\n\n")
	info.WriteString("Name: TestPlayer\n")
	info.WriteString("Current Sector: 123\n")
	info.WriteString("Ship: Imperial StarShip\n")
	info.WriteString("Credits: 50,000\n")
	info.WriteString("Fighters: 100\n")
	info.WriteString("Holds: 50/50\n\n")
	info.WriteString("[cyan]Cargo[-]\n")
	info.WriteString("Fuel Ore: 10\n")
	info.WriteString("Organics: 15\n")
	info.WriteString("Equipment: 25\n")
	
	pc.leftView.SetText(info.String())
}

// UpdateTraderInfo updates the trader information panel using API PlayerInfo
func (pc *PanelComponent) UpdateTraderInfo(playerInfo api.PlayerInfo) {
	var info strings.Builder
	info.WriteString("[yellow]Player Info[-]\n\n")
	
	if playerInfo.Name != "" {
		info.WriteString(fmt.Sprintf("Name: %s\n", playerInfo.Name))
	} else {
		info.WriteString("Name: [gray]Unknown[-]\n")
	}
	
	info.WriteString(fmt.Sprintf("Current Sector: [cyan]%d[-]\n\n", playerInfo.CurrentSector))
	
	// Get current sector details if available
	if pc.proxyAPI != nil {
		sectorInfo, err := pc.proxyAPI.GetSectorInfo(playerInfo.CurrentSector)
		if err == nil {
			info.WriteString("[green]Sector Details[-]\n")
			if sectorInfo.Constellation != "" {
				info.WriteString(fmt.Sprintf("Constellation: %s\n", sectorInfo.Constellation))
			}
			if sectorInfo.NavHaz > 0 {
				info.WriteString(fmt.Sprintf("Nav Hazard: [red]%d[-]\n", sectorInfo.NavHaz))
			}
			if sectorInfo.HasTraders > 0 {
				info.WriteString(fmt.Sprintf("Traders: [yellow]%d[-]\n", sectorInfo.HasTraders))
			}
			if sectorInfo.Beacon != "" {
				info.WriteString(fmt.Sprintf("Beacon: [cyan]%s[-]\n", sectorInfo.Beacon))
			}
			if len(sectorInfo.Warps) > 0 {
				info.WriteString("Warps: ")
				for i, warp := range sectorInfo.Warps {
					if i > 0 {
						info.WriteString(", ")
					}
					info.WriteString(fmt.Sprintf("%d", warp))
				}
				info.WriteString("\n")
			}
		}
	}
	
	pc.leftView.SetText(info.String())
}

// UpdateSectorInfo updates the sector map with current sector info
func (pc *PanelComponent) UpdateSectorInfo(sector api.SectorInfo) {
	if pc.sectorMap != nil {
		pc.sectorMap.UpdateCurrentSector(sector.Number)
	}
}

// SetTraderInfoText sets custom text in the trader info panel
func (pc *PanelComponent) SetTraderInfoText(text string) {
	pc.leftView.SetText(text)
}

// SetPlaceholderPlayerText sets placeholder text for when data is not available
func (pc *PanelComponent) SetPlaceholderPlayerText() {
	pc.leftView.SetText("[yellow]Player Info[-]\n\n[gray]Database not loaded - real data unavailable[-]")
}