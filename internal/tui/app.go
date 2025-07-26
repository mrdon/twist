package tui

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"twist/internal/proxy"
	"twist/internal/terminal"
)

// TwistApp represents the main tview application
type TwistApp struct {
	app    *tview.Application
	logger *log.Logger
	proxy  *proxy.Proxy

	// Core components
	terminal        *terminal.Terminal
	pages           *tview.Pages
	mainGrid        *tview.Grid
	menuBar         *tview.TextView
	terminalView    *tview.TextView
	terminalWrapper *tview.Flex
	leftPanel       *tview.TextView
	leftWrapper     *tview.Flex
	rightPanel      *tview.TextView
	rightWrapper    *tview.Flex

	// State
	connected     bool
	serverAddress string
	inputMode     InputMode
	modalVisible  bool

	// Update channel
	terminalUpdateChan chan struct{}
}

type InputMode int

const (
	InputModeMenu InputMode = iota
	InputModeTerminal
	InputModeModal
)

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

	twistApp := &TwistApp{
		app:                app,
		logger:             logger,
		proxy:              proxyInstance,
		terminal:           term,
		connected:          false,
		serverAddress:      "twgs.geekm0nkey.com:23",
		inputMode:          InputModeMenu,
		terminalUpdateChan: make(chan struct{}, 100),
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

// Run starts the tview application
func (ta *TwistApp) Run() error {
	return ta.app.Run()
}

// setupUI creates and configures all UI components
func (ta *TwistApp) setupUI() {
	// Create menu bar
	ta.menuBar = tview.NewTextView()
	ta.menuBar.SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetText("[black:blue:b] File  Edit  View  Terminal  Help [white:black:-]").
		SetBorder(false)

	// Create left panel (trader info)
	ta.leftPanel = tview.NewTextView()
	ta.leftPanel.SetDynamicColors(true).
		SetText(ta.getTraderInfoText()).
		SetBorder(false) // Remove border from inner view
	
	// Wrap left panel with padding
	ta.leftWrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).                                    // Top padding
		AddItem(tview.NewFlex().
			AddItem(nil, 1, 0, false).                                // Left padding
			AddItem(ta.leftPanel, 0, 1, false).                      // Left panel content
			AddItem(nil, 1, 0, false),                                // Right padding
			0, 1, false).                                             // Middle row (flexible)
		AddItem(nil, 1, 0, false)                                     // Bottom padding
	
	ta.leftWrapper.SetBorder(true).
		SetTitle("Trader Info").
		SetTitleAlign(tview.AlignLeft)

	// Create terminal view
	ta.terminalView = tview.NewTextView()
	ta.terminalView.SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(false).
		SetBorder(false) // Remove border from inner view
	
	// Wrap terminal in a flex container for padding
	ta.terminalWrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).                                    // Top padding
		AddItem(tview.NewFlex().
			AddItem(nil, 1, 0, false).                                // Left padding
			AddItem(ta.terminalView, 0, 1, true).                     // Terminal view (flexible, focusable)
			AddItem(nil, 1, 0, false),                                // Right padding
			0, 1, true).                                              // Middle row (flexible, focusable)
		AddItem(nil, 1, 0, false)                                     // Bottom padding
	
	ta.terminalWrapper.SetBorder(true).
		SetTitle("Terminal").
		SetTitleAlign(tview.AlignLeft)

	// Create right panel (sector info)
	ta.rightPanel = tview.NewTextView()
	ta.rightPanel.SetDynamicColors(true).
		SetText(ta.getSectorInfoText()).
		SetBorder(false) // Remove border from inner view
	
	// Wrap right panel with padding
	ta.rightWrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).                                    // Top padding
		AddItem(tview.NewFlex().
			AddItem(nil, 1, 0, false).                                // Left padding
			AddItem(ta.rightPanel, 0, 1, false).                     // Right panel content
			AddItem(nil, 1, 0, false),                                // Right padding
			0, 1, false).                                             // Middle row (flexible)
		AddItem(nil, 1, 0, false)                                     // Bottom padding
	
	ta.rightWrapper.SetBorder(true).
		SetTitle("Sector Info").
		SetTitleAlign(tview.AlignLeft)

	// Create main grid layout
	ta.mainGrid = tview.NewGrid().
		SetRows(1, 0). // Menu bar row (1 line) + main content (flexible)
		SetColumns(0, 82, 0). // Left panel (flexible) + terminal (80 chars + 2 padding) + right panel (flexible)
		SetBorders(false)

	// Add components to grid
	ta.mainGrid.AddItem(ta.menuBar, 0, 0, 1, 3, 0, 0, false)          // Menu bar spans all columns
	ta.mainGrid.AddItem(ta.leftWrapper, 1, 0, 1, 1, 0, 0, false)      // Left wrapper
	ta.mainGrid.AddItem(ta.terminalWrapper, 1, 1, 1, 1, 0, 0, true)   // Terminal wrapper (focusable)
	ta.mainGrid.AddItem(ta.rightWrapper, 1, 2, 1, 1, 0, 0, false)     // Right wrapper

	// Create pages for modal overlay support
	ta.pages = tview.NewPages().
		AddPage("main", ta.mainGrid, true, true)

	// Set the root
	ta.app.SetRoot(ta.pages, true)

	// Show initial menu
	ta.showMainMenu()
}

