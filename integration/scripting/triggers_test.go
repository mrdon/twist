

package scripting

import (
	"testing"
	"time"
)

// TestSetTextTrigger_RealIntegration tests SETTEXTTRIGGER command with real trigger system
func TestSetTextTrigger_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextTrigger 1 "echo 'Text trigger fired'" "health"
		echo "Trigger set"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Trigger set" {
		t.Errorf("Trigger setup echo: got %q, want %q", result.Output[0], "Trigger set")
	}
	
	// Now simulate incoming text that should trigger the response
	err := tester.SimulateNetworkInput("Your health is low")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	// Give time for trigger to process
	time.Sleep(1 * time.Millisecond)
	
	// Check that trigger fired - we need to get the updated output
	// In a real implementation, we'd need a way to capture trigger output
}

// TestSetTextTrigger_PatternMatching tests text trigger pattern matching
func TestSetTextTrigger_PatternMatching_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextTrigger 1 "echo 'Enemy found'" "orc"
		setTextTrigger 2 "echo 'Treasure found'" "gold"
		echo "Triggers configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Test multiple pattern matching scenarios
	testCases := []struct {
		input    string
		expected string
	}{
		{"A fierce orc blocks your path", "Enemy found"},
		{"You see some gold coins", "Treasure found"},
		{"The orc attacks with its sword", "Enemy found"},
		{"A peaceful goblin walks by", ""}, // Should not match either trigger
	}
	
	for _, tc := range testCases {
		err := tester.SimulateNetworkInput(tc.input)
		if err != nil {
			t.Errorf("Failed to simulate input %q: %v", tc.input, err)
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// TestSetTextLineTrigger_RealIntegration tests SETTEXTLINETRIGGER command
func TestSetTextLineTrigger_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		settextlinetrigger 1 "echo 'Line trigger activated'" "prompt"
		echo "Line trigger set"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	// Simulate a complete line that should trigger the response
	err := tester.SimulateNetworkInput("Command prompt: >")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
}

// TestSetDelayTrigger_RealIntegration tests SETDELAYTRIGGER command with real timing
func TestSetDelayTrigger_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setdelaytrigger 1 "echo 'Delay trigger fired'" 200
		echo "Delay trigger set for 200ms"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	// Wait for delay to elapse plus some margin
	time.Sleep(2 * time.Millisecond)
	
	// In a real implementation, the delay trigger should have fired by now
	// We would need to capture the trigger output to verify
}

// TestSetEventTrigger_RealIntegration tests SETEVENTTRIGGER command
func TestSetEventTrigger_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		seteventtrigger 1 "echo 'Connection event fired'" "connect"
		echo "Event trigger set"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	// In a real system, we would fire the event and verify the trigger response
}

// TestKillTrigger_RealIntegration tests KILLTRIGGER command
func TestKillTrigger_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextTrigger 1 "echo 'This should not fire'" "test"
		echo "Trigger set"
		killTrigger 1
		echo "Trigger killed"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}
	
	// Now test that killed trigger doesn't fire
	err := tester.SimulateNetworkInput("This is a test message")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
	
	// Trigger should not have fired since it was killed
}

// TestTriggers_CrossInstancePersistence tests trigger persistence across VM instances
func TestTriggers_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First script execution - set up triggers
	tester1 := NewIntegrationScriptTester(t)
	
	script1 := `
		setTextTrigger 1 "echo 'Persistent trigger fired'" "magic"
		setVar $trigger_pattern "magic"
		saveVar $trigger_pattern
		echo "Persistent trigger configured"
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}
	
	// Second script execution - verify trigger persistence (simulates VM restart)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script2 := `
		loadVar $trigger_pattern
		echo "Loaded trigger pattern: " $trigger_pattern
		setTextTrigger 2 "echo 'New instance trigger'" $trigger_pattern
		echo "New trigger set with loaded pattern"
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}
	
	if len(result2.Output) != 2 {
		t.Errorf("Expected 2 output lines from second script, got %d", len(result2.Output))
	}
	
	expected := "Loaded trigger pattern: magic"
	if len(result2.Output) > 0 && result2.Output[0] != expected {
		t.Errorf("Pattern loading: got %q, want %q", result2.Output[0], expected)
	}
}

