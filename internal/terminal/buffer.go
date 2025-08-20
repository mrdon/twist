package terminal

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"twist/internal/ansi"
	"unicode/utf8"
)

// ColorChange represents a position where color attributes change
type ColorChange struct {
	X, Y     int    // Position where color changes
	TViewTag string // Direct tview color tag: "[red:blue:b]"
}

// Terminal represents a virtual terminal with a screen buffer
type Terminal struct {
	width  int
	height int

	// Efficient storage
	runes        [][]rune      // Just the characters (2D grid)
	colorChanges []ColorChange // Sparse color data

	cursorX   int
	cursorY   int
	scrollTop int // Top of scrollable region
	scrollBot int // Bottom of scrollable region

	// Current state tracking
	currentColorTag string // Current tview color tag

	// ANSI converter for immediate color conversion
	ansiConverter *ansi.ColorConverter

	// Fixed-size buffer for streaming data processing
	buffer    [8192]byte // 8KB fixed buffer
	bufferLen int        // Current length of data in buffer

	// Update tracking
	dirty      bool
	lastUpdate time.Time

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
func NewTerminalWithConverter(width, height int, converter *ansi.ColorConverter) *Terminal {
	return NewTerminalWithConverterAndLogger(width, height, converter, nil)
}

// NewTerminalWithConverterAndLogger creates a new terminal buffer with an ANSI converter and shared logger
func NewTerminalWithConverterAndLogger(width, height int, converter *ansi.ColorConverter, pvpLogger interface{}) *Terminal {

	t := &Terminal{
		width:  width,
		height: height,

		// Efficient storage
		runes:        make([][]rune, height),
		colorChanges: make([]ColorChange, 0, 100), // Start with capacity for 100 color changes

		scrollTop:       0,
		scrollBot:       height - 1,
		currentColorTag: "[#c0c0c0:#000000]", // Default color tag
		ansiConverter:   converter,
	}

	// Initialize rune buffer
	for i := range t.runes {
		t.runes[i] = make([]rune, width)
		// Default to spaces (rune 0 means empty)
		for j := range t.runes[i] {
			t.runes[i][j] = ' '
		}
	}

	t.clear()
	return t
}

// Resize changes the terminal dimensions
func (t *Terminal) Resize(width, height int) {

	oldRunes := t.runes
	t.width = width
	t.height = height
	t.scrollBot = height - 1

	// Initialize new rune buffer
	t.runes = make([][]rune, height)
	for i := range t.runes {
		t.runes[i] = make([]rune, width)
		for j := range t.runes[i] {
			t.runes[i][j] = ' '
		}
	}

	// Copy old content (best effort)
	copyHeight := height
	if len(oldRunes) < copyHeight {
		copyHeight = len(oldRunes)
	}

	for y := 0; y < copyHeight; y++ {
		copyWidth := width
		if len(oldRunes[y]) < copyWidth {
			copyWidth = len(oldRunes[y])
		}
		copy(t.runes[y][:copyWidth], oldRunes[y][:copyWidth])
	}

	// Adjust cursor position
	if t.cursorX >= width {
		t.cursorX = width - 1
	}
	if t.cursorY >= height {
		t.cursorY = height - 1
	}

	// Clear color changes since positions may no longer be valid
	t.colorChanges = t.colorChanges[:0]
	t.currentColorTag = "[#c0c0c0:#000000]"

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

	// Store new data for incremental updates
	t.newDataMutex.Lock()
	t.newDataBuffer = append(t.newDataBuffer, data...)
	t.newDataMutex.Unlock()

	// Process ANSI escape sequences first, then handle remaining characters
	t.processDataWithANSI(data)

	t.dirty = true
	t.lastUpdate = time.Now()

	// Notify UI of update
	if t.onUpdate != nil {
		t.onUpdate()
	}
}

// processDataWithANSI processes input data with fixed-size buffer
func (t *Terminal) processDataWithANSI(data []byte) {
	// Add new data to buffer (handle case where data is larger than remaining buffer space)
	for len(data) > 0 {
		// How much space is left in buffer?
		spaceLeft := len(t.buffer) - t.bufferLen

		// How much can we add this iteration?
		toAdd := len(data)
		if toAdd > spaceLeft {
			toAdd = spaceLeft
		}

		// Add data to buffer
		copy(t.buffer[t.bufferLen:], data[:toAdd])
		t.bufferLen += toAdd
		data = data[toAdd:]

		// Process everything we can from the buffer
		consumed := 0
		for consumed < t.bufferLen {
			// Try to consume starting from current position
			bytesConsumed := t.tryConsumeSequence(t.buffer[consumed:t.bufferLen])

			if bytesConsumed > 0 {
				// Successfully consumed some bytes
				consumed += bytesConsumed
			} else {
				// Couldn't consume anything - incomplete sequence
				// Leave unconsumed data in buffer
				break
			}
		}

		// Remove consumed data from buffer, keep unconsumed data
		if consumed > 0 {
			copy(t.buffer[:], t.buffer[consumed:t.bufferLen])
			t.bufferLen -= consumed
		}

		// Safety check: if buffer is full and we couldn't consume anything, force consume one character
		if t.bufferLen == len(t.buffer) && consumed == 0 {
			char, size := utf8.DecodeRune(t.buffer[:])
			t.processRune(char)
			copy(t.buffer[:], t.buffer[size:t.bufferLen])
			t.bufferLen -= size
		}
	}
}

// tryConsumeSequence attempts to consume data starting at the beginning of the slice
// Returns number of bytes consumed, or 0 if incomplete sequence
func (t *Terminal) tryConsumeSequence(data []byte) int {
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
					t.processANSISequence(sequence)
					return i
				}
				i++
			}
			// No terminator found - incomplete sequence
			return 0
		} else {
			// \x1b followed by non-[ - treat as regular character
			char, size := utf8.DecodeRune(data)
			t.processRune(char)
			return size
		}
	} else {
		// Regular character
		char, size := utf8.DecodeRune(data)
		if char == utf8.RuneError && size == 1 {
			return 1 // skip invalid byte
		}
		t.processRune(char)
		return size
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
			return
		}

		// Pass complete parameters to converter and get back tview color tag
		colorTag := t.ansiConverter.ConvertANSIParams(params)

		// New approach: Store color change only if color actually changed
		if colorTag != t.currentColorTag {
			t.colorChanges = append(t.colorChanges, ColorChange{
				X:        t.cursorX,
				Y:        t.cursorY,
				TViewTag: colorTag,
			})
			t.currentColorTag = colorTag
		}

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
	if abs(t.cursorX-oldX) > 10 || abs(t.cursorY-oldY) > 5 {
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
		t.newline()
		t.cursorX = 0
	}

	if t.cursorY >= 0 && t.cursorY < t.height && t.cursorX >= 0 && t.cursorX < t.width {
		// Store just the rune
		t.runes[t.cursorY][t.cursorX] = char

		// Debug logging for characters at or near line boundaries
		if t.cursorX >= t.width-3 {
		}

		// Special debug for column 80 (should not happen)
		if t.cursorX >= 80 {
		}
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
	// Move lines up in rune buffer
	for y := t.scrollTop; y < t.scrollBot; y++ {
		copy(t.runes[y], t.runes[y+1])
	}

	// Clear bottom line in rune buffer
	for x := 0; x < t.width; x++ {
		t.runes[t.scrollBot][x] = ' '
	}

	// Update color changes: remove any that scrolled off the top, adjust Y coordinates
	newColorChanges := make([]ColorChange, 0, len(t.colorChanges))
	var lastColorBeforeScrollTop string = "[#c0c0c0:#000000]" // Default color

	// Sort color changes by Y, then X to process them in order
	for _, change := range t.colorChanges {
		if change.Y < t.scrollTop {
			// This color change scrolled off - track the last active color before scroll area
			lastColorBeforeScrollTop = change.TViewTag
		} else if change.Y == t.scrollTop {
			// Color change at scroll line - moves up one line
			newColorChanges = append(newColorChanges, ColorChange{
				X:        change.X,
				Y:        change.Y - 1,
				TViewTag: change.TViewTag,
			})
		} else {
			// Color change below scroll line - moves up one line
			newColorChanges = append(newColorChanges, ColorChange{
				X:        change.X,
				Y:        change.Y - 1,
				TViewTag: change.TViewTag,
			})
		}
	}

	// If we had color changes that scrolled off and the new top line doesn't start with the right color,
	// add a color change at the beginning of the new top line to maintain color continuity
	hasColorAtTopLeft := false
	for _, change := range newColorChanges {
		if change.Y == t.scrollTop && change.X == 0 {
			hasColorAtTopLeft = true
			break
		}
	}

	if !hasColorAtTopLeft && lastColorBeforeScrollTop != "[#c0c0c0:#000000]" {
		// Add color change at top-left to maintain color from scrolled content
		newColorChanges = append([]ColorChange{{
			X:        0,
			Y:        t.scrollTop,
			TViewTag: lastColorBeforeScrollTop,
		}}, newColorChanges...)
	}

	t.colorChanges = newColorChanges
}

