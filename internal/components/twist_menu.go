package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// MenuItem represents a menu item with optional shortcut
type MenuItem struct {
	Label    string
	Shortcut string
}

// MenuBorderStyle defines predefined border character sets for menus
type MenuBorderStyle int

const (
	MenuBorderStyleSingle  MenuBorderStyle = iota // Single-line box drawing characters
	MenuBorderStyleDouble                         // Double-line box drawing characters
	MenuBorderStyleHeavy                          // Heavy/thick box drawing characters
	MenuBorderStyleRounded                        // Rounded corner characters
)

// BorderChars defines the characters used for drawing borders
type BorderChars struct {
	Normal MenuBorderStyle // Style for normal state
	Focus  MenuBorderStyle // Style for focused state
}

// NewBorderChars creates BorderChars with normal and focus styles
func NewBorderChars(normal, focus MenuBorderStyle) *BorderChars {
	return &BorderChars{
		Normal: normal,
		Focus:  focus,
	}
}

// NewSimpleBorderChars creates BorderChars with the same style for normal and focus
func NewSimpleBorderChars(style MenuBorderStyle) *BorderChars {
	return NewBorderChars(style, style)
}

// getRunes returns the actual rune characters for a given style
func getRunes(style MenuBorderStyle) (horizontal, vertical, topLeft, topRight, bottomLeft, bottomRight rune) {
	switch style {
	case MenuBorderStyleSingle:
		return '─', '│', '┌', '┐', '└', '┘'
	case MenuBorderStyleDouble:
		return '═', '║', '╔', '╗', '╚', '╝'
	case MenuBorderStyleHeavy:
		return '━', '┃', '┏', '┓', '┗', '┛'
	case MenuBorderStyleRounded:
		return '─', '│', '╭', '╮', '╰', '╯'
	default:
		return '─', '│', '┌', '┐', '└', '┘'
	}
}

// ToTviewBorders converts our BorderChars to tview's global Borders struct type
func (bc *BorderChars) ToTviewBorders() struct {
	Horizontal       rune
	Vertical         rune
	TopLeft          rune
	TopRight         rune
	BottomLeft       rune
	BottomRight      rune
	LeftT            rune
	RightT           rune
	TopT             rune
	BottomT          rune
	Cross            rune
	HorizontalFocus  rune
	VerticalFocus    rune
	TopLeftFocus     rune
	TopRightFocus    rune
	BottomLeftFocus  rune
	BottomRightFocus rune
} {
	// Get runes for normal state
	normalH, normalV, normalTL, normalTR, normalBL, normalBR := getRunes(bc.Normal)
	// Get runes for focus state
	focusH, focusV, focusTL, focusTR, focusBL, focusBR := getRunes(bc.Focus)

	return struct {
		Horizontal       rune
		Vertical         rune
		TopLeft          rune
		TopRight         rune
		BottomLeft       rune
		BottomRight      rune
		LeftT            rune
		RightT           rune
		TopT             rune
		BottomT          rune
		Cross            rune
		HorizontalFocus  rune
		VerticalFocus    rune
		TopLeftFocus     rune
		TopRightFocus    rune
		BottomLeftFocus  rune
		BottomRightFocus rune
	}{
		Horizontal:  normalH,
		Vertical:    normalV,
		TopLeft:     normalTL,
		TopRight:    normalTR,
		BottomLeft:  normalBL,
		BottomRight: normalBR,
		// Use standard junction characters for T-joints and cross
		LeftT:            '├',
		RightT:           '┤',
		TopT:             '┬',
		BottomT:          '┴',
		Cross:            '┼',
		HorizontalFocus:  focusH,
		VerticalFocus:    focusV,
		TopLeftFocus:     focusTL,
		TopRightFocus:    focusTR,
		BottomLeftFocus:  focusBL,
		BottomRightFocus: focusBR,
	}
}

// TwistMenu wraps a tview.List to provide custom border character support
// This allows us to have different border styles for menus without affecting
// other tview components throughout the application.
type TwistMenu struct {
	*tview.List
	borderChars     *BorderChars
	shortcutManager *ShortcutManager
	menuItems       []MenuItem
}

// NewTwistMenu creates a new TwistMenu with optional custom border characters
func NewTwistMenu(borderChars *BorderChars) *TwistMenu {
	return &TwistMenu{
		List:            tview.NewList(),
		borderChars:     borderChars,
		shortcutManager: NewShortcutManager(),
		menuItems:       make([]MenuItem, 0),
	}
}

// SetBorderChars updates the border characters for this menu
func (tm *TwistMenu) SetBorderChars(borderChars *BorderChars) *TwistMenu {
	tm.borderChars = borderChars
	return tm
}

// AddMenuItem adds a menu item with automatic shortcut registration
func (tm *TwistMenu) AddMenuItem(item MenuItem, callback func()) {
	tm.menuItems = append(tm.menuItems, item)

	// Add to the underlying tview.List
	tm.List.AddItem(item.Label, "", 0, callback)

	// Register shortcut if present
	if item.Shortcut != "" {
		tm.shortcutManager.RegisterShortcut(item.Shortcut, callback)
	}
}

// SetMenuItems replaces all menu items with automatic shortcut registration and proper display formatting
func (tm *TwistMenu) SetMenuItems(items []MenuItem, callbacks []func()) {
	// Clear existing items and shortcuts
	tm.List.Clear()
	tm.shortcutManager = NewShortcutManager() // Reset shortcuts
	tm.menuItems = items

	// Check if any items have shortcuts to determine layout
	hasAnyShortcuts := false
	maxLabelWidth := 0
	maxShortcutWidth := 0

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

	// Add items with proper display formatting
	for i, item := range items {
		var callback func()
		if i < len(callbacks) {
			callback = callbacks[i]
		}

		// Format display text with shortcut if present
		displayText := item.Label
		if hasAnyShortcuts && item.Shortcut != "" {
			// Calculate padding for right-justified shortcut
			padding := maxLabelWidth - len(item.Label) + 3 // 3 spaces minimum between label and shortcut
			spaces := ""
			for j := 0; j < padding; j++ {
				spaces += " "
			}
			displayText = item.Label + spaces + item.Shortcut
		}

		// Add to the underlying tview.List with formatted display text
		tm.List.AddItem(displayText, "", 0, callback)

		// Register shortcut if present
		if item.Shortcut != "" {
			tm.shortcutManager.RegisterShortcut(item.Shortcut, callback)
		}
	}
}

// HandleShortcut processes keyboard shortcuts for menu items
func (tm *TwistMenu) HandleShortcut(event *tcell.EventKey) bool {
	return tm.shortcutManager.HandleKeyEvent(event)
}

// GetMenuItems returns the current menu items
func (tm *TwistMenu) GetMenuItems() []MenuItem {
	return tm.menuItems
}

// Draw overrides the default tview.List Draw method to apply custom border characters
// CUSTOM BORDER IMPLEMENTATION:
// We temporarily modify tview's global Borders variable to use our custom characters,
// call the original Draw method, then restore the original borders. This approach
// allows us to customize borders per-component without forking tview or affecting
// other components in the application.
func (tm *TwistMenu) Draw(screen tcell.Screen) {
	if tm.borderChars != nil {
		// Save the original tview border characters
		originalBorders := tview.Borders

		// Apply our custom border characters globally
		tview.Borders = tm.borderChars.ToTviewBorders()

		// Call the original List.Draw method with our custom borders
		tm.List.Draw(screen)

		// Restore the original border characters to avoid affecting other components
		tview.Borders = originalBorders
	} else {
		// No custom borders - use default tview drawing
		tm.List.Draw(screen)
	}
}
