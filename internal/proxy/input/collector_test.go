package input

import (
	"strings"
	"testing"
)

// MockOutputFunc captures output for testing
type MockOutputFunc struct {
	outputs []string
}

func (m *MockOutputFunc) Send(text string) {
	m.outputs = append(m.outputs, text)
}

func (m *MockOutputFunc) GetOutput() string {
	return strings.Join(m.outputs, "")
}

func (m *MockOutputFunc) GetOutputs() []string {
	return m.outputs
}

func (m *MockOutputFunc) Clear() {
	m.outputs = nil
}

func TestInputCollector_BackspaceHandling(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	// Register a test handler
	collector.RegisterCompletionHandler("TEST", func(menuName, value string) error {
		return nil
	})

	// Start collection
	collector.StartCollection("TEST", "Test prompt")
	mockOutput.Clear() // Clear the prompt output

	// Type some characters
	err := collector.HandleInput("h")
	if err != nil {
		t.Fatalf("Error handling input 'h': %v", err)
	}

	err = collector.HandleInput("e")
	if err != nil {
		t.Fatalf("Error handling input 'e': %v", err)
	}

	err = collector.HandleInput("l")
	if err != nil {
		t.Fatalf("Error handling input 'l': %v", err)
	}

	err = collector.HandleInput("l")
	if err != nil {
		t.Fatalf("Error handling input 'l': %v", err)
	}

	err = collector.HandleInput("o")
	if err != nil {
		t.Fatalf("Error handling input 'o': %v", err)
	}

	// Check buffer content
	if collector.GetBuffer() != "hello" {
		t.Errorf("Expected buffer 'hello', got '%s'", collector.GetBuffer())
	}

	// Send backspace
	err = collector.HandleInput("\b")
	if err != nil {
		t.Fatalf("Error handling backspace: %v", err)
	}

	// Check buffer after backspace
	if collector.GetBuffer() != "hell" {
		t.Errorf("Expected buffer 'hell' after backspace, got '%s'", collector.GetBuffer())
	}

	// Send another backspace
	err = collector.HandleInput("\x7f") // DEL key
	if err != nil {
		t.Fatalf("Error handling DEL key: %v", err)
	}

	// Check buffer after DEL
	if collector.GetBuffer() != "hel" {
		t.Errorf("Expected buffer 'hel' after DEL, got '%s'", collector.GetBuffer())
	}

	// Send third backspace variant
	err = collector.HandleInput("\x08") // BS key
	if err != nil {
		t.Fatalf("Error handling BS key: %v", err)
	}

	// Check buffer after BS
	if collector.GetBuffer() != "he" {
		t.Errorf("Expected buffer 'he' after BS, got '%s'", collector.GetBuffer())
	}

	// Verify backspace visual feedback was sent
	output := mockOutput.GetOutput()
	backspaceCount := strings.Count(output, "\b \b")
	if backspaceCount != 3 {
		t.Errorf("Expected 3 backspace sequences in output, got %d. Output: %q", backspaceCount, output)
	}
}

func TestInputCollector_BackspaceOnEmptyBuffer(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	collector.StartCollection("TEST", "Test prompt")
	mockOutput.Clear()

	// Send backspace on empty buffer - should not crash
	err := collector.HandleInput("\b")
	if err != nil {
		t.Fatalf("Error handling backspace on empty buffer: %v", err)
	}

	// Buffer should still be empty
	if collector.GetBuffer() != "" {
		t.Errorf("Expected empty buffer, got '%s'", collector.GetBuffer())
	}

	// No backspace sequence should be sent for empty buffer
	output := mockOutput.GetOutput()
	if strings.Contains(output, "\b \b") {
		t.Errorf("Unexpected backspace sequence in output for empty buffer: %q", output)
	}
}

func TestInputCollector_CharacterAccumulation(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	var completedValue string
	collector.RegisterCompletionHandler("TEST", func(menuName, value string) error {
		completedValue = value
		return nil
	})

	collector.StartCollection("TEST", "Test prompt")
	mockOutput.Clear()

	// Type characters one by one
	characters := []string{"t", "e", "s", "t", ".", "t", "s"}
	for _, char := range characters {
		err := collector.HandleInput(char)
		if err != nil {
			t.Fatalf("Error handling input '%s': %v", char, err)
		}
	}

	// Check that buffer accumulates correctly
	expected := "test.ts"
	if collector.GetBuffer() != expected {
		t.Errorf("Expected buffer '%s', got '%s'", expected, collector.GetBuffer())
	}

	// Check that each character was echoed
	output := mockOutput.GetOutput()
	for _, char := range characters {
		if !strings.Contains(output, char) {
			t.Errorf("Expected character '%s' to be echoed in output: %q", char, output)
		}
	}

	// Should still be collecting (not completed yet)
	if !collector.IsCollecting() {
		t.Error("Expected collector to still be collecting")
	}

	// Completed value should still be empty
	if completedValue != "" {
		t.Errorf("Expected completed value to be empty, got '%s'", completedValue)
	}
}

