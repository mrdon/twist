package terminal

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"twist/internal/ansi"
	"unicode/utf8"
)

// Cell represents a single character cell in the terminal
type Cell struct {
	Char          rune
	ForegroundHex string // Hex color code (e.g., "#ff0000")
	BackgroundHex string // Hex color code (e.g., "#000000")
	Bold          bool
	Underline     bool
	Reverse       bool
}

// Terminal represents a virtual terminal with a screen buffer
type Terminal struct {
	width     int
	height    int
	buffer    [][]Cell
	cursorX   int
	cursorY   int
	scrollTop int // Top of scrollable region
	scrollBot int // Bottom of scrollable region

	// Scrollback buffer (circular buffer)
	scrollback [][]Cell
	scrollHead int
	scrollSize int

	// Screen history for MUD-style scrolling
	screenHistory     [][][]Cell // Array of complete screens
	screenHistoryHead int
	maxScreenHistory  int

	// Current text attributes (stored as hex colors now)
	currentFgHex string
	currentBgHex string
	bold         bool
	underline    bool
	reverse      bool

	// ANSI converter for immediate color conversion
	ansiConverter ansi.ANSIConverter


	// Partial escape sequence buffer for streaming
	partialEscape []byte

	// Update tracking
	dirty      bool
	lastUpdate time.Time

	// Logger
	logger *log.Logger

	// Update callback for UI notifications
	onUpdate func()

	// Incremental update tracking
	newDataBuffer []byte     // Buffer for new data since last UI update
	newDataMutex  sync.Mutex // Protect concurrent access to newDataBuffer
}

// NewTerminal creates a new terminal buffer
func NewTerminal(width, height int) *Terminal {
	return NewTerminalWithConverter(width, height, nil)
}

// NewTerminalWithConverter creates a new terminal buffer with an ANSI converter
func NewTerminalWithConverter(width, height int, converter ansi.ANSIConverter) *Terminal {
	return NewTerminalWithConverterAndLogger(width, height, converter, nil)
}

// NewTerminalWithConverterAndLogger creates a new terminal buffer with an ANSI converter and shared logger
func NewTerminalWithConverterAndLogger(width, height int, converter ansi.ANSIConverter, pvpLogger *log.Logger) *Terminal {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	logger := log.New(logFile, "[TERM] ", log.LstdFlags|log.Lshortfile)
	logger.Printf("Terminal initialized %dx%d", width, height)

	t := &Terminal{
		width:         width,
		height:        height,
		buffer:        make([][]Cell, height),
		scrollback:    make([][]Cell, 1000), // 1000 lines of scrollback
		scrollSize:    1000,
		scrollTop:     0,
		scrollBot:     height - 1,
		currentFgHex:  "#c0c0c0", // Default white
		currentBgHex:  "#000000", // Default black
		logger:        logger,
		ansiConverter: converter,

		// Screen history for MUD scrolling
		screenHistory:    make([][][]Cell, 100), // Store 100 complete screens
		maxScreenHistory: 100,
	}

	// Initialize buffer with default cells
	for i := range t.buffer {
		t.buffer[i] = make([]Cell, width)
		for j := range t.buffer[i] {
			t.buffer[i][j] = Cell{Char: ' ', ForegroundHex: "#c0c0c0", BackgroundHex: "#000000"}
		}
	}

	// Initialize scrollback
	for i := range t.scrollback {
		t.scrollback[i] = make([]Cell, width)
	}

	t.clear()
	return t
}

