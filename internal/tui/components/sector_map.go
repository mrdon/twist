package components

import (
	"fmt"
	"strings"
	"twist/internal/api"
	"twist/internal/theme"
	
	"github.com/gdamore/tcell/v2"
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
		smc.setWaitingMessage()
		return
	}
	
	// Get player info to find current sector
	playerInfo, err := smc.proxyAPI.GetPlayerInfo()
	if err != nil {
		smc.setWaitingMessage()
		return
	}
	
	
	// Check if we have valid sector data
	if playerInfo.CurrentSector <= 0 {
		smc.setWaitingMessage()
		return
	}
	
	// Set current sector and load its data
	smc.currentSector = playerInfo.CurrentSector
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
	// Add mock sector data with bidirectional warps
	smc.currentSector = 123
	smc.sectorData[123] = api.SectorInfo{
		Number:     123,
		NavHaz:     1,
		HasTraders: 2,
		Warps:      []int{122, 124, 223, 23},
		HasPort:    true,
	}
	smc.sectorData[122] = api.SectorInfo{
		Number:     122, 
		HasTraders: 0, 
		Warps:      []int{123}, // Bidirectional connection
	}
	smc.sectorData[124] = api.SectorInfo{
		Number:     124, 
		HasTraders: 1, 
		Warps:      []int{123}, // Bidirectional connection
		HasPort:    true,
	}
	smc.sectorData[223] = api.SectorInfo{
		Number:     223, 
		HasTraders: 0, 
		Warps:      []int{123}, // Bidirectional connection
	}
	smc.sectorData[23] = api.SectorInfo{
		Number:     23, 
		HasTraders: 3, 
		Warps:      []int{123}, // Bidirectional connection
		HasPort:    true,
	}
	
	// Render the test map
	mapText := smc.renderMap()
	smc.view.SetText(mapText)
}

// UpdateCurrentSector updates the map with the current sector
func (smc *SectorMapComponent) UpdateCurrentSector(sectorNumber int) {
	smc.currentSector = sectorNumber
	smc.refreshMap()
}

// UpdateCurrentSectorWithInfo updates the map with full sector information
func (smc *SectorMapComponent) UpdateCurrentSectorWithInfo(sectorInfo api.SectorInfo) {
	smc.currentSector = sectorInfo.Number
	// Store the sector info directly so we have the warp data immediately
	smc.sectorData[sectorInfo.Number] = sectorInfo
	smc.refreshMap()
}

// UpdateSectorData updates sector data without changing the current sector focus
func (smc *SectorMapComponent) UpdateSectorData(sectorInfo api.SectorInfo) {
	// Update the sector data in our cache
	smc.sectorData[sectorInfo.Number] = sectorInfo
	
	// If this sector is connected to current sector or is the current sector, refresh the map
	if smc.currentSector > 0 && (sectorInfo.Number == smc.currentSector || smc.isSectorConnected(sectorInfo.Number)) {
		smc.refreshMap()
	}
}

