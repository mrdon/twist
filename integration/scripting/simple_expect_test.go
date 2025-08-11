package scripting

import (
	"strings"
	"testing"
)

// TestSimpleExpectBasic tests basic expect functionality
func TestSimpleExpectBasic(t *testing.T) {
	// Test expect engine directly with simulated output
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")
	
	// Simulate script output
	expectEngine.AddOutput("Starting port trading\r\n")
	expectEngine.AddOutput("User selected sector: 2157\r\n")
	expectEngine.AddOutput("User selected times: 3\r\n")
	expectEngine.AddOutput("Trading sector 2157 for 3 times\r\n")
	expectEngine.AddOutput("Trade completed\r\n")

	expectScript := `
timeout "1s"
expect "Starting port trading"
expect "User selected sector: 2157"
expect "Trading sector 2157 for 3 times"
assert "Starting port trading"
assert "Trade completed"
log "Basic expect test passed"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Simple expect test failed: %v", err)
	}
}

// TestSimpleExpectMultipleAsserts tests multiple assertions on script output
func TestSimpleExpectMultipleAsserts(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")
	
	// Add all script output
	expectEngine.AddOutput("User: TestPlayer\r\n")
	expectEngine.AddOutput("Credits: 50000\r\n")
	expectEngine.AddOutput("Sector: 42\r\n")
	expectEngine.AddOutput("Ship: Destroyer\r\n")
	expectEngine.AddOutput("Status: Active\r\n")

	expectScript := `
timeout "1s"
expect "User: TestPlayer"
expect "Credits: 50000"  
expect "Sector: 42"
expect "Ship: Destroyer"
assert "User: TestPlayer"
assert "Credits: 50000"
assert "Sector: 42" 
assert "Ship: Destroyer"
assert "Status: Active"
log "All assertions passed"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Multiple assert test failed: %v", err)
	}
}

// TestSimpleExpectTimeout demonstrates timeout behavior
func TestSimpleExpectTimeout(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")
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
	expectEngine := NewSimpleExpectEngine(t, nil, "\r")
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

// TestSimpleExpectSend tests the send functionality
func TestSimpleExpectSend(t *testing.T) {
	var sentInputs []string
	
	inputSender := func(input string) {
		sentInputs = append(sentInputs, input)
	}
	
	expectEngine := NewSimpleExpectEngine(t, inputSender, "\r")
	
	expectScript := `
send "hello world*"
send "test input*"
log "Send test completed"
`

	err := expectEngine.Run(expectScript)
	if err != nil {
		t.Fatalf("Send test failed: %v", err)
	}
	
	expectedInputs := []string{"hello world\r", "test input\r"}
	if !slicesEqual(sentInputs, expectedInputs) {
		t.Errorf("Expected inputs %v, got %v", expectedInputs, sentInputs)
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