package menus

import (
	twistComponents "twist/internal/components"
	"twist/internal/debug"
)

// MenuHandler interface that all menu implementations must satisfy
type MenuHandler interface {
	HandleMenuAction(action string, app AppInterface) error
	GetMenuItems() []twistComponents.MenuItem
}

// AppInterface defines the methods that menu handlers need from TwistApp
// This prevents circular dependencies and makes testing easier
type AppInterface interface {
	// Connection management
	Connect(address string)
	Disconnect()
	Exit()
	ShowConnectionDialog()

	// Panel management
	ShowPanels()
	HidePanels()
	GetPanelsVisible() bool

	// Terminal operations
	ClearTerminal()

	// Modal management
	ShowModal(title, text string, buttons []string, callback func(int, string))
	CloseModal()

	// Version information
	GetVersion() string
	GetCommit() string
	GetDate() string
}

// MenuManager coordinates all menu handlers
type MenuManager struct {
	registry *MenuRegistry
}

// NewMenuManager creates a new menu manager
func NewMenuManager() *MenuManager {
	return &MenuManager{
		registry: NewMenuRegistry(),
	}
}

// HandleMenuAction delegates to the appropriate menu handler
func (mm *MenuManager) HandleMenuAction(menuName, action string, app AppInterface) error {
	handler := mm.registry.GetMenuHandler(menuName)
	if handler == nil {
		debug.Log("MenuManager: No handler found for menu '%s'", menuName)
		return nil // Don't error, just ignore unhandled menus
	}

	return handler.HandleMenuAction(action, app)
}

// GetMenuItems returns menu items for a specific menu
func (mm *MenuManager) GetMenuItems(menuName string) []twistComponents.MenuItem {
	return mm.registry.GetMenuItems(menuName)
}

// GetMenuOptions returns string options for backward compatibility
func (mm *MenuManager) GetMenuOptions(menuName string) []string {
	items := mm.GetMenuItems(menuName)
	options := make([]string, len(items))
	for i, item := range items {
		options[i] = item.Label
	}
	return options
}

// GetMenuNames returns all menu names in order
func (mm *MenuManager) GetMenuNames() []string {
	return mm.registry.GetMenuNames()
}

// GetDropdownPosition calculates auto-positioned dropdown location
func (mm *MenuManager) GetDropdownPosition(menuName string) int {
	return mm.registry.CalculateDropdownPosition(menuName)
}

// GetAllShortcuts returns all keyboard shortcuts for help
func (mm *MenuManager) GetAllShortcuts() map[string]string {
	return mm.registry.GetAllKeyboardShortcuts()
}

// HandleShortcut processes a keyboard shortcut
func (mm *MenuManager) HandleShortcut(shortcut string, app AppInterface) bool {
	menuName, action, found := mm.registry.GetShortcutHandler(shortcut)
	if !found {
		return false
	}

	if action == "" {
		// This is a menu shortcut (like Alt+S), not implemented here
		// Menu shortcuts are handled by input handlers
		return false
	}

	// This is an action shortcut (like Alt+Q for Quit)
	err := mm.HandleMenuAction(menuName, action, app)
	return err == nil
}
