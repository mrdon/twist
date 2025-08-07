package components

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"twist/internal/api"
	"twist/internal/theme"

	"github.com/BourgeoisBear/rasterm"
	"github.com/dominikbraun/graph"
	"github.com/gdamore/tcell/v2"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/rivo/tview"
	xdraw "golang.org/x/image/draw"
)

// GraphvizSectorMap manages the sector map visualization using graphviz and sixels
type GraphvizSectorMap struct {
	*tview.Box
	proxyAPI      api.ProxyAPI
	currentSector int
	sectorData    map[int]api.SectorInfo
	sectorLevels  map[int]int // Track which level each sector is at (0=current, 1-5=hop levels)
	cachedImage   []byte
	cachedSixel   string
	cachedWidth   int
	cachedHeight  int
	needsRedraw   bool
	hasBorder     bool
	sixelLayer    *SixelLayer
	regionID      string
}

// NewGraphvizSectorMap creates a new graphviz-based sector map component
func NewGraphvizSectorMap(sixelLayer *SixelLayer) *GraphvizSectorMap {
	// Get theme colors for panels
	currentTheme := theme.Current()
	panelColors := currentTheme.PanelColors()
	
	box := tview.NewBox()
	box.SetBackgroundColor(panelColors.Background)
	box.SetBorderColor(panelColors.Border)
	box.SetTitleColor(panelColors.Title)
	
	gsm := &GraphvizSectorMap{
		Box:          box,
		sectorData:   make(map[int]api.SectorInfo),
		sectorLevels: make(map[int]int),
		needsRedraw:  true,
		hasBorder:    false,  // Disable tview border, use only content border
		sixelLayer:   sixelLayer,
		regionID:     "sector_map", // Unique ID for this component
	}
	gsm.SetBorder(false).SetTitle("Sector Map (Graphviz)")
	return gsm
}

// SetProxyAPI sets the API reference for accessing game data
func (gsm *GraphvizSectorMap) SetProxyAPI(proxyAPI api.ProxyAPI) {
	gsm.proxyAPI = proxyAPI
	gsm.needsRedraw = true
	gsm.cachedSixel = "" // Clear sixel cache
}

// Draw renders the graphviz sector map using the proven sixel technique
func (gsm *GraphvizSectorMap) Draw(screen tcell.Screen) {
	// Draw the background first to ensure proper theming
	gsm.Box.DrawForSubclass(screen, gsm)

	x, y, width, height := gsm.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	// Check if dimensions changed and invalidate cache if needed
	if gsm.cachedWidth != width || gsm.cachedHeight != height {
		gsm.cachedImage = nil
		gsm.cachedSixel = ""
		
		// Clear the region thoroughly before updating dimensions
		if gsm.sixelLayer != nil {
			gsm.sixelLayer.ClearRegion(gsm.regionID)
			gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
		}
		
		gsm.cachedWidth = width
		gsm.cachedHeight = height
		gsm.needsRedraw = true
	}

	// Generate map image and sixel if needed
	if gsm.needsRedraw || gsm.cachedImage == nil || gsm.cachedSixel == "" {
		if gsm.currentSector > 0 && gsm.proxyAPI != nil {
			// Clear the region before generating new content to prevent artifacts
			if gsm.sixelLayer != nil {
				gsm.sixelLayer.ClearRegion(gsm.regionID)
				gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
			}
			
			// Generate new graphviz image
			g, err := gsm.buildSectorGraph()
			if err == nil {
				imageData, err := gsm.generateGraphvizImage(g, width, height)
				if err == nil {
					gsm.cachedImage = imageData
					gsm.cachedSixel = "" // Clear sixel cache when image changes
					gsm.needsRedraw = false
				}
			}
		}
	}

	// Register sixel region with the layer if we have cached image
	if gsm.cachedImage != nil && gsm.sixelLayer != nil {
		gsm.registerSixelRegion(x, y, width, height)
	} else {
		// Show status text
		gsm.drawStatusText(screen, x, y, width, height, "Generating sector map...")
		// Hide sixel region when not ready
		if gsm.sixelLayer != nil {
			gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
		}
	}
}

