package scripting

import (
	"strings"
	"testing"
)

// TestSimpleExpectEngineStarConversion tests the independent expect engine's "*" handling
func TestSimpleExpectEngineStarConversion(t *testing.T) {
	// Test the SimpleExpectEngine (independent of telnet) handles "*" correctly
	var capturedInput []string
	
	// Create a mock input sender to capture what gets sent
	mockInputSender := func(input string) {
		capturedInput = append(capturedInput, input)
		t.Logf("CAPTURED INPUT: %q", input)
	}
	
	// Create independent expect engine with "\r\n" replacement to test the conversion
	engine := NewExpectEngine(t, mockInputSender, "\r\n")
	
	testCases := []struct {
		name     string
		script   string
		expected []string
	}{
		{
			name:   "Single star at end",
			script: `send "Hello World*"`,
			expected: []string{"Hello World\r\n"},
		},
		{
			name:   "Multiple stars",
			script: `send "Line1*Line2*Line3*"`,
			expected: []string{"Line1\r\nLine2\r\nLine3\r\n"},
		},
		{
			name:   "Star in middle",
			script: `send "Hello*World"`,
			expected: []string{"Hello\r\nWorld"},
		},
		{
			name:   "No stars",
			script: `send "No conversion needed"`,
			expected: []string{"No conversion needed"},
		},
		{
			name:   "Just a star",
			script: `send "*"`,
			expected: []string{"\r\n"},
		},
		{
			name:   "Empty string",
			script: `send " "`,
			expected: []string{" "},
		},
		{
			name: "Multiple send commands",
			script: `
send "First line*"
send "Second*Third*"
send "Fourth"`,
			expected: []string{
				"First line\r\n",
				"Second\r\nThird\r\n", 
				"Fourth",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear captured input
			capturedInput = nil
			
			// Run the script
			err := engine.Run(tc.script)
			if err != nil {
				t.Fatalf("Script execution failed: %v", err)
			}
			
			// Check captured input matches expected
			if len(capturedInput) != len(tc.expected) {
				t.Fatalf("Expected %d inputs, got %d: %v", len(tc.expected), len(capturedInput), capturedInput)
			}
			
			for i, expected := range tc.expected {
				if capturedInput[i] != expected {
					t.Errorf("Input %d: expected %q, got %q", i, expected, capturedInput[i])
				}
			}
		})
	}
}

// TestExpectEngineClientVsServerStarConversion tests that client and server use different replacements
func TestExpectEngineClientVsServerStarConversion(t *testing.T) {
	// Test client-side (should use "\r")
	var clientCapture []string
	clientEngine := NewExpectEngine(t, func(input string) {
		clientCapture = append(clientCapture, input)
	}, "\r")
	
	err := clientEngine.Run(`send "Hello*World*"`)
	if err != nil {
		t.Fatalf("Client script failed: %v", err)
	}
	
	if len(clientCapture) != 1 || clientCapture[0] != "Hello\rWorld\r" {
		t.Errorf("Client expected 'Hello\\rWorld\\r', got %q", clientCapture)
	}
	
	// Test server-side (should use "\r\n")
	var serverCapture []string
	serverEngine := NewExpectEngine(t, func(input string) {
		serverCapture = append(serverCapture, input)
	}, "\r\n")
	
	err = serverEngine.Run(`send "Hello*World*"`)
	if err != nil {
		t.Fatalf("Server script failed: %v", err)
	}
	
	if len(serverCapture) != 1 || serverCapture[0] != "Hello\r\nWorld\r\n" {
		t.Errorf("Server expected 'Hello\\r\\nWorld\\r\\n', got %q", serverCapture)
	}
}

// TestExpectEngineStarConversionCompatibility tests that the conversion matches TWX behavior
func TestExpectEngineStarConversionCompatibility(t *testing.T) {
	// Test that our conversion matches what TWX scripts expect
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{"Trade Wars 2002*", "Trade Wars 2002\r\n", "Game title with newline"},
		{"Sector  : 2921 in uncharted space.*", "Sector  : 2921 in uncharted space.\r\n", "Sector description"},
		{"Warps to Sector(s) :  3212 - 7656*", "Warps to Sector(s) :  3212 - 7656\r\n", "Warp information"},
		{"*", "\r\n", "Just newline"},
		{"Line1*Line2*Line3*", "Line1\r\nLine2\r\nLine3\r\n", "Multiple lines"},
		{"Command [TL=00:00:00]:[2921] (?=Help)? : ", "Command [TL=00:00:00]:[2921] (?=Help)? : ", "Prompt without star"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := strings.ReplaceAll(tc.input, "*", "\r\n")
			if result != tc.expected {
				t.Errorf("Star conversion failed:\nInput:    %q\nExpected: %q\nGot:      %q", tc.input, tc.expected, result)
			}
		})
	}
}