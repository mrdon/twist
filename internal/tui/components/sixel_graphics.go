package components

import (
	"fmt"
	"math"
	"strings"
)

// SixelCanvas represents a pixel canvas for sixel graphics
type SixelCanvas struct {
	width  int
	height int
	pixels [][]int // 2D array of color indices
}

// NewSixelCanvas creates a new sixel canvas
func NewSixelCanvas(width, height int) *SixelCanvas {
	pixels := make([][]int, height)
	for i := range pixels {
		pixels[i] = make([]int, width)
	}
	return &SixelCanvas{
		width:  width,
		height: height,
		pixels: pixels,
	}
}

// SetPixel sets a pixel to a specific color
func (c *SixelCanvas) SetPixel(x, y, color int) {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		c.pixels[y][x] = color
	}
}

// GetPixel gets the color of a pixel
func (c *SixelCanvas) GetPixel(x, y int) int {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		return c.pixels[y][x]
	}
	return 0
}

// DrawFilledCircle draws a filled circle using Bresenham's algorithm
func (c *SixelCanvas) DrawFilledCircle(centerX, centerY, radius, fillColor, borderColor int) {
	// Draw filled circle using scan line algorithm
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			distance := math.Sqrt(float64(x*x + y*y))
			if distance <= float64(radius) {
				pixelX := centerX + x
				pixelY := centerY + y
				
				if distance >= float64(radius-1) {
					// Border
					c.SetPixel(pixelX, pixelY, borderColor)
				} else {
					// Fill
					c.SetPixel(pixelX, pixelY, fillColor)
				}
			}
		}
	}
}

// DrawLine draws a line using Bresenham's line algorithm
func (c *SixelCanvas) DrawLine(x1, y1, x2, y2, color int) {
	dx := int(math.Abs(float64(x2 - x1)))
	dy := int(math.Abs(float64(y2 - y1)))
	
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	
	err := dx - dy
	x, y := x1, y1
	
	for {
		c.SetPixel(x, y, color)
		
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

// DrawArrow draws an arrow from (x1,y1) to (x2,y2)
func (c *SixelCanvas) DrawArrow(x1, y1, x2, y2, color int) {
	// Draw the main line
	c.DrawLine(x1, y1, x2, y2, color)
	
	// Calculate arrow head
	angle := math.Atan2(float64(y2-y1), float64(x2-x1))
	arrowLength := 8.0
	arrowAngle := math.Pi / 6 // 30 degrees
	
	// Calculate arrow head points
	x3 := x2 - int(arrowLength*math.Cos(angle-arrowAngle))
	y3 := y2 - int(arrowLength*math.Sin(angle-arrowAngle))
	x4 := x2 - int(arrowLength*math.Cos(angle+arrowAngle))
	y4 := y2 - int(arrowLength*math.Sin(angle+arrowAngle))
	
	// Draw arrow head
	c.DrawLine(x2, y2, x3, y3, color)
	c.DrawLine(x2, y2, x4, y4, color)
}

// DrawRectangle draws a filled rectangle
func (c *SixelCanvas) DrawRectangle(x, y, width, height, fillColor, borderColor int) {
	// Fill rectangle
	for py := y; py < y+height; py++ {
		for px := x; px < x+width; px++ {
			if px == x || px == x+width-1 || py == y || py == y+height-1 {
				c.SetPixel(px, py, borderColor)
			} else {
				c.SetPixel(px, py, fillColor)
			}
		}
	}
}

// RenderToSixel converts the canvas to sixel format
func (c *SixelCanvas) RenderToSixel() string {
	var sixel strings.Builder
	
	// Start sixel sequence with proper initialization
	sixel.WriteString("\x1bPq")
	
	// Define color palette with proper RGB values
	sixel.WriteString("#0;2;0;0;0")       // Black background
	sixel.WriteString("#1;2;100;100;0")   // Yellow - current sector
	sixel.WriteString("#2;2;0;80;0")      // Green - empty connected sectors  
	sixel.WriteString("#3;2;0;40;100")    // Blue - port sectors
	sixel.WriteString("#4;2;100;100;100") // White - connection lines
	sixel.WriteString("#5;2;80;0;0")      // Red - dangerous sectors
	sixel.WriteString("#6;2;40;40;40")    // Dark gray - sector borders
	sixel.WriteString("#7;2;60;60;60")    // Light gray - arrow heads
	
	// Convert pixels to sixel format properly
	// Sixels are encoded in 6-pixel high bands
	for band := 0; band < (c.height+5)/6; band++ {
		bandStart := band * 6
		bandEnd := bandStart + 6
		if bandEnd > c.height {
			bandEnd = c.height
		}
		
		// Process each color that has pixels in this band
		for color := 0; color <= 7; color++ {
			var colorData strings.Builder
			hasPixelsInBand := false
			
			for x := 0; x < c.width; x++ {
				sixelChar := 0
				
				// Check 6 pixels in this column for this color
				for y := bandStart; y < bandEnd && y < c.height; y++ {
					if c.GetPixel(x, y) == color {
						sixelChar |= 1 << (y - bandStart)
						hasPixelsInBand = true
					}
				}
				
				// Convert to sixel character (add 63 to make it printable)
				colorData.WriteString(string(rune(sixelChar + 63)))
			}
			
			// Only output data for colors that have pixels in this band
			if hasPixelsInBand {
				sixel.WriteString(fmt.Sprintf("#%d", color))
				sixel.WriteString(colorData.String())
				sixel.WriteString("$") // Carriage return to start of line
			}
		}
		
		if band < (c.height+5)/6-1 {
			sixel.WriteString("-") // Line feed (next band)
		}
	}
	
	// End sixel sequence
	sixel.WriteString("\x1b\\")
	
	return sixel.String()
}

// Clear clears the canvas to background color
func (c *SixelCanvas) Clear() {
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			c.pixels[y][x] = 0 // Background color
		}
	}
}

