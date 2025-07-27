//go:build integration

package scripting

import (
	"testing"
)

// TestComprehensiveWorkflow_RealIntegration tests comprehensive TWX script combining multiple features
func TestComprehensiveWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Comprehensive TWX Script Test
# Testing core functionality

# Test variables and basic assignment
$name := "TWX Test"
$number := 42
$result := ""

echo "Testing basic assignment and concatenation"
echo "Name: " $name
echo "Number: " $number

# Test arrays
array $test_array 5
setarrayelement $test_array 0 "First"
setarrayelement $test_array 1 "Second"
setarrayelement $test_array 2 "Third"

getarrayelement $test_array 1 $result
echo "Array element 1: " $result

arraysize $test_array $size
echo "Array size: " $size

# Test math operations
add 10 20 $sum
subtract 50 15 $diff
multiply 6 7 $product
divide 100 4 $quotient

echo "Math results:"
echo "10 + 20 = " $sum
echo "50 - 15 = " $diff
echo "6 * 7 = " $product
echo "100 / 4 = " $quotient

echo "Comprehensive test completed!"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	expectedOutputs := []string{
		"Testing basic assignment and concatenation",
		"Name: TWX Test",
		"Number: 42",
		"Array element 1: Second",
		"Array size: 5",
		"Math results:",
		"10 + 20 = 30",
		"50 - 15 = 35",
		"6 * 7 = 42",
		"100 / 4 = 25",
		"Comprehensive test completed!",
	}
	
	tester.AssertOutput(result, expectedOutputs)
}

// TestGameScriptSimulation_RealIntegration tests a realistic TWX game automation scenario
func TestGameScriptSimulation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Simulate a simple TWX game interaction script
echo "Starting game automation..."

# Check current location
send "look"

# Move to a specific sector
$target_sector := 100
echo "Moving to sector " $target_sector
send "m " $target_sector

# Trade simulation
echo "Checking port status"
send "p"

# Use arrays to track inventory
array $inventory 3
setarrayelement $inventory 0 "Fuel Ore"
setarrayelement $inventory 1 "Organics" 
setarrayelement $inventory 2 "Equipment"

getarrayelement $inventory 0 $item
echo "First inventory item: " $item

echo "Script completed successfully"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	// Verify game commands were sent
	expectedCommands := []string{"look", "m 100", "p"}
	tester.AssertCommands(result, expectedCommands)
	
	// Verify output
	tester.AssertOutputContains(result, "Starting game automation...")
	tester.AssertOutputContains(result, "Moving to sector 100")
	tester.AssertOutputContains(result, "First inventory item: Fuel Ore")
	tester.AssertOutputContains(result, "Script completed successfully")
}

// TestMathAndLogicWorkflow_RealIntegration tests complex mathematical operations with conditional logic
func TestMathAndLogicWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Test complex mathematical operations and logic
$base := 10
$multiplier := 3
$bonus := 5

# Calculate compound value
multiply $base $multiplier $temp1
add $temp1 $bonus $final_value

echo "Calculation result: " $final_value

# Test loop with calculation
$i := 1
$factorial := 1
while $i <= 4
  multiply $factorial $i $factorial
  echo "Factorial step " $i ": " $factorial
  add $i 1 $i
end

echo "Final factorial: " $factorial
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	expectedOutputs := []string{
		"Calculation result: 35",
		"Factorial step 1: 1",
		"Factorial step 2: 2", 
		"Factorial step 3: 6",
		"Factorial step 4: 24",
		"Final factorial: 24",
	}
	
	tester.AssertOutput(result, expectedOutputs)
}

// TestStringProcessingWorkflow_RealIntegration tests complex string processing workflows
func TestStringProcessingWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Test complex string processing
$input := "Hello World Test"
$processed := ""

# Get each word
getword $input $word1 1
getword $input $word2 2  
getword $input $word3 3

echo "Words: " $word1 " | " $word2 " | " $word3

# Process strings
upper $word1 $upper_word1
lower $word2 $lower_word2
len $word3 $word3_len

echo "Processed: " $upper_word1 " " $lower_word2 " (len=" $word3_len ")"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	expectedOutputs := []string{
		"Words: Hello | World | Test",
		"Processed: HELLO world (len=4)",
	}
	
	tester.AssertOutput(result, expectedOutputs)
}

// TestArrayWorkflow_RealIntegration tests advanced array operations with loops
func TestArrayWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Test advanced array operations
echo "Creating and populating arrays..."

# Create multiple arrays
array $names 3
array $scores 3

# Populate arrays
setarrayelement $names 0 "Alice"
setarrayelement $names 1 "Bob"
setarrayelement $names 2 "Charlie"

setarrayelement $scores 0 "95"
setarrayelement $scores 1 "87"
setarrayelement $scores 2 "92"

# Process arrays with loop
$index := 0
while $index < 3
  getarrayelement $names $index $name
  getarrayelement $scores $index $score
  echo "Student " $name " scored " $score
  add $index 1 $index
end

# Calculate average (simplified)
getarrayelement $scores 0 $s1
getarrayelement $scores 1 $s2
getarrayelement $scores 2 $s3
add $s1 $s2 $temp
add $temp $s3 $total
divide $total 3 $average
int $average $average

echo "Average score: " $average
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	expectedOutputs := []string{
		"Creating and populating arrays...",
		"Student Alice scored 95",
		"Student Bob scored 87",
		"Student Charlie scored 92",
		"Average score: 91",
	}
	
	tester.AssertOutput(result, expectedOutputs)
}

// TestErrorHandlingWorkflow_RealIntegration tests error handling in workflow scenarios
func TestErrorHandlingWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// Test script with array bounds error
	script := `
array $test 2
setarrayelement $test 5 "Out of bounds"
echo "This should not execute"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertError(result)
	
	// Should not contain the echo output
	for _, output := range result.Output {
		if output == "This should not execute" {
			t.Error("Script continued execution after error")
		}
	}
}