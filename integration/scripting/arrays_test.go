//go:build integration

package scripting

import (
	"fmt"
	"testing"
)

func TestArrayCreation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	tests := []struct {
		name string
		size int
	}{
		{"Small array", 5},
		{"Medium array", 50},
		{"Large array", 1000},
		{"Empty array", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(`
				array $test_array %d
				arraysize $test_array $size
				echo "Created array size: " $size
			`, tt.size)
			
			result := tester.ExecuteScript(script)
			if result.Error != nil {
				t.Fatalf("Array creation failed: %v", result.Error)
			}

			expectedOutput := fmt.Sprintf("Created array size: %d", tt.size)
			if len(result.Output) != 1 || result.Output[0] != expectedOutput {
				t.Errorf("Expected %s, got %v", expectedOutput, result.Output)
			}
		})
	}
}

func TestSetGetArrayElement_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	tests := []struct {
		name  string
		index int
		value string
	}{
		{"First element", 0, "first"},
		{"Middle element", 5, "middle"},
		{"Last element", 9, "last"},
		{"Numeric value", 3, "42"},
		{"Empty string", 7, ""},
		{"Special characters", 2, "hello@world!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(`
				array $test_array 10
				setarrayelement $test_array %d "%s"
				getarrayelement $test_array %d $result
				echo "Element %d: [" $result "]"
			`, tt.index, tt.value, tt.index, tt.index)
			
			result := tester.ExecuteScript(script)
			if result.Error != nil {
				t.Fatalf("Set/Get array element failed: %v", result.Error)
			}

			expectedOutput := fmt.Sprintf("Element %d: [%s]", tt.index, tt.value)
			if len(result.Output) != 1 || result.Output[0] != expectedOutput {
				t.Errorf("Expected %s, got %v", expectedOutput, result.Output)
			}
		})
	}
}

func TestArraySize_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	sizes := []int{1, 10, 100, 500}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			script := fmt.Sprintf(`
				array $test_array_%d %d
				arraysize $test_array_%d $size_result
				echo "Array size: " $size_result
			`, size, size, size)
			
			result := tester.ExecuteScript(script)
			if result.Error != nil {
				t.Fatalf("Array size command failed: %v", result.Error)
			}

			expectedOutput := fmt.Sprintf("Array size: %d", size)
			if len(result.Output) != 1 || result.Output[0] != expectedOutput {
				t.Errorf("Expected %s, got %v", expectedOutput, result.Output)
			}
		})
	}
}

func TestClearArray_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		array $test_array 5
		setarrayelement $test_array 0 "first"
		setarrayelement $test_array 1 "second"
		setarrayelement $test_array 2 "third"
		setarrayelement $test_array 3 "fourth"
		setarrayelement $test_array 4 "fifth"
		
		getarrayelement $test_array 2 $before_clear
		echo "Before clear: " $before_clear
		
		cleararray $test_array
		
		getarrayelement $test_array 2 $after_clear
		echo "After clear: [" $after_clear "]"
		
		arraysize $test_array $size_after_clear
		echo "Size after clear: " $size_after_clear
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Clear array test failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Before clear: third",
		"After clear: []",
		"Size after clear: 5",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Clear array output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

func TestArrayIndexErrors_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Create array of size 5 first
	setupScript := `array $test_array 5`
	setupResult := tester.ExecuteScript(setupScript)
	if setupResult.Error != nil {
		t.Fatalf("Array setup failed: %v", setupResult.Error)
	}

	errorTests := []struct {
		name   string
		script string
	}{
		{
			name:   "Set negative index",
			script: `setarrayelement $test_array -1 "invalid"`,
		},
		{
			name:   "Set index too large",
			script: `setarrayelement $test_array 10 "invalid"`,
		},
		{
			name:   "Get negative index",
			script: `getarrayelement $test_array -1 $result`,
		},
		{
			name:   "Get index too large",
			script: `getarrayelement $test_array 10 $result`,
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

func TestArrayTypeErrors_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Create a regular string variable (not an array)
	setupScript := `$not_array := "just a string"`
	setupResult := tester.ExecuteScript(setupScript)
	if setupResult.Error != nil {
		t.Fatalf("Variable setup failed: %v", setupResult.Error)
	}

	errorTests := []struct {
		name   string
		script string
	}{
		{
			name:   "Set element on non-array",
			script: `setarrayelement $not_array 0 "value"`,
		},
		{
			name:   "Get element from non-array",
			script: `getarrayelement $not_array 0 $result`,
		},
		{
			name:   "Get size of non-array",
			script: `arraysize $not_array $size`,
		},
		{
			name:   "Clear non-array",
			script: `cleararray $not_array`,
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

func TestArrayPersistence_CrossInstance_RealIntegration(t *testing.T) {
	// Test that arrays persist across VM instances
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		array $persistent_array 3
		setarrayelement $persistent_array 0 "alpha"
		setarrayelement $persistent_array 1 "beta"
		setarrayelement $persistent_array 2 "gamma"
		savevar $persistent_array
		echo "Array saved"
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Fatalf("Script execution failed: %v", result1.Error)
	}

	if len(result1.Output) != 1 || result1.Output[0] != "Array saved" {
		t.Errorf("Expected 'Array saved', got %v", result1.Output)
	}

	// Create new VM instance sharing same database
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadvar $persistent_array
		getarrayelement $persistent_array 0 $elem0
		getarrayelement $persistent_array 1 $elem1
		getarrayelement $persistent_array 2 $elem2
		arraysize $persistent_array $size
		echo "Loaded elements: " $elem0 " " $elem1 " " $elem2
		echo "Array size: " $size
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Fatalf("Script execution in second VM failed: %v", result2.Error)
	}

	expectedOutputs := []string{
		"Loaded elements: alpha beta gamma",
		"Array size: 3",
	}

	if len(result2.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result2.Output), result2.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result2.Output) && result2.Output[i] != expected {
			t.Errorf("Cross-instance output %d: got %q, want %q", i+1, result2.Output[i], expected)
		}
	}
}

func TestComplexArrayWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Complex workflow: create array, populate it, process it, and clear it
	script := `
		# Create array for storing squared numbers
		array $squares 10
		
		# Populate array with squares of indices
		$i := 0
		:populate_loop
		multiply $i $i $square
		setarrayelement $squares $i $square
		add $i 1 $i
		subtract 10 $i $continue
		branch $continue populate_done
		goto populate_loop
		:populate_done
		
		# Calculate sum of all squares
		$sum := 0
		$j := 0
		:sum_loop
		getarrayelement $squares $j $current_square
		add $sum $current_square $sum
		add $j 1 $j
		subtract 10 $j $continue2
		branch $continue2 sum_done
		goto sum_loop
		:sum_done
		
		# Get array size for verification
		arraysize $squares $final_size
		
		# Clear array and verify first element is empty
		cleararray $squares
		getarrayelement $squares 0 $first_after_clear
		
		echo "Sum of squares 0-9: " $sum
		echo "Array size: " $final_size
		echo "First after clear: [" $first_after_clear "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Complex array workflow failed: %v", result.Error)
	}

	// Sum of squares 0²+1²+2²+...+9² = 0+1+4+9+16+25+36+49+64+81 = 285
	expectedOutputs := []string{
		"Sum of squares 0-9: 285",
		"Array size: 10",
		"First after clear: []",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Complex workflow output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

func TestArrayWithOtherCommands_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test arrays in combination with other TWX commands
	script := `
		# Create array for names
		array $names 3
		setarrayelement $names 0 "Alice"
		setarrayelement $names 1 "Bob"
		setarrayelement $names 2 "Charlie"
		
		# Use array elements in text commands
		getarrayelement $names 0 $first_name
		echo "First name: " $first_name
		
		# Use array elements in comparisons
		getarrayelement $names 1 $second_name
		isequal $second_name "Bob" $is_bob
		echo "Is Bob: " $is_bob
		
		# Use array size in math
		arraysize $names $count
		multiply $count 10 $count_times_10
		echo "Count times 10: " $count_times_10
		
		# Store result back in array
		setarrayelement $names 0 $count_times_10
		getarrayelement $names 0 $modified_first
		echo "Modified first: " $modified_first
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Array integration workflow failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"First name: Alice",
		"Is Bob: 1",
		"Count times 10: 30",
		"Modified first: 30",
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Integration output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

func TestArrayBoundaryConditions_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test boundary conditions
		array $test_array 3
		
		# Set elements at boundaries
		setarrayelement $test_array 0 "first"
		setarrayelement $test_array 2 "last"
		
		# Get elements at boundaries
		getarrayelement $test_array 0 $first
		getarrayelement $test_array 1 $middle
		getarrayelement $test_array 2 $last
		
		echo "First: [" $first "]"
		echo "Middle: [" $middle "]"
		echo "Last: [" $last "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Boundary conditions test failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"First: [first]",
		"Middle: []",    // Should be empty by default
		"Last: [last]",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Boundary condition output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

func TestArrayInitialization_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test that array elements are properly initialized to empty strings
	script := `
		array $init_test 5
		getarrayelement $init_test 0 $elem0
		getarrayelement $init_test 2 $elem2
		getarrayelement $init_test 4 $elem4
		echo "Element 0: [" $elem0 "]"
		echo "Element 2: [" $elem2 "]"  
		echo "Element 4: [" $elem4 "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Array initialization test failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Element 0: []",
		"Element 2: []",
		"Element 4: []",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Initialization output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}