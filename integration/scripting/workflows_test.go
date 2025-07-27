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
setVar $name "TWX Test"
setVar $number 42
setVar $result ""

echo "Testing basic assignment and concatenation"
echo "Name: " $name
echo "Number: " $number

# Test arrays
SETARRAY $test_array 5
setVar $test_array[1] "First"  
setVar $test_array[2] "Second"
setVar $test_array[3] "Third"

setVar $result $test_array[2]
echo "Array element 1: " $result

# Array size is stored in element 0 in TWX
setVar $size $test_array[0]
echo "Array size: " $size

# Test math operations
setVar $sum 10
add $sum 20
setVar $diff 50
subtract $diff 15
setVar $product 6
multiply $product 7
setVar $quotient 100
divide $quotient 4

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
setVar $target_sector 100
echo "Moving to sector " $target_sector
send "m " $target_sector

# Trade simulation
echo "Checking port status"
send "p"

# Use arrays to track inventory
SETARRAY $inventory 3
setVar $inventory[1] "Fuel Ore"
setVar $inventory[2] "Organics"
setVar $inventory[3] "Equipment"

setVar $item $inventory[1]
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
setVar $base 10
setVar $multiplier 3
setVar $bonus 5

# Calculate compound value
setVar $temp1 $base
multiply $temp1 $multiplier
setVar $final_value $temp1
add $final_value $bonus

echo "Calculation result: " $final_value

# Test loop with calculation
setVar $i 1
setVar $factorial 1
while $i <= 4
  multiply $factorial $i
  echo "Factorial step " $i ": " $factorial
  add $i 1
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
setVar $input "Hello World Test"
setVar $processed ""

# Get each word
getWord $input $word1 1
getWord $input $word2 2  
getWord $input $word3 3

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
SETARRAY $names 3
SETARRAY $scores 3

# Populate arrays
setVar $names[1] "Alice"
setVar $names[2] "Bob"
setVar $names[3] "Charlie"

setVar $scores[1] "95"
setVar $scores[2] "87"
setVar $scores[3] "92"

# Access individual elements
setVar $name $names[1]
setVar $score $scores[1]
echo "Student " $name " scored " $score

setVar $name $names[2]
setVar $score $scores[2]
echo "Student " $name " scored " $score

setVar $name $names[3]
setVar $score $scores[3]
echo "Student " $name " scored " $score

# Calculate average (simplified)
setVar $s1 $scores[1]
setVar $s2 $scores[2]
setVar $s3 $scores[3]
setVar $temp $s1
add $temp $s2
setVar $total $temp
add $total $s3
setVar $average $total
divide $average 3
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
SETARRAY $test 2
setVar $test[5] "Out of bounds"
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