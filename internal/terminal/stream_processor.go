package terminal

import (
	"twist/internal/ansi"
	"unicode/utf8"
)

// StreamProcessor handles ANSI escape sequences and streams processed output
type StreamProcessor struct {
	// ANSI converter for color conversion
	ansiConverter *ansi.ColorConverter
	
	// Current color state
	currentColorTag string
	
	// Fixed-size buffer for streaming data processing
	buffer    [8192]byte
	bufferLen int
	
	// Output callback - receives processed text with tview color tags
	onOutput func(text string)
}

// NewStreamProcessor creates a new streaming ANSI processor
func NewStreamProcessor(converter *ansi.ColorConverter, outputCallback func(string)) *StreamProcessor {
	return &StreamProcessor{
		ansiConverter:   converter,
		currentColorTag: "[#c0c0c0:#000000]", // Default color tag
		onOutput:        outputCallback,
	}
}

// Write processes incoming data and streams processed output
func (sp *StreamProcessor) Write(data []byte) {
	sp.processDataWithANSI(data)
}

// processDataWithANSI processes input data with fixed-size buffer
func (sp *StreamProcessor) processDataWithANSI(data []byte) {
	for len(data) > 0 {
		// How much space is left in buffer?
		spaceLeft := len(sp.buffer) - sp.bufferLen
		
		// How much can we add this iteration?
		toAdd := len(data)
		if toAdd > spaceLeft {
			toAdd = spaceLeft
		}
		
		// Add data to buffer
		copy(sp.buffer[sp.bufferLen:], data[:toAdd])
		sp.bufferLen += toAdd
		data = data[toAdd:]
		
		// Process everything we can from the buffer
		consumed := 0
		for consumed < sp.bufferLen {
			// Try to consume starting from current position
			bytesConsumed := sp.tryConsumeSequence(sp.buffer[consumed:sp.bufferLen])
			
			if bytesConsumed > 0 {
				// Successfully consumed some bytes
				consumed += bytesConsumed
			} else {
				// Couldn't consume anything - incomplete sequence
				break
			}
		}
		
		// Remove consumed data from buffer, keep unconsumed data
		if consumed > 0 {
			copy(sp.buffer[:], sp.buffer[consumed:sp.bufferLen])
			sp.bufferLen -= consumed
		}
		
		// Safety check: if buffer is full and we couldn't consume anything, force consume one character
		if sp.bufferLen == len(sp.buffer) && consumed == 0 {
			char, size := utf8.DecodeRune(sp.buffer[:])
			sp.processRune(char)
			copy(sp.buffer[:], sp.buffer[size:sp.bufferLen])
			sp.bufferLen -= size
		}
	}
}

// tryConsumeSequence attempts to consume data starting at the beginning of the slice
func (sp *StreamProcessor) tryConsumeSequence(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	
	if data[0] == '\x1b' {
		// Try to consume escape sequence
		if len(data) < 2 {
			return 0 // Need more data
		}
		
		if data[1] == '[' {
			// ANSI escape sequence - find terminator
			i := 2
			for i < len(data) {
				c := data[i]
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					// Found terminator - consume complete sequence
					i++ // include terminator
					sequence := data[0:i]
					sp.processANSISequence(sequence)
					return i
				}
				i++
			}
			// No terminator found - incomplete sequence
			return 0
		} else {
			// \x1b followed by non-[ - treat as regular character
			char, size := utf8.DecodeRune(data)
			sp.processRune(char)
			return size
		}
	} else {
		// Regular character
		char, size := utf8.DecodeRune(data)
		if char == utf8.RuneError && size == 1 {
			return 1 // skip invalid byte
		}
		sp.processRune(char)
		return size
	}
}

// processANSISequence handles a complete ANSI escape sequence
func (sp *StreamProcessor) processANSISequence(sequence []byte) {
	if len(sequence) < 3 || sequence[0] != '\x1b' || sequence[1] != '[' {
		return
	}

	// Extract the parameter part (everything between [ and the final letter)
	params := string(sequence[2 : len(sequence)-1])
	command := sequence[len(sequence)-1]

	switch command {
	case 'm': // Color/attribute commands
		if sp.ansiConverter == nil {
			return
		}

		// Convert ANSI parameters to tview color tag
		colorTag := sp.ansiConverter.ConvertANSIParams(params)

		// Only output color change if color actually changed
		if colorTag != sp.currentColorTag {
			sp.currentColorTag = colorTag
			// Stream the color tag to output
			if sp.onOutput != nil {
				sp.onOutput(colorTag)
			}
		}

	case 'H', 'f': // Cursor position - ignore for streaming
	case 'A', 'B', 'C', 'D': // Cursor movement - ignore for streaming  
	case 'J': // Erase display - ignore for streaming
	case 'K': // Erase line - ignore for streaming
	default:
		// Unknown command - ignore
	}
}

// processRune handles a single rune of input
func (sp *StreamProcessor) processRune(char rune) {
	// Skip escape characters - they're handled by processDataWithANSI
	if char == 0x1B {
		return
	}

	// Handle control characters
	switch char {
	case '\r': // Carriage return
		if sp.onOutput != nil {
			sp.onOutput("\r")
		}
	case '\n': // Line feed
		if sp.onOutput != nil {
			sp.onOutput("\n")
		}
	case '\t': // Tab
		if sp.onOutput != nil {
			sp.onOutput("\t")
		}
	case '\b': // Backspace - ignore for streaming
	case 0x07: // Bell - ignore
	default:
		// Printable character - stream it
		if char >= 32 && sp.onOutput != nil {
			sp.onOutput(string(char))
		}
	}
}