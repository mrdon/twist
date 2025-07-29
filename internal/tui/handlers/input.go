package handlers

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// InputMode represents the current input mode
type InputMode int

const (
	InputModeMenu InputMode = iota
	InputModeTerminal
	InputModeModal
)

// InputHandler manages input handling for the application
type InputHandler struct {
	app           *tview.Application
	inputMode     InputMode
	modalVisible  bool
	
	// Callbacks
	onConnect     func(string)
	onDisconnect  func()
	onExit        func()
	onShowModal   func(string, []string, func(string))
	onShowDropdown func(string, []string, func(string))
	onCloseModal  func()
	onSendCommand func(string)
	onShowConnectionDialog func()
}

// NewInputHandler creates a new input handler
func NewInputHandler(app *tview.Application) *InputHandler {
	return &InputHandler{
		app:       app,
		inputMode: InputModeMenu,
	}
}

// SetCallbacks sets the callback functions
func (ih *InputHandler) SetCallbacks(
	onConnect func(string),
	onDisconnect func(),
	onExit func(),
	onShowModal func(string, []string, func(string)),
	onCloseModal func(),
	onSendCommand func(string),
) {
	ih.onConnect = onConnect
	ih.onDisconnect = onDisconnect
	ih.onExit = onExit
	ih.onShowModal = onShowModal
	ih.onCloseModal = onCloseModal
	ih.onSendCommand = onSendCommand
}

// SetConnectionDialogCallback sets the callback for showing connection dialog
func (ih *InputHandler) SetConnectionDialogCallback(onShowConnectionDialog func()) {
	ih.onShowConnectionDialog = onShowConnectionDialog
}

// SetDropdownCallback sets the callback for dropdown menus
func (ih *InputHandler) SetDropdownCallback(onShowDropdown func(string, []string, func(string))) {
	ih.onShowDropdown = onShowDropdown
}

// SetInputMode sets the current input mode
func (ih *InputHandler) SetInputMode(mode InputMode) {
	ih.inputMode = mode
}

// SetModalVisible sets the modal visibility state
func (ih *InputHandler) SetModalVisible(visible bool) {
	ih.modalVisible = visible
}

// HandleKeyEvent handles key events based on current input mode
func (ih *InputHandler) HandleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	
	// Modal mode handling
	if ih.modalVisible {
		return ih.handleModalInput(event)
	}
	
	// Mode-specific handling
	switch ih.inputMode {
	case InputModeMenu:
		return ih.handleMenuInput(event)
	case InputModeTerminal:
		return ih.handleTerminalInput(event)
	}
	
	return event
}

// handleMenuInput handles input in menu mode
func (ih *InputHandler) handleMenuInput(event *tcell.EventKey) *tcell.EventKey {
	// Handle Alt+Letter combinations
	if event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModAlt != 0 {
		switch event.Rune() {
		case 's', 'S':
			ih.showSessionMenu()
			return nil
		case 'e', 'E':
			ih.showEditMenu()
			return nil
		case 'v', 'V':
			ih.showViewMenu()
			return nil
		case 't', 'T':
			ih.showTerminalMenu()
			return nil
		case 'h', 'H':
			ih.showHelpMenu()
			return nil
		case 'c', 'C':
			ih.showConnectDialog()
			return nil
		case 'd', 'D':
			if ih.onDisconnect != nil {
				ih.onDisconnect()
			}
			return nil
		}
	}
	
	switch event.Key() {
	case tcell.KeyTab:
		ih.SetInputMode(InputModeTerminal)
		return nil
	}
	
	return event
}