// Resize changes the terminal dimensions
func (t *Terminal) Resize(width, height int) {
	t.logger.Printf("Resizing terminal from %dx%d to %dx%d", t.width, t.height, width, height)

	oldBuffer := t.buffer
	t.width = width
	t.height = height
	t.buffer = make([][]Cell, height)
	t.scrollBot = height - 1

	// Initialize new buffer with default cells
	for i := range t.buffer {
		t.buffer[i] = make([]Cell, width)
		for j := range t.buffer[i] {
			t.buffer[i][j] = Cell{Char: ' ', ForegroundHex: "#ffffff", BackgroundHex: "#000000"}
		}
	}

	// Copy old content (best effort)
	copyHeight := height
	if len(oldBuffer) < copyHeight {
		copyHeight = len(oldBuffer)
	}

	for y := 0; y < copyHeight; y++ {
		copyWidth := width
		if len(oldBuffer[y]) < copyWidth {
			copyWidth = len(oldBuffer[y])
		}
		copy(t.buffer[y][:copyWidth], oldBuffer[y][:copyWidth])
	}

	// Adjust cursor position
	if t.cursorX >= width {
		t.cursorX = width - 1
	}
	if t.cursorY >= height {
		t.cursorY = height - 1
	}

	t.dirty = true
}

// extractContext returns 10 chars before and after the target string
func extractContext(data []byte, target string) string {
	str := string(data)
	index := strings.Index(str, target)
	if index == -1 {
		return ""
	}

	start := index - 10
	if start < 0 {
		start = 0
	}

	end := index + len(target) + 10
	if end > len(str) {
		end = len(str)
	}

	context := str[start:end]
	// Escape ANSI for readability
	context = strings.ReplaceAll(context, "\x1b", "\\x1b")
	return context
}

// Write processes incoming data and updates the terminal buffer
func (t *Terminal) Write(data []byte) {
	t.logger.Printf("Terminal received %d bytes: %q", len(data), string(data))

	// Track NO PVP hex conversion - create shared logger if needed
	var pvpLogger *log.Logger
	if strings.Contains(string(data), "Select") {
		pvpLogFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			pvpLogger = log.New(pvpLogFile, "[PVP] ", log.LstdFlags|log.Lshortfile)
			start := strings.Index(string(data), "Select")
			if start < 0 {
				start = 0
			}
			end := strings.Index(string(data), "Select") + 120
			if end > len(data) {
				end = len(data)
			}
			context := string(data[start:end])
			context = strings.ReplaceAll(context, "\x1b", "\\x1b")
			pvpLogger.Printf("STAGE 4 - TERMINAL: %s", context)
			pvpLogFile.Close()
		}
	}

	// Store new data for incremental updates
	t.newDataMutex.Lock()
	t.newDataBuffer = append(t.newDataBuffer, data...)
	t.newDataMutex.Unlock()

	// Log raw input to separate file for debugging
	rawLogFile, err := os.OpenFile("raw_input.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		rawLogFile.WriteString(fmt.Sprintf("=== %s ===\n", time.Now().Format("15:04:05.000")))
		rawLogFile.WriteString(fmt.Sprintf("Raw bytes (%d): %q\n", len(data), string(data)))
		rawLogFile.WriteString(fmt.Sprintf("Hex: %x\n", data))
		rawLogFile.WriteString("---\n")
		rawLogFile.Close()
	}

	// Process ANSI escape sequences first, then handle remaining characters
	t.processDataWithANSI(data)

	t.dirty = true
	t.lastUpdate = time.Now()

	// Notify UI of update
	if t.onUpdate != nil {
		t.logger.Printf("Calling onUpdate callback")
		t.onUpdate()
	} else {
		t.logger.Printf("No onUpdate callback set")
	}
}


