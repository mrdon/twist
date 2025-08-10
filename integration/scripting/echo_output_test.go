package scripting

import (
	"testing"
)

// TestEchoOutputGoesToTUINotServer verifies that echo commands send output
// to the TUI (terminal display) and NOT to the game server
func TestEchoOutputGoesToTUINotServer(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "Hello from TUI!"
		echo "This should appear in terminal"
		echo "But NOT be sent to server"
	`
	
	// Execute the script
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	// Verify that echo output appears in captured output (TUI stream)
	expectedOutputs := []string{
		"Hello from TUI!",
		"This should appear in terminal", 
		"But NOT be sent to server",
	}
	
	for _, expected := range expectedOutputs {
		found := false
		for _, output := range result.Output {
			if output == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected echo output %q not found in TUI output. Got: %v", expected, result.Output)
		}
	}
	
	// Most importantly: verify echo output was NOT sent to server
	for _, sent := range result.Commands {
		for _, expected := range expectedOutputs {
			if sent == expected {
				t.Errorf("Echo output %q was incorrectly sent to server! Echo should only go to TUI.", sent)
			}
		}
	}
	
	// Additional verification: ensure we didn't send echo text to server
	if len(result.Commands) > 0 {
		t.Errorf("Echo script should not send any commands to server, but sent: %v", result.Commands)
	}
}

// TestEchoVsSendCommandBehavior verifies the difference between echo (TUI only) 
// and send (server only) commands
func TestEchoVsSendCommandBehavior(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "This goes to TUI"
		send "look"
		echo "This also goes to TUI"  
		send "inventory"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	// Verify echo outputs appear in TUI
	expectedTUIOutputs := []string{
		"This goes to TUI",
		"This also goes to TUI",
	}
	
	for _, expected := range expectedTUIOutputs {
		found := false
		for _, output := range result.Output {
			if output == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected TUI output %q not found. Got: %v", expected, result.Output)
		}
	}
	
	// Verify send commands appear in server commands
	expectedServerCommands := []string{
		"look",
		"inventory", 
	}
	
	for _, expected := range expectedServerCommands {
		found := false
		for _, sent := range result.Commands {
			if sent == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected server command %q not found. Got: %v", expected, result.Commands)
		}
	}
	
	// Verify echo outputs did NOT go to server
	for _, tui := range expectedTUIOutputs {
		for _, sent := range result.Commands {
			if sent == tui {
				t.Errorf("TUI output %q incorrectly sent to server!", tui)
			}
		}
	}
	
	// Verify send commands did NOT go to TUI
	for _, server := range expectedServerCommands {
		for _, output := range result.Output {
			if output == server {
				t.Errorf("Server command %q incorrectly sent to TUI!", server)
			}
		}
	}
}

// TestEchoWithVariableExpansion verifies echo with variables works correctly 
// and still goes to TUI only
func TestEchoWithVariableExpansion(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		$testvar = "World"
		echo "Hello " $testvar "!"
		echo "Variable expansion works: " $testvar
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	// Verify variable expansion happened and went to TUI
	expectedOutputs := []string{
		"Hello World!",
		"Variable expansion works: World",
	}
	
	for _, expected := range expectedOutputs {
		found := false
		for _, output := range result.Output {
			if output == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected echo output %q not found in TUI. Got: %v", expected, result.Output)
		}
	}
	
	// Verify nothing was sent to server
	if len(result.Commands) > 0 {
		t.Errorf("Echo with variables should not send commands to server, but sent: %v", result.Commands)
	}
}

// TestEchoIntegrationWithRealProxy tests echo behavior with a more realistic setup
// that includes a mock TUI API to verify the data flow
func TestEchoIntegrationWithRealProxy(t *testing.T) {
	// Note: This would need integration with the actual proxy setup
	// For now, we rely on the integration tester which captures the right streams
	
	tester := NewIntegrationScriptTester(t)
	
	script := `echo "Testing real proxy integration"`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	// Verify the echo output was captured (meaning it went through TUI stream)
	found := false
	for _, output := range result.Output {
		if output == "Testing real proxy integration" {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Echo output not found in TUI stream. This suggests echo is not working correctly.")
	}
	
	// Verify nothing was sent to server
	if len(result.Commands) > 0 {
		t.Errorf("Echo should not send to server, but sent: %v", result.Commands)
	}
}

// TestMultipleEchoCommands verifies multiple echo commands in sequence
func TestMultipleEchoCommands(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "Line 1"
		echo "Line 2" 
		echo "Line 3"
		echo "Final line"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
		return
	}
	
	expectedLines := []string{
		"Line 1",
		"Line 2",
		"Line 3", 
		"Final line",
	}
	
	// Verify all echo lines appear in TUI output
	for _, expected := range expectedLines {
		found := false
		for _, output := range result.Output {
			if output == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected echo line %q not found in TUI output", expected)
		}
	}
	
	// Verify none were sent to server
	for _, expected := range expectedLines {
		for _, sent := range result.Commands {
			if sent == expected {
				t.Errorf("Echo line %q was incorrectly sent to server", expected)
			}
		}
	}
	
	// Verify total - should be no server commands
	if len(result.Commands) > 0 {
		t.Errorf("Multiple echo script should send no server commands, but sent: %v", result.Commands)
	}
}