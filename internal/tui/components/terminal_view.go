package components

import (
	"math"
	"strings"
	"sync"
	"twist/internal/ansi"
	"twist/internal/theme"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TerminalView is a terminal component optimized for terminal emulation with ANSI support
type TerminalView struct {
	*tview.Box

	// Terminal-specific storage - 2D grid for efficient cursor operations
	lines  [][]rune        // Screen lines as runes
	colors [][]tcell.Style // Per-character colors and attributes
	width  int             // Current terminal width
	height int             // Current terminal height

	// Cursor state
	cursorX int
	cursorY int

	// ANSI processing
	ansiConverter *ansi.ColorConverter
	currentStyle  tcell.Style

	// Scrolling
	scrollable      bool
	scrollOffsetRow int
	scrollOffsetCol int

	// ANSI sequence buffering
	buffer    [8192]byte
	bufferLen int

	// Synchronization
	mutex sync.RWMutex

	// Callbacks
	changedFunc func()

	// UI wrapper
	wrapper *tview.Flex
}

// NewTerminalView creates a new terminal view
func NewTerminalView() *TerminalView {
	tv := &TerminalView{
		Box:           tview.NewBox(),
		lines:         make([][]rune, 0),
		colors:        make([][]tcell.Style, 0),
		width:         80,
		height:        24,
		scrollable:    true,
		ansiConverter: ansi.NewColorConverter(),
	}

	// Apply theme colors
	colors := theme.Current().TerminalColors()
	tv.SetBackgroundColor(colors.Background)

	// Set default style using theme colors instead of hardcoded values
	tv.currentStyle = tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)

	// Set up standard tview component styling like TextView
	tv.SetBorder(false)             // Start with no border, can be enabled via SetBorder()
	tv.SetBorderPadding(1, 1, 1, 1) // Default padding: 1 row top/bottom, 1 column left/right

	// Create wrapper with theme colors
	tv.wrapper = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tv, 0, 1, true)

	// Set up change callback for UI updates
	tv.SetChangedFunc(func() {
		// Trigger redraw when terminal content changes
	})

	return tv
}

// resizeBuffer resizes the internal buffers
func (tv *TerminalView) resizeBuffer(width, height int) {
	tv.width = width
	tv.height = height

	// Resize lines buffer
	if len(tv.lines) < height {
		// Need to add lines
		for len(tv.lines) < height {
			tv.lines = append(tv.lines, make([]rune, width))
			newColorLine := make([]tcell.Style, width)
			// Initialize new lines with terminal default colors, not current style
			colors := theme.Current().TerminalColors()
			defaultStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)
			for i := range newColorLine {
				newColorLine[i] = defaultStyle
			}
			tv.colors = append(tv.colors, newColorLine)
		}
	} else if len(tv.lines) > height {
		// Need to remove lines
		tv.lines = tv.lines[:height]
		tv.colors = tv.colors[:height]
	}

	// Resize each line
	for i := range tv.lines {
		if len(tv.lines[i]) < width {
			// Extend line
			newRunes := make([]rune, width)
			copy(newRunes, tv.lines[i])
			tv.lines[i] = newRunes

			newColors := make([]tcell.Style, width)
			copy(newColors, tv.colors[i])
			// Initialize extended area with terminal default colors, not current style
			colors := theme.Current().TerminalColors()
			defaultStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)
			for j := len(tv.colors[i]); j < width; j++ {
				newColors[j] = defaultStyle
			}
			tv.colors[i] = newColors
		} else if len(tv.lines[i]) > width {
			// Truncate line
			tv.lines[i] = tv.lines[i][:width]
			tv.colors[i] = tv.colors[i][:width]
		}
	}
}