// setupInputHandling configures global input handling
func (ta *TwistApp) setupInputHandling() {
	ta.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		ta.logger.Printf("Global key event: Key=%v, Rune=%c, Mode=%d", event.Key(), event.Rune(), ta.inputMode)

		// Handle ESC and Ctrl+C to always allow exit
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyCtrlC {
			if ta.inputMode == InputModeModal {
				ta.logger.Printf("ESC/Ctrl+C in modal - closing modal")
				ta.closeModal()  // Use proper closeModal instead of direct SetRoot
				return nil
			} else if event.Key() == tcell.KeyCtrlC {
				ta.logger.Printf("Ctrl+C - quitting app")
				ta.app.Stop()
				return nil
			}
		}

		// Handle Alt+key combinations for menu (check both modifier flag and key codes)
		isAltKey := event.Modifiers()&tcell.ModAlt != 0 || 
					(event.Key() >= tcell.KeyRune && event.Key() <= tcell.KeyRune+25 && event.Modifiers() != 0)
		
		if isAltKey {
			ta.logger.Printf("Alt combination detected - Key=%v, Rune=%c, Modifiers=%v", event.Key(), event.Rune(), event.Modifiers())
			switch event.Rune() {
			case 'f', 'F':
				ta.logger.Printf("Alt+F detected - showing File menu")
				ta.showFileMenu()
				return nil
			case 'e', 'E':
				ta.logger.Printf("Alt+E detected - showing Edit menu")
				ta.showEditMenu()
				return nil
			case 'v', 'V':
				ta.logger.Printf("Alt+V detected - showing View menu")
				ta.showViewMenu()
				return nil
			case 't', 'T':
				ta.logger.Printf("Alt+T detected - showing Terminal menu")
				ta.showTerminalMenu()
				return nil
			case 'h', 'H':
				ta.logger.Printf("Alt+H detected - showing Help menu")
				ta.showHelpMenu()
				return nil
			}
		}
		
		// Also check for Meta key combinations (some terminals use this for Alt)
		if event.Key() >= tcell.KeyF1 && event.Key() <= tcell.KeyF12 {
			// Skip function keys for now
		} else if event.Key() == tcell.KeyRune && event.Modifiers() == 0 {
			// Check if this might be an Alt sequence we missed
			switch event.Rune() {
			case 'f':
				if ta.inputMode == InputModeMenu {
					ta.logger.Printf("Possible Alt+F fallback")
					ta.showFileMenu()
					return nil
				}
			case 'e':
				if ta.inputMode == InputModeMenu {
					ta.logger.Printf("Possible Alt+E fallback")
					ta.showEditMenu()
					return nil
				}
			}
		}

		switch ta.inputMode {
		case InputModeMenu:
			return ta.handleMenuInput(event)
		case InputModeTerminal:
			return ta.handleTerminalInput(event)
		case InputModeModal:
			// Let modals handle their own input
			return event
		}

		return event
	})
}

// showMainMenu displays the main menu in the terminal view
func (ta *TwistApp) showMainMenu() {
	menuText := `[yellow::b]Trade Wars 2002 Client[white::-]

Main Menu:

[yellow]1.[white] Connect to Server
[yellow]2.[white] Disconnect  
[yellow]3.[white] Quit

[gray]Use number keys to select an option
Press Ctrl+C to quit at any time[white]

Current server: [green]` + ta.serverAddress + `[white]`

	ta.terminalView.SetText(menuText)
	ta.inputMode = InputModeMenu
}

