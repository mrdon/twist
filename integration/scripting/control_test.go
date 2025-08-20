package scripting

import (
	"strings"
	"testing"
)

func TestGotoCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Before goto"
		setVar $counter 1
		goto skip_section
		echo "This should be skipped"
		setVar $counter 999

		:skip_section
		echo "After goto"
		setVar $counter 2
		echo "Counter: " $counter
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}

	// Verify that goto worked - should see "Before goto", "After goto", "Counter: 2"
	// but NOT "This should be skipped"
	expectedOutputs := []string{
		"Before goto",
		"After goto",
		"Counter: 2",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("GOTO output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}

	// Verify that the skipped section was not executed
	for _, line := range result.Output {
		if strings.Contains(line, "This should be skipped") {
			t.Errorf("Found skipped section output: %s", line)
		}
	}
}

func TestGosubReturn_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Main: Start"
		setVar $main_var 1
		gosub subroutine
		echo "Main: After subroutine"
		add $main_var 100
		goto end

		:subroutine
		echo "Subroutine: Start"
		setVar $sub_var 10
		add $main_var $sub_var
		echo "Subroutine: End"
		return

		:end
		echo "Main: End"
		echo "Final main_var: " $main_var
		echo "Final sub_var: " $sub_var
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Main: Start",
		"Subroutine: Start",
		"Subroutine: End",
		"Main: After subroutine",
		"Main: End",
		"Final main_var: 111", // 1 + 10 + 100 = 111
		"Final sub_var: 10",
	}

	if len(result.Output) != 7 {
		t.Errorf("Expected 7 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("GOSUB/RETURN output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

func TestNestedGosub_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $depth 0
		echo "Main: Starting"
		gosub level1
		echo "Main: Done"
		goto end

		:level1
		setVar $depth 1
		echo "Level1: Starting"
		gosub level2
		echo "Level1: After level2"
		add $depth 10
		return

		:level2  
		setVar $depth 2
		echo "Level2: Starting"
		gosub level3
		echo "Level2: After level3"
		add $depth 20
		return

		:level3
		setVar $depth 3
		echo "Level3: Only level"
		add $depth 30
		return

		:end
		echo "Script complete"
		echo "Final depth: " $depth
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}

	// Should see nested execution and final depth of 3 + 30 + 20 + 10 = 63
	hasMainStarting := false
	hasLevel1Starting := false
	hasLevel2Starting := false
	hasLevel3Only := false
	hasFinalDepth63 := false

	for _, line := range result.Output {
		if line == "Main: Starting" {
			hasMainStarting = true
		}
		if line == "Level1: Starting" {
			hasLevel1Starting = true
		}
		if line == "Level2: Starting" {
			hasLevel2Starting = true
		}
		if line == "Level3: Only level" {
			hasLevel3Only = true
		}
		if line == "Final depth: 63" {
			hasFinalDepth63 = true
		}
	}

	if !hasMainStarting || !hasLevel1Starting || !hasLevel2Starting || !hasLevel3Only {
		t.Errorf("Missing expected nested gosub output in: %v", result.Output)
	}

	if !hasFinalDepth63 {
		t.Errorf("Expected final depth 63, output was: %v", result.Output)
	}
}

