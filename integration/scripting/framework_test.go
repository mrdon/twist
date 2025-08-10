package scripting

import (
	"testing"
)

// TestMultipleGetInputUsingFramework demonstrates the new testing framework
// This test reproduces the bug where scripts stop after the first getinput
func TestMultipleGetInputUsingFramework(t *testing.T) {
	// Create the test framework - this sets up everything we need
	framework := NewScriptTestFramework(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		ConnectToTelnetServer()

	// Configure telnet server responses
	framework.ConfigureTelnetResponses([]string{
		"Command [TL=00:00:00]: ",                                       // Response to login
		"You are currently in sector 1\r\nCommand [TL=00:00:00]: ",    // Response to first input
		"Sector 2 entered\r\nCommand [TL=00:00:00]: ",                 // Response to second input
		"Trading completed\r\nCommand [TL=00:00:00]: ",                // Response to third input
	})

	// Create a script with multiple getinput commands
	scriptContent := `
echo "Starting port trading script"
getinput sector_number "Enter sector number: " "1"
echo "Got sector: " + $sector_number
getinput trade_type "Enter trade type (buy/sell): " "buy" 
echo "Got trade type: " + $trade_type
getinput quantity "Enter quantity: " "100"
echo "Got quantity: " + $quantity
echo "Script completed with all inputs"
`
	
	// Create and load the script
	scriptPath := framework.CreateScript("test_multiple_getinput.ts", scriptContent)
	framework.LoadAndRunScript(scriptPath)

	// Send the user inputs that the script is expecting
	framework.
		SendUserInput("5").        // First input: sector number
		SendUserInput("sell").     // Second input: trade type
		SendUserInput("250").      // Third input: quantity
		WaitForScriptCompletion()

	// Check results - this will show if the bug exists
	framework.
		AssertMinimumInputCount(3). // We expect at least 3 inputs to be sent to server
		AssertInputsContain([]string{"5", "sell", "250"}) // Each input should contain the expected values
}

// TestSimpleEchoUsingFramework tests basic echo functionality with the framework
func TestSimpleEchoUsingFramework(t *testing.T) {
	// Much simpler setup for basic functionality
	framework := NewScriptTestFramework(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		ConnectToTelnetServer()

	// Simple script that just echoes some text
	scriptContent := `
echo "Hello World"
echo "Script test completed"
`
	
	scriptPath := framework.CreateScript("test_echo.ts", scriptContent)
	framework.
		LoadAndRunScript(scriptPath).
		WaitForScriptCompletion()

	// For echo test, we just verify the script ran without errors
	// The framework automatically handles all the infrastructure
	framework.AssertInputCount(0) // Echo commands shouldn't send inputs to server
}

// TestSingleGetInputUsingFramework tests that single getinput works correctly
func TestSingleGetInputUsingFramework(t *testing.T) {
	framework := NewScriptTestFramework(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		ConnectToTelnetServer()

	framework.ConfigureTelnetResponses([]string{
		"Command [TL=00:00:00]: ", // Response to user input
	})

	scriptContent := `
echo "Testing single getinput"
getinput user_name "Enter your name: " "Anonymous"
echo "Hello " + $user_name
`
	
	scriptPath := framework.CreateScript("test_single_getinput.ts", scriptContent)
	framework.
		LoadAndRunScript(scriptPath).
		SendUserInput("TestUser").
		WaitForScriptCompletion()

	// Single getinput should work fine - this is our baseline
	framework.
		AssertInputCount(1).
		AssertInputsContain([]string{"TestUser"})
}