// handleMenuInput processes input when in menu mode
func (ta *TwistApp) handleMenuInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case '1':
			ta.showConnectDialog()
			return nil
		case '2':
			ta.disconnect()
			return nil
		case '3', 'q', 'Q':
			ta.app.Stop()
			return nil
		}
	case tcell.KeyCtrlC:
		ta.app.Stop()
		return nil
	case tcell.KeyEscape:
		if ta.connected {
			ta.inputMode = InputModeTerminal
			ta.terminalView.SetTitle("Terminal - Connected")
		}
		return nil
	}
	return event
}

// handleTerminalInput processes input when in terminal mode
func (ta *TwistApp) handleTerminalInput(event *tcell.EventKey) *tcell.EventKey {
	if !ta.connected {
		ta.showMainMenu()
		return nil
	}

	switch event.Key() {
	case tcell.KeyCtrlC:
		ta.app.Stop()
		return nil
	case tcell.KeyEscape:
		ta.showMainMenu()
		return nil
	case tcell.KeyEnter:
		ta.proxy.SendInput("\r\n")
		return nil
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		ta.proxy.SendInput("\b")
		return nil
	case tcell.KeyTab:
		ta.proxy.SendInput("\t")
		return nil
	case tcell.KeyUp:
		if event.Modifiers()&tcell.ModShift != 0 {
			// Shift+Up for scrolling up one line
			row, _ := ta.terminalView.GetScrollOffset()
			if row > 0 {
				ta.terminalView.ScrollTo(row-1, 0)
			}
		} else {
			ta.proxy.SendInput("\x1b[A")
		}
		return nil
	case tcell.KeyDown:
		if event.Modifiers()&tcell.ModShift != 0 {
			// Shift+Down for scrolling down one line
			row, _ := ta.terminalView.GetScrollOffset()
			ta.terminalView.ScrollTo(row+1, 0)
		} else {
			ta.proxy.SendInput("\x1b[B")
		}
		return nil
	case tcell.KeyPgUp:
		if event.Modifiers()&tcell.ModShift != 0 {
			// Shift+PgUp for scrolling up a page
			row, _ := ta.terminalView.GetScrollOffset()
			_, _, _, height := ta.terminalView.GetInnerRect()
			newRow := row - height
			if newRow < 0 {
				newRow = 0
			}
			ta.terminalView.ScrollTo(newRow, 0)
		} else {
			ta.proxy.SendInput("\x1b[5~")
		}
		return nil
	case tcell.KeyPgDn:
		if event.Modifiers()&tcell.ModShift != 0 {
			// Shift+PgDown for scrolling down a page
			row, _ := ta.terminalView.GetScrollOffset()
			_, _, _, height := ta.terminalView.GetInnerRect()
			ta.terminalView.ScrollTo(row+height, 0)
		} else {
			ta.proxy.SendInput("\x1b[6~")
		}
		return nil
	case tcell.KeyRune:
		ta.proxy.SendInput(string(event.Rune()))
		return nil
	}

	return event
}

// closeModal closes any open modal and returns to menu
func (ta *TwistApp) closeModal() {
	ta.logger.Printf("Closing modal, returning to menu (modalVisible was: %v)", ta.modalVisible)
	ta.modalVisible = false
	ta.pages.RemovePage("modal")
	
	// Switch back to main page
	ta.pages.SwitchToPage("main")
	
	// Determine correct input mode based on connection status
	if ta.connected {
		ta.inputMode = InputModeTerminal
		ta.app.SetFocus(ta.terminalWrapper)
		ta.logger.Printf("Modal closed - returning to terminal mode (modalVisible now: %v)", ta.modalVisible)
	} else {
		ta.inputMode = InputModeMenu
		ta.app.SetFocus(ta.terminalWrapper)  // Still focus terminal for consistency
		ta.logger.Printf("Modal closed - returning to menu mode (modalVisible now: %v)", ta.modalVisible)
	}
}

// showConnectDialog shows a modal dialog for server connection  
func (ta *TwistApp) showConnectDialog() {
	options := []string{
		"Connect to " + ta.serverAddress,
		"Change Server Address",
		"Cancel",
	}
	
	ta.showMenuModal("Connect to Server", options, func(option string) {
		switch {
		case strings.HasPrefix(option, "Connect to"):
			ta.inputMode = InputModeTerminal
			go func() {
				ta.connect(ta.serverAddress)
			}()
		case option == "Change Server Address":
			ta.showAddressInputDialog()
		case option == "Cancel":
			// Modal already closed by showMenuModal
		}
	})
}

