package tui

import (
	"log"
	"os"
	"twist/internal/proxy"
	"twist/internal/terminal"
	"twist/internal/tui/components"
	"twist/internal/tui/handlers"

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

	// Input handling
	inputHandler *handlers.InputHandler

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

	// Initialize terminal buffer
	term := terminal.NewTerminal(80, 50)

	// Initialize proxy with the terminal as a writer
	proxyInstance := proxy.New(term)

	// Create the main application
	app := tview.NewApplication()

	// Create UI components
	menuComp := components.NewMenuComponent()
	terminalComp := components.NewTerminalComponent(term)
	panelComp := components.NewPanelComponent()

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
		inputHandler:       inputHandler,
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

	twistApp.setupUI()
	twistApp.setupInputHandling()
	twistApp.startUpdateWorker()

	return twistApp
}

// setupUI configures the user interface layout
func (ta *TwistApp) setupUI() {
	// Create main grid layout
	ta.mainGrid = tview.NewGrid().
		SetRows(1, 0).
		SetColumns(20, 0, 20).
		SetBorders(false)

	// Add menu bar to top row, spanning all columns
	ta.mainGrid.AddItem(ta.menuComponent.GetView(), 0, 0, 1, 3, 0, 0, false)

	// Add panels and terminal to main area
	ta.mainGrid.AddItem(ta.panelComponent.GetLeftWrapper(), 1, 0, 1, 1, 0, 0, false)
	ta.mainGrid.AddItem(ta.terminalComponent.GetWrapper(), 1, 1, 1, 1, 0, 0, true)
	ta.mainGrid.AddItem(ta.panelComponent.GetRightWrapper(), 1, 2, 1, 1, 0, 0, false)

	// Create pages container
	ta.pages = tview.NewPages()
	ta.pages.AddPage("main", ta.mainGrid, true, true)

	ta.app.SetRoot(ta.pages, true)
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

	// Set up global input capture
	ta.app.SetInputCapture(ta.inputHandler.HandleKeyEvent)
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
	ta.terminalComponent.UpdateContent()
}

// showMenuModal displays a modal menu
func (ta *TwistApp) showMenuModal(title string, options []string, callback func(string)) {
	ta.modalVisible = true
	ta.inputHandler.SetModalVisible(true)

	list := tview.NewList()
	for _, option := range options {
		list.AddItem(option, "", 0, func() {
			callback(option)
			ta.closeModal()
		})
	}

	modal := tview.NewModal().
		SetText(title).
		AddButtons([]string{"Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ta.closeModal()
		})

	ta.pages.AddPage("modal", modal, true, true)
}

// closeModal closes the currently displayed modal
func (ta *TwistApp) closeModal() {
	ta.modalVisible = false
	ta.inputHandler.SetModalVisible(false)
	ta.pages.RemovePage("modal")
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

// updatePanels updates the information panels
func (ta *TwistApp) updatePanels() {
	// Update with sample data - in real implementation, this would
	// get data from the game state
	ta.panelComponent.SetTraderInfoText("No trader data available")
	ta.panelComponent.SetSectorInfoText("No sector data available")
}