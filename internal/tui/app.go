package tui

import (
	"twist/internal/ansi"
	"twist/internal/proxy"
	"twist/internal/terminal"
	"twist/internal/tui/components"
	"twist/internal/tui/handlers"
	"twist/internal/theme"
	twistComponents "twist/internal/components"
	"twist/internal/tui/api"
	proxyapi "twist/internal/proxy/api"
	"twist/internal/debug"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TwistApp represents the main tview application - refactored version
type TwistApp struct {
	app    *tview.Application
	proxy  *proxy.Proxy

	// API layer (alongside existing proxy for now)
	proxyClient *api.ProxyClient
	tuiAPI      proxyapi.TuiAPI

	// Core components
	terminal     *terminal.Terminal
	pages        *tview.Pages
	mainGrid     *tview.Grid

	// UI Components
	menuComponent     *components.MenuComponent
	terminalComponent *components.TerminalComponent
	panelComponent    *components.PanelComponent
	statusComponent   *components.StatusComponent

	// Input handling
	inputHandler        *handlers.InputHandler
	globalShortcuts     *twistComponents.GlobalShortcutManager

	// State
	connected     bool
	serverAddress string
	modalVisible  bool

	// Update channel
	terminalUpdateChan chan struct{}
}

// NewApplication creates and configures the tview application
func NewApplication() *TwistApp {
	debug.Log("=== NewApplication called ===")

	// Create the simplified color converter
	ansiConverter := ansi.NewColorConverter()
	
	// Initialize terminal buffer with ANSI converter
	term := terminal.NewTerminalWithConverter(80, 50, ansiConverter)

	// Initialize proxy with the terminal as a writer
	debug.Log("Creating legacy proxy instance (should not be used for connections)")
	proxyInstance := proxy.New(term)
	debug.Log("Legacy proxy created, ensuring it's not connected")

	// Create the main application
	app := tview.NewApplication()

	// Create UI components
	menuComp := components.NewMenuComponent()
	terminalComp := components.NewTerminalComponent(term)
	panelComp := components.NewPanelComponent()
	statusComp := components.NewStatusComponent()

	// Create input handler
	inputHandler := handlers.NewInputHandler(app)

	twistApp := &TwistApp{
		app:                app,
		proxy:              proxyInstance,
		terminal:           term,
		connected:          false,
		serverAddress:      "twgs.geekm0nkey.com:23",
		terminalUpdateChan: make(chan struct{}, 100),
		menuComponent:      menuComp,
		terminalComponent:  terminalComp,
		panelComponent:     panelComp,
		statusComponent:    statusComp,
		inputHandler:       inputHandler,
		globalShortcuts:    twistComponents.NewGlobalShortcutManager(),
	}

	// Create API layer - proxy instances created per connection via static Connect()
	twistApp.proxyClient = api.NewProxyClient()
	twistApp.tuiAPI = api.NewTuiAPI(twistApp)

	// Set up terminal update callback
	term.SetUpdateCallback(func() {
		select {
		case twistApp.terminalUpdateChan <- struct{}{}:
		default:
		}

		// Trigger UI update
		app.QueueUpdateDraw(func() {
			twistApp.updateTerminalView()
		})
	})

	// Wire script manager to status component
	statusComp.SetScriptManager(proxyInstance.GetScriptManager())

	twistApp.setupUI()
	twistApp.setupInputHandling()
	twistApp.registerMenuShortcuts() // Register all menu shortcuts globally
	twistApp.startUpdateWorker()

	return twistApp
}

// setupUI configures the user interface layout
func (ta *TwistApp) setupUI() {
	// Create main grid layout: 3 columns, 3 rows (menu, main content, status)
	ta.mainGrid = tview.NewGrid().
		SetRows(1, 0, 1).
		SetColumns(20, 0, 20).
		SetBorders(false)
	
	// Set main grid background to pure black
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	ta.mainGrid.SetBackgroundColor(defaultColors.Background)

	// Add menu bar to top row, spanning all columns
	ta.mainGrid.AddItem(ta.menuComponent.GetView(), 0, 0, 1, 3, 0, 0, false)

	// Add panels and terminal to main area
	ta.mainGrid.AddItem(ta.panelComponent.GetLeftWrapper(), 1, 0, 1, 1, 0, 0, false)
	ta.mainGrid.AddItem(ta.terminalComponent.GetWrapper(), 1, 1, 1, 1, 0, 0, true)
	ta.mainGrid.AddItem(ta.panelComponent.GetRightWrapper(), 1, 2, 1, 1, 0, 0, false)

	// Add status bar to bottom row, spanning all columns
	ta.mainGrid.AddItem(ta.statusComponent.GetWrapper(), 2, 0, 1, 3, 0, 0, false)

	// Create pages container
	ta.pages = tview.NewPages()
	ta.pages.SetBackgroundColor(defaultColors.Background) // Set pages background to black too
	ta.pages.AddPage("main", ta.mainGrid, true, true)

	ta.app.SetRoot(ta.pages, true)
	
	// Always keep terminal focused and in terminal input mode
	ta.app.SetFocus(ta.terminalComponent.GetWrapper())
	ta.inputHandler.SetInputMode(handlers.InputModeTerminal)
}

// setupInputHandling configures input event handling
func (ta *TwistApp) setupInputHandling() {
	// Set up input handler callbacks
	ta.inputHandler.SetCallbacks(
		ta.connect,           // onConnect
		ta.disconnect,        // onDisconnect
		ta.exit,             // onExit
		ta.showMenuModal,    // onShowModal
		ta.closeModal,       // onCloseModal
		ta.sendCommand,      // onSendCommand
	)
	
	// Set up dropdown callback
	ta.inputHandler.SetDropdownCallback(ta.showDropdownMenu)
	
	// Set up connection dialog callback
	ta.inputHandler.SetConnectionDialogCallback(ta.showConnectionDialog)

	// Set up global input capture
	ta.app.SetInputCapture(ta.handleGlobalKeys)
}

// registerMenuShortcuts registers all menu item shortcuts globally at startup
func (ta *TwistApp) registerMenuShortcuts() {
	
	// Register Session menu shortcuts
	sessionItems := []twistComponents.MenuItem{
		{Label: "Connect", Shortcut: ""},
		{Label: "Recent Connections", Shortcut: ""},
		{Label: "Disconnect", Shortcut: ""},
		{Label: "Save Session", Shortcut: ""},
		{Label: "Quit", Shortcut: "Alt+Q"},
	}
	
	for _, item := range sessionItems {
		if item.Shortcut != "" {
			label := item.Label   // Capture for closure
			shortcut := item.Shortcut
			ta.globalShortcuts.RegisterShortcut(shortcut, func() {
				// Handle the menu item action
				switch label {
				case "Quit":
					ta.exit()
				// Add other menu item actions as needed
				}
			})
		}
	}
	
	// TODO: Register shortcuts for other menus (Edit, View, Terminal, Help) as they get shortcuts
}

// Run starts the TUI application
func (ta *TwistApp) Run() error {
	return ta.app.Run()
}

// connect establishes connection to the game server
func (ta *TwistApp) connect(address string) {
	defer debug.LogFunction("TwistApp.connect")()
	debug.Log("Connect requested to address: %s", address)
	
	// Use API layer exclusively - connection state updated via callbacks
	if err := ta.proxyClient.Connect(address, ta.tuiAPI); err != nil {
		debug.LogError(err, "proxyClient.Connect immediate failure")
		// Handle immediate validation errors
		ta.connected = false
		ta.serverAddress = ""
		ta.menuComponent.SetDisconnectedMenu()
		return
	}
	debug.Log("ProxyClient.Connect returned successfully, waiting for callbacks")
	// Connection state will be updated via OnConnected/OnConnectionError callbacks
}

// disconnect closes the connection to the game server
func (ta *TwistApp) disconnect() {
	// Use API layer exclusively
	ta.proxyClient.Disconnect()
	// Disconnection state will be updated via OnDisconnected callback
}

// exit shuts down the application
func (ta *TwistApp) exit() {
	ta.disconnect()
	ta.app.Stop()
}

// sendCommand sends a command to the game server
func (ta *TwistApp) sendCommand(command string) {
	if ta.proxyClient.IsConnected() {
		ta.proxyClient.SendData([]byte(command))
	}
}

// updateTerminalView updates the terminal display
func (ta *TwistApp) updateTerminalView() {
	debug.Log("updateTerminalView called")
	if ta.terminalComponent == nil {
		debug.Log("ERROR: terminalComponent is nil")
		return
	}
	debug.Log("Calling terminalComponent.UpdateContent()")
	ta.terminalComponent.UpdateContent()
	debug.Log("terminalComponent.UpdateContent() completed")
}

// TuiAPI handler methods - called by API layer
func (ta *TwistApp) HandleConnectionEstablished(info proxyapi.ConnectionInfo) {
	debug.Log("HandleConnectionEstablished called with: %+v", info)
	ta.app.QueueUpdateDraw(func() {
		debug.LogState("TUI", "connection established", info.Address)
		ta.connected = true
		ta.serverAddress = info.Address
		ta.menuComponent.SetConnectedMenu()
		
		// Ensure modal is closed if it's still open
		if ta.modalVisible {
			ta.closeModal()
		}
		
		debug.Log("UI updated to connected state")
	})
}

func (ta *TwistApp) HandleDisconnection(reason string) {
	debug.Log("HandleDisconnection called with reason: %s", reason)
	ta.app.QueueUpdateDraw(func() {
		debug.LogState("TUI", "disconnection", reason)
		ta.connected = false
		ta.serverAddress = ""
		ta.menuComponent.SetDisconnectedMenu()
		debug.Log("UI updated to disconnected state")
	})
}

func (ta *TwistApp) HandleConnectionError(err error) {
	debug.LogError(err, "HandleConnectionError")
	ta.app.QueueUpdateDraw(func() {
		debug.LogState("TUI", "connection error", err.Error())
		ta.connected = false
		ta.serverAddress = ""
		ta.menuComponent.SetDisconnectedMenu()
		
		// Ensure modal is closed if it's still open
		if ta.modalVisible {
			ta.closeModal()
		}
		
		// TODO: Show error modal: ta.showErrorModal(err)
		debug.Log("UI updated to disconnected state due to error")
	})
}

func (ta *TwistApp) HandleTerminalData(data []byte) {
	// Log raw chunk data for debugging ANSI and boundary issues
	ta.logChunkDebug(data)
	
	debug.Log("HandleTerminalData called with %d bytes: %q", len(data), string(data))
	
	// Try direct processing first to see if QueueUpdateDraw is the issue
	debug.Log("Processing terminal data directly (bypassing QueueUpdateDraw temporarily)")
	
	// Add error recovery to catch any panics in terminal processing
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in HandleTerminalData: %v", r)
		}
	}()
	
	debug.Log("Writing to terminal buffer directly")
	ta.terminal.Write(data)
	debug.Log("Terminal.Write completed")
	
	// Now try the QueueUpdateDraw for UI refresh
	ta.app.QueueUpdateDraw(func() {
		debug.Log("In QueueUpdateDraw callback for UI refresh")
		ta.updateTerminalView()
		debug.Log("updateTerminalView() completed")
	})
	
	debug.Log("HandleTerminalData completed")
}

