package scripting

import (
	"os"
	"strings"
	"testing"
	"twist/internal/api"
)

func TestGenerateExpectPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "command prompt with ANSI",
			input:    `\x1b[35mCommand [\x1b[1;33mTL\x1b[0;33m=\x1b[1m00:00:00\x1b[0;35m]? : `,
			expected: "0\x1b[0;35m]? : ",
		},
		{
			name:     "probe destination prompt",
			input:    `\x1b[32mSubSpace Ether Probe loaded\x1b[0m\n\x1b[35mPlease enter a destination for this probe\x1b[1;33m: \x1b[36m `,
			expected: "e\x1b[1;33m: \x1b[36m ",
		},
		{
			name:     "probe self destruct message",
			input:    `\x1b[33mProbe entering sector\x1b[36m493\r\x1b[0m\n\r\n\x1b[1;36mProbe Self Destructs\r\x1b[0m`,
			expected: "Destructs\r\x1b[0m",
		},
		{
			name:     "question mark colon pattern",
			input:    `Some text here? : `,
			expected: "t here? : ",
		},
		{
			name:     "colon space pattern",
			input:    `Enter value: `,
			expected: "er value: ",
		},
		{
			name:     "short text - use whole string",
			input:    `Short`,
			expected: "Short",
		},
		{
			name:     "exactly 5 chars - use whole string",
			input:    `Hello`,
			expected: "Hello",
		},
		{
			name:     "exactly 10 chars - use whole string",
			input:    `HelloWorld`,
			expected: "HelloWorld",
		},
		{
			name:     "long text - use last 10 chars",
			input:    `This is a very long piece of text`,
			expected: "ce of text",
		},
		{
			name:     "long text - use more than 10 if repeated",
			input:    `iece of text and This is a very long piece of text`,
			expected: "ce of text",
		},
		{
			name:     "empty text",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExpectPattern(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func i(t *testing.T) {
	// Test the complete pipeline: file -> parse -> convert -> expect scripts
	scriptContent := `# Probe Test Script
# End-to-end testf

< \x1b[35mCommand [\x1b[1;33mTL\x1b[0;33m=\x1b[1m00:00:00\x1b[0;35m]? : 
> e
< E\r\x1b[0m\n\x1b[35mPlease enter a destination for this probe\x1b[1;33m: \x1b[36m 
> 493
< \x1b[1;36mProbe Self Destructs\r\x1b[0m`

	expectedServerScript := `send "\x1b[35mCommand [\x1b[1;33mTL\x1b[0;33m=\x1b[1m00:00:00\x1b[0;35m]? : "
expect "e"
send "E\r\x1b[0m\n\x1b[35mPlease enter a destination for this probe\x1b[1;33m: \x1b[36m "
expect "493"
send "\x1b[1;36mProbe Self Destructs\r\x1b[0m"`

	expectedClientScript := `expect "0;35m]? : "
send "e"
expect "3m: \x1b[36m "
send "493"`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test_end_to_end_*.script")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test content
	if _, err := tmpFile.WriteString(scriptContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tmpFile.Close()

	// Load script file
	scriptLines, err := LoadScriptFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load script file: %v", err)
	}

	// Convert to expect scripts
	serverScript, clientScript := ConvertToExpectScripts(scriptLines)

	// Verify server script
	if strings.TrimSpace(serverScript) != strings.TrimSpace(expectedServerScript) {
		t.Errorf("Server script mismatch:\nExpected:\n%s\nGot:\n%s", expectedServerScript, serverScript)
	}

	// Verify client script
	if clientScript != expectedClientScript {
		t.Errorf("Client script mismatch:\nExpected:\n%s\nGot:\n%s", expectedClientScript, clientScript)
	}
}

// TestServerExpectScriptProcessesANSI tests that server expect scripts correctly
// convert ANSI escape sequences from string literals to actual binary bytes
func TestServerExpectScriptProcessesANSI(t *testing.T) {
	// Capture what the TUI receives
	var receivedData []byte

	// Server script with ANSI escape sequences as string literals
	serverScript := `send "\\x1b[35mCommand \\x1b[1;33mTL\\x1b[0;33m=\\x1b[1m00:00:00\\x1b[0;35m]? : "
expect "e"
send "E\\r\\x1b[0m\\n\\x1b[32mSubSpace Ether Probe loaded"`

	clientScript := `expect "0;35m]? : "
send "e"
expect "Space Ether"`

	// Execute the test using the proxy framework
	result := Execute(t, serverScript, clientScript, &api.ConnectOptions{})

	// Check the client output from the result
	if result.ClientOutput == "" {
		t.Fatal("No client output received")
	}

	// Use the client output for analysis
	receivedStr := result.ClientOutput

	// Check that actual ANSI escape sequences are present (byte 27 = ESC)
	if !strings.Contains(receivedStr, "\x1b") {
		t.Error("Expected actual ESC character (0x1B) in received data, but found none")
		t.Logf("Received bytes: %v", receivedData)
		t.Logf("Received string: %q", receivedStr)
	}

	// Check for actual carriage return and newline
	if !strings.Contains(receivedStr, "\r") {
		t.Error("Expected actual CR character (0x0D) in received data")
	}

	if !strings.Contains(receivedStr, "\n") {
		t.Error("Expected actual LF character (0x0A) in received data")
	}

	// Verify specific ANSI sequences are properly converted
	expectedSequences := []string{
		"\x1b[35m",   // Magenta color
		"\x1b[1;33m", // Bold yellow
		"\x1b[0;33m", // Yellow
		"\x1b[1m",    // Bold
		"\x1b[0m",    // Reset
		"\x1b[32m",   // Green
	}

	for _, expectedSeq := range expectedSequences {
		if !strings.Contains(receivedStr, expectedSeq) {
			t.Errorf("Expected ANSI sequence %q not found in received data", expectedSeq)
		}
	}

	// Verify that literal strings like "\\x1b" are NOT present
	if strings.Contains(receivedStr, "\\x1b") {
		t.Error("Found literal '\\x1b' in received data - escape sequences were not processed")
		t.Logf("Received string: %q", receivedStr)
	}

	if strings.Contains(receivedStr, "\\r") {
		t.Error("Found literal '\\r' in received data - escape sequences were not processed")
	}

	if strings.Contains(receivedStr, "\\n") {
		t.Error("Found literal '\\n' in received data - escape sequences were not processed")
	}

	t.Logf("Successfully received %d bytes with properly processed ANSI sequences", len(receivedData))
}

// TestEscapeSequenceProcessing tests that client expects with escaped ANSI sequences
// are properly decoded to match actual ANSI sequences from the server
func TestEscapeSequenceProcessing(t *testing.T) {
	// Server script sends literal ANSI sequences
	serverScript := `send "Please enter a destination for this probe \x1b[1;33m: \x1b[36m "`

	// Client script expects literal ANSI sequences (same format)
	clientScript := `expect "this probe \x1b[1;33m: \x1b[36m "`

	// Execute the test
	result := Execute(t, serverScript, clientScript, &api.ConnectOptions{})

	// Check the client output to see what was actually received
	if result.ClientOutput == "" {
		t.Fatal("No client output received")
	}

	// The test should pass - the escaped sequences in clientScript should be decoded
	// to match the actual ANSI sequences sent by the server
	t.Logf("Server sent: %q", result.ClientOutput)
	t.Logf("Client expected (raw): \"this probe \\\\x1b[1;33m: \\\\x1b[36m \"")
	t.Logf("Client expected (decoded): \"this probe \\x1b[1;33m: \\x1b[36m \"")

	// Check if the pattern should match
	expectedPattern := "this probe \x1b[1;33m: \x1b[36m "
	if !strings.Contains(result.ClientOutput, expectedPattern) {
		t.Errorf("Expected client output to contain pattern %q, but it doesn't", expectedPattern)
		t.Logf("Full client output: %q", result.ClientOutput)
	}
}
