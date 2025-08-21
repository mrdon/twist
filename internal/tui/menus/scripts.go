package menus

import (
	"fmt"
	"strings"
	twistComponents "twist/internal/components"
	"twist/internal/debug"
	"twist/internal/tui/components"
)

// ScriptsMenu handles Scripts menu actions
type ScriptsMenu struct{}

// NewScriptsMenu creates a new scripts menu handler
func NewScriptsMenu() *ScriptsMenu {
	return &ScriptsMenu{}
}

// GetMenuItems returns the menu items for the Scripts menu
func (s *ScriptsMenu) GetMenuItems() []twistComponents.MenuItem {
	return []twistComponents.MenuItem{
		{Label: "List", Shortcut: ""},
		{Label: "Burst", Shortcut: ""},
	}
}

// HandleMenuAction processes Scripts menu actions
func (s *ScriptsMenu) HandleMenuAction(action string, app AppInterface) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ScriptsMenu.HandleMenuAction: %v", r)
		}
	}()

	switch action {
	case "List":
		return s.handleList(app)
	case "Burst":
		return s.handleBurst(app)
	default:
		debug.Log("ScriptsMenu: Unknown action '%s'", action)
		return nil
	}
}

// handleList shows a modal listing all loaded scripts and their status
func (s *ScriptsMenu) handleList(app AppInterface) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleList: %v", r)
		}
	}()

	proxyAPI := app.GetProxyAPI()
	if proxyAPI == nil {
		app.ShowModal("Scripts List",
			"Not connected to proxy. Please connect first.",
			[]string{"OK"},
			func(buttonIndex int, buttonLabel string) {
				app.CloseModal()
			})
		return nil
	}

	// Get script list from proxy
	scripts, err := proxyAPI.GetScriptList()
	if err != nil {
		app.ShowModal("Scripts List Error",
			fmt.Sprintf("Error getting script list: %v", err),
			[]string{"OK"},
			func(buttonIndex int, buttonLabel string) {
				app.CloseModal()
			})
		return nil
	}

	// Build script list text
	var listText strings.Builder
	if len(scripts) == 0 {
		listText.WriteString("No scripts loaded.\n\n")
	} else {
		listText.WriteString(fmt.Sprintf("Loaded Scripts (%d):\n\n", len(scripts)))
		for i, script := range scripts {
			status := "Inactive"
			if script.IsActive {
				status = "Active"
			}
			listText.WriteString(fmt.Sprintf("%d. %s\n", i+1, script.Name))
			listText.WriteString(fmt.Sprintf("   File: %s\n", script.Filename))
			listText.WriteString(fmt.Sprintf("   Status: %s\n\n", status))
		}
	}

	app.ShowModal("Scripts List", listText.String(), []string{"Close"},
		func(buttonIndex int, buttonLabel string) {
			app.CloseModal()
		})

	return nil
}

// handleBurst shows a modal for entering and sending a burst command
func (s *ScriptsMenu) handleBurst(app AppInterface) error {
	proxyAPI := app.GetProxyAPI()
	if proxyAPI == nil {
		app.ShowModal("Burst Command",
			"Not connected to proxy. Please connect first.",
			[]string{"OK"},
			func(buttonIndex int, buttonLabel string) {
				app.CloseModal()
			})
		return nil
	}

	// Create input dialog for burst command
	inputDialog := components.NewBurstInputDialog(
		func(burstText string) {
			// Send burst command via proxy API
			err := proxyAPI.SendBurstCommand(burstText)
			if err != nil {
				app.ShowModal("Burst Command Error",
					fmt.Sprintf("Error sending burst command: %v", err),
					[]string{"OK"},
					func(buttonIndex int, buttonLabel string) {
						app.CloseModal()
					})
			} else {
				app.CloseModal()
			}
		},
		func() {
			app.CloseModal()
		},
	)

	// Show the input dialog
	s.showInputDialog(app, inputDialog)
	return nil
}

// showInputDialog displays an input dialog (similar to connection dialog pattern)
func (s *ScriptsMenu) showInputDialog(app AppInterface, dialog *components.BurstInputDialog) {
	app.ShowInputDialog("burst-input-dialog", dialog)
}