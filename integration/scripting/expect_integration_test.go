package scripting

import (
	"strings"
	"testing"
	"time"
)

// TestExpectWithBlackBox demonstrates expect engine integrated with black box framework
func TestExpectWithBlackBox(t *testing.T) {
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	// Create a script that asks for multiple inputs
	script := `
echo "Starting port trading script"
getinput $sector "Enter sector number: " 1
getinput $times "How many times: " 5  
echo "Configuration: sector=" $sector " times=" $times
`

	scriptPath := framework.CreateScript("port_test.ts", script)
	framework.LoadScript(scriptPath)

	// Create expect engine
	var capturedInputs []string
	expectEngine := NewSimpleExpectEngine(t, func(input string) {
		capturedInputs = append(capturedInputs, input)
		framework.TypeInput(input)
	})

	// Hook up output capture from framework to expect engine
	go func() {
		lastLen := 0
		for i := 0; i < 50; i++ { // Run for 5 seconds max
			output := framework.GetUserOutput()
			if len(output) > lastLen {
				newOutput := output[lastLen:]
				expectEngine.AddOutput(newOutput)
				lastLen = len(output)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Simple expect script
	expectScript := `
log "Starting expect-based integration test"
timeout "10s"
expect "Starting port trading script"
expect "Enter sector number:"
send "2157"
expect "How many times:"
send "3"
expect "Configuration: sector=2157 times=3"
log "Port trading test completed successfully!"
`

	// Execute the expect script
	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Expect script failed: %v", err)
	}

	// Verify the inputs were sent correctly
	expectedInputs := []string{"2157", "3"}
	if !expectSlicesEqual(capturedInputs, expectedInputs) {
		t.Errorf("Expected inputs %v, got %v", expectedInputs, capturedInputs)
	}
}

// TestExpectTimeout demonstrates timeout handling
func TestExpectTimeout(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil)
	expectEngine.AddOutput("Some initial output")

	expectScript := `
timeout "100ms"
expect "This text will never appear"
log "Should not reach this line"
`

	err := expectEngine.Run(expectScript)
	if err == nil {
		t.Fatal("Expected timeout error but test passed")
	}

	if !strings.Contains(err.Error(), "timeout waiting") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// Helper function to avoid conflicts
func expectSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}