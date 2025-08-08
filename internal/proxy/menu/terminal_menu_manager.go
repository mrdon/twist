package menu

import (
	"fmt"
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
	
	// Reference to proxy for accessing ScriptManager and Database
	proxyInterface ProxyInterface
	
	// Script-created menus (separate from built-in menus)
	scriptMenus map[string]*ScriptMenuData
	scriptMenuValues map[string]string // Menu values for script menus
}

// ScriptMenuData represents a menu created by script commands
type ScriptMenuData struct {
	Name        string
	Description string
	Hotkey      rune
	Reference   string
	Prompt      string
	CloseMenu   bool
	ScriptOwner string
	Parent      string
	Help        string
	Options     string
	MenuItem    *TerminalMenuItem // Associated terminal menu item
}

// ProxyInterface defines methods needed by menu system
type ProxyInterface interface {
	GetScriptManager() ScriptManagerInterface
	GetDatabase() interface{}
}

// ScriptManagerInterface defines methods needed for script management
type ScriptManagerInterface interface {
	LoadAndRunScript(filename string) error
	Stop() error
	GetStatus() map[string]interface{}
	GetEngine() interface{}
}

// Since Engine is from internal/proxy/scripting, we'll use interface{} and type assertion

// Import the actual database interface via interface embedding
// We can't import it directly to avoid cycles, so we'll use interface{} and type assertions

func NewTerminalMenuManager() *TerminalMenuManager {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in NewTerminalMenuManager: %v", r)
		}
	}()

	return &TerminalMenuManager{
		activeMenus:      make(map[string]*TerminalMenuItem),
		scriptMenus:      make(map[string]*ScriptMenuData),
		scriptMenuValues: make(map[string]string),
		menuKey:          '$',
		isActive:         0, // atomic false
	}
}

func (tmm *TerminalMenuManager) SetInjectDataFunc(injectFunc func([]byte)) {
	tmm.injectDataFunc.Store(injectFunc)
}

func (tmm *TerminalMenuManager) SetProxyInterface(proxy ProxyInterface) {
	tmm.proxyInterface = proxy
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

	// Create and navigate to script submenu
	scriptMenu := tmm.createTWXScriptMenu()
	scriptMenu.Parent = tmm.currentMenu
	tmm.currentMenu = scriptMenu
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleDataMenu(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleDataMenu: %v", r)
		}
	}()

	// Create and navigate to data submenu
	dataMenu := tmm.createTWXDataMenu()
	dataMenu.Parent = tmm.currentMenu
	tmm.currentMenu = dataMenu
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

func (tmm *TerminalMenuManager) createTWXScriptMenu() *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in createTWXScriptMenu: %v", r)
		}
	}()

	scriptMenu := NewTerminalMenuItem(TWX_SCRIPT, "TWX Script Menu", 0)
	
	// Load Script
	loadScriptItem := NewTerminalMenuItem("Load Script", "Load Script", 'L')
	loadScriptItem.Handler = tmm.handleScriptLoad
	scriptMenu.AddChild(loadScriptItem)
	
	// Terminate Script
	terminateScriptItem := NewTerminalMenuItem("Terminate Script", "Terminate Script", 'T')
	terminateScriptItem.Handler = tmm.handleScriptTerminate
	scriptMenu.AddChild(terminateScriptItem)
	
	// Pause Script (placeholder - TWX has this but our engine may not support it yet)
	pauseScriptItem := NewTerminalMenuItem("Pause Script", "Pause Script", 'P')
	pauseScriptItem.Handler = tmm.handleScriptPause
	scriptMenu.AddChild(pauseScriptItem)
	
	// Resume Script (placeholder)
	resumeScriptItem := NewTerminalMenuItem("Resume Script", "Resume Script", 'R')
	resumeScriptItem.Handler = tmm.handleScriptResume
	scriptMenu.AddChild(resumeScriptItem)
	
	// Debug Script
	debugScriptItem := NewTerminalMenuItem("Debug Script", "Debug Script", 'D')
	debugScriptItem.Handler = tmm.handleScriptDebug
	scriptMenu.AddChild(debugScriptItem)
	
	// Variable Dump
	variableDumpItem := NewTerminalMenuItem("Variable Dump", "Variable Dump", 'V')
	variableDumpItem.Handler = tmm.handleVariableDump
	scriptMenu.AddChild(variableDumpItem)
	
	return scriptMenu
}

