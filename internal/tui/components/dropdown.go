package components

import (
	"twist/internal/theme"
	twistComponents "twist/internal/components"
	
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Import the MenuItem type
type MenuItem = twistComponents.MenuItem

// DropdownMenu represents a dropdown menu component
type DropdownMenu struct {
	list            *twistComponents.TwistMenu
	wrapper         *tview.Flex
	visible         bool
	callback        func(string)
	navigationCallback func(direction string) // "left" or "right"
}

// NewDropdownMenu creates a new dropdown menu
func NewDropdownMenu() *DropdownMenu {
	// Use theme factory for convenience
	list := theme.NewTwistMenu()

	dm := &DropdownMenu{
		list:    list,
		visible: false,
	}
	
	// Set up arrow key navigation
	list.SetInputCapture(dm.handleInput)

	return dm
}


// Show displays the dropdown menu with MenuItem structs that can include shortcuts
func (dm *DropdownMenu) Show(menuName string, items []MenuItem, leftOffset int, callback func(string), globalShortcuts *twistComponents.GlobalShortcutManager) *tview.Flex {
	dm.callback = callback
	dm.visible = true
	
	// Create callbacks for each item and register global shortcuts
	callbacks := make([]func(), len(items))
	for i, item := range items {
		label := item.Label // Capture for closure
		itemCallback := func() {
			if dm.callback != nil {
				dm.callback(label)
			}
			dm.Hide()
		}
		callbacks[i] = itemCallback
		
		// Register global shortcut if item has one
		if item.Shortcut != "" && globalShortcuts != nil {
			globalShortcuts.RegisterShortcut(item.Shortcut, itemCallback)
		}
	}
	
	// Set menu items with automatic shortcut registration and display formatting
	dm.list.SetMenuItems(items, callbacks)
	
	// Calculate dropdown width based on content
	maxLabelWidth := 0
	maxShortcutWidth := 0
	hasAnyShortcuts := false
	
	for _, item := range items {
		if len(item.Label) > maxLabelWidth {
			maxLabelWidth = len(item.Label)
		}
		if item.Shortcut != "" {
			hasAnyShortcuts = true
			if len(item.Shortcut) > maxShortcutWidth {  
				maxShortcutWidth = len(item.Shortcut)
			}
		}
	}
	
	var dropdownWidth int
	if hasAnyShortcuts {
		// Width = longest label + gap + longest shortcut + padding
		dropdownWidth = maxLabelWidth + 3 + maxShortcutWidth + 6
	} else {
		// Width = longest label + padding
		dropdownWidth = maxLabelWidth + 6
	}
	
	if dropdownWidth < 15 {
		dropdownWidth = 15 // Minimum width
	}
	
	// Create positioned wrapper - height accounts for all items without scrolling
	// Allocate generous height to ensure all items are visible without scrolling
	menuHeight := len(items) * 2 + 2  // Generous height calculation
	if menuHeight < 6 {
		menuHeight = 6  // Minimum height for border
	}
	
	dm.wrapper = tview.NewFlex().
		AddItem(nil, leftOffset, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 1, 0, false).  // Top spacing (1 row below menu bar)
			AddItem(dm.list, menuHeight, 0, true).  // Height for all items + border
			AddItem(nil, 0, 1, false), dropdownWidth, 0, true).
		AddItem(nil, 0, 1, false)
	
	return dm.wrapper
}

// Hide hides the dropdown menu
func (dm *DropdownMenu) Hide() {
	dm.visible = false
	dm.callback = nil
}

// IsVisible returns whether the dropdown is currently visible
func (dm *DropdownMenu) IsVisible() bool {
	return dm.visible
}

// GetList returns the underlying list component for focus management
func (dm *DropdownMenu) GetList() *twistComponents.TwistMenu {
	return dm.list
}

// SetNavigationCallback sets the callback for left/right arrow navigation
func (dm *DropdownMenu) SetNavigationCallback(callback func(direction string)) {
	dm.navigationCallback = callback
}

// handleInput handles input events for the dropdown
func (dm *DropdownMenu) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// First, check if the list can handle this as a shortcut
	if dm.list.HandleShortcut(event) {
		return nil // Shortcut was handled
	}
	
	// Then handle navigation keys
	switch event.Key() {
	case tcell.KeyLeft:
		if dm.navigationCallback != nil {
			dm.navigationCallback("left")
		}
		return nil
	case tcell.KeyRight:
		if dm.navigationCallback != nil {
			dm.navigationCallback("right")
		}
		return nil
	}
	return event
}