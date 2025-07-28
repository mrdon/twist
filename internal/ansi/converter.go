package ansi

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"twist/internal/theme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ColorState represents the current terminal color state
type ColorState struct {
	foreground tcell.Color // Current foreground color
	background tcell.Color // Current background color
	modifiers  []string    // Current modifiers (bold, underline, etc.)
}

// ThemeAwareANSIWriter wraps tview.ANSIWriter to convert ANSI colors to theme colors in streaming fashion
type ThemeAwareANSIWriter struct {
	ansiWriter    io.Writer
	partialEscape []byte      // Only for incomplete escape sequences at end of data
	state         ColorState  // Current color state
	initialized   bool        // Whether we've set initial state
	debugLogger   *log.Logger // Dedicated logger for ANSI debugging
}

// NewThemeAwareANSIWriter creates a new theme-aware ANSI writer
func NewThemeAwareANSIWriter(textView *tview.TextView) *ThemeAwareANSIWriter {
	// Create dedicated debug log file for ANSI processing
	debugFile, err := os.OpenFile("ansi_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to create ANSI debug log: %v", err)
	}
	debugLogger := log.New(debugFile, "[ANSI] ", log.LstdFlags|log.Lshortfile)

	return &ThemeAwareANSIWriter{
		ansiWriter:    tview.ANSIWriter(textView),
		partialEscape: make([]byte, 0, 16), // Small buffer for incomplete escapes only
		debugLogger:   debugLogger,
	}
}

// initializeState sets up the initial color state with theme defaults
func (w *ThemeAwareANSIWriter) initializeState() {
	if w.initialized {
		return
	}
	
	currentTheme := theme.Current()
	colors := currentTheme.TerminalColors()
	
	w.state.foreground = colors.Foreground
	w.state.background = colors.Background
	w.state.modifiers = nil
	w.initialized = true
	
	w.debugLogger.Printf("STATE: Initialized with theme defaults - FG: %v, BG: %v", colors.Foreground, colors.Background)
}

// stateToANSI converts the current state to a complete ANSI escape sequence
func (w *ThemeAwareANSIWriter) stateToANSI() string {
	fgCode, _ := colorToANSI(w.state.foreground)
	_, bgCode := colorToANSI(w.state.background)
	
	// Build complete sequence: modifiers + foreground + background
	var parts []string
	parts = append(parts, w.state.modifiers...)
	parts = append(parts, fgCode, bgCode)
	
	return fmt.Sprintf("\x1b[%sm", strings.Join(parts, ";"))
}

// Write implements io.Writer, converting ANSI colors to theme colors in streaming fashion
func (w *ThemeAwareANSIWriter) Write(data []byte) (int, error) {
	// Initialize state on first write
	w.initializeState()
	
	// Log what the server sent us
	if len(data) > 0 {
		sample := string(data)
		if len(sample) > 200 {
			sample = sample[:200] + "..."
		}
		w.debugLogger.Printf("SERVER SENT: %q", sample)

		// Check for the specific strings we're looking for
		if strings.Contains(string(data), `\/`) {
			w.debugLogger.Printf("*** FOUND \\/ STRING - Current state: FG=%v BG=%v", w.state.foreground, w.state.background)
			w.debugLogger.Printf("*** Raw data containing \\/: %q", string(data))
		}
		if strings.Contains(string(data), "Y") {
			w.debugLogger.Printf("*** FOUND Y CHARACTER - Current state: FG=%v BG=%v", w.state.foreground, w.state.background)
			w.debugLogger.Printf("*** Raw data containing Y: %q", string(data))
		}
		if strings.Contains(string(data), "/     \\") {
			w.debugLogger.Printf("*** FOUND /     \\ STRING - Current state: FG=%v BG=%v", w.state.foreground, w.state.background)
			w.debugLogger.Printf("*** Raw data containing /     \\: %q", string(data))
		}
	}

	// Combine any partial escape from previous write with new data
	fullData := append(w.partialEscape, data...)

	// Process data and update state
	processed, remaining := w.processData(fullData)

	// Store any incomplete escape sequence for next write
	w.partialEscape = w.partialEscape[:0]
	w.partialEscape = append(w.partialEscape, remaining...)

	// Log how we translated it
	if len(processed) != len(data) || string(processed) != string(data) {
		processedSample := string(processed)
		if len(processedSample) > 200 {
			processedSample = processedSample[:200] + "..."
		}
		w.debugLogger.Printf("TRANSLATION: %q -> %q", string(data), processedSample)
	}

	// Write processed data to underlying ANSIWriter
	_, err := w.ansiWriter.Write(processed)

	return len(data), err // Return original data length
}

