package scripting

import (
	"strings"
	"testing"
)

// TestSaveVarCommand_RealIntegration tests SAVEVAR command with real database persistence
func TestSaveVarCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test saving different types of variables
		setVar $string_var "Hello, World!"
		setVar $number_var 42.5
		setVar $empty_var ""
		
		saveVar $string_var
		saveVar $number_var
		saveVar $empty_var
		
		echo "Saved string variable: " $string_var
		echo "Saved number variable: " $number_var
		echo "Saved empty variable: [" $empty_var "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Saved string variable: Hello, World!",
		"Saved number variable: 42.5",
		"Saved empty variable: []",
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestLoadVarCommand_RealIntegration tests LOADVAR command with real database persistence
func TestLoadVarCommand_RealIntegration(t *testing.T) {
	// First script execution - save variables
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $persistent_string "Persistent Value"
		setVar $persistent_number 123.45
		saveVar $persistent_string
		saveVar $persistent_number
		echo "Variables saved"
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}

	// Second script execution - load variables (simulates VM restart)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		# Variables should be empty initially
		echo "Before load - string: [" $persistent_string "]"
		echo "Before load - number: [" $persistent_number "]"
		
		# Load from database
		loadVar $persistent_string
		loadVar $persistent_number
		
		echo "After load - string: " $persistent_string
		echo "After load - number: " $persistent_number
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}

	if len(result2.Output) != 4 {
		t.Errorf("Expected 4 output lines from second script, got %d", len(result2.Output))
	}

	// Variables should be empty before loading
	if len(result2.Output) > 0 && result2.Output[0] != "Before load - string: []" {
		t.Errorf("Pre-load string: got %q, want %q", result2.Output[0], "Before load - string: []")
	}

	if len(result2.Output) > 1 && result2.Output[1] != "Before load - number: []" {
		t.Errorf("Pre-load number: got %q, want %q", result2.Output[1], "Before load - number: []")
	}

	// Variables should be loaded correctly
	if len(result2.Output) > 2 && result2.Output[2] != "After load - string: Persistent Value" {
		t.Errorf("Post-load string: got %q, want %q", result2.Output[2], "After load - string: Persistent Value")
	}

	if len(result2.Output) > 3 && result2.Output[3] != "After load - number: 123.45" {
		t.Errorf("Post-load number: got %q, want %q", result2.Output[3], "After load - number: 123.45")
	}
}

// TestVariablePersistence_CrossInstance tests variable persistence across multiple VM instances
func TestVariablePersistence_CrossInstance_RealIntegration(t *testing.T) {
	// Instance 1: Set and save variables
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $counter 1
		setVar $message "First instance"
		setVar $pi 3.14159
		
		saveVar $counter
		saveVar $message
		saveVar $pi
		
		echo "Instance 1 saved: counter=" $counter " message=" $message " pi=" $pi
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Instance 1 execution failed: %v", result1.Error)
	}

	// Instance 2: Load, modify, and save variables
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $counter
		loadVar $message
		loadVar $pi
		
		# Modify the variables
		setVar $counter 2
		setVar $message "Second instance"
		# pi stays the same
		
		saveVar $counter
		saveVar $message
		saveVar $pi
		
		echo "Instance 2 loaded and modified: counter=" $counter " message=" $message " pi=" $pi
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Instance 2 execution failed: %v", result2.Error)
	}

	// Instance 3: Load the modified variables
	tester3 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script3 := `
		loadVar $counter
		loadVar $message
		loadVar $pi
		
		echo "Instance 3 loaded: counter=" $counter " message=" $message " pi=" $pi
	`

	result3 := tester3.ExecuteScript(script3)
	if result3.Error != nil {
		t.Errorf("Instance 3 execution failed: %v", result3.Error)
	}

	// Verify final state
	if len(result3.Output) != 1 {
		t.Errorf("Expected 1 output line from instance 3, got %d", len(result3.Output))
	}

	expected := "Instance 3 loaded: counter=2 message=Second instance pi=3.14159"
	if len(result3.Output) > 0 && result3.Output[0] != expected {
		t.Errorf("Final persistence check: got %q, want %q", result3.Output[0], expected)
	}
}