// showAddressInputDialog shows a dialog to change server address
func (ta *TwistApp) showAddressInputDialog() {
	form := tview.NewForm().
		AddInputField("Server Address:", ta.serverAddress, 40, nil, func(text string) {
			ta.serverAddress = text
		}).
		AddButton("Connect", func() {
			ta.closeModal()
			ta.inputMode = InputModeTerminal
			go func() {
				ta.connect(ta.serverAddress)
			}()
		}).
		AddButton("Cancel", func() {
			ta.closeModal()
		})

	form.SetBorder(true).
		SetTitle("Enter Server Address").
		SetTitleAlign(tview.AlignCenter)

	form.SetCancelFunc(func() {
		ta.closeModal()
	})

	// Center the form
	modal := tview.NewGrid().
		SetColumns(0, 60, 0).
		SetRows(0, 10, 0).
		AddItem(form, 1, 1, 1, 1, 0, 0, true)

	// Ensure any existing modal is removed first
	ta.pages.RemovePage("modal")

	// Add modal as overlay page
	ta.pages.AddPage("modal", modal, true, true)
	ta.inputMode = InputModeModal
	ta.app.SetFocus(form)
}

// connect attempts to connect to the specified server
func (ta *TwistApp) connect(address string) {
	ta.logger.Printf("Attempting to connect to %s", address)
	
	err := ta.proxy.Connect(address)
	if err != nil {
		ta.logger.Printf("Connection failed: %v", err)
		ta.terminalView.SetText(fmt.Sprintf("[red]Error: %v[white]\n\nPress any key to return to menu", err))
		return
	}

	ta.logger.Printf("Successfully connected to %s", address) 
	ta.connected = true
	ta.serverAddress = address
	ta.inputMode = InputModeTerminal
	ta.terminalView.SetTitle("Terminal - Connected to " + address)
	
	// Write connection message to terminal
	connText := fmt.Sprintf("Connected to %s\r\n", address)
	ta.terminal.Write([]byte(connText))
}

// disconnect disconnects from the current server
func (ta *TwistApp) disconnect() {
	if ta.connected {
		ta.proxy.Disconnect()
		ta.connected = false
		ta.terminal.Write([]byte("Disconnected\r\n"))
	}
	ta.showMainMenu()
}

// updateTerminalView updates the terminal display with current buffer content
func (ta *TwistApp) updateTerminalView() {
	cells := ta.terminal.GetAllCells()
	lines := ta.convertTerminalCellsToText(cells)
	
	if len(lines) == 0 {
		return
	}

	content := strings.Join(lines, "\n")
	ta.terminalView.SetText(content)
	ta.terminalView.ScrollToEnd()
}

// convertTerminalCellsToText converts terminal cells to tview-formatted text with colors
func (ta *TwistApp) convertTerminalCellsToText(cells [][]terminal.Cell) []string {
	var lines []string

	for _, row := range cells {
		var line strings.Builder
		var currentFg, currentBg int = -1, -1
		var currentBold, currentUnderline bool
		
		for _, cell := range row {
			// Check if we need to change styling
			styleChanged := false
			if cell.Foreground != currentFg || cell.Background != currentBg || 
			   cell.Bold != currentBold || cell.Underline != currentUnderline {
				styleChanged = true
			}
			
			if styleChanged {
				// Apply new style using tview color tags
				fg := ta.ansiToTviewColor(cell.Foreground)
				bg := ta.ansiToTviewColor(cell.Background)
				
				var style strings.Builder
				style.WriteString("[")
				style.WriteString(fg)
				if bg != "black" && bg != "-" {
					style.WriteString(":")
					style.WriteString(bg)
				} else {
					style.WriteString(":-")
				}
				if cell.Bold {
					style.WriteString(":b")
				} else if cell.Underline {
					style.WriteString(":u")
				} else {
					style.WriteString(":-")
				}
				style.WriteString("]")
				
				line.WriteString(style.String())
				
				currentFg = cell.Foreground
				currentBg = cell.Background
				currentBold = cell.Bold
				currentUnderline = cell.Underline
			}
			
			if cell.Char == 0 {
				line.WriteRune(' ')
			} else {
				line.WriteRune(cell.Char)
			}
		}
		lines = append(lines, strings.TrimRight(line.String(), " "))
	}

	return lines
}

