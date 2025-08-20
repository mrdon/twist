package terminal

import (
	"strings"
	"testing"
	"twist/internal/ansi"
)

// MockTerminal to track processANSISequence calls
type MockTerminal struct {
	*Terminal
	processedSequences []string
}

func (mt *MockTerminal) processANSISequence(sequence []byte) {
	mt.processedSequences = append(mt.processedSequences, string(sequence))
	// Call the original method
	mt.Terminal.processANSISequence(sequence)
}

func TestANSISequenceParsing(t *testing.T) {
	// Create a terminal with a real converter
	converter := ansi.NewColorConverter()
	terminal := NewTerminalWithConverter(80, 24, converter)

	// Test the sequence: "\x1b[31m NO \x1b[1mPVP\x1b[36m"
	testData := []byte("\x1b[31m NO \x1b[1mPVP\x1b[36m")

	t.Logf("Processing test data: %q", string(testData))
	t.Logf("Raw bytes: %x", testData)

	// Process the data
	terminal.Write(testData)

	// Verify that the text was written correctly using new rune buffer
	runes := terminal.GetRunes()
	if len(runes) == 0 {
		t.Fatal("No runes in terminal buffer")
	}

	// Check that we have the expected characters
	expectedText := " NO PVP"
	actualText := ""
	for _, char := range runes[0] {
		if char != 0 && char != ' ' || (char == ' ' && len(actualText) > 0) {
			actualText += string(char)
		}
	}

	actualText = strings.TrimSpace(actualText)
	if actualText != strings.TrimSpace(expectedText) {
		t.Errorf("Expected text %q, got %q", strings.TrimSpace(expectedText), actualText)
	}

	t.Logf("✓ Successfully processed ANSI sequences and rendered text: %q", actualText)
}

func TestNewDataStructure(t *testing.T) {
	// Create a terminal with a real converter
	converter := ansi.NewColorConverter()
	terminal := NewTerminalWithConverter(80, 24, converter)

	// Test the sequence: "\x1b[31m NO \x1b[1mPVP\x1b[36m"
	testData := []byte("\x1b[31m NO \x1b[1mPVP\x1b[36m")

	// Process the data
	terminal.Write(testData)

	// Test the new rune buffer
	runes := terminal.GetRunes()
	if len(runes) == 0 {
		t.Fatal("No runes in terminal buffer")
	}

	// Check that we have the expected characters in rune buffer
	expectedText := " NO PVP"
	actualRuneText := ""
	for _, char := range runes[0] {
		if char != 0 && char != ' ' || (char == ' ' && len(actualRuneText) > 0) {
			actualRuneText += string(char)
		}
	}

	actualRuneText = strings.TrimSpace(actualRuneText)
	if actualRuneText != strings.TrimSpace(expectedText) {
		t.Errorf("Rune buffer: Expected text %q, got %q", strings.TrimSpace(expectedText), actualRuneText)
	}

	// Test the color changes
	colorChanges := terminal.GetColorChanges()
	if len(colorChanges) == 0 {
		t.Fatal("No color changes recorded")
	}

	t.Logf("Recorded %d color changes:", len(colorChanges))
	for i, change := range colorChanges {
		t.Logf("  %d: Position (%d,%d) -> %s", i, change.X, change.Y, change.TViewTag)
	}

	// Should have at least 3 color changes (red, bold, cyan)
	if len(colorChanges) < 3 {
		t.Errorf("Expected at least 3 color changes, got %d", len(colorChanges))
	}

	t.Logf("✓ New data structure working: rune buffer has %q, %d color changes recorded", actualRuneText, len(colorChanges))
}

// TerminalState holds both the runes and the current terminal state
type TerminalState struct {
	Runes                [][]rune
	ColorChanges         []ColorChange
	CurrentForegroundHex string
	CurrentBackgroundHex string
	CurrentBold          bool
	CurrentUnderline     bool
	CurrentReverse       bool
}

