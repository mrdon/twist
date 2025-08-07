package components

import (
	"fmt"
	"math"
	"os"
	"strings"
	"twist/internal/api"
	"twist/internal/theme"
	
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SixelSectorMapComponent manages the sixel-based sector map visualization
type SixelSectorMapComponent struct {
	view         *SixelView
	proxyAPI     api.ProxyAPI
	currentSector int
	sectorData   map[int]api.SectorInfo
	width        int
	height       int
	pendingSixel string // Store sixel data for output
}

// NewSixelSectorMapComponent creates a new sixel-based sector map component
func NewSixelSectorMapComponent() *SixelSectorMapComponent {
	mapView := NewSixelView()
	mapView.SetTitle("Sector Map (Sixel)")
	
	smc := &SixelSectorMapComponent{
		view:         mapView,
		sectorData:   make(map[int]api.SectorInfo),
		width:        30,
		height:       20,
	}
	
	// Set initial fallback text
	smc.view.SetSixelData("", "[cyan]Sixel Sector Map[-]\n\n[yellow]Connect and load database to see sector map[-]")
	
	return smc
}

// GetView returns the tview component
func (smc *SixelSectorMapComponent) GetView() tview.Primitive {
	return smc.view
}

// SetProxyAPI sets the API reference for accessing game data
func (smc *SixelSectorMapComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
	smc.proxyAPI = proxyAPI
	if proxyAPI != nil {
		smc.setWaitingMessage()
	}
}

// LoadRealMapData loads real sector data from the API
func (smc *SixelSectorMapComponent) LoadRealMapData() {
	smc.loadRealMapData()
}

// loadRealMapData loads real sector data from the API
func (smc *SixelSectorMapComponent) loadRealMapData() {
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
func (smc *SixelSectorMapComponent) setWaitingMessage() {
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	waitingText := fmt.Sprintf("[cyan]Sixel Sector Map[-]\n\n[%s::bl]Waiting...[-]", 
		defaultColors.Waiting.String())
	smc.view.SetSixelData("", waitingText)
}

// UpdateCurrentSector updates the map with the current sector
func (smc *SixelSectorMapComponent) UpdateCurrentSector(sectorNumber int) {
	smc.currentSector = sectorNumber
	smc.refreshMap()
}

// UpdateCurrentSectorWithInfo updates the map with full sector information
func (smc *SixelSectorMapComponent) UpdateCurrentSectorWithInfo(sectorInfo api.SectorInfo) {
	smc.currentSector = sectorInfo.Number
	smc.sectorData[sectorInfo.Number] = sectorInfo
	smc.refreshMap()
}

// refreshMap refreshes the entire map display
func (smc *SixelSectorMapComponent) refreshMap() {
	if smc.proxyAPI == nil {
		smc.view.SetSixelData("", "[red]No proxy API available[-]")
		return
	}
	
	if smc.currentSector == 0 {
		smc.view.SetSixelData("", "[yellow]Waiting for sector data...[-]")
		return
	}
	
	// Get current sector info and connected sectors
	currentInfo, hasCurrentInfo := smc.sectorData[smc.currentSector]
	if !hasCurrentInfo {
		var err error
		currentInfo, err = smc.proxyAPI.GetSectorInfo(smc.currentSector)
		if err != nil {
			smc.view.SetSixelData("", fmt.Sprintf("[red]Error: %s[-]", err.Error()))
			return
		}
		smc.sectorData[smc.currentSector] = currentInfo
	}
	
	// Get connected sector data
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	for _, sectorNum := range connectedSectors {
		if sectorNum > 0 {
			info, err := smc.proxyAPI.GetSectorInfo(sectorNum)
			if err == nil {
				smc.sectorData[sectorNum] = info
			}
		}
	}
	
	// Render the sixel map
	sixelOutput := smc.renderSixelMap()
	
	// Try embedding the sixel data directly in the tview content
	// This might allow tview to pass through the escape sequences
	smc.view.SetSixelData(sixelOutput, "")
	
	// Also log for debugging
	
	// Write debug file
	err := os.WriteFile("sector_map_sixel_debug.txt", []byte(sixelOutput), 0644)
	if err != nil {
	}
}

// getConnectedSectors gets the sector numbers connected to the given sector
func (smc *SixelSectorMapComponent) getConnectedSectors(sectorNum int) []int {
	if info, exists := smc.sectorData[sectorNum]; exists {
		return info.Warps
	}
	return []int{}
}

// renderSixelMap creates a sixel representation of the sector map
func (smc *SixelSectorMapComponent) renderSixelMap() string {
	if smc.currentSector == 0 {
		return ""
	}
	
	// Get current sector info
	currentInfo, exists := smc.sectorData[smc.currentSector]
	if !exists {
		return ""
	}
	
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	
	// Create a proper graphical directed graph visualization
	return smc.renderGraphicalSixelMap(currentInfo, connectedSectors)
}

// renderGraphicalSixelMap creates a true graphical directed graph using sixels
func (smc *SixelSectorMapComponent) renderGraphicalSixelMap(currentInfo api.SectorInfo, connectedSectors []int) string {
	// Create a proper pixel canvas
	canvasWidth := 150
	canvasHeight := 100
	canvas := NewSixelCanvas(canvasWidth, canvasHeight)
	
	// Clear canvas to background
	canvas.Clear()
	
	// Calculate positions for 3x3 grid layout
	gridCols := 3
	gridRows := 3
	cellWidth := canvasWidth / gridCols
	cellHeight := canvasHeight / gridRows
	
	// Position map for 3x3 grid
	positions := map[string][2]int{
		"NW": {cellWidth/2, cellHeight/2},
		"N":  {cellWidth + cellWidth/2, cellHeight/2},
		"NE": {2*cellWidth + cellWidth/2, cellHeight/2},
		"W":  {cellWidth/2, cellHeight + cellHeight/2},
		"C":  {cellWidth + cellWidth/2, cellHeight + cellHeight/2},
		"E":  {2*cellWidth + cellWidth/2, cellHeight + cellHeight/2},
		"SW": {cellWidth/2, 2*cellHeight + cellHeight/2},
		"S":  {cellWidth + cellWidth/2, 2*cellHeight + cellHeight/2},
		"SE": {2*cellWidth + cellWidth/2, 2*cellHeight + cellHeight/2},
	}
	
	// Place current sector at center
	centerPos := positions["C"]
	smc.drawSectorNodeOnCanvas(canvas, centerPos[0], centerPos[1], smc.currentSector, true, currentInfo)
	
	// Place connected sectors around the center
	positionKeys := []string{"N", "E", "S", "W", "NE", "SE", "SW", "NW"}
	sectorPositions := make(map[int][2]int)
	sectorPositions[smc.currentSector] = centerPos
	
	for i, sector := range connectedSectors {
		if i < len(positionKeys) {
			pos := positions[positionKeys[i]]
			sectorPositions[sector] = pos
			
			info, hasInfo := smc.sectorData[sector]
			if hasInfo {
				smc.drawSectorNodeOnCanvas(canvas, pos[0], pos[1], sector, false, info)
			} else {
				// Create minimal info for unknown sectors
				unknownInfo := api.SectorInfo{Number: sector}
				smc.drawSectorNodeOnCanvas(canvas, pos[0], pos[1], sector, false, unknownInfo)
			}
		}
	}
	
	// Draw directed edges (warp connections)
	smc.drawWarpConnectionsOnCanvas(canvas, sectorPositions)
	
	// Convert canvas to sixel format
	return canvas.RenderToSixel()
}

// drawSectorNodeOnCanvas draws a graphical node representing a sector on the canvas
func (smc *SixelSectorMapComponent) drawSectorNodeOnCanvas(canvas *SixelCanvas, x, y, sectorNum int, isCurrent bool, info api.SectorInfo) {
	nodeRadius := 12 // pixels
	
	// Choose colors based on sector properties (using color indices)
	var fillColor, borderColor int
	if isCurrent {
		fillColor = 1  // Yellow
		borderColor = 6 // Dark gray border
	} else if info.HasTraders > 0 {
		fillColor = 3  // Blue for ports
		borderColor = 6
	} else if info.NavHaz > 0 {
		fillColor = 5  // Red for dangerous sectors
		borderColor = 6
	} else {
		fillColor = 2  // Green for empty sectors
		borderColor = 6
	}
	
	// Draw a circular node
	canvas.DrawFilledCircle(x, y, nodeRadius, fillColor, borderColor)
	
	// Add sector number as text overlay
	smc.drawSectorLabelOnCanvas(canvas, x, y, sectorNum)
}

// drawSectorLabelOnCanvas adds a sector number label on the canvas
func (smc *SixelSectorMapComponent) drawSectorLabelOnCanvas(canvas *SixelCanvas, x, y, sectorNum int) {
	// Draw sector number in white text
	sectorText := fmt.Sprintf("%d", sectorNum)
	textColor := 4 // White
	
	// Center the text approximately
	textWidth := len(sectorText) * 6 // Approximate character width
	textX := x - textWidth/2
	textY := y - 4 // Center vertically
	
	canvas.DrawText(textX, textY, sectorText, textColor)
}

// drawWarpConnectionsOnCanvas draws directed arrows showing warp connections on the canvas
func (smc *SixelSectorMapComponent) drawWarpConnectionsOnCanvas(canvas *SixelCanvas, sectorPositions map[int][2]int) {
	// Draw connections from current sector to its warps
	currentPos, hasCurrentPos := sectorPositions[smc.currentSector]
	if !hasCurrentPos {
		return
	}
	
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	
	for _, targetSector := range connectedSectors {
		targetPos, hasTargetPos := sectorPositions[targetSector]
		if hasTargetPos {
			// Calculate connection points on the edge of circles (not center to center)
			lineColor := 4 // White for connection lines
			nodeRadius := 12
			
			// Calculate direction vector
			dx := targetPos[0] - currentPos[0]
			dy := targetPos[1] - currentPos[1]
			distance := math.Sqrt(float64(dx*dx + dy*dy))
			
			if distance > 0 {
				// Normalize and scale to edge of circles
				edgeX1 := currentPos[0] + int(float64(dx)/distance*float64(nodeRadius))
				edgeY1 := currentPos[1] + int(float64(dy)/distance*float64(nodeRadius))
				edgeX2 := targetPos[0] - int(float64(dx)/distance*float64(nodeRadius))
				edgeY2 := targetPos[1] - int(float64(dy)/distance*float64(nodeRadius))
				
				// Draw directed arrow from current to target
				canvas.DrawArrow(edgeX1, edgeY1, edgeX2, edgeY2, lineColor)
				
				// Check for bidirectional connection
				if smc.hasDirectConnection(targetSector, smc.currentSector) {
					// Draw return arrow with slight offset to avoid overlap
					offsetX := int(float64(-dy)/distance * 3) // Perpendicular offset
					offsetY := int(float64(dx)/distance * 3)
					
					canvas.DrawArrow(edgeX2+offsetX, edgeY2+offsetY, 
									edgeX1+offsetX, edgeY1+offsetY, lineColor)
				}
			}
		}
	}
}


// renderFallbackText creates an enhanced text-based graphical fallback
func (smc *SixelSectorMapComponent) renderFallbackText(currentInfo api.SectorInfo, connectedSectors []int) string {
	var builder strings.Builder
	
	builder.WriteString("[cyan]Sixel Sector Map[-]\n")
	builder.WriteString("[dim](Enhanced Unicode Graphics)[-]\n\n")
	
	// Create a visual ASCII/Unicode sector map
	if len(connectedSectors) > 0 {
		// Build a 3x3 visual grid using Unicode box drawing characters
		grid := make(map[string]int)
		grid["C"] = smc.currentSector // Center
		
		// Place connected sectors around center
		positions := []string{"N", "E", "S", "W", "NE", "SE", "SW", "NW"}
		for i, sector := range connectedSectors {
			if i < len(positions) {
				grid[positions[i]] = sector
			}
		}
		
		// Render visual grid
		smc.renderUnicodeGrid(&builder, grid)
	} else {
		// Single sector display
		builder.WriteString("       ┌─────────┐\n")
		builder.WriteString(fmt.Sprintf("       │ [yellow]%7d[-] │\n", smc.currentSector))
		builder.WriteString("       │   YOU   │\n")
		builder.WriteString("       └─────────┘\n")
	}
	
	// Add legend
	builder.WriteString("\n[dim]Legend:[-]\n")
	builder.WriteString("[yellow]●[-] Current [cyan]●[-] Port [green]●[-] Empty [red]●[-] Danger\n")
	
	return builder.String()
}

// renderUnicodeGrid creates a visual grid using Unicode characters
func (smc *SixelSectorMapComponent) renderUnicodeGrid(builder *strings.Builder, grid map[string]int) {
	// Grid positions:
	// NW | N  | NE
	// W  | C  | E  
	// SW | S  | SE
	
	rows := [][]string{
		{"NW", "N", "NE"},
		{"W", "C", "E"},
		{"SW", "S", "SE"},
	}
	
	for rowIdx, row := range rows {
		// Top border
		if rowIdx == 0 {
			for colIdx, pos := range row {
				if _, exists := grid[pos]; exists {
					if colIdx == 0 {
						builder.WriteString("┌─────────")
					} else {
						builder.WriteString("┬─────────")
					}
				} else {
					builder.WriteString("          ")
				}
			}
			if grid["NE"] != 0 {
				builder.WriteString("┐")
			}
			builder.WriteString("\n")
		}
		
		// Sector content row
		for _, pos := range row {
			sector, exists := grid[pos]
			if exists {
				// Get sector color and symbol
				symbol := "●"
				color := "[green]" // Default
				
				if sector == smc.currentSector {
					color = "[yellow]"
					symbol = "★"
				} else if info, hasInfo := smc.sectorData[sector]; hasInfo {
					if info.HasTraders > 0 {
						color = "[cyan]"
						symbol = "●"
					} else if info.NavHaz > 0 {
						color = "[red]"
						symbol = "●"
					}
				}
				
				builder.WriteString(fmt.Sprintf("│%s%s %5d[-] ", color, symbol, sector))
			} else {
				builder.WriteString("          ")
			}
		}
		if grid["NE"] != 0 || grid["E"] != 0 || grid["SE"] != 0 {
			builder.WriteString("│")
		}
		builder.WriteString("\n")
		
		// Connection row (show arrows between sectors)
		for colIdx, pos := range row {
			if _, exists := grid[pos]; exists {
				// Show connection indicators
				connections := ""
				if colIdx < len(row)-1 {
					rightPos := row[colIdx+1]
					if rightSector, rightExists := grid[rightPos]; rightExists {
						leftSector := grid[pos]
						if smc.hasDirectConnection(leftSector, rightSector) {
							if smc.hasDirectConnection(rightSector, leftSector) {
								connections = "  ↔  "
							} else {
								connections = "  →  "
							}
						} else if smc.hasDirectConnection(rightSector, leftSector) {
							connections = "  ←  "
						} else {
							connections = "     "
						}
					} else {
						connections = "     "
					}
				}
				builder.WriteString("│   " + connections)
			} else {
				builder.WriteString("          ")
			}
		}
		if grid["NE"] != 0 || grid["E"] != 0 || grid["SE"] != 0 {
			builder.WriteString("│")
		}
		builder.WriteString("\n")
		
		// Bottom border or middle border
		if rowIdx < len(rows)-1 {
			// Middle border
			for colIdx, pos := range row {
				if _, exists := grid[pos]; exists {
					if colIdx == 0 {
						builder.WriteString("├─────────")
					} else {
						builder.WriteString("┼─────────")
					}
				} else {
					builder.WriteString("          ")
				}
			}
			if grid["NE"] != 0 || grid["E"] != 0 || grid["SE"] != 0 {
				builder.WriteString("┤")
			}
			builder.WriteString("\n")
		} else {
			// Bottom border
			for colIdx, pos := range row {
				if _, exists := grid[pos]; exists {
					if colIdx == 0 {
						builder.WriteString("└─────────")
					} else {
						builder.WriteString("┴─────────")
					}
				} else {
					builder.WriteString("          ")
				}
			}
			if grid["SE"] != 0 {
				builder.WriteString("┘")
			}
			builder.WriteString("\n")
		}
	}
}

// outputSixelToTerminal outputs sixel graphics directly to the terminal
func (smc *SixelSectorMapComponent) outputSixelToTerminal(sixelData string) {
	if len(sixelData) == 0 {
		return
	}
	
	// Save cursor position
	fmt.Print("\x1b[s")
	
	// Position cursor in the sector map panel area (right side of screen)
	// Approximate coordinates for the sector map panel based on typical layout
	fmt.Print("\x1b[3;85H")
	
	// Output sixel data directly to stdout
	// This will work with sixel-capable terminals
	fmt.Print(sixelData)
	
	// Restore cursor position
	fmt.Print("\x1b[u")
	
	// Force immediate output
	os.Stdout.Sync()
	
}

// storeSixelForOutput stores sixel data and triggers output after a delay
func (smc *SixelSectorMapComponent) storeSixelForOutput(sixelData string) {
	smc.pendingSixel = sixelData
	
	// Output after a short delay to let tview settle
	go func() {
		// Wait for tview to finish its rendering cycle
		// time.Sleep(200 * time.Millisecond)
		
		// Output the stored sixel data
		if smc.pendingSixel != "" {
			smc.outputSixelToTerminal(smc.pendingSixel)
		}
	}()
}

// FlushSixelOutput forces immediate output of any pending sixel data
func (smc *SixelSectorMapComponent) FlushSixelOutput() {
	if smc.pendingSixel != "" {
		smc.outputSixelToTerminal(smc.pendingSixel)
		smc.pendingSixel = ""
	}
}

// renderAdvancedSixelMap creates a more sophisticated sixel map
func (smc *SixelSectorMapComponent) renderAdvancedSixelMap() string {
	if smc.currentSector == 0 {
		return ""
	}
	
	_, exists := smc.sectorData[smc.currentSector]
	if !exists {
		return ""
	}
	
	connectedSectors := smc.getConnectedSectors(smc.currentSector)
	
	var sixel strings.Builder
	
	// Start sixel sequence
	sixel.WriteString("\x1bPq")
	
	// Enhanced color palette
	sixel.WriteString("#0;2;0;0;0")       // Black background
	sixel.WriteString("#1;2;100;100;0")   // Yellow - current sector
	sixel.WriteString("#2;2;0;80;0")      // Green - empty connected sectors
	sixel.WriteString("#3;2;0;0;100")     // Blue - port sectors
	
	// Simple representation - just show that sixel is working
	sixel.WriteString("#1") // Yellow color
	for i := 0; i < 8; i++ { // 8 sixel rows for the center sector
		sixel.WriteString("????") // Each ? represents 6 pixels
	}
	
	// Draw connected sectors around it
	sixel.WriteString("#2") // Green color for connected sectors
	for range connectedSectors {
		sixel.WriteString("??") // Smaller representation for connected sectors
	}
	
	// End sixel sequence
	sixel.WriteString("\x1b\\")
	
	return sixel.String()
}

// drawSixelSector draws a single sector in sixel format
func (smc *SixelSectorMapComponent) drawSixelSector(sixel *strings.Builder, x, y, sectorNum int, isCurrent bool, info api.SectorInfo) {
	// Choose color based on sector type
	var colorCode string
	if isCurrent {
		colorCode = "#1" // Yellow
	} else if info.HasTraders > 0 {
		colorCode = "#3" // Blue for ports
	} else if info.NavHaz > 0 {
		colorCode = "#5" // Red for dangerous sectors
	} else {
		colorCode = "#2" // Green for empty sectors
	}
	
	sixel.WriteString(colorCode)
	
	// Move to position and draw a filled rectangle
	// Simplified: draw a small rectangle for each sector
	for row := 0; row < 8; row++ { // 8 sixel rows per sector
		sixel.WriteString("???") // 3 characters = 18 pixels wide
	}
}

// drawSixelConnections draws connection lines between sectors
func (smc *SixelSectorMapComponent) drawSixelConnections(sixel *strings.Builder, positions map[string][2]int, connectedSectors []int) {
	// Use white color for connections
	sixel.WriteString("#4")
	
	// Draw simple connection lines (simplified representation)
	// In a real implementation, you'd calculate exact pixel positions
	// and draw lines between connected sectors
	
	for _, sector := range connectedSectors {
		if smc.hasDirectConnection(smc.currentSector, sector) {
			// Draw a simple connection indicator
			sixel.WriteString("?") // Placeholder for connection line
		}
	}
}

// hasDirectConnection checks if sector1 has a direct warp to sector2
func (smc *SixelSectorMapComponent) hasDirectConnection(sector1, sector2 int) bool {
	if info, exists := smc.sectorData[sector1]; exists {
		for _, warp := range info.Warps {
			if warp == sector2 {
				return true
			}
		}
	}
	return false
}

// colorToString converts tcell.Color to tview color string
func (smc *SixelSectorMapComponent) colorToString(color tcell.Color) string {
	r, g, b := color.RGB()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}