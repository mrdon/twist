package scripting

import (
	"strings"
	"testing"
	"twist/internal/api"
)

// TestANSIProcessing tests that server expect scripts correctly
// convert ANSI escape sequences from string literals to actual binary bytes
func TestANSIProcessing(t *testing.T) {
	// Simple test: server sends red text
	serverScript := `send "\\x1b[31mHello World"`
	clientScript := `expect "Hello World"`

	// Execute test and get result
	result := Execute(t, serverScript, clientScript, &api.ConnectOptions{})
	
	// Check we got output in client
	if result.ClientOutput == "" {
		t.Fatal("No client output received")
	}
	
	// Should contain actual ESC character (byte 27), not literal "\\x1b"
	if !strings.Contains(result.ClientOutput, "\x1b[31m") {
		t.Errorf("Expected actual ANSI sequence \\x1b[31m, got: %q", result.ClientOutput)
	}
	
	// Should NOT contain literal backslash-x-1-b
	if strings.Contains(result.ClientOutput, "\\x1b") {
		t.Errorf("Found literal \\x1b instead of actual escape character in: %q", result.ClientOutput)
	}
}