// ProcessANSIString processes an ANSI string using the real ANSI converter and returns the resulting runes and terminal state for testing
func ProcessANSIString(ansiString string, width, height int) TerminalState {
	// Create the simplified color converter
	converter := ansi.NewColorConverter()

	// Create terminal with the real converter
	terminal := NewTerminalWithConverter(width, height, converter)

	// Process the ANSI string
	terminal.Write([]byte(ansiString))

	// Get the runes and current state
	runes := terminal.GetRunes()
	colorChanges := terminal.GetColorChanges()
	fgHex, bgHex, bold, underline, reverse := terminal.GetCurrentColors()

	return TerminalState{
		Runes:                runes,
		ColorChanges:         colorChanges,
		CurrentForegroundHex: fgHex,
		CurrentBackgroundHex: bgHex,
		CurrentBold:          bold,
		CurrentUnderline:     underline,
		CurrentReverse:       reverse,
	}
}

// TestProcessANSIString tests the ProcessANSIString helper function
func TestProcessANSIString(t *testing.T) {
	state := ProcessANSIString("\x1b[0mHello", 80, 24)

	// Check terminal state after reset
	if state.CurrentForegroundHex != "#c0c0c0" {
		t.Errorf("Expected terminal FG #c0c0c0, got %s", state.CurrentForegroundHex)
	}
	if state.CurrentBackgroundHex != "#000000" {
		t.Errorf("Expected terminal BG #000000, got %s", state.CurrentBackgroundHex)
	}
	if state.CurrentBold {
		t.Errorf("Expected terminal bold false, got %t", state.CurrentBold)
	}

	// Check characters in rune buffer
	expected := "Hello"
	for i, expectedChar := range expected {
		if i >= len(state.Runes[0]) {
			t.Errorf("Character %d: missing from rune buffer", i)
			continue
		}
		char := state.Runes[0][i]
		if char != expectedChar {
			t.Errorf("Character %d: expected '%c', got '%c'", i, expectedChar, char)
		}
		// Note: Color information is now stored separately in ColorChanges
		// For this reset test, there should be minimal color changes
	}
}

// TestColorSequences tests various color sequences
func TestColorSequences(t *testing.T) {
	// Test red text
	state := ProcessANSIString("\x1b[31mR", 80, 24)
	if len(state.Runes[0]) == 0 || state.Runes[0][0] != 'R' {
		t.Error("Expected character 'R' in rune buffer")
	}
	if len(state.ColorChanges) == 0 {
		t.Error("Expected color change for red text")
	}
	if state.CurrentForegroundHex != "#800000" {
		t.Errorf("Expected terminal FG #800000, got %s", state.CurrentForegroundHex)
	}

	// Test bold
	state = ProcessANSIString("\x1b[1mB", 80, 24)
	if len(state.Runes[0]) == 0 || state.Runes[0][0] != 'B' {
		t.Error("Expected character 'B' in rune buffer")
	}
	if !state.CurrentBold {
		t.Error("Expected terminal bold state")
	}

	// Test green background
	state = ProcessANSIString("\x1b[42mG", 80, 24)
	if len(state.Runes[0]) == 0 || state.Runes[0][0] != 'G' {
		t.Error("Expected character 'G' in rune buffer")
	}
	if state.CurrentBackgroundHex != "#008000" {
		t.Errorf("Expected terminal BG #008000, got %s", state.CurrentBackgroundHex)
	}
}

