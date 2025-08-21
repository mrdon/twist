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
	app               *tview.Application
	inputMode         InputMode
	modalVisible      bool
	isDropdownVisible func() bool // Function to check if dropdown is visible

	// Callbacks
	onConnect              func(string)
	onDisconnect           func()
	onExit                 func()
	onShowModal            func(string, []string, func(string))
	onShowDropdown         func(string, []string, func(string))
	onCloseModal           func()
	onSendCommand          func(string)
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

// SetDropdownVisibilityChecker sets the function to check if dropdown is visible
func (ih *InputHandler) SetDropdownVisibilityChecker(isDropdownVisible func() bool) {
	ih.isDropdownVisible = isDropdownVisible
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
		// Don't handle Enter in the input handler - let the focused component handle it
		// This allows menus, modals, and terminal input to handle Enter naturally
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
	ih.showMenu("Session")
}

func (ih *InputHandler) showViewMenu() {
	ih.showMenu("View")
}

func (ih *InputHandler) showTerminalMenu() {
	ih.showMenu("Terminal")
}

func (ih *InputHandler) showHelpMenu() {
	ih.showMenu("Help")
}

// showMenu displays any menu using the centralized menu system
func (ih *InputHandler) showMenu(menuName string) {
	if ih.onShowDropdown != nil {
		// Use empty options - showDropdownMenu in app.go will get the real items from the menu manager
		ih.onShowDropdown(menuName, []string{}, func(selected string) {
			// This callback is not used - showDropdownMenu handles everything
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