func TestInputCollector_EnterKeyCompletion(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	var completedValue string
	var completedMenu string
	collector.RegisterCompletionHandler("SCRIPT_LOAD", func(menuName, value string) error {
		completedValue = value
		completedMenu = menuName
		return nil
	})

	collector.StartCollection("SCRIPT_LOAD", "Script filename")
	mockOutput.Clear()

	// Type filename
	filename := "login.ts"
	for _, char := range filename {
		err := collector.HandleInput(string(char))
		if err != nil {
			t.Fatalf("Error handling input '%c': %v", char, err)
		}
	}

	// Press Enter to complete
	err := collector.HandleInput("\r")
	if err != nil {
		t.Fatalf("Error handling Enter key: %v", err)
	}

	// Should no longer be collecting
	if collector.IsCollecting() {
		t.Error("Expected collector to stop collecting after Enter")
	}

	// Should have completed with correct value
	if completedValue != filename {
		t.Errorf("Expected completed value '%s', got '%s'", filename, completedValue)
	}

	if completedMenu != "SCRIPT_LOAD" {
		t.Errorf("Expected completed menu 'SCRIPT_LOAD', got '%s'", completedMenu)
	}

	// Buffer should be cleared
	if collector.GetBuffer() != "" {
		t.Errorf("Expected buffer to be cleared after completion, got '%s'", collector.GetBuffer())
	}
}

func TestInputCollector_EnterKeyVariants(t *testing.T) {
	tests := []struct {
		name     string
		enterKey string
	}{
		{"Carriage Return", "\r"},
		{"Line Feed", "\n"},
		{"CRLF", "\r\n"},
		{"Empty String", ""}, // Common in terminal applications
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOutput := &MockOutputFunc{}
			collector := NewInputCollector(mockOutput.Send)

			var completedValue string
			collector.RegisterCompletionHandler("TEST", func(menuName, value string) error {
				completedValue = value
				return nil
			})

			collector.StartCollection("TEST", "Test")

			// Type some text
			collector.HandleInput("test")

			// Press Enter variant
			err := collector.HandleInput(tt.enterKey)
			if err != nil {
				t.Fatalf("Error handling %s: %v", tt.name, err)
			}

			// Should complete
			if collector.IsCollecting() {
				t.Errorf("Expected collector to stop collecting after %s", tt.name)
			}

			if completedValue != "test" {
				t.Errorf("Expected completed value 'test', got '%s'", completedValue)
			}
		})
	}
}

func TestInputCollector_EscapeSequences(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	var completedValue string
	collector.RegisterCompletionHandler("TEST", func(menuName, value string) error {
		completedValue = value
		return nil
	})

	collector.StartCollection("TEST", "Test prompt")
	mockOutput.Clear()

	// Type some text
	collector.HandleInput("s")
	collector.HandleInput("o")
	collector.HandleInput("m")
	collector.HandleInput("e")

	// Check buffer has content
	if collector.GetBuffer() != "some" {
		t.Errorf("Expected buffer 'some', got '%s'", collector.GetBuffer())
	}

	// Send escape sequence
	err := collector.HandleInput("\\")
	if err != nil {
		t.Fatalf("Error handling escape: %v", err)
	}

	// Should no longer be collecting
	if collector.IsCollecting() {
		t.Error("Expected collector to stop collecting after escape")
	}

	// Should not have completed (cancelled)
	if completedValue != "" {
		t.Errorf("Expected no completed value after escape, got '%s'", completedValue)
	}

	// Buffer should be cleared
	if collector.GetBuffer() != "" {
		t.Errorf("Expected buffer to be cleared after escape, got '%s'", collector.GetBuffer())
	}

	// Should have cancellation message
	output := mockOutput.GetOutput()
	if !strings.Contains(output, "Input cancelled") {
		t.Errorf("Expected cancellation message in output: %q", output)
	}
}

