package menus

import (
	twistComponents "twist/internal/components"
	"twist/internal/debug"
)

// ViewMenu handles View menu actions
type ViewMenu struct{}

// NewViewMenu creates a new view menu handler
func NewViewMenu() *ViewMenu {
	return &ViewMenu{}
}

// GetMenuItems returns the menu items for the View menu
func (v *ViewMenu) GetMenuItems() []twistComponents.MenuItem {
	return []twistComponents.MenuItem{
		{Label: "Scripts", Shortcut: ""},
		{Label: "Zoom In", Shortcut: ""},
		{Label: "Zoom Out", Shortcut: ""},
		{Label: "Full Screen", Shortcut: ""},
		{Label: "Panels", Shortcut: ""},
	}
}

// HandleMenuAction processes View menu actions
func (v *ViewMenu) HandleMenuAction(action string, app AppInterface) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Error("PANIC in ViewMenu.HandleMenuAction", "error", r)
		}
	}()

	switch action {
	case "Scripts":
		return v.handleScripts(app)
	case "Zoom In":
		return v.handleZoomIn(app)
	case "Zoom Out":
		return v.handleZoomOut(app)
	case "Full Screen":
		return v.handleFullScreen(app)
	case "Panels":
		return v.handlePanels(app)
	default:
		debug.Info("ViewMenu: Unknown action", "action", action)
		return nil
	}
}

// handleScripts opens script management (not implemented yet)
func (v *ViewMenu) handleScripts(app AppInterface) error {
	app.ShowModal("Scripts",
		"Script management feature not yet implemented.\n\n"+
			"Note: Script functionality is available through the terminal\n"+
			"menu system when connected to a game server.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleZoomIn increases font size (not implemented yet)
func (v *ViewMenu) handleZoomIn(app AppInterface) error {
	app.ShowModal("Zoom In",
		"Zoom In feature not yet implemented.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleZoomOut decreases font size (not implemented yet)
func (v *ViewMenu) handleZoomOut(app AppInterface) error {
	app.ShowModal("Zoom Out",
		"Zoom Out feature not yet implemented.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handleFullScreen toggles full screen mode (not implemented yet)
func (v *ViewMenu) handleFullScreen(app AppInterface) error {
	app.ShowModal("Full Screen",
		"Full Screen toggle feature not yet implemented.",
		[]string{"OK"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})
	return nil
}

// handlePanels toggles panel visibility
func (v *ViewMenu) handlePanels(app AppInterface) error {
	if app.GetPanelsVisible() {
		app.HidePanels()
		debug.Info("ViewMenu: Hiding panels")
	} else {
		app.ShowPanels()
		debug.Info("ViewMenu: Showing panels")
	}
	return nil
}
