package menu

import (
	"fmt"
	"strings"
	"sync/atomic"

	"twist/internal/debug"
	"twist/internal/proxy/menu/display"
	"twist/internal/proxy/menu/input"
)

type TerminalMenuManager struct {
	currentMenu   *TerminalMenuItem
	activeMenus   map[string]*TerminalMenuItem
	menuKey       rune // default '$'
	isActive      int32 // atomic bool (0 = false, 1 = true)
	
	// Function to inject data into the stream - will be set by proxy
	// This is the only field that needs protection since it's set by another goroutine
	injectDataFunc atomic.Value // stores func([]byte)
	
	// Reference to proxy for accessing ScriptManager and Database
	proxyInterface ProxyInterface
	
	// Script-created menus (separate from built-in menus)
	scriptMenus map[string]*ScriptMenuData
	scriptMenuValues map[string]string // Menu values for script menus
	
	// Separate components for advanced features
	inputCollector *input.InputCollector // Two-stage input collection
	helpSystem     *HelpSystem           // Contextual help system
	
	// Burst command storage (like TWX LastBurst)
	lastBurst string // Last burst command sent
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
	SendInput(input string) // For burst commands and other menu actions
	SendDirectToServer(input string) // Direct server communication bypassing menu system
}

// ScriptManagerInterface defines methods needed for script management
type ScriptManagerInterface interface {
	LoadAndRunScript(filename string) error
	Stop() error
	GetStatus() map[string]interface{}
	GetEngine() interface{}
}

func NewTerminalMenuManager() *TerminalMenuManager {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in NewTerminalMenuManager: %v", r)
		}
	}()

	tmm := &TerminalMenuManager{
		activeMenus:      make(map[string]*TerminalMenuItem),
		scriptMenus:      make(map[string]*ScriptMenuData),
		scriptMenuValues: make(map[string]string),
		menuKey:          '$',
		isActive:         0, // atomic false
		lastBurst:        "",
	}

	// Initialize input collector with output function
	tmm.inputCollector = input.NewInputCollector(tmm.sendOutput)
	
	// Initialize help system with output function
	tmm.helpSystem = NewHelpSystem(tmm.sendOutput)
	
	// Register input completion handlers
	tmm.setupInputHandlers()

	return tmm
}

