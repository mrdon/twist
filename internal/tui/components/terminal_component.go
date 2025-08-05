package components

import (
	_ "twist/internal/debug"
	"twist/internal/theme"
	
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TerminalComponent wraps TerminalView with configuration and styling
type TerminalComponent struct {
	terminalView *TerminalView
	wrapper      *tview.Flex
	starfield    *StarfieldComponent
	showStarfield bool
	app          *tview.Application
}

// NewTerminalComponent creates a new terminal component with proper styling
func NewTerminalComponent(app *tview.Application) *TerminalComponent {
	// Create the core terminal view
	terminalView := NewTerminalView()
	
	// Create starfield component with app reference
	starfield := NewStarfieldComponent(150, app)
	
	// Create wrapper layout - show terminal directly (temporary fix)
	wrapper := theme.NewFlex().SetDirection(tview.FlexRow)
	wrapper.AddItem(terminalView, 0, 1, true)
	
	tc := &TerminalComponent{
		terminalView: terminalView,
		wrapper:      wrapper,
		starfield:    starfield,
		showStarfield: false,  // Disable starfield for now
		app:          app,
	}
	
	return tc
}


// GetView returns the main wrapper component
func (tc *TerminalComponent) GetView() tview.Primitive {
	return tc.wrapper
}

// GetWrapper returns the wrapper (for compatibility)
func (tc *TerminalComponent) GetWrapper() *tview.Flex {
	return tc.wrapper
}

// TransitionToTerminal switches from starfield to terminal view
func (tc *TerminalComponent) TransitionToTerminal() {
	if !tc.showStarfield {
		return
	}
	
	tc.showStarfield = false
	tc.starfield.Stop()
	
	// Clear wrapper and add terminal view
	tc.wrapper.Clear()
	tc.wrapper.AddItem(tc.terminalView, 0, 1, true)
	
	// Ensure terminal view gets focus after transition
	if tc.app != nil {
		tc.app.QueueUpdateDraw(func() {
			tc.app.SetFocus(tc.terminalView)
		})
	}
}

// Write implements io.Writer - delegates to terminal view and triggers transition
func (tc *TerminalComponent) Write(p []byte) (n int, err error) {
	// If starfield is showing, transition to terminal on first write
	if tc.showStarfield {
		tc.TransitionToTerminal()
	}
	
	// Always write to terminal view, even during transition
	n, err = tc.terminalView.Write(p)
	
	// Don't add extra QueueUpdateDraw here - the terminal view handles its own updates
	// via the changedFunc callback to avoid double-drawing
	
	return n, err
}

// SetChangedFunc sets the callback for content changes
func (tc *TerminalComponent) SetChangedFunc(handler func()) *TerminalComponent {
	tc.terminalView.SetChangedFunc(handler)
	return tc
}

// GetCursor returns current cursor position
func (tc *TerminalComponent) GetCursor() (int, int) {
	return tc.terminalView.GetCursor()
}

// GetLineCount returns the number of lines in the terminal
func (tc *TerminalComponent) GetLineCount() int {
	return tc.terminalView.GetLineCount()
}

// UpdateContent triggers a redraw
func (tc *TerminalComponent) UpdateContent() {
	tc.terminalView.UpdateContent()
}

// ScrollTo scrolls to specific position
func (tc *TerminalComponent) ScrollTo(row, column int) *TerminalComponent {
	tc.terminalView.ScrollTo(row, column)
	return tc
}

// GetScrollOffset returns current scroll position
func (tc *TerminalComponent) GetScrollOffset() (row, column int) {
	return tc.terminalView.GetScrollOffset()
}

// SetScrollable sets whether the view can be scrolled
func (tc *TerminalComponent) SetScrollable(scrollable bool) *TerminalComponent {
	tc.terminalView.SetScrollable(scrollable)
	return tc
}

// Clear clears all content
func (tc *TerminalComponent) Clear() *TerminalComponent {
	tc.terminalView.Clear()
	return tc
}

// GetInnerRect returns the terminal's inner drawing area
func (tc *TerminalComponent) GetInnerRect() (int, int, int, int) {
	// Delegate to the terminal view which handles padding properly
	return tc.terminalView.GetInnerRect()
}

// Configuration methods for styling

// SetBorderVisible controls whether border is visible
func (tc *TerminalComponent) SetBorderVisible(visible bool) *TerminalComponent {
	tc.terminalView.SetBorder(visible)
	return tc
}

// SetPadding sets the padding around the terminal content
func (tc *TerminalComponent) SetPadding(top, bottom, left, right int) *TerminalComponent {
	tc.terminalView.SetBorderPadding(top, bottom, left, right)
	return tc
}

// SetBorderColor sets the border color
func (tc *TerminalComponent) SetBorderColor(color tcell.Color) *TerminalComponent {
	tc.terminalView.SetBorderColor(color)
	return tc
}

// SetBackgroundColor sets the background color
func (tc *TerminalComponent) SetBackgroundColor(color tcell.Color) *TerminalComponent {
	tc.terminalView.SetBackgroundColor(color)
	return tc
}