package menus

import (
	twistComponents "twist/internal/components"
)

// MenuConfig defines a complete menu with all its properties
type MenuConfig struct {
	Name     string                     // Menu name (e.g., "Session")
	Shortcut string                     // Alt key shortcut (e.g., "Alt+S")
	Items    []twistComponents.MenuItem // Menu items with their shortcuts
	Handler  MenuHandler                // Handler for this menu
}

// MenuRegistry provides centralized menu configuration
type MenuRegistry struct {
	menus []MenuConfig
}

// NewMenuRegistry creates a new centralized menu registry
func NewMenuRegistry() *MenuRegistry {
	registry := &MenuRegistry{}
	registry.initializeMenus()
	return registry
}

// initializeMenus defines ALL menus in one place
func (mr *MenuRegistry) initializeMenus() {
	mr.menus = []MenuConfig{
		{
			Name:     "Session",
			Shortcut: "Alt+S",
			Items: []twistComponents.MenuItem{
				{Label: "Connect", Shortcut: "Alt+C"},
				{Label: "Disconnect", Shortcut: "Alt+D"},
				{Label: "Quit", Shortcut: "Alt+Q"},
			},
			Handler: NewSessionMenu(),
		},
		{
			Name:     "View",
			Shortcut: "Alt+V",
			Items: []twistComponents.MenuItem{
				{Label: "Panels", Shortcut: ""},
			},
			Handler: NewViewMenu(),
		},
		{
			Name:     "Terminal",
			Shortcut: "Alt+T",
			Items: []twistComponents.MenuItem{
				{Label: "Clear", Shortcut: ""},
			},
			Handler: NewTerminalMenu(),
		},
		{
			Name:     "Help",
			Shortcut: "Alt+H",
			Items: []twistComponents.MenuItem{
				{Label: "Keyboard Shortcuts", Shortcut: "F1"},
				{Label: "About", Shortcut: ""},
			},
			Handler: NewHelpMenu(),
		},
	}
}

// GetMenus returns all menu configurations
func (mr *MenuRegistry) GetMenus() []MenuConfig {
	return mr.menus
}

// GetMenuNames returns just the menu names in order
func (mr *MenuRegistry) GetMenuNames() []string {
	names := make([]string, len(mr.menus))
	for i, menu := range mr.menus {
		names[i] = menu.Name
	}
	return names
}

// GetMenuConfig returns the config for a specific menu
func (mr *MenuRegistry) GetMenuConfig(name string) *MenuConfig {
	for i := range mr.menus {
		if mr.menus[i].Name == name {
			return &mr.menus[i]
		}
	}
	return nil
}

// GetMenuHandler returns the handler for a specific menu
func (mr *MenuRegistry) GetMenuHandler(name string) MenuHandler {
	config := mr.GetMenuConfig(name)
	if config != nil {
		return config.Handler
	}
	return nil
}

// GetMenuItems returns the items for a specific menu
func (mr *MenuRegistry) GetMenuItems(name string) []twistComponents.MenuItem {
	config := mr.GetMenuConfig(name)
	if config != nil {
		return config.Items
	}
	return []twistComponents.MenuItem{}
}

// CalculateDropdownPosition calculates the X position for a dropdown menu
// This auto-calculates based on menu order and length
func (mr *MenuRegistry) CalculateDropdownPosition(menuName string) int {
	position := 1 // Start with 1 space padding

	for _, menu := range mr.menus {
		if menu.Name == menuName {
			return position
		}
		// Add the length of this menu name plus spacing (2 spaces between menus)
		position += len(menu.Name) + 2
	}

	return 1 // Default fallback
}

// GetAllKeyboardShortcuts returns all shortcuts for help documentation
func (mr *MenuRegistry) GetAllKeyboardShortcuts() map[string]string {
	shortcuts := make(map[string]string)

	// Add menu shortcuts
	for _, menu := range mr.menus {
		if menu.Shortcut != "" {
			shortcuts[menu.Shortcut] = menu.Name + " menu"
		}

		// Add item shortcuts
		for _, item := range menu.Items {
			if item.Shortcut != "" {
				shortcuts[item.Shortcut] = item.Label
			}
		}
	}

	// Add global shortcuts
	shortcuts["ESC"] = "Close dialogs and menus"
	shortcuts["Ctrl+C"] = "Exit application"

	return shortcuts
}

// GetShortcutHandler returns the handler for a specific shortcut
func (mr *MenuRegistry) GetShortcutHandler(shortcut string) (string, string, bool) {
	// Check menu shortcuts first
	for _, menu := range mr.menus {
		if menu.Shortcut == shortcut {
			return menu.Name, "", true // Menu shortcut, no specific action
		}

		// Check item shortcuts
		for _, item := range menu.Items {
			if item.Shortcut == shortcut {
				return menu.Name, item.Label, true // Menu and action
			}
		}
	}

	return "", "", false
}
