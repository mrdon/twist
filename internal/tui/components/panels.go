package components

import (
	"fmt"
	"strings"
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
	sectorMap    *SectorMapComponent             // Original sector map component
	sixelMap     *ProperSixelSectorMapComponent // Sixel-based sector map component
	graphvizMap  *GraphvizSectorMap              // Graphviz-based sector map component (new default)
	useGraphviz  bool                     // Flag to switch between map types
	proxyAPI     api.ProxyAPI             // API access for game data
	sixelLayer   *SixelLayer              // Sixel rendering layer
	lastContentHeight int                 // Track last calculated content height
	lastPlayerStats   *api.PlayerStatsInfo // Store last received player stats
	mapRemoved   bool                     // Track if map has been removed to prevent redundant clearing
}

// NewPanelComponent creates new panel components
func NewPanelComponent(sixelLayer *SixelLayer) *PanelComponent {
	// Get theme colors for consistent styling
	currentTheme := theme.Current()
	panelColors := currentTheme.PanelColors()
	
	// Left panel for trader info using theme
	leftPanel := theme.NewPanelView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	leftPanel.SetBorder(true).SetTitle("Trader Info")
	leftPanel.SetText("[yellow]Player Info[-]\n\n[cyan]Connect and load database to see player info[-]")
	
	leftWrapper := tview.NewFlex().SetDirection(tview.FlexRow)
	
	// Explicitly set the left wrapper background to panel colors for consistency
	leftWrapper.SetBackgroundColor(panelColors.Background)
	
	// Create all sector map components for right panel
	sectorMap := NewSectorMapComponent()
	sixelMap := NewProperSixelSectorMapComponent()
	graphvizMap := NewGraphvizSectorMap(sixelLayer) // Use graphviz as default
	
	// Use graphviz map as default
	useGraphviz := true
	
	// Right panel shows the active map
	var activeMapView tview.Primitive
	if useGraphviz {
		activeMapView = graphvizMap // graphvizMap is a tview.Primitive directly
	} else {
		activeMapView = sectorMap.GetView()
	}
	
	// Create a simple black container for the right panel to isolate from background bleeding
	// This will serve as a solid black background that can't be affected by other components
	rightContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	rightContainer.SetBackgroundColor(panelColors.Background) // Explicit black background
	rightContainer.AddItem(activeMapView, 0, 1, false) // Sector map fills the container
	
	// The wrapper just contains our isolated container
	rightWrapper := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(rightContainer, 0, 1, false)
	
	// Explicitly set the right wrapper background to panel colors 
	rightWrapper.SetBackgroundColor(panelColors.Background)
	
	pc := &PanelComponent{
		leftView:     leftPanel,
		leftWrapper:  leftWrapper,
		rightWrapper: rightWrapper,
		sectorMap:    sectorMap,
		sixelMap:     sixelMap,
		graphvizMap:  graphvizMap,
		useGraphviz:  useGraphviz,
		sixelLayer:   sixelLayer,
	}
	
	// Set initial size based on content
	pc.UpdateLeftPanelSize()
	
	return pc
}

// GetLeftWrapper returns the left panel wrapper
func (pc *PanelComponent) GetLeftWrapper() *tview.Flex {
	return pc.leftWrapper
}

// GetRightWrapper returns the right panel wrapper
func (pc *PanelComponent) GetRightWrapper() *tview.Flex {
	return pc.rightWrapper
}

// RenderSixelGraphics renders sixel graphics for the active map
func (pc *PanelComponent) RenderSixelGraphics() {
	if pc.useGraphviz && pc.graphvizMap != nil {
		// GraphvizSectorMap handles sixel rendering in its Draw() method
		return
	} else if pc.sixelMap != nil {
		pc.sixelMap.RenderSixelGraphics()
	}
}

// SetProxyAPI sets the API reference for accessing game data
func (pc *PanelComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
	pc.proxyAPI = proxyAPI
	if pc.sectorMap != nil {
		pc.sectorMap.SetProxyAPI(proxyAPI)
	}
	if pc.sixelMap != nil {
		pc.sixelMap.SetProxyAPI(proxyAPI)
	}
	if pc.graphvizMap != nil {
		pc.graphvizMap.SetProxyAPI(proxyAPI)
	}
	
	// Don't load data immediately - wait for database to be ready
	// Data will be loaded when panels become visible via animation completion
	if proxyAPI != nil {
		pc.setWaitingMessage()
	}
}