// isSectorConnected checks if a sector is connected to the current sector
func (smc *SectorMapComponent) isSectorConnected(sectorNumber int) bool {
	if currentSectorInfo, exists := smc.sectorData[smc.currentSector]; exists {
		for _, warp := range currentSectorInfo.Warps {
			if warp == sectorNumber {
				return true
			}
		}
	}
	return false
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
	// First check if we already have sector data (from UpdateCurrentSectorWithInfo)
	currentInfo, hasCurrentInfo := smc.sectorData[smc.currentSector]
	if !hasCurrentInfo {
		// If we don't have it, fetch from API
		var err error
		currentInfo, err = smc.proxyAPI.GetSectorInfo(smc.currentSector)
		if err != nil {
			smc.view.SetText(fmt.Sprintf("[red]Error: %s[-]", err.Error()))
			return
		}
		// Store current sector data
		smc.sectorData[smc.currentSector] = currentInfo
	}
	
	// Get connected sector data
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	// Debug: Check what warp data we have for current sector
	if currentData, exists := smc.sectorData[smc.currentSector]; exists {
		_ = currentData // Keep for debugging if needed
	}
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

// renderMap creates the visual representation of the sector map with background colors
func (smc *SectorMapComponent) renderMap() string {
	if smc.currentSector == 0 {
		return "[yellow]Waiting for sector data...[-]"
	}
	
	// Get current sector info
	currentInfo, exists := smc.sectorData[smc.currentSector]
	if !exists {
		return fmt.Sprintf("[red]No data for sector %d[-]", smc.currentSector)
	}
	
	// Get theme colors
	currentTheme := theme.Current()
	mapColors := currentTheme.SectorMapColors()
	
	// Get panel dimensions from the view
	_, _, width, height := smc.view.GetInnerRect()
	
	// Calculate available space (subtract title and margins)
	availableWidth := width - 2  // Account for borders
	availableHeight := height - 4 // Account for title and some margin
	
	// Get connected sectors (up to 6 max)
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	
	// Build the map based on available space
	return smc.renderResponsiveMap(currentInfo, connectedSectors, mapColors, availableWidth, availableHeight)
}

// renderResponsiveMap renders a traditional 3x3 sector map
func (smc *SectorMapComponent) renderResponsiveMap(currentInfo api.SectorInfo, connectedSectors []int, mapColors theme.SectorMapColors, availableWidth, availableHeight int) string {
	var builder strings.Builder
	
	// Create 3x3 grid layout
	grid := make(map[string]int)
	
	// Place current sector in center
	grid["C"] = smc.currentSector
	
	// Intelligently place connected sectors to maximize visible connections
	smc.placeConnectedSectors(grid, connectedSectors)
	
	// Calculate sector box size based on available space
	// Expanded width to accommodate directional arrows and better formatting
	sectorWidth := 9  // "  12345  " (5 digits + 4 spaces padding for arrows)
	sectorHeight := 2 // Base: 2 rows per sector (number + port info)
	
	// Clean sector map - no warp destination display
	
	// Check if we have space for the full 3x3 grid
	requiredWidth := 3*sectorWidth + 2 // 3 sectors + 2 connection chars
	requiredHeight := 3*sectorHeight + 2 // 3 sector rows (2 lines each) + 2 connection rows
	
	// Calculate vertical centering
	mapHeight := requiredHeight
	if availableWidth >= requiredWidth && availableHeight >= requiredHeight {
		// Add vertical padding to center the map
		verticalPadding := (availableHeight - mapHeight) / 2
		for i := 0; i < verticalPadding; i++ {
			builder.WriteString("\n")
		}
		
		// Render full 3x3 grid with connections
		smc.render3x3Grid(&builder, grid, sectorWidth, mapColors)
	} else {
		// Fallback to compact single-sector view
		smc.renderCompactView(&builder, smc.currentSector, connectedSectors, mapColors)
	}
	
	return builder.String()
}

// render3x3Grid renders a traditional 3x3 sector grid with connections
func (smc *SectorMapComponent) render3x3Grid(builder *strings.Builder, grid map[string]int, sectorWidth int, mapColors theme.SectorMapColors) {
	// Grid layout:
	// NW | N  | NE
	// W  | C  | E
	// SW | S  | SE
	
	rows := [][]string{
		{"NW", "N", "NE"},
		{"W", "C", "E"},
		{"SW", "S", "SE"},
	}
	
	for rowIdx, row := range rows {
		// Render each row of sectors (2 lines per sector: number + port info)
		smc.renderSectorRow(builder, grid, row, sectorWidth, mapColors)
		
		// Add connection lines between sector rows (vertical and diagonal)
		if rowIdx < len(rows)-1 {
			smc.renderAllConnections3x3(builder, grid, rows[rowIdx], rows[rowIdx+1], sectorWidth, mapColors)
		}
	}
}

// renderSectorRow renders one row of sectors (2 lines per sector: number + port info)
func (smc *SectorMapComponent) renderSectorRow(builder *strings.Builder, grid map[string]int, row []string, sectorWidth int, mapColors theme.SectorMapColors) {
	// Render first line of sectors (sector numbers)
	for colIdx, pos := range row {
		sector, exists := grid[pos]
		
		if exists {
			// Render sector number with background color
			var bgColor, fgColor string
			if sector == smc.currentSector {
				bgColor = smc.colorToString(mapColors.CurrentSectorBg)
				fgColor = smc.colorToString(mapColors.CurrentSectorFg)
			} else {
				// Check if it's a port
				info, hasInfo := smc.sectorData[sector]
				if hasInfo && info.HasTraders > 0 {
					bgColor = smc.colorToString(mapColors.PortSectorBg)
					fgColor = smc.colorToString(mapColors.PortSectorFg)
				} else {
					bgColor = smc.colorToString(mapColors.EmptySectorBg)
					fgColor = smc.colorToString(mapColors.EmptySectorFg)
				}
			}
			
			builder.WriteString(fmt.Sprintf("[%s:%s]  %5d  [:-]", fgColor, bgColor, sector))
		} else {
			// Empty space
			builder.WriteString(strings.Repeat(" ", sectorWidth))
		}
		
		// Add spacing between sectors (no connection lines on first row)
		if colIdx < len(row)-1 {
			builder.WriteString(" ")
		}
	}
	builder.WriteString("\n")
	
	// Render second line of sectors (port info/symbols)
	for colIdx, pos := range row {
		sector, exists := grid[pos]
		
		if exists {
			// Render port info with same background color
			var bgColor, fgColor, portInfo string
			if sector == smc.currentSector {
				bgColor = smc.colorToString(mapColors.CurrentSectorBg)
				fgColor = smc.colorToString(mapColors.CurrentSectorFg)
				portInfo = "   YOU   "
			} else {
				// Check if it's a port
				info, hasInfo := smc.sectorData[sector]
				if hasInfo && info.HasTraders > 0 {
					bgColor = smc.colorToString(mapColors.PortSectorBg)
					fgColor = smc.colorToString(mapColors.PortSectorFg)
					if info.HasPort {
						// Get actual port type from API
						if portData, err := smc.proxyAPI.GetPortInfo(sector); err == nil && portData != nil {
							portInfo = fmt.Sprintf("  (%s)  ", portData.ClassType.String()[:3]) // Show port type as "(BBS)"
						} else {
							portInfo = fmt.Sprintf("   (P)   ") // Port exists but couldn't get details
						}
					} else {
						portInfo = fmt.Sprintf("   (%d)   ", info.HasTraders) // Fallback to trader count as "(2)"
					}
				} else {
					bgColor = smc.colorToString(mapColors.EmptySectorBg)
					fgColor = smc.colorToString(mapColors.EmptySectorFg)
					portInfo = "         " // Empty (9 spaces to match sectorWidth)
				}
			}
			
			builder.WriteString(fmt.Sprintf("[%s:%s]%s[:-]", fgColor, bgColor, portInfo))
		} else {
			// Empty space
			builder.WriteString(strings.Repeat(" ", sectorWidth))
		}
		
		// Add horizontal connection line for second row
		if colIdx < len(row)-1 {
			smc.renderHorizontalConnection3x3(builder, grid, pos, row[colIdx+1], mapColors)
		}
	}
	builder.WriteString("\n")
	
	// No additional lines - just clean sector boxes
	
	// Note: Vertical connections are handled in the main render3x3Grid function
}

// placeConnectedSectors intelligently places connected sectors in the 3x3 grid
func (smc *SectorMapComponent) placeConnectedSectors(grid map[string]int, connectedSectors []int) {
	
	// Adjacent positions to center (only orthogonal neighbors for cleaner connections)
	adjacentPositions := []string{"N", "E", "S", "W"}
	
	// Place up to 4 connected sectors in adjacent positions
	placed := 0
	for _, sector := range connectedSectors {
		if placed < len(adjacentPositions) {
			grid[adjacentPositions[placed]] = sector
			placed++
		}
	}
	
	
	// If we have more sectors, place them in diagonal positions
	diagonalPositions := []string{"NE", "SE", "SW", "NW"}
	for i := placed; i < len(connectedSectors) && (i-placed) < len(diagonalPositions); i++ {
		sector := connectedSectors[i]
		pos := diagonalPositions[i-placed]
		grid[pos] = sector
	}
	
}

// renderAllConnections3x3 renders all connection lines between two rows (vertical and diagonal)
func (smc *SectorMapComponent) renderAllConnections3x3(builder *strings.Builder, grid map[string]int, topRow, bottomRow []string, sectorWidth int, mapColors theme.SectorMapColors) {
	// Create a connection line that shows vertical and diagonal connections
	// Pattern for 3 sectors: "X X X" where X can be vertical (│), diagonal (\,/), or space
	
	for colIdx := 0; colIdx < len(topRow); colIdx++ {
		topPos := topRow[colIdx]
		bottomPos := bottomRow[colIdx]
		
		topSector, topExists := grid[topPos]
		bottomSector, bottomExists := grid[bottomPos]
		
		// Calculate padding to center the connection character
		padding := (sectorWidth - 1) / 2
		builder.WriteString(strings.Repeat(" ", padding))
		
		// Render vertical connection
		if topExists && bottomExists {
			// Check connections in both directions
			topToBottom := smc.hasDirectConnection(topSector, bottomSector)
			bottomToTop := smc.hasDirectConnection(bottomSector, topSector)
			
			connColor := smc.colorToString(mapColors.ConnectionLine)
			
			if topToBottom && bottomToTop {
				// Bidirectional connection
				builder.WriteString(fmt.Sprintf("[%s]↕[-]", connColor))
			} else if topToBottom {
				// Top to bottom only
				builder.WriteString(fmt.Sprintf("[%s]↓[-]", connColor))
			} else if bottomToTop {
				// Bottom to top only
				builder.WriteString(fmt.Sprintf("[%s]↑[-]", connColor))
			} else {
				builder.WriteString(" ")
			}
		} else {
			builder.WriteString(" ")
		}
		
		builder.WriteString(strings.Repeat(" ", sectorWidth-padding-1))
		
		// Add diagonal connections between columns
		if colIdx < len(topRow)-1 {
			smc.renderDiagonalConnections(builder, grid, topRow[colIdx], topRow[colIdx+1], 
				bottomRow[colIdx], bottomRow[colIdx+1], mapColors)
		}
	}
	builder.WriteString("\n")
}

// renderDiagonalConnections renders diagonal connection lines between 4 positions
func (smc *SectorMapComponent) renderDiagonalConnections(builder *strings.Builder, grid map[string]int, 
	topLeft, topRight, bottomLeft, bottomRight string, mapColors theme.SectorMapColors) {
	
	// Get sector numbers
	tlSector, tlExists := grid[topLeft]
	trSector, trExists := grid[topRight]
	blSector, blExists := grid[bottomLeft]
	brSector, brExists := grid[bottomRight]
	
	connColor := smc.colorToString(mapColors.ConnectionLine)
	
	// Check for diagonal connections
	// Top-left to bottom-right (\)
	if tlExists && brExists && smc.areConnected(tlSector, brSector) {
		builder.WriteString(fmt.Sprintf("[%s]\\[-]", connColor))
	} else if trExists && blExists && smc.areConnected(trSector, blSector) {
		// Top-right to bottom-left (/)
		builder.WriteString(fmt.Sprintf("[%s]/[-]", connColor))
	} else {
		builder.WriteString(" ")
	}
}

// renderHorizontalConnection3x3 renders horizontal connection between adjacent positions with directional indicators
func (smc *SectorMapComponent) renderHorizontalConnection3x3(builder *strings.Builder, grid map[string]int, leftPos, rightPos string, mapColors theme.SectorMapColors) {
	leftSector, leftExists := grid[leftPos]
	rightSector, rightExists := grid[rightPos]
	
	if leftExists && rightExists {
		// Check connections in both directions
		leftToRight := smc.hasDirectConnection(leftSector, rightSector)
		rightToLeft := smc.hasDirectConnection(rightSector, leftSector)
		
		connColor := smc.colorToString(mapColors.ConnectionLine)
		
		if leftToRight && rightToLeft {
			// Bidirectional connection
			builder.WriteString(fmt.Sprintf("[%s]↔[-]", connColor))
		} else if leftToRight {
			// Left to right only
			builder.WriteString(fmt.Sprintf("[%s]→[-]", connColor))
		} else if rightToLeft {
			// Right to left only
			builder.WriteString(fmt.Sprintf("[%s]←[-]", connColor))
		} else {
			builder.WriteString(" ")
		}
	} else {
		builder.WriteString(" ")
	}
}

// renderVerticalConnections3x3 renders vertical connection lines between rows
func (smc *SectorMapComponent) renderVerticalConnections3x3(builder *strings.Builder, grid map[string]int, topRow, bottomRow []string, sectorWidth int, mapColors theme.SectorMapColors) {
	for colIdx := 0; colIdx < len(topRow); colIdx++ {
		topPos := topRow[colIdx]
		bottomPos := bottomRow[colIdx]
		
		topSector, topExists := grid[topPos]
		bottomSector, bottomExists := grid[bottomPos]
		
		// Center the vertical line in the sector space
		padding := (sectorWidth - 1) / 2
		builder.WriteString(strings.Repeat(" ", padding))
		
		if topExists && bottomExists {
			// Check connections in both directions
			topToBottom := smc.hasDirectConnection(topSector, bottomSector)
			bottomToTop := smc.hasDirectConnection(bottomSector, topSector)
			
			connColor := smc.colorToString(mapColors.ConnectionLine)
			
			if topToBottom && bottomToTop {
				// Bidirectional connection
				builder.WriteString(fmt.Sprintf("[%s]↕[-]", connColor))
			} else if topToBottom {
				// Top to bottom only
				builder.WriteString(fmt.Sprintf("[%s]↓[-]", connColor))
			} else if bottomToTop {
				// Bottom to top only
				builder.WriteString(fmt.Sprintf("[%s]↑[-]", connColor))
			} else {
				builder.WriteString(" ")
			}
		} else {
			builder.WriteString(" ")
		}
		
		builder.WriteString(strings.Repeat(" ", sectorWidth-padding-1))
		
		// Add spacing between columns
		if colIdx < len(topRow)-1 {
			builder.WriteString(" ")
		}
	}
	builder.WriteString("\n")
}

// renderCompactView renders a compact view when space is limited
func (smc *SectorMapComponent) renderCompactView(builder *strings.Builder, currentSector int, connectedSectors []int, mapColors theme.SectorMapColors) {
	// Show current sector
	bgColor := smc.colorToString(mapColors.CurrentSectorBg)
	fgColor := smc.colorToString(mapColors.CurrentSectorFg)
	builder.WriteString(fmt.Sprintf("[%s:%s] %5d [:-]\n", fgColor, bgColor, currentSector))
	
	// Show connected sectors in a simple list
	if len(connectedSectors) > 0 {
		builder.WriteString("\nConnected to: ")
		for i, sector := range connectedSectors {
			if i > 0 {
				builder.WriteString(", ")
			}
			
			// Check if it's a port
			info, hasInfo := smc.sectorData[sector]
			var sectorBgColor, sectorFgColor string
			if hasInfo && info.HasTraders > 0 {
				sectorBgColor = smc.colorToString(mapColors.PortSectorBg)
				sectorFgColor = smc.colorToString(mapColors.PortSectorFg)
			} else {
				sectorBgColor = smc.colorToString(mapColors.EmptySectorBg)
				sectorFgColor = smc.colorToString(mapColors.EmptySectorFg)
			}
			
			builder.WriteString(fmt.Sprintf("[%s:%s]%d[:-]", sectorFgColor, sectorBgColor, sector))
		}
		builder.WriteString("\n")
	}
}

// hasDirectConnection checks if sector1 has a direct warp to sector2 (one-way check)
func (smc *SectorMapComponent) hasDirectConnection(sector1, sector2 int) bool {
	if info, exists := smc.sectorData[sector1]; exists {
		for _, warp := range info.Warps {
			if warp == sector2 {
				return true
			}
		}
	}
	return false
}

// areConnected checks if two sectors have a warp connection (bidirectional check)
func (smc *SectorMapComponent) areConnected(sector1, sector2 int) bool {
	return smc.hasDirectConnection(sector1, sector2) || smc.hasDirectConnection(sector2, sector1)
}

// colorToString converts tcell.Color to tview color string
func (smc *SectorMapComponent) colorToString(color tcell.Color) string {
	// Convert tcell color to hex string for tview
	r, g, b := color.RGB()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// placeSectorOnGrid places a sector representation on the grid (legacy method - can be removed)
func (smc *SectorMapComponent) placeSectorOnGrid(grid [][]string, sectorNum, x, y int, isCurrent bool) {
	// Get sector info for port type
	info, exists := smc.sectorData[sectorNum]
	
	// Format sector number (pad to 5 chars for largest sector numbers)
	sectorStr := fmt.Sprintf("%5d", sectorNum)
	
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
	
	// Place sector number (5 characters wide, centered)
	if y >= 0 && y < len(grid) && x-2 >= 0 && x+2 < len(grid[y]) {
		grid[y][x-2] = colorCode + string(sectorStr[0]) + "[-]"
		grid[y][x-1] = colorCode + string(sectorStr[1]) + "[-]"
		grid[y][x] = colorCode + string(sectorStr[2]) + "[-]"
		grid[y][x+1] = colorCode + string(sectorStr[3]) + "[-]"
		grid[y][x+2] = colorCode + string(sectorStr[4]) + "[-]"
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