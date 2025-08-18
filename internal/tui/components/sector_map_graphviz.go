package components

import (
	"bytes"
	"container/list"
	"context"
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"sort"
	"strings"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/theme"

	"github.com/BourgeoisBear/rasterm"
	"github.com/dominikbraun/graph"
	"github.com/gdamore/tcell/v2"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/rivo/tview"
	xdraw "golang.org/x/image/draw"
)

// CachedGraphData represents cached image and sixel data keyed by content hash
type CachedGraphData struct {
	ImageData []byte
	SixelData string
	Width     int
	Height    int
}

// lruCacheItem represents an item in the LRU cache
type lruCacheItem struct {
	key  string
	data *CachedGraphData
}

// LRUCache implements a simple LRU cache with maximum size
type LRUCache struct {
	maxSize int
	items   map[string]*list.Element
	order   *list.List
}

// NewLRUCache creates a new LRU cache with the specified maximum size
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		maxSize: maxSize,
		items:   make(map[string]*list.Element),
		order:   list.New(),
	}
}

// Get retrieves a value from the cache, marking it as recently used
func (c *LRUCache) Get(key string) (*CachedGraphData, bool) {
	if element, exists := c.items[key]; exists {
		// Move to front (most recently used)
		c.order.MoveToFront(element)
		return element.Value.(*lruCacheItem).data, true
	}
	return nil, false
}

// Put stores a value in the cache
func (c *LRUCache) Put(key string, data *CachedGraphData) {
	if element, exists := c.items[key]; exists {
		// Update existing item and move to front
		element.Value.(*lruCacheItem).data = data
		c.order.MoveToFront(element)
		return
	}

	// Add new item
	item := &lruCacheItem{key: key, data: data}
	element := c.order.PushFront(item)
	c.items[key] = element

	// Check if we need to evict the least recently used item
	if c.order.Len() > c.maxSize {
		// Remove least recently used (back of the list)
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			oldestItem := oldest.Value.(*lruCacheItem)
			delete(c.items, oldestItem.key)
		}
	}
}

// GraphvizSectorMap manages the sector map visualization using graphviz and sixels
type GraphvizSectorMap struct {
	*tview.Box
	proxyAPI      api.ProxyAPI
	currentSector int
	sectorData    map[int]api.SectorInfo
	sectorLevels  map[int]int // Track which level each sector is at (0=current, 1-5=hop levels)
	
	// Content-hash based LRU caching
	graphCache    *LRUCache // LRU cache keyed by MD5 hash of DOT content
	currentHashKey string   // Current hash key being displayed
	
	needsRedraw   bool
	hasBorder     bool
	sixelLayer    *SixelLayer
	regionID      string
	isGenerating  bool        // Track when image generation is in progress
}

// NewGraphvizSectorMap creates a new graphviz-based sector map component
func NewGraphvizSectorMap(sixelLayer *SixelLayer) *GraphvizSectorMap {
	// Get theme colors - use default colors for proper background
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	panelColors := currentTheme.PanelColors()
	
	r, g, b := defaultColors.Background.RGB()
	debug.Log("GraphvizSectorMap: Constructor - theme default background RGB(%d,%d,%d)", r, g, b)
	
	r2, g2, b2 := panelColors.Background.RGB()
	debug.Log("GraphvizSectorMap: Constructor - theme panel background RGB(%d,%d,%d)", r2, g2, b2)
	
	box := tview.NewBox()
	box.SetBackgroundColor(defaultColors.Background)  // Use theme's default background
	box.SetBorderColor(panelColors.Border)
	box.SetTitleColor(panelColors.Title)
	
	gsm := &GraphvizSectorMap{
		Box:          box,
		sectorData:   make(map[int]api.SectorInfo),
		sectorLevels: make(map[int]int),
		graphCache:   NewLRUCache(100), // Initialize LRU cache with max size 100
		needsRedraw:  true,
		hasBorder:    false,  // No border, just background
		sixelLayer:   sixelLayer,
		regionID:     "sector_map", // Unique ID for this component
	}
	gsm.SetBorder(false).SetTitle("")
	return gsm
}