// processDataWithANSI processes input data by handling ANSI sequences at the correct positions
func (t *Terminal) processDataWithANSI(data []byte) {
	// Debug log to verify this method is being called
	if strings.Contains(string(data), "Disconnect") {
		debugFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			debugLogger := log.New(debugFile, "[NEW] ", log.LstdFlags|log.Lshortfile)
			debugLogger.Printf("NEW ANSI PROCESSING: %q", string(data))
			debugFile.Close()
		}
	}

	// Combine any partial escape from previous write with new data
	fullData := append(t.partialEscape, data...)
	if len(t.partialEscape) > 0 {
		t.logger.Printf("Combined partial escape %q with new data %q -> %q", 
			string(t.partialEscape), string(data), string(fullData))
	}
	t.partialEscape = t.partialEscape[:0] // Clear the buffer

	// Process data sequentially, but handle ANSI sequences completely before continuing
	i := 0
	for i < len(fullData) {
		if fullData[i] == '\x1b' {
			// Check if we have a complete \x1b[ sequence
			if i+1 < len(fullData) && fullData[i+1] == '[' {
			// Found ANSI escape sequence - find the end and process it
			start := i
			t.logger.Printf("*** ESCAPE START: Found \\x1b[ at pos %d ***", start)
			i += 2 // Skip \x1b[

			// Debug logging for parsing
			if strings.Contains(string(data), "NO") && strings.Contains(string(data), "PVP") {
				debugFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err == nil {
					debugLogger := log.New(debugFile, "[PARSE] ", log.LstdFlags|log.Lshortfile)
					debugLogger.Printf("Found ANSI start at pos %d: %q", start, string(fullData[start:start+5]))
					debugFile.Close()
				}
			}

			// Find the end of the escape sequence (letter that terminates it)
			for i < len(fullData) && !((fullData[i] >= 'a' && fullData[i] <= 'z') || (fullData[i] >= 'A' && fullData[i] <= 'Z')) {
				i++
			}

			if i < len(fullData) {
				// Include the terminating letter
				i++
				// Process the complete ANSI sequence immediately
				sequence := fullData[start:i]

				// Debug logging for all sequences
				t.logger.Printf("Processing complete ANSI sequence: %q (hex: %x)", string(sequence), sequence)

				// Debug logging for each sequence found
				if strings.Contains(string(data), "NO") && strings.Contains(string(data), "PVP") {
					debugFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
					if err == nil {
						debugLogger := log.New(debugFile, "[PARSE] ", log.LstdFlags|log.Lshortfile)
						debugLogger.Printf("Processing sequence: %q (hex: %x)", string(sequence), sequence)
						debugFile.Close()
					}
				}

				t.processANSISequence(sequence)
			} else {
				// Incomplete escape sequence at end of data - save for next write
				incomplete := fullData[start:]
				t.partialEscape = append(t.partialEscape, incomplete...)
				t.logger.Printf("*** INCOMPLETE ANSI: Saved %q (hex: %x) for next write ***", string(incomplete), incomplete)
				break
			}
			} else {
				// We have \x1b but not \x1b[ - could be incomplete \x1b at end or other escape
				if i+1 >= len(fullData) {
					// \x1b at very end of data - save for next write
					t.partialEscape = append(t.partialEscape, fullData[i:]...)
					t.logger.Printf("*** INCOMPLETE ESC: Saved \\x1b at end of data ***")
					break
				} else {
					// \x1b followed by something other than [ - treat as regular character
					char, size := utf8.DecodeRune(fullData[i:])
					t.processRune(char)
					i += size
				}
			}
		} else {
			// Regular character - decode UTF-8 properly
			char, size := utf8.DecodeRune(fullData[i:])
			if char == utf8.RuneError && size == 1 {
				// Invalid UTF-8, skip this byte
				i++
				continue
			}

			// Debug: catch potential ANSI fragments being processed as text
			// Only warn if '[' or 'm' appears near escape characters (likely fragments)
			if char == '[' || char == 'm' {
				start := i - 5
				if start < 0 {
					start = 0
				}
				end := i + 5
				if end > len(fullData) {
					end = len(fullData)
				}
				context := string(fullData[start:end])
				// Only log if this looks like a real ANSI fragment (has \x1b nearby)
				if strings.Contains(context, "\x1b") && char == '[' {
					t.logger.Printf("*** POTENTIAL ANSI FRAGMENT: char='%c' at pos %d, context: %q ***", char, i, context)
				}
			}

			// Debug logging for block drawing characters
			if char == '▄' || char == '▀' || char == '█' {
				t.logger.Printf("BLOCK CHAR: '%c' (U+%04X) with FG=%s BG=%s Bold=%t", char, char, t.currentFgHex, t.currentBgHex, t.bold)
			}
			
			// Debug logging for NO PVP characters
			if char == 'N' || char == 'O' || char == 'P' || char == 'V' || char == ' ' {
				debugFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err == nil {
					debugLogger := log.New(debugFile, "[CHAR] ", log.LstdFlags|log.Lshortfile)
					debugLogger.Printf("Processing char '%c' with FG=%s Bold=%t", char, t.currentFgHex, t.bold)
					debugFile.Close()
				}
			}

			t.processRune(char)
			i += size // Advance by the number of bytes consumed
		}
	}
}

