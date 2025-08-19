package tui

import (
	"fmt"
	"time"
	"twist/internal/debug"
	"twist/internal/terminal"
	"twist/internal/tui/components"
	"twist/internal/tui/handlers"
	"twist/internal/theme"
	twistComponents "twist/internal/components"
	"twist/internal/tui/api"
	coreapi "twist/internal/api"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TwistApp represents the main tview application - refactored version
type TwistApp struct {
	app    *tview.Application

	// API layer (now exclusive) 
	proxyClient *api.ProxyClient
	tuiAPI      coreapi.TuiAPI

	// Core components
	terminal     *terminal.Terminal
	pages        *tview.Pages
	mainGrid     *tview.Grid

	// UI Components
	menuComponent      *components.MenuComponent
	terminalComponent  *components.TerminalComponent
	panelComponent     *components.PanelComponent
	statusComponent    *components.StatusComponent

	// Input handling
	inputHandler        *handlers.InputHandler
	globalShortcuts     *twistComponents.GlobalShortcutManager

	// Sixel rendering layer
	sixelLayer *components.SixelLayer

	// State
	connected     bool
	serverAddress string
	modalVisible  bool
	panelsVisible bool
	animating     bool

	// Update channel
	terminalUpdateChan chan struct{}

	// Initial script to load on connection
	initialScript string
}

// NewApplication creates and configures the tview application
func NewApplication() *TwistApp {


	// Create the main application
	app := tview.NewApplication()

	// Create sixel layer first
	sixelLayer := components.NewSixelLayer()

	// Create UI components
	menuComp := components.NewMenuComponent()
	terminalComp := components.NewTerminalComponent(app)
	panelComp := components.NewPanelComponent(sixelLayer)
	statusComp := components.NewStatusComponent()

	// Set up width coordination between menu and status bar
	statusComp.SetMenuComponent(menuComp)

	// Create input handler
	inputHandler := handlers.NewInputHandler(app)

	twistApp := &TwistApp{
		app:                app,
		terminal:           nil, // No longer using terminal buffer
		connected:          false,
		serverAddress:      "twgs.geekm0nkey.com:23",
		terminalUpdateChan: make(chan struct{}, 100),
		menuComponent:      menuComp,
		terminalComponent:  terminalComp,
		panelComponent:     panelComp,
		statusComponent:    statusComp,
		inputHandler:       inputHandler,
		globalShortcuts:    twistComponents.NewGlobalShortcutManager(),
		sixelLayer:         sixelLayer,
		panelsVisible:      false, // Start with panels hidden
		animating:          false,
	}

	// Create API layer - proxy instances created per connection via static Connect()
	twistApp.proxyClient = api.NewProxyClient()
	twistApp.tuiAPI = api.NewTuiAPI(twistApp)

	// Set up terminal update callback for TerminalComponent
	terminalComp.SetChangedFunc(func() {
		app.QueueUpdateDraw(func() {
			// Terminal update
		})
	})

	// Script manager will be set via API after connection established
	// Script manager setup removed - will be handled in Phase 3

	twistApp.setupUI()
	twistApp.setupInputHandling()
	twistApp.registerMenuShortcuts() // Register all menu shortcuts globally
	// twistApp.startUpdateWorker() // Commented out - appears to be unused legacy code causing double redraws

	// Auto-show connection dialog on startup for easy testing
	go func() {
		// Small delay to ensure UI is fully initialized
		twistApp.app.QueueUpdateDraw(func() {
			twistApp.showConnectionDialog()
		})
	}()

	return twistApp
}

// setupUI configures the user interface layout
func (ta *TwistApp) setupUI() {
	// Set main grid background to pure black
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	// Initialize the UI without panels (they start hidden)
	ta.setupUILayout()

	// Create pages container
	ta.pages = tview.NewPages()
	ta.pages.SetBackgroundColor(defaultColors.Background) // Set pages background to black too
	ta.pages.AddPage("main", ta.mainGrid, true, true)

	ta.app.SetRoot(ta.pages, true)
	
	// Set up sixel layer rendering after tview draws
	ta.app.SetAfterDrawFunc(func(screen tcell.Screen) {
		// Render sixel layer after tview completes its drawing
		if ta.sixelLayer != nil {
			ta.sixelLayer.Render()
		}
	})
	
	// Always keep terminal focused and in terminal input mode
	ta.app.SetFocus(ta.terminalComponent.GetView())
	ta.inputHandler.SetInputMode(handlers.InputModeTerminal)
}

// setupUILayout creates the main grid layout based on panel visibility
func (ta *TwistApp) setupUILayout() {
	currentTheme := theme.Current()
	defaultColors := currentTheme.DefaultColors()
	
	if ta.panelsVisible {
		// Create main grid layout: 3 columns, 3 rows (menu, main content, status)
		// Left panel: 20 chars, Terminal: fixed 80 chars, Right panel: remaining space
		ta.mainGrid = tview.NewGrid().
			SetRows(1, 0, 1).
			SetColumns(30, 80, 0).
			SetBorders(false)
		
		ta.mainGrid.SetBackgroundColor(defaultColors.Background)

		// Add menu bar to top row, spanning all columns
		ta.mainGrid.AddItem(ta.menuComponent.GetView(), 0, 0, 1, 3, 0, 0, false)

		// Add panels and terminal to main area
		ta.mainGrid.AddItem(ta.panelComponent.GetLeftWrapper(), 1, 0, 1, 1, 0, 0, false)
		ta.mainGrid.AddItem(ta.terminalComponent.GetWrapper(), 1, 1, 1, 1, 0, 0, true)
		ta.mainGrid.AddItem(ta.panelComponent.GetRightWrapper(), 1, 2, 1, 1, 0, 0, false)

		// Add status bar to bottom row, spanning all columns
		ta.mainGrid.AddItem(ta.statusComponent.GetWrapper(), 2, 0, 1, 3, 0, 0, false)
	} else {
		// Create main grid layout: 1 column, 3 rows (menu, terminal, status)
		ta.mainGrid = tview.NewGrid().
			SetRows(1, 0, 1).
			SetColumns(0).
			SetBorders(false)
		
		ta.mainGrid.SetBackgroundColor(defaultColors.Background)

		// Add menu bar to top row
		ta.mainGrid.AddItem(ta.menuComponent.GetView(), 0, 0, 1, 1, 0, 0, false)

		// Add terminal to main area (no panels)
		ta.mainGrid.AddItem(ta.terminalComponent.GetWrapper(), 1, 0, 1, 1, 0, 0, true)

		// Add status bar to bottom row
		ta.mainGrid.AddItem(ta.statusComponent.GetWrapper(), 2, 0, 1, 1, 0, 0, false)
	}
}

// showPanels makes the side panels visible with animation
func (ta *TwistApp) showPanels() {
	if ta.panelsVisible || ta.animating {
		return
	}
	ta.animatePanels(true)
}

// hidePanels hides the side panels with animation
func (ta *TwistApp) hidePanels() {
	if !ta.panelsVisible || ta.animating {
		return
	}
	ta.animatePanels(false)
}

// animatePanels performs smooth panel show/hide animation
func (ta *TwistApp) animatePanels(show bool) {
	ta.animating = true
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
			}
		}()
		
		const animationFrames = 8
		const frameDuration = 30 * time.Millisecond
		
		// Get current theme for consistent colors
		currentTheme := theme.Current()
		defaultColors := currentTheme.DefaultColors()
		
		for frame := 0; frame <= animationFrames; frame++ {
			// Calculate animation progress (0.0 to 1.0)
			var progress float64
			if show {
				progress = float64(frame) / float64(animationFrames)
			} else {
				progress = 1.0 - float64(frame)/float64(animationFrames)
			}
			
			// Calculate panel widths based on progress
			leftPanelWidth := int(30.0 * progress)
			terminalWidth := 80
			// Right panel uses remaining space (0 means use remaining space in tview grid)
			
			// Ensure minimum widths
			if leftPanelWidth < 1 && progress > 0 {
				leftPanelWidth = 1
			}
			
			ta.app.QueueUpdateDraw(func() {
				// Create new grid with animated panel sizes
				if leftPanelWidth > 0 {
					// Panels are visible - create 3-column layout
					ta.mainGrid = tview.NewGrid().
						SetRows(1, 0, 1).
						SetColumns(leftPanelWidth, terminalWidth, 0).
						SetBorders(false)
					
					ta.mainGrid.SetBackgroundColor(defaultColors.Background)
					
					// Add components
					ta.mainGrid.AddItem(ta.menuComponent.GetView(), 0, 0, 1, 3, 0, 0, false)
					ta.mainGrid.AddItem(ta.panelComponent.GetLeftWrapper(), 1, 0, 1, 1, 0, 0, false)
					ta.mainGrid.AddItem(ta.terminalComponent.GetWrapper(), 1, 1, 1, 1, 0, 0, true)
					ta.mainGrid.AddItem(ta.panelComponent.GetRightWrapper(), 1, 2, 1, 1, 0, 0, false)
					// Add status bar to bottom row, spanning all columns
					ta.mainGrid.AddItem(ta.statusComponent.GetWrapper(), 2, 0, 1, 3, 0, 0, false)
				} else {
					// Panels are hidden - create 1-column layout
					ta.mainGrid = tview.NewGrid().
						SetRows(1, 0, 1).
						SetColumns(0).
						SetBorders(false)
					
					ta.mainGrid.SetBackgroundColor(defaultColors.Background)
					
					// Add components
					ta.mainGrid.AddItem(ta.menuComponent.GetView(), 0, 0, 1, 1, 0, 0, false)
					ta.mainGrid.AddItem(ta.terminalComponent.GetWrapper(), 1, 0, 1, 1, 0, 0, true)
					// Add status bar to bottom row
		ta.mainGrid.AddItem(ta.statusComponent.GetWrapper(), 2, 0, 1, 1, 0, 0, false)
				}
				
				// Update the page
				ta.pages.RemovePage("main")
				ta.pages.AddPage("main", ta.mainGrid, true, true)
				ta.app.SetFocus(ta.terminalComponent.GetView())
			})
			
			// Wait for next frame (except on last frame)
			if frame < animationFrames {
				time.Sleep(frameDuration)
			}
		}
		
		// Animation complete - update final state
		ta.panelsVisible = show
		ta.animating = false
		
		// Load real data when panels become visible
		if show && ta.panelComponent != nil && ta.proxyClient.IsConnected() {
			// Since GetPlayerInfo() returns CurrentSector: 0, we'll rely on sector change events
			// Try to get the last known sector from sector change events, or use a default
			// This is a workaround until GetPlayerInfo() is fixed
			ta.app.QueueUpdateDraw(func() {
				// Restore map component after animation completes
				ta.panelComponent.RestoreMapComponent()
				ta.panelComponent.LoadRealData()
			})
		} else {
			debug.Log("Panel visibility: show=%v, panelComponent=%v, connected=%v", 
				show, ta.panelComponent != nil, ta.proxyClient.IsConnected())
		}
	}()
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

