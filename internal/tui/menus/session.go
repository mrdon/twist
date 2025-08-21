package menus

import (
	twistComponents "twist/internal/components"
	"twist/internal/debug"
)

// SessionMenu handles Session menu actions
type SessionMenu struct{}

// NewSessionMenu creates a new session menu handler
func NewSessionMenu() *SessionMenu {
	return &SessionMenu{}
}

// GetMenuItems returns the menu items for the Session menu
func (s *SessionMenu) GetMenuItems() []twistComponents.MenuItem {
	return []twistComponents.MenuItem{
		{Label: "Connect", Shortcut: ""},
		{Label: "Recent Connections", Shortcut: ""},
		{Label: "Disconnect", Shortcut: ""},
		{Label: "Save Session", Shortcut: ""},
		{Label: "Quit", Shortcut: "Alt+Q"},
	}
}

// HandleMenuAction processes Session menu actions
func (s *SessionMenu) HandleMenuAction(action string, app AppInterface) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in SessionMenu.HandleMenuAction: %v", r)
		}
	}()

	switch action {
	case "Connect":
		return s.handleConnect(app)
	case "Recent Connections":
		return s.handleRecentConnections(app)
	case "Disconnect":
		return s.handleDisconnect(app)
	case "Save Session":
		return s.handleSaveSession(app)
	case "Quit":
		return s.handleQuit(app)
	default:
		debug.Log("SessionMenu: Unknown action '%s'", action)
		return nil
	}
}

// handleConnect shows the connection dialog
func (s *SessionMenu) handleConnect(app AppInterface) error {
	app.ShowConnectionDialog()
	return nil
}

// handleRecentConnections shows recent connections (not implemented yet)
func (s *SessionMenu) handleRecentConnections(app AppInterface) error {
	app.ShowModal("Recent Connections",
		"Recent connections feature not yet implemented.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleDisconnect disconnects from the server
func (s *SessionMenu) handleDisconnect(app AppInterface) error {
	app.Disconnect()
	return nil
}

// handleSaveSession saves the current session (not implemented yet)
func (s *SessionMenu) handleSaveSession(app AppInterface) error {
	app.ShowModal("Save Session",
		"Save session feature not yet implemented.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleQuit exits the application
func (s *SessionMenu) handleQuit(app AppInterface) error {
	app.Exit()
	return nil
}