// processData processes input data, updating state only when ANSI sequences are encountered
func (w *ThemeAwareANSIWriter) processData(data []byte) (processed []byte, remaining []byte) {
	if len(data) == 0 {
		return data, nil
	}

	// Handle the case where we need to prepend initial state for plain text
	needsInitialState := !w.initialized && !containsANSI(data)
	if needsInitialState {
		w.initializeState()
		stateSeq := w.stateToANSI()
		processed = append(processed, []byte(stateSeq)...)
		processed = append(processed, data...)
		return processed, nil
	}

	processed = make([]byte, 0, len(data)) // Pre-allocate same size
	i := 0
	textStart := 0 // Track start of plain text

	for i < len(data) {
		if data[i] == '\x1b' && i+1 < len(data) && data[i+1] == '[' {
			// Found start of ANSI escape sequence
			
			// First, copy any plain text before this escape sequence
			if i > textStart {
				processed = append(processed, data[textStart:i]...)
			}
			
			escStart := i
			i += 2 // Skip \x1b[

			// Find the end of the escape sequence
			paramStart := i
			for i < len(data) && (data[i] >= '0' && data[i] <= '9' || data[i] == ';') {
				i++
			}

			// Check if we have a complete escape sequence
			if i >= len(data) {
				// Incomplete escape sequence at end of data - save for next write
				remaining = make([]byte, len(data)-escStart)
				copy(remaining, data[escStart:])
				return processed, remaining
			}

			// Check if this is a color command (ends with 'm')
			if data[i] == 'm' {
				// Extract parameters and update state
				paramStr := string(data[paramStart:i])
				w.updateState(paramStr)
				
				// Check if this is a reset that needs special handling
				if paramStr == "0" || paramStr == "" {
					// For reset, explicitly turn off attributes and set theme defaults
					currentTheme := theme.Current()
					colors := currentTheme.TerminalColors()
					fgCode, _ := colorToANSI(colors.Foreground)
					_, bgCode := colorToANSI(colors.Background)
					// Use 22 (no bold), 24 (no underline), 27 (no reverse) + colors
					resetSeq := fmt.Sprintf("\x1b[22;24;27;%s;%sm", fgCode, bgCode)
					processed = append(processed, []byte(resetSeq)...)
					w.debugLogger.Printf("RESET: Generated explicit attribute reset: %q", resetSeq)
				} else {
					// Output current state as complete ANSI sequence (replaces server's sequence)
					stateSeq := w.stateToANSI()
					w.debugLogger.Printf("GENERATED SEQUENCE: %q for params %q", stateSeq, paramStr)
					processed = append(processed, []byte(stateSeq)...)
				}
				i++ // Skip the 'm'
			} else {
				// Non-color escape sequence - copy as-is
				processed = append(processed, data[escStart:i+1]...)
				i++
			}
			
			// Update text start for next plain text segment
			textStart = i
		} else {
			// Regular character - just advance, we'll copy it with the text segment
			i++
		}
	}
	
	// Copy any remaining plain text at the end
	if textStart < len(data) {
		processed = append(processed, data[textStart:]...)
	}

	return processed, nil
}

// containsANSI checks if data contains any ANSI escape sequences
func containsANSI(data []byte) bool {
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\x1b' && data[i+1] == '[' {
			return true
		}
	}
	return false
}