// TestLoadVar_NonExistentVariable tests loading variables that don't exist in database
func TestLoadVar_NonExistentVariable_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Try to load variables that don't exist
		loadVar $nonexistent_var
		loadVar $another_missing_var
		
		echo "Loaded nonexistent: [" $nonexistent_var "]"
		echo "Loaded another missing: [" $another_missing_var "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	// Non-existent variables should load as empty strings
	if len(result.Output) > 0 && result.Output[0] != "Loaded nonexistent: []" {
		t.Errorf("Non-existent var 1: got %q, want %q", result.Output[0], "Loaded nonexistent: []")
	}

	if len(result.Output) > 1 && result.Output[1] != "Loaded another missing: []" {
		t.Errorf("Non-existent var 2: got %q, want %q", result.Output[1], "Loaded another missing: []")
	}
}

// TestVariablePersistence_ComplexWorkflow tests a complex workflow with multiple save/load cycles
func TestVariablePersistence_ComplexWorkflow_RealIntegration(t *testing.T) {
	// Workflow step 1: Initialize game state
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $player_name "TestPlayer"
		setVar $player_level 1
		setVar $player_gold 100
		setVar $current_sector 1
		
		saveVar $player_name
		saveVar $player_level
		saveVar $player_gold
		saveVar $current_sector
		
		echo "Game initialized - Player: " $player_name " Level: " $player_level " Gold: " $player_gold
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Workflow step 1 failed: %v", result1.Error)
	}

	// Workflow step 2: Simulate gameplay (level up, earn gold, move)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		# Load current state
		loadVar $player_name
		loadVar $player_level
		loadVar $player_gold
		loadVar $current_sector
		
		# Simulate gaining a level and earning gold
		setVar $player_level 2
		setVar $player_gold 250
		setVar $current_sector 5
		
		# Save updated state
		saveVar $player_level
		saveVar $player_gold
		saveVar $current_sector
		
		echo "Progress update - " $player_name " reached Level: " $player_level " Gold: " $player_gold " Sector: " $current_sector
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Workflow step 2 failed: %v", result2.Error)
	}

	// Workflow step 3: Load final state and verify
	tester3 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script3 := `
		loadVar $player_name
		loadVar $player_level
		loadVar $player_gold
		loadVar $current_sector
		
		echo "Final state - Player: " $player_name " Level: " $player_level " Gold: " $player_gold " Sector: " $current_sector
	`

	result3 := tester3.ExecuteScript(script3)
	if result3.Error != nil {
		t.Errorf("Workflow step 3 failed: %v", result3.Error)
	}

	if len(result3.Output) != 1 {
		t.Errorf("Expected 1 output line from final verification, got %d", len(result3.Output))
	}

	expected := "Final state - Player: TestPlayer Level: 2 Gold: 250 Sector: 5"
	if len(result3.Output) > 0 && result3.Output[0] != expected {
		t.Errorf("Final workflow state: got %q, want %q", result3.Output[0], expected)
	}
}

// TestVariablePersistence_SpecialCharacters tests persistence of variables with special characters
func TestVariablePersistence_SpecialCharacters_RealIntegration(t *testing.T) {
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $special_chars "Hello\nWorld\tTab'Quote'Apostrophe"
		setVar $unicode_text "Test αβγ 中文 🚀"
		setVar $with_numbers "Mix123ed-Ch@rs!"
		
		saveVar $special_chars
		saveVar $unicode_text
		saveVar $with_numbers
		
		echo "Saved special characters"
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Save special chars failed: %v", result1.Error)
	}

	// Load in new instance
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $special_chars
		loadVar $unicode_text
		loadVar $with_numbers
		
		echo "Loaded special: " $special_chars
		echo "Loaded unicode: " $unicode_text
		echo "Loaded mixed: " $with_numbers
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Load special chars failed: %v", result2.Error)
	}

	if len(result2.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result2.Output))
	}

	// Verify special characters were preserved
	if len(result2.Output) > 0 && !strings.Contains(result2.Output[0], "Hello") {
		t.Errorf("Special chars not preserved: %q", result2.Output[0])
	}

	if len(result2.Output) > 1 && !strings.Contains(result2.Output[1], "Test") {
		t.Errorf("Unicode text not preserved: %q", result2.Output[1])
	}

	if len(result2.Output) > 2 && !strings.Contains(result2.Output[2], "Mix123ed") {
		t.Errorf("Mixed chars not preserved: %q", result2.Output[2])
	}
}

