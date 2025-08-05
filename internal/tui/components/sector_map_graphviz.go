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
	"twist/internal/debug"

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
	gsm := &GraphvizSectorMap{
		Box:         tview.NewBox(),
		sectorData:  make(map[int]api.SectorInfo),
		needsRedraw: true,
		hasBorder:   true,
		sixelLayer:  sixelLayer,
		regionID:    "sector_map", // Unique ID for this component
	}
	gsm.SetBorder(true).SetTitle("Sector Map (Graphviz)")
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
	// Skip Box.DrawForSubclass to avoid screen clearing that causes flicker
	// Instead, manually draw border if needed
	if gsm.hasBorder {
		gsm.drawCustomBorder(screen)
	}

	x, y, width, height := gsm.GetInnerRect()
	debug.Log("GraphvizSectorMap: Draw() GetInnerRect returned x=%d, y=%d, width=%d, height=%d", x, y, width, height)

	if width <= 0 || height <= 0 {
		debug.Log("GraphvizSectorMap: Invalid dimensions, skipping draw")
		return
	}

	// Check if dimensions changed and invalidate cache if needed
	if gsm.cachedWidth != width || gsm.cachedHeight != height {
		debug.Log("GraphvizSectorMap: Dimensions changed from %dx%d to %dx%d, clearing caches", gsm.cachedWidth, gsm.cachedHeight, width, height)
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
			debug.Log("GraphvizSectorMap: Failed to decode PNG: %v", err)
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
			debug.Log("GraphvizSectorMap: Failed to encode sixel: %v", err)
			return
		}

		gsm.cachedSixel = buf.String()
		debug.Log("GraphvizSectorMap: Generated and cached sixel data, size: %d bytes", len(gsm.cachedSixel))
	}

	// Register with the sixel layer instead of direct TTY writing
	region := &SixelRegion{
		X:         x,
		Y:         y + 2, // Offset to avoid border
		Width:     width,
		Height:    height,
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
	err := g.AddVertex(gsm.currentSector)
	if err != nil {
		return nil, fmt.Errorf("failed to add current sector vertex: %w", err)
	}

	// Build complete graph with all warp connections
	// Step 1: Add all first-level vertices and edges from current sector
	for _, warpSector := range currentInfo.Warps {
		if warpSector <= 0 {
			continue
		}
		g.AddVertex(warpSector) // Ignore errors - vertex might already exist
		g.AddEdge(gsm.currentSector, warpSector) // Ignore errors - edge might already exist
	}

	// Step 2: Fetch warp info for all first-level sectors and add their connections
	for _, warpSector := range currentInfo.Warps {
		if warpSector <= 0 {
			continue
		}

		// Get warp sector info
		warpInfo, err := gsm.proxyAPI.GetSectorInfo(warpSector)
		if err != nil {
			continue // Skip sectors we can't get info for
		}
		gsm.sectorData[warpSector] = warpInfo

		// Add all connections from this sector
		for _, targetSector := range warpInfo.Warps {
			if targetSector <= 0 {
				continue
			}
			g.AddVertex(targetSector) // Ignore errors - vertex might already exist
			g.AddEdge(warpSector, targetSector) // Ignore errors - edge might already exist

			// Store basic info for target sectors (avoid infinite recursion)
			if _, exists := gsm.sectorData[targetSector]; !exists {
				gsm.sectorData[targetSector] = api.SectorInfo{Number: targetSector}
			}
		}
	}

	return g, nil
}