// SetProxyAPI sets the API reference for accessing game data
func (gsm *GraphvizSectorMap) SetProxyAPI(proxyAPI api.ProxyAPI) {
	gsm.proxyAPI = proxyAPI
	gsm.needsRedraw = true
	// LRU cache will handle eviction automatically
}

// Draw renders the graphviz sector map using the proven sixel technique
func (gsm *GraphvizSectorMap) Draw(screen tcell.Screen) {
	debug.Log("GraphvizSectorMap.Draw: Starting draw")
	
	// Get the component area
	x, y, width, height := gsm.GetRect()
	debug.Log("GraphvizSectorMap.Draw: Component rect x=%d y=%d w=%d h=%d", x, y, width, height)

	if width <= 0 || height <= 0 {
		debug.Log("GraphvizSectorMap.Draw: Invalid dimensions, returning")
		return
	}

	// Generate map image and sixel if needed
	needsGeneration := gsm.needsRedraw
	debug.Log("GraphvizSectorMap.Draw: needsGeneration=%t (needsRedraw=%t), isGenerating=%t", 
		needsGeneration, gsm.needsRedraw, gsm.isGenerating)
	
	
	if needsGeneration {
		if gsm.currentSector > 0 && gsm.proxyAPI != nil {
			debug.Log("GraphvizSectorMap.Draw: Generating new sector map for sector %d", gsm.currentSector)
			gsm.isGenerating = true  // Mark that we're generating
			
			// Clear the region before generating new content to prevent artifacts
			if gsm.sixelLayer != nil {
				gsm.sixelLayer.ClearRegion(gsm.regionID)
				gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
			}
			
			// Generate new graphviz image
			g, err := gsm.buildSectorGraph()
			if err == nil {
				debug.Log("GraphvizSectorMap.Draw: Graph built successfully, generating image")
				_, err := gsm.generateGraphvizImage(g, width, height)
				if err == nil {
					debug.Log("GraphvizSectorMap.Draw: Image generated successfully")
					// Image data is now cached in LRU cache, cachedImage/cachedSixel set by generateGraphvizImage
					gsm.needsRedraw = false
					gsm.isGenerating = false  // Mark generation complete
				} else {
					debug.Log("GraphvizSectorMap.Draw: Error generating image: %v", err)
					gsm.isGenerating = false
				}
			} else {
				debug.Log("GraphvizSectorMap.Draw: Error building graph: %v", err)
				gsm.isGenerating = false
			}
		} else {
			debug.Log("GraphvizSectorMap.Draw: Cannot generate - currentSector=%d, proxyAPI!=nil=%t", 
				gsm.currentSector, gsm.proxyAPI != nil)
		}
	}

	// Register sixel region with the layer if we have cached image
	if gsm.currentHashKey != "" && gsm.sixelLayer != nil {
		debug.Log("GraphvizSectorMap.Draw: Registering sixel region")
		gsm.registerSixelRegion(x, y, width, height)
	} else {
		debug.Log("GraphvizSectorMap.Draw: Showing status text - currentHashKey=%s, sixelLayer!=nil=%t", 
			gsm.currentHashKey, gsm.sixelLayer != nil)
		
		// Show status text
		gsm.drawStatusText(screen, x, y, width, height, "Generating sector map...")
		// Hide sixel region when not ready
		if gsm.sixelLayer != nil {
			gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
		}
	}
	
	debug.Log("GraphvizSectorMap.Draw: Draw complete")
}

