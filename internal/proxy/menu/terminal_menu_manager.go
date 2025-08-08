package menu

import (
	"strings"
	"sync/atomic"

	"twist/internal/debug"
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

	// For now, create a simple main menu
	// This will be expanded in Phase 3
	mainMenu := NewTerminalMenuItem("TWX_MAIN", "Main Menu", 0)
	
	// Add placeholder items
	mainMenu.AddChild(NewTerminalMenuItem("Load Script", "Load a script file", 'L'))
	mainMenu.AddChild(NewTerminalMenuItem("Script Menu", "Script management", 'S'))
	mainMenu.AddChild(NewTerminalMenuItem("View Data Menu", "View game data", 'V'))
	mainMenu.AddChild(NewTerminalMenuItem("Port Menu", "Port operations", 'P'))
	
	tmm.currentMenu = mainMenu
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

	output := "\r\n" + tmm.currentMenu.Description + "\r\n"
	output += strings.Repeat("-", len(tmm.currentMenu.Description)) + "\r\n"

	for _, child := range tmm.currentMenu.Children {
		output += "(" + string(child.Hotkey) + ")" + child.Description + "\r\n"
	}

	if tmm.currentMenu.Parent != nil {
		output += "(Q)Back to " + tmm.currentMenu.Parent.Name + "\r\n"
	} else {
		output += "(Q)Exit Menu\r\n"
	}
	
	output += "(?)Help\r\n"
	output += "\r\nSelection: "
	
	tmm.sendOutput(output)
}

func (tmm *TerminalMenuManager) showHelp() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in showHelp: %v", r)
		}
	}()

	help := "\r\n=== Menu Help ===\r\n"
	help += "Use the letter keys to navigate menus.\r\n"
	help += "'Q' - Go back or exit menu\r\n"
	help += "'?' - Show this help\r\n"
	help += "Enter - Refresh current menu\r\n"
	help += "==================\r\n\r\n"
	
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