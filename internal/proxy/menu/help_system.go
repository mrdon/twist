package menu

import (
	"strings"

	"twist/internal/debug"
	"twist/internal/proxy/menu/display"
)

// HelpSystem provides contextual help for the terminal menu system
type HelpSystem struct {
	// Output function to send help text to stream
	sendOutput func(string)
	
	// Help text for different menu contexts
	menuHelp        map[string]string
	generalHelp     string
	inputHelp       string
	navigationHelp  string
}

// NewHelpSystem creates a new help system
func NewHelpSystem(sendOutput func(string)) *HelpSystem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in NewHelpSystem: %v", r)
		}
	}()

	hs := &HelpSystem{
		sendOutput: sendOutput,
		menuHelp:   make(map[string]string),
	}
	
	// Initialize default help texts
	hs.initializeDefaultHelp()
	
	return hs
}

// initializeDefaultHelp sets up the default help texts
func (hs *HelpSystem) initializeDefaultHelp() {
	hs.generalHelp = "Use the letter keys to navigate menus.\n" +
		"'Q' - Go back or exit menu\n" +
		"'?' - Show this help\n" +
		"Enter - Refresh current menu"
	
	hs.inputHelp = "Input Collection Help:\n" +
		"- Type your value and press Enter to submit\n" +
		"- Press Enter alone to submit empty value\n" +
		"- Press '\\' to cancel input collection\n" +
		"- Press '?' for this help"
	
	hs.navigationHelp = "Menu Navigation:\n" +
		"- Use hotkeys (letters) to select menu items\n" +
		"- Press 'Q' to go back to parent menu or exit\n" +
		"- Press '?' to show help for current context\n" +
		"- Press Enter to refresh current menu display"
	
	// Set up specific menu help
	hs.menuHelp[TWX_MAIN] = "TWX Main Menu:\n" +
		"B - Burst Commands (send quick game commands)\n" +
		"L - Load Script (load and run a TWX script)\n" +
		"T - Terminate Script (stop running scripts)\n" +
		"S - Script Menu (advanced script management)\n" +
		"V - View Data Menu (display game database info)\n" +
		"P - Port Menu (port and trading information)"
	
	hs.menuHelp[TWX_SCRIPT] = "TWX Script Menu:\n" +
		"L - Load Script (load and run a new script)\n" +
		"T - Terminate Script (stop all running scripts)\n" +
		"P - Pause Script (pause execution - not implemented)\n" +
		"R - Resume Script (resume paused scripts - not implemented)\n" +
		"D - Debug Script (show script debugging info)\n" +
		"V - Variable Dump (display script variables)"
	
	hs.menuHelp[TWX_DATA] = "TWX Data Menu:\n" +
		"S - Sector Display (show sector information from database)\n" +
		"T - Trader List (show trader information - not implemented)\n" +
		"P - Port List (show port information from database)\n" +
		"R - Route Plot (show trading routes - not implemented)\n" +
		"B - Bubble Info (show space bubble info - not implemented)"
	
	hs.menuHelp["TWX_BURST"] = "TWX Burst Menu:\n" +
		"B - Send burst (send a new burst command to game)\n" +
		"R - Repeat last burst (repeat the previous burst command)\n" +
		"E - Edit/Send last burst (modify and send previous burst)\n" +
		"\nBurst commands use '*' character for ENTER:\n" +
		"Examples: 'lt1*' (list trader 1), 'bp100*' (buy 100 product)"
}

// ShowGeneralHelp displays general menu navigation help
func (hs *HelpSystem) ShowGeneralHelp() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ShowGeneralHelp: %v", r)
		}
	}()

	help := "\r\n" + display.FormatHelpText(hs.generalHelp) + "\r\n"
	hs.sendOutput(help)
}

// ShowInputHelp displays help for input collection mode
func (hs *HelpSystem) ShowInputHelp() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ShowInputHelp: %v", r)
		}
	}()

	help := "\r\n" + display.FormatHelpText(hs.inputHelp) + "\r\n"
	hs.sendOutput(help)
}

// ShowNavigationHelp displays help for menu navigation
func (hs *HelpSystem) ShowNavigationHelp() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ShowNavigationHelp: %v", r)
		}
	}()

	help := "\r\n" + display.FormatHelpText(hs.navigationHelp) + "\r\n"
	hs.sendOutput(help)
}

// ShowContextualHelp displays help specific to the current menu context
func (hs *HelpSystem) ShowContextualHelp(currentMenu *TerminalMenuItem) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ShowContextualHelp: %v", r)
		}
	}()

	if currentMenu == nil {
		hs.ShowGeneralHelp()
		return
	}

	// Try to find specific help for this menu
	if helpText, exists := hs.menuHelp[currentMenu.Name]; exists {
		help := "\r\n" + display.FormatHelpText(helpText) + "\r\n"
		hs.sendOutput(help)
	} else {
		// Generate dynamic help based on menu structure
		hs.showDynamicMenuHelp(currentMenu)
	}
}

