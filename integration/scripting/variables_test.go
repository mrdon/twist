//go:build integration

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
		$string_var := "Hello, World!"
		$number_var := 42.5
		$empty_var := ""
		
		savevar $string_var
		savevar $number_var
		savevar $empty_var
		
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
		$persistent_string := "Persistent Value"
		$persistent_number := 123.45
		savevar $persistent_string
		savevar $persistent_number
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
		loadvar $persistent_string
		loadvar $persistent_number
		
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
		$counter := 1
		$message := "First instance"
		$pi := 3.14159
		
		savevar $counter
		savevar $message
		savevar $pi
		
		echo "Instance 1 saved: counter=" $counter " message=" $message " pi=" $pi
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Instance 1 execution failed: %v", result1.Error)
	}
	
	// Instance 2: Load, modify, and save variables
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script2 := `
		loadvar $counter
		loadvar $message
		loadvar $pi
		
		# Modify the variables
		$counter := 2
		$message := "Second instance"
		# pi stays the same
		
		savevar $counter
		savevar $message
		savevar $pi
		
		echo "Instance 2 loaded and modified: counter=" $counter " message=" $message " pi=" $pi
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Instance 2 execution failed: %v", result2.Error)
	}
	
	// Instance 3: Load the modified variables
	tester3 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script3 := `
		loadvar $counter
		loadvar $message
		loadvar $pi
		
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
		loadvar $nonexistent_var
		loadvar $another_missing_var
		
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
		$player_name := "TestPlayer"
		$player_level := 1
		$player_gold := 100
		$current_sector := 1
		
		savevar $player_name
		savevar $player_level
		savevar $player_gold
		savevar $current_sector
		
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
		loadvar $player_name
		loadvar $player_level
		loadvar $player_gold
		loadvar $current_sector
		
		# Simulate gaining a level and earning gold
		$player_level := 2
		$player_gold := 250
		$current_sector := 5
		
		# Save updated state
		savevar $player_level
		savevar $player_gold
		savevar $current_sector
		
		echo "Progress update - " $player_name " reached Level: " $player_level " Gold: " $player_gold " Sector: " $current_sector
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Workflow step 2 failed: %v", result2.Error)
	}
	
	// Workflow step 3: Load final state and verify
	tester3 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script3 := `
		loadvar $player_name
		loadvar $player_level
		loadvar $player_gold
		loadvar $current_sector
		
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
		$special_chars := "Hello\nWorld\tTab\"Quote'Apostrophe"
		$unicode_text := "Test αβγ 中文 🚀"
		$with_numbers := "Mix123ed-Ch@rs!"
		
		savevar $special_chars
		savevar $unicode_text
		savevar $with_numbers
		
		echo "Saved special characters"
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Save special chars failed: %v", result1.Error)
	}
	
	// Load in new instance
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script2 := `
		loadvar $special_chars
		loadvar $unicode_text
		loadvar $with_numbers
		
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
		$overwrite_test := "Original Value"
		savevar $overwrite_test
		echo "Saved original: " $overwrite_test
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Save original failed: %v", result1.Error)
	}
	
	// Overwrite with new value
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script2 := `
		$overwrite_test := "New Value"
		savevar $overwrite_test
		echo "Saved new: " $overwrite_test
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Save new value failed: %v", result2.Error)
	}
	
	// Verify overwrite worked
	tester3 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script3 := `
		loadvar $overwrite_test
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