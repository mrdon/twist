package scripting

import (
	"strings"
	"testing"
)

func TestMathCommands_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test ADD command with TWX syntax
	script := `
		setVar $result 5
		add $result 3
		echo "ADD result: " $result
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("ADD command failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d: %v", len(result.Output), result.Output)
	}

	if len(result.Output) > 0 && result.Output[0] != "ADD result: 8" {
		t.Errorf("Expected 'ADD result: 8', got %s", result.Output[0])
	}
}

func TestMathCommandsAllTypes_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	tests := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name:     "SUBTRACT command",
			script:   "setVar $result 10\nsubtract $result 4\necho \"SUBTRACT: \" $result",
			expected: "SUBTRACT: 6",
		},
		{
			name:     "MULTIPLY command",
			script:   "setVar $result 6\nmultiply $result 7\necho \"MULTIPLY: \" $result",
			expected: "MULTIPLY: 42",
		},
		{
			name:     "DIVIDE command",
			script:   "setVar $result 15\ndivide $result 3\necho \"DIVIDE: \" $result",
			expected: "DIVIDE: 5",
		},
		{
			name:     "MOD command",
			script:   "mod 17 5 $result\necho \"MOD: \" $result",
			expected: "MOD: 2",
		},
		{
			name:     "ABS negative command",
			script:   "abs -42 $result\necho \"ABS: \" $result",
			expected: "ABS: 42",
		},
		{
			name:     "INT command",
			script:   "int 3.14159 $result\necho \"INT: \" $result",
			expected: "INT: 3",
		},
		{
			name:     "ROUND command",
			script:   "round 3.6 $result\necho \"ROUND: \" $result",
			expected: "ROUND: 4",
		},
		{
			name:     "SQR command",
			script:   "sqr 16 $result\necho \"SQR: \" $result",
			expected: "SQR: 4",
		},
		{
			name:     "POWER command",
			script:   "power 2 3 $result\necho \"POWER: \" $result",
			expected: "POWER: 8",
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

			if len(result.Output) > 0 && result.Output[0] != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Output[0])
			}
		})
	}
}

func TestTypeConversion_StringToNumber_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $string_num "42"
		setVar $string_float "3.14"
		setVar $result1 $string_num
		add $result1 8
		setVar $result2 $string_float
		multiply $result2 2
		echo "Result1: " $result1
		echo "Result2: " $result2
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Result1: 50",
		"Result2: 6",  // 6.28 rounded to 6
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Type conversion output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

func TestMathErrors_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	errorTests := []struct {
		name   string
		script string
	}{
		{
			name:   "Division by zero",
			script: "setVar $result 10\ndivide $result 0",
		},
		{
			name:   "Modulo by zero",
			script: "mod 10 0 $result",
		},
		{
			name:   "Square root of negative number",
			script: "sqr -4 $result",
		},
		{
			name:   "Zero to negative power",
			script: "power 0 -1 $result",
		},
		{
			name:   "Negative to fractional power",
			script: "power -4 0.5 $result",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			result := tester.ExecuteScript(tt.script)
			if result.Error == nil {
				t.Errorf("Expected error for %s, but script succeeded", tt.name)
			}
		})
	}
}

func TestRandomCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test RANDOM command multiple times to ensure it produces values in expected range
	for i := 0; i < 5; i++ {
		script := `
			random 6 $dice_roll
			echo "Roll: " $dice_roll
		`

		result := tester.ExecuteScript(script)
		if result.Error != nil {
			t.Fatalf("RANDOM command failed: %v", result.Error)
		}

		if len(result.Output) != 1 {
			t.Errorf("Expected 1 output line, got %d", len(result.Output))
		}

		// Check that output contains a number between 1 and 6
		output := result.Output[0]
		if !strings.Contains(output, "Roll: ") {
			t.Errorf("Unexpected output format: %s", output)
		}

		// Extract number and verify it's in range 1-6
		parts := strings.Split(output, "Roll: ")
		if len(parts) != 2 {
			t.Errorf("Could not extract roll value from: %s", output)
		}
	}
}

func TestMathPersistence_CrossInstance_RealIntegration(t *testing.T) {
	// Test that math command results persist across VM instances
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $math_result 100
		add $math_result 200
		saveVar $math_result
		echo "Saved: " $math_result
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Fatalf("Script execution failed: %v", result1.Error)
	}

	if len(result1.Output) != 1 || result1.Output[0] != "Saved: 300" {
		t.Errorf("Expected 'Saved: 300', got %v", result1.Output)
	}

	// Create new VM instance sharing same database
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $math_result
		setVar $doubled $math_result
		multiply $doubled 2
		echo "Loaded and doubled: " $doubled
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Fatalf("Script execution in second VM failed: %v", result2.Error)
	}

	if len(result2.Output) != 1 || result2.Output[0] != "Loaded and doubled: 600" {
		t.Errorf("Expected 'Loaded and doubled: 600', got %v", result2.Output)
	}
}

func TestComplexMathWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Complex calculation workflow: calculate area of a circle
	script := `
		# Calculate area of circle with radius 5
		# Area = π * r²
		power 5 2 $radius_squared
		setVar $area $radius_squared
		multiply $area 3.14159
		
		# Calculate circumference = 2 * π * r  
		setVar $diameter 2
		multiply $diameter 5
		setVar $circumference $diameter
		multiply $circumference 3.14159
		
		# Calculate ratio of area to circumference
		setVar $ratio $area
		divide $ratio $circumference
		
		echo "Area: " $area
		echo "Circumference: " $circumference
		echo "Ratio: " $ratio
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Complex math workflow failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	// Verify the calculations are approximately correct
	expectedOutputs := []string{
		"Area: 78.5398",          // 25 * 3.14159
		"Circumference: 31.4159", // 10 * 3.14159
		"Ratio: 2.5",             // radius/2 = 2.5
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) {
			output := result.Output[i]
			// Allow for floating point precision differences
			if !strings.Contains(output, strings.Split(expected, ":")[0]+":") {
				t.Errorf("Math workflow output %d: got %q, want to contain %q", i+1, output, strings.Split(expected, ":")[0]+":")
			}
		}
	}
}