// registerSixelRegion registers this component's sixel region with the layer
func (gsm *GraphvizSectorMap) registerSixelRegion(x, y, width, height int) {
	// Generate sixel data if not cached
	if gsm.cachedSixel == "" {
		// Decode the cached PNG image
		img, err := png.Decode(bytes.NewReader(gsm.cachedImage))
		if err != nil {
			return
		}

		// Convert to paletted image using Go's built-in Plan9 palette
		bounds := img.Bounds()
		palettedImg := image.NewPaletted(bounds, palette.Plan9)
		draw.FloydSteinberg.Draw(palettedImg, bounds, img, bounds.Min)

		// Encode as sixel using rasterm
		var buf bytes.Buffer
		err = rasterm.SixelWriteImage(&buf, palettedImg)
		if err != nil {
			return
		}

		gsm.cachedSixel = buf.String()
	}

	// Register with the sixel layer instead of direct TTY writing
	region := &SixelRegion{
		X:         x,
		Y:         y + 1, // Minimal offset to avoid title overlap
		Width:     width,
		Height:    height - 1, // Minimal height adjustment
		SixelData: gsm.cachedSixel,
		Visible:   true,
	}

	gsm.sixelLayer.AddRegion(gsm.regionID, region)
}

// drawCustomBorder draws border without clearing background
func (gsm *GraphvizSectorMap) drawCustomBorder(screen tcell.Screen) {
	x, y, width, height := gsm.GetRect()
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	// Top border
	for i := x; i < x+width; i++ {
		screen.SetContent(i, y, '─', nil, style)
	}

	// Bottom border
	for i := x; i < x+width; i++ {
		screen.SetContent(i, y+height-1, '─', nil, style)
	}

	// Left border
	for i := y; i < y+height; i++ {
		screen.SetContent(x, i, '│', nil, style)
	}

	// Right border
	for i := y; i < y+height; i++ {
		screen.SetContent(x+width-1, i, '│', nil, style)
	}

	// Corners
	screen.SetContent(x, y, '┌', nil, style)
	screen.SetContent(x+width-1, y, '┐', nil, style)
	screen.SetContent(x, y+height-1, '└', nil, style)
	screen.SetContent(x+width-1, y+height-1, '┘', nil, style)

	// Title
	if gsm.Box != nil {
		// Use reflection or a different approach since GetTitle() might not be available
		titleX := x + 2
		title := "Sector Map (Graphviz)" // Hardcode for now
		for i, r := range title {
			if titleX+i < x+width-1 {
				screen.SetContent(titleX+i, y, r, nil, style)
			}
		}
	}
}

// drawStatusText draws simple status text in the panel
func (gsm *GraphvizSectorMap) drawStatusText(screen tcell.Screen, x, y, width, height int, text string) {
	style := tcell.StyleDefault.Foreground(tcell.ColorYellow)

	// Center the text
	startX := x + (width-len(text))/2
	startY := y + height/2

	for i, char := range text {
		if startX+i < x+width {
			screen.SetContent(startX+i, startY, char, nil, style)
		}
	}
}

// UpdateCurrentSector updates the map with the current sector
func (gsm *GraphvizSectorMap) UpdateCurrentSector(sectorNumber int) {
	if gsm.currentSector != sectorNumber {
		gsm.currentSector = sectorNumber
		gsm.needsRedraw = true
		gsm.cachedSixel = "" // Clear sixel cache
		gsm.sectorLevels = make(map[int]int) // Clear sector levels for fresh tracking

		// Hide the region while regenerating to prevent overlap
		if gsm.sixelLayer != nil {
			gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
		}
	}
}

// UpdateCurrentSectorWithInfo updates the map with full sector information
func (gsm *GraphvizSectorMap) UpdateCurrentSectorWithInfo(sectorInfo api.SectorInfo) {
	if gsm.currentSector != sectorInfo.Number {
		gsm.currentSector = sectorInfo.Number
		gsm.needsRedraw = true
		gsm.cachedSixel = "" // Clear sixel cache
		gsm.sectorLevels = make(map[int]int) // Clear sector levels for fresh tracking

		// Hide the region while regenerating to prevent overlap
		if gsm.sixelLayer != nil {
			gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
		}
	}
	gsm.sectorData[sectorInfo.Number] = sectorInfo
}

