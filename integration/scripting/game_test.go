//go:build integration

package scripting

import (
	"strings"
	"testing"
	"time"
)

// TestSendCommand_RealIntegration tests SEND command with real VM and database
func TestSendCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		send "look"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// SEND commands don't produce echo output, they send to game server
	// We can verify the command executed without error
}

// TestSendCommand_MultipleParameters tests SEND with multiple parameters
func TestSendCommand_MultipleParameters_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		$action := "look"
		$target := "north"
		send $action " " $target
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
}

// TestSendCommand_WithVariables tests SEND using variables with persistence
func TestSendCommand_WithVariables_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		$command := "examine"
		$object := "sword"
		savevar $command
		savevar $object
		send $command " " $object
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
}

// TestWaitForCommand_RealIntegration tests WAITFOR command
func TestWaitForCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// This test simulates a WAITFOR that should complete when matching text arrives
	script := `
		echo "Starting wait"
		waitfor "test pattern"
		echo "Wait completed"
	`
	
	// Start script execution in a goroutine since WAITFOR will block
	done := make(chan *IntegrationTestResult, 1)
	go func() {
		result := tester.ExecuteScript(script)
		done <- result
	}()
	
	// Give script time to start and reach WAITFOR
	time.Sleep(100 * time.Millisecond)
	
	// Simulate incoming network text that matches the pattern
	err := tester.SimulateNetworkInput("This contains test pattern in the middle")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	// Wait for script completion
	select {
	case result := <-done:
		if result.Error != nil {
			t.Errorf("Script execution failed: %v", result.Error)
		}
		
		// Should have both echo messages
		if len(result.Output) < 2 {
			t.Errorf("Expected at least 2 output lines, got %d", len(result.Output))
		}
		
		if len(result.Output) > 0 && result.Output[0] != "Starting wait" {
			t.Errorf("First echo: got %q, want %q", result.Output[0], "Starting wait")
		}
		
		if len(result.Output) > 1 && result.Output[1] != "Wait completed" {
			t.Errorf("Second echo: got %q, want %q", result.Output[1], "Wait completed")
		}
		
	case <-time.After(5 * time.Second):
		t.Error("WAITFOR test timed out - WAITFOR may not be working correctly")
	}
}

// TestWaitForCommand_NoMatch tests WAITFOR with pattern that doesn't match
func TestWaitForCommand_NoMatch_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "Starting wait"
		waitfor "never matches"
		echo "This should not be reached"
	`
	
	// Start script execution in a goroutine
	done := make(chan *IntegrationTestResult, 1)
	go func() {
		result := tester.ExecuteScript(script)
		done <- result
	}()
	
	// Give script time to start and reach WAITFOR
	time.Sleep(100 * time.Millisecond)
	
	// Simulate non-matching network text
	err := tester.SimulateNetworkInput("This does not match the pattern")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	// Wait briefly - script should still be waiting
	select {
	case result := <-done:
		// If we get here, the script completed unexpectedly
		t.Errorf("Script completed unexpectedly with error: %v", result.Error)
		
	case <-time.After(500 * time.Millisecond):
		// Expected - script should still be waiting
		// This is the correct behavior
	}
}

// TestPauseCommand_RealIntegration tests PAUSE command
func TestPauseCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "Before pause"
		pause
		echo "After pause"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// PAUSE should allow script to continue and complete
	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Before pause" {
		t.Errorf("First echo: got %q, want %q", result.Output[0], "Before pause")
	}
	
	if len(result.Output) > 1 && result.Output[1] != "After pause" {
		t.Errorf("Second echo: got %q, want %q", result.Output[1], "After pause")
	}
}

// TestHaltCommand_RealIntegration tests HALT command
func TestHaltCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "Before halt"
		halt
		echo "This should not execute"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// HALT should stop execution, so only first echo should appear
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line after HALT, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Before halt" {
		t.Errorf("Echo before halt: got %q, want %q", result.Output[0], "Before halt")
	}
}

// TestGameCommands_CrossInstancePersistence tests game commands with persistent variables
func TestGameCommands_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First script execution - save command template
	tester1 := NewIntegrationScriptTester(t)
	
	script1 := `
		$base_command := "examine"
		$default_target := "room"
		savevar $base_command
		savevar $default_target
		echo "Saved command template"
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}
	
	// Second script execution - load and use template (simulates VM restart)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script2 := `
		loadvar $base_command
		loadvar $default_target
		send $base_command " " $default_target
		echo "Sent: " $base_command " " $default_target
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}
	
	if len(result2.Output) != 1 {
		t.Errorf("Expected 1 output line from second script, got %d", len(result2.Output))
	}
	
	expected := "Sent: examine room"
	if len(result2.Output) > 0 && result2.Output[0] != expected {
		t.Errorf("Cross-instance command: got %q, want %q", result2.Output[0], expected)
	}
}