// SetInitialScript sets the script to load on connection
func (ta *TwistApp) SetInitialScript(scriptName string) {
	ta.initialScript = scriptName
}

// Run starts the TUI application
func (ta *TwistApp) Run() error {
	return ta.app.Run()
}

// connect establishes connection to the game server
func (ta *TwistApp) connect(address string) {
	// Close modal immediately
	if ta.modalVisible {
		ta.closeModal()
	}
	
	// Use API layer exclusively - connection should be non-blocking
	// Proxy will call HandleConnecting, then HandleConnectionEstablished/HandleConnectionError
	if err := ta.proxyClient.ConnectWithScript(address, ta.tuiAPI, ta.initialScript); err != nil {
		// Handle immediate validation errors
		ta.connected = false
		ta.serverAddress = ""
		ta.menuComponent.SetDisconnectedMenu()
		ta.statusComponent.SetConnectionStatus(false, "Connection failed: "+err.Error())
		return
	}
	// Connection state will be updated via proxy callbacks
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
	if ta.sixelLayer != nil {
		ta.sixelLayer.Close()
	}
	ta.app.Stop()
}

// sendCommand sends a command to the game server
func (ta *TwistApp) sendCommand(command string) {
	if ta.proxyClient.IsConnected() {
		ta.proxyClient.SendData([]byte(command))
	}
	// When disconnected, we don't send commands to server, but local UI operations still work
}