// LoadRealMapData loads real sector data from the API
func (gsm *GraphvizSectorMap) LoadRealMapData() {
	if gsm.proxyAPI == nil {
		return
	}

	playerInfo, err := gsm.proxyAPI.GetPlayerInfo()
	if err != nil {
		return
	}

	if playerInfo.CurrentSector <= 0 {
		return
	}

	if gsm.currentSector != playerInfo.CurrentSector {
		gsm.currentSector = playerInfo.CurrentSector
		gsm.needsRedraw = true
		gsm.cachedSixel = "" // Clear sixel cache
		gsm.sectorLevels = make(map[int]int) // Clear sector levels for fresh tracking
	}
}

// Note: refreshMap and renderSixelMap methods removed - now handled in Draw() method

// buildSectorGraph creates a graph structure using dominikbraun/graph
func (gsm *GraphvizSectorMap) buildSectorGraph() (graph.Graph[int, int], error) {
	// Create a new directed graph with proper hash function
	g := graph.New(func(i int) int { return i }, graph.Directed())

	// Get current sector info
	currentInfo, hasCurrentInfo := gsm.sectorData[gsm.currentSector]
	if !hasCurrentInfo {
		var err error
		currentInfo, err = gsm.proxyAPI.GetSectorInfo(gsm.currentSector)
		if err != nil {
			return nil, fmt.Errorf("failed to get current sector info: %w", err)
		}
		gsm.sectorData[gsm.currentSector] = currentInfo
	}

	// Add current sector as vertex
	var err error
	err = g.AddVertex(gsm.currentSector)
	if err != nil {
		return nil, fmt.Errorf("failed to add current sector vertex: %w", err)
	}

	// Build complete graph with all warp connections
	// Track which sectors we've fully processed to avoid infinite recursion
	processed := make(map[int]bool)
	
	// Clear and initialize sector levels tracking
	gsm.sectorLevels = make(map[int]int)
	gsm.sectorLevels[gsm.currentSector] = 0 // Current sector is level 0
	
	// Step 1: Add all first-level vertices and edges from current sector
	for _, warpSector := range currentInfo.Warps {
		if warpSector <= 0 {
			continue
		}
		g.AddVertex(warpSector) // Ignore errors - vertex might already exist
		g.AddEdge(gsm.currentSector, warpSector) // Ignore errors - edge might already exist
		gsm.sectorLevels[warpSector] = 1 // First level sectors
	}
	processed[gsm.currentSector] = true

	// Step 2: Fetch warp info for all first-level sectors and add their connections
	secondLevelSectors := make([]int, 0)
	for _, warpSector := range currentInfo.Warps {
		if warpSector <= 0 || processed[warpSector] {
			continue
		}

		// Get warp sector info
		warpInfo, err := gsm.proxyAPI.GetSectorInfo(warpSector)
		if err != nil {
			continue // Skip sectors we can't get info for
		}
		gsm.sectorData[warpSector] = warpInfo
		processed[warpSector] = true

		// Add all connections from this sector
		for _, targetSector := range warpInfo.Warps {
			if targetSector <= 0 {
				continue
			}
			g.AddVertex(targetSector) // Ignore errors - vertex might already exist
			g.AddEdge(warpSector, targetSector) // Ignore errors - edge might already exist

			// Track sectors for next level processing if not already processed
			if !processed[targetSector] {
				secondLevelSectors = append(secondLevelSectors, targetSector)
				// Set level for new sectors if not already set
				if _, exists := gsm.sectorLevels[targetSector]; !exists {
					gsm.sectorLevels[targetSector] = 2 // Second level sectors
				}
			}
		}
	}

	// Step 3: Fetch warp info for all second-level sectors and add their connections (3rd level)
	thirdLevelSectors := make([]int, 0)
	for _, secondLevelSector := range secondLevelSectors {
		if secondLevelSector <= 0 || processed[secondLevelSector] {
			continue
		}

		// Get second-level sector info
		secondLevelInfo, err := gsm.proxyAPI.GetSectorInfo(secondLevelSector)
		if err != nil {
			continue // Skip sectors we can't get info for
		}
		gsm.sectorData[secondLevelSector] = secondLevelInfo
		processed[secondLevelSector] = true

		// Add all connections from this second-level sector (creating 3rd level)
		for _, thirdLevelSector := range secondLevelInfo.Warps {
			if thirdLevelSector <= 0 {
				continue
			}
			g.AddVertex(thirdLevelSector) // Ignore errors - vertex might already exist
			g.AddEdge(secondLevelSector, thirdLevelSector) // Ignore errors - edge might already exist

			// Track sectors for next level processing if not already processed
			if !processed[thirdLevelSector] {
				thirdLevelSectors = append(thirdLevelSectors, thirdLevelSector)
				// Set level for new sectors if not already set
				if _, exists := gsm.sectorLevels[thirdLevelSector]; !exists {
					gsm.sectorLevels[thirdLevelSector] = 3 // Third level sectors
				}
			}
		}
	}

	// Step 4: Fetch warp info for all third-level sectors and add their connections (4th level)
	fourthLevelSectors := make([]int, 0)
	for _, thirdLevelSector := range thirdLevelSectors {
		if thirdLevelSector <= 0 || processed[thirdLevelSector] {
			continue
		}

		// Get third-level sector info
		thirdLevelInfo, err := gsm.proxyAPI.GetSectorInfo(thirdLevelSector)
		if err != nil {
			continue // Skip sectors we can't get info for
		}
		gsm.sectorData[thirdLevelSector] = thirdLevelInfo
		processed[thirdLevelSector] = true

		// Add all connections from this third-level sector (creating 4th level)
		for _, fourthLevelSector := range thirdLevelInfo.Warps {
			if fourthLevelSector <= 0 {
				continue
			}
			g.AddVertex(fourthLevelSector) // Ignore errors - vertex might already exist
			g.AddEdge(thirdLevelSector, fourthLevelSector) // Ignore errors - edge might already exist

			// Track sectors for next level processing if not already processed
			if !processed[fourthLevelSector] {
				fourthLevelSectors = append(fourthLevelSectors, fourthLevelSector)
				// Set level for new sectors if not already set
				if _, exists := gsm.sectorLevels[fourthLevelSector]; !exists {
					gsm.sectorLevels[fourthLevelSector] = 4 // Fourth level sectors
				}
			}
		}
	}

	// Step 5: Fetch warp info for all fourth-level sectors and add their connections (5th level)
	for _, fourthLevelSector := range fourthLevelSectors {
		if fourthLevelSector <= 0 || processed[fourthLevelSector] {
			continue
		}

		// Get fourth-level sector info
		fourthLevelInfo, err := gsm.proxyAPI.GetSectorInfo(fourthLevelSector)
		if err != nil {
			continue // Skip sectors we can't get info for
		}
		gsm.sectorData[fourthLevelSector] = fourthLevelInfo
		processed[fourthLevelSector] = true

		// Add all connections from this fourth-level sector (creating 5th level)
		for _, fifthLevelSector := range fourthLevelInfo.Warps {
			if fifthLevelSector <= 0 {
				continue
			}
			g.AddVertex(fifthLevelSector) // Ignore errors - vertex might already exist
			g.AddEdge(fourthLevelSector, fifthLevelSector) // Ignore errors - edge might already exist

			// Store basic info for fifth-level sectors only if not already processed
			// This prevents infinite expansion while allowing recursive connections
			if !processed[fifthLevelSector] {
				if _, exists := gsm.sectorData[fifthLevelSector]; !exists {
					gsm.sectorData[fifthLevelSector] = api.SectorInfo{Number: fifthLevelSector}
				}
				// Set level for new sectors if not already set - this is the outermost level
				if _, exists := gsm.sectorLevels[fifthLevelSector]; !exists {
					gsm.sectorLevels[fifthLevelSector] = 5 // Fifth level sectors (outermost)
				}
			}
		}
	}

	return g, nil
}