// RemoveMapComponent removes the map component and replaces it with a blank view
func (pc *PanelComponent) RemoveMapComponent() {
	// Prevent redundant clearing
	if pc.mapRemoved {
		return
	}
	
	// Use the aggressive clearing that actually worked
	if pc.sixelLayer != nil {
		pc.sixelLayer.ClearAllRegions()
	}
	
	// Replace with blank view
	pc.rightWrapper.Clear()
	blankView := tview.NewTextView().SetText("")
	pc.rightWrapper.AddItem(blankView, 0, 1, false)
	pc.mapRemoved = true
}

// RestoreMapComponent restores the map component
func (pc *PanelComponent) RestoreMapComponent() {
	// Just restore the original map
	pc.rightWrapper.Clear()
	var activeMapView tview.Primitive
	if pc.useGraphviz {
		activeMapView = pc.graphvizMap
	} else {
		activeMapView = pc.sectorMap.GetView()
	}
	pc.rightWrapper.AddItem(activeMapView, 0, 1, false)
	pc.mapRemoved = false // Reset the flag
}

// LoadRealData loads real player and sector data from the database (like TWX)
func (pc *PanelComponent) LoadRealData() {
	
	if pc.proxyAPI == nil {
		pc.setWaitingMessage()
		return
	}
	
	// Get player info from database
	playerInfo, err := pc.proxyAPI.GetPlayerInfo()
	if err != nil {
		pc.setWaitingMessage()
		return
	}
	
	
	// Check if we have valid sector data
	if playerInfo.CurrentSector <= 0 {
		pc.setWaitingMessage()
		return
	}
	
	// Get current sector info
	sectorInfo, err := pc.proxyAPI.GetSectorInfo(playerInfo.CurrentSector)
	if err != nil {
		// Show player info even if sector info fails
		if pc.lastPlayerStats != nil {
			pc.UpdatePlayerStats(*pc.lastPlayerStats)
		} else {
			pc.UpdateTraderInfo(playerInfo)
		}
		return
	}
	
	
	// Update displays with real data
	// Use detailed player stats if available, otherwise fall back to basic player info
	if pc.lastPlayerStats != nil {
		pc.UpdatePlayerStats(*pc.lastPlayerStats)
	} else {
		pc.UpdateTraderInfo(playerInfo)
	}
	pc.UpdateSectorInfo(sectorInfo)
	
	// Also trigger sector map data loading
	if pc.useGraphviz && pc.graphvizMap != nil {
		pc.graphvizMap.LoadRealMapData()
	} else if pc.sixelMap != nil {
		pc.sixelMap.LoadRealMapData()
	} else if pc.sectorMap != nil {
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
	
	// Update panel size based on new content
	pc.UpdateLeftPanelSize()
}

// setErrorMessage displays an error message in the left panel
func (pc *PanelComponent) setErrorMessage(message string) {
	errorText := fmt.Sprintf("[red]Error[-]\n\n%s", message)
	pc.leftView.SetText(errorText)
	
	// Update panel size based on new content
	pc.UpdateLeftPanelSize()
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
	
	// Update panel size based on new content
	pc.UpdateLeftPanelSize()
}

// UpdateTraderInfo updates the trader information panel using API PlayerInfo
func (pc *PanelComponent) UpdateTraderInfo(playerInfo api.PlayerInfo) {
	var info strings.Builder
	
	// Helper function to format a labeled line with proper alignment (no backgrounds)
	formatLine := func(label, value, valueColor string) string {
		// Pad label to fit the layout, right-align the colon and value
		return fmt.Sprintf("%-12s : [%s]%s[-]\n", label, valueColor, value)
	}
	
	// Header section - simple yellow text
	info.WriteString("[yellow]Trader Info[-]\n\n")
	
	// Get current sector details if available
	var sectorInfo api.SectorInfo
	var hasSectorInfo bool
	if pc.proxyAPI != nil {
		if si, err := pc.proxyAPI.GetSectorInfo(playerInfo.CurrentSector); err == nil {
			sectorInfo = si
			hasSectorInfo = true
		}
	}
	
	// Top section - just sector, credits, turns
	sectorValue := fmt.Sprintf("%d", playerInfo.CurrentSector)
	info.WriteString(formatLine("Sector", sectorValue, "cyan"))
	info.WriteString(formatLine("Credits", "?", "gray"))
	info.WriteString(formatLine("Turns", "?", "gray"))
	
	// Cargo section 
	info.WriteString("\n[yellow]Cargo[-]\n")
	info.WriteString(formatLine("Fuel Ore", "?", "gray"))
	info.WriteString(formatLine("Organics", "?", "gray"))
	info.WriteString(formatLine("Equipment", "?", "gray"))
	info.WriteString(formatLine("Colonists", "?", "gray"))
	info.WriteString(formatLine("Empty", "?", "gray"))
	
	// Traders in sector
	if hasSectorInfo && sectorInfo.HasTraders > 0 {
		info.WriteString(fmt.Sprintf("\n[white]%d Trader(s) are here.[-]\n", sectorInfo.HasTraders))
	}
	
	// Beacon
	if hasSectorInfo && sectorInfo.Beacon != "" {
		info.WriteString(formatLine("Beacon", sectorInfo.Beacon, "cyan"))
	}
	
	// Ship Info section
	info.WriteString("\n[yellow]Ship Info[-]\n")
	info.WriteString(formatLine("Ship Type", "?", "gray"))
	info.WriteString(formatLine("Fighters", "?", "gray"))
	info.WriteString(formatLine("Shields", "?", "gray"))
	info.WriteString(formatLine("Holds", "?/?", "gray"))
	info.WriteString(formatLine("Photons", "?", "gray"))
	info.WriteString(formatLine("Armids", "?", "gray"))
	info.WriteString(formatLine("Limpets", "?", "gray"))
	info.WriteString(formatLine("Alignment", "?", "gray"))
	info.WriteString(formatLine("Experience", "?", "gray"))
	
	pc.leftView.SetText(info.String())
	
	// Update panel size based on new content
	pc.UpdateLeftPanelSize()
}

// UpdateTraderData updates trader panel with actual trader data from sector
func (pc *PanelComponent) UpdateTraderData(sectorNumber int, traders []api.TraderInfo) {
	var info strings.Builder
	
	// Helper function to format a labeled line with proper alignment
	formatLine := func(label, value, valueColor string) string {
		return fmt.Sprintf("%-12s : [%s]%s[-]\n", label, valueColor, value)
	}
	
	// Header section
	info.WriteString("[yellow]Trader Info[-]\n\n")
	
	// Show current sector
	sectorValue := fmt.Sprintf("%d", sectorNumber)
	info.WriteString(formatLine("Sector", sectorValue, "cyan"))
	
	// Show trader count and details
	if len(traders) == 0 {
		info.WriteString("\n[gray]No traders in this sector.[-]\n")
	} else {
		info.WriteString(fmt.Sprintf("\n[white]%d Trader(s) in sector:[-]\n\n", len(traders)))
		
		for i, trader := range traders {
			// Trader number and name
			traderNum := fmt.Sprintf("Trader %d", i+1)
			info.WriteString(fmt.Sprintf("[yellow]%s[-]\n", traderNum))
			
			// Trader details
			info.WriteString(formatLine("Name", trader.Name, "white"))
			
			if trader.ShipName != "" {
				info.WriteString(formatLine("Ship", trader.ShipName, "cyan"))
			}
			
			if trader.ShipType != "" {
				info.WriteString(formatLine("Type", trader.ShipType, "cyan"))
			}
			
			if trader.Fighters > 0 {
				fighterStr := fmt.Sprintf("%d", trader.Fighters)
				info.WriteString(formatLine("Fighters", fighterStr, "red"))
			}
			
			if trader.Alignment != "" {
				alignColor := "white"
				switch strings.ToLower(trader.Alignment) {
				case "good":
					alignColor = "green"
				case "evil":
					alignColor = "red"
				case "neutral":
					alignColor = "yellow"
				}
				info.WriteString(formatLine("Alignment", trader.Alignment, alignColor))
			}
			
			if i < len(traders)-1 {
				info.WriteString("\n")
			}
		}
	}
	
	pc.leftView.SetText(info.String())
	pc.UpdateLeftPanelSize()
}

// formatNumber formats large numbers with k/m/b suffixes
func formatNumber(n int) string {
	if n >= 1000000000 {
		return fmt.Sprintf("%.1fb", float64(n)/1000000000)
	} else if n >= 1000000 {
		return fmt.Sprintf("%.1fm", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// UpdatePlayerStats updates trader panel with current player statistics
func (pc *PanelComponent) UpdatePlayerStats(stats api.PlayerStatsInfo) {
	// Store the player stats for future use
	pc.lastPlayerStats = &stats
	
	var info strings.Builder
	
	// Helper function to format a labeled line with proper alignment - no individual colors
	formatLine := func(label, value string) string {
		return fmt.Sprintf("%-12s : %s\n", label, value)
	}
	
	// Header section
	info.WriteString("[yellow]Player Stats[-]\n\n")
	
	// Basic info
	info.WriteString(formatLine("Sector", fmt.Sprintf("%d", stats.CurrentSector)))
	info.WriteString(formatLine("Credits", formatNumber(stats.Credits)))
	info.WriteString(formatLine("Turns", formatNumber(stats.Turns)))
	
	// Cargo section 
	info.WriteString("\n[yellow]Cargo[-]\n")
	info.WriteString(formatLine("Fuel Ore", fmt.Sprintf("%d", stats.OreHolds)))
	info.WriteString(formatLine("Organics", fmt.Sprintf("%d", stats.OrgHolds)))
	info.WriteString(formatLine("Equipment", fmt.Sprintf("%d", stats.EquHolds)))
	info.WriteString(formatLine("Colonists", fmt.Sprintf("%d", stats.ColHolds)))
	
	empty := stats.TotalHolds - stats.OreHolds - stats.OrgHolds - stats.EquHolds - stats.ColHolds
	if empty < 0 {
		empty = 0
	}
	info.WriteString(formatLine("Empty", fmt.Sprintf("%d", empty)))
	
	// Ship Info section
	info.WriteString("\n[yellow]Ship Info[-]\n")
	info.WriteString(formatLine("Ship Type", stats.ShipClass))
	info.WriteString(formatLine("Fighters", fmt.Sprintf("%d", stats.Fighters)))
	info.WriteString(formatLine("Shields", fmt.Sprintf("%d", stats.Shields)))
	info.WriteString(formatLine("Holds", fmt.Sprintf("%d/%d", stats.TotalHolds-empty, stats.TotalHolds)))
	info.WriteString(formatLine("Photons", fmt.Sprintf("%d", stats.Photons)))
	info.WriteString(formatLine("Armids", fmt.Sprintf("%d", stats.Armids)))
	info.WriteString(formatLine("Limpets", fmt.Sprintf("%d", stats.Limpets)))
	info.WriteString(formatLine("Alignment", fmt.Sprintf("%d", stats.Alignment)))
	info.WriteString(formatLine("Experience", formatNumber(stats.Experience)))
	
	pc.leftView.SetText(info.String())
	pc.UpdateLeftPanelSize()
}

// UpdateSectorInfo updates the sector map with current sector info
func (pc *PanelComponent) UpdateSectorInfo(sector api.SectorInfo) {
	if pc.useGraphviz && pc.graphvizMap != nil {
		pc.graphvizMap.UpdateCurrentSectorWithInfo(sector)
	} else if pc.sixelMap != nil {
		pc.sixelMap.UpdateCurrentSectorWithInfo(sector)
	} else if pc.sectorMap != nil {
		pc.sectorMap.UpdateCurrentSectorWithInfo(sector)
	}
}

// UpdateSectorData updates sector data in maps without changing the current sector focus
func (pc *PanelComponent) UpdateSectorData(sector api.SectorInfo) {
	if pc.useGraphviz && pc.graphvizMap != nil {
		pc.graphvizMap.UpdateSectorData(sector)
	} else if pc.sixelMap != nil {
		pc.sixelMap.UpdateSectorData(sector)
	} else if pc.sectorMap != nil {
		pc.sectorMap.UpdateSectorData(sector)
	}
}

// SetTraderInfoText sets custom text in the trader info panel
func (pc *PanelComponent) SetTraderInfoText(text string) {
	pc.leftView.SetText(text)
	
	// Update panel size based on new content
	pc.UpdateLeftPanelSize()
}

// SetPlaceholderPlayerText sets placeholder text for when data is not available
func (pc *PanelComponent) SetPlaceholderPlayerText() {
	pc.leftView.SetText("[yellow]Player Info[-]\n\n[gray]Database not loaded - real data unavailable[-]")
	
	// Update panel size based on new content
	pc.UpdateLeftPanelSize()
}

// ToggleMapType switches between graphviz, sixel, and traditional sector maps
func (pc *PanelComponent) ToggleMapType() {
	// Cycle through: graphviz -> sixel -> traditional -> graphviz
	if pc.useGraphviz {
		pc.useGraphviz = false
		// Now using sixel map
	}
	// Add more toggle states here if needed
	
	// Update the right wrapper to show the new map type
	pc.rightWrapper.Clear()
	
	var activeMapView tview.Primitive
	
	if pc.useGraphviz && pc.graphvizMap != nil {
		activeMapView = pc.graphvizMap
	} else if pc.sixelMap != nil {
		activeMapView = pc.sixelMap
	} else {
		activeMapView = pc.sectorMap.GetView()
	}
	
	pc.rightWrapper.AddItem(activeMapView, 0, 1, false)
	
	// Ensure right wrapper has correct background color
	currentTheme := theme.Current()
	panelColors := currentTheme.PanelColors()
	pc.rightWrapper.SetBackgroundColor(panelColors.Background)
	
	// Trigger data reload for the new active map
	if pc.proxyAPI != nil {
		if pc.useGraphviz && pc.graphvizMap != nil {
			pc.graphvizMap.LoadRealMapData()
		} else if pc.sixelMap != nil {
			pc.sixelMap.LoadRealMapData()
		} else if pc.sectorMap != nil {
			pc.sectorMap.LoadRealMapData()
		}
	}
}

// GetMapType returns the current map type (true for graphviz, false for others)
func (pc *PanelComponent) GetMapType() bool {
	return pc.useGraphviz
}

// CalculateContentHeight calculates the required height for the trader info content
func (pc *PanelComponent) CalculateContentHeight() int {
	text := pc.leftView.GetText(false) // Get text without color tags
	if text == "" {
		return 5 // Minimum height for empty content
	}
	
	// Count lines in the text
	lines := strings.Split(text, "\n")
	contentHeight := len(lines)
	
	// Add padding for border and title (2 for borders + 1 for title)
	totalHeight := contentHeight + 3
	
	// Set reasonable bounds
	minHeight := 8  // Minimum useful height
	maxHeight := 25 // Maximum height to avoid taking too much space
	
	if totalHeight < minHeight {
		totalHeight = minHeight
	}
	if totalHeight > maxHeight {
		totalHeight = maxHeight
	}
	
	pc.lastContentHeight = totalHeight
	return totalHeight
}

// GetContentHeight returns the last calculated content height
func (pc *PanelComponent) GetContentHeight() int {
	if pc.lastContentHeight == 0 {
		return pc.CalculateContentHeight()
	}
	return pc.lastContentHeight
}

// UpdateLeftPanelSize updates the left panel size based on content
func (pc *PanelComponent) UpdateLeftPanelSize() {
	requiredHeight := pc.CalculateContentHeight()
	
	// Clear and rebuild the wrapper with new height
	pc.leftWrapper.Clear()
	pc.leftWrapper.AddItem(pc.leftView, requiredHeight, 0, false) // Fixed height, no flex
	
}

// loadPlayerStatsFromAPI loads current player stats from the live parser
func (pc *PanelComponent) loadPlayerStatsFromAPI() {
	if pc.proxyAPI == nil {
		debug.Log("loadPlayerStatsFromAPI: proxyAPI is nil")
		return
	}
	
	// Get player stats from API (single source of truth)
	playerStats, err := pc.proxyAPI.GetPlayerStats()
	if err != nil {
		debug.Log("loadPlayerStatsFromAPI: failed to load player stats: %v", err)
		return
	}
	
	if playerStats != nil {
		// Store and display the stats
		pc.lastPlayerStats = playerStats
		debug.Log("loadPlayerStatsFromAPI: successfully loaded stats - credits: %d, turns: %d, sector: %d", 
			playerStats.Credits, playerStats.Turns, playerStats.CurrentSector)
		pc.UpdatePlayerStats(*pc.lastPlayerStats)
	} else {
		debug.Log("loadPlayerStatsFromAPI: playerStats is nil")
	}
}

// HasDetailedPlayerStats returns true if we have detailed player stats available
func (pc *PanelComponent) HasDetailedPlayerStats() bool {
	return pc.lastPlayerStats != nil
}

// UpdatePlayerStatsSector updates the current sector in existing player stats and refreshes display
func (pc *PanelComponent) UpdatePlayerStatsSector(sectorNumber int) {
	debug.Log("UpdatePlayerStatsSector: called with sector %d, lastPlayerStats is nil: %v", sectorNumber, pc.lastPlayerStats == nil)
	
	if pc.lastPlayerStats == nil {
		// First sector change - try to load from API
		debug.Log("UpdatePlayerStatsSector: attempting to load from API")
		pc.loadPlayerStatsFromAPI()
		if pc.lastPlayerStats != nil {
			// Successfully loaded from API, update sector and display
			pc.lastPlayerStats.CurrentSector = sectorNumber
			debug.Log("UpdatePlayerStatsSector: loaded from API and updating sector to %d", sectorNumber)
			pc.UpdatePlayerStats(*pc.lastPlayerStats)
		} else {
			debug.Log("UpdatePlayerStatsSector: failed to load from API")
		}
		return
	}
	
	// Update the sector in the existing stats
	pc.lastPlayerStats.CurrentSector = sectorNumber
	debug.Log("UpdatePlayerStatsSector: updating existing stats sector to %d", sectorNumber)
	
	// Refresh the display with updated stats
	pc.UpdatePlayerStats(*pc.lastPlayerStats)
}