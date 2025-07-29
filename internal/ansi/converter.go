package ansi

import (
	"fmt"
	"strconv"
	"strings"
	"twist/internal/theme"

	"github.com/gdamore/tcell/v2"
)

// ColorConverter handles conversion of ANSI color parameters to hex colors
type ColorConverter struct {
	currentForeground tcell.Color
	currentBackground tcell.Color
	bold              bool
	underline         bool
	reverse           bool
}

// NewColorConverter creates a new color converter
func NewColorConverter() *ColorConverter {
	// Initialize with theme defaults
	currentTheme := theme.Current()
	colors := currentTheme.TerminalColors()

	return &ColorConverter{
		currentForeground: colors.Foreground,
		currentBackground: colors.Background,
	}
}

// ConvertColorParams converts ANSI color parameters to hex colors and attributes
func (c *ColorConverter) ConvertColorParams(params string) (fgHex, bgHex string, bold, underline, reverse bool) {
	// Get theme colors
	currentTheme := theme.Current()
	colors := currentTheme.TerminalColors()
	ansiPalette := currentTheme.ANSIColorPalette()

	// Handle reset/clear FIRST before setting any state
	if params == "" || params == "0" {
		// Update internal state completely
		c.currentForeground = colors.Foreground
		c.currentBackground = colors.Background
		c.bold = false
		c.underline = false
		c.reverse = false
		// Return theme defaults with all attributes reset
		fgR, fgG, fgB := colors.Foreground.RGB()
		bgR, bgG, bgB := colors.Background.RGB()
		fgHex := fmt.Sprintf("#%02x%02x%02x", fgR, fgG, fgB)
		bgHex := fmt.Sprintf("#%02x%02x%02x", bgR, bgG, bgB)
		return fgHex, bgHex, false, false, false
	}

	// Start with current state
	fg := c.currentForeground
	bg := c.currentBackground
	bold = c.bold
	underline = c.underline
	reverse = c.reverse

	// Process parameters sequentially
	parts := strings.Split(params, ";")
	for _, part := range parts {
		code, err := strconv.Atoi(part)
		if err != nil {
			continue // Skip non-numeric
		}

		switch {
		case code == 0: // Reset
			fg = colors.Foreground
			bg = colors.Background
			bold, underline, reverse = false, false, false
			
		case code == 1: // Bold
			bold = true
			
		case code == 4: // Underline
			underline = true
			
		case code == 7: // Reverse
			reverse = true
			
		case code >= 30 && code <= 37: // Standard foreground colors
			colorIndex := code - 30
			if bold {
				// Bold makes colors bright
				fg = ansiPalette[colorIndex+8]
			} else {
				fg = ansiPalette[colorIndex]
			}
			
		case code >= 40 && code <= 47: // Standard background colors
			colorIndex := code - 40
			bg = ansiPalette[colorIndex]
			
		case code >= 90 && code <= 97: // Bright foreground colors
			colorIndex := code - 90
			fg = ansiPalette[colorIndex+8]
			
		case code >= 100 && code <= 107: // Bright background colors
			colorIndex := code - 100
			bg = ansiPalette[colorIndex+8]
			
		case code == 39: // Default foreground
			fg = colors.Foreground
			
		case code == 49: // Default background
			bg = colors.Background
		}
	}
	
	// If bold was set, re-evaluate existing colors
	if bold {
		// Check if the current foreground color is a standard color that should become bright
		for i, standardColor := range ansiPalette[:8] {
			if fg == standardColor {
				// Convert standard color to bright color
				fg = ansiPalette[i+8]
				break
			}
		}
	}
	
	// Update internal state with new values
	c.currentForeground = fg
	c.currentBackground = bg
	c.bold = bold
	c.underline = underline
	c.reverse = reverse
	
	// Convert to hex
	fgR, fgG, fgB := fg.RGB()
	bgR, bgG, bgB := bg.RGB()
	fgHex = fmt.Sprintf("#%02x%02x%02x", fgR, fgG, fgB)
	bgHex = fmt.Sprintf("#%02x%02x%02x", bgR, bgG, bgB)
	
	return fgHex, bgHex, bold, underline, reverse
}

// ConvertANSIParams converts ANSI parameters to tview color tag
func (c *ColorConverter) ConvertANSIParams(params string) string {
	fgHex, bgHex, bold, underline, reverse := c.ConvertColorParams(params)
	return c.buildTViewColorTag(fgHex, bgHex, bold, underline, reverse)
}

// buildTViewColorTag converts color attributes to tview color tag format
func (c *ColorConverter) buildTViewColorTag(fgHex, bgHex string, bold, underline, reverse bool) string {
	// Build tview color tag: [foreground:background:attributes]
	var tag strings.Builder
	tag.WriteString("[")
	
	// Add foreground color (use hex format)
	tag.WriteString(fgHex)
	tag.WriteString(":")
	
	// Add background color (use hex format)
	tag.WriteString(bgHex)
	
	// Add attributes
	if bold {
		tag.WriteString(":b")
	}
	if underline {
		tag.WriteString(":u")
	}
	if reverse {
		tag.WriteString(":r")
	}
	
	tag.WriteString("]")
	return tag.String()
}