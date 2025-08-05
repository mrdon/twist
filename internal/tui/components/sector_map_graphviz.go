package components

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/png"
	"os"
	"twist/internal/api"
	"twist/internal/debug"

	"github.com/BourgeoisBear/rasterm"
	"github.com/dominikbraun/graph"
	"github.com/gdamore/tcell/v2"
	"github.com/goccy/go-graphviz"
	"github.com/rivo/tview"
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
}

// NewGraphvizSectorMap creates a new graphviz-based sector map component
func NewGraphvizSectorMap() *GraphvizSectorMap {
	gsm := &GraphvizSectorMap{
		Box:         tview.NewBox(),
		sectorData:  make(map[int]api.SectorInfo),
		needsRedraw: true,
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
	gsm.Box.DrawForSubclass(screen, gsm)

	x, y, width, height := gsm.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	// Check if dimensions changed and invalidate cache if needed
	if gsm.cachedWidth != width || gsm.cachedHeight != height {
		debug.Log("GraphvizSectorMap: Dimensions changed from %dx%d to %dx%d, clearing caches", gsm.cachedWidth, gsm.cachedHeight, width, height)
		gsm.cachedImage = nil
		gsm.cachedSixel = ""
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

	// Render sixel graphics if we have cached image
	if gsm.cachedImage != nil {
		gsm.renderSixelInPanel(screen, x, y, width, height)
	} else {
		// Show status text
		gsm.drawStatusText(screen, x, y, width, height, "Generating sector map...")
	}
}

// renderSixelInPanel renders sixel graphics within the tview panel
func (gsm *GraphvizSectorMap) renderSixelInPanel(screen tcell.Screen, x, y, width, height int) {
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

	// Use the proven dual-method approach from working sixel implementation

	// Method 1: Try the original tview-sixel approach
	screen.ShowCursor(x, y+2)
	//fmt.Print(gsm.cachedSixel)
	screen.Sync()

	// Method 2: Bypass modern tview screen buffer isolation
	if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		defer tty.Close()
		// Position cursor and output sixel to bypass tview
		fmt.Fprintf(tty, "\x1b[%d;%dH%s", y+3, x+1, gsm.cachedSixel)
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
	}
}

// UpdateCurrentSectorWithInfo updates the map with full sector information
func (gsm *GraphvizSectorMap) UpdateCurrentSectorWithInfo(sectorInfo api.SectorInfo) {
	if gsm.currentSector != sectorInfo.Number {
		gsm.currentSector = sectorInfo.Number
		gsm.needsRedraw = true
		gsm.cachedSixel = "" // Clear sixel cache
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

	// Add connected sectors and their connections
	for _, warpSector := range currentInfo.Warps {
		if warpSector <= 0 {
			continue
		}

		// Add warp sector as vertex
		err = g.AddVertex(warpSector)
		if err != nil {
			debug.Log("GraphvizSectorMap: Failed to add vertex %d: %v", warpSector, err)
			continue
		}

		// Add edge from current sector to warp sector
		err = g.AddEdge(gsm.currentSector, warpSector)
		if err != nil {
			debug.Log("GraphvizSectorMap: Failed to add edge %d->%d: %v", gsm.currentSector, warpSector, err)
		} else {
			debug.Log("GraphvizSectorMap: Added edge %d->%d to dominikbraun graph", gsm.currentSector, warpSector)
		}

		// Get warp sector info to find its connections
		warpInfo, err := gsm.proxyAPI.GetSectorInfo(warpSector)
		if err != nil {
			debug.Log("GraphvizSectorMap: Failed to get info for sector %d: %v", warpSector, err)
			continue
		}
		gsm.sectorData[warpSector] = warpInfo

		// Add second-level connections (warps of warps)
		for _, secondLevelWarp := range warpInfo.Warps {
			if secondLevelWarp <= 0 || secondLevelWarp == gsm.currentSector {
				continue // Skip invalid sectors and avoid loops back to current
			}

			// Add second-level sector as vertex
			err = g.AddVertex(secondLevelWarp)
			if err != nil {
				debug.Log("GraphvizSectorMap: Failed to add second-level vertex %d: %v", secondLevelWarp, err)
				continue
			}

			// Add edge from warp sector to second-level warp
			err = g.AddEdge(warpSector, secondLevelWarp)
			if err != nil {
				debug.Log("GraphvizSectorMap: Failed to add second-level edge %d->%d: %v", warpSector, secondLevelWarp, err)
			} else {
				debug.Log("GraphvizSectorMap: Added second-level edge %d->%d to dominikbraun graph", warpSector, secondLevelWarp)
			}

			// Store basic info for the second-level sector (don't fetch full details to avoid deep recursion)
			if _, exists := gsm.sectorData[secondLevelWarp]; !exists {
				gsm.sectorData[secondLevelWarp] = api.SectorInfo{Number: secondLevelWarp}
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

	// Set graph attributes for layout that fits component dimensions
	aspectRatio := float64(componentWidth) / float64(componentHeight)

	// Choose layout direction based on aspect ratio
	if aspectRatio > 1.5 {
		gvGraph.SetRankDir("LR") // Left to right for wide components
	} else {
		gvGraph.SetRankDir("TB") // Top to bottom for tall/square components
	}

	// Set size based on component dimensions (scale down to fit)
	graphWidth := float64(componentWidth) * 0.15 // Scale to character units
	graphHeight := float64(componentHeight) * 0.3
	gvGraph.SetSize(graphWidth, graphHeight)

	gvGraph.SetNodeSeparator(0.2)       // Tight node separation
	gvGraph.SetRankSeparator(0.3)       // Tight rank separation
	gvGraph.SetBackgroundColor("black") // Black background

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

		var label, color, fillColor string
		if sector == gsm.currentSector {
			label = fmt.Sprintf("YOU\\n%d", sector)
			color = "red"
			fillColor = "yellow"
		} else if exists && sectorInfo.HasTraders > 0 {
			portType := sectorInfo.PortType
			if portType == "" {
				portType = fmt.Sprintf("T%d", sectorInfo.HasTraders)
			}
			label = fmt.Sprintf("%d\\n(%s)", sector, portType)
			color = "blue"
			fillColor = "lightblue"
		} else {
			label = fmt.Sprintf("%d", sector)
			color = "white"
			fillColor = "gray"
		}

		node, err := gvGraph.CreateNodeByName(fmt.Sprintf("s%d", sector))
		if err != nil {
			debug.Log("GraphvizSectorMap: Failed to create node for sector %d: %v", sector, err)
			continue
		}

		node.SetLabel(label)
		node.SetColor(color)
		node.SetFillColor(fillColor)
		node.SetStyle("filled")
		node.SetShape("box")
		node.SetPenWidth(2.0) // Make borders more visible

		gvNodes[sector] = node
	}

	// Add edges using the adjacency map we already have
	debug.Log("GraphvizSectorMap: AdjacencyMap has %d sources", len(adjacencyMap))

	edgeCount := 0
	for source, targets := range adjacencyMap {
		debug.Log("GraphvizSectorMap: Source %d has %d targets", source, len(targets))
		sourceNode, sourceExists := gvNodes[source]
		if !sourceExists {
			debug.Log("GraphvizSectorMap: Source node %d not found in gvNodes", source)
			continue
		}

		for target := range targets {
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

			// Style the edge to be highly visible on black background
			edge.SetColor("cyan")  // Bright cyan should be more visible
			edge.SetPenWidth(12.0) // Much thicker
			edge.SetStyle("solid")

			// Check if it's a bidirectional connection and set arrow direction
			if reverseTargets, exists := adjacencyMap[target]; exists {
				if _, isBidirectional := reverseTargets[source]; isBidirectional {
					edge.SetDir("both") // Bidirectional arrows
					edge.SetArrowHead("normal")
					edge.SetArrowTail("normal")
					debug.Log("GraphvizSectorMap: Created bidirectional edge %d<->%d", source, target)
				} else {
					edge.SetDir("forward") // Unidirectional arrow
					edge.SetArrowHead("normal")
					debug.Log("GraphvizSectorMap: Created unidirectional edge %d->%d", source, target)
				}
			} else {
				edge.SetDir("forward") // Default to unidirectional
				edge.SetArrowHead("normal")
				debug.Log("GraphvizSectorMap: Created default unidirectional edge %d->%d", source, target)
			}

			edgeCount++
		}
	}

	debug.Log("GraphvizSectorMap: Created %d total edges", edgeCount)

	// Render to PNG
	var buf bytes.Buffer
	err = gv.Render(ctx, gvGraph, graphviz.PNG, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to render graph to PNG: %w", err)
	}

	// Validate PNG output has content
	if buf.Len() == 0 {
		return nil, fmt.Errorf("graphviz render produced no PNG output")
	}

	debug.Log("GraphvizSectorMap: Generated PNG image, size: %d bytes", buf.Len())

	return buf.Bytes(), nil
}

// Note: outputSixelImage and outputSixelToTerminal methods removed - now handled in renderSixelInPanel