func (ta *TwistApp) logChunkDebug(data []byte) {
	// Count visible characters (excluding ANSI sequences) in the entire chunk
	visibleChars := 0
	inEscape := false
	for _, b := range data {
		if b == 0x1B {
			inEscape = true
		} else if inEscape && (b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z') {
			inEscape = false
		} else if !inEscape {
			visibleChars++
		}
	}
	
	debug.Log("CHUNK [len=%d, visible=%d]: %q", len(data), visibleChars, string(data))
	debug.Log("CHUNK hex: %x", data)
	
	if visibleChars > 80 {
		debug.Log("*** WARNING: Chunk has %d visible chars > 80 ***", visibleChars)
	}
	
	// Check for partial ANSI sequences at chunk boundaries
	if len(data) > 0 {
		if data[0] == 0x1B {
			debug.Log("*** CHUNK STARTS WITH ESC - possible continuation ***")
		}
		if data[len(data)-1] == 0x1B {
			debug.Log("*** CHUNK ENDS WITH ESC - likely incomplete sequence ***")
		}
		
		// Look for incomplete ANSI sequences at the end
		for i := len(data) - 1; i >= 0 && i >= len(data)-10; i-- {
			if data[i] == 0x1B && i+1 < len(data) && data[i+1] == '[' {
				// Check if this escape sequence is complete
				complete := false
				for j := i + 2; j < len(data); j++ {
					if (data[j] >= 'a' && data[j] <= 'z') || (data[j] >= 'A' && data[j] <= 'Z') {
						complete = true
						break
					}
				}
				if !complete {
					debug.Log("*** INCOMPLETE ANSI at end: %q ***", string(data[i:]))
				}
				break
			}
		}
	}
}

