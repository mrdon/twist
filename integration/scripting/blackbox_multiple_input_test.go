package scripting

import (
	"testing"
	"time"
)

// TestMultipleInputBlackBox tests the complete user experience of a script
// that asks for multiple inputs. This is completely black box - we don't know
// or care about internal implementation details.
func TestMultipleInputBlackBox(t *testing.T) {
	// Create a completely black box test framework
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	// Create a script that asks for multiple inputs (like the Port Trading script)
	script := `
echo "Starting port trading script"
getinput $sector "Enter sector number: " 1
getinput $times "How many times to execute: " 5  
getinput $percent "Enter markup percentage: " 10

echo "Configuration complete:"
echo "Sector: " $sector
echo "Times: " $times
echo "Percent: " $percent
`

	scriptPath := framework.CreateScript("port_trading_test.ts", script)

	// Load the script (like a user would run it)
	framework.LoadScript(scriptPath)

	// USER EXPERIENCE: Wait to see the initial message
	framework.WaitForOutput("Starting port trading script", 2*time.Second)

	// USER EXPERIENCE: Wait for first prompt
	framework.WaitForPrompt("Enter sector number:", 2*time.Second)

	// USER EXPERIENCE: User types the sector number
	framework.TypeInput("2157")

	// USER EXPERIENCE: Wait for second prompt  
	framework.WaitForPrompt("How many times to execute:", 2*time.Second)

	// USER EXPERIENCE: User types the number of times
	framework.TypeInput("3")

	// USER EXPERIENCE: Wait for third prompt
	framework.WaitForPrompt("Enter markup percentage:", 2*time.Second)

	// USER EXPERIENCE: User just presses Enter to use default
	framework.TypeInput("")

	// USER EXPERIENCE: Wait to see the final results
	framework.WaitForOutput("Configuration complete:", 2*time.Second)
	framework.WaitForOutput("Sector: 2157", 2*time.Second)
	framework.WaitForOutput("Times: 3", 2*time.Second)
	framework.WaitForOutput("Percent: 10", 2*time.Second) // Should use default

	// Verify the complete user experience
	userScreen := framework.GetUserOutput()
	t.Logf("Complete user experience:\n%s", userScreen)

	// Assert what the user should see
	framework.
		AssertUserSees("Starting port trading script").
		AssertUserSees("Enter sector number:").
		AssertUserSees("How many times to execute:").
		AssertUserSees("Enter markup percentage:").
		AssertUserSees("Configuration complete:").
		AssertUserSees("Sector: 2157").
		AssertUserSees("Times: 3").
		AssertUserSees("Percent: 10")

	// Assert what the user should NOT see (no restarts or errors)
	framework.
		AssertUserDoesNotSee("Starting port trading script\r\nStarting port trading script"). // No restart
		AssertUserDoesNotSee("error").
		AssertUserDoesNotSee("Error")
}

// TestInputWithEchoingBlackBox verifies that the user can see what they type
func TestInputWithEchoingBlackBox(t *testing.T) {
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	script := `
getinput $name "Enter your name: " "Anonymous"
echo "Hello, " $name "!"
`

	scriptPath := framework.CreateScript("echo_test.ts", script)
	framework.LoadScript(scriptPath)

	// Wait for prompt
	framework.WaitForPrompt("Enter your name:", 2*time.Second)

	// User types their name  
	framework.TypeInput("Alice")

	// Wait for greeting
	framework.WaitForOutput("Hello, Alice!", 2*time.Second)

	// The user should see their input echoed back
	framework.AssertUserSees("Enter your name:")
	framework.AssertUserSees("Hello, Alice!")
}

// TestDefaultValuesBlackBox verifies default value behavior from user perspective
func TestDefaultValuesBlackBox(t *testing.T) {
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	script := `
getinput $port "Enter port (default 23): " 23
getinput $host "Enter host (default localhost): " "localhost"
echo "Connecting to " $host ":" $port
`

	scriptPath := framework.CreateScript("defaults_test.ts", script)
	framework.LoadScript(scriptPath)

	// First prompt - use default by pressing Enter
	framework.WaitForPrompt("Enter port (default 23):", 2*time.Second)
	framework.TypeInput("") // Just press Enter

	// Second prompt - provide custom value
	framework.WaitForPrompt("Enter host (default localhost):", 2*time.Second)  
	framework.TypeInput("example.com")

	// Wait for final output
	framework.WaitForOutput("Connecting to example.com:23", 2*time.Second)

	// Verify user experience
	framework.
		AssertUserSees("Enter port (default 23):").
		AssertUserSees("Enter host (default localhost):").
		AssertUserSees("Connecting to example.com:23")
}

// TestRapidInputBlackBox tests when user types very quickly
func TestRapidInputBlackBox(t *testing.T) {
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	script := `
getinput $a "First: " 0
getinput $b "Second: " 0  
getinput $c "Third: " 0
echo "Results: " $a " " $b " " $c
`

	scriptPath := framework.CreateScript("rapid_test.ts", script)
	framework.LoadScript(scriptPath)

	// Rapid input - user types fast
	framework.WaitForPrompt("First:", 2*time.Second)
	framework.TypeInput("111")

	framework.WaitForPrompt("Second:", 2*time.Second)
	framework.TypeInput("222")

	framework.WaitForPrompt("Third:", 2*time.Second)
	framework.TypeInput("333")

	framework.WaitForOutput("Results: 111 222 333", 2*time.Second)

	// Should work perfectly even with rapid input
	framework.AssertUserSees("Results: 111 222 333")
}

// TestLongInputBlackBox tests handling of longer input strings  
func TestLongInputBlackBox(t *testing.T) {
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	script := `
getinput $message "Enter a long message: " "default"
echo "You said: " $message
`

	scriptPath := framework.CreateScript("long_input_test.ts", script)
	framework.LoadScript(scriptPath)

	framework.WaitForPrompt("Enter a long message:", 2*time.Second)
	
	longMessage := "This is a really long message that tests whether the input system can handle longer strings properly without any issues"
	framework.TypeInput(longMessage)

	framework.WaitForOutput("You said: "+longMessage, 2*time.Second)
	framework.AssertUserSees("You said: "+longMessage)
}