// updateTerminalView updates the terminal display
func (ta *TwistApp) updateTerminalView() {
	if ta.terminalComponent == nil {
		return
	}
	ta.terminalComponent.UpdateContent()
}

// TuiAPI handler methods - called by API layer
func (ta *TwistApp) HandleConnectionStatusChanged(status coreapi.ConnectionStatus, address string) {
	// Handle status change asynchronously to ensure non-blocking
	go func() {
		// Add panic recovery to prevent crashes during callback
		defer func() {
			if r := recover(); r != nil {
			}
		}()
		
		ta.app.QueueUpdateDraw(func() {
			switch status {
			case coreapi.ConnectionStatusConnecting:
				ta.statusComponent.SetConnectionStatus(false, "Connecting to "+address+"...")
				
			case coreapi.ConnectionStatusConnected:
				ta.connected = true
				ta.serverAddress = address
				ta.menuComponent.SetConnectedMenu()
				ta.statusComponent.SetConnectionStatus(true, address)
				
				// Set ProxyAPI on status component after connection established
				if ta.proxyClient.IsConnected() {
					currentAPI := ta.proxyClient.GetCurrentAPI()
					ta.statusComponent.SetProxyAPI(currentAPI)
					ta.panelComponent.SetProxyAPI(currentAPI) // Add panel component API setup
				}
				
				if ta.modalVisible {
					ta.closeModal()
				}
				
			case coreapi.ConnectionStatusDisconnected:
				ta.connected = false
				ta.serverAddress = ""
				ta.menuComponent.SetDisconnectedMenu()
				ta.statusComponent.SetConnectionStatus(false, "")
				
				// Remove map component immediately to clear graphics
				if ta.panelComponent != nil {
					ta.panelComponent.RemoveMapComponent()
				}
				
				// Hide panels with animation when disconnecting
				ta.hidePanels()
				
				// Clear ProxyAPI from UI components to prevent stale references
				ta.statusComponent.SetProxyAPI(nil)
				ta.panelComponent.SetProxyAPI(nil)
				
				// Show disconnect message in terminal
				disconnectMsg := "\r\x1b[K\x1b[31;1m*** DISCONNECTED ***\x1b[0m\n"
				ta.terminalComponent.Write([]byte(disconnectMsg))
				
				// Ensure terminal keeps focus after disconnection
				ta.app.SetFocus(ta.terminalComponent.GetView())
			}
		})
	}()
}