// generateGraphvizImage creates a PNG image from the graph using graphviz
func (gsm *GraphvizSectorMap) generateGraphvizImage(g graph.Graph[int, int], componentWidth, componentHeight int) ([]byte, error) {
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create graphviz instance: %w", err)
	}
	defer gv.Close()

	// Create a new graphviz graph
	gvGraph, err := gv.Graph()
	if err != nil {
		return nil, fmt.Errorf("failed to create graphviz graph: %w", err)
	}
	defer gvGraph.Close()

	// Use neato engine with increased spacing for better layout
	gvGraph.SetLayout("neato")          // Force-directed layout engine
	gvGraph.SetBackgroundColor("black") // Black background
	gvGraph.SetDPI(150.0)              // Higher DPI for better border rendering
	
	// Set default edge color to white for visibility on black background
	_, err = gvGraph.Attr(int(cgraph.EDGE), "color", "white")
	if err != nil {
	}
	
	// Set default node attributes with visible borders and rounded corners
	_, err = gvGraph.Attr(int(cgraph.NODE), "style", "filled,rounded")
	if err != nil {
	}
	_, err = gvGraph.Attr(int(cgraph.NODE), "penwidth", "3")
	if err != nil {
	}
	_, err = gvGraph.Attr(int(cgraph.NODE), "color", "white")
	if err != nil {
	}

	// Calculate aspect ratio - we expect much more height than width
	_ = float64(componentWidth) / float64(componentHeight)

	// Configure layout spacing for neato engine using proper neato attributes
	gvGraph.SetOverlap(false)      // Prevent node overlap
	gvGraph.SetSplines("true")     // Enable curved edges for better readability
	gvGraph.Set("center", "true")  // Center the graph
	
	// Use neato-specific attributes for better spacing
	gvGraph.Set("len", "3.0")           // Preferred edge length in inches - larger for more spacing
	gvGraph.Set("sep", "1.0")           // Margin around nodes when removing overlap 
	gvGraph.Set("defaultdist", "4.0")   // Distance between separate components
	gvGraph.Set("overlap_scaling", "2.0") // Scale layout to reduce overlap

	// Create a map of graphviz nodes
	gvNodes := make(map[int]*graphviz.Node)

	// Get adjacency map which contains all vertices as keys
	adjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get adjacency map: %w", err)
	}

	// Create graphviz nodes for each vertex
	for sector := range adjacencyMap {
		// Create node with sector information
		sectorInfo, exists := gsm.sectorData[sector]

		var label, fillColor string
		if sector == gsm.currentSector {
			label = fmt.Sprintf("YOU\\n%d", sector)
			fillColor = "yellow"
		} else if exists && len(sectorInfo.Warps) > 0 {
			// Explored sector - has warp data from database
			if sectorInfo.HasTraders > 0 {
				var portType string
				if sectorInfo.HasPort {
					// Get actual port type from API
					if gsm.proxyAPI != nil {
						if portData, err := gsm.proxyAPI.GetPortInfo(sector); err == nil && portData != nil {
							portType = portData.ClassType.String() // Show actual port type like "BBS"
						} else {
							portType = "PORT" // Port exists but couldn't get details
						}
					} else {
						portType = "PORT" // No API access
					}
				} else {
					portType = fmt.Sprintf("T%d", sectorInfo.HasTraders)
				}
				label = fmt.Sprintf("%d\\n(%s)", sector, portType)
				fillColor = "lightblue"
			} else {
				label = fmt.Sprintf("%d", sector)
				fillColor = "gray"
			}
		} else {
			// Unexplored sector - only known from warp references
			label = fmt.Sprintf("%d", sector)
			fillColor = "lightcoral"
		}

		node, err := gvGraph.CreateNodeByName(fmt.Sprintf("s%d", sector))
		if err != nil {
			continue
		}

		node.SetLabel(label)
		node.SetFillColor(fillColor)
		node.SetShape("box")
		// DO NOT set fixed size - let graphviz size based on content
		node.SetFontSize(18.0)     // Large readable font
		node.SetFontColor("black") // Black text on colored background

		// Apply dotted border style only to 5th level (outermost) sectors
		if level, exists := gsm.sectorLevels[sector]; exists && level == 5 {
			node.SetStyle("filled,rounded,dotted")
		} else {
			node.SetStyle("filled,rounded")
		}

		gvNodes[sector] = node
	}

	// Add edges using the adjacency map, avoiding duplicates for bidirectional edges

	edgeCount := 0
	processedEdges := make(map[string]bool) // Track processed edge pairs

	for source, targets := range adjacencyMap {
		sourceNode, sourceExists := gvNodes[source]
		if !sourceExists {
			continue
		}

		for target := range targets {
			// Create a unique key for this edge pair (always smaller->larger to avoid duplicates)
			var edgeKey string
			if source < target {
				edgeKey = fmt.Sprintf("%d-%d", source, target)
			} else {
				edgeKey = fmt.Sprintf("%d-%d", target, source)
			}

			// Skip if we've already processed this edge pair
			if processedEdges[edgeKey] {
				continue
			}
			processedEdges[edgeKey] = true

			targetNode, targetExists := gvNodes[target]
			if !targetExists {
				continue
			}

			edge, err := gvGraph.CreateEdgeByName("", sourceNode, targetNode)
			if err != nil {
				continue
			}

			// Style the edge with thinner lines and better arrow spacing
			edge.SetPenWidth(1.5) // Thinner line thickness
			edge.SetStyle("solid")
			edge.SetConstraint(true) // Keep layout constraints
			edge.SetArrowSize(0.8)   // Smaller arrows to reduce overlap with nodes

			// Check if it's a bidirectional connection
			if reverseTargets, exists := adjacencyMap[target]; exists {
				if _, isBidirectional := reverseTargets[source]; isBidirectional {
					edge.SetDir("both")         // Bidirectional arrows
					edge.SetArrowHead("normal") // Standard arrow shape
					edge.SetArrowTail("normal") // Standard arrow shape
				} else {
					edge.SetDir("forward")      // Unidirectional arrow
					edge.SetArrowHead("normal") // Standard arrow shape
				}
			} else {
				edge.SetDir("forward")      // Default to unidirectional
				edge.SetArrowHead("normal") // Standard arrow shape
			}

			edgeCount++
		}
	}


	// Save warp direction analysis for debugging
	var warpDebug strings.Builder
	warpDebug.WriteString("=== SECTOR WARP ANALYSIS ===\n\n")

	// List all sectors and their warps
	warpDebug.WriteString("Raw sector warp data:\n")
	for sector, info := range gsm.sectorData {
		warpDebug.WriteString(fmt.Sprintf("Sector %d warps to: %v\n", sector, info.Warps))
	}

	warpDebug.WriteString("\nAdjacency map analysis:\n")
	for source, targets := range adjacencyMap {
		warpDebug.WriteString(fmt.Sprintf("Source %d connects to: ", source))
		targetList := make([]int, 0, len(targets))
		for target := range targets {
			targetList = append(targetList, target)
		}
		warpDebug.WriteString(fmt.Sprintf("%v\n", targetList))
	}

	warpDebug.WriteString("\nBidirectional analysis:\n")
	for source, targets := range adjacencyMap {
		for target := range targets {
			if reverseTargets, exists := adjacencyMap[target]; exists {
				if _, isBidirectional := reverseTargets[source]; isBidirectional {
					warpDebug.WriteString(fmt.Sprintf("BIDIRECTIONAL: %d <-> %d\n", source, target))
				} else {
					warpDebug.WriteString(fmt.Sprintf("UNIDIRECTIONAL: %d -> %d (no reverse)\n", source, target))
				}
			} else {
				warpDebug.WriteString(fmt.Sprintf("UNIDIRECTIONAL: %d -> %d (target not in adjacency map)\n", source, target))
			}
		}
	}

	// Write to file
	if err := os.WriteFile("/tmp/sector_debug.txt", []byte(warpDebug.String()), 0644); err != nil {
	} else {
	}

	// Save DOT file for debugging
	var dotBuf bytes.Buffer
	err = gv.Render(ctx, gvGraph, "dot", &dotBuf)
	if err == nil {
		if err := os.WriteFile("/tmp/sector_map.dot", dotBuf.Bytes(), 0644); err != nil {
		} else {
		}
	}

	// Use command line graphviz as the primary approach since it renders borders properly
	// The go-graphviz library's WASM backend doesn't render borders correctly
	cmd := exec.Command("neato", "-Tpng", "/tmp/sector_map.dot")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	
	err = cmd.Run()
	if err != nil {
		// Fallback to library rendering as last resort
		buf.Reset()
		err = gv.Render(ctx, gvGraph, graphviz.PNG, &buf)
		if err != nil {
			return nil, fmt.Errorf("both command line and library rendering failed: %w", err)
		}
	} else {
	}

	// Validate PNG output has content
	if buf.Len() == 0 {
		return nil, fmt.Errorf("graphviz render produced no PNG output")
	}

	// Decode the natural-sized image
	img, err := png.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Get the natural dimensions
	bounds := img.Bounds()
	naturalWidth := bounds.Dx()
	naturalHeight := bounds.Dy()

	// Fixed font size approach - maintain consistent text size regardless of graph size
	targetFontSizePixels := 12.0   // Target font size in final rendered image (pixels)
	graphvizFontSizePoints := 18.0 // The font size we set in graphviz (from node.SetFontSize)
	graphvizDPI := 150.0          // The DPI we set in graphviz (from gvGraph.SetDPI)
	
	// Calculate what the graphviz font renders as in pixels
	graphvizFontPixels := (graphvizFontSizePoints / 72.0) * graphvizDPI
	
	// Calculate the scale needed to achieve our target font size
	fontScale := targetFontSizePixels / graphvizFontPixels

	// Calculate panel size in pixels using typical terminal character dimensions
	terminalFontSize := 11.0       // Typical terminal font size  
	terminalDPI := 96.0           // Standard screen DPI
	charWidthRatio := 0.6         // Monospace width ratio
	lineHeightRatio := 0.85       // Line height ratio
	
	pixelsPerPoint := terminalDPI / 72.0
	charHeightPixels := terminalFontSize * pixelsPerPoint * lineHeightRatio
	charWidthPixels := terminalFontSize * pixelsPerPoint * charWidthRatio
	
	adjustedHeight := componentHeight - 1 // Reserve space for title
	panelPixelWidth := int(float64(componentWidth) * charWidthPixels)
	panelPixelHeight := int(float64(adjustedHeight) * charHeightPixels)

	// Use the font-based scale as our primary scale
	scale := fontScale
	
	// But ensure we don't exceed panel bounds - if the scaled image would be too big, we'll crop
	scaledWidth := int(float64(naturalWidth) * scale)
	scaledHeight := int(float64(naturalHeight) * scale)
	shouldCrop := false
	
	if scaledWidth > panelPixelWidth || scaledHeight > panelPixelHeight {
		shouldCrop = true
	}
	
	// Set reasonable bounds on scaling to prevent extreme cases
	maxScale := 2.0 // Don't scale up too much
	minScale := 0.2 // Don't scale down too much - text becomes unreadable
	
	if scale > maxScale {
		scale = maxScale
	} else if scale < minScale {
		scale = minScale
	}

	newWidth := int(float64(naturalWidth) * scale)
	newHeight := int(float64(naturalHeight) * scale)

	// Scale the image using golang.org/x/image/draw
	scaledImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.BiLinear.Scale(scaledImg, scaledImg.Bounds(), img, bounds, xdraw.Over, nil)
	
	// Find the actual content bounds of the scaled image (non-black pixels)
	contentBounds := findContentBounds(scaledImg)

	// Create a panel-sized canvas to center the scaled image
	panelImg := image.NewRGBA(image.Rect(0, 0, panelPixelWidth, panelPixelHeight))
	
	// Fill with black background
	black := color.RGBA{0, 0, 0, 255}
	for y := 0; y < panelPixelHeight; y++ {
		for x := 0; x < panelPixelWidth; x++ {
			panelImg.Set(x, y, black)
		}
	}
	
	// Handle centering and cropping based on whether we're cropping or fitting
	var centerX, centerY, srcX, srcY, targetWidth, targetHeight int
	
	if shouldCrop || newWidth > panelPixelWidth || newHeight > panelPixelHeight {
		// Cropping mode: center the source image and crop edges that don't fit
		centerX = 0
		centerY = 0
		targetWidth = panelPixelWidth
		targetHeight = panelPixelHeight
		
		// Calculate which part of the source image to show (center crop)
		srcX = (newWidth - panelPixelWidth) / 2
		srcY = (newHeight - panelPixelHeight) / 2
		
		// Ensure source coordinates are not negative
		if srcX < 0 {
			srcX = 0
			targetWidth = newWidth
		}
		if srcY < 0 {
			srcY = 0
			targetHeight = newHeight
		}
		
		// Ensure target dimensions don't exceed panel or scaled image size
		if targetWidth > panelPixelWidth {
			targetWidth = panelPixelWidth
		}
		if targetHeight > panelPixelHeight {
			targetHeight = panelPixelHeight
		}
		if targetWidth > newWidth - srcX {
			targetWidth = newWidth - srcX
		}
		if targetHeight > newHeight - srcY {
			targetHeight = newHeight - srcY
		}
	} else {
		// Fitting mode: center the entire scaled image in the panel
		centerX = (panelPixelWidth - newWidth) / 2
		centerY = (panelPixelHeight - newHeight) / 2
		srcX = 0
		srcY = 0
		targetWidth = newWidth
		targetHeight = newHeight
	}
	
	// Draw the scaled image centered (and clipped if necessary) in the panel
	if targetWidth > 0 && targetHeight > 0 {
		targetRect := image.Rect(centerX, centerY, centerX+targetWidth, centerY+targetHeight)
		sourceRect := image.Rect(srcX, srcY, srcX+targetWidth, srcY+targetHeight)
		draw.Draw(panelImg, targetRect, scaledImg, sourceRect.Min, draw.Over)
	}

	// Encode the final panel-sized image
	var panelBuf bytes.Buffer
	err = png.Encode(&panelBuf, panelImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode panel PNG: %w", err)
	}

	// Re-encode the panel image before adding borders
	panelBuf.Reset()
	err = png.Encode(&panelBuf, panelImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode panel PNG before borders: %w", err)
	}

	// Add borders around the content area
	drawContentBorders(panelImg, contentBounds, centerX, centerY)

	// Final encode with borders
	panelBuf.Reset()
	err = png.Encode(&panelBuf, panelImg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode final panel PNG: %w", err)
	}

	return panelBuf.Bytes(), nil
}