// showMenuModal displays a modal menu
func (ta *TwistApp) showMenuModal(title string, options []string, callback func(string)) {
	ta.modalVisible = true
	ta.inputHandler.SetModalVisible(true)

	// Close any existing dropdown menus before showing modal
	ta.pages.RemovePage("dropdown-menu")
	if ta.menuComponent.IsDropdownVisible() {
		ta.menuComponent.HideDropdown()
	}

	// Use the new DOS-style modal list component
	modalList := components.NewModalList(title, options, func(selected string) {
		callback(selected)
		ta.closeModal()
	})

	// Set up ESC key handling to close modal
	modalList.SetDoneFunc(func() {
		ta.closeModal()
	})

	ta.pages.AddPage("modal", modalList.GetView(), true, true)
}

// closeModal closes the currently displayed modal
func (ta *TwistApp) closeModal() {
	ta.modalVisible = false
	ta.inputHandler.SetModalVisible(false)
	
	// Hide dropdown if visible
	if ta.menuComponent.IsDropdownVisible() {
		ta.menuComponent.HideDropdown()
	}
	
	// Remove any possible modal pages
	ta.pages.RemovePage("modal")
	ta.pages.RemovePage("script-modal")
	ta.pages.RemovePage("message-modal")
	ta.pages.RemovePage("help-modal")
	ta.pages.RemovePage("dropdown-menu")
	ta.pages.RemovePage("connection-dialog")
}