func TestInputCollector_ControlCharacterFiltering(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	collector.StartCollection("TEST", "Test prompt")
	mockOutput.Clear()

	// Send various control characters - should be ignored
	controlChars := []string{
		"\x00", // NULL
		"\x01", // SOH
		"\x02", // STX
		"\x1b", // ESC (not backslash escape)
		"\x1f", // Unit separator
	}

	for _, char := range controlChars {
		err := collector.HandleInput(char)
		if err != nil {
			t.Fatalf("Error handling control character %q: %v", char, err)
		}
	}

	// Buffer should still be empty
	if collector.GetBuffer() != "" {
		t.Errorf("Expected empty buffer after control characters, got '%s'", collector.GetBuffer())
	}

	// Now send a printable character
	err := collector.HandleInput("a")
	if err != nil {
		t.Fatalf("Error handling printable character: %v", err)
	}

	// Buffer should now contain the printable character
	if collector.GetBuffer() != "a" {
		t.Errorf("Expected buffer 'a', got '%s'", collector.GetBuffer())
	}
}

func TestInputCollector_PrintableCharacterRange(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	collector.StartCollection("TEST", "Test prompt")
	mockOutput.Clear()

	// Test boundary cases for printable ASCII
	testCases := []struct {
		char        string
		shouldAdd   bool
		description string
	}{
		{string(rune(31)), false, "Below printable range"}, // Control character
		{string(rune(32)), true, "Space character"},        // First printable
		{string(rune(126)), true, "Tilde character"},       // Last printable
		{string(rune(127)), false, "DEL character"},        // Above printable range
	}

	for _, tc := range testCases {
		// Reset collector for each test
		collector.exitCollection()
		collector.StartCollection("TEST", "Test prompt")
		mockOutput.Clear()

		err := collector.HandleInput(tc.char)
		if err != nil {
			t.Fatalf("Error handling character %q (%s): %v", tc.char, tc.description, err)
		}

		buffer := collector.GetBuffer()
		if tc.shouldAdd {
			if buffer != tc.char {
				t.Errorf("%s: Expected buffer '%s', got '%s'", tc.description, tc.char, buffer)
			}
		} else {
			if buffer != "" {
				t.Errorf("%s: Expected empty buffer for non-printable char, got '%s'", tc.description, buffer)
			}
		}
	}
}

func TestInputCollector_HelpCommand(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	collector.StartCollection("TEST", "Test prompt")
	mockOutput.Clear()

	// Type some text first
	collector.HandleInput("h")
	collector.HandleInput("e")

	// Send help command
	err := collector.HandleInput("?")
	if err != nil {
		t.Fatalf("Error handling help command: %v", err)
	}

	// Should still be collecting
	if !collector.IsCollecting() {
		t.Error("Expected collector to still be collecting after help")
	}

	// Buffer should be preserved
	if collector.GetBuffer() != "he" {
		t.Errorf("Expected buffer 'he' after help, got '%s'", collector.GetBuffer())
	}

	// Should have help text in output
	output := mockOutput.GetOutput()
	if !strings.Contains(output, "Input Collection Help") {
		t.Errorf("Expected help text in output: %q", output)
	}

	if !strings.Contains(output, "Current input: he") {
		t.Errorf("Expected current input in help output: %q", output)
	}
}

func TestInputCollector_NoPrematireCompletion(t *testing.T) {
	mockOutput := &MockOutputFunc{}
	collector := NewInputCollector(mockOutput.Send)

	var completedValue string
	var completionCount int
	collector.RegisterCompletionHandler("SCRIPT_LOAD", func(menuName, value string) error {
		completedValue = value
		completionCount++
		return nil
	})

	collector.StartCollection("SCRIPT_LOAD", "Script filename")

	// Type each character of a filename individually
	filename := "test.ts"
	for i, char := range filename {
		err := collector.HandleInput(string(char))
		if err != nil {
			t.Fatalf("Error handling character %d (%c): %v", i, char, err)
		}

		// Should still be collecting after each character
		if !collector.IsCollecting() {
			t.Errorf("Unexpected completion after character %d (%c)", i, char)
		}

		// Should not have completed yet
		if completionCount > 0 {
			t.Errorf("Premature completion after character %d (%c)", i, char)
		}
	}

	// Only after Enter should it complete
	err := collector.HandleInput("\r")
	if err != nil {
		t.Fatalf("Error handling Enter: %v", err)
	}

	// Now it should be completed
	if completionCount != 1 {
		t.Errorf("Expected exactly 1 completion, got %d", completionCount)
	}

	if completedValue != filename {
		t.Errorf("Expected completed value '%s', got '%s'", filename, completedValue)
	}
}
