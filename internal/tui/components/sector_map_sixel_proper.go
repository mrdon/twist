package components

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/theme"
	
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-sixel"
	"github.com/rivo/tview"
)

// ProperSixelSectorMapComponent uses proper sixel integration with tview
type ProperSixelSectorMapComponent struct {
	*tview.Box
	proxyAPI      api.ProxyAPI
	currentSector int
	sectorData    map[int]api.SectorInfo
	image         image.Image
	needsRedraw   bool
	lastSixelData string
	lastX, lastY  int
}

// NewProperSixelSectorMapComponent creates a new properly integrated sixel sector map
func NewProperSixelSectorMapComponent() *ProperSixelSectorMapComponent {
	// Get theme colors for panels
	currentTheme := theme.Current()
	panelColors := currentTheme.PanelColors()
	
	box := tview.NewBox()
	box.SetBackgroundColor(panelColors.Background)
	box.SetBorderColor(panelColors.Border)
	box.SetTitleColor(panelColors.Title)
	
	component := &ProperSixelSectorMapComponent{
		Box:           box,
		sectorData:    make(map[int]api.SectorInfo),
		needsRedraw:   true,
	}
	
	component.SetBorder(true).SetTitle("Sector Map (Sixel)")
	return component
}

// SetProxyAPI sets the API reference for accessing game data
func (psmc *ProperSixelSectorMapComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
	psmc.proxyAPI = proxyAPI
	psmc.needsRedraw = true
}

// UpdateCurrentSector updates the map with the current sector
func (psmc *ProperSixelSectorMapComponent) UpdateCurrentSector(sectorNumber int) {
	psmc.currentSector = sectorNumber
	psmc.needsRedraw = true
}

// UpdateCurrentSectorWithInfo updates the map with full sector information
func (psmc *ProperSixelSectorMapComponent) UpdateCurrentSectorWithInfo(sectorInfo api.SectorInfo) {
	psmc.currentSector = sectorInfo.Number
	psmc.sectorData[sectorInfo.Number] = sectorInfo
	psmc.needsRedraw = true
}

// LoadRealMapData loads real sector data from the API
func (psmc *ProperSixelSectorMapComponent) LoadRealMapData() {
	if psmc.proxyAPI == nil {
		debug.Log("ProperSixelSectorMapComponent: No proxyAPI available")
		return
	}
	
	playerInfo, err := psmc.proxyAPI.GetPlayerInfo()
	if err != nil {
		debug.Log("ProperSixelSectorMapComponent: GetPlayerInfo failed: %v", err)
		return
	}
	
	if playerInfo.CurrentSector <= 0 {
		debug.Log("ProperSixelSectorMapComponent: Invalid sector number: %d", playerInfo.CurrentSector)
		return
	}
	
	psmc.currentSector = playerInfo.CurrentSector
	debug.Log("ProperSixelSectorMapComponent: Loading map data for sector %d", psmc.currentSector)
	psmc.needsRedraw = true
}

