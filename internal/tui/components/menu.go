package components

import (
	"twist/internal/theme"
	twistComponents "twist/internal/components"
	
	"github.com/rivo/tview"
)


// MenuComponent manages the menu bar component
type MenuComponent struct {
	view          *tview.TextView
	dropdown      *DropdownMenu
	dropdownPages *tview.Pages
	activeMenu    string
	connected     bool
}

// NewMenuComponent creates a new menu component
func NewMenuComponent() *MenuComponent {
	// Use theme factory for menu bar styling
	menuBar := theme.NewMenuBar().
		SetDynamicColors(true).
		SetRegions(false).
		SetWrap(false).
		SetTextAlign(tview.AlignLeft)
	
	// Set initial menu text with traditional menu bar style
	menuBar.SetText(" Session  Edit  View  Terminal  Help")
	
	return &MenuComponent{
		view:          menuBar,
		dropdown:      NewDropdownMenu(),
		dropdownPages: tview.NewPages(),
		activeMenu:    "",
		connected:     false,
	}
}

// GetView returns the menu view
func (mc *MenuComponent) GetView() *tview.TextView {
	return mc.view
}

// UpdateMenu updates the menu bar text
func (mc *MenuComponent) UpdateMenu(connected bool) {
	mc.connected = connected
	mc.updateMenuWithHighlight()
}

// updateMenuWithHighlight updates the menu text with active menu highlighting
func (mc *MenuComponent) updateMenuWithHighlight() {
	
	// Create menu items array
	var menus []string
	menus = []string{"Session", "Edit", "View", "Terminal", "Help"}
	
	// Build menu text with highlighting using theme colors
	menuText := " "
	for _, menu := range menus {
		if menu == mc.activeMenu {
			// Highlight active menu: white text on red background
			menuText += "[white:red]" + menu + "[:-]  "
		} else {
			menuText += menu + "  "
		}
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

// ShowDropdown displays a dropdown menu with MenuItem structs that support shortcuts
func (mc *MenuComponent) ShowDropdown(menuName string, items []MenuItem, callback func(string), navCallback func(string), globalShortcuts *twistComponents.GlobalShortcutManager) *tview.Flex {
	// Calculate position based on menu item
	var leftOffset int
	switch menuName {
	case "Session":
		leftOffset = 1
	case "Edit":
		leftOffset = 10
	case "View":
		leftOffset = 16
	case "Terminal":
		leftOffset = 22
	case "Help":
		leftOffset = 32
	default:
		leftOffset = 1
	}
	
	// Set navigation callback for arrow keys
	mc.dropdown.SetNavigationCallback(navCallback)
	
	return mc.dropdown.Show(menuName, items, leftOffset, callback, globalShortcuts)
}

// HideDropdown hides the current dropdown menu
func (mc *MenuComponent) HideDropdown() {
	mc.dropdown.Hide()
}

// IsDropdownVisible returns whether a dropdown is currently visible
func (mc *MenuComponent) IsDropdownVisible() bool {
	return mc.dropdown.IsVisible()
}

// GetDropdownList returns the dropdown list for focus management
func (mc *MenuComponent) GetDropdownList() *twistComponents.TwistMenu {
	return mc.dropdown.GetList()
}