// ansiToTviewColor converts ANSI color codes to tview color names
func (ta *TwistApp) ansiToTviewColor(colorCode int) string {
	colors := map[int]string{
		0: "black",
		1: "red", 
		2: "green",
		3: "yellow",
		4: "blue",
		5: "purple",
		6: "teal",
		7: "white",
		8: "gray",
		9: "red",
		10: "lime",
		11: "yellow", 
		12: "blue",
		13: "fuchsia",
		14: "aqua",
		15: "white",
	}
	
	if color, ok := colors[colorCode]; ok {
		return color
	}
	
	// Handle 256-color mode - map to closest basic color
	if colorCode >= 16 && colorCode <= 231 {
		// 216 color cube
		return "white"
	} else if colorCode >= 232 && colorCode <= 255 {
		// Grayscale
		return "gray"
	}
	
	return "white" // Default
}

// startUpdateWorker starts a goroutine to handle terminal updates
func (ta *TwistApp) startUpdateWorker() {
	go func() {
		for range ta.terminalUpdateChan {
			ta.app.QueueUpdateDraw(func() {
				ta.updateTerminalView()
			})
		}
	}()

	// Start error listener
	go func() {
		for err := range ta.proxy.GetErrorChan() {
			ta.logger.Printf("Received error: %v", err)
			ta.app.QueueUpdateDraw(func() {
				errorText := fmt.Sprintf("[red]Error: %v[white]", err)
				ta.terminalView.SetText(errorText)
			})
		}
	}()
}

// getTraderInfoText returns the trader info panel content
func (ta *TwistApp) getTraderInfoText() string {
	return `[yellow::b]Trader Info[white::-]

Sector:     5379
Turns:      150
Experience: 1087
Alignment:  -33
Credits:    142,439

[yellow::b]Holds[white::-]

Total:      150
Fuel Ore:   0
Organics:   0
Equipment:  150
Colonists:  0
Empty:      0

[yellow::b]Quick Query[white::-]

[Input field here]

[yellow::b]Stats[white::-]

Profit:     0`
}

// getSectorInfoText returns the sector info panel content  
func (ta *TwistApp) getSectorInfoText() string {
	return `[yellow::b]Sector 5379[white::-]

Port:       Class 9 (Stardock)
Density:    101 Fighters
NavHaz:     0%
Anom:       No

[yellow::b]Visual Map[white::-]

        *     *
    *       *   *
  *   * [5379] *
    *   *   *
      *   *

[yellow::b]Notepad[white::-]

Notes will appear here...`
}

// showFileMenu shows the File menu modal
func (ta *TwistApp) showFileMenu() {
	ta.showMenuModal("File Menu", []string{
		"Connect to Server",
		"Disconnect",
		"---",
		"Exit",
	}, func(option string) {
		switch option {
		case "Connect to Server":
			ta.showConnectDialog()
		case "Disconnect":
			ta.disconnect()
		case "Exit":
			ta.app.Stop()
		}
	})
}

// showEditMenu shows the Edit menu modal
func (ta *TwistApp) showEditMenu() {
	ta.showMenuModal("Edit Menu", []string{
		"Copy",
		"Paste",
		"---",
		"Clear Terminal",
	}, func(option string) {
		switch option {
		case "Clear Terminal":
			ta.terminalView.SetText("")
		}
	})
}

// showViewMenu shows the View menu modal
func (ta *TwistApp) showViewMenu() {
	ta.showMenuModal("View Menu", []string{
		"Toggle Full Screen",
		"---",
		"Zoom In",
		"Zoom Out",
	}, func(option string) {
		// Placeholder for view options
	})
}

// showTerminalMenu shows the Terminal menu modal
func (ta *TwistApp) showTerminalMenu() {
	ta.showMenuModal("Terminal Menu", []string{
		"Connect...",
		"Disconnect",
		"---",
		"Reset Terminal",
	}, func(option string) {
		switch option {
		case "Connect...":
			ta.showConnectDialog()
		case "Disconnect":
			ta.disconnect()
		case "Reset Terminal":
			ta.terminalView.SetText("")
		}
	})
}

// showHelpMenu shows the Help menu modal
func (ta *TwistApp) showHelpMenu() {
	ta.showMenuModal("Help Menu", []string{
		"About",
		"---",
		"Keyboard Shortcuts",
	}, func(option string) {
		switch option {
		case "About":
			ta.showAboutDialog()
		case "Keyboard Shortcuts":
			ta.showShortcutsDialog()
		}
	})
}