// Draw renders the sixel sector map using proper tview integration
func (psmc *ProperSixelSectorMapComponent) Draw(screen tcell.Screen) {
	psmc.Box.DrawForSubclass(screen, psmc)
	
	x, y, width, height := psmc.GetInnerRect()
	debug.Log("ProperSixelSectorMapComponent: Draw called - coords=(%d,%d) size=%dx%d", x, y, width, height)
	
	if width <= 0 || height <= 0 {
		debug.Log("ProperSixelSectorMapComponent: Invalid dimensions, skipping draw")
		return
	}
	
	// Debug text removed - sixel graphics are working
	
	// Debug output removed to prevent screen interference
	// debug.Log("ProperSixelSectorMapComponent: Draw called - sector=%d, needsRedraw=%v, hasImage=%v", 
	//	psmc.currentSector, psmc.needsRedraw, psmc.image != nil)
	
	// Always use sector map now that we know colors work
	// if psmc.currentSector == 0 {
	// 	// Create a simple test image to verify sixel rendering works
	// 	if psmc.image == nil {
	// 		psmc.generateTestImage(200, 150) // Fixed reasonable size
	// 	}
	// 	if psmc.image != nil {
	// 		psmc.renderSixelImage(screen, x, y, width, height)
	// 		// Also show text to confirm the component is working
	// 		psmc.drawStatusText(screen, x, y+5, width, height-5, "SIXEL RENDERED")
	// 	} else {
	// 		psmc.drawStatusText(screen, x, y, width, height, "Waiting for sector data...")
	// 	}
	// 	return
	// }
	
	// Generate or update the sector map image if needed
	if psmc.needsRedraw || psmc.image == nil {
		psmc.generateSectorMapImage(200, 150) // Use fixed reasonable size like test image
		psmc.needsRedraw = false
	}
	
	// Render sixel graphics using the working tview-sixel approach
	if psmc.image != nil {
		// Generate sixel data
		var buf bytes.Buffer
		encoder := sixel.NewEncoder(&buf)
		encoder.Dither = false
		
		if err := encoder.Encode(psmc.image); err == nil {
			sixelData := buf.String()
			
			// Try multiple approaches to work around modern tview/tcell changes
			
			// Method 1: Original working approach
			screen.ShowCursor(x, y+2)
			fmt.Print(sixelData)
			screen.Sync()
			
			// Method 2: Try bypassing buffering with direct terminal access
			// This works around potential tcell v2 buffer isolation
			go func() {
				if tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
					defer tty.Close()
					// Position cursor and output sixel
					fmt.Fprintf(tty, "\x1b[%d;%dH%s", y+3, x+1, sixelData)
				}
			}()
			
			// Debug output removed to prevent screen interference
			// debug.Log("ProperSixelSectorMapComponent: Trying multiple output methods for %d bytes", len(sixelData))
		}
		
		// Show sector info as text too
		psmc.drawStatusText(screen, x, y+10, width, height-10, fmt.Sprintf("Sector %d", psmc.currentSector))
	} else {
		psmc.drawStatusText(screen, x, y, width, height, "Generating sector map...")
	}
}

