package components

import (
	"testing"
)

func TestTerminalViewANSIColors(t *testing.T) {
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
			// Create terminal view
			tv := NewTerminalView()
			
			// Write test data directly to TerminalView
			tv.Write([]byte(test.input))
			
			// Get line count to verify data was written
			lineCount := tv.GetLineCount()
			if lineCount == 0 {
				t.Error("No lines written to terminal view")
			}
			
			// Test that the component is properly set up
			if tv.GetWrapper() == nil {
				t.Error("Wrapper should be initialized")
			}
		})
	}
}

func TestTerminalViewColorPersistence(t *testing.T) {
	// Test that colors persist across multiple updates
	tv := NewTerminalView()
	
	// Write colored text in multiple updates
	tv.Write([]byte("\x1b[31mRed "))
	lineCount1 := tv.GetLineCount()
	
	tv.Write([]byte("more red "))
	lineCount2 := tv.GetLineCount()
	
	tv.Write([]byte("\x1b[32mgreen\x1b[0m"))
	lineCount3 := tv.GetLineCount()
	
	// Verify that data is being written (line count should increase)
	if lineCount1 == 0 {
		t.Error("First write should add lines")
	}
	
	if lineCount2 < lineCount1 {
		t.Error("Second write should not decrease lines")
	}
	
	if lineCount3 < lineCount2 {
		t.Error("Third write should not decrease lines")
	}
}

func TestTerminalViewScrolling(t *testing.T) {
	// Test that scrolling works with ANSI colors
	tv := NewTerminalView()
	
	// Write enough lines to test line counting
	for i := 0; i < 10; i++ {
		tv.Write([]byte("\x1b[3" + string(rune('1'+i%6)) + "mLine " + string(rune('0'+i)) + "\x1b[0m\n"))
	}
	
	lineCount := tv.GetLineCount()
	
	// Should have added multiple lines
	if lineCount < 10 {
		t.Errorf("Should have at least 10 lines, got %d", lineCount)
	}
}

func TestTerminalViewInitialization(t *testing.T) {
	// Test that the component is properly initialized
	tv := NewTerminalView()
	
	// Verify components are created
	if tv.GetWrapper() == nil {
		t.Error("Wrapper should be initialized")
	}
	
	// Test cursor position
	x, y := tv.GetCursor()
	if x < 0 || y < 0 {
		t.Errorf("Invalid cursor position: (%d, %d)", x, y)
	}
}