// findContentBounds finds the bounding box of non-black pixels in an image
func findContentBounds(img *image.RGBA) image.Rectangle {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Check if pixel is not black (allowing for small variations)
			if a > 0 && (r > 1000 || g > 1000 || b > 1000) {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	// If no content found, return zero rectangle
	if minX > maxX || minY > maxY {
		return image.Rectangle{}
	}

	// Add a small padding around the content
	padding := 3
	minX = max(bounds.Min.X, minX-padding)
	minY = max(bounds.Min.Y, minY-padding)
	maxX = min(bounds.Max.X-1, maxX+padding)
	maxY = min(bounds.Max.Y-1, maxY+padding)

	return image.Rect(minX, minY, maxX+1, maxY+1)
}

// drawContentBorders draws high-tech styled borders around the content area in the panel
func drawContentBorders(panelImg *image.RGBA, contentBounds image.Rectangle, offsetX, offsetY int) {
	if contentBounds.Empty() {
		return
	}

	// Simple white border
	white := color.RGBA{255, 255, 255, 255}         // White lines
	
	// Adjust content bounds by the centering offset with small padding for simple border
	borderWidth := 3
	left := contentBounds.Min.X + offsetX - borderWidth
	top := contentBounds.Min.Y + offsetY - borderWidth
	right := contentBounds.Max.X + offsetX + borderWidth - 1
	bottom := contentBounds.Max.Y + offsetY + borderWidth - 1

	panelBounds := panelImg.Bounds()
	
	// Ensure borders are within the panel bounds
	left = max(0, left)
	top = max(0, top)
	right = min(panelBounds.Max.X-1, right)
	bottom = min(panelBounds.Max.Y-1, bottom)

	// Draw simple white line border
	// Top and bottom borders
	for x := left; x <= right; x++ {
		if x >= 0 && x < panelBounds.Max.X {
			if top >= 0 && top < panelBounds.Max.Y {
				panelImg.Set(x, top, white)
			}
			if bottom >= 0 && bottom < panelBounds.Max.Y && bottom != top {
				panelImg.Set(x, bottom, white)
			}
		}
	}

	// Left and right borders
	for y := top; y <= bottom; y++ {
		if y >= 0 && y < panelBounds.Max.Y {
			if left >= 0 && left < panelBounds.Max.X {
				panelImg.Set(left, y, white)
			}
			if right >= 0 && right < panelBounds.Max.X && right != left {
				panelImg.Set(right, y, white)
			}
		}
	}
}


// Note: outputSixelImage and outputSixelToTerminal methods removed - now handled in renderSixelInPanel
