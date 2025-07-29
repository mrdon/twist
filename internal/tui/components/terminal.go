package components

import (
	"strings"
	"twist/internal/terminal"
	"twist/internal/theme"

	"github.com/rivo/tview"
)

// TerminalComponent manages the terminal display component
type TerminalComponent struct {
	view     *tview.TextView
	wrapper  *tview.Flex
	terminal *terminal.Terminal
}

// NewTerminalComponent creates a new terminal component
func NewTerminalComponent(term *terminal.Terminal) *TerminalComponent {

	// Use theme factory for proper styling
	terminalView := theme.NewTextView().
		SetRegions(true).
		SetWordWrap(false).
		SetScrollable(true)

	// Explicitly set theme colors for text without ANSI codes
	colors := theme.Current().TerminalColors()
	terminalView.SetTextColor(colors.Foreground).
		SetBackgroundColor(colors.Background)

	// Enable dynamic colors for ANSI support
	terminalView.SetDynamicColors(true)

	// Disable border to prevent width conflicts with 80-column terminal content
	terminalView.SetBorder(false)

	// Create wrapper with theme colors
	wrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(terminalView, 0, 1, false)

	// The terminal buffer now handles ANSI conversion internally,
	// so we use the tview ANSI writer directly
	tc := &TerminalComponent{
		view:     terminalView,
		wrapper:  wrapper,
		terminal: term,
	}

	// Set up direct streaming from terminal to view (no full rewrites)
	if term != nil {
		term.SetUpdateCallback(tc.streamUpdate)
	}

	return tc
}

// GetWrapper returns the wrapper component
func (tc *TerminalComponent) GetWrapper() *tview.Flex {
	return tc.wrapper
}

// GetView returns the text view component
func (tc *TerminalComponent) GetView() *tview.TextView {
	return tc.view
}

// streamUpdate handles incremental terminal updates (called from terminal buffer)
func (tc *TerminalComponent) streamUpdate() {
	defer func() {
		if r := recover(); r != nil {
			// Silently recover from panics
		}
	}()

	if tc.terminal == nil {
		return
	}

	// Get only the new data that was just written to the terminal
	newData := tc.terminal.GetNewData()
	if len(newData) == 0 {
		return
	}

	// For streaming updates, fall back to full refresh for now
	tc.UpdateContent()
}

// UpdateContent updates the terminal content (for full refresh scenarios)
func (tc *TerminalComponent) UpdateContent() {
	defer func() {
		if r := recover(); r != nil {
			// Silently recover from panics
		}
	}()

	if tc.terminal == nil {
		return
	}

	// Get new efficient data structures
	runes := tc.terminal.GetRunes()
	colorChanges := tc.terminal.GetColorChanges()

	// Clear view and render using new efficient method
	tc.view.Clear()
	tc.renderRunesWithColorTags(runes, colorChanges)
	tc.view.ScrollToEnd()
}

// renderRunesWithColorTags renders using the new efficient data structure
func (tc *TerminalComponent) renderRunesWithColorTags(runes [][]rune, colorChanges []terminal.ColorChange) {
	colorIndex := 0
	
	for y, row := range runes {
		var lineBuilder strings.Builder
		
		// Get terminal width to prevent rendering extra columns
		terminalWidth, _ := tc.terminal.GetSize()
		maxX := terminalWidth
		if maxX > len(row) {
			maxX = len(row)
		}
		
		for x := 0; x < maxX; x++ {
			// Insert color tag if position matches a color change
			if colorIndex < len(colorChanges) {
				change := colorChanges[colorIndex]
				if change.Y == y && change.X == x {
					lineBuilder.WriteString(change.TViewTag)
					colorIndex++
				}
			}
			
			char := row[x]
			// Skip null characters but include spaces if they're explicit
			if char != 0 {
				lineBuilder.WriteRune(char)
			}
		}
		
		// Add newline except for last row
		if y < len(runes)-1 {
			lineBuilder.WriteRune('\n')
		}
		
		// Write line to view
		tc.view.Write([]byte(lineBuilder.String()))
	}
}