// Write implements io.Writer - processes ANSI sequences and updates terminal
func (tv *TerminalView) Write(p []byte) (n int, err error) {
	tv.mutex.Lock()
	defer tv.mutex.Unlock()

	// Process the data through ANSI sequence handling instead of just appending
	tv.processDataWithANSI(p)

	// Auto-scroll to bottom when new content is added (but only if not positioned elsewhere)
	// This should happen during content addition, not during drawing
	_, _, _, height := tv.GetInnerRect()
	if len(tv.lines) > height {
		// Original logic: cursor near bottom
		// Additional fix: also autoscroll if we're already viewing near the bottom,
		// regardless of cursor position (fixes issue with ANSI cursor positioning)
		cursorNearBottom := tv.scrollOffsetRow >= len(tv.lines)-height-3
		isViewingNearBottom := tv.cursorY >= len(tv.lines)-height
		maxScrollOffset := len(tv.lines) - height

		// New fix: After clear screen, if we're at scroll position 0 and there's content
		// beyond the visible area, auto-scroll to show the latest content
		isAfterClearScreen := tv.scrollOffsetRow == 0 && len(tv.lines) > height

		// Additional fix: If cursor is writing outside the current view area, follow it
		cursorOutsideView := tv.cursorY < tv.scrollOffsetRow || tv.cursorY >= tv.scrollOffsetRow+height

		if cursorOutsideView {
			// If cursor is outside current view, follow it immediately
			// Center the cursor in the view for optimal visibility
			tv.scrollOffsetRow = tv.cursorY - height/2
			if tv.scrollOffsetRow < 0 {
				tv.scrollOffsetRow = 0
			}
			if tv.scrollOffsetRow > maxScrollOffset {
				tv.scrollOffsetRow = maxScrollOffset
			}
		} else if cursorNearBottom || isViewingNearBottom || isAfterClearScreen {
			tv.scrollOffsetRow = maxScrollOffset
			if tv.scrollOffsetRow < 0 {
				tv.scrollOffsetRow = 0
			}
		}
	}

	if tv.changedFunc != nil {
		// Make callback non-blocking to avoid deadlocks
		go tv.changedFunc()
	}

	return len(p), nil
}

// processDataWithANSI processes input data with ANSI sequence handling
func (tv *TerminalView) processDataWithANSI(data []byte) {
	for len(data) > 0 {
		// Add data to buffer
		spaceLeft := len(tv.buffer) - tv.bufferLen
		toAdd := len(data)
		if toAdd > spaceLeft {
			toAdd = spaceLeft
		}

		copy(tv.buffer[tv.bufferLen:], data[:toAdd])
		tv.bufferLen += toAdd
		data = data[toAdd:]

		// Process buffer contents
		consumed := 0
		for consumed < tv.bufferLen {
			bytesConsumed := tv.tryConsumeSequence(tv.buffer[consumed:tv.bufferLen])
			if bytesConsumed > 0 {
				consumed += bytesConsumed
			} else {
				break
			}
		}

		// Remove consumed bytes
		if consumed > 0 {
			copy(tv.buffer[:], tv.buffer[consumed:tv.bufferLen])
			tv.bufferLen -= consumed
		}

		// Safety: force consume if buffer is full
		if tv.bufferLen == len(tv.buffer) && consumed == 0 {
			char, size := utf8.DecodeRune(tv.buffer[:])
			tv.putChar(char)
			copy(tv.buffer[:], tv.buffer[size:tv.bufferLen])
			tv.bufferLen -= size
		}
	}
}

// tryConsumeSequence tries to consume an ANSI sequence or character
func (tv *TerminalView) tryConsumeSequence(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	if data[0] == '\x1b' {
		if len(data) < 2 {
			return 0 // Need more data
		}

		if data[1] == '[' {
			// ANSI escape sequence
			i := 2
			for i < len(data) {
				c := data[i]
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					// Found terminator
					i++
					sequence := data[0:i]
					tv.processANSISequence(sequence)
					return i
				}
				i++
			}
			return 0 // Incomplete sequence
		} else {
			// Not an ANSI sequence, treat as regular char
			char, size := utf8.DecodeRune(data)
			tv.processChar(char)
			return size
		}
	} else {
		// Regular character
		char, size := utf8.DecodeRune(data)
		if char == utf8.RuneError && size == 1 {
			return 1 // Skip invalid byte
		}
		tv.processChar(char)
		return size
	}
}