// TestComplexANSILine tests a complex ANSI line that should be exactly 80 characters
func TestComplexANSILine(t *testing.T) {
	complexLine := "\x1b[0m\x1b[1;42m▄▄\x1b[0;32m▄ \x1b[35m▀ \x1b[31m▄\x1b[1;41m▄\x1b[0;31m▀██▀█▄▄\x1b[1;41m▄▄███\x1b[47m▄  ▀\x1b[41m█▄\x1b[0;31m▄▀\x1b[41m \x1b[40m▄ \x1b[1;30;47m▀\x1b[40m▄  ▄ \x1b[0;36m▄\x1b[1;46m▄\x1b[0;36m▄ \x1b[1;30m▄  ▄\x1b[47m▀\x1b[40m \x1b[0;31m▄█▀▄\x1b[1;41m▄██\x1b[47m  ▄\x1b[41m█████▄▄\x1b[0;31m▄▄▄▀███▓ \x1b[32m▀ ▄\x1b[1;37;42m▄\x1b[32m▄\x1b[0m"

	state := ProcessANSIString(complexLine, 80, 24)

	// Count characters in first row
	totalChars := 0
	for _, char := range state.Runes[0] {
		if char != 0 {
			totalChars++
		}
	}

	if totalChars != 80 {
		t.Errorf("Expected exactly 80 characters, got %d", totalChars)
	}

	// Should be single row
	hasSecondRowContent := false
	if len(state.Runes) > 1 {
		for _, char := range state.Runes[1] {
			if char != 0 && char != ' ' {
				hasSecondRowContent = true
				break
			}
		}
	}
	if hasSecondRowContent {
		t.Error("Expected single row, found content on second row")
	}

	// Terminal state should be reset due to \x1b[0m at end
	if state.CurrentForegroundHex != "#c0c0c0" {
		t.Errorf("Expected terminal FG #c0c0c0, got %s", state.CurrentForegroundHex)
	}
	if state.CurrentBackgroundHex != "#000000" {
		t.Errorf("Expected terminal BG #000000, got %s", state.CurrentBackgroundHex)
	}
	if state.CurrentBold {
		t.Error("Expected terminal bold false")
	}
}

// TestTwoLineString tests a string that should produce exactly two 80-character lines
func TestTwoLineString(t *testing.T) {
	twoLineString := " \x1b[1;31;41m█\x1b[0;31m▐\x1b[1;47m▓█\x1b[41m█\x1b[0;31m█  \x1b[35m▄\x1b[1m▀ \x1b[0;35m▄\x1b[1m▀  \x1b[0;35m▀\x1b[1;45m▀\x1b[0;35m▄ \x1b[1;30m▀■ \x1b[0;35m▀\x1b[1;45m▀\x1b[40m▄\x1b[0m▄▄\x1b[1;35;45m▄\x1b[40m▄\x1b[0;35m▄▀   ▄ █ \x1b[1;30m▀ \x1b[0;35m█ ▄ \x1b[1;30m▄▄\x1b[0;35m▀▄\x1b[1m▄\x1b[45m▄\x1b[0m▄▄\x1b[1;35m▄\x1b[45m▀\x1b[0;35m▀ \x1b[1;30m■▀ \x1b[0;35m▄\x1b[1;45m▀\x1b[0;35m▀  \x1b[1m▀\x1b[0;35m▄ \x1b[1m▀\x1b[0;35m▄  \x1b[31m█\x1b[1;41m███\x1b[0;31m▌\x1b[1;41m██\x1b[0m\r\n \x1b[31m█ \x1b[1;41m█\x1b[0m \x1b[31m________      \x1b[1;30m▀ ▄\x1b[0m   \x1b[1;35;45m▀\x1b[47m█\x1b[0m  \x1b[31m__      _____ \x1b[1;35;45m▄▀\x1b[0m    \x1b[1;35;45m▄▀\x1b[0;31m __    __  \x1b[1;30m▄▀    \x1b[0;35m▄\x1b[1;45m░\x1b[40m \x1b[0;31m▀ ▀\x1b[1;41m█▀\x1b[40m \x1b[41m▀\x1b[0;31m▀\x1b[37m"

	state := ProcessANSIString(twoLineString, 80, 24)

	// Count characters in both rows
	firstRowChars := 0
	for _, char := range state.Runes[0] {
		if char != 0 {
			firstRowChars++
		}
	}

	secondRowChars := 0
	for _, char := range state.Runes[1] {
		if char != 0 {
			secondRowChars++
		}
	}

	if firstRowChars != 80 {
		t.Errorf("Expected first row 80 characters, got %d", firstRowChars)
	}
	if secondRowChars != 80 {
		t.Errorf("Expected second row 80 characters, got %d", secondRowChars)
	}

	// Check terminal state for proper color handling
	// Note: Character-level color info is now in ColorChanges array
	if len(state.Runes[1]) == 0 || state.Runes[1][0] != ' ' {
		t.Error("Expected first character of row 2 to be space")
	}
}
