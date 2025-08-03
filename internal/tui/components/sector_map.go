package components

import (
	"fmt"
	"strings"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/theme"
	
	"github.com/rivo/tview"
)

// SectorMapComponent manages the sector map visualization
type SectorMapComponent struct {
	view         *tview.TextView
	proxyAPI     api.ProxyAPI
	currentSector int
	sectorData   map[int]api.SectorInfo
	width        int
	height       int
}

// NewSectorMapComponent creates a new sector map component
func NewSectorMapComponent() *SectorMapComponent {
	mapView := theme.NewPanelView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	mapView.SetBorder(true).SetTitle("Sector Map")
	
	smc := &SectorMapComponent{
		view:         mapView,
		sectorData:   make(map[int]api.SectorInfo),
		width:        30, // Larger width for the expanded right panel
		height:       20, // More height for better map display
	}
	
	// Set initial text
	smc.view.SetText("[cyan]Sector Map[-]\n\n[yellow]Connect and load database to see sector map[-]")
	
	return smc
}

// GetView returns the tview component
func (smc *SectorMapComponent) GetView() *tview.TextView {
	return smc.view
}

// SetProxyAPI sets the API reference for accessing game data
func (smc *SectorMapComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
	smc.proxyAPI = proxyAPI
	if proxyAPI != nil {
		smc.setWaitingMessage()
		// Don't load data immediately - wait for database to be ready
		// Data will be loaded when panels become visible
	}
}

// LoadRealMapData loads real sector data from the API (public method)
func (smc *SectorMapComponent) LoadRealMapData() {
	smc.loadRealMapData()
}

// loadRealMapData loads real sector data from the API
func (smc *SectorMapComponent) loadRealMapData() {
	if smc.proxyAPI == nil {
		debug.Log("SectorMapComponent: No proxyAPI available")
		smc.setWaitingMessage()
		return
	}
	
	// Get player info to find current sector
	playerInfo, err := smc.proxyAPI.GetPlayerInfo()
	if err != nil {
		debug.Log("SectorMapComponent: GetPlayerInfo failed: %v", err)
		smc.setWaitingMessage()
		return
	}
	
	debug.Log("SectorMapComponent: Got player info - CurrentSector: %d", playerInfo.CurrentSector)
	
	// Check if we have valid sector data
	if playerInfo.CurrentSector <= 0 {
		debug.Log("SectorMapComponent: Invalid sector number: %d", playerInfo.CurrentSector)
		smc.setWaitingMessage()
		return
	}
	
	// Set current sector and load its data
	smc.currentSector = playerInfo.CurrentSector
	debug.Log("SectorMapComponent: Loading map data for sector %d", smc.currentSector)
	smc.refreshMap()
}

// setWaitingMessage displays a waiting message in the sector map
func (smc *SectorMapComponent) setWaitingMessage() {
	// Use theme colors for the waiting message
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	// Create blinking gray text using the themed waiting color
	waitingText := fmt.Sprintf("[cyan]Sector Map[-]\n\n[%s::bl]Waiting...[-]", 
		defaultColors.Waiting.String())
	smc.view.SetText(waitingText)
}

// showTestMap displays a test map to verify rendering works (kept for fallback)
func (smc *SectorMapComponent) showTestMap() {
	// Add mock sector data
	smc.currentSector = 123
	smc.sectorData[123] = api.SectorInfo{
		Number:     123,
		NavHaz:     1,
		HasTraders: 2,
		Warps:      []int{122, 124, 223, 23},
	}
	smc.sectorData[122] = api.SectorInfo{Number: 122, HasTraders: 0}
	smc.sectorData[124] = api.SectorInfo{Number: 124, HasTraders: 1}
	smc.sectorData[223] = api.SectorInfo{Number: 223, HasTraders: 0}
	smc.sectorData[23] = api.SectorInfo{Number: 23, HasTraders: 3}
	
	// Render the test map
	mapText := smc.renderMap()
	smc.view.SetText(mapText)
}

// UpdateCurrentSector updates the map with the current sector
func (smc *SectorMapComponent) UpdateCurrentSector(sectorNumber int) {
	smc.currentSector = sectorNumber
	smc.refreshMap()
}

// refreshMap refreshes the entire map display
func (smc *SectorMapComponent) refreshMap() {
	if smc.proxyAPI == nil {
		smc.view.SetText("[red]No proxy API available[-]")
		return
	}
	
	if smc.currentSector == 0 {
		smc.view.SetText("[yellow]Waiting for sector data...[-]")
		return
	}
	
	// Get current sector info and connected sectors
	currentInfo, err := smc.proxyAPI.GetSectorInfo(smc.currentSector)
	if err != nil {
		smc.view.SetText(fmt.Sprintf("[red]Error: %s[-]", err.Error()))
		return
	}
	
	// Store current sector data
	smc.sectorData[smc.currentSector] = currentInfo
	
	// Get connected sector data
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	for _, sectorNum := range connectedSectors {
		if sectorNum > 0 { // Valid sector number
			info, err := smc.proxyAPI.GetSectorInfo(sectorNum)
			if err == nil {
				smc.sectorData[sectorNum] = info
			}
		}
	}
	
	// Render the map
	mapText := smc.renderMap()
	smc.view.SetText(mapText)
}

