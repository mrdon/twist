package components

import (
	"github.com/rivo/tview"
)

// MenuComponent manages the menu bar component
type MenuComponent struct {
	view *tview.TextView
}

// NewMenuComponent creates a new menu component
func NewMenuComponent() *MenuComponent {
	menuBar := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)
	
	// Set initial menu text
	menuBar.SetText("[yellow][F1]File [F2]Edit [F3]View [F4]Terminal [F5]Help [F10]Exit[-]")
	
	return &MenuComponent{
		view: menuBar,
	}
}

// GetView returns the menu view
func (mc *MenuComponent) GetView() *tview.TextView {
	return mc.view
}

// UpdateMenu updates the menu bar text
func (mc *MenuComponent) UpdateMenu(connected bool) {
	var menuText string
	
	if connected {
		menuText = "[yellow][F1]File [F2]Edit [F3]View [F4]Terminal [F5]Help [F9]Disconnect [F10]Exit[-]"
	} else {
		menuText = "[yellow][F1]File [F2]Edit [F3]View [F4]Terminal [F5]Help [F8]Connect [F10]Exit[-]"
	}
	
	mc.view.SetText(menuText)
}

// SetConnectedMenu sets the menu for connected state
func (mc *MenuComponent) SetConnectedMenu() {
	mc.UpdateMenu(true)
}

// SetDisconnectedMenu sets the menu for disconnected state
func (mc *MenuComponent) SetDisconnectedMenu() {
	mc.UpdateMenu(false)
}