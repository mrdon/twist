package menus

import (
	"twist/internal/api"
	twistComponents "twist/internal/components"
	"twist/internal/debug"
)

// MenuHandler interface that all menu implementations must satisfy
type MenuHandler interface {
	HandleMenuAction(action string, app AppInterface) error
	GetMenuItems() []twistComponents.MenuItem
}

// ModalAwareMenuHandler is an optional interface that menu handlers can implement
// to indicate whether their actions create modals, eliminating the need for hardcoded lists
type ModalAwareMenuHandler interface {
	MenuHandler
	ActionCreatesModal(action string) bool
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
	ShowInputDialog(pageName string, dialog interface{}) // For showing custom input dialogs
	CloseModal()

	// Terminal info for dynamic sizing
	GetTerminalWidth() int

	// Version information
	GetVersion() string
	GetCommit() string
	GetDate() string

	// Proxy API access
	GetProxyAPI() api.ProxyAPI
	IsConnected() bool // Returns true if connected to a game server
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

	// Check if the action corresponds to an enabled menu item
	enabledItems := mm.GetEnabledMenuItems(menuName, app)
	for _, enabledItem := range enabledItems {
		if enabledItem.MenuItem.Label == action {
			if !enabledItem.Enabled {
				debug.Log("MenuManager: Action '%s' is disabled for menu '%s'", action, menuName)
				return nil // Don't execute disabled actions
			}
			break
		}
	}

	return handler.HandleMenuAction(action, app)
}

// GetMenuItems returns menu items for a specific menu
func (mm *MenuManager) GetMenuItems(menuName string) []twistComponents.MenuItem {
	return mm.registry.GetMenuItems(menuName)
}

// GetEnabledMenuItems returns menu items with enablement status evaluated
func (mm *MenuManager) GetEnabledMenuItems(menuName string, app AppInterface) []EnabledMenuItem {
	config := mm.registry.GetMenuConfig(menuName)
	if config == nil {
		return []EnabledMenuItem{}
	}

	result := make([]EnabledMenuItem, len(config.Items))
	for i, item := range config.Items {
		enabled := true // Default to enabled

		// Check if we have an enablement checker for this item
		if i < len(config.ItemEnabledChecks) && config.ItemEnabledChecks[i] != nil {
			enabled = config.ItemEnabledChecks[i](app)
		}

		result[i] = EnabledMenuItem{
			MenuItem: item,
			Enabled:  enabled,
		}
	}

	return result
}

// EnabledMenuItem wraps a MenuItem with its enabled status
type EnabledMenuItem struct {
	MenuItem twistComponents.MenuItem
	Enabled  bool
}

// GetMenuItem returns the wrapped MenuItem (implements EnabledMenuItemInterface)
func (e EnabledMenuItem) GetMenuItem() twistComponents.MenuItem {
	return e.MenuItem
}

// IsEnabled returns whether this menu item is enabled (implements EnabledMenuItemInterface)
func (e EnabledMenuItem) IsEnabled() bool {
	return e.Enabled
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

// ActionCreatesModal checks if a menu action creates a modal dialog
func (mm *MenuManager) ActionCreatesModal(menuName, action string) bool {
	handler := mm.registry.GetMenuHandler(menuName)
	if handler == nil {
		return false
	}

	// Check if the handler implements ModalAwareMenuHandler
	if modalAware, ok := handler.(ModalAwareMenuHandler); ok {
		return modalAware.ActionCreatesModal(action)
	}

	// Fallback: assume actions don't create modals unless explicitly declared
	return false
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
