package handlers

import (
	"log"
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
	logger        *log.Logger
	inputMode     InputMode
	modalVisible  bool
	
	// Callbacks
	onConnect     func(string)
	onDisconnect  func()
	onExit        func()
	onShowModal   func(string, []string, func(string))
	onCloseModal  func()
	onSendCommand func(string)
}

// NewInputHandler creates a new input handler
func NewInputHandler(app *tview.Application, logger *log.Logger) *InputHandler {
	return &InputHandler{
		app:       app,
		logger:    logger,
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
	ih.logger.Printf("Key event: %v, mode: %d, modal: %t", event.Key(), ih.inputMode, ih.modalVisible)
	
	// Global keys that work in all modes
	if event.Key() == tcell.KeyF10 {
		if ih.onExit != nil {
			ih.onExit()
		}
		return nil
	}
	
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
	switch event.Key() {
	case tcell.KeyF1:
		ih.showFileMenu()
		return nil
	case tcell.KeyF2:
		ih.showEditMenu()
		return nil
	case tcell.KeyF3:
		ih.showViewMenu()
		return nil
	case tcell.KeyF4:
		ih.showTerminalMenu()
		return nil
	case tcell.KeyF5:
		ih.showHelpMenu()
		return nil
	case tcell.KeyF8:
		ih.showConnectDialog()
		return nil
	case tcell.KeyF9:
		if ih.onDisconnect != nil {
			ih.onDisconnect()
		}
		return nil
	case tcell.KeyTab:
		ih.SetInputMode(InputModeTerminal)
		return nil
	}
	
	return event
}

// handleTerminalInput handles input in terminal mode
func (ih *InputHandler) handleTerminalInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab:
		ih.SetInputMode(InputModeMenu)
		return nil
	case tcell.KeyEscape:
		ih.SetInputMode(InputModeMenu)
		return nil
	case tcell.KeyEnter:
		// Handle command input
		return event
	}
	
	// Pass other keys to terminal for input handling
	if event.Key() == tcell.KeyRune {
		char := string(event.Rune())
		if ih.onSendCommand != nil {
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
func (ih *InputHandler) showFileMenu() {
	if ih.onShowModal != nil {
		options := []string{"New Script", "Open Script", "Save Script", "Exit"}
		ih.onShowModal("File Menu", options, func(selected string) {
			ih.logger.Printf("File menu selection: %s", selected)
		})
	}
}

func (ih *InputHandler) showEditMenu() {
	if ih.onShowModal != nil {
		options := []string{"Cut", "Copy", "Paste", "Find", "Replace"}
		ih.onShowModal("Edit Menu", options, func(selected string) {
			ih.logger.Printf("Edit menu selection: %s", selected)
		})
	}
}

func (ih *InputHandler) showViewMenu() {
	if ih.onShowModal != nil {
		options := []string{"Zoom In", "Zoom Out", "Full Screen", "Panels"}
		ih.onShowModal("View Menu", options, func(selected string) {
			ih.logger.Printf("View menu selection: %s", selected)
		})
	}
}

func (ih *InputHandler) showTerminalMenu() {
	if ih.onShowModal != nil {
		options := []string{"Clear", "Scroll Up", "Scroll Down", "Copy Selection"}
		ih.onShowModal("Terminal Menu", options, func(selected string) {
			ih.logger.Printf("Terminal menu selection: %s", selected)
		})
	}
}

func (ih *InputHandler) showHelpMenu() {
	if ih.onShowModal != nil {
		options := []string{"Keyboard Shortcuts", "About", "User Manual"}
		ih.onShowModal("Help Menu", options, func(selected string) {
			ih.logger.Printf("Help menu selection: %s", selected)
		})
	}
}

func (ih *InputHandler) showConnectDialog() {
	if ih.onConnect != nil {
		ih.onConnect("twgs.geekm0nkey.com:23") // Default address
	}
}