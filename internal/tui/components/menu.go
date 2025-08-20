package components

import (
	"strings"
	twistComponents "twist/internal/components"
	"twist/internal/theme"

	"github.com/rivo/tview"
)

// MenuComponent manages the menu bar component
type MenuComponent struct {
	view          *tview.TextView
	dropdown      *DropdownMenu
	dropdownPages *tview.Pages
	activeMenu    string
	connected     bool
	targetWidth   int // Target width to match status bar
}

// NewMenuComponent creates a new menu component
func NewMenuComponent() *MenuComponent {
	// Use theme factory for menu bar styling
	menuBar := theme.NewMenuBar().
		SetDynamicColors(true).
		SetRegions(false).
		SetWrap(false).
		SetTextAlign(tview.AlignLeft)

	// Set component background to theme background to prevent bleeding
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	menuBar.SetBackgroundColor(defaultColors.Background)

	mc := &MenuComponent{
		view:          menuBar,
		dropdown:      NewDropdownMenu(),
		dropdownPages: tview.NewPages(),
		activeMenu:    "",
		connected:     false,
		targetWidth:   0,
	}

	// Set initial menu text
	mc.updateMenuWithHighlight()

	return mc
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
	// Get theme colors
	currentTheme := theme.Current()
	menuColors := currentTheme.MenuColors()

	// Create menu items array
	menus := []string{"Session", "Edit", "View", "Terminal", "Help"}

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

	// Always add one space at the end
	menuText += " "

	// Calculate content length (without color tags)
	plainText := mc.stripColorTags(menuText)
	contentLength := len(plainText)

	// Minimum width is first two panels (30 + 80 = 110) or target width from status bar
	minPanelWidth := 110
	targetWidth := mc.targetWidth
	if targetWidth == 0 {
		targetWidth = minPanelWidth
	}

	// Final width is the larger of content length or target width
	if targetWidth > contentLength {
		// Add padding spaces to reach target width
		paddingNeeded := targetWidth - contentLength
		menuText += strings.Repeat(" ", paddingNeeded)
	}

	// Apply explicit background color to the padded menu content
	finalText := "[:" + menuColors.Background.String() + "]" + menuText + "[-:-]"
	mc.view.SetText(finalText)
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

// SetTargetWidth sets the target width to match status bar
func (mc *MenuComponent) SetTargetWidth(width int) {
	mc.targetWidth = width
	mc.updateMenuWithHighlight()
}

// stripColorTags removes tview color tags from text to calculate actual display length
func (mc *MenuComponent) stripColorTags(text string) string {
	result := text

	// Remove color reset tags [-] and [-:-]
	result = strings.ReplaceAll(result, "[-]", "")
	result = strings.ReplaceAll(result, "[-:-]", "")

	// Remove simple color tags by finding patterns like [colorname] and [color:background]
	for strings.Contains(result, "[") && strings.Contains(result, "]") {
		start := strings.Index(result, "[")
		end := strings.Index(result[start:], "]")
		if end != -1 {
			result = result[:start] + result[start+end+1:]
		} else {
			break
		}
	}

	return result
}