// TestGameCommands_ConditionalSending tests sending with comparison results
func TestGameCommands_ConditionalSending_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Test comparison-based logic without complex conditionals
		$health := 50
		$health_threshold := 75
		
		isequal $health $health_threshold $health_result
		echo "Health comparison result (50 == 75): " $health_result
		
		# Test other comparison
		isgreater $health_threshold $health $greater_result
		echo "Threshold comparison result (75 > 50): " $greater_result
		
		# Simple sending based on known values
		send "check health status"
		echo "Health check command sent"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Should have 3 echo outputs
	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Health comparison result (50 == 75): 0" {
		t.Errorf("Equality comparison: got %q, want %q", result.Output[0], "Health comparison result (50 == 75): 0")
	}
	
	if len(result.Output) > 1 && result.Output[1] != "Threshold comparison result (75 > 50): 1" {
		t.Errorf("Greater comparison: got %q, want %q", result.Output[1], "Threshold comparison result (75 > 50): 1")
	}
	
	if len(result.Output) > 2 && result.Output[2] != "Health check command sent" {
		t.Errorf("Send command echo: got %q, want %q", result.Output[2], "Health check command sent")
	}
}

// TestGameCommands_StringConcatenation tests complex string building for commands
func TestGameCommands_StringConcatenation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		$verb := "tell"
		$target := "merchant"
		$message := "I want to buy sword"
		$punctuation := "."
		
		send $verb " " $target " " $message $punctuation
		echo "Sent message to " $target
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	expected := "Sent message to merchant"
	if len(result.Output) > 0 && result.Output[0] != expected {
		t.Errorf("String concatenation: got %q, want %q", result.Output[0], expected)
	}
}

// TestGameCommands_EmptyAndSpecialCommands tests edge cases
func TestGameCommands_EmptyAndSpecialCommands_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		$empty := ""
		$newline := "\n"
		$space := " "
		
		send $empty
		echo "Sent empty command"
		
		send "look" $newline "inventory"
		echo "Sent multiline command"
		
		send "say" $space "hello world"
		echo "Sent spaced command"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}
	
	expectedOutputs := []string{
		"Sent empty command",
		"Sent multiline command", 
		"Sent spaced command",
	}
	
	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestGameCommands_NumberToStringConversion tests sending numeric values
func TestGameCommands_NumberToStringConversion_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		$sector_num := 1234
		$credits := 5000.50
		
		send "warp " $sector_num
		echo "Warped to sector " $sector_num
		
		send "offer " $credits " credits"
		echo "Offered " $credits " credits"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && !strings.Contains(result.Output[0], "1234") {
		t.Errorf("Sector echo should contain '1234': got %q", result.Output[0])
	}
	
	if len(result.Output) > 1 && !strings.Contains(result.Output[1], "5000.5") {
		t.Errorf("Credits echo should contain '5000.5': got %q", result.Output[1])
	}
}

// TestWaitForCommand_WithTriggerInteraction tests WAITFOR working with triggers
func TestWaitForCommand_WithTriggerInteraction_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// This complex test sets up a trigger and then uses WAITFOR
	script := `
		echo "Setting up trigger"
		settexttrigger 1 "echo 'Trigger fired'" "test trigger"
		
		echo "Starting wait"
		waitfor "continue"
		echo "Wait completed"
	`
	
	done := make(chan *IntegrationTestResult, 1)
	go func() {
		result := tester.ExecuteScript(script)
		done <- result
	}()
	
	// Give script time to set up trigger and reach WAITFOR
	time.Sleep(200 * time.Millisecond)
	
	// Simulate text that triggers both the trigger and satisfies WAITFOR
	err := tester.SimulateNetworkInput("test trigger message")
	if err != nil {
		t.Errorf("Failed to simulate first network input: %v", err)
	}
	
	// Give trigger time to fire
	time.Sleep(100 * time.Millisecond)
	
	// Now send the text that satisfies WAITFOR
	err = tester.SimulateNetworkInput("continue with the script")
	if err != nil {
		t.Errorf("Failed to simulate second network input: %v", err)
	}
	
	// Wait for script completion
	select {
	case result := <-done:
		if result.Error != nil {
			t.Errorf("Script execution failed: %v", result.Error)
		}
		
		// Should have trigger setup, start wait, trigger fired, and wait completed
		if len(result.Output) < 4 {
			t.Errorf("Expected at least 4 output lines, got %d", len(result.Output))
		}
		
	case <-time.After(5 * time.Second):
		t.Error("WAITFOR with trigger test timed out")
	}
}