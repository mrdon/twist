package scripting

import (
	"strings"
	"testing"
	"time"
)

// TestSimpleExpectBasic demonstrates the minimal expect approach
func TestSimpleExpectBasic(t *testing.T) {
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	// Create a simple script
	script := `
echo "Starting port trading"
getinput $sector "Enter sector: " 1
getinput $times "Enter times: " 5  
echo "Trading sector " $sector " for " $times " times"
`

	scriptPath := framework.CreateScript("simple_test.ts", script)
	framework.LoadScript(scriptPath)

	// Create simple expect engine that feeds from framework output
	var capturedInputs []string
	expectEngine := NewSimpleExpectEngine(t, func(input string) {
		capturedInputs = append(capturedInputs, input)
		framework.TypeInput(input)
	})

	// Hook up output capture
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

	// Ultra-simple expect script - just the essentials!
	expectScript := `
# Simple deterministic test
timeout "3s"
expect "Starting port trading"
expect "Enter sector:"
send "2157"  
expect "Enter times:"
send "3"
expect "Trading sector 2157 for 3 times"
assert "Starting port trading"
assert "Trading sector 2157 for 3 times"
log "Test completed successfully"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Simple expect test failed: %v", err)
	}

	// Verify inputs were sent correctly
	expectedInputs := []string{"2157", "3"}
	if !slicesEqual(capturedInputs, expectedInputs) {
		t.Errorf("Expected inputs %v, got %v", expectedInputs, capturedInputs)
	}
}

// TestSimpleExpectMultipleAsserts shows multiple assertions
func TestSimpleExpectMultipleAsserts(t *testing.T) {
	framework := NewBlackBoxTestFramework(t).
		SetupDatabase().
		SetupProxy()

	script := `
echo "User: TestPlayer"
echo "Credits: 50000"
echo "Sector: 42"
echo "Ship: Destroyer"
`

	scriptPath := framework.CreateScript("assert_test.ts", script)
	framework.LoadScript(scriptPath)

	expectEngine := NewSimpleExpectEngine(t, func(input string) {
		framework.TypeInput(input)
	})

	// Hook output
	go func() {
		lastLen := 0
		for i := 0; i < 30; i++ {
			output := framework.GetUserOutput()
			if len(output) > lastLen {
				expectEngine.AddOutput(output[lastLen:])
				lastLen = len(output)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	expectScript := `
timeout "2s"
expect "User: TestPlayer"
expect "Credits: 50000"  
expect "Sector: 42"
expect "Ship: Destroyer"
assert "User: TestPlayer"
assert "Credits: 50000"
assert "Sector: 42"
assert "Ship: Destroyer"
log "All assertions passed"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Multiple assert test failed: %v", err)
	}
}

// TestSimpleExpectTimeout demonstrates timeout behavior
func TestSimpleExpectTimeout(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil)
	expectEngine.AddOutput("Some output")

	expectScript := `
timeout "100ms"
expect "This will never appear"
log "Should not reach here"
`

	err := expectEngine.Run(expectScript)
	if err == nil {
		t.Fatal("Expected timeout error but test passed")
	}

	if !strings.Contains(err.Error(), "timeout waiting") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// TestSimpleExpectRegex demonstrates regex patterns
func TestSimpleExpectRegex(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil)
	expectEngine.AddOutput("Player Level: 25")
	expectEngine.AddOutput("Credits: 1,234,567")
	expectEngine.AddOutput("Ship Class: Imperial Starship")

	expectScript := `
timeout "1s"
expect "Player Level: [0-9]+"
expect "Credits: [0-9,]+"
expect "Ship Class: [A-Za-z ]+"
assert "Level: 25"
assert "Credits.*567"
log "Regex patterns matched"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Regex test failed: %v", err)
	}
}

// slicesEqual compares two string slices
func slicesEqual(a, b []string) bool {
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