// registerSixelRegion registers this component's sixel region with the layer
func (gsm *GraphvizSectorMap) registerSixelRegion(x, y, width, height int) {
	// Get cached data from LRU cache
	cached, found := gsm.graphCache.Get(gsm.currentHashKey)
	if !found {
		debug.Log("GraphvizSectorMap.registerSixelRegion: No cached data found for hash %s", gsm.currentHashKey)
		return
	}

	// Generate sixel data if not already generated for this cached item
	if cached.SixelData == "" {
		// Decode the cached PNG image
		img, err := png.Decode(bytes.NewReader(cached.ImageData))
		if err != nil {
			debug.Log("GraphvizSectorMap.registerSixelRegion: Failed to decode PNG: %v", err)
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
			debug.Log("GraphvizSectorMap.registerSixelRegion: Failed to encode sixel: %v", err)
			return
		}

		// Update the cached data with sixel
		cached.SixelData = buf.String()
		gsm.graphCache.Put(gsm.currentHashKey, cached) // Update cache with sixel data
	}

	// Register with the sixel layer
	region := &SixelRegion{
		X:         x,
		Y:         y + 1, // Minimal offset to avoid title overlap
		Width:     width,
		Height:    height - 1, // Minimal height adjustment
		SixelData: cached.SixelData,
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
	// Get theme colors
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	// Let tview handle the background - just draw the text
	// Draw text using theme's waiting color on theme's background
	style := tcell.StyleDefault.Foreground(defaultColors.Waiting).Background(defaultColors.Background)

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
		gsm.sectorLevels = make(map[int]int) // Clear sector levels for fresh tracking
		// Note: Don't clear sectorData or graphCache - let hash-based caching handle it

		// Hide the region while regenerating to prevent overlap
		if gsm.sixelLayer != nil {
			gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
		}
	}
}

// UpdateCurrentSectorWithInfo updates the map with full sector information
func (gsm *GraphvizSectorMap) UpdateCurrentSectorWithInfo(sectorInfo api.SectorInfo) {
	oldSector := gsm.currentSector
	currentHashKey := gsm.currentHashKey
	
	// Always update the sector data first
	gsm.sectorData[sectorInfo.Number] = sectorInfo
	
	if gsm.currentSector != sectorInfo.Number {
		// Current sector changed - force redraw
		gsm.currentSector = sectorInfo.Number
		gsm.needsRedraw = true
		gsm.currentHashKey = "" // Clear current hash key
		gsm.sectorLevels = make(map[int]int) // Clear sector levels for fresh tracking

		debug.Log("GraphvizSectorMap: UpdateCurrentSectorWithInfo - Current sector changed from %d to %d, triggering redraw", 
			oldSector, sectorInfo.Number)

		// Hide the region while regenerating to prevent overlap
		if gsm.sixelLayer != nil {
			gsm.sixelLayer.SetRegionVisible(gsm.regionID, false)
		}
	} else {
		// Same sector but data might have changed - check if graph content would change
		if newHash, err := gsm.generateDOTContentHash(); err == nil {
			debug.Log("GraphvizSectorMap: UpdateCurrentSectorWithInfo - Hash comparison for sector %d: current='%s', new='%s'", 
				sectorInfo.Number, currentHashKey, newHash)
			if newHash != currentHashKey {
				debug.Log("GraphvizSectorMap: UpdateCurrentSectorWithInfo - DOT content changed for sector %d, triggering redraw (old hash: %s, new hash: %s)", 
					sectorInfo.Number, currentHashKey, newHash)
				gsm.needsRedraw = true
				gsm.currentHashKey = "" // Clear current hash key to force regeneration
			} else {
				debug.Log("GraphvizSectorMap: UpdateCurrentSectorWithInfo - DOT content unchanged for sector %d, skipping redraw (hash: %s)", 
					sectorInfo.Number, currentHashKey)
			}
		} else {
			// If we can't generate hash, fall back to always redrawing
			debug.Log("GraphvizSectorMap: UpdateCurrentSectorWithInfo - Failed to generate DOT hash for sector %d, falling back to redraw: %v", 
				sectorInfo.Number, err)
			gsm.needsRedraw = true
			gsm.currentHashKey = "" // Clear current hash key to force regeneration
		}
	}
}