// processANSISequence handles ANSI escape sequences
func (tv *TerminalView) processANSISequence(sequence []byte) {
	if len(sequence) < 3 || sequence[0] != '\x1b' || sequence[1] != '[' {
		return
	}

	params := string(sequence[2 : len(sequence)-1])
	command := sequence[len(sequence)-1]

	switch command {
	case 'm': // Color/style
		tv.handleColorSequence(params)
	case 'H', 'f': // Cursor position
		tv.handleCursorPosition(params)
	case 'A': // Cursor up
		tv.handleCursorUp(params)
	case 'B': // Cursor down
		tv.handleCursorDown(params)
	case 'C': // Cursor right
		tv.handleCursorRight(params)
	case 'D': // Cursor left
		tv.handleCursorLeft(params)
	case 'J': // Erase display
		tv.handleEraseDisplay(params)
	case 'K': // Erase line
		tv.handleEraseLine(params)
	default:
		// Unknown ANSI command - ignore silently
	}
}

// processChar handles regular characters and control codes
func (tv *TerminalView) processChar(char rune) {
	switch char {
	case '\r': // Carriage return
		tv.cursorX = 0
	case '\n': // Line feed
		tv.newline()
	case '\t': // Tab
		tv.cursorX = ((tv.cursorX / 8) + 1) * 8
		if tv.cursorX >= tv.width {
			tv.cursorX = tv.width - 1
		}
	case '\b': // Backspace
		if tv.cursorX > 0 {
			tv.cursorX--
		}
	default:
		if char >= 32 { // Printable character
			tv.putChar(char)
		}
	}
}

// putChar places a character at the current cursor position
func (tv *TerminalView) putChar(char rune) {
	// Auto-wrap to next line if needed
	if tv.cursorX >= tv.width {
		tv.newline()
		tv.cursorX = 0
	}

	// Expand buffer if needed
	for tv.cursorY >= len(tv.lines) {
		tv.lines = append(tv.lines, make([]rune, tv.width))
		tv.colors = append(tv.colors, make([]tcell.Style, tv.width))
		// Initialize new line with terminal default colors, not current style
		colors := theme.Current().TerminalColors()
		defaultStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)
		for i := range tv.colors[len(tv.colors)-1] {
			tv.colors[len(tv.colors)-1][i] = defaultStyle
		}
	}

	if tv.cursorY >= 0 && tv.cursorY < len(tv.lines) &&
		tv.cursorX >= 0 && tv.cursorX < tv.width {
		tv.lines[tv.cursorY][tv.cursorX] = char
		tv.colors[tv.cursorY][tv.cursorX] = tv.currentStyle
	}

	tv.cursorX++
}

// newline moves to next line
func (tv *TerminalView) newline() {
	tv.cursorY++
}

// SetChangedFunc sets the callback for when content changes
func (tv *TerminalView) SetChangedFunc(handler func()) *TerminalView {
	tv.changedFunc = handler
	return tv
}

// GetCursor returns current cursor position
func (tv *TerminalView) GetCursor() (int, int) {
	tv.mutex.RLock()
	defer tv.mutex.RUnlock()
	return tv.cursorX, tv.cursorY
}

// GetLineCount returns the number of lines in the terminal
func (tv *TerminalView) GetLineCount() int {
	tv.mutex.RLock()
	defer tv.mutex.RUnlock()
	return len(tv.lines)
}

