package input

import (
	"runtime"
	"strings"

	"twist/internal/log"
)

// debugSendOutput logs output with stack trace for debugging
func debugSendOutput(output string, sendFunc func(string)) {
	// Get stack trace
	pc := make([]uintptr, 10)
	n := runtime.Callers(2, pc) // Skip runtime.Callers and this function
	frames := runtime.CallersFrames(pc[:n])

	log.Info("INPUT_COLLECTOR sendOutput", "output", output)
	for {
		frame, more := frames.Next()
		log.Info("  at", "file", frame.File, "line", frame.Line, "function", frame.Function)
		if !more {
			break
		}
	}

	sendFunc(output)
}

// InputCollector handles two-stage input collection for menu operations
type InputCollector struct {
	// Collection state
	isCollecting bool
	menuName     string
	prompt       string
	buffer       string

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
			log.Error("PANIC in NewInputCollector", "error", r)
		}
	}()

	return &InputCollector{
		isCollecting:       false,
		menuName:           "",
		prompt:             "",
		buffer:             "",
		sendOutput:         sendOutput,
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
			log.Error("PANIC in StartCollection", "error", r)
		}
	}()

	ic.isCollecting = true
	ic.menuName = menuName
	ic.prompt = prompt
	ic.buffer = "" // Clear any previous input

	// Display the input prompt (scripts handle their own prompting)
	if prompt != "" && !strings.HasPrefix(menuName, "SCRIPT_INPUT_") {
		debugSendOutput("\r\n"+prompt+"\r\n", ic.sendOutput)
	}

	// Show help for input collection (but not for script inputs - TWX doesn't show this)
	if !strings.HasPrefix(menuName, "SCRIPT_INPUT_") {
		debugSendOutput("(Enter value, or '\\' to cancel)\r\n", ic.sendOutput)
	}
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
			log.Error("PANIC in HandleInput", "error", r)
		}
	}()

	if !ic.isCollecting {
		return nil
	}

	// Check if input ends with Enter key and extract the value
	var actualValue string
	var hasEnter bool

	if strings.HasSuffix(input, "\r\n") {
		actualValue = strings.TrimSuffix(input, "\r\n")
		hasEnter = true
	} else if strings.HasSuffix(input, "\r") {
		actualValue = strings.TrimSuffix(input, "\r")
		hasEnter = true
	} else if strings.HasSuffix(input, "\n") {
		actualValue = strings.TrimSuffix(input, "\n")
		hasEnter = true
	} else if input == "" {
		actualValue = ""
		hasEnter = true
	} else {
		actualValue = input
		hasEnter = false
	}

	if hasEnter {
		// Complete collection with the value (add to existing buffer first)
		result := ic.buffer + actualValue
		ic.buffer = ""
		return ic.completeCollection(result)
	}

	// Handle backspace/delete keys
	if input == "\b" || input == "\x7f" || input == "\x08" {
		if len(ic.buffer) > 0 {
			ic.buffer = ic.buffer[:len(ic.buffer)-1]
			// Send backspace sequence to terminal to visually remove character
			debugSendOutput("\b \b", ic.sendOutput)
		}
		return nil
	}

	// Handle escape sequences and special commands
	trimmedInput := strings.TrimSpace(input)
	if trimmedInput == "\\" || trimmedInput == "\\quit" {
		// Escape input collection
		debugSendOutput("Input cancelled.\r\n", ic.sendOutput)
		ic.buffer = ""
		ic.cancelCollection()
		return nil
	}

	if trimmedInput == "?" {
		// Show help for input collection mode
		ic.showCollectionHelp()
		return nil
	}

	// Only process printable characters (ignore empty input and control chars)
	if len(input) > 0 && input[0] >= 32 && input[0] < 127 {
		ic.buffer += input
		// Echo the character back for visual feedback
		debugSendOutput(input, ic.sendOutput)
	}

	return nil
}

// showCollectionHelp displays help for input collection mode
func (ic *InputCollector) showCollectionHelp() {
	debugSendOutput("\r\nInput Collection Help:\r\n", ic.sendOutput)
	debugSendOutput("- Type your value and press Enter to submit\r\n", ic.sendOutput)
	debugSendOutput("- Press Enter alone to submit empty value\r\n", ic.sendOutput)
	debugSendOutput("- Press '\\' to cancel input collection\r\n", ic.sendOutput)
	debugSendOutput("Current input: "+ic.buffer+"\r\n", ic.sendOutput)
}

// completeCollection completes the input collection and calls the appropriate handler
func (ic *InputCollector) completeCollection(value string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC in completeCollection", "error", r)
		}
	}()

	menuName := ic.menuName
	ic.exitCollection()

	if handler, exists := ic.completionHandlers[menuName]; exists {
		return handler(menuName, value)
	}
	// Default behavior - show success message (but not for script inputs - TWX doesn't show this)
	if !strings.HasPrefix(menuName, "SCRIPT_INPUT") {
		if value != "" {
			debugSendOutput("Value set: "+value+"\r\n", ic.sendOutput)
		} else {
			debugSendOutput("Value cleared\r\n", ic.sendOutput)
		}
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
			log.Error("PANIC in exitCollection", "error", r)
		}
	}()

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