// updateState updates the current color state based on ANSI escape parameters
func (w *ThemeAwareANSIWriter) updateState(params string) {
	currentTheme := theme.Current()
	colors := currentTheme.TerminalColors()
	ansiPalette := currentTheme.ANSIColorPalette()

	w.debugLogger.Printf("STATE: Updating state with params: %q", params)

	// Handle reset/clear - restore theme defaults
	if params == "" || params == "0" {
		w.state.foreground = colors.Foreground
		w.state.background = colors.Background
		w.state.modifiers = nil
		w.debugLogger.Printf("STATE: Reset to theme defaults - FG: %v, BG: %v", colors.Foreground, colors.Background)
		return
	}

	// Check for true color sequences - parse RGB values
	if strings.Contains(params, "38;2;") || strings.Contains(params, "48;2;") {
		// For true colors, we could parse RGB but for now pass through
		// TODO: Parse RGB values and update state.foreground/background
		w.debugLogger.Printf("STATE: True color sequence (not fully parsed): %q", params)
		return
	}

	if strings.Contains(params, "38;5;") || strings.Contains(params, "48;5;") {
		// For 256-color, we could parse but for now pass through
		w.debugLogger.Printf("STATE: 256-color sequence (not fully parsed): %q", params)
		return
	}

	// Process basic ANSI color codes
	parts := strings.Split(params, ";")
	var newModifiers []string
	isBold := false

	for _, part := range parts {
		code, err := strconv.Atoi(part)
		if err != nil {
			// Keep non-numeric codes as modifiers
			newModifiers = append(newModifiers, part)
			continue
		}

		if code == 1 {
			// Bold modifier - affects subsequent colors in same sequence
			isBold = true
			newModifiers = append(newModifiers, part)
			w.debugLogger.Printf("STATE: Added bold modifier")
		} else if code >= 30 && code <= 37 {
			// Standard foreground colors
			if isBold {
				// Bold + basic color = bright color
				w.state.foreground = ansiPalette[code-30+8]
				w.debugLogger.Printf("STATE: Bold FG %d -> bright color %v", code, w.state.foreground)
			} else {
				w.state.foreground = ansiPalette[code-30]
				w.debugLogger.Printf("STATE: FG %d -> color %v", code, w.state.foreground)
			}
		} else if code >= 90 && code <= 97 {
			// Bright foreground colors
			w.state.foreground = ansiPalette[code-90+8]
			w.debugLogger.Printf("STATE: Bright FG %d -> color %v", code, w.state.foreground)
		} else if code >= 40 && code <= 47 {
			// Standard background colors - background doesn't get bright with bold
			w.state.background = ansiPalette[code-40]
			w.debugLogger.Printf("STATE: BG %d -> color %v", code, w.state.background)
		} else if code >= 100 && code <= 107 {
			// Bright background colors
			w.state.background = ansiPalette[code-100+8]
			w.debugLogger.Printf("STATE: Bright BG %d -> color %v", code, w.state.background)
		} else if code != 1 {
			// Other modifiers (underline, blink, reverse, etc.)
			newModifiers = append(newModifiers, part)
			w.debugLogger.Printf("STATE: Added modifier: %s", part)
		}
	}

	// Update modifiers
	w.state.modifiers = newModifiers
	w.debugLogger.Printf("STATE: Final state - FG: %v, BG: %v, Modifiers: %v", w.state.foreground, w.state.background, w.state.modifiers)
}

