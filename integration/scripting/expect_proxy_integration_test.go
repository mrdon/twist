package scripting

import (
	"strings"
	"testing"
	"time"
)

// TestExpectProxyIntegration demonstrates expect engine connected to real proxy via telnet
func TestExpectProxyIntegration(t *testing.T) {
	// Create bridge with real proxy, real database, telnet server
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	// Give the connection time to establish
	time.Sleep(500 * time.Millisecond)

	// Now test dynamic server data sending
	bridge.SendServerData("Welcome to the game!\r\n")
	bridge.SendServerData("Current sector: 1000\r\n")
	bridge.SendServerData("Credits: 50,000\r\n")

	// Now run expect script to test the interaction
	expectScript := `
log "Starting expect-proxy integration test"
timeout "5s"

# First wait for initial server data
expect "Trade Wars 2002"
log "Saw initial game title"

expect "Enter your login name:"
log "Saw login prompt - sending username"
send "testuser"

# Now expect our dynamically sent data
expect "Welcome to the game!"
log "Saw dynamic welcome message"

expect "Current sector: 1000"
log "Saw sector information"

expect "Credits: 50,000"
log "Saw credits information"

log "Integration test completed successfully"
`

	err := bridge.RunExpectScript(expectScript)
	if err != nil {
		t.Fatalf("Expect-proxy integration test failed: %v", err)
	}
}

// TestExpectProxyScriptInteraction demonstrates script input through real proxy
func TestExpectProxyScriptInteraction(t *testing.T) {
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	// Load a TWX script that asks for input
	script := `
echo "Starting script test"
getinput $name "Enter your name: " "Anonymous"
echo "Hello, " $name "!"
`

	bridge.LoadScript(script)

	// Give script time to start
	time.Sleep(200 * time.Millisecond)

	// Use expect engine to interact with the script
	expectScript := `
log "Testing script interaction through real proxy"
timeout "10s"

# Script should output initial message
expect "Starting script test"
log "Script started successfully"

# Script should prompt for input
expect "Enter your name:"
log "Got name prompt - sending name"
send "Alice"

# Script should respond with greeting
expect "Hello, Alice!"
log "Got expected greeting - test passed!"
`

	err := bridge.RunExpectScript(expectScript)
	if err != nil {
		t.Fatalf("Script interaction test failed: %v", err)
	}
}

// TestExpectProxyComplexFlow demonstrates complex multi-step interaction
func TestExpectProxyComplexFlow(t *testing.T) {
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	// Simulate a more complex game flow
	script := `
echo "Port Trading Assistant"
echo "====================="
getinput $sector "Target sector: " 1
getinput $product "Product (1=Ore, 2=Org, 3=Equ): " 1
getinput $quantity "Quantity to trade: " 100

echo ""
echo "Trade Configuration:"
echo "Sector: " $sector
echo "Product: " $product  
echo "Quantity: " $quantity
echo ""
echo "Starting trade sequence..."
`

	bridge.LoadScript(script)
	time.Sleep(300 * time.Millisecond)

	expectScript := `
log "Testing complex trading flow"
timeout "15s"

# Initial banner
expect "Port Trading Assistant"
expect "====================="

# First input - sector
expect "Target sector:"
send "2157"

# Second input - product  
expect "Product \\(1=Ore, 2=Org, 3=Equ\\):"
send "2"

# Third input - quantity
expect "Quantity to trade:"
send "500"

# Verify final output
expect "Trade Configuration:"
expect "Sector: 2157"
expect "Product: 2"
expect "Quantity: 500"
expect "Starting trade sequence..."

log "Complex trading flow test passed!"
`

	err := bridge.RunExpectScript(expectScript)
	if err != nil {
		t.Fatalf("Complex flow test failed: %v", err)
	}
}

// TestExpectProxyGameServerData demonstrates server data simulation
func TestExpectProxyGameServerData(t *testing.T) {
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	time.Sleep(300 * time.Millisecond)

	// Simulate game server sending various data
	expectScript := `
log "Testing game server data simulation"
timeout "10s"

# We'll send server data and expect to see it processed by proxy
log "Test started - expecting server data"
`

	// Start expect script in background
	go func() {
		err := bridge.RunExpectScript(expectScript)
		if err != nil {
			t.Errorf("Background expect script failed: %v", err)
		}
	}()

	// Send various server data patterns
	bridge.SendServerData("Current Sector: 1000\r\n")
	bridge.SendServerData("Credits: 50,000\r\n") 
	bridge.SendServerData("Ship: Imperial StarShip\r\n")
	bridge.SendServerData("\r\nCommand [TL=10:30:45]:[1000] (?=Help)? :")

	// Now run expect to verify the data was processed
	verifyScript := `
timeout "5s"
expect "Current Sector: 1000"
expect "Credits: 50,000"
expect "Ship: Imperial StarShip"
expect "Command.*1000.*Help"
log "Server data simulation test passed!"
`

	err := bridge.RunExpectScript(verifyScript)
	if err != nil {
		t.Fatalf("Server data simulation test failed: %v", err)
	}
}

// TestExpectProxyErrorHandling demonstrates error condition testing
func TestExpectProxyErrorHandling(t *testing.T) {
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer().
		SetupProxy().
		SetupExpectEngine()

	time.Sleep(200 * time.Millisecond)

	// Test timeout behavior
	expectScript := `
log "Testing timeout behavior"
timeout "500ms"

# This should timeout since we won't send this data
expect "This text will never appear"
log "Should not reach this point"
`

	err := bridge.RunExpectScript(expectScript)
	if err == nil {
		t.Fatal("Expected timeout error but test passed")
	}

	// Verify it was a timeout error
	if !expectContains(err.Error(), "timeout waiting") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// expectContains checks if string contains substring
func expectContains(s, substr string) bool {
	return strings.Contains(s, substr)
}