// showMenuModal creates a generic menu modal
func (ta *TwistApp) showMenuModal(title string, options []string, callback func(string)) {
	ta.logger.Printf("Creating menu modal: %s (current mode: %d, modalVisible: %v)", title, ta.inputMode, ta.modalVisible)
	
	// Prevent showing modal if one is already visible (known tview limitation)
	if ta.modalVisible {
		ta.logger.Printf("Modal already visible - ignoring request")
		return
	}
	
	// Filter out separators and create button list
	var buttons []string
	var buttonCallbacks []func()
	
	for _, option := range options {
		if option != "---" {
			buttons = append(buttons, option)
			// Capture option in closure
			opt := option
			buttonCallbacks = append(buttonCallbacks, func() {
				ta.logger.Printf("Menu option selected: %s", opt)
				ta.closeModal()  
				callback(opt)
			})
		}
	}
	
	ta.logger.Printf("Modal will have %d buttons: %v", len(buttons), buttons)
	
	// Create text content
	content := fmt.Sprintf("[yellow::b]%s[white::-]\n\nSelect an option:", title)
	
	// Always create a completely new modal instance (critical for tview modal reuse)
	modal := tview.NewModal().
		SetText(content).
		AddButtons(buttons).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ta.logger.Printf("Modal button pressed: %s (index %d)", buttonLabel, buttonIndex)
			if buttonIndex >= 0 && buttonIndex < len(buttonCallbacks) {
				buttonCallbacks[buttonIndex]()
			}
		})
	
	// Reset modal focus to first button (recommended practice)
	modal.SetFocus(0)
	
	// Clean removal of any existing modal page
	ta.pages.RemovePage("modal")
	ta.logger.Printf("Removed existing modal page, adding new modal instance")
	
	// Add the new modal as overlay page
	ta.pages.AddPage("modal", modal, true, true)
	ta.logger.Printf("Added modal page")
	
	// Set state and focus
	ta.modalVisible = true
	ta.inputMode = InputModeModal
	ta.pages.SwitchToPage("modal")
	ta.app.SetFocus(modal)
	ta.logger.Printf("Switched to modal page and set focus")
	
	ta.logger.Printf("Menu modal should now be visible: %s", title)
}

// showAboutDialog shows an about dialog
func (ta *TwistApp) showAboutDialog() {
	ta.showInfoModal("About", `[yellow::b]Trade Wars 2002 Client[white::-]

A modern TUI client for Trade Wars 2002

Built with:
• Go
• tview
• Custom telnet protocol handler

[gray]Press ESC to close[white]`)
}

// showShortcutsDialog shows keyboard shortcuts
func (ta *TwistApp) showShortcutsDialog() {
	ta.showInfoModal("Keyboard Shortcuts", `[yellow::b]Global Shortcuts[white::-]

[yellow]Alt+F[white] - File menu
[yellow]Alt+E[white] - Edit menu  
[yellow]Alt+V[white] - View menu
[yellow]Alt+T[white] - Terminal menu
[yellow]Alt+H[white] - Help menu

[yellow::b]Terminal Mode[white::-]

[yellow]Shift+Up/Down[white] - Scroll line by line
[yellow]Shift+PgUp/PgDn[white] - Scroll page by page
[yellow]ESC[white] - Return to main menu
[yellow]Ctrl+C[white] - Quit application

[gray]Press ESC to close[white]`)
}

// showInfoModal creates a generic info modal
func (ta *TwistApp) showInfoModal(title, content string) {
	ta.logger.Printf("Creating info modal: %s", title)
	
	text := tview.NewTextView().
		SetDynamicColors(true).
		SetText(content).
		SetTextAlign(tview.AlignCenter)
	
	text.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignCenter).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	
	// Handle ESC to close
	text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		ta.logger.Printf("Info modal key event: Key=%v, Rune=%c", event.Key(), event.Rune())
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			ta.logger.Printf("Closing info modal: %s", title)
			ta.closeModal()
			return nil
		}
		return event
	})
	
	// Create modal layout
	modal := tview.NewGrid().
		SetColumns(0, 60, 0).
		SetRows(0, 20, 0).
		AddItem(text, 1, 1, 1, 1, 0, 0, true)
	
	// Ensure any existing modal is removed first
	ta.pages.RemovePage("modal")
	
	// Add modal as overlay page
	ta.pages.AddPage("modal", modal, true, true)
	ta.inputMode = InputModeModal
	ta.app.SetFocus(text)
	
	ta.logger.Printf("Info modal should now be visible: %s", title)
}