// startUpdateWorker starts the background update worker
func (ta *TwistApp) startUpdateWorker() {
	go func() {
		for range ta.terminalUpdateChan {
			ta.app.QueueUpdateDraw(func() {
				ta.updateTerminalView()
				ta.updatePanels()
			})
		}
	}()
}

// handleGlobalKeys handles global key events
func (ta *TwistApp) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	
	// Check global shortcuts first (including menu shortcuts like Alt+Q)
	if ta.globalShortcuts.HandleKeyEvent(event) {
		return nil
	}
	
	// ESC key to close modal if visible
	if event.Key() == tcell.KeyEscape && ta.modalVisible {
		ta.closeModal()
		return nil
	}
	
	// F1 key for help
	if event.Key() == tcell.KeyF1 {
		ta.showHelpModal()
		return nil
	}
	
	// Pass to input handler for menu Alt+keys and other keys
	return ta.inputHandler.HandleKeyEvent(event)
}


// showHelpModal displays help information
func (ta *TwistApp) showHelpModal() {
	// Close any existing dropdown menus before showing help modal
	ta.pages.RemovePage("dropdown-menu")
	if ta.menuComponent.IsDropdownVisible() {
		ta.menuComponent.HideDropdown()
	}

	helpText := "TWIST Terminal Interface\n\n" +
		"Menu Navigation:\n" +
		"Alt+S = Session menu\n" +
		"Alt+E = Edit menu\n" +
		"Alt+V = View menu\n" +
		"Alt+T = Terminal menu\n" +
		"Alt+H = Help menu\n" +
		"Alt+C = Connect\n" +
		"Alt+D = Disconnect\n" +
		"Alt+Q = Quit\n\n" +
		"Function Keys:\n" +
		"F1 = Help (this screen)\n" +
		"ESC = Close dialogs\n\n" +
		"Script management is available in the View menu."

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ta.closeModal()
		})
	ta.pages.AddPage("help-modal", modal, true, true)
	ta.modalVisible = true
}

// showDropdownMenu displays a dropdown menu below the menu bar
func (ta *TwistApp) showDropdownMenu(menuName string, options []string, callback func(string)) {
	
	// Convert string options to MenuItems with shortcuts for specific menus
	items := make([]twistComponents.MenuItem, len(options))
	for i, option := range options {
		var shortcut string
		// Add shortcuts for Session menu
		if menuName == "Session" {
			switch option {
			case "Quit":
				shortcut = "Alt+Q"
			// All other Session menu items have no shortcuts
			}
		}
		items[i] = twistComponents.MenuItem{Label: option, Shortcut: shortcut}
	}
	
	// Special handling for Session menu
	var dropdownCallback func(string)
	if menuName == "Session" {
		dropdownCallback = func(selected string) {
			switch selected {
			case "Connect":
				ta.showConnectionDialog()
				// Don't call ta.closeModal() here - let the dialog manage its own lifecycle
			case "Disconnect":
				ta.disconnect()
				ta.closeModal()
			case "Quit":
				ta.exit()
				// No need to close modal as app is exiting
			default:
				callback(selected)
				ta.closeModal()
			}
		}
	} else {
		// Standard dropdown behavior for other menus
		dropdownCallback = func(selected string) {
			callback(selected)
			ta.closeModal()
		}
	}
	
	dropdown := ta.menuComponent.ShowDropdown(menuName, items, dropdownCallback, func(direction string) {
		// Handle left/right arrow navigation between menus
		ta.navigateMenu(menuName, direction)
	}, ta.globalShortcuts)
	ta.pages.AddPage("dropdown-menu", dropdown, true, true)
	ta.modalVisible = true
}