// setupInputHandlers registers completion handlers for different input operations
func (tmm *TerminalMenuManager) setupInputHandlers() {
	// Register handlers for built-in menu operations
	tmm.inputCollector.RegisterCompletionHandler("SCRIPT_LOAD", func(menuName, value string) error {
		return tmm.handleScriptLoadInput(value)
	})
	
	tmm.inputCollector.RegisterCompletionHandler("SCRIPT_TERMINATE", func(menuName, value string) error {
		return tmm.handleScriptTerminateInput(value)
	})
	
	tmm.inputCollector.RegisterCompletionHandler("BURST_SEND", func(menuName, value string) error {
		return tmm.handleBurstSendInput(value)
	})
	
	tmm.inputCollector.RegisterCompletionHandler("BURST_EDIT", func(menuName, value string) error {
		return tmm.handleBurstEditInput(value)
	})
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
	
	// Debug logging to see what's happening
	debug.Log("MenuText called with input: '%s', inputCollectionMode: %v", input, tmm.inputCollector.IsCollecting())
	
	// Handle two-stage input collection mode using the input collector
	if tmm.inputCollector.IsCollecting() {
		debug.Log("Handling input collection for: '%s'", input)
		return tmm.inputCollector.HandleInput(input)
	}
	
	// Handle special cases
	switch input {
	case "?":
		tmm.helpSystem.ShowContextualHelp(tmm.currentMenu)
		tmm.displayCurrentMenu()
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

	// Create and navigate to burst submenu
	burstMenu := tmm.createTWXBurstMenu()
	burstMenu.Parent = tmm.currentMenu
	tmm.currentMenu = burstMenu
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleLoadScript(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleLoadScript: %v", r)
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

	// For now, prompt for a script filename
	tmm.sendOutput("\r\nEnter script filename to load (e.g., 'myscript.ts'):\r\n")
	tmm.sendOutput("Common scripts: login.ts, autorun.ts, trading.ts\r\n")
	
	// Start input collection for script filename
	tmm.inputCollector.StartCollection("SCRIPT_LOAD", "Script filename")
	return nil
}

func (tmm *TerminalMenuManager) handleTerminateScript(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleTerminateScript: %v", r)
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

	// Show current script status first
	status := scriptManager.GetStatus()
	tmm.sendOutput("\r\nCurrent Script Status:\r\n")
	for key, value := range status {
		tmm.sendOutput(fmt.Sprintf("- %s: %v\r\n", key, value))
	}
	
	tmm.sendOutput("\r\nEnter script name to terminate (or 'ALL' for all scripts):\r\n")
	
	// Start input collection for script termination
	tmm.inputCollector.StartCollection("SCRIPT_TERMINATE", "Script to terminate")
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

func (tmm *TerminalMenuManager) createTWXBurstMenu() *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in createTWXBurstMenu: %v", r)
		}
	}()

	burstMenu := NewTerminalMenuItem("TWX_BURST", "TWX Burst Menu", 0)
	
	// Send burst
	sendBurstItem := NewTerminalMenuItem("Send burst", "Send burst", 'B')
	sendBurstItem.Handler = tmm.handleSendBurst
	burstMenu.AddChild(sendBurstItem)
	
	// Repeat last burst
	repeatBurstItem := NewTerminalMenuItem("Repeat last burst", "Repeat last burst", 'R')
	repeatBurstItem.Handler = tmm.handleRepeatBurst
	burstMenu.AddChild(repeatBurstItem)
	
	// Edit/Send last burst
	editBurstItem := NewTerminalMenuItem("Edit/Send last burst", "Edit/Send last burst", 'E')
	editBurstItem.Handler = tmm.handleEditBurst
	burstMenu.AddChild(editBurstItem)
	
	return burstMenu
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

	scriptManager := tmm.proxyInterface.GetScriptManager()
	if scriptManager == nil {
		tmm.sendOutput(display.FormatErrorMessage("Error: Script manager not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	// Show current script status first
	var output strings.Builder
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle("Script Loading"))
	
	// Show current running scripts
	engine := scriptManager.GetEngine()
	if engine != nil {
		if scriptEngine, ok := engine.(interface{ GetRunningScripts() []interface{} }); ok {
			scripts := scriptEngine.GetRunningScripts()
			output.WriteString("Currently running scripts: " + fmt.Sprintf("%d", len(scripts)) + "\r\n")
		}
	}
	
	output.WriteString("\r\nEnter script filename to load:\r\n")
	output.WriteString("Examples: login.ts, autorun.ts, trading.ts\r\n")
	tmm.sendOutput(output.String())
	
	// Start input collection for script filename
	tmm.inputCollector.StartCollection("SCRIPT_LOAD", "Script filename")
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

	var output strings.Builder
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle("Variable Dump"))
	
	// Get the script engine and dump variables
	engine := scriptManager.GetEngine()
	if engine != nil {
		// Get running scripts and their variables
		if scriptEngine, ok := engine.(interface{ GetRunningScripts() []interface{} }); ok {
			scripts := scriptEngine.GetRunningScripts()
			if len(scripts) == 0 {
				output.WriteString("No running scripts - no variables to dump.\r\n")
			} else {
				output.WriteString("Found " + fmt.Sprintf("%d", len(scripts)) + " running scripts:\r\n")
				
				// For each script, show some basic info
				for i := range scripts {
					output.WriteString(fmt.Sprintf("Script %d: (script details)\r\n", i+1))
				}
				
				output.WriteString("\r\nVariable dump shows TWX script variables.\r\n")
				output.WriteString("Individual script variable inspection would require\r\n")
				output.WriteString("additional VM interface methods.\r\n")
			}
		} else {
			output.WriteString("Unable to access script engine details.\r\n")
		}
	} else {
		output.WriteString("Script engine not available.\r\n")
	}
	
	// Show engine status
	status := scriptManager.GetStatus()
	output.WriteString("\r\nEngine Status:\r\n")
	for key, value := range status {
		output.WriteString(fmt.Sprintf("- %s: %v\r\n", key, value))
	}
	
	tmm.sendOutput(output.String())
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

	// Set the menu value to the reference initially (this is how TWX works)
	tmm.scriptMenuValues[menuName] = scriptMenu.Reference

	// Check if we need to collect input from the user
	if scriptMenu.Prompt != "" {
		// Two-stage input collection: Start collecting input with the custom prompt
		tmm.inputCollector.StartCollection(menuName, scriptMenu.Prompt)
		
		// Show options if available
		if scriptMenu.Options != "" {
			tmm.sendOutput("Options: " + scriptMenu.Options + "\r\n")
		}
		
		// Don't redisplay menu - we're now in input collection mode
		return nil
	} else {
		// No prompt - just activate the menu item
		tmm.sendOutput(display.FormatSuccessMessage("Menu item activated: " + scriptMenu.Description))
		
		// If options are set, display them
		if scriptMenu.Options != "" {
			tmm.sendOutput("Options: " + scriptMenu.Options + "\r\n")
		}

		tmm.displayCurrentMenu()
		return nil
	}
}

// handleScriptLoadInput handles the actual script loading after input collection
func (tmm *TerminalMenuManager) handleScriptLoadInput(filename string) error {
	if strings.TrimSpace(filename) == "" {
		tmm.sendOutput(display.FormatErrorMessage("No filename provided"))
		return nil
	}

	scriptManager := tmm.proxyInterface.GetScriptManager()
	if scriptManager == nil {
		tmm.sendOutput(display.FormatErrorMessage("Script manager not available"))
		return nil
	}

	tmm.sendOutput("Loading script: " + filename + "...\r\n")
	
	err := scriptManager.LoadAndRunScript(filename)
	if err != nil {
		tmm.sendOutput(display.FormatErrorMessage("Failed to load script: " + err.Error()))
	} else {
		tmm.sendOutput(display.FormatSuccessMessage("Script loaded and started: " + filename))
	}
	
	return nil
}

// handleScriptTerminateInput handles the actual script termination after input collection  
func (tmm *TerminalMenuManager) handleScriptTerminateInput(scriptName string) error {
	scriptName = strings.TrimSpace(scriptName)
	
	scriptManager := tmm.proxyInterface.GetScriptManager()
	if scriptManager == nil {
		tmm.sendOutput(display.FormatErrorMessage("Script manager not available"))
		return nil
	}

	if scriptName == "" || scriptName == "ALL" || scriptName == "all" {
		// Terminate all scripts
		tmm.sendOutput("Terminating all running scripts...\r\n")
		err := scriptManager.Stop()
		if err != nil {
			tmm.sendOutput(display.FormatErrorMessage("Failed to terminate scripts: " + err.Error()))
		} else {
			tmm.sendOutput(display.FormatSuccessMessage("All scripts terminated"))
		}
	} else {
		// For individual script termination, we'd need to extend the ScriptManager interface
		// For now, just terminate all
		tmm.sendOutput("Individual script termination not yet supported. Terminating all scripts...\r\n")
		err := scriptManager.Stop()
		if err != nil {
			tmm.sendOutput(display.FormatErrorMessage("Failed to terminate scripts: " + err.Error()))
		} else {
			tmm.sendOutput(display.FormatSuccessMessage("All scripts terminated"))
		}
	}
	
	return nil
}

// Burst Command Handlers

// handleSendBurst handles the "Send burst" menu item
func (tmm *TerminalMenuManager) handleSendBurst(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleSendBurst: %v", r)
		}
	}()

	tmm.sendOutput("\r\n" + display.FormatMenuTitle("Send Burst Command"))
	tmm.sendOutput("Enter burst text to send to server:\r\n")
	tmm.sendOutput("Use '*' character for ENTER (e.g., 'lt1*' lists trader #1)\r\n")
	tmm.sendOutput("Examples: 'bp100*' (buy 100 product), 'sp50*' (sell 50 product), 'tw1234*' (transwarp to sector 1234)\r\n")
	
	// Start input collection for burst command
	tmm.inputCollector.StartCollection("BURST_SEND", "Burst command")
	return nil
}