// drawStatusText draws simple status text in the panel
func (psmc *ProperSixelSectorMapComponent) drawStatusText(screen tcell.Screen, x, y, width, height int, text string) {
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

// generateSectorMapImage creates an image representation of the sector map
func (psmc *ProperSixelSectorMapComponent) generateSectorMapImage(imgWidth, imgHeight int) {
	debug.Log("ProperSixelSectorMapComponent: generateSectorMapImage called - sector=%d, size=%dx%d", 
		psmc.currentSector, imgWidth, imgHeight)
		
	if psmc.currentSector == 0 {
		debug.Log("ProperSixelSectorMapComponent: No current sector, skipping image generation")
		return
	}
	
	debug.Log("ProperSixelSectorMapComponent: Generating sector map for sector %d", psmc.currentSector)
	
	// Create a new RGBA image
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	
	// Fill with black background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.ZP, draw.Src)
	
	// Get current sector info from real data
	currentInfo, hasCurrentInfo := psmc.sectorData[psmc.currentSector]
	if !hasCurrentInfo && psmc.proxyAPI != nil {
		var err error
		currentInfo, err = psmc.proxyAPI.GetSectorInfo(psmc.currentSector)
		if err != nil {
			debug.Log("ProperSixelSectorMapComponent: Error getting sector info: %v", err)
			return
		}
		psmc.sectorData[psmc.currentSector] = currentInfo
		debug.Log("ProperSixelSectorMapComponent: Fetched sector info for %d, warps=%v", 
			psmc.currentSector, currentInfo.Warps)
	}
	
	// Draw sector map using pixel operations
	psmc.drawSectorMapOnImage(img, currentInfo)
	
	psmc.image = img
	debug.Log("ProperSixelSectorMapComponent: Generated %dx%d sector map image with %d connected sectors", 
		imgWidth, imgHeight, len(currentInfo.Warps))
}

// drawSectorMapOnImage draws the sector map directly on the image
func (psmc *ProperSixelSectorMapComponent) drawSectorMapOnImage(img *image.RGBA, currentInfo api.SectorInfo) {
	bounds := img.Bounds()
	centerX := bounds.Dx() / 2
	centerY := bounds.Dy() / 2
	nodeRadius := 25
	
	// Dark blue background for contrast
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 50, 255}}, image.ZP, draw.Src)
	
	// Draw current sector (bright yellow circle with black border)
	psmc.drawFilledCircle(img, centerX, centerY, nodeRadius+2, color.RGBA{0, 0, 0, 255}) // Black border
	psmc.drawFilledCircle(img, centerX, centerY, nodeRadius, color.RGBA{255, 255, 0, 255}) // Yellow center
	
	// Get connected sectors from real data
	connectedSectors := currentInfo.Warps
	// Debug log removed to prevent screen interference
	// debug.Log("ProperSixelSectorMapComponent: Current sector %d has %d warps: %v", 
	//	currentInfo.Number, len(connectedSectors), connectedSectors)
	
	if len(connectedSectors) == 0 {
		// debug.Log("ProperSixelSectorMapComponent: No warps found, showing isolated sector")
		return
	}
	
	// Position connected sectors around the center
	positions := []struct{ x, y int }{
		{centerX, centerY - 60}, // North
		{centerX + 60, centerY}, // East  
		{centerX, centerY + 60}, // South
		{centerX - 60, centerY}, // West
		{centerX + 40, centerY - 40}, // NE
		{centerX + 40, centerY + 40}, // SE
		{centerX - 40, centerY + 40}, // SW
		{centerX - 40, centerY - 40}, // NW
	}
	
	// Draw connected sectors
	for i, sectorNum := range connectedSectors {
		if i >= len(positions) {
			break
		}
		
		pos := positions[i]
		
		// Choose color based on sector type (using real sector data)
		sectorColor := color.RGBA{0, 200, 0, 255} // Bright green default
		if info, hasInfo := psmc.sectorData[sectorNum]; hasInfo {
			if info.HasTraders > 0 {
				sectorColor = color.RGBA{0, 150, 255, 255} // Bright blue for ports
			} else if info.NavHaz > 0 {
				sectorColor = color.RGBA{255, 0, 0, 255} // Red for dangerous
			}
		}
		
		// Draw sector circle with black border
		psmc.drawFilledCircle(img, pos.x, pos.y, nodeRadius-1, color.RGBA{0, 0, 0, 255}) // Black border
		psmc.drawFilledCircle(img, pos.x, pos.y, nodeRadius-3, sectorColor) // Colored center
		
		// Draw connection line from center to this sector
		psmc.drawLine(img, centerX, centerY, pos.x, pos.y, color.RGBA{255, 255, 255, 255})
	}
	
	// debug.Log("ProperSixelSectorMapComponent: Drew sector map with %d connected sectors", len(connectedSectors))
}