// clear clears the entire screen
func (t *Terminal) clear() {
	// Clear rune buffer
	for y := 0; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.runes[y][x] = ' '
		}
	}

	// Clear color changes (start fresh)
	t.colorChanges = t.colorChanges[:0]
	t.currentColorTag = "[#c0c0c0:#000000]" // Reset to default

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

// GetCursor returns the current cursor position
func (t *Terminal) GetCursor() (int, int) {
	return t.cursorX, t.cursorY
}

// GetSize returns the terminal dimensions
func (t *Terminal) GetSize() (int, int) {
	return t.width, t.height
}

// GetRunes returns the character data without color information
func (t *Terminal) GetRunes() [][]rune {
	return t.runes
}

// GetColorChanges returns the sparse color change data
func (t *Terminal) GetColorChanges() []ColorChange {
	return t.colorChanges
}

// GetCurrentColors returns the current terminal color state for testing
func (t *Terminal) GetCurrentColors() (string, string, bool, bool, bool) {
	// Parse the current color tag to extract hex colors and attributes
	fgHex, bgHex := "#c0c0c0", "#000000" // defaults
	bold, underline, reverse := false, false, false

	if t.currentColorTag != "" {
		// This is a simplified parser for the current tview tag
		// In the new system, we track colors differently, but for testing
		// we can infer from the current color tag
		tag := t.currentColorTag
		if strings.Contains(tag, ":b") {
			bold = true
		}
		if strings.Contains(tag, ":u") {
			underline = true
		}
		if strings.Contains(tag, ":r") {
			reverse = true
		}
		// Extract hex colors from tag like [#800000:#000000:b]
		if len(tag) > 3 && tag[0] == '[' {
			parts := strings.Split(tag[1:len(tag)-1], ":")
			if len(parts) >= 2 {
				if parts[0] != "" && parts[0] != "-" {
					fgHex = parts[0]
				}
				if parts[1] != "" && parts[1] != "-" {
					bgHex = parts[1]
				}
			}
		}
	}

	return fgHex, bgHex, bold, underline, reverse
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
		t.runes[t.cursorY][x] = ' '
	}
	// Clear all lines below
	for y := t.cursorY + 1; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.runes[y][x] = ' '
		}
	}
}

func (t *Terminal) clearToCursor() {
	// Clear all lines above
	for y := 0; y < t.cursorY; y++ {
		for x := 0; x < t.width; x++ {
			t.runes[y][x] = ' '
		}
	}
	// Clear from beginning of current line to cursor
	for x := 0; x <= t.cursorX && x < t.width; x++ {
		t.runes[t.cursorY][x] = ' '
	}
}

func (t *Terminal) clearLineFromCursor() {
	for x := t.cursorX; x < t.width; x++ {
		t.runes[t.cursorY][x] = ' '
	}
}

func (t *Terminal) clearLineToCursor() {
	for x := 0; x <= t.cursorX && x < t.width; x++ {
		t.runes[t.cursorY][x] = ' '
	}
}

func (t *Terminal) clearLine() {
	for x := 0; x < t.width; x++ {
		t.runes[t.cursorY][x] = ' '
	}
}