// Draw renders the terminal view
func (tv *TerminalView) Draw(screen tcell.Screen) {
	tv.mutex.RLock()
	defer tv.mutex.RUnlock()

	tv.Box.DrawForSubclass(screen, tv)
	x, y, width, height := tv.GetInnerRect()

	// If we have no content, just show empty terminal
	if len(tv.lines) == 0 {
		return
	}

	// Debug: show what the first few lines actually contain

	// Note: Auto-scroll logic moved to Write() method to avoid interfering with cursor positioning

	// Calculate visible range
	startRow := tv.scrollOffsetRow
	endRow := startRow + height
	if endRow > len(tv.lines) {
		endRow = len(tv.lines)
	}

	// Draw visible lines
	for row := startRow; row < endRow && row-startRow < height; row++ {
		if row < 0 || row >= len(tv.lines) {
			continue
		}

		screenY := y + (row - startRow)
		line := tv.lines[row]
		colors := tv.colors[row]

		startCol := tv.scrollOffsetCol
		endCol := startCol + width
		if endCol > len(line) {
			endCol = len(line)
		}

		for col := startCol; col < endCol && col-startCol < width; col++ {
			if col < 0 || col >= len(line) {
				continue
			}

			screenX := x + (col - startCol)
			char := line[col]
			style := colors[col]

			if char == 0 {
				char = ' '
			}

			screen.SetContent(screenX, screenY, char, nil, style)
		}
	}
}

// ANSI sequence handlers
func (tv *TerminalView) handleColorSequence(params string) {
	if tv.ansiConverter == nil {
		return
	}

	// Convert ANSI params directly to tcell.Style (much more efficient!)
	tv.currentStyle = tv.ansiConverter.ConvertToTCellStyle(params)
}

func (tv *TerminalView) handleCursorPosition(params string) {
	row, col := 1, 1

	if params != "" {
		parts := strings.Split(params, ";")
		if len(parts) >= 1 {
			if r := parseInt(parts[0]); r > 0 {
				row = r
			}
		}
		if len(parts) >= 2 {
			if c := parseInt(parts[1]); c > 0 {
				col = c
			}
		}
	}

	// Cursor position should be relative to the current visible area, not absolute buffer position
	tv.cursorY = tv.scrollOffsetRow + (row - 1)
	tv.cursorX = col - 1

	// Bounds checking
	if tv.cursorY < 0 {
		tv.cursorY = 0
	}
	if tv.cursorX < 0 {
		tv.cursorX = 0
	}

	// If cursor moved significantly backward after a clear screen operation,
	// adjust scroll position to show the cursor area
	_, _, _, height := tv.GetInnerRect()
	if tv.cursorY < tv.scrollOffsetRow && len(tv.lines) > height {
		// Cursor moved to an area that's not visible, adjust scroll to show it
		tv.scrollOffsetRow = tv.cursorY
		if tv.scrollOffsetRow < 0 {
			tv.scrollOffsetRow = 0
		}
	}
}

func (tv *TerminalView) handleCursorUp(params string) {
	n := 1
	if params != "" {
		if parsed := parseInt(params); parsed > 0 {
			n = parsed
		}
	}
	tv.cursorY -= n
	if tv.cursorY < 0 {
		tv.cursorY = 0
	}
}

func (tv *TerminalView) handleCursorDown(params string) {
	n := 1
	if params != "" {
		if parsed := parseInt(params); parsed > 0 {
			n = parsed
		}
	}
	tv.cursorY += n
}

func (tv *TerminalView) handleCursorRight(params string) {
	n := 1
	if params != "" {
		if parsed := parseInt(params); parsed > 0 {
			n = parsed
		}
	}
	tv.cursorX += n
}

func (tv *TerminalView) handleCursorLeft(params string) {
	n := 1
	if params != "" {
		if parsed := parseInt(params); parsed > 0 {
			n = parsed
		}
	}
	tv.cursorX -= n
	if tv.cursorX < 0 {
		tv.cursorX = 0
	}
}