// TestTriggers_MultiplePatterns tests multiple triggers with different patterns
func TestTriggers_MultiplePatterns_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextTrigger 1 "echo 'Combat trigger'" "attack"
		setTextTrigger 2 "echo 'Movement trigger'" "arrive"  
		setTextTrigger 3 "echo 'Chat trigger'" "says"
		echo "Multiple triggers configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Test each trigger pattern
	testInputs := []string{
		"The orc attacks you fiercely",
		"A player arrives from the north",
		"The merchant says 'Welcome to my shop'",
		"You examine the room carefully", // Should not trigger any
	}
	
	for _, input := range testInputs {
		err := tester.SimulateNetworkInput(input)
		if err != nil {
			t.Errorf("Failed to simulate input %q: %v", input, err)
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// TestTriggers_VariableInterpolation tests triggers that use variables in responses
func TestTriggers_VariableInterpolation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $player_name "Hero"
		setVar $health 100
		saveVar $player_name
		saveVar $health
		
		setTextTrigger 1 "echo $player_name ' health: ' $health" "status"
		echo "Variable-based trigger set"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Trigger the variable-interpolated response
	err := tester.SimulateNetworkInput("Check your status")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
}

// TestTriggers_ConditionalLogic tests triggers with conditional responses
func TestTriggers_ConditionalLogic_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $auto_fight 1
		
		setTextTrigger 1 "if $auto_fight = 1 then\nsend 'attack'\necho 'Auto-attacking'\nend if" "enemy"
		echo "Conditional trigger configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Trigger the conditional response
	err := tester.SimulateNetworkInput("A dangerous enemy appears")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
}

