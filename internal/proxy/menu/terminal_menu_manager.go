package menu

import (
	"strings"
	"sync/atomic"

	"twist/internal/debug"
	"twist/internal/proxy/menu/display"
)

type TerminalMenuManager struct {
	currentMenu   *TerminalMenuItem
	activeMenus   map[string]*TerminalMenuItem
	menuKey       rune // default '$'
	inputBuffer   string
	isActive      int32 // atomic bool (0 = false, 1 = true)
	
	// Function to inject data into the stream - will be set by proxy
	// This is the only field that needs protection since it's set by another goroutine
	injectDataFunc atomic.Value // stores func([]byte)
}

func NewTerminalMenuManager() *TerminalMenuManager {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in NewTerminalMenuManager: %v", r)
		}
	}()

	return &TerminalMenuManager{
		activeMenus: make(map[string]*TerminalMenuItem),
		menuKey:     '$',
		isActive:    0, // atomic false
	}
}

func (tmm *TerminalMenuManager) SetInjectDataFunc(injectFunc func([]byte)) {
	tmm.injectDataFunc.Store(injectFunc)
}

func (tmm *TerminalMenuManager) ProcessMenuKey(data string) bool {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ProcessMenuKey: %v", r)
		}
	}()

	if strings.Contains(data, string(tmm.menuKey)) {
		tmm.ActivateMainMenu()
		return true // Consumed the input - don't send to server
	}
	
	return false // Let input pass through to server
}

func (tmm *TerminalMenuManager) MenuText(input string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in MenuText: %v", r)
		}
	}()

	if atomic.LoadInt32(&tmm.isActive) == 0 {
		return nil
	}

	input = strings.TrimSpace(input)
	
	// Handle special cases
	switch input {
	case "?":
		tmm.showHelp()
		return nil
	case "q", "Q":
		tmm.closeCurrentMenu()
		return nil
	case "":
		// Just redisplay current menu
		tmm.displayCurrentMenu()
		return nil
	}

	// Try to find menu item by hotkey
	if len(input) == 1 {
		hotkey := rune(strings.ToUpper(input)[0])
		if tmm.currentMenu != nil && tmm.currentMenu.HasChildren() {
			child := tmm.currentMenu.FindChildByHotkey(hotkey)
			if child != nil {
				return tmm.selectMenuItem(child)
			}
		}
	}

	// Invalid input
	tmm.sendOutput("Invalid selection. Press '?' for help.\r\n")
	tmm.displayCurrentMenu()
	
	return nil
}

func (tmm *TerminalMenuManager) ActivateMainMenu() error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in ActivateMainMenu: %v", r)
		}
	}()

	// Create the proper TWX_MAIN menu structure
	mainMenu := tmm.createTWXMainMenu()
	
	// Store the menu and activate the menu system
	tmm.currentMenu = mainMenu
	tmm.activeMenus[TWX_MAIN] = mainMenu
	atomic.StoreInt32(&tmm.isActive, 1) // atomic true
	
	tmm.displayCurrentMenu()
	
	return nil
}

func (tmm *TerminalMenuManager) selectMenuItem(item *TerminalMenuItem) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in selectMenuItem: %v", r)
		}
	}()

	if item.HasChildren() {
		// Navigate to submenu
		tmm.currentMenu = item
		tmm.displayCurrentMenu()
	} else {
		// Execute item handler
		if item.Handler != nil {
			return item.Execute(item.Parameters)
		} else {
			// Default behavior for items without handlers
			tmm.sendOutput("Menu item not implemented: " + item.Name + "\r\n")
			tmm.displayCurrentMenu()
		}
		
		if item.CloseMenu {
			tmm.closeCurrentMenu()
		}
	}
	
	return nil
}

func (tmm *TerminalMenuManager) displayCurrentMenu() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in displayCurrentMenu: %v", r)
		}
	}()

	if tmm.currentMenu == nil {
		return
	}

	// Build output with ANSI formatting
	var output strings.Builder
	
	// Add menu title with formatting
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle(tmm.currentMenu.Description))
	
	// Add menu options with ANSI formatting
	for _, child := range tmm.currentMenu.Children {
		output.WriteString(display.FormatMenuOption(child.Hotkey, child.Description, true))
		output.WriteString("\r\n")
	}

	// Add standard navigation options
	if tmm.currentMenu.Parent != nil {
		output.WriteString(display.FormatMenuOption('Q', "Back to " + tmm.currentMenu.Parent.Name, true))
	} else {
		output.WriteString(display.FormatMenuOption('Q', "Exit Menu", true))
	}
	output.WriteString("\r\n")
	
	output.WriteString(display.FormatMenuOption('?', "Help", true))
	output.WriteString("\r\n")
	
	// Add input prompt
	output.WriteString(display.FormatInputPrompt("Selection"))
	
	tmm.sendOutput(output.String())
}

