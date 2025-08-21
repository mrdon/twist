package components

import (
	twistComponents "twist/internal/components"
	"twist/internal/theme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Import the MenuItem type
type MenuItem = twistComponents.MenuItem

// DropdownMenu represents a dropdown menu component
type DropdownMenu struct {
	list               *twistComponents.TwistMenu
	wrapper            *tview.Flex
	visible            bool
	callback           func(string)
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

	// Set up basic input handling
	list.SetInputCapture(dm.handleInput)

	return dm
}

// Show displays the dropdown menu with MenuItem structs that can include shortcuts
func (dm *DropdownMenu) Show(menuName string, items []MenuItem, leftOffset int, callback func(string), globalShortcuts *twistComponents.GlobalShortcutManager) *tview.Flex {
	dm.callback = callback
	dm.visible = true

	// Recreate the list component to ensure fresh sizing for each menu
	dm.list = theme.NewTwistMenu()
	dm.list.SetInputCapture(dm.handleInput)

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

	// Calculate proper height for dropdown menu
	// tview.List rendering: each item takes 1 row, no separators between items
	// Required space: top border (1) + items (N) + bottom border (1) + internal padding (1-2)
	itemCount := len(items)
	calculatedHeight := calculateMenuHeight(itemCount)

	dm.wrapper = tview.NewFlex().
		AddItem(nil, leftOffset, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 1, 0, false).                   // Top spacing (1 row below menu bar)
			AddItem(dm.list, calculatedHeight, 0, true). // Calculated height, weight 0
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

	// Handle menu navigation (left/right between different menus)
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
	case tcell.KeyEnter:
		// Handle Enter key to trigger menu item and consume it
		currentItem := dm.list.GetCurrentItem()
		if currentItem >= 0 && currentItem < dm.list.GetItemCount() {
			// Get the callback from TwistMenu and call it
			callbacks := dm.list.GetCallbacks()
			if currentItem < len(callbacks) && callbacks[currentItem] != nil {
				callbacks[currentItem]()
			}
		}
		return nil // Consume Enter to prevent terminal input
	default:
		// Let tview handle all other keys (including up/down for item navigation)
		return event
	}
}

// calculateMenuHeight determines the proper height for a dropdown menu
// Formula derived from empirical data points:
// 1 item → 4 rows, 2 items → 6 rows, 3 items → 8 rows
// Pattern: 4, 6, 8... = 2 * itemCount + 2
func calculateMenuHeight(itemCount int) int {
	if itemCount <= 0 {
		return 4
	}

	// Mathematical formula: height = 2 * items + 2
	// This gives us: 1→4, 2→6, 3→8, 4→10, etc.
	return 2*itemCount + 2
}