func (tmm *TerminalMenuManager) createTWXDataMenu() *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in createTWXDataMenu: %v", r)
		}
	}()

	dataMenu := NewTerminalMenuItem(TWX_DATA, "TWX Data Menu", 0)
	
	// Sector Display
	sectorDisplayItem := NewTerminalMenuItem("Sector Display", "Sector Display", 'S')
	sectorDisplayItem.Handler = tmm.handleSectorDisplay
	dataMenu.AddChild(sectorDisplayItem)
	
	// Trader List
	traderListItem := NewTerminalMenuItem("Trader List", "Trader List", 'T')
	traderListItem.Handler = tmm.handleTraderList
	dataMenu.AddChild(traderListItem)
	
	// Port List
	portListItem := NewTerminalMenuItem("Port List", "Port List", 'P')
	portListItem.Handler = tmm.handlePortList
	dataMenu.AddChild(portListItem)
	
	// Route Plot
	routePlotItem := NewTerminalMenuItem("Route Plot", "Route Plot", 'R')
	routePlotItem.Handler = tmm.handleRoutePlot
	dataMenu.AddChild(routePlotItem)
	
	// Bubble Info
	bubbleInfoItem := NewTerminalMenuItem("Bubble Info", "Bubble Info", 'B')
	bubbleInfoItem.Handler = tmm.handleBubbleInfo
	dataMenu.AddChild(bubbleInfoItem)
	
	return dataMenu
}

