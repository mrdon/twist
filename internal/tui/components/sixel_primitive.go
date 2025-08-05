package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SixelPrimitive is a custom tview primitive that doesn't clear the background
type SixelPrimitive struct {
	x, y, width, height int
	visible             bool
	hasFrameBuffer      bool
	borderColor         tcell.Color
	backgroundColor     tcell.Color
	title               string
	drawFunc            func(screen tcell.Screen, x, y, width, height int)
}

// NewSixelPrimitive creates a new sixel primitive
func NewSixelPrimitive() *SixelPrimitive {
	return &SixelPrimitive{
		visible:         true,
		borderColor:     tcell.ColorWhite,
		backgroundColor: tcell.ColorDefault,
	}
}

// Draw implements tview.Primitive
func (sp *SixelPrimitive) Draw(screen tcell.Screen) {
	if !sp.visible {
		return
	}

	// Draw border if needed (without clearing background)
	if sp.title != "" {
		sp.drawBorder(screen)
	}

	// Calculate inner rect
	innerX, innerY, innerWidth, innerHeight := sp.getInnerRect()

	// Call custom draw function
	if sp.drawFunc != nil {
		sp.drawFunc(screen, innerX, innerY, innerWidth, innerHeight)
	}
}

// drawBorder draws just the border without clearing
func (sp *SixelPrimitive) drawBorder(screen tcell.Screen) {
	style := tcell.StyleDefault.Foreground(sp.borderColor)
	
	// Top border
	for x := sp.x; x < sp.x+sp.width; x++ {
		screen.SetContent(x, sp.y, '─', nil, style)
	}
	
	// Bottom border
	for x := sp.x; x < sp.x+sp.width; x++ {
		screen.SetContent(x, sp.y+sp.height-1, '─', nil, style)
	}
	
	// Left border
	for y := sp.y; y < sp.y+sp.height; y++ {
		screen.SetContent(sp.x, y, '│', nil, style)
	}
	
	// Right border
	for y := sp.y; y < sp.y+sp.height; y++ {
		screen.SetContent(sp.x+sp.width-1, y, '│', nil, style)
	}
	
	// Corners
	screen.SetContent(sp.x, sp.y, '┌', nil, style)
	screen.SetContent(sp.x+sp.width-1, sp.y, '┐', nil, style)
	screen.SetContent(sp.x, sp.y+sp.height-1, '└', nil, style)
	screen.SetContent(sp.x+sp.width-1, sp.y+sp.height-1, '┘', nil, style)
	
	// Title
	if sp.title != "" && len(sp.title) < sp.width-4 {
		titleX := sp.x + 2
		for i, r := range sp.title {
			screen.SetContent(titleX+i, sp.y, r, nil, style)
		}
	}
}

// getInnerRect returns the inner drawing area
func (sp *SixelPrimitive) getInnerRect() (int, int, int, int) {
	if sp.title != "" {
		return sp.x + 1, sp.y + 1, sp.width - 2, sp.height - 2
	}
	return sp.x, sp.y, sp.width, sp.height
}

// GetRect implements tview.Primitive
func (sp *SixelPrimitive) GetRect() (int, int, int, int) {
	return sp.x, sp.y, sp.width, sp.height
}

// SetRect implements tview.Primitive
func (sp *SixelPrimitive) SetRect(x, y, width, height int) {
	sp.x, sp.y, sp.width, sp.height = x, y, width, height
}

// InputHandler implements tview.Primitive
func (sp *SixelPrimitive) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// Handle input if needed
	}
}

// Focus implements tview.Primitive
func (sp *SixelPrimitive) Focus(delegate func(p tview.Primitive)) {
	// Handle focus if needed
}

// HasFocus implements tview.Primitive
func (sp *SixelPrimitive) HasFocus() bool {
	return false
}

// Blur implements tview.Primitive
func (sp *SixelPrimitive) Blur() {
	// Handle blur if needed
}

// SetDrawFunc sets the custom drawing function
func (sp *SixelPrimitive) SetDrawFunc(drawFunc func(screen tcell.Screen, x, y, width, height int)) *SixelPrimitive {
	sp.drawFunc = drawFunc
	return sp
}

// SetTitle sets the border title
func (sp *SixelPrimitive) SetTitle(title string) *SixelPrimitive {
	sp.title = title
	return sp
}

// SetBorderColor sets the border color
func (sp *SixelPrimitive) SetBorderColor(color tcell.Color) *SixelPrimitive {
	sp.borderColor = color
	return sp
}