func (tv *TerminalView) handleEraseDisplay(params string) {
	n := 0
	if params != "" {
		n = parseInt(params)
	}

	switch n {
	case 0: // Clear from cursor to end of screen
		tv.clearFromCursor()
	case 1: // Clear from beginning to cursor
		tv.clearToCursor()
	case 2: // Clear entire screen
		tv.clearScreen()
	}
}

func (tv *TerminalView) handleEraseLine(params string) {
	n := 0
	if params != "" {
		n = parseInt(params)
	}

	switch n {
	case 0: // Clear from cursor to end of line
		tv.clearLineFromCursor()
	case 1: // Clear from beginning of line to cursor
		tv.clearLineToCursor()
	case 2: // Clear entire line
		tv.clearLine()
	}
}

// Helper function to parse integers
func parseInt(s string) int {
	result := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			break
		}
	}
	return result
}

// Clear helper functions
func (tv *TerminalView) clearFromCursor() {
	if tv.cursorY < 0 || tv.cursorY >= len(tv.lines) {
		return
	}

	// Use terminal default colors for clearing
	colors := theme.Current().TerminalColors()
	clearStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)

	// Clear from cursor to end of current line
	for x := tv.cursorX; x < tv.width && x < len(tv.lines[tv.cursorY]); x++ {
		tv.lines[tv.cursorY][x] = ' '
		tv.colors[tv.cursorY][x] = clearStyle
	}

	// Clear all lines below
	for y := tv.cursorY + 1; y < len(tv.lines); y++ {
		for x := 0; x < len(tv.lines[y]); x++ {
			tv.lines[y][x] = ' '
			tv.colors[y][x] = clearStyle
		}
	}
}

func (tv *TerminalView) clearToCursor() {
	// Use terminal default colors for clearing
	colors := theme.Current().TerminalColors()
	clearStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)

	// Clear all lines above
	for y := 0; y < tv.cursorY && y < len(tv.lines); y++ {
		for x := 0; x < len(tv.lines[y]); x++ {
			tv.lines[y][x] = ' '
			tv.colors[y][x] = clearStyle
		}
	}

	// Clear from beginning of current line to cursor
	if tv.cursorY >= 0 && tv.cursorY < len(tv.lines) {
		for x := 0; x <= tv.cursorX && x < len(tv.lines[tv.cursorY]); x++ {
			tv.lines[tv.cursorY][x] = ' '
			tv.colors[tv.cursorY][x] = clearStyle
		}
	}
}

func (tv *TerminalView) clearScreen() {
	_, _, _, height := tv.GetInnerRect()

	// Clear screen means: make old content scroll off-screen
	// Add enough empty lines to push all existing content above the visible area
	if len(tv.lines) > 0 {
		// Add height empty lines so existing content scrolls off-screen
		for i := 0; i < height; i++ {
			tv.lines = append(tv.lines, make([]rune, tv.width))
			newColorLine := make([]tcell.Style, tv.width)
			// Initialize with terminal default colors
			colors := theme.Current().TerminalColors()
			defaultStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)
			for j := range newColorLine {
				newColorLine[j] = defaultStyle
			}
			tv.colors = append(tv.colors, newColorLine)
		}

		// Set scroll position so the new empty lines are at the top of the view
		tv.scrollOffsetRow = len(tv.lines) - height
	} else {
		tv.scrollOffsetRow = 0
	}
	tv.scrollOffsetCol = 0

	// Cursor position will be set by subsequent cursor home command relative to current scroll position
}

func (tv *TerminalView) clearLineFromCursor() {
	if tv.cursorY < 0 || tv.cursorY >= len(tv.lines) {
		return
	}

	// Use terminal default colors for clearing
	colors := theme.Current().TerminalColors()
	clearStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)

	for x := tv.cursorX; x < len(tv.lines[tv.cursorY]); x++ {
		tv.lines[tv.cursorY][x] = ' '
		tv.colors[tv.cursorY][x] = clearStyle
	}
}