// TestTWXLooping_WhileLoop tests TWX while loop syntax
func TestTWXLooping_WhileLoop_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test TWX while loop
		setVar $counter 1
		
		while ($counter <= 3)
			echo "Loop iteration: " $counter
			add $counter 1
		end
		
		echo "Loop complete"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX while loop script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Loop iteration: 1",
		"Loop iteration: 2",
		"Loop iteration: 3",
		"Loop complete",
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("While loop output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXConditionals_IfElse tests TWX if/else syntax
func TestTWXConditionals_IfElse_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test TWX if/else conditionals
		setVar $value 42
		
		if ($value = 42)
			echo "Value is 42"
		else
			echo "Value is not 42"
		end
		
		if ($value > 50)
			echo "Value is greater than 50"
		elseif ($value > 30)
			echo "Value is greater than 30"
		else
			echo "Value is 30 or less"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX if/else script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Value is 42",
		"Value is greater than 30",
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("If/else output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXComplexScript_RealWorldPattern tests a pattern similar to real TWX scripts
func TestTWXComplexScript_RealWorldPattern_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// This script mimics the structure found in real TWX scripts
	script := `
		# Complex script similar to real TWX patterns
		setVar $scriptName "Test Script"
		setVar $version "1.0"
		
		# Show banner like real scripts
		echo "**" $scriptName " v" $version "**"
		
		# Initialize variables
		setVar $counter 0
		setVar $maxRuns 3
		
		# Main loop
		:main_loop
		if ($counter >= $maxRuns)
			goto script_end
		end
		
		echo "Run " $counter ": Processing..."
		
		# Simulate some work
		gosub do_work
		
		add $counter 1
		goto main_loop
		
		:do_work
		setVar $workResult "completed"
		echo "Work " $workResult
		return
		
		:script_end
		echo "Script finished after " $counter " runs"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX complex script failed: %v", result.Error)
	}

	// Should see banner, 3 run messages with work results, and final message
	if len(result.Output) != 8 {
		t.Errorf("Expected 8 output lines, got %d: %v", len(result.Output), result.Output)
	}

	// Check banner
	if len(result.Output) > 0 && !strings.Contains(result.Output[0], "Test Script v1") {
		t.Errorf("Banner incorrect: got %q", result.Output[0])
	}

	// Check final message
	lastLine := result.Output[len(result.Output)-1]
	if !strings.Contains(lastLine, "Script finished after 3 runs") {
		t.Errorf("Final message incorrect: got %q", lastLine)
	}
}

func TestBranchCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	tests := []struct {
		name   string
		script string
		expect string
	}{
		{
			name: "Branch on zero",
			script: `
				setVar $test_val 0
				setVar $result 1
				branch $test_val mylabel
				setVar $result 999
				goto end
				:mylabel
				setVar $result 2
				:end
				echo "Result: " $result
			`,
			expect: "Result: 2", // Should branch because test_val is 0
		},
		{
			name: "Branch on non-one value",
			script: `
				setVar $test_val 5
				setVar $result 1
				branch $test_val mylabel
				setVar $result 3
				goto end
				:mylabel
				setVar $result 999
				:end
				echo "Result: " $result
			`,
			expect: "Result: 999", // Should branch because test_val (5) is not equal to 1
		},
		{
			name: "Branch on empty string",
			script: `
				setVar $test_val ""
				setVar $result 1
				branch $test_val mylabel
				setVar $result 999
				goto end
				:mylabel
				setVar $result 4
				:end
				echo "Result: " $result
			`,
			expect: "Result: 4", // Should branch because test_val is empty (converts to 0, not equal to 1)
		},
		{
			name: "No branch on one",
			script: `
				setVar $test_val 1
				setVar $result 1
				branch $test_val mylabel
				setVar $result 5
				goto end
				:mylabel
				setVar $result 999
				:end
				echo "Result: " $result
			`,
			expect: "Result: 5", // Should NOT branch because test_val equals 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tester.ExecuteScript(tt.script)
			if result.Error != nil {
				t.Fatalf("Script execution failed: %v", result.Error)
			}

			if len(result.Output) != 1 {
				t.Errorf("Expected 1 output line, got %d: %v", len(result.Output), result.Output)
			}

			if len(result.Output) > 0 && result.Output[0] != tt.expect {
				t.Errorf("Expected %s, got %s", tt.expect, result.Output[0])
			}
		})
	}
}

func TestControlFlowPersistence_CrossInstance_RealIntegration(t *testing.T) {
	// Test that variables set by control flow persist across VM instances
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $counter 0
		gosub increment_counter
		gosub increment_counter  
		gosub increment_counter
		saveVar $counter
		goto end

		:increment_counter
		add $counter 1
		return

		:end
		echo "Final counter: " $counter
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 || result.Output[0] != "Final counter: 3" {
		t.Errorf("Expected 'Final counter: 3', got %v", result.Output)
	}

	// Create new VM instance sharing same database
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester.setupData)

	script2 := `
		loadVar $counter
		gosub double_counter
		saveVar $counter
		goto end

		:double_counter
		multiply $counter 2
		return

		:end
		echo "Doubled counter: " $counter
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Fatalf("Script execution in second VM failed: %v", result2.Error)
	}

	if len(result2.Output) != 1 || result2.Output[0] != "Doubled counter: 6" {
		t.Errorf("Expected 'Doubled counter: 6', got %v", result2.Output)
	}
}

