//go:build integration

package scripting

import (
	"strings"
	"testing"
)

// TestArrayVariables_PascalCompatibility_RealIntegration validates our array implementation against Pascal TWX behavior
func TestArrayVariables_PascalCompatibility_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Test basic auto-vivification like Pascal TVarParam - use simpler approach
		# Based on working syntax from completed Phase 1
		SETARRAY sectors 10
		SETVAR sectors[1] "123"
		SETVAR sectors[5] "456"
		echo "Basic arrays work"
		
		# Test basic retrieval using GETVAR instead of direct access
		GETVAR sectors[1] $result1
		GETVAR sectors[5] $result5  
		echo "Results: " $result1 " and " $result5
		
		# Test multi-dimensional - if supported
		SETARRAY data[3][3] 
		SETVAR data[1][2] "nested"
		GETVAR data[1][2] $nested_result
		echo "Multi-dim: " $nested_result
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) < 3 {
		t.Errorf("Expected at least 3 output lines, got %d", len(result.Output))
	}
	
	expectedOutputs := []string{
		"Basic arrays work",
		"Results: 123 and 456", 
		"Multi-dim: nested",
	}
	
	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Pascal compatibility test %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestArrayVariables_SetArrayBehavior_RealIntegration tests Pascal SetArray method behavior
func TestArrayVariables_SetArrayBehavior_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Pascal SetArray creates elements with default "0" values
		# This should create sectors[1], sectors[2], sectors[3] all with value "0"
		setarray $sectors 3
		echo "Array element 1: [" $sectors[1] "]"
		echo "Array element 2: [" $sectors[2] "]" 
		echo "Array element 3: [" $sectors[3] "]"
		
		# Test that we can override the default values
		setvar $sectors[2] "modified"
		echo "Modified element 2: [" $sectors[2] "]"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d", len(result.Output))
	}
	
	expectedOutputs := []string{
		"Array element 1: [0]",      // Pascal default initialization
		"Array element 2: [0]",      // Pascal default initialization  
		"Array element 3: [0]",      // Pascal default initialization
		"Modified element 2: [modified]", // Override works
	}
	
	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("SetArray behavior %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestArrayVariables_StaticArrayBounds_RealIntegration tests Pascal static array bounds checking
func TestArrayVariables_StaticArrayBounds_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// Test accessing within bounds (should work)
	script1 := `
		setarray $static_test 3
		setvar $static_test[1] "valid"
		setvar $static_test[3] "also_valid"
		echo "Within bounds: " $static_test[1] " " $static_test[3]
	`
	
	result1 := tester.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("Valid bounds script failed: %v", result1.Error)
	}
	
	if len(result1.Output) != 1 {
		t.Errorf("Expected 1 output line for valid bounds, got %d", len(result1.Output))
	}
	
	if len(result1.Output) > 0 && result1.Output[0] != "Within bounds: valid also_valid" {
		t.Errorf("Valid bounds output: got %q, want %q", result1.Output[0], "Within bounds: valid also_valid")
	}
	
	// Test accessing out of bounds (should error like Pascal)
	tester2 := NewIntegrationScriptTester(t)
	script2 := `
		setarray $static_test 3
		setvar $static_test[5] "out_of_bounds"
		echo "Should not reach here"
	`
	
	result2 := tester2.ExecuteScript(script2)
	
	// Should have Pascal-style error message
	if result2.Error == nil {
		t.Errorf("Expected error for out-of-bounds access, but got none")
	}
	
	if result2.Error != nil {
		errorMsg := result2.Error.Error()
		if !strings.Contains(errorMsg, "out of range") || !strings.Contains(errorMsg, "must be 1-3") {
			t.Errorf("Expected Pascal-style bounds error, got: %v", result2.Error)
		}
	}
	
	// No output should be produced when error occurs
	if len(result2.Output) != 0 {
		t.Errorf("Expected no output when bounds error occurs, got %d lines", len(result2.Output))
	}
}

// TestArrayVariables_MultiParameterSetVar_RealIntegration tests Pascal setVar concatenation
func TestArrayVariables_MultiParameterSetVar_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Pascal supports: setVar $result "part1" "part2" "part3"
		# Should concatenate all parameters after the variable
		setvar $message "Hello" " " "World" "!"
		echo "Concatenated: " $message
		
		# Test with array variables
		setvar $sectors[1] "Sector" " " "123"
		echo "Array concat: " $sectors[1]
		
		# Test with mixed parameter types
		setvar $mixed "Count: " 42 " items"
		echo "Mixed types: " $mixed
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}
	
	expectedOutputs := []string{
		"Concatenated: Hello World!",
		"Array concat: Sector 123", 
		"Mixed types: Count: 42 items",
	}
	
	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Multi-parameter setVar %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestArrayVariables_PascalIndexing_RealIntegration tests that TWX uses 1-based indexing consistently
func TestArrayVariables_PascalIndexing_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Pascal TWX uses 1-based indexing throughout
		setarray $test 3
		
		# Elements should be created as test[1], test[2], test[3]
		# NOT test[0], test[1], test[2]
		setvar $test[1] "first"
		setvar $test[2] "second" 
		setvar $test[3] "third"
		
		echo "1-based indexing: " $test[1] " " $test[2] " " $test[3]
		
		# Accessing index 0 should be out of bounds for static array
		# (This will be tested in bounds checking test)
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	expected := "1-based indexing: first second third"
	if len(result.Output) > 0 && result.Output[0] != expected {
		t.Errorf("1-based indexing test: got %q, want %q", result.Output[0], expected)
	}
}

// TestArrayVariables_DatabasePersistence_RealIntegration tests array persistence across VM instances
func TestArrayVariables_DatabasePersistence_RealIntegration(t *testing.T) {
	// First script execution - create and save arrays
	tester1 := NewIntegrationScriptTester(t)
	
	script1 := `
		setvar $persistent[1] "saved_value_1"
		setvar $persistent[2] "saved_value_2"
		setarray $static_array 2
		setvar $static_array[1] "static_saved"
		savevar $persistent[1]
		savevar $persistent[2] 
		savevar $static_array[1]
		echo "Saved arrays"
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}
	
	// Second script execution - load arrays from database (simulates VM restart)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script2 := `
		loadvar $persistent[1]
		loadvar $persistent[2]
		loadvar $static_array[1] 
		echo "Loaded dynamic: " $persistent[1] " " $persistent[2]
		echo "Loaded static: " $static_array[1]
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}
	
	if len(result2.Output) != 2 {
		t.Errorf("Expected 2 output lines from second script, got %d", len(result2.Output))
	}
	
	expectedOutputs := []string{
		"Loaded dynamic: saved_value_1 saved_value_2",
		"Loaded static: static_saved",
	}
	
	for i, expected := range expectedOutputs {
		if i < len(result2.Output) && result2.Output[i] != expected {
			t.Errorf("Array persistence %d: got %q, want %q", i+1, result2.Output[i], expected)
		}
	}
}