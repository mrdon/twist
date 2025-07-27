package components

import (
	"strings"
	"twist/internal/terminal"
	
	"github.com/rivo/tview"
)

// TerminalComponent manages the terminal display component
type TerminalComponent struct {
	view    *tview.TextView
	wrapper *tview.Flex
	terminal *terminal.Terminal
}

// NewTerminalComponent creates a new terminal component
func NewTerminalComponent(term *terminal.Terminal) *TerminalComponent {
	terminalView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)
	
	terminalView.SetBorder(true).SetTitle("Terminal")
	
	wrapper := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(terminalView, 0, 1, false)
	
	return &TerminalComponent{
		view:     terminalView,
		wrapper:  wrapper,
		terminal: term,
	}
}

// GetWrapper returns the wrapper component
func (tc *TerminalComponent) GetWrapper() *tview.Flex {
	return tc.wrapper
}

// GetView returns the text view component
func (tc *TerminalComponent) GetView() *tview.TextView {
	return tc.view
}

// UpdateContent updates the terminal content
func (tc *TerminalComponent) UpdateContent() {
	if tc.terminal == nil {
		return
	}
	
	cells := tc.terminal.GetCells()
	lines := tc.convertTerminalCellsToText(cells)
	content := strings.Join(lines, "\n")
	
	tc.view.SetText(content)
	tc.view.ScrollToEnd()
}

// convertTerminalCellsToText converts terminal cells to tview-compatible text
func (tc *TerminalComponent) convertTerminalCellsToText(cells [][]terminal.Cell) []string {
	var lines []string
	
	for _, row := range cells {
		var line strings.Builder
		currentColor := ""
		
		for _, cell := range row {
			// Convert terminal colors to tview color tags
			cellColor := tc.ansiToTviewColor(cell.Foreground)
			
			if cellColor != currentColor {
				if currentColor != "" {
					line.WriteString("[-:-:-]") // Reset color
				}
				if cellColor != "" {
					line.WriteString("[" + cellColor + ":-:-]")
				}
				currentColor = cellColor
			}
			
			line.WriteRune(cell.Char)
		}
		
		if currentColor != "" {
			line.WriteString("[-:-:-]") // Reset color at end of line
		}
		
		lines = append(lines, line.String())
	}
	
	return lines
}

// ansiToTviewColor converts ANSI color codes to tview color names
func (tc *TerminalComponent) ansiToTviewColor(colorCode int) string {
	switch colorCode {
	case 30: return "black"
	case 31: return "red"
	case 32: return "green"
	case 33: return "yellow"
	case 34: return "blue"
	case 35: return "purple"
	case 36: return "teal"
	case 37: return "white"
	case 90: return "gray"
	case 91: return "red"
	case 92: return "lime"
	case 93: return "yellow"
	case 94: return "blue"
	case 95: return "fuchsia"
	case 96: return "aqua"
	case 97: return "white"
	default: return ""
	}
}