package scripting

import (
	"strings"
	"testing"
	"time"
)

// TestReal1PortScript_ActualBugReproduction tests the exact pattern from 1_Port.ts
// This should reproduce the actual bug the user is experiencing
func TestReal1PortScript_ActualBugReproduction(t *testing.T) {
	framework := NewScriptTestFramework(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		ConnectToTelnetServer()

	// Configure telnet responses to simulate the game
	framework.ConfigureTelnetResponses([]string{
		"Command [TL=00:00:00]: ",                    // Response after first input  
		"Command [TL=00:00:00]: ",                    // Response after second input
		"Command [TL=00:00:00]: ",                    // Response after third input
		"Sector  : 1 in uncharted space.\r\nCommand [TL=00:00:00]: ", // Response to "d" command
	})

	// Use the EXACT pattern from the real 1_Port.ts script (lines 56-58)
	scriptContent := `
# Simplified version of the 1_Port.ts pattern
echo "**     --===| Port Pair Trading v2.00 |===--**"
echo "No registration is required to use this script,*it is completely open source and can be opened*in notepad."
echo "**For your own safety, please read the warnings*written at the top of the script before*using it!*"

# Test with even simpler pattern - just echo after getinput
getinput $sector2 "Enter sector to trade to" 0
echo "After first getinput"
getinput $timesLeft "Enter times to execute script" 0  
echo "After second getinput"
getinput $percent "Enter markup/markdown percentage" 5
echo "After third getinput - script should continue"
send "d"
echo "After send command"
`
	
	scriptPath := framework.CreateScript("real_1_port_pattern.ts", scriptContent)
	framework.LoadAndRunScript(scriptPath)

	// Test with multi-character inputs like the user is experiencing
	// Simulate how real telnet clients send input character-by-character, then Enter
	// This should reproduce the bug where each getinput gets only one character
	
	// Send "2157" character by character followed by ENTER to simulate real typing
	// Before fix: each character goes to different getinput commands  
	// After fix: characters should be buffered until ENTER, then sent as complete input
	
	// Send character-by-character input like real TUI does for FIRST input only
	// This should reproduce the real bug: first input works, but second getinput doesn't wait
	proxy := framework.proxy
	proxy.SendInput("6")        // Should be buffered
	proxy.SendInput("6")        // Should be buffered  
	proxy.SendInput("6")        // Should be buffered
	proxy.SendInput("0")        // Should be buffered
	proxy.SendInput("\r")       // ENTER - should send "6660" to first getinput
	time.Sleep(200 * time.Millisecond)
	
	// After this, the script should automatically continue to the second getinput
	// BUT the second getinput should NOT wait for user input - it should immediately
	// use the default value [0] and continue, just like in the real app
	
	// No additional input needed - the bug is that second getinput doesn't wait
	
	// Wait longer to see if script continues automatically (using defaults)
	time.Sleep(2000 * time.Millisecond)
	
	framework.WaitForScriptCompletion()
	
	// Wait a bit more to ensure the send command completes
	time.Sleep(200 * time.Millisecond)

	// Get the inputs that were actually sent to the telnet server
	inputs := framework.GetTelnetInputs()
	t.Logf("Real 1_Port pattern test: telnet server received %d inputs", len(inputs))
	t.Logf("Inputs: %v", inputs)

	// The bug: if the script only processes the first getinput, we should only see 2 inputs:
	// 1. The telnet negotiation + "6660" 
	// 2. The "d" command from send "d"
	// 
	// If the script is working correctly, we should see 4 inputs:
	// 1. Telnet negotiation + "6660"
	// 2. "5" 
	// 3. "10"
	// 4. "d"
	
	// Test should reproduce the REAL bug: first input works but second getinput doesn't wait
	// Expected behavior with the bug:
	// 1. First input: "6660" sent correctly (input buffering works)
	// 2. Second input: automatically uses default [0] without waiting for user input
	// 3. Third input: automatically uses default [5] without waiting for user input  
	// 4. Script continues with send "d" command
	
	// We should see: telnet negotiation + "6660", then "d" command (no second or third inputs)
	expectedFirstInput := "6660"
	expectedSendCommand := "d"
	foundFirstInput := false
	foundSendCommand := false
	foundExtraInputs := 0
	
	for _, actualInput := range inputs {
		cleanInput := strings.TrimSpace(strings.Trim(actualInput, "\xff\xfb\x18\xff\xfb\x1f\xff\xfd\x01\xff\xfb\x03\xff\xfd\x03"))
		
		if cleanInput == expectedFirstInput {
			foundFirstInput = true
		} else if cleanInput == expectedSendCommand {
			foundSendCommand = true
		} else if cleanInput != "" {
			// Any other non-empty input indicates the script waited for more input (which would be correct behavior)
			foundExtraInputs++
		}
	}
	
	if foundFirstInput && foundSendCommand && foundExtraInputs == 0 {
		t.Errorf("BUG REPRODUCED: Second and third getinput commands did not wait for user input")
		t.Errorf("Expected: Script should wait for user input on all three getinput commands")
		t.Errorf("Actual: Only first getinput ('%s') waited, then script used defaults and sent '%s'", expectedFirstInput, expectedSendCommand)
		t.Errorf("This matches the real app bug where subsequent getinput commands don't pause for input")
		t.Errorf("All inputs received: %v", inputs)
	} else if foundExtraInputs > 0 {
		t.Logf("SUCCESS: Script correctly waited for multiple inputs")
		t.Logf("Found first input '%s', send command '%s', and %d additional inputs", expectedFirstInput, expectedSendCommand, foundExtraInputs)
		t.Logf("All getinput commands properly paused for user input")
	} else {
		t.Logf("UNEXPECTED: Found first input: %v, send command: %v, extra inputs: %d", foundFirstInput, foundSendCommand, foundExtraInputs)
		t.Logf("All inputs received: %v", inputs)
	}
}