// processANSISequence handles a complete ANSI escape sequence
func (t *Terminal) processANSISequence(sequence []byte) {
	if len(sequence) < 3 || sequence[0] != '\x1b' || sequence[1] != '[' {
		return
	}

	// Extract the parameter part (everything between [ and the final letter)
	params := string(sequence[2 : len(sequence)-1])
	command := sequence[len(sequence)-1]

	switch command {
	case 'm': // Color/attribute commands
		// Always require converter - no fallback
		if t.ansiConverter == nil {
			// Log error if converter is missing
			debugFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				debugLogger := log.New(debugFile, "[ERROR] ", log.LstdFlags|log.Lshortfile)
				debugLogger.Printf("MISSING CONVERTER: params=%q - terminal has no converter!", params)
				debugFile.Close()
			}
			return
		}

		// Pass complete parameters to converter and get back current state
		fgHex, bgHex, bold, underline, reverse := t.ansiConverter.ConvertANSIParams(params)

		// Debug logging for ALL sequences passed to converter
		debugFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			debugLogger := log.New(debugFile, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
			debugLogger.Printf("ANSI CONVERT: params=%q -> FG:%s BG:%s Bold:%t", params, fgHex, bgHex, bold)
			debugFile.Close()
		}

		// Update terminal's current colors (converter is source of truth)
		t.currentFgHex = fgHex
		t.currentBgHex = bgHex
		t.bold = bold
		t.underline = underline
		t.reverse = reverse

	case 'H', 'f': // Cursor position
		t.handleCursorPosition(params)
	case 'A': // Cursor up
		t.handleCursorUp(params)
	case 'B': // Cursor down
		t.handleCursorDown(params)
	case 'C': // Cursor right
		t.handleCursorRight(params)
	case 'D': // Cursor left
		t.handleCursorLeft(params)
	case 'J': // Erase display
		t.handleEraseDisplay(params)
	case 'K': // Erase line
		t.handleEraseLine(params)
	default:
		// Unknown command - log it
		t.logger.Printf("Unknown ANSI command: %c with params: %q", command, params)
	}
}

// parseParams parses semicolon-separated integer parameters
func (t *Terminal) parseParams(params string) []int {
	var paramList []int
	if params == "" {
		return paramList
	}
	
	parts := strings.Split(params, ";")
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err == nil {
			paramList = append(paramList, num)
		} else {
			paramList = append(paramList, 0)
		}
	}
	return paramList
}

