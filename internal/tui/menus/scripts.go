package menus

import (
	"fmt"
	"strings"
	"twist/internal/api"
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

// ActionCreatesModal implements ModalAwareMenuHandler
func (s *ScriptsMenu) ActionCreatesModal(action string) bool {
	switch action {
	case "List", "Burst":
		return true // These actions create modal dialogs
	default:
		return false
	}
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

	debug.Log("ScriptsMenu.HandleMenuAction: Received action '%s'", action)

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

	debug.Log("ScriptsMenu.handleList: Starting")

	proxyAPI := app.GetProxyAPI()
	if proxyAPI == nil {
		debug.Log("ScriptsMenu.handleList: ProxyAPI is nil, showing not connected modal")
		app.ShowModal("Scripts List",
			"Not connected to proxy. Please connect first.",
			[]string{"OK"},
			func(buttonIndex int, buttonLabel string) {
				app.CloseModal()
			})
		return nil
	}

	// Get script list from proxy
	debug.Log("ScriptsMenu.handleList: ProxyAPI available, getting script list")
	scripts, err := proxyAPI.GetScriptList()
	if err != nil {
		debug.Log("ScriptsMenu.handleList: Error getting script list: %v", err)
		app.ShowModal("Scripts List Error",
			fmt.Sprintf("Error getting script list: %v", err),
			[]string{"OK"},
			func(buttonIndex int, buttonLabel string) {
				app.CloseModal()
			})
		return nil
	}

	// Build script list text - let modal auto-size to content
	debug.Log("ScriptsMenu.handleList: Got %d scripts, building list", len(scripts))
	var listText strings.Builder
	if len(scripts) == 0 {
		listText.WriteString("No scripts loaded.\n\n")
	} else {
		listText.WriteString(fmt.Sprintf("Loaded Scripts (%d):\n\n", len(scripts)))

		// Create a reasonably sized table that will fit comfortably
		tableText := s.buildReasonableTable(scripts)
		listText.WriteString(tableText)
	}

	debug.Log("ScriptsMenu.handleList: Showing modal with scripts list")
	debug.Log("ScriptsMenu.handleList: Modal text content: '%s'", listText.String())
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

// buildReasonableTable creates a simple, well-proportioned table
func (s *ScriptsMenu) buildReasonableTable(scripts []api.ScriptInfo) string {
	var table strings.Builder

	// Simple table with reasonable fixed column widths and checkmarks
	table.WriteString("┌────┬─────────────────┬─────────────────┬────────┐\n")
	table.WriteString("│ ID │ Name            │ File            │ Status │\n")
	table.WriteString("├────┼─────────────────┼─────────────────┼────────┤\n")

	for i, script := range scripts {
		// Use checkmark for active, X for inactive (no colors)
		status := "✓"
		if !script.IsActive {
			status = "✗"
		}

		// Truncate to reasonable lengths
		name := script.Name
		if len(name) > 15 {
			name = name[:12] + "..."
		}

		filename := script.Filename
		// Show just filename without path
		if strings.Contains(filename, "/") {
			parts := strings.Split(filename, "/")
			filename = parts[len(parts)-1]
		}
		if len(filename) > 15 {
			filename = filename[:12] + "..."
		}

		table.WriteString(fmt.Sprintf("│ %2d │ %-15s │ %-15s │   %s    │\n",
			i+1, name, filename, status))
	}

	table.WriteString("└────┴─────────────────┴─────────────────┴────────┘\n")
	return table.String()
}

// showInputDialog displays an input dialog (similar to connection dialog pattern)
func (s *ScriptsMenu) showInputDialog(app AppInterface, dialog *components.BurstInputDialog) {
	app.ShowInputDialog("burst-input-dialog", dialog)
}
