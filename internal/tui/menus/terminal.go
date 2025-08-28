package menus

import (
	twistComponents "twist/internal/components"
	"twist/internal/debug"
)

// TerminalMenu handles Terminal menu actions
type TerminalMenu struct{}

// NewTerminalMenu creates a new terminal menu handler
func NewTerminalMenu() *TerminalMenu {
	return &TerminalMenu{}
}

// GetMenuItems returns the menu items for the Terminal menu
func (t *TerminalMenu) GetMenuItems() []twistComponents.MenuItem {
	return []twistComponents.MenuItem{
		{Label: "Clear", Shortcut: ""},
		{Label: "Scroll Up", Shortcut: ""},
		{Label: "Scroll Down", Shortcut: ""},
		{Label: "Copy Selection", Shortcut: ""},
	}
}

// HandleMenuAction processes Terminal menu actions
func (t *TerminalMenu) HandleMenuAction(action string, app AppInterface) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Error("PANIC in TerminalMenu.HandleMenuAction", "error", r)
		}
	}()

	switch action {
	case "Clear":
		return t.handleClear(app)
	case "Scroll Up":
		return t.handleScrollUp(app)
	case "Scroll Down":
		return t.handleScrollDown(app)
	case "Copy Selection":
		return t.handleCopySelection(app)
	default:
		debug.Info("TerminalMenu: Unknown action", "action", action)
		return nil
	}
}

// handleClear clears the terminal content
func (t *TerminalMenu) handleClear(app AppInterface) error {
	app.ClearTerminal()
	debug.Info("TerminalMenu: Cleared terminal")
	return nil
}

// handleScrollUp scrolls terminal up (not implemented yet)
func (t *TerminalMenu) handleScrollUp(app AppInterface) error {
	app.ShowModal("Scroll Up",
		"Scroll Up feature not yet implemented.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleScrollDown scrolls terminal down (not implemented yet)
func (t *TerminalMenu) handleScrollDown(app AppInterface) error {
	app.ShowModal("Scroll Down",
		"Scroll Down feature not yet implemented.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleCopySelection copies selected terminal text (not implemented yet)
func (t *TerminalMenu) handleCopySelection(app AppInterface) error {
	app.ShowModal("Copy Selection",
		"Copy Selection feature not yet implemented.\n\n"+
			"This would require implementing text selection\n"+
			"and clipboard integration in the terminal component.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}