// Script Menu Handlers
func (tmm *TerminalMenuManager) handleScriptLoad(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleScriptLoad: %v", r)
		}
	}()

	if tmm.proxyInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: No proxy interface available"))
		tmm.displayCurrentMenu()
		return nil
	}

	// TODO: Implement file picker or prompt for filename
	// For now, show current status
	scriptManager := tmm.proxyInterface.GetScriptManager()
	if scriptManager == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: Script manager not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	var output strings.Builder
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle("Script Loading"))
	output.WriteString("Current running scripts:\r\n")
	
	// Use type assertion to get the engine and its running scripts
	engine := scriptManager.GetEngine()
	if engine != nil {
		// Type assert to get the actual engine interface with GetRunningScripts method
		// This is safe since we know the engine type from our architecture
		if scriptEngine, ok := engine.(interface{ GetRunningScripts() []interface{} }); ok {
			scripts := scriptEngine.GetRunningScripts()
			if len(scripts) == 0 {
				output.WriteString("No scripts currently running.\r\n")
			} else {
				output.WriteString("Found " + fmt.Sprintf("%d", len(scripts)) + " running scripts.\r\n")
			}
		} else {
			output.WriteString("Unable to access script information.\r\n")
		}
	} else {
		output.WriteString("Script engine not available.\r\n")
	}
	
	output.WriteString("\r\nScript loading functionality will be implemented in Phase 5.\r\n")
	tmm.sendOutput(output.String())
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleScriptTerminate(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleScriptTerminate: %v", r)
		}
	}()

	if tmm.proxyInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: No proxy interface available"))
		tmm.displayCurrentMenu()
		return nil
	}

	scriptManager := tmm.proxyInterface.GetScriptManager()
	if scriptManager == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: Script manager not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	// Terminate all running scripts
	err := scriptManager.Stop()
	if err != nil {
		tmm.sendOutput(display.FormatErrorMessage("Error stopping scripts: " + err.Error()))
	} else {
		tmm.sendOutput(display.FormatSuccessMessage("All scripts terminated successfully"))
	}
	
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleScriptPause(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleScriptPause: %v", r)
		}
	}()

	tmm.sendOutput("Script pause functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleScriptResume(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleScriptResume: %v", r)
		}
	}()

	tmm.sendOutput("Script resume functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleScriptDebug(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleScriptDebug: %v", r)
		}
	}()

	if tmm.proxyInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: No proxy interface available"))
		tmm.displayCurrentMenu()
		return nil
	}

	scriptManager := tmm.proxyInterface.GetScriptManager()
	if scriptManager == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: Script manager not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	status := scriptManager.GetStatus()
	var output strings.Builder
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle("Script Debug Information"))
	
	for key, value := range status {
		output.WriteString(key + ": " + fmt.Sprintf("%v", value) + "\r\n")
	}
	
	tmm.sendOutput(output.String())
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleVariableDump(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleVariableDump: %v", r)
		}
	}()

	tmm.sendOutput("Variable dump functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

// Data Menu Handlers
func (tmm *TerminalMenuManager) handleSectorDisplay(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleSectorDisplay: %v", r)
		}
	}()

	if tmm.proxyInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: No proxy interface available"))
		tmm.displayCurrentMenu()
		return nil
	}

	dbInterface := tmm.proxyInterface.GetDatabase()
	if dbInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: Database not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	var output strings.Builder
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle("Sector Information"))

	// Use type assertions to access database methods
	if db, ok := dbInterface.(interface {
		GetDatabaseOpen() bool
		GetSectors() int
		LoadSector(int) (interface{}, error)
	}); ok {
		if !db.GetDatabaseOpen() {
			output.WriteString("Error: Database not open\r\n")
		} else {
			sectorCount := db.GetSectors()
			output.WriteString("Total sectors in database: " + fmt.Sprintf("%d", sectorCount) + "\r\n")
			
			if sectorCount > 0 {
				// Show first few sectors as sample
				output.WriteString("\r\nSample sectors:\r\n")
				for i := 1; i <= min(5, sectorCount); i++ {
					sectorData, err := db.LoadSector(i)
					if err == nil && sectorData != nil {
						// Type assert the sector data to get constellation
						// Since we know this returns TSector struct, we can try to access it directly
						output.WriteString("- Sector " + fmt.Sprintf("%d", i) + ": (sector data loaded)\r\n")
					} else {
						output.WriteString("- Sector " + fmt.Sprintf("%d", i) + ": (no data)\r\n")
					}
				}
			}
		}
	} else {
		output.WriteString("Error: Invalid database interface\r\n")
	}
	
	tmm.sendOutput(output.String())
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleTraderList(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleTraderList: %v", r)
		}
	}()

	tmm.sendOutput("Trader list functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handlePortList(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handlePortList: %v", r)
		}
	}()

	if tmm.proxyInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: No proxy interface available"))
		tmm.displayCurrentMenu()
		return nil
	}

	dbInterface := tmm.proxyInterface.GetDatabase()
	if dbInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: Database not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	var output strings.Builder
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle("Port Information"))
	
	// Use type assertions to access database methods
	if db, ok := dbInterface.(interface {
		GetDatabaseOpen() bool
		FindPortsByClass(int) ([]interface{}, error)
	}); ok {
		if !db.GetDatabaseOpen() {
			output.WriteString("Error: Database not open\r\n")
		} else {
			// Try to find ports by class (class 0 = special ports)
			portsData, err := db.FindPortsByClass(0)
			if err != nil {
				output.WriteString("Error querying ports: " + err.Error() + "\r\n")
			} else {
				output.WriteString("Special ports (Class 0): " + fmt.Sprintf("%d", len(portsData)) + "\r\n")
				for i := range portsData {
					if i >= 10 { // Limit display
						output.WriteString("... and " + fmt.Sprintf("%d", len(portsData)-i) + " more\r\n")
						break
					}
					// Since we know this returns TPort struct, show basic info
					output.WriteString("- Port " + fmt.Sprintf("%d", i+1) + ": (port data available)\r\n")
				}
			}
		}
	} else {
		output.WriteString("Error: Invalid database interface\r\n")
	}
	
	tmm.sendOutput(output.String())
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleRoutePlot(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleRoutePlot: %v", r)
		}
	}()

	tmm.sendOutput("Route plot functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleBubbleInfo(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleBubbleInfo: %v", r)
		}
	}()

	tmm.sendOutput("Bubble info functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

// Helper function for minimum
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Script Menu Management Methods

// AddScriptMenu adds a script-created menu
func (tmm *TerminalMenuManager) AddScriptMenu(name, description, parent, reference, prompt, scriptOwner string, hotkey rune, closeMenu bool) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in AddScriptMenu: %v", r)
		}
	}()

	// Create script menu data
	scriptMenu := &ScriptMenuData{
		Name:        name,
		Description: description,
		Hotkey:      hotkey,
		Reference:   reference,
		Prompt:      prompt,
		CloseMenu:   closeMenu,
		ScriptOwner: scriptOwner,
		Parent:      parent,
		Help:        "",
		Options:     "",
	}

	// Create terminal menu item with script handler
	menuItem := NewTerminalMenuItem(name, description, hotkey)
	menuItem.Handler = func(item *TerminalMenuItem, params []string) error {
		return tmm.handleScriptMenuItem(item, params, name)
	}
	menuItem.CloseMenu = closeMenu
	scriptMenu.MenuItem = menuItem

	// Store the script menu
	tmm.scriptMenus[name] = scriptMenu

	// Add to parent menu if specified
	if parent != "" && parent != "MAIN" {
		if parentMenu, exists := tmm.activeMenus[parent]; exists {
			parentMenu.AddChild(menuItem)
		} else if parentScriptMenu, exists := tmm.scriptMenus[parent]; exists {
			parentScriptMenu.MenuItem.AddChild(menuItem)
		}
	} else {
		// Add to main menu
		if mainMenu, exists := tmm.activeMenus[TWX_MAIN]; exists {
			mainMenu.AddChild(menuItem)
		}
	}

	// Initialize empty value
	tmm.scriptMenuValues[name] = ""

	return nil
}