// handleCursorPosition handles cursor positioning commands (H, f)
func (t *Terminal) handleCursorPosition(params string) {
	paramList := t.parseParams(params)
	row, col := 1, 1
	if len(paramList) >= 1 && paramList[0] > 0 {
		row = paramList[0]
	}
	if len(paramList) >= 2 && paramList[1] > 0 {
		col = paramList[1]
	}
	
	oldX, oldY := t.cursorX, t.cursorY
	t.cursorY = row - 1
	t.cursorX = col - 1
	
	// Debug suspicious cursor movements
	if t.cursorX >= t.width || t.cursorY >= t.height {
		t.logger.Printf("CURSOR_MOVE: params=%q -> (%d,%d) BEFORE clamp, terminal=%dx%d", params, t.cursorX, t.cursorY, t.width, t.height)
	}
	
	if t.cursorY >= t.height {
		t.cursorY = t.height - 1
	}
	if t.cursorX >= t.width {
		t.cursorX = t.width - 1
	}
	if t.cursorY < 0 {
		t.cursorY = 0
	}
	if t.cursorX < 0 {
		t.cursorX = 0
	}
	
	// Debug significant cursor movements
	if abs(t.cursorX - oldX) > 10 || abs(t.cursorY - oldY) > 5 {
		t.logger.Printf("CURSOR_JUMP: (%d,%d) -> (%d,%d) from params=%q", oldX, oldY, t.cursorX, t.cursorY, params)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// handleCursorUp handles cursor up command (A)
func (t *Terminal) handleCursorUp(params string) {
	paramList := t.parseParams(params)
	n := 1
	if len(paramList) >= 1 && paramList[0] > 0 {
		n = paramList[0]
	}
	t.cursorY -= n
	if t.cursorY < 0 {
		t.cursorY = 0
	}
}

// handleCursorDown handles cursor down command (B)
func (t *Terminal) handleCursorDown(params string) {
	paramList := t.parseParams(params)
	n := 1
	if len(paramList) >= 1 && paramList[0] > 0 {
		n = paramList[0]
	}
	t.cursorY += n
	if t.cursorY >= t.height {
		t.cursorY = t.height - 1
	}
}

// handleCursorRight handles cursor right command (C)
func (t *Terminal) handleCursorRight(params string) {
	paramList := t.parseParams(params)
	n := 1
	if len(paramList) >= 1 && paramList[0] > 0 {
		n = paramList[0]
	}
	t.cursorX += n
	if t.cursorX >= t.width {
		t.cursorX = t.width - 1
	}
}

// handleCursorLeft handles cursor left command (D)
func (t *Terminal) handleCursorLeft(params string) {
	paramList := t.parseParams(params)
	n := 1
	if len(paramList) >= 1 && paramList[0] > 0 {
		n = paramList[0]
	}
	t.cursorX -= n
	if t.cursorX < 0 {
		t.cursorX = 0
	}
}

// handleEraseDisplay handles erase display commands (J)
func (t *Terminal) handleEraseDisplay(params string) {
	paramList := t.parseParams(params)
	n := 0
	if len(paramList) >= 1 {
		n = paramList[0]
	}
	switch n {
	case 0: // Clear from cursor to end of screen
		t.clearFromCursor()
	case 1: // Clear from beginning of screen to cursor
		t.clearToCursor()
	case 2: // Clear entire screen
		t.saveCurrentScreen() // Save before clearing
		t.clear()
	}
}

// handleEraseLine handles erase line commands (K)
func (t *Terminal) handleEraseLine(params string) {
	paramList := t.parseParams(params)
	n := 0
	if len(paramList) >= 1 {
		n = paramList[0]
	}
	switch n {
	case 0: // Clear from cursor to end of line
		t.clearLineFromCursor()
	case 1: // Clear from beginning of line to cursor
		t.clearLineToCursor()
	case 2: // Clear entire line
		t.clearLine()
	}
}

// processRune handles a single rune of input (UTF-8 aware)
// Note: ANSI escape sequences are now handled separately by processDataWithANSI
func (t *Terminal) processRune(char rune) {
	// Skip escape characters - they're handled by processDataWithANSI
	if char == 0x1B {
		return
	}

	// Handle control characters
	switch char {
	case '\r': // Carriage return
		t.cursorX = 0
	case '\n': // Line feed
		t.newline()
	case '\t': // Tab
		t.cursorX = ((t.cursorX / 8) + 1) * 8
		if t.cursorX >= t.width {
			t.cursorX = t.width - 1
		}
	case '\b': // Backspace
		if t.cursorX > 0 {
			t.cursorX--
		}
	case 0x07: // Bell
		// Ignore bell for now
	default:
		// Printable character - now handles full Unicode range
		if char >= 32 {
			t.putChar(char)
		}
	}
}

// putChar places a character at the current cursor position
func (t *Terminal) putChar(char rune) {
	if t.cursorX >= t.width {
		t.logger.Printf("WRAP: char='%c' at cursor (%d,%d), width=%d - wrapping to next line", char, t.cursorX, t.cursorY, t.width)
		t.newline()
		t.cursorX = 0
	}

	if t.cursorY >= 0 && t.cursorY < t.height && t.cursorX >= 0 && t.cursorX < t.width {
		cell := Cell{
			Char:          char,
			ForegroundHex: t.currentFgHex,
			BackgroundHex: t.currentBgHex,
			Bold:          t.bold,
			Underline:     t.underline,
			Reverse:       t.reverse,
		}

		// Debug logging for characters at or near line boundaries
		if t.cursorX >= t.width-3 {
			t.logger.Printf("NEAR_END: char='%c' at (%d,%d) width=%d FG=%s BG=%s", char, t.cursorX, t.cursorY, t.width, cell.ForegroundHex, cell.BackgroundHex)
		}
		
		// Count characters being written to each row for debugging
		if t.cursorY < 5 {
			// Count non-null characters in current row
			rowCellCount := 0
			for x := 0; x < t.width; x++ {
				if t.buffer[t.cursorY][x].Char != 0 {
					rowCellCount++
				}
			}
			if rowCellCount >= t.width-2 {
				t.logger.Printf("ROW_FILLING: row %d has %d non-null chars, about to add char '%c' at col %d", t.cursorY, rowCellCount, char, t.cursorX)
			}
		}

		// Log hex conversion for NO PVP characters
		if char == 'N' || char == 'O' || char == ' ' || char == 'P' || char == 'V' {
			pvpLogFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				pvpLogger := log.New(pvpLogFile, "[PVP] ", log.LstdFlags|log.Lshortfile)
				pvpLogger.Printf("STAGE 5 - HEX: '%c' -> FG=%s BG=%s Bold=%t", char, cell.ForegroundHex, cell.BackgroundHex, cell.Bold)
				pvpLogFile.Close()
			}
		}

		t.buffer[t.cursorY][t.cursorX] = cell
	}

	t.cursorX++
}

// newline moves cursor to next line, scrolling if necessary
func (t *Terminal) newline() {
	t.cursorY++
	if t.cursorY > t.scrollBot {
		t.scroll()
		t.cursorY = t.scrollBot
	}
}

// scroll moves content up by one line
func (t *Terminal) scroll() {
	// Save top line to scrollback
	t.scrollback[t.scrollHead] = make([]Cell, t.width)
	copy(t.scrollback[t.scrollHead], t.buffer[t.scrollTop])
	t.scrollHead = (t.scrollHead + 1) % t.scrollSize

	// Move lines up
	for y := t.scrollTop; y < t.scrollBot; y++ {
		copy(t.buffer[y], t.buffer[y+1])
	}

	// Clear bottom line
	for x := 0; x < t.width; x++ {
		t.buffer[t.scrollBot][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
	}
}

// clear clears the entire screen
func (t *Terminal) clear() {
	for y := 0; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.buffer[y][x] = Cell{Char: ' ', ForegroundHex: "#c0c0c0", BackgroundHex: "#000000"}
		}
	}
	t.cursorX = 0
	t.cursorY = 0
}