// TestVariableOverwrite_RealIntegration tests overwriting existing variables
func TestVariableOverwrite_RealIntegration(t *testing.T) {
	// Save initial value
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $overwrite_test "Original Value"
		saveVar $overwrite_test
		echo "Saved original: " $overwrite_test
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Save original failed: %v", result1.Error)
	}

	// Overwrite with new value
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		setVar $overwrite_test "New Value"
		saveVar $overwrite_test
		echo "Saved new: " $overwrite_test
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Save new value failed: %v", result2.Error)
	}

	// Verify overwrite worked
	tester3 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script3 := `
		loadVar $overwrite_test
		echo "Loaded after overwrite: " $overwrite_test
	`

	result3 := tester3.ExecuteScript(script3)
	if result3.Error != nil {
		t.Errorf("Load after overwrite failed: %v", result3.Error)
	}

	if len(result3.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result3.Output))
	}

	expected := "Loaded after overwrite: New Value"
	if len(result3.Output) > 0 && result3.Output[0] != expected {
		t.Errorf("Overwrite verification: got %q, want %q", result3.Output[0], expected)
	}
}

// TestTWXSetVar_OriginalSyntax tests the original TWX setVar syntax
func TestTWXSetVar_OriginalSyntax_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test original TWX setVar syntax
		setVar $counter 1
		setVar $message "Hello World"
		setVar $decimal 42.5
		setVar $empty ""
		
		echo "Counter: " $counter
		echo "Message: " $message
		echo "Decimal: " $decimal
		echo "Empty: [" $empty "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX setVar script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Counter: 1",
		"Message: Hello World",
		"Decimal: 42.5",
		"Empty: []",
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("SetVar output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXSetVar_WithPersistence tests setVar with database persistence
func TestTWXSetVar_WithPersistence_RealIntegration(t *testing.T) {
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $persistent_var "TWX Style"
		saveVar $persistent_var
		echo "Saved using setVar: " $persistent_var
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Fatalf("TWX setVar persistence script failed: %v", result1.Error)
	}

	// Load in new instance
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $persistent_var
		echo "Loaded: " $persistent_var
		
		setVar $persistent_var "Modified TWX Style"
		echo "Modified: " $persistent_var
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Fatalf("TWX setVar load script failed: %v", result2.Error)
	}

	expectedOutputs := []string{
		"Loaded: TWX Style",
		"Modified: Modified TWX Style",
	}

	if len(result2.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result2.Output), result2.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result2.Output) && result2.Output[i] != expected {
			t.Errorf("SetVar persistence output %d: got %q, want %q", i+1, result2.Output[i], expected)
		}
	}
}

// TestGoStyleAssignmentRejection tests that Go-style variable assignments are properly rejected
func TestGoStyleAssignmentRejection(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test script with Go-style assignment that should be rejected
	script := `
		$invalid_var := "this should fail"
		echo "This should not execute"
	`

	result := tester.ExecuteScript(script)

	// The script should fail due to invalid syntax
	if result.Error == nil {
		t.Error("Expected script to fail with Go-style assignment syntax, but it succeeded")
	}

	// Verify the error message indicates syntax issue
	errorMsg := result.Error.Error()
	if !strings.Contains(strings.ToLower(errorMsg), "syntax") &&
		!strings.Contains(strings.ToLower(errorMsg), "invalid") &&
		!strings.Contains(strings.ToLower(errorMsg), "unknown") {
		t.Errorf("Expected syntax-related error message, got: %v", errorMsg)
	}
}