func (tv *TerminalView) clearLineToCursor() {
	if tv.cursorY < 0 || tv.cursorY >= len(tv.lines) {
		return
	}

	// Use terminal default colors for clearing
	colors := theme.Current().TerminalColors()
	clearStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)

	for x := 0; x <= tv.cursorX && x < len(tv.lines[tv.cursorY]); x++ {
		tv.lines[tv.cursorY][x] = ' '
		tv.colors[tv.cursorY][x] = clearStyle
	}
}

func (tv *TerminalView) clearLine() {
	if tv.cursorY < 0 || tv.cursorY >= len(tv.lines) {
		return
	}

	// Use terminal default colors for clearing
	colors := theme.Current().TerminalColors()
	clearStyle := tcell.StyleDefault.Foreground(colors.Foreground).Background(colors.Background)

	for x := 0; x < len(tv.lines[tv.cursorY]); x++ {
		tv.lines[tv.cursorY][x] = ' '
		tv.colors[tv.cursorY][x] = clearStyle
	}
}

// GetWrapper returns the wrapper component
func (tv *TerminalView) GetWrapper() *tview.Flex {
	return tv.wrapper
}

// UpdateContent triggers a redraw
func (tv *TerminalView) UpdateContent() {
	// The TerminalView handles its own updates internally
}

// ScrollTo scrolls to specific position
func (tv *TerminalView) ScrollTo(row, column int) *TerminalView {
	tv.mutex.Lock()
	defer tv.mutex.Unlock()

	tv.scrollOffsetRow = int(math.Max(0, float64(row)))
	tv.scrollOffsetCol = int(math.Max(0, float64(column)))
	return tv
}

// GetScrollOffset returns current scroll position
func (tv *TerminalView) GetScrollOffset() (row, column int) {
	tv.mutex.RLock()
	defer tv.mutex.RUnlock()
	return tv.scrollOffsetRow, tv.scrollOffsetCol
}

// SetScrollable sets whether the view can be scrolled
func (tv *TerminalView) SetScrollable(scrollable bool) *TerminalView {
	tv.scrollable = scrollable
	return tv
}

// SetBackgroundColor sets the background color
func (tv *TerminalView) SetBackgroundColor(color tcell.Color) *TerminalView {
	tv.Box.SetBackgroundColor(color)
	return tv
}

// SetBorder sets whether to show a border
func (tv *TerminalView) SetBorder(show bool) *TerminalView {
	tv.Box.SetBorder(show)
	return tv
}

// Clear clears all content
func (tv *TerminalView) Clear() *TerminalView {
	tv.mutex.Lock()
	defer tv.mutex.Unlock()

	tv.lines = tv.lines[:0]
	tv.colors = tv.colors[:0]
	tv.cursorX = 0
	tv.cursorY = 0
	tv.scrollOffsetRow = 0
	tv.scrollOffsetCol = 0

	return tv
}

// InputHandler handles key events for terminal scrolling
func (tv *TerminalView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return tv.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		tv.mutex.Lock()
		defer tv.mutex.Unlock()

		_, _, _, height := tv.GetInnerRect()
		totalLines := len(tv.lines)

		switch event.Key() {
		case tcell.KeyPgUp:
			// Page up - scroll up by page height
			newRow := tv.scrollOffsetRow - height
			if newRow < 0 {
				newRow = 0
			}
			tv.scrollOffsetRow = newRow

		case tcell.KeyPgDn:
			// Page down - scroll down by page height
			newRow := tv.scrollOffsetRow + height
			if newRow+height > totalLines {
				newRow = totalLines - height
				if newRow < 0 {
					newRow = 0
				}
			}
			tv.scrollOffsetRow = newRow

		default:
			// Pass other keys to parent handler
			if tv.Box.InputHandler() != nil {
				tv.Box.InputHandler()(event, setFocus)
			}
		}
	})
}