// handleTerminalInput handles input in terminal mode
func (ih *InputHandler) handleTerminalInput(event *tcell.EventKey) *tcell.EventKey {
	
	// Handle Alt+Letter combinations for menu access
	if event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModAlt != 0 {
		switch event.Rune() {
		case 's', 'S':
			ih.showSessionMenu()
			return nil
		case 'e', 'E':
			ih.showEditMenu()
			return nil
		case 'v', 'V':
			ih.showViewMenu()
			return nil
		case 't', 'T':
			ih.showTerminalMenu()
			return nil
		case 'h', 'H':
			ih.showHelpMenu()
			return nil
		case 'c', 'C':
			ih.showConnectDialog()
			return nil
		case 'd', 'D':
			if ih.onDisconnect != nil {
				ih.onDisconnect()
			}
			return nil
		}
	}
	
	// Don't send Control key combinations (except let tview handle them)
	if event.Modifiers()&tcell.ModCtrl != 0 {
		return event
	}
	
	switch event.Key() {
	case tcell.KeyTab:
		ih.SetInputMode(InputModeMenu)
		return nil
	case tcell.KeyEscape:
		ih.SetInputMode(InputModeMenu)
		return nil
	case tcell.KeyEnter:
		// Send carriage return to terminal only if no modals are open
		if !ih.modalVisible && ih.onSendCommand != nil {
			ih.onSendCommand("\r")
		}
		return event
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		// Send backspace control character only if no modals are open
		if !ih.modalVisible && ih.onSendCommand != nil {
			ih.onSendCommand("\b")
		}
		return event
	case tcell.KeyDelete:
		// Send delete escape sequence only if no modals are open
		if !ih.modalVisible && ih.onSendCommand != nil {
			ih.onSendCommand("\x1b[3~")
		}
		return event
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyRight, tcell.KeyLeft, tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn:
		// Don't send navigation keys to terminal - let tview handle them for UI navigation
		return event
	}
	
	// Pass other keys to terminal for input handling only if no modals are open
	if event.Key() == tcell.KeyRune {
		if !ih.modalVisible && ih.onSendCommand != nil {
			char := string(event.Rune())
			ih.onSendCommand(char)
		}
	}
	
	return event
}

// handleModalInput handles input when a modal is visible
func (ih *InputHandler) handleModalInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		if ih.onCloseModal != nil {
			ih.onCloseModal()
		}
		return nil
	}
	
	return event
}

// Menu display functions
func (ih *InputHandler) showSessionMenu() {
	// We need a special callback that doesn't auto-close when Connect is selected
	// This will be handled by the app's navigation logic instead
	if ih.onShowDropdown != nil {
		options := []string{"Connect", "Disconnect", "Quit"}
		ih.onShowDropdown("Session", options, func(selected string) {
			// Note: The actual handling happens in app.go showDropdownMenu
			// This callback gets overridden by the showDropdownMenu auto-close behavior
		})
	} else {
	}
}

func (ih *InputHandler) showEditMenu() {
	if ih.onShowDropdown != nil {
		options := []string{"Cut", "Copy", "Paste", "Find", "Replace"}
		ih.onShowDropdown("Edit", options, func(selected string) {
		})
	}
}

func (ih *InputHandler) showViewMenu() {
	if ih.onShowDropdown != nil {
		options := []string{"Scripts", "Zoom In", "Zoom Out", "Full Screen", "Panels"}
		ih.onShowDropdown("View", options, func(selected string) {
			// Handle Scripts selection - for now just log
			if selected == "Scripts" {
			}
		})
	}
}

func (ih *InputHandler) showTerminalMenu() {
	if ih.onShowDropdown != nil {
		options := []string{"Clear", "Scroll Up", "Scroll Down", "Copy Selection"}
		ih.onShowDropdown("Terminal", options, func(selected string) {
		})
	}
}

func (ih *InputHandler) showHelpMenu() {
	if ih.onShowDropdown != nil {
		options := []string{"Keyboard Shortcuts", "About", "User Manual"}
		ih.onShowDropdown("Help", options, func(selected string) {
		})
	}
}

func (ih *InputHandler) showConnectDialog() {
	if ih.onShowConnectionDialog != nil {
		ih.onShowConnectionDialog()
	} else {
		// Fallback to direct connection
		if ih.onConnect != nil {
			ih.onConnect("twgs.geekm0nkey.com:23")
		} else {
		}
	}
}