// handleRepeatBurst handles the "Repeat last burst" menu item
func (tmm *TerminalMenuManager) handleRepeatBurst(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleRepeatBurst: %v", r)
		}
	}()

	if tmm.lastBurst == "" {
		tmm.sendOutput(display.FormatErrorMessage("No previous burst command to repeat"))
		tmm.displayCurrentMenu()
		return nil
	}

	tmm.sendOutput("Repeating last burst: " + tmm.lastBurst + "\r\n")
	
	// Send the burst command (replace * with newline)
	burstText := strings.ReplaceAll(tmm.lastBurst, "*", "\r\n")
	if tmm.proxyInterface != nil {
		// Send through the proxy interface
		// We need to access the proxy's SendInput method
		// For now, we'll use a simple approach by injecting the data
		tmm.sendBurstToServer(burstText)
		tmm.sendOutput(display.FormatSuccessMessage("Burst command repeated and sent"))
		
		// Exit menu system after sending burst command so user input goes to game
		atomic.StoreInt32(&tmm.isActive, 0) // atomic false
		tmm.currentMenu = nil
		tmm.sendOutput("\r\nExiting menu system to allow game interaction.\r\n")
	} else {
		tmm.sendOutput(display.FormatErrorMessage("Unable to send burst - no connection"))
		tmm.displayCurrentMenu()
	}
	
	return nil
}

