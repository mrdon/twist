package tui

import (
	"log"
	"os"
	"twist/internal/ansi"
	"twist/internal/proxy"
	"twist/internal/terminal"
	"twist/internal/tui/components"
	"twist/internal/tui/handlers"
	"twist/internal/theme"
	twistComponents "twist/internal/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TwistApp represents the main tview application - refactored version
type TwistApp struct {
	app    *tview.Application
	logger *log.Logger
	proxy  *proxy.Proxy

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
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	logger := log.New(logFile, "[TVIEW] ", log.LstdFlags|log.Lshortfile)
	logger.Println("TView TUI initialized")

	// Create a dummy text view to create the ANSI converter
	// (we need this to get the theme-aware converter)
	dummyView := theme.NewTextView()
	ansiConverter := ansi.NewThemeAwareANSIWriter(dummyView)
	
	// Initialize terminal buffer with ANSI converter
	term := terminal.NewTerminalWithConverter(80, 50, ansiConverter)

	// Initialize proxy with the terminal as a writer
	proxyInstance := proxy.New(term)

	// Create the main application
	app := tview.NewApplication()

	// Create UI components
	menuComp := components.NewMenuComponent()
	terminalComp := components.NewTerminalComponent(term)
	panelComp := components.NewPanelComponent()
	statusComp := components.NewStatusComponent()

	// Create input handler
	inputHandler := handlers.NewInputHandler(app, logger)

	twistApp := &TwistApp{
		app:                app,
		logger:             logger,
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

	// Set up terminal update callback
	term.SetUpdateCallback(func() {
		logger.Printf("Terminal update callback triggered")
		select {
		case twistApp.terminalUpdateChan <- struct{}{}:
			logger.Printf("Sent terminal update message")
		default:
			logger.Printf("Terminal update channel full, skipping")
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
	ta.logger.Printf("=== APP: Setting connection dialog callback ===")
	ta.inputHandler.SetConnectionDialogCallback(ta.showConnectionDialog)
	ta.logger.Printf("=== APP: Connection dialog callback set ===")

	// Set up global input capture
	ta.app.SetInputCapture(ta.handleGlobalKeys)
}

// registerMenuShortcuts registers all menu item shortcuts globally at startup
func (ta *TwistApp) registerMenuShortcuts() {
	ta.logger.Printf("GLOBAL: Registering menu shortcuts")
	
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
			ta.logger.Printf("GLOBAL: Registering shortcut %s for %s", shortcut, label)
			ta.globalShortcuts.RegisterShortcut(shortcut, func() {
				ta.logger.Printf("GLOBAL: Shortcut %s triggered for %s", shortcut, label)
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
	ta.logger.Printf("Connecting to %s", address)
	
	if err := ta.proxy.Connect(address); err != nil {
		ta.logger.Printf("Connection failed: %v", err)
		return
	}
	
	ta.connected = true
	ta.serverAddress = address
	ta.menuComponent.SetConnectedMenu()
	ta.logger.Printf("Connected to %s", address)
}

// disconnect closes the connection to the game server
func (ta *TwistApp) disconnect() {
	if ta.connected {
		ta.proxy.Disconnect()
		ta.connected = false
		ta.menuComponent.SetDisconnectedMenu()
		ta.logger.Printf("Disconnected from server")
	}
}

// exit shuts down the application
func (ta *TwistApp) exit() {
	ta.disconnect()
	ta.app.Stop()
}

// sendCommand sends a command to the game server
func (ta *TwistApp) sendCommand(command string) {
	if ta.connected {
		ta.proxy.SendInput(command)
	}
}

// updateTerminalView updates the terminal display
func (ta *TwistApp) updateTerminalView() {
	ta.logger.Printf("DEBUG: updateTerminalView() called")
	if ta.terminalComponent == nil {
		ta.logger.Printf("ERROR: terminalComponent is nil!")
		return
	}
	ta.logger.Printf("DEBUG: About to call UpdateContent()")
	ta.terminalComponent.UpdateContent()
	ta.logger.Printf("DEBUG: UpdateContent() call completed")
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
			ta.logger.Printf("DEBUG: Update worker received message, queueing draw")
			ta.app.QueueUpdateDraw(func() {
				ta.logger.Printf("DEBUG: QueueUpdateDraw callback executing")
				ta.updateTerminalView()
				ta.updatePanels()
			})
		}
	}()
}

// handleGlobalKeys handles global key events
func (ta *TwistApp) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	ta.logger.Printf("GLOBAL KEY: Key=%v, Rune=%c, Modifiers=%v", event.Key(), event.Rune(), event.Modifiers())
	
	// Check global shortcuts first (including menu shortcuts like Alt+Q)
	if ta.globalShortcuts.HandleKeyEvent(event) {
		ta.logger.Printf("GLOBAL: Handled by global shortcut")
		return nil
	}
	
	// ESC key to close modal if visible
	if event.Key() == tcell.KeyEscape && ta.modalVisible {
		ta.logger.Printf("GLOBAL: Closing modal with ESC")
		ta.closeModal()
		return nil
	}
	
	// F1 key for help
	if event.Key() == tcell.KeyF1 {
		ta.logger.Printf("GLOBAL: Opening help with F1")
		ta.showHelpModal()
		return nil
	}
	
	// Pass to input handler for menu Alt+keys and other keys
	ta.logger.Printf("GLOBAL: Passing to input handler")
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
	ta.logger.Printf("DROPDOWN: Creating dropdown for %s", menuName)
	
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
			ta.logger.Printf("DROPDOWN: Selected %s from Session menu", selected)
			switch selected {
			case "Connect":
				ta.logger.Printf("=== DROPDOWN: Connect selected, showing dialog but NOT closing modal ===")
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
			ta.logger.Printf("DROPDOWN: Selected %s", selected)
			callback(selected)
			ta.closeModal()
		}
	}
	
	dropdown := ta.menuComponent.ShowDropdown(menuName, items, dropdownCallback, func(direction string) {
		// Handle left/right arrow navigation between menus
		ta.logger.Printf("DROPDOWN: Navigation %s from %s", direction, menuName)
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
	ta.logger.Printf("DROPDOWN: Switching from %s to %s", currentMenu, nextMenu)
	
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
			ta.logger.Printf("DROPDOWN: Selected %s", selected)
			switch selected {
			case "Connect":
				ta.logger.Printf("=== DROPDOWN: Connect selected, showing dialog but NOT closing modal ===")
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
			ta.logger.Printf("DROPDOWN: Navigation %s from %s", direction, "Session")
			ta.navigateMenu("Session", direction)
		}, ta.globalShortcuts)
		ta.pages.AddPage("dropdown-menu", dropdown, true, true)
		ta.modalVisible = true
	case "Edit":
		options := []string{"Cut", "Copy", "Paste", "Find", "Replace"}
		ta.showDropdownMenu("Edit", options, func(selected string) {
			ta.logger.Printf("Edit menu selection: %s", selected)
		})
	case "View":
		options := []string{"Scripts", "Zoom In", "Zoom Out", "Full Screen", "Panels"}
		ta.showDropdownMenu("View", options, func(selected string) {
			ta.logger.Printf("View menu selection: %s", selected)
		})
	case "Terminal":
		options := []string{"Clear", "Scroll Up", "Scroll Down", "Copy Selection"}
		ta.showDropdownMenu("Terminal", options, func(selected string) {
			ta.logger.Printf("Terminal menu selection: %s", selected)
		})
	case "Help":
		options := []string{"Keyboard Shortcuts", "About", "User Manual"}
		ta.showDropdownMenu("Help", options, func(selected string) {
			ta.logger.Printf("Help menu selection: %s", selected)
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
	ta.logger.Printf("=== APP: showConnectionDialog() called ===")
	ta.logger.Printf("=== APP: Setting modal visible to true ===")
	ta.modalVisible = true
	ta.inputHandler.SetModalVisible(true)

	ta.logger.Printf("=== APP: Closing any existing dropdown menus ===")
	// Remove dropdown menu page first
	ta.pages.RemovePage("dropdown-menu")
	// Hide dropdown if visible
	if ta.menuComponent.IsDropdownVisible() {
		ta.menuComponent.HideDropdown()
	}

	ta.logger.Printf("=== APP: Creating connection dialog component ===")
	// Create connection dialog
	connectionDialog := components.NewConnectionDialog(
		func(address string) {
			ta.logger.Printf("=== APP: Connection dialog callback: connecting to %s ===", address)
			ta.connect(address)
			ta.closeModal()
		},
		func() {
			ta.logger.Printf("=== APP: Connection dialog cancelled ===")
			ta.closeModal()
		},
	)

	ta.logger.Printf("=== APP: Setting up ESC key handling ===")
	// Set up ESC key handling to close dialog
	connectionDialog.SetDoneFunc(func() {
		ta.logger.Printf("=== APP: Connection dialog ESC pressed ===")
		ta.closeModal()
	})

	ta.logger.Printf("=== APP: Adding connection-dialog page to pages ===")
	ta.pages.AddPage("connection-dialog", connectionDialog.GetView(), true, true)
	ta.logger.Printf("=== APP: Setting focus to connection dialog form ===")
	ta.app.SetFocus(connectionDialog.GetForm())
	ta.logger.Printf("=== APP: showConnectionDialog() complete ===")
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