// generateGraphvizImage creates a PNG image from the graph using graphviz
func (gsm *GraphvizSectorMap) generateGraphvizImage(g graph.Graph[int, int], componentWidth, componentHeight int) ([]byte, error) {
	debug.Log("GraphvizSectorMap: generateGraphvizImage called with component dimensions %dx%d", componentWidth, componentHeight)
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
		debug.Log("GraphvizSectorMap: Failed to set default edge color: %v", err)
	}
	
	// Set default node attributes with visible borders and rounded corners
	_, err = gvGraph.Attr(int(cgraph.NODE), "style", "filled,rounded")
	if err != nil {
		debug.Log("GraphvizSectorMap: Failed to set default node style: %v", err)
	}
	_, err = gvGraph.Attr(int(cgraph.NODE), "penwidth", "3")
	if err != nil {
		debug.Log("GraphvizSectorMap: Failed to set default node penwidth: %v", err)
	}
	_, err = gvGraph.Attr(int(cgraph.NODE), "color", "white")
	if err != nil {
		debug.Log("GraphvizSectorMap: Failed to set default node border color: %v", err)
	}

	// Calculate aspect ratio - we expect much more height than width
	aspectRatio := float64(componentWidth) / float64(componentHeight)
	debug.Log("GraphvizSectorMap: Component aspect ratio: %.3f (width=%d, height=%d)", aspectRatio, componentWidth, componentHeight)

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
		} else if exists && sectorInfo.HasTraders > 0 {
			portType := sectorInfo.PortType
			if portType == "" {
				portType = fmt.Sprintf("T%d", sectorInfo.HasTraders)
			}
			label = fmt.Sprintf("%d\\n(%s)", sector, portType)
			fillColor = "lightblue"
		} else {
			label = fmt.Sprintf("%d", sector)
			fillColor = "gray"
		}

		node, err := gvGraph.CreateNodeByName(fmt.Sprintf("s%d", sector))
		if err != nil {
			debug.Log("GraphvizSectorMap: Failed to create node for sector %d: %v", sector, err)
			continue
		}

		node.SetLabel(label)
		node.SetFillColor(fillColor)
		node.SetShape("box")
		// DO NOT set fixed size - let graphviz size based on content
		node.SetFontSize(18.0)     // Large readable font
		node.SetFontColor("black") // Black text on colored background

		gvNodes[sector] = node
	}

	// Add edges using the adjacency map, avoiding duplicates for bidirectional edges
	debug.Log("GraphvizSectorMap: AdjacencyMap has %d sources", len(adjacencyMap))

	edgeCount := 0
	processedEdges := make(map[string]bool) // Track processed edge pairs

	for source, targets := range adjacencyMap {
		debug.Log("GraphvizSectorMap: Source %d has %d targets", source, len(targets))
		sourceNode, sourceExists := gvNodes[source]
		if !sourceExists {
			debug.Log("GraphvizSectorMap: Source node %d not found in gvNodes", source)
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
				debug.Log("GraphvizSectorMap: Skipping duplicate edge pair %s", edgeKey)
				continue
			}
			processedEdges[edgeKey] = true

			targetNode, targetExists := gvNodes[target]
			if !targetExists {
				debug.Log("GraphvizSectorMap: Target node %d not found in gvNodes", target)
				continue
			}

			edge, err := gvGraph.CreateEdgeByName("", sourceNode, targetNode)
			if err != nil {
				debug.Log("GraphvizSectorMap: Failed to create edge %d->%d: %v", source, target, err)
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
					debug.Log("GraphvizSectorMap: Created bidirectional edge %d<->%d", source, target)
				} else {
					edge.SetDir("forward")      // Unidirectional arrow
					edge.SetArrowHead("normal") // Standard arrow shape
					debug.Log("GraphvizSectorMap: Created unidirectional edge %d->%d", source, target)
				}
			} else {
				edge.SetDir("forward")      // Default to unidirectional
				edge.SetArrowHead("normal") // Standard arrow shape
				debug.Log("GraphvizSectorMap: Created default unidirectional edge %d->%d", source, target)
			}

			edgeCount++
		}
	}

	debug.Log("GraphvizSectorMap: Created %d total edges", edgeCount)

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
		debug.Log("GraphvizSectorMap: Failed to save debug file: %v", err)
	} else {
		debug.Log("GraphvizSectorMap: Saved warp analysis to /tmp/sector_debug.txt")
	}

	// Save DOT file for debugging
	var dotBuf bytes.Buffer
	err = gv.Render(ctx, gvGraph, "dot", &dotBuf)
	if err == nil {
		if err := os.WriteFile("/tmp/sector_map.dot", dotBuf.Bytes(), 0644); err != nil {
			debug.Log("GraphvizSectorMap: Failed to save DOT file: %v", err)
		} else {
			debug.Log("GraphvizSectorMap: Saved DOT file to /tmp/sector_map.dot")
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
		debug.Log("GraphvizSectorMap: Command line neato failed: %v, output: %s", err, buf.String())
		// Fallback to library rendering as last resort
		buf.Reset()
		err = gv.Render(ctx, gvGraph, graphviz.PNG, &buf)
		if err != nil {
			return nil, fmt.Errorf("both command line and library rendering failed: %w", err)
		}
		debug.Log("GraphvizSectorMap: Used library PNG rendering as fallback")
	} else {
		debug.Log("GraphvizSectorMap: Successfully used command line neato, PNG size: %d bytes", buf.Len())
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

	// Convert character dimensions to pixel dimensions based on typical monospace font metrics
	// For a 12pt monospace font at 72 DPI (more appropriate for terminal character sizing):
	// - Character width: ~7.2 pixels (12pt * 72dpi / 72pts_per_inch * 0.6 width_ratio)
	// - Character height: ~15.6 pixels (12pt * 72dpi / 72pts_per_inch * 1.3 line_height)
	fontSize := 12.0                // Font size in points
	dpi := 72.0                    // Lower DPI for more appropriate terminal sizing
	pointsPerInch := 72.0          // Standard points per inch
	monospaceWidthRatio := 0.6     // Monospace chars are typically 60% of their height
	lineHeightRatio := 1.3         // Line height is typically 130% of font size
	
	pixelsPerPoint := dpi / pointsPerInch
	charHeightPixels := fontSize * pixelsPerPoint * lineHeightRatio
	charWidthPixels := fontSize * pixelsPerPoint * monospaceWidthRatio
	
	componentWidthPixels := int(float64(componentWidth) * charWidthPixels)
	componentHeightPixels := int(float64(componentHeight) * charHeightPixels)

	debug.Log("GraphvizSectorMap: Font metrics - charWidth=%.1fpx, charHeight=%.1fpx", charWidthPixels, charHeightPixels)

	// Scale to fit the estimated pixel component size while maintaining aspect ratio
	scaleX := float64(componentWidthPixels) / float64(naturalWidth)
	scaleY := float64(componentHeightPixels) / float64(naturalHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Apply a conservative scaling factor - use 85% to leave some margin for borders/padding
	scale = scale * 0.85

	// Ensure reasonable scale bounds to preserve quality and readability
	if scale < 0.4 {
		scale = 0.4 // Don't scale down too much or we lose readability
	}
	if scale > 1.5 {
		scale = 1.5 // Don't scale up too much or we get pixelation
	}

	newWidth := int(float64(naturalWidth) * scale)
	newHeight := int(float64(naturalHeight) * scale)

	debug.Log("GraphvizSectorMap: Natural size %dx%d pixels, component size %dx%d chars (%dx%d pixels)", 
		naturalWidth, naturalHeight, componentWidth, componentHeight, componentWidthPixels, componentHeightPixels)
	debug.Log("GraphvizSectorMap: Calculated scale factors: X=%.2f, Y=%.2f, chosen=%.2f, adjusted=%.2f", scaleX, scaleY, scale/0.85, scale)
	debug.Log("GraphvizSectorMap: Scaling to %dx%d", newWidth, newHeight)

	// Scale the image using golang.org/x/image/draw
	scaledImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.BiLinear.Scale(scaledImg, scaledImg.Bounds(), img, bounds, xdraw.Over, nil)

	// Create a panel-sized canvas to center the scaled image
	panelImg := image.NewRGBA(image.Rect(0, 0, componentWidthPixels, componentHeightPixels))
	
	// Fill with black background
	black := color.RGBA{0, 0, 0, 255}
	for y := 0; y < componentHeightPixels; y++ {
		for x := 0; x < componentWidthPixels; x++ {
			panelImg.Set(x, y, black)
		}
	}
	
	// Calculate position to center the scaled image in the panel
	centerX := (componentWidthPixels - newWidth) / 2
	centerY := (componentHeightPixels - newHeight) / 2
	
	// Ensure we don't go negative (clip if image is larger than panel)
	if centerX < 0 {
		centerX = 0
	}
	if centerY < 0 {
		centerY = 0
	}
	
	// Calculate clipping if the scaled image is larger than the panel
	srcX := 0
	srcY := 0
	targetWidth := newWidth
	targetHeight := newHeight
	
	if centerX + newWidth > componentWidthPixels {
		targetWidth = componentWidthPixels - centerX
	}
	if centerY + newHeight > componentHeightPixels {
		targetHeight = componentHeightPixels - centerY
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

	debug.Log("GraphvizSectorMap: Centered %dx%d scaled image in %dx%d panel at (%d,%d)", 
		newWidth, newHeight, componentWidthPixels, componentHeightPixels, centerX, centerY)

	return panelBuf.Bytes(), nil
}

// Note: outputSixelImage and outputSixelToTerminal methods removed - now handled in renderSixelInPanel
