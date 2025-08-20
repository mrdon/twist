package scripting

import (
	"strings"
	"testing"
)

// TestSimpleExpectBasic tests basic expect functionality
func TestSimpleExpectBasic(t *testing.T) {
	// Test expect engine directly with simulated output
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")

	// Simulate script output
	expectEngine.AddOutput("Starting port trading\r\n")
	expectEngine.AddOutput("User selected sector: 2157\r\n")
	expectEngine.AddOutput("User selected times: 3\r\n")
	expectEngine.AddOutput("Trading sector 2157 for 3 times\r\n")
	expectEngine.AddOutput("Trade completed\r\n")

	expectScript := `
timeout "1s"
expect "Starting port trading"
expect "User selected sector: 2157"
expect "Trading sector 2157 for 3 times"
assert "Starting port trading"
assert "Trade completed"
log "Basic expect test passed"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Simple expect test failed: %v", err)
	}
}

// TestSimpleExpectMultipleAsserts tests multiple assertions on script output
func TestSimpleExpectMultipleAsserts(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")

	// Add all script output
	expectEngine.AddOutput("User: TestPlayer\r\n")
	expectEngine.AddOutput("Credits: 50000\r\n")
	expectEngine.AddOutput("Sector: 42\r\n")
	expectEngine.AddOutput("Ship: Destroyer\r\n")
	expectEngine.AddOutput("Status: Active\r\n")

	expectScript := `
timeout "1s"
expect "User: TestPlayer"
expect "Credits: 50000"  
expect "Sector: 42"
expect "Ship: Destroyer"
assert "User: TestPlayer"
assert "Credits: 50000"
assert "Sector: 42" 
assert "Ship: Destroyer"
assert "Status: Active"
log "All assertions passed"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Multiple assert test failed: %v", err)
	}
}

// TestSimpleExpectTimeout demonstrates timeout behavior
func TestSimpleExpectTimeout(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")
	expectEngine.AddOutput("Some output")

	expectScript := `
timeout "100ms"
expect "This will never appear"
log "Should not reach here"
`

	err := expectEngine.Run(expectScript)
	if err == nil {
		t.Fatal("Expected timeout error but test passed")
	}

	if !strings.Contains(err.Error(), "timeout waiting") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestSimpleExpectLiteral demonstrates literal string matching
func TestSimpleExpectLiteral(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")
	expectEngine.AddOutput("Player Level: 25")
	expectEngine.AddOutput("Credits: 1,234,567")
	expectEngine.AddOutput("Ship Class: Imperial Starship")

	expectScript := `
timeout "1s"
expect "Player Level: 25"
expect "Credits: 1,234,567"
expect "Ship Class: Imperial Starship"
assert "Level: 25"
assert "Credits: 1,234,567"
log "Literal patterns matched"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Literal test failed: %v", err)
	}
}

// TestSimpleExpectSend tests the send functionality
func TestSimpleExpectSend(t *testing.T) {
	var sentInputs []string

	inputSender := func(input string) {
		sentInputs = append(sentInputs, input)
	}

	expectEngine := NewSimpleExpectEngine(t, inputSender, "\r")

	expectScript := `
send "hello world*"
send "test input*"
log "Send test completed"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Send test failed: %v", err)
	}

	expectedInputs := []string{"hello world\r", "test input\r"}
	if !slicesEqual(sentInputs, expectedInputs) {
		t.Errorf("Expected inputs %v, got %v", expectedInputs, sentInputs)
	}
}

// slicesEqual compares two string slices
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestProcessEscapeSequences tests the escape sequence processing function
func TestProcessEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "carriage return",
			input:    "hello\\rworld",
			expected: "hello\rworld",
		},
		{
			name:     "newline",
			input:    "hello\\nworld",
			expected: "hello\nworld",
		},
		{
			name:     "tab",
			input:    "hello\\tworld",
			expected: "hello\tworld",
		},
		{
			name:     "escape sequence",
			input:    "\\x1b[31mred\\x1b[0m",
			expected: "\x1b[31mred\x1b[0m",
		},
		{
			name:     "escaped backslash",
			input:    "hello\\\\world",
			expected: "hello\\world",
		},
		{
			name:     "multiple escape sequences",
			input:    "\\r\\n\\x1b[35mCommand\\x1b[0m\\r\\n",
			expected: "\r\n\x1b[35mCommand\x1b[0m\r\n",
		},
		{
			name:     "hex sequences with different values",
			input:    "\\x00\\x1b\\xff",
			expected: "\x00\x1b\xff",
		},
		{
			name:     "invalid hex sequence - too short",
			input:    "\\x1",
			expected: "\\x1",
		},
		{
			name:     "invalid hex sequence - non-hex chars",
			input:    "\\xzz",
			expected: "\\xzz",
		},
		{
			name:     "no escape sequences",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "backslash at end",
			input:    "text\\",
			expected: "text\\",
		},
		{
			name:     "unknown escape sequence",
			input:    "\\q",
			expected: "\\q",
		},
		{
			name:     "complex probe data",
			input:    "E\\r\\x1b[0m\\n\\x1b[32mSubSpace Ether Probe loaded in launch tube, \\x1b[1;33m13 \\x1b[0;32mremaining.\\r\\x1b[0m\\n",
			expected: "E\r\x1b[0m\n\x1b[32mSubSpace Ether Probe loaded in launch tube, \x1b[1;33m13 \x1b[0;32mremaining.\r\x1b[0m\n",
		},
		{
			name:     "probe sector data",
			input:    "\\x1b[33mProbe entering sector \\x1b[1m: \\x1b[36m274\\r\\x1b[0m\\n\\r\\n\\x1b[1;32mSector  \\x1b[33m: \\x1b[36m274 \\x1b[0;32min \\x1b[34muncharted space.\\r\\x1b[0m\\n",
			expected: "\x1b[33mProbe entering sector \x1b[1m: \x1b[36m274\r\x1b[0m\n\r\n\x1b[1;32mSector  \x1b[33m: \x1b[36m274 \x1b[0;32min \x1b[34muncharted space.\r\x1b[0m\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processEscapeSequences(tt.input)
			if result != tt.expected {
				t.Errorf("processEscapeSequences(%q) = %q; expected %q", tt.input, result, tt.expected)

				// Show byte-by-byte comparison for debugging
				t.Logf("Input bytes: %v", []byte(tt.input))
				t.Logf("Result bytes: %v", []byte(result))
				t.Logf("Expected bytes: %v", []byte(tt.expected))
			}
		})
	}
}

// TestProcessEscapeSequences_ByteValues tests specific byte values to ensure correct conversion
func TestProcessEscapeSequences_ByteValues(t *testing.T) {
	input := "\\r\\n\\t\\x1b\\x00\\xff"
	result := processEscapeSequences(input)
	expected := []byte{13, 10, 9, 27, 0, 255} // CR, LF, TAB, ESC, NULL, 255

	resultBytes := []byte(result)
	if len(resultBytes) != len(expected) {
		t.Fatalf("Length mismatch: got %d bytes, expected %d", len(resultBytes), len(expected))
	}

	for i, expectedByte := range expected {
		if resultBytes[i] != expectedByte {
			t.Errorf("Byte %d: got %d, expected %d", i, resultBytes[i], expectedByte)
		}
	}
}