func (tmm *TerminalMenuManager) showHelp() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in showHelp: %v", r)
		}
	}()

	helpText := "Use the letter keys to navigate menus.\n" +
		"'Q' - Go back or exit menu\n" +
		"'?' - Show this help\n" +
		"Enter - Refresh current menu"
	
	help := "\r\n" + display.FormatHelpText(helpText) + "\r\n"
	
	tmm.sendOutput(help)
	tmm.displayCurrentMenu()
}

func (tmm *TerminalMenuManager) closeCurrentMenu() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in closeCurrentMenu: %v", r)
		}
	}()

	if tmm.currentMenu != nil && tmm.currentMenu.Parent != nil {
		// Go back to parent menu
		tmm.currentMenu = tmm.currentMenu.Parent
		tmm.displayCurrentMenu()
	} else {
		// Exit menu system
		atomic.StoreInt32(&tmm.isActive, 0) // atomic false
		tmm.currentMenu = nil
		tmm.sendOutput("\r\nExiting menu system.\r\n")
	}
}

func (tmm *TerminalMenuManager) sendOutput(text string) {
	if fn := tmm.injectDataFunc.Load(); fn != nil {
		fn.(func([]byte))([]byte(text))
	}
}

func (tmm *TerminalMenuManager) IsActive() bool {
	return atomic.LoadInt32(&tmm.isActive) == 1
}

func (tmm *TerminalMenuManager) AddCustomMenu(name string, parent *TerminalMenuItem) *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in AddCustomMenu: %v", r)
		}
	}()

	menu := NewTerminalMenuItem(name, name, 0)
	tmm.activeMenus[name] = menu
	
	if parent != nil {
		parent.AddChild(menu)
	}
	
	return menu
}

func (tmm *TerminalMenuManager) GetMenu(name string) *TerminalMenuItem {
	return tmm.activeMenus[name]
}

func (tmm *TerminalMenuManager) RemoveMenu(name string) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in RemoveMenu: %v", r)
		}
	}()

	if menu, exists := tmm.activeMenus[name]; exists {
		if menu.Parent != nil {
			menu.Parent.RemoveChild(menu)
		}
		delete(tmm.activeMenus, name)
	}
}

func (tmm *TerminalMenuManager) SetMenuKey(key rune) {
	tmm.menuKey = key
}

func (tmm *TerminalMenuManager) GetMenuKey() rune {
	return tmm.menuKey
}

func (tmm *TerminalMenuManager) createTWXMainMenu() *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in createTWXMainMenu: %v", r)
		}
	}()

	mainMenu := NewTerminalMenuItem(TWX_MAIN, "TWX Main Menu", 0)
	
	// Add menu items matching TWX structure
	burstItem := NewTerminalMenuItem("Burst Commands", "Burst Commands", 'B')
	burstItem.Handler = tmm.handleBurstCommands
	mainMenu.AddChild(burstItem)
	
	loadScriptItem := NewTerminalMenuItem("Load Script", "Load Script", 'L')
	loadScriptItem.Handler = tmm.handleLoadScript
	mainMenu.AddChild(loadScriptItem)
	
	terminateScriptItem := NewTerminalMenuItem("Terminate Script", "Terminate Script", 'T')
	terminateScriptItem.Handler = tmm.handleTerminateScript
	mainMenu.AddChild(terminateScriptItem)
	
	scriptMenuItem := NewTerminalMenuItem("Script Menu", "Script Menu", 'S')
	scriptMenuItem.Handler = tmm.handleScriptMenu
	mainMenu.AddChild(scriptMenuItem)
	
	dataMenuItem := NewTerminalMenuItem("View Data Menu", "View Data Menu", 'V')
	dataMenuItem.Handler = tmm.handleDataMenu
	mainMenu.AddChild(dataMenuItem)
	
	portMenuItem := NewTerminalMenuItem("Port Menu", "Port Menu", 'P')
	portMenuItem.Handler = tmm.handlePortMenu
	mainMenu.AddChild(portMenuItem)
	
	return mainMenu
}

func (tmm *TerminalMenuManager) handleBurstCommands(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleBurstCommands: %v", r)
		}
	}()

	// TODO: Implement burst commands functionality
	tmm.sendOutput("Burst commands functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleLoadScript(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleLoadScript: %v", r)
		}
	}()

	// TODO: Implement script loading functionality
	tmm.sendOutput("Script loading functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleTerminateScript(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleTerminateScript: %v", r)
		}
	}()

	// TODO: Implement script termination functionality
	tmm.sendOutput("Script termination functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleScriptMenu(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleScriptMenu: %v", r)
		}
	}()

	// This handler will navigate to the script submenu
	tmm.sendOutput("Script menu functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleDataMenu(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleDataMenu: %v", r)
		}
	}()

	// This handler will navigate to the data submenu
	tmm.sendOutput("Data menu functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handlePortMenu(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handlePortMenu: %v", r)
		}
	}()

	// This handler will navigate to the port submenu
	tmm.sendOutput("Port menu functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}