// OpenScriptMenu opens a script-created menu
func (tmm *TerminalMenuManager) OpenScriptMenu(menuName string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in OpenScriptMenu: %v", r)
		}
	}()

	scriptMenu, exists := tmm.scriptMenus[menuName]
	if !exists {
		return fmt.Errorf("script menu '%s' not found", menuName)
	}

	// Set the current menu to the script menu
	tmm.currentMenu = scriptMenu.MenuItem
	atomic.StoreInt32(&tmm.isActive, 1)
	tmm.displayCurrentMenu()

	return nil
}

// CloseScriptMenu closes a script-created menu
func (tmm *TerminalMenuManager) CloseScriptMenu(menuName string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in CloseScriptMenu: %v", r)
		}
	}()

	scriptMenu, exists := tmm.scriptMenus[menuName]
	if !exists {
		return fmt.Errorf("script menu '%s' not found", menuName)
	}

	// If this is the current menu, go back to parent or exit
	if tmm.currentMenu == scriptMenu.MenuItem {
		if scriptMenu.MenuItem.Parent != nil {
			tmm.currentMenu = scriptMenu.MenuItem.Parent
			tmm.displayCurrentMenu()
		} else {
			atomic.StoreInt32(&tmm.isActive, 0)
			tmm.currentMenu = nil
			tmm.sendOutput("\r\nExiting menu system.\r\n")
		}
	}

	return nil
}

// GetScriptMenuValue gets the value of a script-created menu
func (tmm *TerminalMenuManager) GetScriptMenuValue(menuName string) (string, error) {
	if value, exists := tmm.scriptMenuValues[menuName]; exists {
		return value, nil
	}
	return "", fmt.Errorf("script menu '%s' not found", menuName)
}