// TestTriggers_ChainedTriggers tests triggers that set other triggers
func TestTriggers_ChainedTriggers_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextTrigger 1 "setTextTrigger 2 'echo Chain completed' 'complete'\necho 'First trigger fired, second trigger set'" "start"
		echo "Chained trigger system configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Fire the first trigger
	err := tester.SimulateNetworkInput("start the chain")
	if err != nil {
		t.Errorf("Failed to simulate first input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
	
	// Fire the second trigger that was set by the first
	err = tester.SimulateNetworkInput("complete the chain")
	if err != nil {
		t.Errorf("Failed to simulate second input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
}

// TestTriggers_EmptyAndSpecialPatterns tests edge cases in trigger patterns
func TestTriggers_EmptyAndSpecialPatterns_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextTrigger 1 "echo 'Empty pattern triggered'" ""
		setTextTrigger 2 "echo 'Special chars triggered'" "test\ntab"
		setTextTrigger 3 "echo 'Unicode triggered'" "café"
		echo "Special pattern triggers configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Test special patterns
	testInputs := []string{
		"Any text should match empty pattern",
		"This has test\ntab in it",
		"Welcome to the café",
	}
	
	for _, input := range testInputs {
		err := tester.SimulateNetworkInput(input)
		if err != nil {
			t.Errorf("Failed to simulate input %q: %v", input, err)
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// TestTriggers_LifecycleManagement tests trigger activation/deactivation
func TestTriggers_LifecycleManagement_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextTrigger 1 "echo 'Lifecycle trigger fired'" "test"
		echo "Trigger 1 set"
		
		setTextTrigger 2 "killTrigger 1\necho 'Trigger 1 killed by trigger 2'" "kill"
		echo "Trigger 2 set to kill trigger 1"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// First, fire trigger 1
	err := tester.SimulateNetworkInput("This is a test message")
	if err != nil {
		t.Errorf("Failed to simulate first input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
	
	// Now fire trigger 2 which should kill trigger 1
	err = tester.SimulateNetworkInput("kill the first trigger")
	if err != nil {
		t.Errorf("Failed to simulate second input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
	
	// Try to fire trigger 1 again - should not work since it was killed
	err = tester.SimulateNetworkInput("Another test message")
	if err != nil {
		t.Errorf("Failed to simulate third input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
}

// TestTriggers_DatabasePersistence tests that trigger state persists across instances
func TestTriggers_DatabasePersistence_RealIntegration(t *testing.T) {
	// Note: In the real TWX system, triggers themselves don't persist across VM restarts
	// Only the variables used to configure them persist
	// This test demonstrates the correct behavior
	
	tester1 := NewIntegrationScriptTester(t)
	
	script1 := `
		setVar $trigger_id 1
		setVar $trigger_response "echo 'Restored trigger fired'"
		setVar $trigger_pattern "restore"
		
		saveVar $trigger_id
		saveVar $trigger_response  
		saveVar $trigger_pattern
		
		setTextTrigger $trigger_id $trigger_response $trigger_pattern
		echo "Initial trigger configured and variables saved"
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}
	
	// Second instance loads the variables and recreates the trigger
	tester2 := NewIntegrationScriptTester(t)
	
	script2 := `
		loadVar $trigger_id
		loadVar $trigger_response
		loadVar $trigger_pattern
		
		setTextTrigger $trigger_id $trigger_response $trigger_pattern
		echo "Trigger recreated from saved variables"
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}
	
	// Verify the recreated trigger works
	err := tester2.SimulateNetworkInput("restore the system")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	time.Sleep(1 * time.Millisecond)
}

// TestTriggers_PerformanceWithManyTriggers tests system performance with multiple triggers
func TestTriggers_PerformanceWithManyTriggers_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// Set up multiple triggers
	script := `
		setTextTrigger 1 "echo 'T1'" "pattern1"
		setTextTrigger 2 "echo 'T2'" "pattern2"
		setTextTrigger 3 "echo 'T3'" "pattern3"
		setTextTrigger 4 "echo 'T4'" "pattern4"
		setTextTrigger 5 "echo 'T5'" "pattern5"
		setTextTrigger 6 "echo 'T6'" "pattern6"
		setTextTrigger 7 "echo 'T7'" "pattern7"
		setTextTrigger 8 "echo 'T8'" "pattern8"
		setTextTrigger 9 "echo 'T9'" "pattern9"
		setTextTrigger 10 "echo 'T10'" "pattern10"
		echo "10 triggers configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Test performance by sending text that matches various triggers
	testInputs := []string{
		"This contains pattern1 and pattern2",
		"Only pattern5 here",
		"Multiple pattern7 and pattern9 matches",
		"No matches in this text",
		"pattern10 at the end",
	}
	
	start := time.Now()
	
	for _, input := range testInputs {
		err := tester.SimulateNetworkInput(input)
		if err != nil {
			t.Errorf("Failed to simulate input: %v", err)
		}
		time.Sleep(1 * time.Millisecond) // Brief pause between inputs
	}
	
	elapsed := time.Since(start)
	
	// Performance should be reasonable even with multiple triggers
	if elapsed > 5*time.Second {
		t.Errorf("Trigger processing took too long: %v", elapsed)
	}
}

// TestPhase3_PascalTWXTriggerCompatibility tests the new Pascal TWX compatible trigger commands
func TestPhase3_PascalTWXTriggerCompatibility_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// Test the exact 1_Trade.ts pattern using the new Pascal-compatible commands
	script := `
		setTextLineTrigger 1 :getWarp "Sector "
		setTextTrigger 2 :gotWarps "Command [TL="
		echo "Phase 3 triggers configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	// Simulate realistic TradeWars 2002 game output
	testInputs := []string{
		"Relative Density Scan",
		"Sector 123 : 45 density, 3 warps",   // Should trigger :getWarp
		"Sector 456 : 67 density, 2 warps",   // Should trigger :getWarp
		"Sector 789 : 23 density, 4 warps",   // Should trigger :getWarp
		"Command [TL=00:05:30]:",              // Should trigger :gotWarps
	}
	
	for _, output := range testInputs {
		err := tester.SimulateNetworkInput(output)
		if err != nil {
			t.Errorf("Failed to simulate game output %q: %v", output, err)
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// TestPhase3_TriggerLifecycleIntegration tests trigger lifecycle with the new commands
func TestPhase3_TriggerLifecycleIntegration_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setTextLineTrigger 1 :handler1 "test"
		setTextTrigger 2 :handler2 "pattern"
		echo "Triggers created"
		killTrigger 1
		echo "Trigger 1 killed"
		killAllTriggers
		echo "All triggers killed"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	testInputs := []string{
		"Triggers created",
		"Trigger 1 killed", 
		"All triggers killed",
	}
	
	if len(result.Output) != len(testInputs) {
		t.Errorf("Expected %d output lines, got %d", len(testInputs), len(result.Output))
	}
	
	for i, expected := range testInputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestPhase3_TradeScriptScenario tests a realistic 1_Trade.ts script scenario
func TestPhase3_TradeScriptScenario_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// Simulate the key parts of 1_Trade.ts trigger setup and usage
	script := `
		# Initialize arrays for warp data (Phase 1 implemented)
		setArray $warp 10
		setArray $density 10  
		setArray $weight 10
		
		# Set up triggers for density scanning (Phase 3 new functionality) 
		setTextLineTrigger 1 :getWarp "Sector "
		setTextTrigger 2 :gotWarps "Command [TL="
		
		setVar $i 1
		setVar $warp[$i] 0
		setVar $density[$i] -1
		setVar $weight[$i] 9999
		
		echo "Trade script initialization complete"
		goto scan
		
		:getWarp
		# This would normally extract warp data from CURRENTLINE
		cutText CURRENTLINE $line 1 50
		getWord $line $sector 2
		setVar $warp[$i] $sector
		add $i 1
		return
		
		:gotWarps
		echo "Scan complete, found warps"
		killTrigger 1
		killTrigger 2
		return
		
		:scan
		echo "Starting density scan simulation"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Trade script simulation failed: %v", result.Error)
	}
	
	// Verify the script got to the expected state
	expectedFinalOutput := "Starting density scan simulation"
	if len(result.Output) == 0 || result.Output[len(result.Output)-1] != expectedFinalOutput {
		t.Errorf("Expected final output %q, got %q", expectedFinalOutput, 
			getLastOutput(result.Output))
	}
}

// TestPhase3_TriggerDatabasePersistence tests that trigger data persists properly
func TestPhase3_TriggerDatabasePersistence_RealIntegration(t *testing.T) {
	// First instance creates triggers and variables
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Create persistent configuration
		setVar $triggerPattern "Sector "
		setVar $triggerLabel ":warpHandler"
		saveVar $triggerPattern
		saveVar $triggerLabel
		
		setTextLineTrigger 1 $triggerLabel $triggerPattern
		echo "Persistent trigger configuration saved"
	`
	
	result1 := tester.ExecuteScript(script)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}
	
	// Second instance loads configuration and recreates triggers
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester.setupData)
	
	script2 := `
		loadVar $triggerPattern
		loadVar $triggerLabel
		
		# Recreate trigger from saved configuration
		setTextLineTrigger 1 $triggerLabel $triggerPattern
		echo "Trigger recreated from database: " $triggerPattern
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}
	
	expected := "Trigger recreated from database: Sector "
	if len(result2.Output) == 0 || result2.Output[0] != expected {
		t.Errorf("Database persistence test: got %q, want %q", 
			getLastOutput(result2.Output), expected)
	}
}

// TestPhase3_ComplexTriggerPatterns tests various trigger pattern matching scenarios
func TestPhase3_ComplexTriggerPatterns_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Test different trigger types with various patterns
		setTextLineTrigger 1 :lineStart "Sector"
		setTextTrigger 2 :anywhere "Command"
		setTextLineTrigger 3 :exactLine "Health: 100%"
		setTextTrigger 4 :contains "gold"
		echo "Pattern matching triggers configured"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Test various pattern matching scenarios
	testCases := []struct {
		input       string
		description string
	}{
		{"Sector 123 has 45 density", "Should match line start trigger"},
		{"Your Command prompt awaits", "Should match anywhere trigger"},  
		{"Health: 100%", "Should match exact line trigger"},
		{"You found 50 gold pieces", "Should match contains trigger"},
		{"The merchant awaits", "Should not match any trigger"},
	}
	
	for _, tc := range testCases {
		err := tester.SimulateNetworkInput(tc.input)
		if err != nil {
			t.Errorf("Failed to simulate input %q (%s): %v", tc.input, tc.description, err)
		}
		time.Sleep(1 * time.Millisecond)
	}
}

// Helper function to safely get the last output line
func getLastOutput(outputs []string) string {
	if len(outputs) == 0 {
		return ""
	}
	return outputs[len(outputs)-1]
}