// drawFilledCircle draws a filled circle on the image
func (psmc *ProperSixelSectorMapComponent) drawFilledCircle(img *image.RGBA, centerX, centerY, radius int, c color.RGBA) {
	bounds := img.Bounds()
	
	for y := centerY - radius; y <= centerY + radius; y++ {
		for x := centerX - radius; x <= centerX + radius; x++ {
			if x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
				dx := x - centerX
				dy := y - centerY
				if dx*dx + dy*dy <= radius*radius {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
}

// drawLine draws a line between two points
func (psmc *ProperSixelSectorMapComponent) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	// Simple line drawing using Bresenham's algorithm
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	
	err := dx - dy
	x, y := x1, y1
	
	bounds := img.Bounds()
	
	for {
		if x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
			img.SetRGBA(x, y, c)
		}
		
		if x == x2 && y == y2 {
			break
		}
		
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

// renderSixelImage renders the image as sixel graphics to the screen
func (psmc *ProperSixelSectorMapComponent) renderSixelImage(screen tcell.Screen, x, y, width, height int) {
	if psmc.image == nil {
		return
	}
	
	// Create a buffer to encode sixel data
	var buf bytes.Buffer
	
	// Create sixel encoder with explicit settings
	encoder := sixel.NewEncoder(&buf)
	encoder.Dither = false  // Disable dithering for cleaner colors
	
	// Encode image as sixel
	err := encoder.Encode(psmc.image)
	if err != nil {
		debug.Log("ProperSixelSectorMapComponent: Error encoding sixel: %v", err)
		return
	}
	
	// Output sixel data after tview finishes drawing
	sixelData := buf.String()
	
	// Try writing directly to /dev/tty to bypass tview's screen management
	if ttyFile, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		defer ttyFile.Close()
		
		// Position cursor and output sixel to raw terminal
		fmt.Fprintf(ttyFile, "\x1b[%d;%dH%s", y+2, x+1, sixelData)
		
		debug.Log("ProperSixelSectorMapComponent: Output sixel directly to /dev/tty")
	} else {
		// Fallback to stdout
		fmt.Printf("\x1b[%d;%dH%s", y+2, x+1, sixelData)
		debug.Log("ProperSixelSectorMapComponent: Output sixel to stdout (fallback)")
	}
	
	debug.Log("ProperSixelSectorMapComponent: Rendered sixel image (%d bytes) at panel position (%d,%d)", 
		buf.Len(), x, y)
}

// generateTestImage creates a simple test image to verify sixel rendering
func (psmc *ProperSixelSectorMapComponent) generateTestImage(imgWidth, imgHeight int) {
	debug.Log("ProperSixelSectorMapComponent: generateTestImage called - size=%dx%d", imgWidth, imgHeight)
	
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	
	// Create a high-contrast pattern - white background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.ZP, draw.Src)
	
	// Add a black circle in the center for maximum contrast
	centerX := imgWidth / 2
	centerY := imgHeight / 2
	radius := imgWidth / 4 // Large radius relative to image size
	
	psmc.drawFilledCircle(img, centerX, centerY, radius, color.RGBA{0, 0, 0, 255})
	
	// Add pure red corners 
	cornerSize := 20
	psmc.drawFilledCircle(img, cornerSize, cornerSize, cornerSize/2, color.RGBA{255, 0, 0, 255})
	psmc.drawFilledCircle(img, imgWidth-cornerSize, cornerSize, cornerSize/2, color.RGBA{0, 255, 0, 255})
	psmc.drawFilledCircle(img, cornerSize, imgHeight-cornerSize, cornerSize/2, color.RGBA{0, 0, 255, 255})
	psmc.drawFilledCircle(img, imgWidth-cornerSize, imgHeight-cornerSize, cornerSize/2, color.RGBA{255, 255, 0, 255})
	
	psmc.image = img
	debug.Log("ProperSixelSectorMapComponent: Generated bright test image %dx%d", imgWidth, imgHeight)
}

// RenderSixelGraphics outputs the sixel graphics after tview screen updates
func (psmc *ProperSixelSectorMapComponent) RenderSixelGraphics() {
	if psmc.lastSixelData == "" {
		return
	}
	
	// Output sixel graphics directly to terminal
	if ttyFile, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		defer ttyFile.Close()
		
		// Position cursor and output sixel to raw terminal
		fmt.Fprintf(ttyFile, "\x1b[%d;%dH%s", psmc.lastY+2, psmc.lastX+1, psmc.lastSixelData)
		
		debug.Log("ProperSixelSectorMapComponent: Rendered sixel graphics to /dev/tty")
	} else {
		// Fallback to stdout
		fmt.Printf("\x1b[%d;%dH%s", psmc.lastY+2, psmc.lastX+1, psmc.lastSixelData)
		debug.Log("ProperSixelSectorMapComponent: Rendered sixel graphics to stdout")
	}
}

// Helper function - use existing abs from sector_map.go