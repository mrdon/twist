package scripting

import (
	"testing"
)

// TestSendStarToCarriageReturn verifies that send commands convert * to carriage returns for server communication
func TestSendStarToCarriageReturn_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		send "mrdon" "*"
		send "password123" "*" 
		send "*"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	// Verify the correct commands were sent to the server (with carriage returns)
	expectedCommands := []string{
		"mrdon\r",
		"password123\r",
		"\r",
	}
	
	if len(result.Commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands sent to server, got %d. Commands: %v", len(expectedCommands), len(result.Commands), result.Commands)
		return
	}
	
	for i, expected := range expectedCommands {
		if i >= len(result.Commands) {
			t.Errorf("Missing command %d: expected %q", i, expected)
			continue
		}
		if result.Commands[i] != expected {
			t.Errorf("Command %d mismatch.\nExpected: %q\nGot: %q", i, expected, result.Commands[i])
			// Show character-by-character comparison for debugging
			expectedRunes := []rune(expected)
			actualRunes := []rune(result.Commands[i])
			t.Logf("Expected length: %d, Got length: %d", len(expectedRunes), len(actualRunes))
			for j := 0; j < len(expectedRunes) && j < len(actualRunes); j++ {
				if expectedRunes[j] != actualRunes[j] {
					t.Logf("  Diff at [%d]: expected %q (\\x%02x), got %q (\\x%02x)", j, string(expectedRunes[j]), int(expectedRunes[j]), string(actualRunes[j]), int(actualRunes[j]))
				}
			}
		}
	}
	
	// Verify nothing was echoed to TUI
	if len(result.Output) > 0 {
		t.Errorf("Send commands should not echo to TUI, but got output: %v", result.Output)
	}
}

// TestSendStarConversionWithVariables verifies * to carriage return conversion works with variables
func TestSendStarConversionWithVariables_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $username "testuser"
		setVar $password "secret123" 
		send $username "*"
		send $password "*"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	expectedCommands := []string{
		"testuser\r",
		"secret123\r",
	}
	
	if len(result.Commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d. Commands: %v", len(expectedCommands), len(result.Commands), result.Commands)
		return
	}
	
	for i, expected := range expectedCommands {
		if result.Commands[i] != expected {
			t.Errorf("Command %d mismatch.\nExpected: %q\nGot: %q", i, expected, result.Commands[i])
		}
	}
}

// TestSendVsEchoStarBehavior verifies different * handling between send (server) and echo (TUI)
func TestSendVsEchoStarBehavior_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		send "login*command*sequence"
		echo "display*with*stars"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	// Send should convert * to carriage returns for server
	expectedServerCommand := "login\rcommand\rsequence"
	if len(result.Commands) != 1 {
		t.Errorf("Expected 1 server command, got %d: %v", len(result.Commands), result.Commands)
	} else if result.Commands[0] != expectedServerCommand {
		t.Errorf("Server command mismatch.\nExpected: %q\nGot: %q", expectedServerCommand, result.Commands[0])
	}
	
	// Echo should convert * to CRLF for display
	expectedTUIOutput := "display\r\nwith\r\nstars"
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 TUI output, got %d: %v", len(result.Output), result.Output)
	} else if result.Output[0] != expectedTUIOutput {
		t.Errorf("TUI output mismatch.\nExpected: %q\nGot: %q", expectedTUIOutput, result.Output[0])
	}
}

// TestSendNoStarCharacters verifies normal send behavior without * characters
func TestSendNoStarCharacters_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		send "look"
		send "inventory"
		send "quit"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	expectedCommands := []string{
		"look",
		"inventory", 
		"quit",
	}
	
	if len(result.Commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d. Commands: %v", len(expectedCommands), len(result.Commands), result.Commands)
		return
	}
	
	for i, expected := range expectedCommands {
		if result.Commands[i] != expected {
			t.Errorf("Command %d mismatch.\nExpected: %q\nGot: %q", i, expected, result.Commands[i])
		}
	}
	
	// Verify nothing was echoed to TUI
	if len(result.Output) > 0 {
		t.Errorf("Send commands should not echo to TUI, but got output: %v", result.Output)
	}
}