// handleEditBurst handles the "Edit/Send last burst" menu item
func (tmm *TerminalMenuManager) handleEditBurst(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleEditBurst: %v", r)
		}
	}()

	if tmm.lastBurst == "" {
		tmm.sendOutput(display.FormatErrorMessage("No previous burst command to edit"))
		tmm.displayCurrentMenu()
		return nil
	}

	tmm.sendOutput("\r\n" + display.FormatMenuTitle("Edit Last Burst Command"))
	tmm.sendOutput("Previous burst: " + tmm.lastBurst + "\r\n")
	tmm.sendOutput("Edit and press Enter to send (or cancel with 'q'):\r\n")
	
	// Pre-fill the input collection with the last burst
	tmm.inputCollector.StartCollection("BURST_EDIT", "Edit burst command")
	// Set the current input to the last burst for editing
	return nil
}

// handleBurstSendInput handles input collection for sending a new burst command
func (tmm *TerminalMenuManager) handleBurstSendInput(burstText string) error {
	burstText = strings.TrimSpace(burstText)
	
	if burstText == "" {
		tmm.sendOutput(display.FormatErrorMessage("Empty burst command cancelled"))
		return nil
	}

	// Store as last burst
	tmm.lastBurst = burstText
	
	// Send the burst command (replace * with newline)
	expandedText := strings.ReplaceAll(burstText, "*", "\r\n")
	tmm.sendBurstToServer(expandedText)
	
	tmm.sendOutput(display.FormatSuccessMessage("Burst command sent: " + burstText))
	
	// Exit menu system after sending burst command so user input goes to game
	atomic.StoreInt32(&tmm.isActive, 0) // atomic false
	tmm.currentMenu = nil
	tmm.sendOutput("\r\nExiting menu system to allow game interaction.\r\n")
	return nil
}

// handleBurstEditInput handles input collection for editing and sending a burst command
func (tmm *TerminalMenuManager) handleBurstEditInput(burstText string) error {
	burstText = strings.TrimSpace(burstText)
	
	if burstText == "" {
		tmm.sendOutput(display.FormatErrorMessage("Empty burst command cancelled"))
		return nil
	}

	// Store as last burst
	tmm.lastBurst = burstText
	
	// Send the burst command (replace * with newline)
	expandedText := strings.ReplaceAll(burstText, "*", "\r\n")
	tmm.sendBurstToServer(expandedText)
	
	tmm.sendOutput(display.FormatSuccessMessage("Edited burst command sent: " + burstText))
	
	// Exit menu system after sending burst command so user input goes to game
	atomic.StoreInt32(&tmm.isActive, 0) // atomic false
	tmm.currentMenu = nil
	tmm.sendOutput("\r\nExiting menu system to allow game interaction.\r\n")
	return nil
}

// sendBurstToServer sends burst text to the server through the proxy
func (tmm *TerminalMenuManager) sendBurstToServer(text string) {
	if tmm.proxyInterface == nil {
		debug.Log("Cannot send burst - no proxy interface")
		return
	}
	
	// Split into individual commands (separated by \r\n from * expansion)
	commands := strings.Split(text, "\r\n")
	for _, cmd := range commands {
		if strings.TrimSpace(cmd) != "" {
			// Send each command directly to server bypassing menu system
			tmm.proxyInterface.SendDirectToServer(strings.TrimSpace(cmd) + "\r\n")
			debug.Log("Burst command sent directly to server: %s", strings.TrimSpace(cmd))
		}
	}
}

// GetMenuManager returns the terminal menu manager for script integration
func (tmm *TerminalMenuManager) GetMenuManager() *TerminalMenuManager {
	return tmm
}