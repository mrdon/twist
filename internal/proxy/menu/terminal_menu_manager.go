package menu

import (
	"fmt"
	"strings"
	"sync/atomic"

	"twist/internal/debug"
	"twist/internal/proxy/database"
	"twist/internal/proxy/menu/display"
	"twist/internal/proxy/menu/input"
	"twist/internal/proxy/scripting/types"
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
	
	tmm.inputCollector.RegisterCompletionHandler("SECTOR_DISPLAY", func(menuName, value string) error {
		return tmm.handleSectorDisplayInput(value)
	})
	
	tmm.inputCollector.RegisterCompletionHandler("PORT_DISPLAY", func(menuName, value string) error {
		return tmm.handlePortDisplayInput(value)
	})
	
	tmm.inputCollector.RegisterCompletionHandler("VARIABLE_DUMP", func(menuName, value string) error {
		return tmm.handleVariableDumpInput(value)
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

	// Create and navigate to port submenu
	portMenu := tmm.createTWXPortMenu()
	portMenu.Parent = tmm.currentMenu
	tmm.currentMenu = portMenu
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
	
	// Display sector as last seen (D) - matches TWX
	sectorDisplayItem := NewTerminalMenuItem("Display sector as last seen", "Display sector as last seen", 'D')
	sectorDisplayItem.Handler = tmm.handleSectorDisplay
	dataMenu.AddChild(sectorDisplayItem)
	
	// Show all sectors with foreign fighters (F) - matches TWX
	fightersItem := NewTerminalMenuItem("Show all sectors with foreign fighters", "Show all sectors with foreign fighters", 'F')
	fightersItem.Handler = tmm.handleShowFighters
	dataMenu.AddChild(fightersItem)
	
	// Show all sectors with mines (M) - matches TWX
	minesItem := NewTerminalMenuItem("Show all sectors with mines", "Show all sectors with mines", 'M')
	minesItem.Handler = tmm.handleShowMines
	dataMenu.AddChild(minesItem)
	
	// Show all sectors by density comparison (S) - matches TWX
	densityItem := NewTerminalMenuItem("Show all sectors by density comparison", "Show all sectors by density comparison", 'S')
	densityItem.Handler = tmm.handleShowDensity
	dataMenu.AddChild(densityItem)
	
	// Show all sectors with Anomaly (A) - matches TWX
	anomalyItem := NewTerminalMenuItem("Show all sectors with Anomaly", "Show all sectors with Anomaly", 'A')
	anomalyItem.Handler = tmm.handleShowAnomaly
	dataMenu.AddChild(anomalyItem)
	
	// Show all sectors with traders (R) - matches TWX
	tradersItem := NewTerminalMenuItem("Show all sectors with traders", "Show all sectors with traders", 'R')
	tradersItem.Handler = tmm.handleShowTraders
	dataMenu.AddChild(tradersItem)
	
	// Plot warp course (C) - matches TWX
	plotCourseItem := NewTerminalMenuItem("Plot warp course", "Plot warp course", 'C')
	plotCourseItem.Handler = tmm.handlePlotCourse
	dataMenu.AddChild(plotCourseItem)
	
	return dataMenu
}

func (tmm *TerminalMenuManager) createTWXPortMenu() *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in createTWXPortMenu: %v", r)
		}
	}()

	portMenu := NewTerminalMenuItem("TWX_PORT", "TWX Port Menu", 0)
	
	// Show port details as last seen (D) - matches TWX
	showPortItem := NewTerminalMenuItem("Show port details as last seen", "Show port details as last seen", 'D')
	showPortItem.Handler = tmm.handleShowPort
	portMenu.AddChild(showPortItem)
	
	// Show all class 0/9 port sectors (0) - matches TWX
	specialPortsItem := NewTerminalMenuItem("Show all class 0/9 port sectors", "Show all class 0/9 port sectors", '0')
	specialPortsItem.Handler = tmm.handleShowSpecialPorts
	portMenu.AddChild(specialPortsItem)
	
	// List all ports (L) - matches TWX
	listPortsItem := NewTerminalMenuItem("List all ports", "List all ports", 'L')
	listPortsItem.Handler = tmm.handlePortList
	portMenu.AddChild(listPortsItem)
	
	// List all heavily upgraded ports (U) - matches TWX
	upgradedPortsItem := NewTerminalMenuItem("List all heavily upgraded ports", "List all heavily upgraded ports", 'U')
	upgradedPortsItem.Handler = tmm.handleListUpgradedPorts
	portMenu.AddChild(upgradedPortsItem)
	
	return portMenu
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

	// Follow TWX pattern: ask for variable name pattern first
	tmm.sendOutput("\r\nEnter a full or partial variable name to search for (or blank to list them all):\r\n")
	
	// Start input collection for variable pattern
	tmm.inputCollector.StartCollection("VARIABLE_DUMP", "Variable pattern")
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

	// Check if database is available and open
	if db, ok := dbInterface.(database.Database); ok {
		if !db.GetDatabaseOpen() {
			tmm.sendOutput(display.FormatErrorMessage("Error: Database not open"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		sectorCount := db.GetSectors()
		tmm.sendOutput("\r\nEnter sector number to display (1-" + fmt.Sprintf("%d", sectorCount) + "):\r\n")
		
		// Start input collection for sector number
		tmm.inputCollector.StartCollection("SECTOR_DISPLAY", "Sector number")
		return nil
	} else {
		tmm.sendOutput(display.FormatErrorMessage("Error: Invalid database interface"))
		tmm.displayCurrentMenu()
		return nil
	}
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

	if db, ok := dbInterface.(database.Database); ok {
		if !db.GetDatabaseOpen() {
			tmm.sendOutput(display.FormatErrorMessage("Error: Database not open"))
			tmm.displayCurrentMenu()
			return nil
		}

		var output strings.Builder
		output.WriteString("\r\n")
		output.WriteString("Sector Class Fuel Ore     Organics     Equipment    Updated\r\n")
		output.WriteString("-------------------------------------------------------------\r\n")
		output.WriteString("\r\n")

		sectorCount := db.GetSectors()
		portCount := 0

		// Scan through all sectors looking for ports (like TWX does)
		for i := 1; i <= sectorCount; i++ {
			// Try to load port for this sector
			port, err := db.LoadPort(i)
			if err == nil && port.Name != "" && port.ClassIndex > 0 && port.ClassIndex < 9 {
				// Display port summary (like TWX DisplayPortSummary)
				tmm.displayPortSummary(&output, i, port)
				portCount++
			}
		}

		if portCount == 0 {
			output.WriteString("No ports found in database.\r\n")
		}

		output.WriteString("\r\n")
		tmm.sendOutput(output.String())
	} else {
		tmm.sendOutput(display.FormatErrorMessage("Error: Invalid database interface"))
	}

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

// handleSectorDisplayInput handles input collection for sector display
func (tmm *TerminalMenuManager) handleSectorDisplayInput(sectorStr string) error {
	sectorStr = strings.TrimSpace(sectorStr)
	
	if sectorStr == "" {
		tmm.sendOutput(display.FormatErrorMessage("No sector number provided"))
		tmm.displayCurrentMenu()
		return nil
	}
	
	sectorNum := 0
	if _, err := fmt.Sscanf(sectorStr, "%d", &sectorNum); err != nil {
		tmm.sendOutput(display.FormatErrorMessage("Invalid sector number: " + sectorStr))
		tmm.displayCurrentMenu()
		return nil
	}
	
	dbInterface := tmm.proxyInterface.GetDatabase()
	if dbInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Database not available"))
		tmm.displayCurrentMenu()
		return nil
	}
	
	if db, ok := dbInterface.(database.Database); ok {
		if !db.GetDatabaseOpen() {
			tmm.sendOutput(display.FormatErrorMessage("Database not open"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		sectorCount := db.GetSectors()
		if sectorNum < 1 || sectorNum > sectorCount {
			tmm.sendOutput(display.FormatErrorMessage("That is not a valid sector"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		sectorData, err := db.LoadSector(sectorNum)
		if err != nil {
			tmm.sendOutput(display.FormatErrorMessage("Error loading sector: " + err.Error()))
			tmm.displayCurrentMenu()
			return nil
		}
		
		// Display the sector information in TWX format
		tmm.displaySectorInTWXFormat(sectorData, sectorNum)
	} else {
		tmm.sendOutput(display.FormatErrorMessage("Invalid database interface"))
		tmm.displayCurrentMenu()
	}
	
	return nil
}

// displaySectorInTWXFormat displays a sector in the TWX format
func (tmm *TerminalMenuManager) displaySectorInTWXFormat(sector database.TSector, sectorIndex int) {
	var output strings.Builder
	
	// Last seen date/time (TWX format)
	output.WriteString("\r\nLast seen on " + sector.UpDate.Format("01/02/2006") + " at " + sector.UpDate.Format("15:04:05") + "\r\n\r\n")
	
	// Sector and constellation
	constellation := sector.Constellation
	if constellation == "" || constellation == "uncharted space." {
		constellation = "uncharted space."
	}
	output.WriteString("Sector  : " + fmt.Sprintf("%d", sectorIndex) + " in " + constellation + "\r\n")
	
	// Check if sector has not been recorded (unexplored)
	if sector.Explored == database.EtNo {
		output.WriteString("\r\nThat sector has not been recorded\r\n\r\n")
		tmm.sendOutput(output.String())
		tmm.displayCurrentMenu()
		return
	}
	
	// Density (if available) 
	// Note: TWX shows density with segments, but we'll show the raw number for now
	// TODO: Implement the Segment() function equivalent
	if sector.Density > -1 {
		output.WriteString("Density : " + fmt.Sprintf("%d", sector.Density))
		if sector.Anomaly {
			output.WriteString(" (Anomaly)")
		}
		output.WriteString("\r\n")
	}
	
	// Beacon
	if sector.Beacon != "" {
		output.WriteString("Beacon  : " + sector.Beacon + "\r\n")
	}
	
	// Port information (from separate port table)
	tmm.displayPortInformation(&output, sectorIndex)
	
	// Planets
	if len(sector.Planets) > 0 {
		for i, planet := range sector.Planets {
			if i == 0 {
				output.WriteString("Planets : " + planet.Name + "\r\n")
			} else {
				output.WriteString("          " + planet.Name + "\r\n")
			}
		}
	}
	
	// Traders
	if len(sector.Traders) > 0 {
		for i, trader := range sector.Traders {
			if i == 0 {
				output.WriteString("Traders : " + trader.Name + ", w/ " + fmt.Sprintf("%d", trader.Figs) + " ftrs,\r\n")
			} else {
				output.WriteString("          " + trader.Name + ", w/ " + fmt.Sprintf("%d", trader.Figs) + " ftrs,\r\n")
			}
			output.WriteString("           in " + trader.ShipName + " (" + trader.ShipType + ")\r\n")
		}
	}
	
	// Ships
	if len(sector.Ships) > 0 {
		for i, ship := range sector.Ships {
			if i == 0 {
				output.WriteString("Ships   : " + ship.Name + " [Owned by] " + ship.Owner + ", w/ " + fmt.Sprintf("%d", ship.Figs) + " ftrs,\r\n")
			} else {
				output.WriteString("          " + ship.Name + " [Owned by] " + ship.Owner + ", w/ " + fmt.Sprintf("%d", ship.Figs) + " ftrs,\r\n")
			}
			output.WriteString("           (" + ship.ShipType + ")\r\n")
		}
	}
	
	// Fighters
	if sector.Figs.Quantity > 0 {
		output.WriteString("Fighters: " + fmt.Sprintf("%d", sector.Figs.Quantity) + " (" + sector.Figs.Owner + ") ")
		switch sector.Figs.FigType {
		case database.FtToll:
			output.WriteString("[Toll]")
		case database.FtDefensive:
			output.WriteString("[Defensive]") 
		case database.FtOffensive:
			output.WriteString("[Offensive]")
		default:
			output.WriteString("[Unknown]")
		}
		output.WriteString("\r\n")
	}
	
	// NavHaz
	if sector.NavHaz > 0 {
		output.WriteString("NavHaz  : " + fmt.Sprintf("%d", sector.NavHaz) + "% (Space Debris/Asteroids)\r\n")
	}
	
	// Mines
	if sector.MinesArmid.Quantity > 0 {
		output.WriteString("Mines   : " + fmt.Sprintf("%d", sector.MinesArmid.Quantity) + " (Type 1 Armid) (" + sector.MinesArmid.Owner + ")\r\n")
		if sector.MinesLimpet.Quantity > 0 {
			output.WriteString("        : " + fmt.Sprintf("%d", sector.MinesLimpet.Quantity) + " (Type 2 Limpet) (" + sector.MinesLimpet.Owner + ")\r\n")
		}
	} else if sector.MinesLimpet.Quantity > 0 {
		output.WriteString("Mines   : " + fmt.Sprintf("%d", sector.MinesLimpet.Quantity) + " (Type 2 Limpet) (" + sector.MinesLimpet.Owner + ")\r\n")
	}
	
	// Warps
	output.WriteString("Warps to Sector(s) :  ")
	firstWarp := true
	for i, warp := range sector.Warp {
		if warp > 0 {
			if !firstWarp {
				output.WriteString(" - ")
			}
			output.WriteString(fmt.Sprintf("%d", warp))
			firstWarp = false
		} else if i > 0 { // TWX checks warps 2-6, we check 1-5 (0-based)
			break
		}
	}
	
	// TODO: Add backdoors information when available
	
	output.WriteString("\r\n\r\n\r\n")
	tmm.sendOutput(output.String())
	tmm.displayCurrentMenu()
}

// displayPortInformation displays port information for a sector
func (tmm *TerminalMenuManager) displayPortInformation(output *strings.Builder, sectorIndex int) {
	dbInterface := tmm.proxyInterface.GetDatabase()
	if dbInterface == nil {
		return
	}
	
	if db, ok := dbInterface.(database.Database); ok {
		port, err := db.LoadPort(sectorIndex)
		if err == nil && port.Name != "" && !port.Dead {
			output.WriteString("Ports   : " + port.Name + ", Class " + fmt.Sprintf("%d", port.ClassIndex) + " (")
			
			if port.ClassIndex == 0 || port.ClassIndex == 9 {
				output.WriteString("Special")
			} else {
				// Show buy/sell pattern (B=buy, S=sell)
				// Product indices: 0=Fuel Ore, 1=Organics, 2=Equipment
				if len(port.BuyProduct) >= 3 {
					if port.BuyProduct[0] {
						output.WriteString("B")
					} else {
						output.WriteString("S")
					}
					if port.BuyProduct[1] {
						output.WriteString("B")
					} else {
						output.WriteString("S")
					}
					if port.BuyProduct[2] {
						output.WriteString("B")
					} else {
						output.WriteString("S")
					}
				}
			}
			output.WriteString(")\r\n")
			
			// Construction status
			if port.BuildTime > 0 {
				output.WriteString("           (Under Construction - " + fmt.Sprintf("%d", port.BuildTime) + " days left)\r\n")
			}
		}
	}
}

// handleShowPort handles the "Show port details as last seen" menu option
func (tmm *TerminalMenuManager) handleShowPort(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleShowPort: %v", r)
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

	// Check if database is available and open
	if db, ok := dbInterface.(database.Database); ok {
		if !db.GetDatabaseOpen() {
			tmm.sendOutput(display.FormatErrorMessage("Error: Database not open"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		sectorCount := db.GetSectors()
		tmm.sendOutput("\r\nEnter sector number to show port details (1-" + fmt.Sprintf("%d", sectorCount) + "):\r\n")
		
		// Start input collection for sector number
		tmm.inputCollector.StartCollection("PORT_DISPLAY", "Sector number")
		return nil
	} else {
		tmm.sendOutput(display.FormatErrorMessage("Error: Invalid database interface"))
		tmm.displayCurrentMenu()
		return nil
	}
}

// displayPortSummary displays a port summary line (like TWX DisplayPortSummary)
func (tmm *TerminalMenuManager) displayPortSummary(output *strings.Builder, sectorIndex int, port database.TPort) {
	// Format: sector number, class, buy/sell pattern, product amounts and percentages, update time
	sectorStr := fmt.Sprintf("%6d", sectorIndex)
	classStr := fmt.Sprintf("%5d", port.ClassIndex)
	
	// Build buy/sell pattern (BSS format)
	pattern := ""
	if port.BuyProduct[0] {
		pattern += "B"
	} else {
		pattern += "S"
	}
	if port.BuyProduct[1] {
		pattern += "B"
	} else {
		pattern += "S"
	}
	if port.BuyProduct[2] {
		pattern += "B"
	} else {
		pattern += "S"
	}
	
	// Format product amounts and percentages
	fuelOreStr := fmt.Sprintf("%5d (%3d%%)", port.ProductAmount[0], port.ProductPercent[0])
	organicsStr := fmt.Sprintf("%5d (%3d%%)", port.ProductAmount[1], port.ProductPercent[1])
	equipmentStr := fmt.Sprintf("%5d (%3d%%)", port.ProductAmount[2], port.ProductPercent[2])
	
	// Format update time
	updateStr := port.UpDate.Format("15:04")
	
	output.WriteString(fmt.Sprintf("%s %s %s %s %s %s %s\r\n",
		sectorStr,
		classStr,
		pattern,
		fuelOreStr,
		organicsStr,
		equipmentStr,
		updateStr))
}

// handlePortDisplayInput handles input collection for port display
func (tmm *TerminalMenuManager) handlePortDisplayInput(sectorStr string) error {
	sectorStr = strings.TrimSpace(sectorStr)
	
	if sectorStr == "" {
		tmm.sendOutput(display.FormatErrorMessage("No sector number provided"))
		tmm.displayCurrentMenu()
		return nil
	}
	
	sectorNum := 0
	if _, err := fmt.Sscanf(sectorStr, "%d", &sectorNum); err != nil {
		tmm.sendOutput(display.FormatErrorMessage("Invalid sector number: " + sectorStr))
		tmm.displayCurrentMenu()
		return nil
	}
	
	dbInterface := tmm.proxyInterface.GetDatabase()
	if dbInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Database not available"))
		tmm.displayCurrentMenu()
		return nil
	}
	
	if db, ok := dbInterface.(database.Database); ok {
		if !db.GetDatabaseOpen() {
			tmm.sendOutput(display.FormatErrorMessage("Database not open"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		sectorCount := db.GetSectors()
		if sectorNum < 1 || sectorNum > sectorCount {
			tmm.sendOutput(display.FormatErrorMessage("That is not a valid sector"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		// Load the port data for this sector
		port, err := db.LoadPort(sectorNum)
		if err != nil || port.Name == "" {
			tmm.sendOutput(display.FormatErrorMessage("That port has not been recorded or does not exist"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		// Check if port has been updated (TWX checks S.SPort.Update = 0)
		if port.UpDate.IsZero() {
			tmm.sendOutput(display.FormatErrorMessage("That port has not been recorded"))
			tmm.displayCurrentMenu()
			return nil
		}
		
		// Display the port information in TWX format
		tmm.displayPortInTWXFormat(port, sectorNum)
	} else {
		tmm.sendOutput(display.FormatErrorMessage("Invalid database interface"))
		tmm.displayCurrentMenu()
	}
	
	return nil
}

// displayPortInTWXFormat displays a port in the TWX commerce report format
func (tmm *TerminalMenuManager) displayPortInTWXFormat(port database.TPort, sectorIndex int) {
	var output strings.Builder
	
	// Commerce report header (like TWX DisplayPort)
	output.WriteString("\r\nCommerce report for " + port.Name + " (sector " + fmt.Sprintf("%d", sectorIndex) + ") : ")
	output.WriteString(port.UpDate.Format("15:04:05 01/02/2006") + "\r\n\r\n")
	
	// Product table header
	output.WriteString(" Items     Status  Trading % of max\r\n")
	output.WriteString(" -----     ------  ------- --------\r\n")
	
	// Fuel Ore
	output.WriteString("Fuel Ore   ")
	if port.BuyProduct[0] {
		output.WriteString("Buying   ")
	} else {
		output.WriteString("Selling  ")
	}
	output.WriteString(fmt.Sprintf("%5d    %3d%%\r\n", port.ProductAmount[0], port.ProductPercent[0]))
	
	// Organics
	output.WriteString("Organics   ")
	if port.BuyProduct[1] {
		output.WriteString("Buying   ")
	} else {
		output.WriteString("Selling  ")
	}
	output.WriteString(fmt.Sprintf("%5d    %3d%%\r\n", port.ProductAmount[1], port.ProductPercent[1]))
	
	// Equipment
	output.WriteString("Equipment  ")
	if port.BuyProduct[2] {
		output.WriteString("Buying   ")
	} else {
		output.WriteString("Selling  ")
	}
	output.WriteString(fmt.Sprintf("%5d    %3d%%\r\n", port.ProductAmount[2], port.ProductPercent[2]))
	
	output.WriteString("\r\n\r\n")
	tmm.sendOutput(output.String())
	tmm.displayCurrentMenu()
}

// Placeholder handlers for Data Menu items (to be implemented later)
func (tmm *TerminalMenuManager) handleShowFighters(item *TerminalMenuItem, params []string) error {
	tmm.sendOutput("Show foreign fighters functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleShowMines(item *TerminalMenuItem, params []string) error {
	tmm.sendOutput("Show mines functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleShowDensity(item *TerminalMenuItem, params []string) error {
	tmm.sendOutput("Show density comparison functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleShowAnomaly(item *TerminalMenuItem, params []string) error {
	tmm.sendOutput("Show anomaly functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleShowTraders(item *TerminalMenuItem, params []string) error {
	tmm.sendOutput("Show traders functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handlePlotCourse(item *TerminalMenuItem, params []string) error {
	tmm.sendOutput("Plot course functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}

// Placeholder handlers for Port Menu items (to be implemented later)
func (tmm *TerminalMenuManager) handleShowSpecialPorts(item *TerminalMenuItem, params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in handleShowSpecialPorts: %v", r)
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

	if db, ok := dbInterface.(database.Database); ok {
		if !db.GetDatabaseOpen() {
			tmm.sendOutput(display.FormatErrorMessage("Error: Database not open"))
			tmm.displayCurrentMenu()
			return nil
		}

		tmm.sendOutput("\r\nShowing all sectors with class 0 or 9 ports...\r\n")

		sectorCount := db.GetSectors()
		foundPorts := 0

		// Loop through all sectors looking for class 0 or 9 ports (like TWX)
		for i := 1; i <= sectorCount; i++ {
			// Load port for this sector
			port, err := db.LoadPort(i)
			if err == nil && port.Name != "" && (port.ClassIndex == 0 || port.ClassIndex == 9) {
				// Load the sector and display it (like TWX DisplaySector)
				sector, err := db.LoadSector(i)
				if err == nil {
					tmm.displaySectorInTWXFormat(sector, i)
					foundPorts++
				}
			}
		}

		if foundPorts == 0 {
			tmm.sendOutput("No class 0 or 9 ports found in database.\r\n")
		}

		tmm.sendOutput("\r\nCompleted.\r\n")
	} else {
		tmm.sendOutput(display.FormatErrorMessage("Error: Invalid database interface"))
	}

	tmm.displayCurrentMenu()
	return nil
}

func (tmm *TerminalMenuManager) handleListUpgradedPorts(item *TerminalMenuItem, params []string) error {
	tmm.sendOutput("List upgraded ports functionality not yet implemented.\r\n")
	tmm.displayCurrentMenu()
	return nil
}


// handleVariableDumpInput handles input collection for variable dump
func (tmm *TerminalMenuManager) handleVariableDumpInput(pattern string) error {
	if tmm.proxyInterface == nil {
		tmm.sendOutput(display.FormatErrorMessage("Script manager not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	scriptManager := tmm.proxyInterface.GetScriptManager()
	if scriptManager == nil {
		tmm.sendOutput(display.FormatErrorMessage("Script manager not available"))
		tmm.displayCurrentMenu()
		return nil
	}

	pattern = strings.TrimSpace(pattern)
	
	var output strings.Builder
	output.WriteString("\r\n")
	output.WriteString(display.FormatMenuTitle("Variable Dump"))
	
	if pattern != "" {
		output.WriteString("Searching for variables matching: '" + pattern + "'\r\n")
	} else {
		output.WriteString("Showing all variables:\r\n")
	}
	output.WriteString("\r\n")

	// Get the VM engine and try to access its variables
	engine := scriptManager.GetEngine()
	variableCount := 0
	
	if engine != nil {
		// Try to access the VM's variable manager using our new interface
		if vm, ok := engine.(interface{ 
			GetAllVariables() map[string]*types.Value 
		}); ok {
			variables := vm.GetAllVariables()
			for name, value := range variables {
				if pattern == "" || strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
					// Format the value based on its type
					var valueStr string
					switch value.Type {
					case types.StringType:
						valueStr = value.String
					case types.NumberType:
						valueStr = fmt.Sprintf("%.0f", value.Number)
					case types.ArrayType:
						// Show array size
						valueStr = fmt.Sprintf("[Array with %d elements]", len(value.Array))
					default:
						valueStr = fmt.Sprintf("%v", value)
					}
					output.WriteString(fmt.Sprintf("%-20s = %s\r\n", name, valueStr))
					variableCount++
				}
			}
		} else {
			// Fallback: script engine found but doesn't implement our interface
			output.WriteString("Script engine found, but variable access interface not available.\r\n")
			output.WriteString("Engine type: " + fmt.Sprintf("%T", engine) + "\r\n")
		}
	} else {
		output.WriteString("No script engine available.\r\n")
	}

	if variableCount > 0 {
		output.WriteString(fmt.Sprintf("\r\nFound %d matching variable(s).\r\n", variableCount))
	} else if pattern != "" {
		output.WriteString("\r\nNo variables found matching pattern '" + pattern + "'.\r\n")
	} else {
		output.WriteString("\r\nNo variables found in running scripts.\r\n")
	}

	// Show engine status for debugging
	status := scriptManager.GetStatus()
	output.WriteString("\r\nEngine Status:\r\n")
	for key, value := range status {
		output.WriteString(fmt.Sprintf("- %s: %v\r\n", key, value))
	}
	
	output.WriteString("\r\nVariable Dump Complete.\r\n")
	tmm.sendOutput(output.String())
	tmm.displayCurrentMenu()
	
	return nil
}

// GetMenuManager returns the terminal menu manager for script integration
func (tmm *TerminalMenuManager) GetMenuManager() *TerminalMenuManager {
	return tmm
}