// UpdateSectorData updates sector data without changing the current sector focus
func (gsm *GraphvizSectorMap) UpdateSectorData(sectorInfo api.SectorInfo) {
	// Update the sector data in our cache
	gsm.sectorData[sectorInfo.Number] = sectorInfo
	
	// If this sector is part of the currently displayed map, check if we need a redraw
	// but don't change the current sector focus
	if gsm.currentSector > 0 {
		// Only check for redraw if the updated sector is within our display range
		// (current sector or connected sectors)
		if sectorInfo.Number == gsm.currentSector || gsm.isSectorInDisplayRange(sectorInfo.Number) {
			// If we don't have a current hash, we need to redraw regardless
			if gsm.currentHashKey == "" {
				debug.Log("GraphvizSectorMap: UpdateSectorData - No current hash for sector %d, triggering redraw", 
					sectorInfo.Number)
				gsm.needsRedraw = true
			} else {
				// Check if the graph would actually change by comparing DOT content hash
				if newHash, err := gsm.generateDOTContentHash(); err == nil {
					debug.Log("GraphvizSectorMap: UpdateSectorData - Hash comparison for sector %d: current='%s', new='%s'", 
						sectorInfo.Number, gsm.currentHashKey, newHash)
					if newHash != gsm.currentHashKey {
						debug.Log("GraphvizSectorMap: UpdateSectorData - DOT content changed for sector %d, triggering redraw (old hash: %s, new hash: %s)", 
							sectorInfo.Number, gsm.currentHashKey, newHash)
						gsm.needsRedraw = true
						gsm.currentHashKey = "" // Clear current hash key to force regeneration
					} else {
						debug.Log("GraphvizSectorMap: UpdateSectorData - DOT content unchanged for sector %d, skipping redraw (hash: %s)", 
							sectorInfo.Number, gsm.currentHashKey)
					}
				} else {
					// If we can't generate hash, fall back to always redrawing
					debug.Log("GraphvizSectorMap: UpdateSectorData - Failed to generate DOT hash for sector %d, falling back to redraw: %v", 
						sectorInfo.Number, err)
					gsm.needsRedraw = true
					gsm.currentHashKey = "" // Clear current hash key to force regeneration
				}
			}
		}
	}
}

