package scripting

import (
	"testing"
)

// TestAdvancedGosub_BasicSubroutine_RealIntegration tests basic GOSUB/RETURN with real VM
func TestAdvancedGosub_BasicSubroutine_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
# Test basic subroutine call
echo "Starting main program"
setVar $counter 0
gosub SUBROUTINE
echo "Back in main, counter = " $counter
echo "Program finished"
goto END

:SUBROUTINE
echo "In subroutine"
add $counter 1
echo "Subroutine counter = " $counter
return

:END
echo "Done"
`

	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)

	expectedOutputs := []string{
		"Starting main program",
		"In subroutine",
		"Subroutine counter = 1",
		"Back in main, counter = 1",
		"Program finished",
		"Done",
	}
	tester.AssertOutput(result, expectedOutputs)
}

// TestAdvancedGosub_NestedSubroutines_RealIntegration tests deeply nested GOSUB calls with real stack
func TestAdvancedGosub_NestedSubroutines_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
# Test nested subroutine calls
echo "Main program start"
setVar $depth 0
gosub LEVEL1
echo "Back in main"
goto END

:LEVEL1
echo "In LEVEL1"
add $depth 1
echo "Depth = " $depth
gosub LEVEL2
echo "Back in LEVEL1, depth = " $depth
return

:LEVEL2
echo "In LEVEL2"
add $depth 1
echo "Depth = " $depth
gosub LEVEL3
echo "Back in LEVEL2, depth = " $depth
return

:LEVEL3
echo "In LEVEL3"
add $depth 1
echo "Depth = " $depth
echo "At deepest level"
return

:END
echo "Final depth = " $depth
`

	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)

	expectedOutputs := []string{
		"Main program start",
		"In LEVEL1",
		"Depth = 1",
		"In LEVEL2",
		"Depth = 2",
		"In LEVEL3",
		"Depth = 3",
		"At deepest level",
		"Back in LEVEL2, depth = 3",
		"Back in LEVEL1, depth = 3",
		"Back in main",
		"Final depth = 3",
	}
	tester.AssertOutput(result, expectedOutputs)
}

// TestAdvancedGosub_MultipleCallsSameSubroutine_RealIntegration tests multiple calls to same subroutine
func TestAdvancedGosub_MultipleCallsSameSubroutine_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
# Test multiple calls to same subroutine
echo "Testing multiple calls"
setVar $total 0

echo "First call:"
gosub ADDER
echo "Total after first call: " $total

echo "Second call:"
gosub ADDER
echo "Total after second call: " $total

echo "Third call:"
gosub ADDER
echo "Final total: " $total
goto END

:ADDER
add $total 10
echo "Added 10, current total: " $total
return

:END
echo "Done with multiple calls"
`

	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)

	expectedOutputs := []string{
		"Testing multiple calls",
		"First call:",
		"Added 10, current total: 10",
		"Total after first call: 10",
		"Second call:",
		"Added 10, current total: 20",
		"Total after second call: 20",
		"Third call:",
		"Added 10, current total: 30",
		"Final total: 30",
		"Done with multiple calls",
	}
	tester.AssertOutput(result, expectedOutputs)
}

// TestAdvancedGosub_ParameterPassing_RealIntegration tests parameter passing to subroutines
func TestAdvancedGosub_ParameterPassing_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
# Test parameter passing via variables
echo "Testing parameter passing"

# Pass parameters via global variables
setVar $param1 15
setVar $param2 25
gosub MULTIPLY
echo "Result: " $result

# Try different parameters
setVar $param1 7
setVar $param2 8
gosub MULTIPLY
echo "Result: " $result

goto END

:MULTIPLY
setVar $result $param1
multiply $result $param2
echo "Multiplying " $param1 " by " $param2 " = " $result
return

:END
echo "Parameter passing complete"
`

	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)

	expectedOutputs := []string{
		"Testing parameter passing",
		"Multiplying 15 by 25 = 375",
		"Result: 375",
		"Multiplying 7 by 8 = 56",
		"Result: 56",
		"Parameter passing complete",
	}
	tester.AssertOutput(result, expectedOutputs)
}

// TestAdvancedGosub_CrossInstancePersistence_RealIntegration tests GOSUB across VM instances
func TestAdvancedGosub_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First VM instance - sets up variables and saves state
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
# Set up subroutine state
setVar $counter 0
gosub COUNTER_SUB
saveVar $counter
echo "Final counter: " $counter
halt

:COUNTER_SUB
add $counter 5
echo "Counter in subroutine: " $counter
return
`

	result1 := tester1.ExecuteScript(script1)
	tester1.AssertNoError(result1)

	// Second VM instance - loads state and continues
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
# Load previous state and continue
loadVar $counter
echo "Loaded counter: " $counter
gosub COUNTER_SUB
echo "Final counter: " $counter
halt

:COUNTER_SUB
add $counter 3
echo "Counter in subroutine: " $counter
return
`

	result2 := tester2.ExecuteScript(script2)
	tester2.AssertNoError(result2)

	// Verify persistence worked
	tester2.AssertOutputContains(result2, "Loaded counter: 5")
	tester2.AssertOutputContains(result2, "Counter in subroutine: 8")
	tester2.AssertOutputContains(result2, "Final counter: 8")
}
