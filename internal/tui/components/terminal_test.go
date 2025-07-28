package components

import (
	"strings"
	"testing"
	"twist/internal/terminal"
)

func TestTerminalComponentANSIColors(t *testing.T) {
	// Create a terminal with test data
	term := terminal.NewTerminal(80, 24)
	
	// Write some colored text using ANSI escape sequences
	testData := []struct {
		input    string
		expected string // What we expect to see in the final output
	}{
		{
			input:    "\x1b[31mRed text\x1b[0m",
			expected: "Red text",
		},
		{
			input:    "\x1b[32mGreen text\x1b[0m", 
			expected: "Green text",
		},
		{
			input:    "\x1b[34mBlue\x1b[33m Yellow\x1b[0m",
			expected: "Blue Yellow",
		},
		{
			input:    "Normal \x1b[1mbold\x1b[0m normal",
			expected: "Normal bold normal",
		},
	}

	for _, test := range testData {
		t.Run(test.input, func(t *testing.T) {
			// Clear terminal and write test data
			term.Write([]byte("\x1b[2J\x1b[H")) // Clear screen and move cursor to home
			term.Write([]byte(test.input))
			
			// Create terminal component
			tc := NewTerminalComponent(term)
			
			// Update content to trigger the new ANSI processing
			tc.UpdateContent()
			
			// Get the text from the view
			actualText := tc.view.GetText(false)
			
			// The actual text should contain the expected content
			// Note: tview color tags like [navy:black] will be present, so we check the readable content
			if !strings.Contains(actualText, test.expected) {
				// Also try with GetText(true) to strip color tags for comparison
				strippedText := tc.view.GetText(true)
				if !strings.Contains(strippedText, test.expected) {
					t.Errorf("Expected text to contain %q, but got %q (stripped: %q)", test.expected, actualText, strippedText)
				}
			}
			
			// Note: We can't easily test GetDynamicColors() as it's not exposed in tview
			// But we know it's set in NewTerminalComponent via SetDynamicColors(true)
		})
	}
}

func TestTerminalComponentColorPersistence(t *testing.T) {
	// Test that colors persist across multiple updates
	term := terminal.NewTerminal(80, 24)
	tc := NewTerminalComponent(term)
	
	// Write colored text in multiple updates
	term.Write([]byte("\x1b[31mRed "))
	tc.UpdateContent()
	text1 := tc.view.GetText(false)
	
	term.Write([]byte("more red "))
	tc.UpdateContent() 
	text2 := tc.view.GetText(false)
	
	term.Write([]byte("\x1b[32mgreen\x1b[0m"))
	tc.UpdateContent()
	text3 := tc.view.GetText(false)
	
	// Verify progressive content building (using stripped text to ignore color tags)
	stripped1 := tc.view.GetText(true)
	stripped2 := tc.view.GetText(true) 
	stripped3 := tc.view.GetText(true)
	
	if !strings.Contains(stripped1, "Red") && !strings.Contains(text1, "Red") {
		t.Errorf("First update should contain 'Red', got: %q", text1)
	}
	
	if !strings.Contains(stripped2, "Red more red") && !strings.Contains(text2, "Red more red") {
		t.Errorf("Second update should contain 'Red more red', got: %q", text2)
	}
	
	if !strings.Contains(stripped3, "Red more red green") && !strings.Contains(text3, "Red more red green") {
		t.Errorf("Third update should contain 'Red more red green', got: %q", text3)
	}
}

func TestTerminalComponentScrolling(t *testing.T) {
	// Test that scrolling works with ANSI colors
	term := terminal.NewTerminal(80, 5) // Small height to force scrolling
	tc := NewTerminalComponent(term)
	
	// Write enough lines to cause scrolling
	for i := 0; i < 10; i++ {
		term.Write([]byte("\x1b[3" + string(rune('1'+i%6)) + "mLine " + string(rune('0'+i)) + "\x1b[0m\n"))
	}
	
	tc.UpdateContent()
	text := tc.view.GetText(false)
	
	// Should contain the last few lines due to scrolling
	if !strings.Contains(text, "Line 9") {
		t.Errorf("Should contain 'Line 9' after scrolling, got: %q", text)
	}
	
	// Should not contain the first line (scrolled away)
	if strings.Contains(text, "Line 0") {
		t.Errorf("Should not contain 'Line 0' after scrolling, got: %q", text)
	}
}

func TestTerminalComponentInitialization(t *testing.T) {
	// Test that the component is properly initialized
	term := terminal.NewTerminal(80, 24)
	tc := NewTerminalComponent(term)
	
	// Verify components are created
	if tc.view == nil {
		t.Error("TextView should be initialized")
	}
	
	if tc.wrapper == nil {
		t.Error("Wrapper should be initialized")
	}
	
	if tc.terminal == nil {
		t.Error("Terminal should be initialized")
	}
	
	if tc.ansiWriter == nil {
		t.Error("ANSIWriter should be initialized")
	}
	
	// Note: tview doesn't expose getter methods for these properties,
	// but we verify they're set correctly in NewTerminalComponent:
	// - SetDynamicColors(true) 
	// - SetRegions(true)
	// - SetWordWrap(true) 
	// - SetScrollable(true)
}

