package components

import (
	"fmt"
	"os"
	"strings"
	"twist/internal/theme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SixelView is a custom tview primitive that can render sixel graphics
type SixelView struct {
	*tview.Box
	sixelData    string
	fallbackText string
	title        string
}

// NewSixelView creates a new SixelView
func NewSixelView() *SixelView {
	sv := &SixelView{
		Box:   tview.NewBox(),
		title: "Sector Map (Sixel)",
	}

	// Set proper panel background color to prevent status bar red bleeding through
	currentTheme := theme.Current()
	panelColors := currentTheme.PanelColors()
	sv.SetBackgroundColor(panelColors.Background)
	sv.SetBorderColor(panelColors.Border)
	sv.SetTitleColor(panelColors.Title)

	sv.SetBorder(true).SetTitle(sv.title)
	return sv
}

// SetSixelData sets the sixel graphics data and fallback text
func (sv *SixelView) SetSixelData(sixelData, fallbackText string) {
	sv.sixelData = sixelData
	sv.fallbackText = fallbackText
}

// Draw renders the sixel view
func (sv *SixelView) Draw(screen tcell.Screen) {
	// Draw the box first
	sv.Box.DrawForSubclass(screen, sv)

	// Get the inner area
	x, y, width, height := sv.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	// If we have sixel data, try to output it directly bypassing tview
	if sv.sixelData != "" {
		// Output sixel data directly to stdout at the panel location
		sv.outputSixelAtPosition(x, y)
	}

	// Draw fallback text if provided
	if sv.fallbackText != "" {
		lines := strings.Split(sv.fallbackText, "\n")
		for i, line := range lines {
			if i >= height {
				break
			}

			// Center the line horizontally
			lineWidth := len(line)
			startX := x + (width-lineWidth)/2
			if startX < x {
				startX = x
			}

			for j, char := range line {
				if startX+j >= x+width {
					break
				}

				style := tcell.StyleDefault.Foreground(tcell.ColorWhite)
				screen.SetContent(startX+j, y+i, char, nil, style)
			}
		}
	}
}

// outputSixelDirectly attempts to output sixel data directly to terminal
func (sv *SixelView) outputSixelDirectly(x, y int) {
	// This is experimental - try to position and output sixel
	// Note: This may not work perfectly with tview's screen management

	// Save current cursor position
	fmt.Print("\x1b[s")

	// Move to the desired position (convert tview coordinates to terminal coordinates)
	// This is approximate and may need adjustment
	fmt.Printf("\x1b[%d;%dH", y+1, x+1) // Terminal is 1-indexed

	// Output the sixel data
	fmt.Print(sv.sixelData)

	// Restore cursor position
	fmt.Print("\x1b[u")

	// Force flush
	os.Stdout.Sync()
}

// GetTitle returns the title
func (sv *SixelView) GetTitle() string {
	return sv.title
}

// SetTitle sets the title
func (sv *SixelView) SetTitle(title string) {
	sv.title = title
	sv.Box.SetTitle(title)
}

// outputSixelAtPosition outputs sixel data at the specified tview coordinates
func (sv *SixelView) outputSixelAtPosition(x, y int) {
	if sv.sixelData == "" {
		return
	}

	// Save cursor position
	fmt.Print("\x1b[s")

	// Convert tview coordinates to terminal coordinates
	// Add 1 because terminal coordinates are 1-indexed
	fmt.Printf("\x1b[%d;%dH", y+1, x+1)

	// Output the sixel data
	fmt.Print(sv.sixelData)

	// Restore cursor position
	fmt.Print("\x1b[u")

	// Force immediate output
	os.Stdout.Sync()

}