// navigateMenu handles navigation between menu categories
func (ta *TwistApp) navigateMenu(currentMenu, direction string) {
	menus := []string{"Session", "Edit", "View", "Terminal", "Help"}
	currentIndex := -1
	
	// Find current menu index
	for i, menu := range menus {
		if menu == currentMenu {
			currentIndex = i
			break
		}
	}
	
	if currentIndex == -1 {
		return
	}
	
	// Calculate next menu
	var nextIndex int
	if direction == "left" {
		nextIndex = (currentIndex - 1 + len(menus)) % len(menus)
	} else {
		nextIndex = (currentIndex + 1) % len(menus)
	}
	
	nextMenu := menus[nextIndex]
	
	// Close current dropdown and open next one
	ta.closeModal()
	
	// Show the appropriate menu based on next menu
	switch nextMenu {
	case "Session":
		items := []twistComponents.MenuItem{
			{Label: "Connect", Shortcut: ""},
			{Label: "Recent Connections", Shortcut: ""},
			{Label: "Disconnect", Shortcut: ""},
			{Label: "Save Session", Shortcut: ""},
			{Label: "Quit", Shortcut: "Alt+Q"},
		}
		// Custom dropdown handler for Session menu that doesn't auto-close on Connect
		dropdown := ta.menuComponent.ShowDropdown("Session", items, func(selected string) {
			switch selected {
			case "Connect":
				ta.showConnectionDialog()
				// Don't call ta.closeModal() here - let the dialog manage its own lifecycle
			case "Disconnect":
				ta.disconnect()
				ta.closeModal()
			case "Quit":
				ta.exit()
				// No need to close modal as app is exiting
			}
		}, func(direction string) {
			// Handle left/right arrow navigation between menus
			ta.navigateMenu("Session", direction)
		}, ta.globalShortcuts)
		ta.pages.AddPage("dropdown-menu", dropdown, true, true)
		ta.modalVisible = true
	case "Edit":
		options := []string{"Cut", "Copy", "Paste", "Find", "Replace"}
		ta.showDropdownMenu("Edit", options, func(selected string) {
		})
	case "View":
		options := []string{"Scripts", "Zoom In", "Zoom Out", "Full Screen", "Panels"}
		ta.showDropdownMenu("View", options, func(selected string) {
		})
	case "Terminal":
		options := []string{"Clear", "Scroll Up", "Scroll Down", "Copy Selection"}
		ta.showDropdownMenu("Terminal", options, func(selected string) {
		})
	case "Help":
		options := []string{"Keyboard Shortcuts", "About", "User Manual"}
		ta.showDropdownMenu("Help", options, func(selected string) {
		})
	}
}

// showMessage displays a temporary message modal
func (ta *TwistApp) showMessage(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ta.closeModal()
		})
	ta.pages.AddPage("message-modal", modal, true, true)
	ta.modalVisible = true
}

// showConnectionDialog displays the connection dialog
func (ta *TwistApp) showConnectionDialog() {
	ta.modalVisible = true
	ta.inputHandler.SetModalVisible(true)

	// Remove dropdown menu page first
	ta.pages.RemovePage("dropdown-menu")
	// Hide dropdown if visible
	if ta.menuComponent.IsDropdownVisible() {
		ta.menuComponent.HideDropdown()
	}

	// Create connection dialog
	connectionDialog := components.NewConnectionDialog(
		func(address string) {
			debug.Log("Connection dialog submitted with address: %s", address)
			ta.connect(address)
			// Don't close modal immediately - let connection callbacks handle it
		},
		func() {
			debug.Log("Connection dialog cancelled")
			ta.closeModal()
		},
	)

	// Set up ESC key handling to close dialog
	connectionDialog.SetDoneFunc(func() {
		ta.closeModal()
	})

	ta.pages.AddPage("connection-dialog", connectionDialog.GetView(), true, true)
	ta.app.SetFocus(connectionDialog.GetForm())
}

// updatePanels updates the information panels
func (ta *TwistApp) updatePanels() {
	// Update with sample data - in real implementation, this would
	// get data from the game state
	ta.panelComponent.SetTraderInfoText("No trader data available")
	ta.panelComponent.SetSectorInfoText("No sector data available")
	
	// Update status bar
	ta.statusComponent.UpdateStatus()
}