func (ta *TwistApp) HandleConnectionError(err error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
			}
		}()
		
		ta.app.QueueUpdateDraw(func() {
			ta.connected = false
			ta.serverAddress = ""
			ta.menuComponent.SetDisconnectedMenu()
			ta.statusComponent.SetConnectionStatus(false, "")
			
			// Remove map component immediately to clear graphics
			if ta.panelComponent != nil {
				ta.panelComponent.RemoveMapComponent()
			}
			
			// Hide panels with animation when connection fails
			ta.hidePanels()
			
			// Clear ProxyAPI from UI components to prevent stale references
			ta.statusComponent.SetProxyAPI(nil)
			ta.panelComponent.SetProxyAPI(nil)
			
			// Ensure modal is closed if it's still open
			if ta.modalVisible {
				ta.closeModal()
			}
			
			// TODO: Show error modal: ta.showErrorModal(err)
		})
	}()
}

func (ta *TwistApp) HandleTerminalData(data []byte) {
	// Terminal data should be handled asynchronously to avoid blocking
	go func() {
		// Add error recovery to catch any panics in terminal processing
		defer func() {
			if r := recover(); r != nil {
			}
		}()
		
		// Write directly to the TerminalComponent
		ta.terminalComponent.Write(data)
		
		// UI refresh is handled by the TerminalView's change callback
	}()
}

// Script event handlers
func (ta *TwistApp) HandleScriptStatusChanged(status coreapi.ScriptStatusInfo) {
	// Status component will be updated automatically via SetProxyAPI
	// For now, just output to terminal for visibility
	msg := fmt.Sprintf("Script status: %d active, %d total\n", 
		status.ActiveCount, status.TotalCount)
	ta.terminalComponent.Write([]byte(msg))
}

func (ta *TwistApp) HandleScriptError(scriptName string, err error) {
	// Output script error to terminal
	msg := fmt.Sprintf("Script error in %s: %s\n", scriptName, err.Error())
	ta.terminalComponent.Write([]byte(msg))
}

// HandleDatabaseStateChanged processes database loading/unloading events
func (ta *TwistApp) HandleDatabaseStateChanged(info coreapi.DatabaseStateInfo) {
	
	// Handle database state change asynchronously to ensure non-blocking
	go func() {
		defer func() {
			if r := recover(); r != nil {
			}
		}()
		
		ta.app.QueueUpdateDraw(func() {
			// Update status bar to show active game information
			ta.statusComponent.SetGameInfo(info.GameName, info.ServerHost, info.ServerPort, info.IsLoaded)
			
			// Show/hide panels based on database loading state
			if info.IsLoaded {
				// Don't restore map component here - wait for animation to complete
				ta.showPanels()
			} else {
				// Remove map component when game unloads
				if ta.panelComponent != nil {
					ta.panelComponent.RemoveMapComponent()
				}
				ta.hidePanels()
			}
		})
	}()
}

// HandleCurrentSectorChanged processes sector change events
func (ta *TwistApp) HandleCurrentSectorChanged(sectorInfo coreapi.SectorInfo) {
	ta.app.QueueUpdateDraw(func() {
		// Update panels directly with sector info (no need to re-fetch)
		if ta.panelComponent != nil && ta.proxyClient.IsConnected() {
			ta.refreshPanelDataWithInfo(sectorInfo)
		}
	})
}

