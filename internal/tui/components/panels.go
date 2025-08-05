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
	
	leftWrapper := theme.NewFlex().SetDirection(tview.FlexRow)
	
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
	
	rightWrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(activeMapView, 0, 1, false)
	
	// Explicitly set the right wrapper background to panel colors to fix red background bleeding
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

// LoadRealData loads real player and sector data from the database (like TWX)
func (pc *PanelComponent) LoadRealData() {
	debug.Log("PanelComponent: LoadRealData called")
	
	if pc.proxyAPI == nil {
		debug.Log("PanelComponent: No proxyAPI available")
		pc.setWaitingMessage()
		return
	}
	
	// Get player info from database
	playerInfo, err := pc.proxyAPI.GetPlayerInfo()
	if err != nil {
		debug.Log("PanelComponent: GetPlayerInfo failed: %v", err)
		pc.setWaitingMessage()
		return
	}
	
	debug.Log("PanelComponent: Got player info - Name: %s, CurrentSector: %d", playerInfo.Name, playerInfo.CurrentSector)
	
	// Check if we have valid sector data
	if playerInfo.CurrentSector <= 0 {
		debug.Log("PanelComponent: Invalid sector number: %d", playerInfo.CurrentSector)
		pc.setWaitingMessage()
		return
	}
	
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
	if pc.useGraphviz && pc.graphvizMap != nil {
		debug.Log("PanelComponent: Triggering graphviz sector map data load")
		pc.graphvizMap.LoadRealMapData()
	} else if pc.sixelMap != nil {
		debug.Log("PanelComponent: Triggering sixel sector map data load")
		pc.sixelMap.LoadRealMapData()
	} else if pc.sectorMap != nil {
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

// UpdateSectorInfo updates the sector map with current sector info
func (pc *PanelComponent) UpdateSectorInfo(sector api.SectorInfo) {
	debug.Log("PanelComponent: UpdateSectorInfo called for sector %d, useGraphviz=%v", sector.Number, pc.useGraphviz)
	if pc.useGraphviz && pc.graphvizMap != nil {
		pc.graphvizMap.UpdateCurrentSectorWithInfo(sector)
	} else if pc.sixelMap != nil {
		pc.sixelMap.UpdateCurrentSectorWithInfo(sector)
	} else if pc.sectorMap != nil {
		pc.sectorMap.UpdateCurrentSectorWithInfo(sector)
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
	var mapTypeName string
	
	if pc.useGraphviz && pc.graphvizMap != nil {
		activeMapView = pc.graphvizMap
		mapTypeName = "graphviz map"
	} else if pc.sixelMap != nil {
		activeMapView = pc.sixelMap
		mapTypeName = "sixel map"
	} else {
		activeMapView = pc.sectorMap.GetView()
		mapTypeName = "traditional map"
	}
	
	debug.Log("PanelComponent: Switched to %s", mapTypeName)
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
	debug.Log("PanelComponent: Calculated content height: %d lines, total height: %d", contentHeight, totalHeight)
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
	
	debug.Log("PanelComponent: Updated left panel size to height %d", requiredHeight)
}