package components

import (
	"fmt"
	"os"
	"sync"
)

// SixelRegion represents a screen region that contains sixel graphics
type SixelRegion struct {
	X, Y          int    // Screen coordinates
	Width, Height int    // Current region dimensions
	MaxWidth, MaxHeight int // Maximum dimensions ever used (for clearing)
	SixelData     string // The sixel sequence
	Visible       bool   // Whether to render this region
}

// SixelLayer manages direct terminal sixel rendering outside of tview
type SixelLayer struct {
	regions map[string]*SixelRegion
	mutex   sync.RWMutex
	tty     *os.File
}

// NewSixelLayer creates a new sixel rendering layer
func NewSixelLayer() *SixelLayer {
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		tty = nil // Fallback to stdout
	}
	
	return &SixelLayer{
		regions: make(map[string]*SixelRegion),
		tty:     tty,
	}
}

// AddRegion adds or updates a sixel region
func (sl *SixelLayer) AddRegion(id string, region *SixelRegion) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	
	// If region already exists, clear it first using max dimensions
	if existingRegion, exists := sl.regions[id]; exists {
		sl.clearRegionArea(existingRegion)
		
		// Update max dimensions for the region
		if region.Width > existingRegion.MaxWidth {
			existingRegion.MaxWidth = region.Width
		}
		if region.Height > existingRegion.MaxHeight {
			existingRegion.MaxHeight = region.Height
		}
		
		// Update the existing region with new data
		existingRegion.X = region.X
		existingRegion.Y = region.Y
		existingRegion.Width = region.Width
		existingRegion.Height = region.Height
		existingRegion.SixelData = region.SixelData
		existingRegion.Visible = region.Visible
	} else {
		// New region - initialize max dimensions to current dimensions
		region.MaxWidth = region.Width
		region.MaxHeight = region.Height
		sl.regions[id] = region
	}
}

// RemoveRegion removes a sixel region
func (sl *SixelLayer) RemoveRegion(id string) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	delete(sl.regions, id)
}

// UpdateRegion updates an existing region's sixel data
func (sl *SixelLayer) UpdateRegion(id string, sixelData string) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	if region, exists := sl.regions[id]; exists {
		// Clear the region first using max dimensions, then update
		sl.clearRegionArea(region)
		region.SixelData = sixelData
	}
}

// SetRegionVisible sets the visibility of a region
func (sl *SixelLayer) SetRegionVisible(id string, visible bool) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	if region, exists := sl.regions[id]; exists {
		// If hiding the region, clear it from the terminal
		if !visible && region.Visible {
			sl.clearRegionArea(region)
		}
		region.Visible = visible
	}
}

// ClearRegion explicitly clears a region's display area
func (sl *SixelLayer) ClearRegion(id string) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	if region, exists := sl.regions[id]; exists {
		sl.clearRegionArea(region)
	}
}

// ResetRegionMaxDimensions resets the max dimensions tracking for a region
func (sl *SixelLayer) ResetRegionMaxDimensions(id string) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	if region, exists := sl.regions[id]; exists {
		region.MaxWidth = region.Width
		region.MaxHeight = region.Height
	}
}

// Render renders all visible sixel regions to the terminal
func (sl *SixelLayer) Render() {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()
	
	output := sl.tty
	if output == nil {
		output = os.Stdout
	}
	
	for _, region := range sl.regions {
		if region.Visible && region.SixelData != "" {
			// Position cursor and output sixel
			fmt.Fprintf(output, "\x1b[%d;%dH%s", region.Y+1, region.X+1, region.SixelData)
		}
	}
}

// Clear clears all regions
func (sl *SixelLayer) Clear() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	sl.regions = make(map[string]*SixelRegion)
}

// clearRegionArea clears a rectangular area on the terminal using max dimensions
func (sl *SixelLayer) clearRegionArea(region *SixelRegion) {
	output := sl.tty
	if output == nil {
		output = os.Stdout
	}
	
	// Use maximum dimensions to ensure we clear any remnants from larger previous graphics
	clearWidth := region.MaxWidth
	clearHeight := region.MaxHeight
	
	// Fallback to current dimensions if max dimensions aren't set
	if clearWidth == 0 {
		clearWidth = region.Width
	}
	if clearHeight == 0 {
		clearHeight = region.Height
	}
	
	// Use a more aggressive clearing approach for sixel graphics
	// Sixel graphics can leave artifacts, so we use a wider clear pattern
	clearWidth = clearWidth + 2  // Add some padding to be safe
	clearHeight = clearHeight + 2 // Add some padding to be safe
	
	// Position cursor at the start of the region (with padding)
	startX := region.X
	startY := region.Y
	if startX > 0 {
		startX = startX - 1  // Start one character earlier if possible
	}
	if startY > 0 {
		startY = startY - 1  // Start one row earlier if possible
	}
	
	// Clear the region by overwriting with spaces
	// This uses ANSI escape sequences to clear the area
	for row := 0; row < clearHeight; row++ {
		// Position cursor at the start of each row in the region
		fmt.Fprintf(output, "\x1b[%d;%dH", startY+row+1, startX+1)
		// Clear to end of line within the region width
		// Use spaces to clear the specific width instead of clearing entire line
		for col := 0; col < clearWidth; col++ {
			fmt.Fprint(output, " ")
		}
	}
}

// Close closes the TTY handle
func (sl *SixelLayer) Close() {
	if sl.tty != nil {
		sl.tty.Close()
	}
}