// HandlePortUpdated processes port information update events
func (ta *TwistApp) HandlePortUpdated(portInfo coreapi.PortInfo) {
	
	ta.app.QueueUpdateDraw(func() {
		// Port updates don't affect map visualization (which only cares about warps)
		// Skip calling UpdateSectorData to avoid triggering unnecessary map redraws
		if ta.panelComponent != nil && ta.proxyClient.IsConnected() {
			debug.Log("TwistApp: Handling port update for sector %d - skipping map update (port info doesn't affect warp display)", portInfo.SectorID)
			// TODO: If we need to update port information in other components, do it here
			// without calling UpdateSectorData which triggers map redraws
		}
	})
}

// HandleTraderDataUpdated processes trader information update events
func (ta *TwistApp) HandleTraderDataUpdated(sectorNumber int, traders []coreapi.TraderInfo) {
	ta.app.QueueUpdateDraw(func() {
		// Update trader panel with new trader data
		if ta.panelComponent != nil && ta.proxyClient.IsConnected() {
			ta.panelComponent.UpdateTraderData(sectorNumber, traders)
		}
	})
}

// HandlePlayerStatsUpdated processes player statistics update events  
func (ta *TwistApp) HandlePlayerStatsUpdated(stats coreapi.PlayerStatsInfo) {
	ta.app.QueueUpdateDraw(func() {
		// Update trader panel with current player stats (for display context)
		if ta.panelComponent != nil && ta.proxyClient.IsConnected() {
			ta.panelComponent.UpdatePlayerStats(stats)
		}
	})
}

// HandleSectorUpdated processes sector data update events (e.g. from etherprobe)
func (ta *TwistApp) HandleSectorUpdated(sectorInfo coreapi.SectorInfo) {
	ta.app.QueueUpdateDraw(func() {
		// Update sector data in maps without changing current sector focus
		if ta.panelComponent != nil && ta.proxyClient.IsConnected() {
			debug.Log("TwistApp: Handling sector data update for sector %d", sectorInfo.Number)
			ta.panelComponent.UpdateSectorData(sectorInfo)
		}
	})
}

// refreshPanelDataWithInfo refreshes panel data using provided sector info
func (ta *TwistApp) refreshPanelDataWithInfo(sectorInfo coreapi.SectorInfo) {
	
	// Only refresh if panels are visible
	if !ta.panelsVisible {
		return
	}
	
	// Update panels directly with the provided sector info
	ta.panelComponent.UpdateSectorInfo(sectorInfo)
	
	// Always attempt to load/update player stats when sector changes
	debug.Log("refreshPanelDataWithInfo: sector %d, hasDetailedStats: %v", sectorInfo.Number, ta.panelComponent.HasDetailedPlayerStats())
	debug.Log("refreshPanelDataWithInfo: always calling UpdatePlayerStatsSector")
	ta.panelComponent.UpdatePlayerStatsSector(sectorInfo.Number)
}

// refreshPanelData refreshes panel data using API calls
func (ta *TwistApp) refreshPanelData(sectorNumber int) {
	
	// Only refresh if panels are visible
	if !ta.panelsVisible {
		return
	}
	
	proxyAPI := ta.proxyClient.GetCurrentAPI()
	if proxyAPI != nil {
		// Get sector info and update panel
		sectorInfo, err := proxyAPI.GetSectorInfo(sectorNumber)
		if err == nil {
			ta.panelComponent.UpdateSectorInfo(sectorInfo)
		} else {
		}
		
		// Create fake player info with the current sector since GetPlayerInfo() is broken
		playerInfo := coreapi.PlayerInfo{
			Name:          "Player", // We don't have the real name from GetPlayerInfo
			CurrentSector: sectorNumber,
		}
		ta.panelComponent.UpdateTraderInfo(playerInfo)
		
		// Also update sector map
		if ta.panelComponent != nil {
			ta.panelComponent.UpdateSectorInfo(sectorInfo)
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
	
	// Ctrl+C - exit the application (check multiple ways)
	if event.Key() == tcell.KeyCtrlC {
		ta.exit()
		return nil
	}
	
	// Also check for Ctrl+C via rune and modifiers
	if event.Rune() == 'c' && event.Modifiers()&tcell.ModCtrl != 0 {
		ta.exit()
		return nil
	}
	
	// Also check for key code 3 (ETX) which is the ASCII value for Ctrl+C
	if event.Key() == tcell.KeyETX {
		ta.exit()
		return nil
	}
	
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
				ta.connect(address)
			// Don't close modal immediately - let connection callbacks handle it
		},
		func() {
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
	
	// Update status bar
	ta.statusComponent.UpdateStatus()
}

