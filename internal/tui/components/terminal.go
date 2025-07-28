package components

import (
	"log"
	"os"
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
	logger   *log.Logger
}

// NewTerminalComponent creates a new terminal component
func NewTerminalComponent(term *terminal.Terminal) *TerminalComponent {
	// Set up debug logging to the same file as the app
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	logger := log.New(logFile, "[TERMINAL] ", log.LstdFlags|log.Lshortfile)

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

	terminalView.SetBorder(true).SetTitle("Terminal")

	// Create wrapper with theme colors
	wrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(terminalView, 0, 1, false)

	// The terminal buffer now handles ANSI conversion internally,
	// so we use the tview ANSI writer directly
	tc := &TerminalComponent{
		view:     terminalView,
		wrapper:  wrapper,
		terminal: term,
		logger:   logger,
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
			tc.logger.Printf("ERROR: Panic in streamUpdate(): %v", r)
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

	tc.logger.Printf("DEBUG: Streaming %d bytes of new data", len(newData))

	// Log sample only if it contains ANSI sequences for debugging
	if len(newData) > 0 && strings.Contains(string(newData), "\x1b[") {
		sample := string(newData)
		if len(sample) > 100 {
			sample = sample[:100] + "..."
		}
		tc.logger.Printf("DEBUG: Streaming ANSI data sample: %q", sample)
	}

	// For streaming updates, fall back to full refresh for now
	// TODO: Implement incremental cell-based updates
	tc.UpdateContent()
}

// UpdateContent updates the terminal content (for full refresh scenarios)
func (tc *TerminalComponent) UpdateContent() {
	defer func() {
		if r := recover(); r != nil {
			tc.logger.Printf("ERROR: Panic in UpdateContent(): %v", r)
		}
	}()

	tc.logger.Printf("DEBUG: UpdateContent() called - full refresh")
	if tc.terminal == nil {
		tc.logger.Printf("DEBUG: Terminal is nil, returning")
		return
	}

	// Get cells directly and render without double ANSI conversion
	cells := tc.terminal.GetCells()
	tc.logger.Printf("DEBUG: Got %d rows of cells from terminal", len(cells))

	// Clear view and render cells directly
	tc.logger.Printf("DEBUG: Full refresh - clearing view and rendering cells")
	tc.view.Clear()
	tc.renderCellsDirect(cells)
	tc.view.ScrollToEnd()
	tc.logger.Printf("DEBUG: UpdateContent() completed")
}

// renderCellsDirect renders terminal cells directly without ANSI conversion
func (tc *TerminalComponent) renderCellsDirect(cells [][]terminal.Cell) {
	for y, row := range cells {
		var lineBuilder strings.Builder

		// Get terminal width to prevent rendering extra columns
		terminalWidth := len(row)
		if tc.terminal != nil {
			terminalWidth, _ = tc.terminal.GetSize()
		}

		// Limit iteration to prevent extra column rendering that causes line wrapping
		maxX := terminalWidth
		if maxX > len(row) {
			maxX = len(row)
		}

		for x := 0; x < maxX; x++ {
			cell := row[x]
			// Skip null characters
			if cell.Char == 0 {
				continue
			}

			// Apply colors directly using tview color tags
			if cell.BackgroundHex != "#000000" || cell.ForegroundHex != "#c0c0c0" || cell.Bold {
				// Convert hex to tview color format
				fgColor := tc.hexToTViewColor(cell.ForegroundHex)
				bgColor := tc.hexToTViewColor(cell.BackgroundHex)

				// Build tview color tag: [foreground:background:attributes]
				var colorTag strings.Builder
				colorTag.WriteString("[")
				colorTag.WriteString(fgColor)
				colorTag.WriteString(":")
				colorTag.WriteString(bgColor)
				if cell.Bold {
					colorTag.WriteString(":b")
				}
				if cell.Underline {
					colorTag.WriteString(":u")
				}
				if cell.Reverse {
					colorTag.WriteString(":r")
				}
				colorTag.WriteString("]")

				// Debug block characters specifically
				if cell.Char == '▄' || cell.Char == '▀' || cell.Char == '█' {
					tc.logger.Printf("RENDER BLOCK: '%c' with tag: %s (FG=%s->%s, BG=%s->%s)",
						cell.Char, colorTag.String(), cell.ForegroundHex, fgColor, cell.BackgroundHex, bgColor)
				}

				lineBuilder.WriteString(colorTag.String())
			}

			lineBuilder.WriteRune(cell.Char)
		}

		// Add newline except for last row
		if y < len(cells)-1 {
			lineBuilder.WriteRune('\n')
		}

		// Write line to view
		tc.view.Write([]byte(lineBuilder.String()))
	}
}

// hexToTViewColor converts hex color to tview color format
func (tc *TerminalComponent) hexToTViewColor(hex string) string {
	if hex == "#000000" {
		return "black"
	}
	if hex == "#c0c0c0" {
		return "silver"
	}
	if hex == "#008000" {
		return "green"
	}
	if hex == "#000080" {
		return "navy"
	}
	// For other colors, use hex format (tview supports #rrggbb)
	return hex
}