// hexToRGB converts hex color to RGB values
func hexToRGB(hex string) (r, g, b int) {
	if len(hex) != 7 || hex[0] != '#' {
		return 192, 192, 192 // Default to light gray
	}

	fmt.Sscanf(hex[1:], "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// GetLines returns the current screen content as strings with true color ANSI codes
func (t *Terminal) GetLines() []string {
	lines := make([]string, t.height)
	for y := 0; y < t.height; y++ {
		var line strings.Builder
		// Track last state to minimize escape sequences
		lastFgHex, lastBgHex := "", ""
		lastBold, lastUnderline, lastReverse := false, false, false

		// Find the actual end of content (skip trailing spaces and null chars)
		endX := t.width - 1
		for endX >= 0 && (t.buffer[y][endX].Char == ' ' || t.buffer[y][endX].Char == 0) {
			endX--
		}
		endX++ // Include one position past the last non-space char

		for x := 0; x <= endX && x < t.width; x++ {
			cell := t.buffer[y][x]

			// Skip null characters completely
			if cell.Char == 0 {
				continue
			}

			// Check if any attribute has changed
			attributeChanged := cell.ForegroundHex != lastFgHex ||
				cell.BackgroundHex != lastBgHex ||
				cell.Bold != lastBold ||
				cell.Underline != lastUnderline ||
				cell.Reverse != lastReverse

			if attributeChanged {
				// Build complete true color ANSI sequence
				var ansiParts []string

				// Add true color foreground
				if cell.ForegroundHex != "#c0c0c0" { // Not default white
					fgR, fgG, fgB := hexToRGB(cell.ForegroundHex)
					ansiParts = append(ansiParts, fmt.Sprintf("38;2;%d;%d;%d", fgR, fgG, fgB))
				}

				// Add true color background
				if cell.BackgroundHex != "#000000" { // Not default black
					bgR, bgG, bgB := hexToRGB(cell.BackgroundHex)
					ansiParts = append(ansiParts, fmt.Sprintf("48;2;%d;%d;%d", bgR, bgG, bgB))
				}

				// Add attributes
				if cell.Bold {
					ansiParts = append(ansiParts, "1")
				}
				if cell.Underline {
					ansiParts = append(ansiParts, "4")
				}
				if cell.Reverse {
					ansiParts = append(ansiParts, "7")
				}

				// Output the complete sequence
				if len(ansiParts) > 0 {
					line.WriteString(fmt.Sprintf("\x1b[%sm", strings.Join(ansiParts, ";")))
				} else {
					// All attributes are default - reset
					line.WriteString("\x1b[0m")
				}

				// Update tracking variables
				lastFgHex, lastBgHex = cell.ForegroundHex, cell.BackgroundHex
				lastBold, lastUnderline, lastReverse = cell.Bold, cell.Underline, cell.Reverse
			}

			line.WriteRune(cell.Char)
		}

		// Reset at end of line if any non-default attributes are active
		if lastFgHex != "#c0c0c0" || lastBgHex != "#000000" || lastBold || lastUnderline || lastReverse {
			line.WriteString("\x1b[0m")
		}

		lines[y] = strings.TrimRight(line.String(), " ")
	}
	return lines
}

// GetCursor returns the current cursor position
func (t *Terminal) GetCursor() (int, int) {
	return t.cursorX, t.cursorY
}

// GetSize returns the terminal dimensions
func (t *Terminal) GetSize() (int, int) {
	return t.width, t.height
}

// GetCells returns the raw cell data for external processing
func (t *Terminal) GetCells() [][]Cell {
	return t.buffer
}

// GetAllCells returns scrollback + current buffer for scrolling
func (t *Terminal) GetAllCells() [][]Cell {
	var allCells [][]Cell
	scrollbackLines := 0

	// Add scrollback content (oldest first) - only lines with visible content
	for i := 0; i < t.scrollSize; i++ {
		scrollIdx := (t.scrollHead + i) % t.scrollSize
		if t.scrollback[scrollIdx] != nil && len(t.scrollback[scrollIdx]) > 0 {
			// Check if this line has any visible content (be more lenient)
			hasContent := false
			nonSpaceCount := 0
			for _, cell := range t.scrollback[scrollIdx] {
				if cell.Char != 0 && cell.Char != ' ' {
					nonSpaceCount++
					if nonSpaceCount >= 1 { // At least 1 non-space character
						hasContent = true
						break
					}
				}
			}
			if hasContent {
				// Make a copy to avoid reference issues
				lineCopy := make([]Cell, len(t.scrollback[scrollIdx]))
				copy(lineCopy, t.scrollback[scrollIdx])
				allCells = append(allCells, lineCopy)
				scrollbackLines++
			}
		}
	}

	// Add current screen buffer
	allCells = append(allCells, t.buffer...)

	t.logger.Printf("GetAllCells: %d scrollback lines + %d buffer lines = %d total",
		scrollbackLines, len(t.buffer), len(allCells))

	return allCells
}

// saveCurrentScreen saves the current screen to history before clearing
func (t *Terminal) saveCurrentScreen() {
	// Only save if there's actual content
	hasContent := false
	for _, line := range t.buffer {
		for _, cell := range line {
			if cell.Char > 32 && cell.Char != 127 {
				hasContent = true
				break
			}
		}
		if hasContent {
			break
		}
	}

	if hasContent {
		// Create a deep copy of the current screen
		screenCopy := make([][]Cell, len(t.buffer))
		for i, line := range t.buffer {
			screenCopy[i] = make([]Cell, len(line))
			copy(screenCopy[i], line)
		}

		// Store in circular buffer
		t.screenHistory[t.screenHistoryHead] = screenCopy
		t.screenHistoryHead = (t.screenHistoryHead + 1) % t.maxScreenHistory

		t.logger.Printf("Saved complete screen to history (head: %d)", t.screenHistoryHead)
	}
}

// GetScreenHistory returns all saved screens for scrolling
func (t *Terminal) GetScreenHistory() [][][]Cell {
	var screens [][][]Cell

	// Collect all non-nil screens from history
	for i := 0; i < t.maxScreenHistory; i++ {
		idx := (t.screenHistoryHead + i) % t.maxScreenHistory
		if t.screenHistory[idx] != nil {
			screens = append(screens, t.screenHistory[idx])
		}
	}

	// Add current screen at the end
	currentScreen := make([][]Cell, len(t.buffer))
	for i, line := range t.buffer {
		currentScreen[i] = make([]Cell, len(line))
		copy(currentScreen[i], line)
	}
	screens = append(screens, currentScreen)

	return screens
}

// IsDirty returns whether the terminal has been updated since last check
func (t *Terminal) IsDirty() bool {
	return t.dirty
}

// ClearDirty marks the terminal as clean
func (t *Terminal) ClearDirty() {
	t.dirty = false
}

// SetUpdateCallback sets a function to be called when the terminal is updated
func (t *Terminal) SetUpdateCallback(callback func()) {
	t.onUpdate = callback
}

// GetNewData returns new data since the last call and clears the buffer
func (t *Terminal) GetNewData() []byte {
	t.newDataMutex.Lock()
	defer t.newDataMutex.Unlock()

	if len(t.newDataBuffer) == 0 {
		return nil
	}

	// Return a copy of the data and clear the buffer
	data := make([]byte, len(t.newDataBuffer))
	copy(data, t.newDataBuffer)
	t.newDataBuffer = t.newDataBuffer[:0] // Clear the buffer

	return data
}


// Clear helper functions
func (t *Terminal) clearFromCursor() {
	// Clear from cursor to end of current line
	for x := t.cursorX; x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
	}
	// Clear all lines below
	for y := t.cursorY + 1; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.buffer[y][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
		}
	}
}

func (t *Terminal) clearToCursor() {
	// Clear all lines above
	for y := 0; y < t.cursorY; y++ {
		for x := 0; x < t.width; x++ {
			t.buffer[y][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
		}
	}
	// Clear from beginning of current line to cursor
	for x := 0; x <= t.cursorX && x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
	}
}

func (t *Terminal) clearLineFromCursor() {
	for x := t.cursorX; x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
	}
}

func (t *Terminal) clearLineToCursor() {
	for x := 0; x <= t.cursorX && x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
	}
}

func (t *Terminal) clearLine() {
	for x := 0; x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', ForegroundHex: t.currentFgHex, BackgroundHex: t.currentBgHex}
	}
}