// getConnectedSectors gets the sector numbers connected to the given sector
func (smc *SectorMapComponent) getConnectedSectors(sectorNum int) []int {
	// Get the warps from the stored sector data
	if info, exists := smc.sectorData[sectorNum]; exists {
		return info.Warps
	}
	return []int{}
}

// renderMap creates the visual representation of the sector map
func (smc *SectorMapComponent) renderMap() string {
	var builder strings.Builder
	
	// Create a grid layout for the map
	grid := make([][]string, smc.height)
	for i := range grid {
		grid[i] = make([]string, smc.width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}
	
	// Center position for current sector
	centerX := smc.width / 2
	centerY := smc.height / 2
	
	// Place current sector at center
	smc.placeSectorOnGrid(grid, smc.currentSector, centerX, centerY, true)
	
	// Place connected sectors around the center in a cleaner layout
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	positions := []struct{ x, y int }{
		{centerX, centerY - 4}, // North
		{centerX + 6, centerY - 2}, // Northeast  
		{centerX + 6, centerY + 2}, // Southeast
		{centerX, centerY + 4}, // South
		{centerX - 6, centerY + 2}, // Southwest
		{centerX - 6, centerY - 2}, // Northwest
	}
	
	for i, sectorNum := range connectedSectors {
		if i < len(positions) && sectorNum > 0 {
			pos := positions[i]
			if pos.x >= 0 && pos.x < smc.width && pos.y >= 0 && pos.y < smc.height {
				smc.placeSectorOnGrid(grid, sectorNum, pos.x, pos.y, false)
				// Draw connection line
				smc.drawConnection(grid, centerX, centerY, pos.x, pos.y)
			}
		}
	}
	
	// Convert grid to string
	for y := 0; y < smc.height; y++ {
		for x := 0; x < smc.width; x++ {
			builder.WriteString(grid[y][x])
		}
		if y < smc.height-1 {
			builder.WriteString("\n")
		}
	}
	
	return builder.String()
}

// placeSectorOnGrid places a sector representation on the grid
func (smc *SectorMapComponent) placeSectorOnGrid(grid [][]string, sectorNum, x, y int, isCurrent bool) {
	// Get sector info for port type
	info, exists := smc.sectorData[sectorNum]
	
	// Format sector number (pad to 3 chars)
	sectorStr := fmt.Sprintf("%3d", sectorNum)
	
	// Determine port type indicator
	portType := "?"
	if exists {
		if info.HasTraders > 0 {
			portType = "T" // Traders present
		} else {
			portType = "E" // Empty
		}
	}
	
	// Color coding for current vs connected sectors
	var colorCode, portColor string
	if isCurrent {
		colorCode = "[black:yellow]" // Current sector highlighted
		portColor = "[black:yellow]"
	} else {
		colorCode = "[green]" // Connected sectors
		portColor = "[cyan]"
	}
	
	// Place sector number (3 characters wide, centered)
	if y >= 0 && y < len(grid) && x-1 >= 0 && x+1 < len(grid[y]) {
		grid[y][x-1] = colorCode + string(sectorStr[0]) + "[-]"
		grid[y][x] = colorCode + string(sectorStr[1]) + "[-]"
		grid[y][x+1] = colorCode + string(sectorStr[2]) + "[-]"
	}
	
	// Place port type below sector number
	if y+1 < len(grid) && x >= 0 && x < len(grid[y+1]) {
		grid[y+1][x] = portColor + portType + "[-]"
	}
}

// drawConnection draws a line connection between two points
func (smc *SectorMapComponent) drawConnection(grid [][]string, x1, y1, x2, y2 int) {
	// Simple line drawing - use different characters for different directions
	dx := x2 - x1
	dy := y2 - y1
	
	// Determine line character based on direction
	var lineChar string
	if abs(dx) > abs(dy) {
		// Horizontal line
		lineChar = "[white]─[-]"
		// Draw horizontal line
		start := min(x1, x2)
		end := max(x1, x2)
		lineY := (y1 + y2) / 2
		if lineY >= 0 && lineY < len(grid) {
			for x := start + 1; x < end; x++ {
				if x >= 0 && x < len(grid[lineY]) && grid[lineY][x] == " " {
					grid[lineY][x] = lineChar
				}
			}
		}
	} else {
		// Vertical line
		lineChar = "[white]│[-]"
		// Draw vertical line
		start := min(y1, y2)
		end := max(y1, y2)
		lineX := (x1 + x2) / 2
		if lineX >= 0 && lineX < len(grid[0]) {
			for y := start + 1; y < end; y++ {
				if y >= 0 && y < len(grid) && grid[y][lineX] == " " {
					grid[y][lineX] = lineChar
				}
			}
		}
	}
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}