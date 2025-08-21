package menus

import (
	twistComponents "twist/internal/components"
	"twist/internal/debug"
)

// HelpMenu handles Help menu actions
type HelpMenu struct{}

// NewHelpMenu creates a new help menu handler
func NewHelpMenu() *HelpMenu {
	return &HelpMenu{}
}

// GetMenuItems returns the menu items for the Help menu
func (h *HelpMenu) GetMenuItems() []twistComponents.MenuItem {
	return []twistComponents.MenuItem{
		{Label: "Keyboard Shortcuts", Shortcut: "F1"},
		{Label: "About", Shortcut: ""},
		{Label: "User Manual", Shortcut: ""},
	}
}

// HandleMenuAction processes Help menu actions
func (h *HelpMenu) HandleMenuAction(action string, app AppInterface) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in HelpMenu.HandleMenuAction: %v", r)
		}
	}()

	switch action {
	case "Keyboard Shortcuts":
		return h.handleKeyboardShortcuts(app)
	case "About":
		return h.handleAbout(app)
	case "User Manual":
		return h.handleUserManual(app)
	default:
		debug.Log("HelpMenu: Unknown action '%s'", action)
		return nil
	}
}

// handleKeyboardShortcuts shows the keyboard shortcuts help
func (h *HelpMenu) handleKeyboardShortcuts(app AppInterface) error {
	helpText := "TWIST Terminal Interface - Keyboard Shortcuts\n\n" +
		"Menu Navigation:\n" +
		"Alt+S = Session menu\n" +
		"Alt+V = View menu\n" +
		"Alt+T = Terminal menu\n" +
		"Alt+H = Help menu\n\n" +
		"Quick Actions:\n" +
		"Alt+C = Connect to server\n" +
		"Alt+D = Disconnect from server\n" +
		"Alt+Q = Quit application\n\n" +
		"Function Keys:\n" +
		"F1 = Show this help screen\n" +
		"ESC = Close dialogs and menus\n\n" +
		"Global Keys:\n" +
		"Ctrl+C = Exit application"

	app.ShowModal("Keyboard Shortcuts", helpText, []string{"Close"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleAbout shows version and build information
func (h *HelpMenu) handleAbout(app AppInterface) error {
	aboutText := "TWIST Terminal Interface\n\n" +
		"Version: " + app.GetVersion() + "\n" +
		"Commit: " + app.GetCommit() + "\n" +
		"Build Date: " + app.GetDate() + "\n\n" +
		"A Trade Wars 2002 proxy client with scripting support.\n" +
		"Built with Go and tview for cross-platform terminal interfaces.\n\n" +
		"Features:\n" +
		"• Real-time game data parsing\n" +
		"• TWX-compatible scripting engine\n" +
		"• Sector mapping and navigation\n" +
		"• Database integration\n" +
		"• Cross-platform terminal UI"

	app.ShowModal("About TWIST", aboutText, []string{"Close"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleUserManual shows user manual information (not implemented yet)
func (h *HelpMenu) handleUserManual(app AppInterface) error {
	app.ShowModal("User Manual",
		"User Manual feature not yet implemented.\n\n"+
			"For now, use F1 or Help → Keyboard Shortcuts\n"+
			"for basic usage information.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}