// SetScriptMenuValue sets the value of a script-created menu
func (tmm *TerminalMenuManager) SetScriptMenuValue(menuName, value string) error {
	if _, exists := tmm.scriptMenus[menuName]; exists {
		tmm.scriptMenuValues[menuName] = value
		return nil
	}
	return fmt.Errorf("script menu '%s' not found", menuName)
}

// SetScriptMenuHelp sets the help text for a script-created menu
func (tmm *TerminalMenuManager) SetScriptMenuHelp(menuName, helpText string) error {
	if scriptMenu, exists := tmm.scriptMenus[menuName]; exists {
		scriptMenu.Help = helpText
		return nil
	}
	return fmt.Errorf("script menu '%s' not found", menuName)
}

// SetScriptMenuOptions sets the options for a script-created menu
func (tmm *TerminalMenuManager) SetScriptMenuOptions(menuName, options string) error {
	if scriptMenu, exists := tmm.scriptMenus[menuName]; exists {
		scriptMenu.Options = options
		return nil
	}
	return fmt.Errorf("script menu '%s' not found", menuName)
}

// RemoveScriptMenusByOwner removes all menus owned by a specific script
func (tmm *TerminalMenuManager) RemoveScriptMenusByOwner(scriptID string) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in RemoveScriptMenusByOwner: %v", r)
		}
	}()

	menusToRemove := make([]string, 0)
	for name, menu := range tmm.scriptMenus {
		if menu.ScriptOwner == scriptID {
			menusToRemove = append(menusToRemove, name)
		}
	}

	for _, menuName := range menusToRemove {
		tmm.removeScriptMenu(menuName)
	}
}

// removeScriptMenu removes a script menu completely
func (tmm *TerminalMenuManager) removeScriptMenu(menuName string) {
	if scriptMenu, exists := tmm.scriptMenus[menuName]; exists {
		// Remove from parent if it has one
		if scriptMenu.MenuItem.Parent != nil {
			scriptMenu.MenuItem.Parent.RemoveChild(scriptMenu.MenuItem)
		}

		// Remove from our tracking
		delete(tmm.scriptMenus, menuName)
		delete(tmm.scriptMenuValues, menuName)

		// If this was the current menu, go back to parent or exit
		if tmm.currentMenu == scriptMenu.MenuItem {
			if scriptMenu.MenuItem.Parent != nil {
				tmm.currentMenu = scriptMenu.MenuItem.Parent
				tmm.displayCurrentMenu()
			} else {
				atomic.StoreInt32(&tmm.isActive, 0)
				tmm.currentMenu = nil
			}
		}
	}
}

// handleScriptMenuItem handles execution of script-created menu items
func (tmm *TerminalMenuManager) handleScriptMenuItem(item *TerminalMenuItem, params []string, menuName string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleScriptMenuItem: %v", r)
		}
	}()

	scriptMenu, exists := tmm.scriptMenus[menuName]
	if !exists {
		tmm.sendOutput(display.FormatErrorMessage("Script menu not found: " + menuName))
		tmm.displayCurrentMenu()
		return nil
	}

	// Set the menu value to the reference (this is how TWX works)
	tmm.scriptMenuValues[menuName] = scriptMenu.Reference

	// Show menu-specific prompt if available
	if scriptMenu.Prompt != "" {
		tmm.sendOutput(display.FormatInputPrompt(scriptMenu.Prompt))
	} else {
		tmm.sendOutput(display.FormatSuccessMessage("Menu item activated: " + scriptMenu.Description))
	}

	// If options are set, display them
	if scriptMenu.Options != "" {
		tmm.sendOutput("Options: " + scriptMenu.Options + "\r\n")
	}

	tmm.displayCurrentMenu()
	return nil
}

// GetMenuManager returns the terminal menu manager for script integration
func (tmm *TerminalMenuManager) GetMenuManager() *TerminalMenuManager {
	return tmm
}