// DrawText draws simple text (very basic implementation)
func (c *SixelCanvas) DrawText(x, y int, text string, color int) {
	// Very simplified text rendering - just draw small rectangles for digits
	// In a full implementation, you'd have proper font rendering
	
	charWidth := 6
	charHeight := 8
	
	for i, char := range text {
		charX := x + i*charWidth
		
		// Draw a simple representation of each character
		switch char {
		case '0':
			c.DrawRectangle(charX, y, charWidth-1, charHeight, 0, color)
		case '1':
			c.DrawLine(charX+2, y, charX+2, y+charHeight, color)
		case '2':
			c.DrawLine(charX, y, charX+charWidth-1, y, color)
			c.DrawLine(charX, y+charHeight/2, charX+charWidth-1, y+charHeight/2, color)
			c.DrawLine(charX, y+charHeight-1, charX+charWidth-1, y+charHeight-1, color)
		case '3':
			c.DrawLine(charX, y, charX+charWidth-1, y, color)
			c.DrawLine(charX, y+charHeight/2, charX+charWidth-1, y+charHeight/2, color)
			c.DrawLine(charX, y+charHeight-1, charX+charWidth-1, y+charHeight-1, color)
		case '4':
			c.DrawLine(charX, y, charX, y+charHeight/2, color)
			c.DrawLine(charX, y+charHeight/2, charX+charWidth-1, y+charHeight/2, color)
			c.DrawLine(charX+charWidth-1, y, charX+charWidth-1, y+charHeight, color)
		case '5':
			c.DrawLine(charX, y, charX+charWidth-1, y, color)
			c.DrawLine(charX, y+charHeight/2, charX+charWidth-1, y+charHeight/2, color)
			c.DrawLine(charX, y+charHeight-1, charX+charWidth-1, y+charHeight-1, color)
		case '6':
			c.DrawRectangle(charX, y, charWidth-1, charHeight, 0, color)
			c.DrawLine(charX, y+charHeight/2, charX+charWidth-1, y+charHeight/2, color)
		case '7':
			c.DrawLine(charX, y, charX+charWidth-1, y, color)
			c.DrawLine(charX+charWidth-1, y, charX+charWidth-1, y+charHeight, color)
		case '8':
			c.DrawRectangle(charX, y, charWidth-1, charHeight, 0, color)
			c.DrawLine(charX, y+charHeight/2, charX+charWidth-1, y+charHeight/2, color)
		case '9':
			c.DrawRectangle(charX, y, charWidth-1, charHeight, 0, color)
			c.DrawLine(charX, y+charHeight/2, charX+charWidth-1, y+charHeight/2, color)
		default:
			// Draw a small rectangle for unknown characters
			c.DrawRectangle(charX+1, y+2, 2, 4, color, color)
		}
	}
}