func TestErrorHandling_InvalidLabel_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test GOTO to non-existent label
	result1 := tester.ExecuteScript("goto nonexistent_label")
	if result1.Error == nil {
		t.Errorf("Expected error for GOTO to non-existent label, but script succeeded")
	}

	// Test GOSUB to non-existent label
	result2 := tester.ExecuteScript("gosub nonexistent_subroutine")
	if result2.Error == nil {
		t.Errorf("Expected error for GOSUB to non-existent label, but script succeeded")
	}
}

func TestErrorHandling_ReturnWithoutGosub_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test RETURN without preceding GOSUB
	result := tester.ExecuteScript("return")
	if result.Error == nil {
		t.Errorf("Expected error for RETURN without GOSUB, but script succeeded")
	}
}

func TestComplexControlFlow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Complex control flow that tests multiple features
	script := `
		setVar $factorial_input 5
		setVar $factorial_result 1
		gosub calculate_factorial
		echo "Factorial result: " $factorial_result
		goto end

		:calculate_factorial
		setVar $counter $factorial_input
		:factorial_loop
		if ($counter <= 0)
			goto factorial_done
		end
		multiply $factorial_result $counter
		subtract $counter 1
		goto factorial_loop
		:factorial_done
		return

		:end
		echo "Calculation complete"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Complex control flow script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Factorial result: 120", // 5! = 120
		"Calculation complete",
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Complex control flow output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// ======= NEW PHASE 2 TESTS: Macro Preprocessor Integration =======

// TestMacroPreprocessor_SimpleIf tests basic IF/END macro expansion
func TestMacroPreprocessor_SimpleIf_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $test_value 42
		
		if ($test_value = 42)
			echo "Condition is true"
			setVar $result "success"
		end
		
		echo "After if block"
		echo "Result: " $result
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Simple IF macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Condition is true",
		"After if block",
		"Result: success",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Simple IF output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestMacroPreprocessor_IfElse tests IF/ELSE/END macro expansion
func TestMacroPreprocessor_IfElse_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $test_value 10
		
		if ($test_value > 20)
			echo "Value is greater than 20"
			setVar $result "greater"
		else
			echo "Value is 20 or less"
			setVar $result "less_or_equal"
		end
		
		echo "Final result: " $result
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("IF/ELSE macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Value is 20 or less",
		"Final result: less_or_equal",
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("IF/ELSE output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestMacroPreprocessor_IfElseIf tests IF/ELSEIF/ELSE/END macro expansion
func TestMacroPreprocessor_IfElseIf_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $grade 85
		
		if ($grade >= 90)
			echo "Grade: A"
			setVar $letter "A"
		elseif ($grade >= 80)
			echo "Grade: B"
			setVar $letter "B"
		elseif ($grade >= 70)
			echo "Grade: C"
			setVar $letter "C"
		else
			echo "Grade: F"
			setVar $letter "F"
		end
		
		echo "Letter grade: " $letter
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("IF/ELSEIF/ELSE macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Grade: B",
		"Letter grade: B",
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("IF/ELSEIF/ELSE output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestMacroPreprocessor_SimpleWhile tests basic WHILE/END macro expansion
func TestMacroPreprocessor_SimpleWhile_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $counter 1
		setVar $sum 0
		
		while ($counter <= 5)
			echo "Counter: " $counter
			add $sum $counter
			add $counter 1
		end
		
		echo "Sum: " $sum
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("WHILE macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Counter: 1",
		"Counter: 2",
		"Counter: 3",
		"Counter: 4",
		"Counter: 5",
		"Sum: 15",
	}

	if len(result.Output) != 6 {
		t.Errorf("Expected 6 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("WHILE output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestMacroPreprocessor_NestedIfWhile tests nested control structures
func TestMacroPreprocessor_NestedIfWhile_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $outer 1
		
		while ($outer <= 3)
			echo "Outer loop: " $outer
			setVar $inner 1
			
			while ($inner <= 2)
				if ($inner = 1)
					echo "  Inner first iteration"
				else
					echo "  Inner second iteration"
				end
				add $inner 1
			end
			
			add $outer 1
		end
		
		echo "Nested loops complete"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Nested IF/WHILE macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Outer loop: 1",
		"  Inner first iteration",
		"  Inner second iteration",
		"Outer loop: 2",
		"  Inner first iteration",
		"  Inner second iteration",
		"Outer loop: 3",
		"  Inner first iteration",
		"  Inner second iteration",
		"Nested loops complete",
	}

	if len(result.Output) != 10 {
		t.Errorf("Expected 10 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Nested control output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestMacroPreprocessor_ComplexExpressions tests control flow with complex expressions (from Phase 1)
func TestMacroPreprocessor_ComplexExpressions_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $port 2
		setVar $location "Command"
		setVar $turns 50
		
		# Test complex TWX expressions that were implemented in Phase 1
		if (($port = 2) or ($port = 3)) and ($location <> "Combat")
			echo "Valid port and location"
			
			if ($turns > 0) and ($turns < 100)
				echo "Turns in valid range"
				setVar $status "ready"
			else
				echo "Invalid turn count"
				setVar $status "error"
			end
		else
			echo "Invalid port or location"
			setVar $status "invalid"
		end
		
		echo "Final status: " $status
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Complex expressions macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Valid port and location",
		"Turns in valid range",
		"Final status: ready",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Complex expression output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestMacroPreprocessor_StringConcatenation tests string concatenation in control flow
func TestMacroPreprocessor_StringConcatenation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $shipId 123
		setVar $count 5
		
		if ($count > 0)
			setVar $message ("Ship " & $shipId & " has " & $count & " items")
			echo $message
		else
			echo "No items found"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("String concatenation macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Ship 123 has 5 items",
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d: %v", len(result.Output), result.Output)
	}

	if len(result.Output) > 0 && result.Output[0] != expectedOutputs[0] {
		t.Errorf("String concatenation output: got %q, want %q", result.Output[0], expectedOutputs[0])
	}
}

// TestMacroPreprocessor_ErrorHandling tests error cases in macro preprocessing
func TestMacroPreprocessor_ErrorHandling_RealIntegration(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "IF without END",
			script: `
				if ($value = 1)
					echo "test"
				# Missing END
			`,
			shouldError: true,
			errorMsg:    "IF/WHILE .. END structure mismatch",
		},
		{
			name: "ELSE without IF",
			script: `
				echo "before"
				else
					echo "should error"
				end
			`,
			shouldError: true,
			errorMsg:    "ELSE without IF",
		},
		{
			name: "ELSEIF without IF",
			script: `
				echo "before"
				elseif ($test = 1)
					echo "should error"
				end
			`,
			shouldError: true,
			errorMsg:    "ELSEIF without IF",
		},
		{
			name: "END without IF",
			script: `
				echo "before"
				end
			`,
			shouldError: true,
			errorMsg:    "END without IF",
		},
		{
			name: "ELSE with WHILE",
			script: `
				while ($counter < 5)
					echo "in loop"
				else
					echo "should error"
				end
			`,
			shouldError: true,
			errorMsg:    "cannot use ELSE with WHILE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := NewIntegrationScriptTester(t)
			result := tester.ExecuteScript(tt.script)

			if tt.shouldError {
				if result.Error == nil {
					t.Errorf("Expected error containing %q, but script succeeded", tt.errorMsg)
				} else if !strings.Contains(result.Error.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errorMsg, result.Error)
				}
			} else {
				if result.Error != nil {
					t.Errorf("Expected success, got error: %v", result.Error)
				}
			}
		})
	}
}

// TestMacroPreprocessor_SST_Pattern tests a pattern from the 1_SST.ts script
func TestMacroPreprocessor_SST_Pattern_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// This mimics the pattern found in the 1_SST.ts script
	script := `
		# Simulate the 1_SST.ts script pattern
		setVar $location "Command"
		
		if ($location <> "Command")
			echo "This script must be run from the game command menu"
			# halt would be here in real script
		end
		
		setVar $shipNumber1 42
		echo "Ship ID: " $shipNumber1
		
		setVar $sectorNumber1 100
		setVar $portExists 1
		
		if ($portExists = 1)
			echo "Port found in sector " $sectorNumber1
			
			setVar $credits 1000
			if ($credits >= 500)
				echo "Sufficient credits for trading"
				setVar $canTrade 1
			else
				echo "Insufficient credits"
				setVar $canTrade 0
			end
		else
			echo "No port in this sector"
			setVar $canTrade 0
		end
		
		if ($canTrade = 1)
			echo "Initiating trade sequence"
		else
			echo "Cannot trade at this time"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("SST pattern macro script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Ship ID: 42",
		"Port found in sector 100",
		"Sufficient credits for trading",
		"Initiating trade sequence",
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("SST pattern output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}