// isSectorInDisplayRange checks if a sector is within the current display range
func (gsm *GraphvizSectorMap) isSectorInDisplayRange(sectorNumber int) bool {
	// Check if sector is in our current sector data (which means it's being displayed)
	_, exists := gsm.sectorData[sectorNumber]
	return exists
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
		gsm.currentHashKey = "" // Clear current hash key
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

	// Get theme colors for consistent styling
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	// Use neato engine with increased spacing for better layout
	gvGraph.SetLayout("neato")          // Force-directed layout engine
	gvGraph.SetBackgroundColor(gsm.colorToString(defaultColors.Background)) // Use theme's default background
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

	// Create graphviz nodes for each vertex - sort for deterministic ordering
	var sectors []int
	for sector := range adjacencyMap {
		sectors = append(sectors, sector)
	}
	sort.Ints(sectors)
	
	for _, sector := range sectors {
		// Create node with sector information
		sectorInfo, exists := gsm.sectorData[sector]

		var label, fillColor string
		if sector == gsm.currentSector {
			label = fmt.Sprintf("YOU\\n%d", sector)
			fillColor = "yellow"
		} else if exists && sectorInfo.Visited {
			// Truly visited sector - player has been here (EtHolo)
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
			} else if sectorInfo.HasPort {
				// Sector has port but no traders
				var portType string
				if gsm.proxyAPI != nil {
					if portData, err := gsm.proxyAPI.GetPortInfo(sector); err == nil && portData != nil {
						portType = portData.ClassType.String() // Show actual port type like "BSB"
					} else {
						portType = "PORT" // Port exists but couldn't get details
					}
				} else {
					portType = "PORT" // No API access
				}
				label = fmt.Sprintf("%d\\n(%s)", sector, portType)
				fillColor = "lightgreen"
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

	for _, source := range sectors {
		targets := adjacencyMap[source]
		sourceNode, sourceExists := gvNodes[source]
		if !sourceExists {
			continue
		}

		// Sort targets for deterministic edge ordering
		var targetList []int
		for target := range targets {
			targetList = append(targetList, target)
		}
		sort.Ints(targetList)

		for _, target := range targetList {
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

	// Generate DOT content and create MD5 hash for caching
	var dotBuf bytes.Buffer
	err = gv.Render(ctx, gvGraph, "dot", &dotBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DOT content: %w", err)
	}

	// Create MD5 hash of DOT content for cache key
	dotContent := dotBuf.Bytes()
	hash := md5.Sum(dotContent)
	hashKey := fmt.Sprintf("%x", hash)

	// Detailed logging for hash comparison debugging
	debug.Log("GraphvizSectorMap: generateGraphvizImage - Generated hash %s for sector %d", hashKey, gsm.currentSector)

	// Check if we have cached data for this hash
	if cached, found := gsm.graphCache.Get(hashKey); found {
		debug.Log("GraphvizSectorMap: Using cached image for hash %s (cache hit)", hashKey)
		gsm.currentHashKey = hashKey
		return cached.ImageData, nil
	}

	debug.Log("GraphvizSectorMap: Generating new image for hash %s (cache miss)", hashKey)
	gsm.currentHashKey = hashKey

	// Save DOT file for debugging
	if err := os.WriteFile("/tmp/sector_map.dot", dotContent, 0644); err != nil {
	} else {
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
	
	// Ensure panel dimensions are strictly bounded by component dimensions
	// This prevents any possibility of the image exceeding terminal bounds
	maxAllowedWidth := componentWidth * 8  // Conservative character width estimate
	maxAllowedHeight := adjustedHeight * 16 // Conservative character height estimate
	
	if panelPixelWidth > maxAllowedWidth {
		panelPixelWidth = maxAllowedWidth
	}
	if panelPixelHeight > maxAllowedHeight {
		panelPixelHeight = maxAllowedHeight
	}

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
	
	// Get theme colors and fill with theme's default background
	currentTheme = theme.Current()
	defaultColors = currentTheme.DefaultColors()
	r32, g32, b32 := defaultColors.Background.RGB()
	bgColor := color.RGBA{uint8(r32), uint8(g32), uint8(b32), 255}
	for y := 0; y < panelPixelHeight; y++ {
		for x := 0; x < panelPixelWidth; x++ {
			panelImg.Set(x, y, bgColor)
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

	finalImageData := panelBuf.Bytes()

	// Cache the result with the hash key
	cachedData := &CachedGraphData{
		ImageData: finalImageData,
		SixelData: "", // Sixel will be generated later when needed
		Width:     panelPixelWidth,
		Height:    panelPixelHeight,
	}
	gsm.graphCache.Put(hashKey, cachedData)

	return finalImageData, nil
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

// generateDOTContentHash creates a DOT content hash without generating the full image
func (gsm *GraphvizSectorMap) generateDOTContentHash() (string, error) {
	if gsm.currentSector <= 0 || gsm.proxyAPI == nil {
		return "", fmt.Errorf("no current sector or proxy API")
	}

	// Build the graph structure (same logic as buildSectorGraph)
	g, err := gsm.buildSectorGraph()
	if err != nil {
		return "", err
	}

	// Create a lightweight graphviz context just for DOT generation
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return "", err
	}
	defer gv.Close()

	// Create a new graphviz graph with same settings as generateGraphvizImage
	gvGraph, err := gv.Graph()
	if err != nil {
		return "", err
	}
	defer gvGraph.Close()

	// Apply same graph settings for consistent hashing
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	gvGraph.SetLayout("neato")
	gvGraph.SetBackgroundColor(gsm.colorToString(defaultColors.Background))
	gvGraph.SetDPI(150.0)
	
	gvGraph.Attr(int(cgraph.EDGE), "color", "white")
	gvGraph.Attr(int(cgraph.NODE), "style", "filled,rounded")
	gvGraph.Attr(int(cgraph.NODE), "penwidth", "3")
	gvGraph.Attr(int(cgraph.NODE), "color", "white")
	
	gvGraph.SetOverlap(false)
	gvGraph.SetSplines("true")
	gvGraph.Set("center", "true")
	gvGraph.Set("len", "3.0")
	gvGraph.Set("sep", "1.0")
	gvGraph.Set("defaultdist", "4.0")
	gvGraph.Set("overlap_scaling", "2.0")

	// Create nodes and edges (same logic as generateGraphvizImage)
	gvNodes := make(map[int]*graphviz.Node)
	adjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		return "", err
	}

	// Create graphviz nodes - sort sectors for deterministic ordering
	var sectors []int
	for sector := range adjacencyMap {
		sectors = append(sectors, sector)
	}
	sort.Ints(sectors)
	
	for _, sector := range sectors {
		sectorInfo, exists := gsm.sectorData[sector]

		var label, fillColor string
		if sector == gsm.currentSector {
			label = fmt.Sprintf("YOU\\\\n%d", sector)
			fillColor = "yellow"
		} else if exists && sectorInfo.Visited {
			if sectorInfo.HasTraders > 0 {
				var portType string
				if sectorInfo.HasPort {
					if gsm.proxyAPI != nil {
						if portData, err := gsm.proxyAPI.GetPortInfo(sector); err == nil && portData != nil {
							portType = portData.ClassType.String()
						} else {
							portType = "PORT"
						}
					} else {
						portType = "PORT"
					}
				} else {
					portType = fmt.Sprintf("T%d", sectorInfo.HasTraders)
				}
				label = fmt.Sprintf("%d\\\\n(%s)", sector, portType)
				fillColor = "lightblue"
			} else if sectorInfo.HasPort {
				var portType string
				if gsm.proxyAPI != nil {
					if portData, err := gsm.proxyAPI.GetPortInfo(sector); err == nil && portData != nil {
						portType = portData.ClassType.String()
					} else {
						portType = "PORT"
					}
				} else {
					portType = "PORT"
				}
				label = fmt.Sprintf("%d\\\\n(%s)", sector, portType)
				fillColor = "lightgreen"
			} else {
				label = fmt.Sprintf("%d", sector)
				fillColor = "gray"
			}
		} else {
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
		node.SetFontSize(18.0)
		node.SetFontColor("black")

		if level, exists := gsm.sectorLevels[sector]; exists && level == 5 {
			node.SetStyle("filled,rounded,dotted")
		} else {
			node.SetStyle("filled,rounded")
		}

		gvNodes[sector] = node
	}

	// Add edges - sort for deterministic ordering
	processedEdges := make(map[string]bool)
	for _, source := range sectors {
		targets := adjacencyMap[source]
		sourceNode, sourceExists := gvNodes[source]
		if !sourceExists {
			continue
		}

		// Sort targets for deterministic edge ordering
		var targetList []int
		for target := range targets {
			targetList = append(targetList, target)
		}
		sort.Ints(targetList)

		for _, target := range targetList {
			var edgeKey string
			if source < target {
				edgeKey = fmt.Sprintf("%d-%d", source, target)
			} else {
				edgeKey = fmt.Sprintf("%d-%d", target, source)
			}

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

			edge.SetPenWidth(1.5)
			edge.SetStyle("solid")
			edge.SetConstraint(true)
			edge.SetArrowSize(0.8)

			if reverseTargets, exists := adjacencyMap[target]; exists {
				if _, isBidirectional := reverseTargets[source]; isBidirectional {
					edge.SetDir("both")
					edge.SetArrowHead("normal")
					edge.SetArrowTail("normal")
				} else {
					edge.SetDir("forward")
					edge.SetArrowHead("normal")
				}
			} else {
				edge.SetDir("forward")
				edge.SetArrowHead("normal")
			}
		}
	}

	// Generate DOT content
	var dotBuf bytes.Buffer
	err = gv.Render(ctx, gvGraph, "dot", &dotBuf)
	if err != nil {
		return "", err
	}

	// Create MD5 hash and save DOT content for debugging
	dotContent := dotBuf.Bytes()
	hash := md5.Sum(dotContent)
	hashStr := fmt.Sprintf("%x", hash)
	
	
	return hashStr, nil
}

// colorToString converts tcell.Color to hex string for graphviz
func (gsm *GraphvizSectorMap) colorToString(color tcell.Color) string {
	r, g, b := color.RGB()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// Note: outputSixelImage and outputSixelToTerminal methods removed - now handled in renderSixelInPanel
