package terminal

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Cell represents a single character cell in the terminal
type Cell struct {
	Char       rune
	Foreground int // ANSI color code
	Background int // ANSI color code
	Bold       bool
	Underline  bool
	Reverse    bool
}

// Terminal represents a virtual terminal with a screen buffer
type Terminal struct {
	width      int
	height     int
	buffer     [][]Cell
	cursorX    int
	cursorY    int
	scrollTop  int // Top of scrollable region
	scrollBot  int // Bottom of scrollable region
	
	// Scrollback buffer (circular buffer)
	scrollback [][]Cell
	scrollHead int
	scrollSize int
	
	// Current text attributes
	fg         int
	bg         int
	bold       bool
	underline  bool
	reverse    bool
	
	// State
	escapeSeq  string
	inEscape   bool
	
	// Update tracking
	dirty      bool
	lastUpdate time.Time
	
	// Logger
	logger     *log.Logger
	
	// Update callback for UI notifications  
	onUpdate   func()
}

// NewTerminal creates a new terminal buffer
func NewTerminal(width, height int) *Terminal {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	
	logger := log.New(logFile, "[TERM] ", log.LstdFlags|log.Lshortfile)
	logger.Printf("Terminal initialized %dx%d", width, height)
	
	t := &Terminal{
		width:      width,
		height:     height,
		buffer:     make([][]Cell, height),
		scrollback: make([][]Cell, 1000), // 1000 lines of scrollback
		scrollSize: 1000,
		scrollTop:  0,
		scrollBot:  height - 1,
		fg:         7, // Default white
		bg:         0, // Default black
		logger:     logger,
	}
	
	// Initialize buffer with default cells
	for i := range t.buffer {
		t.buffer[i] = make([]Cell, width)
		for j := range t.buffer[i] {
			t.buffer[i][j] = Cell{Char: ' ', Foreground: 7, Background: 0}
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
			t.buffer[i][j] = Cell{Char: ' ', Foreground: 7, Background: 0}
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

// Write processes incoming data and updates the terminal buffer
func (t *Terminal) Write(data []byte) {
	t.logger.Printf("Terminal received %d bytes: %q", len(data), string(data))
	
	// Convert to string to handle UTF-8 properly, then process rune by rune
	text := string(data)
	for _, r := range text {
		t.processRune(r)
	}
	
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

// processRune handles a single rune of input (UTF-8 aware)
func (t *Terminal) processRune(char rune) {
	
	// Handle escape sequences
	if t.inEscape {
		t.escapeSeq += string(char)
		if t.handleEscapeSequence(char) {
			t.inEscape = false
			t.escapeSeq = ""
		}
		return
	}
	
	// Start of escape sequence
	if char == 0x1B { // ESC
		t.inEscape = true
		t.escapeSeq = string(char)
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
		t.buffer[t.cursorY][t.cursorX] = Cell{
			Char:       char,
			Foreground: t.fg,
			Background: t.bg,
			Bold:       t.bold,
			Underline:  t.underline,
			Reverse:    t.reverse,
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
		t.buffer[t.scrollBot][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
	}
}

// clear clears the entire screen
func (t *Terminal) clear() {
	for y := 0; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.buffer[y][x] = Cell{Char: ' ', Foreground: 7, Background: 0}
		}
	}
	t.cursorX = 0
	t.cursorY = 0
}

// GetLines returns the current screen content as strings with ANSI color codes
func (t *Terminal) GetLines() []string {
	lines := make([]string, t.height)
	for y := 0; y < t.height; y++ {
		var line strings.Builder
		lastFg, lastBg := -1, -1 // Track last colors to minimize escape sequences
		
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
			
			// Add color changes only when needed
			if cell.Foreground != lastFg || cell.Background != lastBg {
				if cell.Foreground != 7 || cell.Background != 0 { // Not default colors
					line.WriteString(fmt.Sprintf("\x1b[%d;%dm", 30+cell.Foreground, 40+cell.Background))
				} else {
					line.WriteString("\x1b[0m") // Reset to default
				}
				lastFg, lastBg = cell.Foreground, cell.Background
			}
			
			line.WriteRune(cell.Char)
		}
		
		// Reset colors at end of line if not default
		if lastFg != 7 || lastBg != 0 {
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

// GetCells returns the raw cell data for external processing
func (t *Terminal) GetCells() [][]Cell {
	return t.buffer
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

// ANSI escape sequence regex patterns
var (
	csiPattern = regexp.MustCompile(`^\x1B\[([0-9;]*)([A-Za-z])$`)
)

// handleEscapeSequence processes ANSI escape sequences
func (t *Terminal) handleEscapeSequence(char rune) bool {
	// Check if sequence is complete
	if len(t.escapeSeq) < 2 {
		return false
	}
	
	// Handle CSI sequences (ESC[...)
	if strings.HasPrefix(t.escapeSeq, "\x1B[") {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') {
			t.handleCSI(t.escapeSeq)
			return true
		}
		return false
	}
	
	// Handle other escape sequences
	if len(t.escapeSeq) >= 2 {
		return true // Unknown sequence, consume it
	}
	
	return false
}

// handleCSI processes CSI (Control Sequence Introducer) sequences
func (t *Terminal) handleCSI(seq string) {
	matches := csiPattern.FindStringSubmatch(seq)
	if len(matches) != 3 {
		t.logger.Printf("Invalid CSI sequence: %q", seq)
		return
	}
	
	params := matches[1]
	command := matches[2]
	
	t.logger.Printf("CSI command: %s, params: %s", command, params)
	
	// Parse parameters
	var paramList []int
	if params != "" {
		parts := strings.Split(params, ";")
		for _, part := range parts {
			if num, err := strconv.Atoi(part); err == nil {
				paramList = append(paramList, num)
			} else {
				paramList = append(paramList, 0)
			}
		}
	}
	
	switch command {
	case "H", "f": // Cursor position
		row, col := 1, 1
		if len(paramList) >= 1 && paramList[0] > 0 {
			row = paramList[0]
		}
		if len(paramList) >= 2 && paramList[1] > 0 {
			col = paramList[1]
		}
		t.cursorY = row - 1
		t.cursorX = col - 1
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
		
	case "A": // Cursor up
		n := 1
		if len(paramList) >= 1 && paramList[0] > 0 {
			n = paramList[0]
		}
		t.cursorY -= n
		if t.cursorY < 0 {
			t.cursorY = 0
		}
		
	case "B": // Cursor down
		n := 1
		if len(paramList) >= 1 && paramList[0] > 0 {
			n = paramList[0]
		}
		t.cursorY += n
		if t.cursorY >= t.height {
			t.cursorY = t.height - 1
		}
		
	case "C": // Cursor right
		n := 1
		if len(paramList) >= 1 && paramList[0] > 0 {
			n = paramList[0]
		}
		t.cursorX += n
		if t.cursorX >= t.width {
			t.cursorX = t.width - 1
		}
		
	case "D": // Cursor left
		n := 1
		if len(paramList) >= 1 && paramList[0] > 0 {
			n = paramList[0]
		}
		t.cursorX -= n
		if t.cursorX < 0 {
			t.cursorX = 0
		}
		
	case "J": // Erase display
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
		
	case "K": // Erase line
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
		
	case "m": // Set graphics mode (colors, attributes)
		if len(paramList) == 0 {
			paramList = []int{0} // Reset if no params
		}
		for _, param := range paramList {
			t.handleSGR(param)
		}
	}
}

// handleSGR handles Select Graphic Rendition (color/attribute) codes
func (t *Terminal) handleSGR(code int) {
	switch {
	case code == 0: // Reset all
		t.fg = 7
		t.bg = 0
		t.bold = false
		t.underline = false
		t.reverse = false
		
	case code == 1: // Bold
		t.bold = true
		
	case code == 4: // Underline
		t.underline = true
		
	case code == 7: // Reverse
		t.reverse = true
		
	case code == 22: // Normal intensity
		t.bold = false
		
	case code == 24: // No underline
		t.underline = false
		
	case code == 27: // No reverse
		t.reverse = false
		
	case code >= 30 && code <= 37: // Foreground colors
		t.fg = code - 30
		
	case code >= 40 && code <= 47: // Background colors
		t.bg = code - 40
		
	case code == 39: // Default foreground
		t.fg = 7
		
	case code == 49: // Default background
		t.bg = 0
	}
}

// Clear helper functions
func (t *Terminal) clearFromCursor() {
	// Clear from cursor to end of current line
	for x := t.cursorX; x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
	}
	// Clear all lines below
	for y := t.cursorY + 1; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.buffer[y][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
		}
	}
}

func (t *Terminal) clearToCursor() {
	// Clear all lines above
	for y := 0; y < t.cursorY; y++ {
		for x := 0; x < t.width; x++ {
			t.buffer[y][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
		}
	}
	// Clear from beginning of current line to cursor
	for x := 0; x <= t.cursorX && x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
	}
}

func (t *Terminal) clearLineFromCursor() {
	for x := t.cursorX; x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
	}
}

func (t *Terminal) clearLineToCursor() {
	for x := 0; x <= t.cursorX && x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
	}
}

func (t *Terminal) clearLine() {
	for x := 0; x < t.width; x++ {
		t.buffer[t.cursorY][x] = Cell{Char: ' ', Foreground: t.fg, Background: t.bg}
	}
}