// showDynamicMenuHelp generates help text based on the current menu structure
func (hs *HelpSystem) showDynamicMenuHelp(menu *TerminalMenuItem) {
	var helpBuilder strings.Builder
	
	helpBuilder.WriteString("Menu: " + menu.Description + "\n\n")
	helpBuilder.WriteString("Available options:\n")
	
	for _, child := range menu.Children {
		if child.Hotkey != 0 {
			helpBuilder.WriteString(string(child.Hotkey) + " - " + child.Description + "\n")
		}
	}
	
	helpBuilder.WriteString("\nNavigation:\n")
	if menu.Parent != nil {
		helpBuilder.WriteString("Q - Back to " + menu.Parent.Name + "\n")
	} else {
		helpBuilder.WriteString("Q - Exit Menu\n")
	}
	helpBuilder.WriteString("? - Show this help\n")
	helpBuilder.WriteString("Enter - Refresh menu")
	
	help := "\r\n" + display.FormatHelpText(helpBuilder.String()) + "\r\n"
	hs.sendOutput(help)
}

// SetMenuHelp sets custom help text for a specific menu
func (hs *HelpSystem) SetMenuHelp(menuName, helpText string) {
	hs.menuHelp[menuName] = helpText
}

// GetMenuHelp gets help text for a specific menu
func (hs *HelpSystem) GetMenuHelp(menuName string) string {
	if helpText, exists := hs.menuHelp[menuName]; exists {
		return helpText
	}
	return ""
}

// ShowScriptMenuHelp displays help for script-created menus
func (hs *HelpSystem) ShowScriptMenuHelp(menuName, customHelp string, options string) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ShowScriptMenuHelp: %v", r)
		}
	}()

	var helpBuilder strings.Builder
	
	if customHelp != "" {
		helpBuilder.WriteString(customHelp + "\n\n")
	} else {
		helpBuilder.WriteString("Script Menu: " + menuName + "\n\n")
	}
	
	if options != "" {
		helpBuilder.WriteString("Available options: " + options + "\n\n")
	}
	
	helpBuilder.WriteString("Navigation:\n")
	helpBuilder.WriteString("Q - Go back or exit menu\n")
	helpBuilder.WriteString("? - Show this help\n")
	helpBuilder.WriteString("Enter - Refresh menu")
	
	help := "\r\n" + display.FormatHelpText(helpBuilder.String()) + "\r\n"
	hs.sendOutput(help)
}

// ShowBreadcrumbs displays the current menu navigation path
func (hs *HelpSystem) ShowBreadcrumbs(currentMenu *TerminalMenuItem) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ShowBreadcrumbs: %v", r)
		}
	}()

	if currentMenu == nil {
		return
	}

	// Build breadcrumb path
	var path []string
	menu := currentMenu
	for menu != nil {
		path = append([]string{menu.Name}, path...)
		menu = menu.Parent
	}

	if len(path) > 0 {
		breadcrumbText := "Location: " + strings.Join(path, " > ")
		hs.sendOutput("\r\n" + display.FormatMenuTitle(breadcrumbText) + "\r\n")
	}
}

// ShowAllMenus displays a tree view of all available menus (for debugging)
func (hs *HelpSystem) ShowAllMenus(rootMenu *TerminalMenuItem, activeMenus map[string]*TerminalMenuItem) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ShowAllMenus: %v", r)
		}
	}()

	var helpBuilder strings.Builder
	
	helpBuilder.WriteString("Available Menus:\n\n")
	
	if rootMenu != nil {
		hs.buildMenuTree(&helpBuilder, rootMenu, 0)
	}
	
	if len(activeMenus) > 0 {
		helpBuilder.WriteString("\nActive Menus:\n")
		for name := range activeMenus {
			helpBuilder.WriteString("- " + name + "\n")
		}
	}
	
	help := "\r\n" + display.FormatHelpText(helpBuilder.String()) + "\r\n"
	hs.sendOutput(help)
}

// buildMenuTree recursively builds a tree representation of the menu structure
func (hs *HelpSystem) buildMenuTree(builder *strings.Builder, menu *TerminalMenuItem, depth int) {
	indent := strings.Repeat("  ", depth)
	
	if menu.Hotkey != 0 {
		builder.WriteString(indent + "(" + string(menu.Hotkey) + ") " + menu.Description + "\n")
	} else {
		builder.WriteString(indent + menu.Description + "\n")
	}
	
	for _, child := range menu.Children {
		hs.buildMenuTree(builder, child, depth+1)
	}
}