// colorToANSI converts a tcell.Color to true color ANSI codes using exact RGB values
func colorToANSI(color tcell.Color) (fg, bg string) {
	// Get exact RGB values from the theme color
	r, g, b := color.RGB()

	// Use true color (24-bit) ANSI codes to ensure exact color reproduction
	// Format: 38;2;r;g;b for foreground, 48;2;r;g;b for background
	fg = fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
	bg = fmt.Sprintf("48;2;%d;%d;%d", r, g, b)

	return fg, bg
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ANSIConverter interface for converting ANSI codes during terminal parsing
type ANSIConverter interface {
	ConvertANSIParams(params string) (fgHex, bgHex string, bold, underline, reverse bool)
}

// ConvertANSIParams converts ANSI escape parameters to true color hex codes
func (w *ThemeAwareANSIWriter) ConvertANSIParams(params string) (fgHex, bgHex string, bold, underline, reverse bool) {
	// Initialize state if needed
	w.initializeState()
	
	// Get theme colors
	currentTheme := theme.Current()
	colors := currentTheme.TerminalColors()
	ansiPalette := currentTheme.ANSIColorPalette()

	w.debugLogger.Printf("CONVERT: Converting ANSI params: %q", params)

	// Handle reset/clear FIRST before setting any state
	if params == "" || params == "0" {
		w.debugLogger.Printf("CONVERT: Reset to theme defaults")
		// Update internal state completely
		w.state.foreground = colors.Foreground
		w.state.background = colors.Background
		w.state.modifiers = nil
		// Return theme defaults with all attributes reset
		fgR, fgG, fgB := colors.Foreground.RGB()
		bgR, bgG, bgB := colors.Background.RGB()
		return fmt.Sprintf("#%02x%02x%02x", fgR, fgG, fgB), fmt.Sprintf("#%02x%02x%02x", bgR, bgG, bgB), false, false, false
	}

	// Start with current state, not theme defaults
	fg := w.state.foreground
	bg := w.state.background
	
	// Get current modifiers from state
	bold = contains(w.state.modifiers, "1")
	underline = contains(w.state.modifiers, "4") 
	reverse = contains(w.state.modifiers, "7")

	// Process parameters sequentially
	parts := strings.Split(params, ";")
	for _, part := range parts {
		code, err := strconv.Atoi(part)
		if err != nil {
			continue // Skip non-numeric
		}

		switch {
		case code == 0: // Reset
			fg = colors.Foreground
			bg = colors.Background
			bold, underline, reverse = false, false, false
			w.debugLogger.Printf("CONVERT: Reset all")
			
		case code == 1: // Bold
			bold = true
			w.debugLogger.Printf("CONVERT: Set bold")
			
		case code == 4: // Underline
			underline = true
			w.debugLogger.Printf("CONVERT: Set underline")
			
		case code == 7: // Reverse
			reverse = true
			w.debugLogger.Printf("CONVERT: Set reverse")
			
		case code >= 30 && code <= 37: // Standard foreground colors
			colorIndex := code - 30
			if bold {
				// Bold makes colors bright
				fg = ansiPalette[colorIndex+8]
				w.debugLogger.Printf("CONVERT: Bold FG %d -> bright color %v", code, fg)
			} else {
				fg = ansiPalette[colorIndex]
				w.debugLogger.Printf("CONVERT: FG %d -> color %v", code, fg)
			}
			
		case code >= 40 && code <= 47: // Standard background colors
			colorIndex := code - 40
			bg = ansiPalette[colorIndex]
			bgR, bgG, bgB := bg.RGB()
			w.debugLogger.Printf("CONVERT: BG %d -> color %v (RGB: %d,%d,%d)", code, bg, bgR, bgG, bgB)
			if code == 40 {
				w.debugLogger.Printf("*** BLACK BACKGROUND SET: code=40, colorIndex=%d, result=%v ***", colorIndex, bg)
			}
			if code == 42 {
				w.debugLogger.Printf("*** GREEN BACKGROUND SET: code=42, colorIndex=%d, result=%v ***", colorIndex, bg)
			}
			
		case code >= 90 && code <= 97: // Bright foreground colors
			colorIndex := code - 90
			fg = ansiPalette[colorIndex+8]
			w.debugLogger.Printf("CONVERT: Bright FG %d -> color %v", code, fg)
			
		case code >= 100 && code <= 107: // Bright background colors
			colorIndex := code - 100
			bg = ansiPalette[colorIndex+8]
			w.debugLogger.Printf("CONVERT: Bright BG %d -> color %v", code, bg)
			
		case code == 39: // Default foreground
			fg = colors.Foreground
			w.debugLogger.Printf("CONVERT: Default FG")
			
		case code == 49: // Default background
			bg = colors.Background
			w.debugLogger.Printf("CONVERT: Default BG")
		}
	}
	
	// IMPORTANT: After processing all parameters, re-evaluate existing colors if bold was set
	// This handles the case where bold is applied as a standalone modifier (e.g., \x1b[1m)
	if bold {
		// Check if the current foreground color is a standard color that should become bright
		for i, standardColor := range ansiPalette[:8] {
			if fg == standardColor {
				// Convert standard color to bright color
				fg = ansiPalette[i+8]
				w.debugLogger.Printf("CONVERT: Re-evaluated existing color %d to bright color %v due to bold", i+30, fg)
				break
			}
		}
	}
	
	// Update internal state with new values
	w.state.foreground = fg
	w.state.background = bg
	
	// Update modifiers
	var newModifiers []string
	if bold {
		newModifiers = append(newModifiers, "1")
	}
	if underline {
		newModifiers = append(newModifiers, "4")
	}
	if reverse {
		newModifiers = append(newModifiers, "7")
	}
	w.state.modifiers = newModifiers
	
	// Convert to hex
	fgR, fgG, fgB := fg.RGB()
	bgR, bgG, bgB := bg.RGB()
	fgHex = fmt.Sprintf("#%02x%02x%02x", fgR, fgG, fgB)
	bgHex = fmt.Sprintf("#%02x%02x%02x", bgR, bgG, bgB)
	
	// Special logging for black background
	if bgR == 0 && bgG == 0 && bgB == 0 {
		w.debugLogger.Printf("*** FINAL STATE: BLACK BACKGROUND SET - BG RGB: (%d,%d,%d) HEX: %s ***", bgR, bgG, bgB, bgHex)
	}
	
	w.debugLogger.Printf("CONVERT: Final result - FG: %s, BG: %s, Bold: %t, Underline: %t, Reverse: %t", fgHex, bgHex, bold, underline, reverse)
	return fgHex, bgHex, bold, underline, reverse
}