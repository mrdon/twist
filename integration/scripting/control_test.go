//go:build integration

package scripting

import (
	"strings"
	"testing"
)

func TestGotoCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Before goto"
		$counter := 1
		goto skip_section
		echo "This should be skipped"
		$counter := 999

		:skip_section
		echo "After goto"
		$counter := 2
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
		$main_var := 1
		gosub subroutine
		echo "Main: After subroutine"
		add $main_var 100 $main_var
		goto end

		:subroutine
		echo "Subroutine: Start"
		$sub_var := 10
		add $main_var $sub_var $main_var
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
		"Final main_var: 111",  // 1 + 10 + 100 = 111
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
		$depth := 0
		echo "Main: Starting"
		gosub level1
		echo "Main: Done"
		goto end

		:level1
		$depth := 1
		echo "Level1: Starting"
		gosub level2
		echo "Level1: After level2"
		add $depth 10 $depth
		return

		:level2  
		$depth := 2
		echo "Level2: Starting"
		gosub level3
		echo "Level2: After level3"
		add $depth 20 $depth
		return

		:level3
		$depth := 3
		echo "Level3: Only level"
		add $depth 30 $depth
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
				$test_val := 0
				$result := 1
				branch $test_val branch_target
				$result := 999
				goto end
				:branch_target
				$result := 2
				:end
				echo "Result: " $result
			`,
			expect: "Result: 2", // Should branch because test_val is 0
		},
		{
			name: "No branch on non-zero",
			script: `
				$test_val := 5
				$result := 1
				branch $test_val branch_target
				$result := 3
				goto end
				:branch_target
				$result := 999
				:end
				echo "Result: " $result
			`,
			expect: "Result: 3", // Should NOT branch because test_val is non-zero
		},
		{
			name: "Branch on empty string",
			script: `
				$test_val := ""
				$result := 1
				branch $test_val branch_target
				$result := 999
				goto end
				:branch_target
				$result := 4
				:end
				echo "Result: " $result
			`,
			expect: "Result: 4", // Should branch because test_val is empty
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
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		$counter := 0
		gosub increment_counter
		gosub increment_counter  
		gosub increment_counter
		savevar $counter
		goto end

		:increment_counter
		add $counter 1 $counter
		return

		:end
		echo "Final counter: " $counter
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Fatalf("Script execution failed: %v", result1.Error)
	}

	if len(result1.Output) != 1 || result1.Output[0] != "Final counter: 3" {
		t.Errorf("Expected 'Final counter: 3', got %v", result1.Output)
	}

	// Create new VM instance sharing same database
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadvar $counter
		gosub double_counter
		savevar $counter
		goto end

		:double_counter
		multiply $counter 2 $counter
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
		$factorial_input := 5
		$factorial_result := 1
		gosub calculate_factorial
		echo "Factorial result: " $factorial_result
		goto end

		:calculate_factorial
		$counter := $factorial_input
		:factorial_loop
		branch $counter factorial_done
		multiply $factorial_result $counter $factorial_result
		subtract $counter 1 $counter
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
		"Factorial result: 120",  // 5! = 120
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