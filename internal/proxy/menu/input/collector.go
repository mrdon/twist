package input

import (
	"strings"

	"twist/internal/debug"
	"twist/internal/proxy/menu/display"
)

// InputCollector handles two-stage input collection for menu operations
type InputCollector struct {
	// Collection state
	isCollecting  bool
	menuName      string
	prompt        string
	buffer        string
	
	// Output function to send data to stream
	sendOutput func(string)
	
	// Completion handlers for different input types
	completionHandlers map[string]CompletionHandler
}

// CompletionHandler processes completed input for specific menu operations
type CompletionHandler func(menuName, value string) error

// NewInputCollector creates a new input collector
func NewInputCollector(sendOutput func(string)) *InputCollector {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in NewInputCollector: %v", r)
		}
	}()

	return &InputCollector{
		isCollecting:       false,
		menuName:          "",
		prompt:            "",
		buffer:            "",
		sendOutput:        sendOutput,
		completionHandlers: make(map[string]CompletionHandler),
	}
}

// RegisterCompletionHandler registers a handler for a specific menu operation
func (ic *InputCollector) RegisterCompletionHandler(menuName string, handler CompletionHandler) {
	ic.completionHandlers[menuName] = handler
}

// StartCollection begins collecting input for a menu operation
func (ic *InputCollector) StartCollection(menuName, prompt string) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in StartCollection: %v", r)
		}
	}()

	debug.Log("Starting input collection for menu: %s, prompt: %s", menuName, prompt)
	ic.isCollecting = true
	ic.menuName = menuName
	ic.prompt = prompt
	ic.buffer = "" // Clear any previous input
	
	// Display the input prompt
	if prompt != "" {
		ic.sendOutput("\r\n" + display.FormatInputPrompt(prompt))
	} else {
		ic.sendOutput("\r\n" + display.FormatInputPrompt("Enter value"))
	}
	
	// Show help for input collection
	ic.sendOutput("(Enter value, or '\\' to cancel)\r\n")
}

// IsCollecting returns whether input collection is active
func (ic *InputCollector) IsCollecting() bool {
	return ic.isCollecting
}

// GetCurrentMenu returns the menu name being collected for
func (ic *InputCollector) GetCurrentMenu() string {
	return ic.menuName
}

// HandleInput processes user input during collection
func (ic *InputCollector) HandleInput(input string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in HandleInput: %v", r)
		}
	}()

	if !ic.isCollecting {
		return nil
	}

	// Check for special keys first (before trimming)
	if input == "\r" || input == "\n" || input == "\r\n" || input == "" {
		// Enter pressed - complete collection with current buffer
		result := ic.buffer
		ic.buffer = ""
		debug.Log("Enter pressed, completing input collection with: '%s'", result)
		return ic.completeCollection(result)
	}

	// Handle special control sequences
	trimmedInput := strings.TrimSpace(input)
	switch trimmedInput {
	case "\\", "\\quit":
		// Escape input collection (only backslash commands)
		ic.sendOutput("Input cancelled.\r\n")
		ic.buffer = ""
		ic.cancelCollection()
		return nil
	case "?":
		// Show help for input collection mode
		ic.showCollectionHelp()
		return nil
	}

	// Handle backspace (remove last character from buffer)
	if input == "\b" || input == "\x7f" {
		if len(ic.buffer) > 0 {
			ic.buffer = ic.buffer[:len(ic.buffer)-1]
			debug.Log("Backspace pressed, buffer now: '%s'", ic.buffer)
		}
		return nil
	}

	// Accumulate printable characters in the buffer and echo them
	if len(trimmedInput) > 0 {
		ic.buffer += trimmedInput
		debug.Log("Added '%s' to buffer, buffer now: '%s'", trimmedInput, ic.buffer)
		// Echo the character to provide visual feedback
		ic.sendOutput(trimmedInput)
	}

	return nil
}

// showCollectionHelp displays help for input collection mode
func (ic *InputCollector) showCollectionHelp() {
	ic.sendOutput("\r\nInput Collection Help:\r\n")
	ic.sendOutput("- Type your value and press Enter to submit\r\n")
	ic.sendOutput("- Press Enter alone to submit empty value\r\n")
	ic.sendOutput("- Press '\\' to cancel input collection\r\n")
	ic.sendOutput("Current input: " + ic.buffer + "\r\n")
}

// completeCollection completes the input collection and calls the appropriate handler
func (ic *InputCollector) completeCollection(value string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in completeCollection: %v", r)
		}
	}()

	menuName := ic.menuName
	ic.exitCollection()
	
	if handler, exists := ic.completionHandlers[menuName]; exists {
		return handler(menuName, value)
	}
	
	// Default behavior - show success message
	if value != "" {
		ic.sendOutput(display.FormatSuccessMessage("Value set: " + value))
	} else {
		ic.sendOutput(display.FormatSuccessMessage("Value cleared"))
	}
	
	return nil
}

// cancelCollection cancels input collection without processing
func (ic *InputCollector) cancelCollection() {
	ic.exitCollection()
}

// exitCollection exits input collection mode
func (ic *InputCollector) exitCollection() {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in exitCollection: %v", r)
		}
	}()

	debug.Log("Exiting input collection mode")
	ic.isCollecting = false
	ic.menuName = ""
	ic.prompt = ""
	ic.buffer = ""
}

// GetBuffer returns the current input buffer (for testing/debugging)
func (ic *InputCollector) GetBuffer() string {
	return ic.buffer
}

// SetBuffer sets the input buffer (for pre-filling input)
func (ic *InputCollector) SetBuffer(value string) {
	ic.buffer = value
}