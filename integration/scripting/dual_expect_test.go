package scripting

import (
	"testing"
	"time"
)

// TestDualExpectScripts demonstrates server-side and client-side expect scripts working together
func TestDualExpectScripts(t *testing.T) {
	// Create bridge with expect-enabled server
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer()

	// Set server-side expect script
	serverScript := `
log "Server script starting"
timeout "10s"

# Send initial game welcome
send "Trade Wars 2002 - Enhanced Edition\r\n"
send "Enter your login name: "

# Wait for username
expect "testuser"
log "Server received username: testuser"

# Send welcome and main menu
send "Welcome testuser!\r\n"
send "Current Sector: 1 (Safe Haven)\r\n" 
send "Credits: 50,000\r\n"
send "\r\nCommands: (L)ook (M)ove (T)rade (Q)uit\r\n"
send "Command: "

# Wait for 'L' command (look)
expect "L"
log "Server received Look command"

# Send sector description
send "Sector 1 - Safe Haven\r\n"
send "A peaceful sector protected by Federation forces.\r\n"
send "Traders present: 0\r\n"
send "Ports: StarDock (Class 0)\r\n"
send "\r\nCommand: "

# Wait for 'Q' command (quit)
expect "Q" 
log "Server received Quit command"

# Send goodbye
send "Thanks for playing!\r\n"
send "Connection terminated.\r\n"

log "Server script completed successfully"
`

	bridge.SetServerScript(serverScript).
		SetupProxy().
		SetupExpectEngine()

	// Give connection time to establish and server script to start
	time.Sleep(300 * time.Millisecond)

	// Client-side expect script
	clientScript := `
log "Client script starting"
timeout "15s"

# Expect game title and respond with username
expect "Trade Wars 2002 - Enhanced Edition"
log "Client saw game title"

expect "Enter your login name:"
log "Client saw login prompt"
send "testuser"

# Expect welcome messages
expect "Welcome testuser!"
log "Client saw welcome message"

expect "Current Sector: 1"
log "Client saw sector info"

expect "Credits: 50,000"
log "Client saw credits"

expect "Commands:"
log "Client saw command menu"

expect "Command:"
log "Client saw command prompt - sending Look command"
send "L"

# Expect look response
expect "Sector 1 - Safe Haven"
log "Client saw sector description"

expect "A peaceful sector protected by Federation forces"
log "Client saw sector details"

expect "Traders present: 0"
log "Client saw trader count"

expect "Ports: StarDock"
log "Client saw port info"

expect "Command:"
log "Client saw second command prompt - sending Quit"
send "Q"

# Expect goodbye
expect "Thanks for playing!"
log "Client saw goodbye message"

expect "Connection terminated"
log "Client saw connection terminated"

log "Client script completed successfully"
`

	// Run client script
	err := bridge.RunExpectScript(clientScript)
	if err != nil {
		t.Fatalf("Client expect script failed: %v", err)
	}

	// Wait for server script to complete
	err = bridge.WaitForServerScript(5 * time.Second)
	if err != nil {
		t.Fatalf("Server script failed: %v", err)
	}

	t.Log("Dual expect scripts completed successfully!")
}

// TestDualExpectPortTrading demonstrates a complex port trading scenario
func TestDualExpectPortTrading(t *testing.T) {
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer()

	// Server script simulating a port trading session
	serverScript := `
log "Port trading server script starting"
timeout "20s"

# Initial connection
send "Trade Wars 2002\r\nLogin: "
expect "trader123"
send "Welcome trader123!\r\nSector 1000\r\nCommand: "

# Player wants to move to a port sector
expect "M 2000"
log "Server: Player moving to sector 2000"
send "Moving to sector 2000...\r\n"
send "Arrived at sector 2000\r\n" 
send "Port McKenzie - Class 5 Trading Post\r\n"
send "Products: Fuel Ore (500 units @ 15 credits)\r\n"
send "          Organics (300 units @ 25 credits)\r\n"
send "Command: "

# Player wants to trade
expect "T"
log "Server: Player initiated trade"
send "Port Trading Menu:\r\n"
send "(B)uy (S)ell (E)xit\r\n"
send "Choice: "

expect "B"
send "Buy Menu:\r\n"
send "1. Fuel Ore (15 credits/unit) - 500 available\r\n"
send "2. Organics (25 credits/unit) - 300 available\r\n"
send "Product: "

expect "1"
send "How many Fuel Ore units? (max 500): "

expect "100"
send "Purchasing 100 units of Fuel Ore for 1,500 credits...\r\n"
send "Transaction complete!\r\n"
send "Credits remaining: 48,500\r\n"
send "Command: "

expect "Q"
send "Thanks for trading! Safe travels!\r\n"

log "Port trading server script completed"
`

	bridge.SetServerScript(serverScript).
		SetupProxy().
		SetupExpectEngine()

	time.Sleep(300 * time.Millisecond)

	// Client script for port trading
	clientScript := `
log "Port trading client script starting" 
timeout "25s"

# Login
expect "Trade Wars 2002"
expect "Login:"
send "trader123"

expect "Welcome trader123!"
expect "Sector 1000"
expect "Command:"
log "Logged in successfully - moving to port"
send "M 2000"

# Arrive at port
expect "Moving to sector 2000"
expect "Arrived at sector 2000"
expect "Port McKenzie - Class 5 Trading Post"
expect "Fuel Ore.*500 units.*15 credits"
expect "Organics.*300 units.*25 credits"
log "Arrived at port with products available"

expect "Command:"
send "T"

# Trading menu
expect "Port Trading Menu"
expect "Buy.*Sell.*Exit"
expect "Choice:"
send "B"

expect "Buy Menu"
expect "1. Fuel Ore.*15 credits.*500 available"
expect "2. Organics.*25 credits.*300 available"
expect "Product:"
send "1"

expect "How many Fuel Ore units.*max 500"
send "100"

expect "Purchasing 100 units of Fuel Ore for 1,500 credits"
expect "Transaction complete"
expect "Credits remaining: 48,500"
log "Successfully purchased fuel ore"

expect "Command:"
send "Q"

expect "Thanks for trading! Safe travels!"
log "Port trading completed successfully"
`

	err := bridge.RunExpectScript(clientScript)
	if err != nil {
		t.Fatalf("Port trading client script failed: %v", err)
	}

	err = bridge.WaitForServerScript(5 * time.Second)
	if err != nil {
		t.Fatalf("Port trading server script failed: %v", err)
	}

	t.Log("Dual expect port trading test completed successfully!")
}

// TestDualExpectScriptInput demonstrates script input handling with dual expect
func TestDualExpectScriptInput(t *testing.T) {
	bridge := NewExpectTelnetBridge(t).
		SetupDatabase().
		SetupTelnetServer()

	// Server script that simulates script prompts
	serverScript := `
log "Script input server starting"
timeout "15s"

# Initial connection
send "Trade Wars 2002\r\nConnected.\r\n"

# Simulate a script starting and asking for input
send "Starting port trading script\r\n"
send "Enter sector number: "

expect "2157"
log "Server received sector: 2157"

send "How many times to execute: "

expect "5"
log "Server received times: 5"

send "Enter markup percentage: "

# User presses enter for default
expect ""
log "Server received empty input (using default)"

# Show script results
send "Configuration complete:\r\n"
send "Sector: 2157\r\n"
send "Times: 5\r\n" 
send "Percentage: 10 (default)\r\n"
send "Script execution complete.\r\n"

log "Script input server completed"
`

	bridge.SetServerScript(serverScript).
		SetupProxy()

	// Load a script that matches the server expectations
	script := `
echo "Starting port trading script"
getinput $sector "Enter sector number: " 0
getinput $times "How many times to execute: " 0
getinput $percent "Enter markup percentage: " 10

echo "Configuration complete:"
echo "Sector: " $sector
echo "Times: " $times
echo "Percentage: " $percent
echo "Script execution complete."
`

	bridge.LoadScript(script).SetupExpectEngine()

	time.Sleep(300 * time.Millisecond)

	// Client expect script 
	clientScript := `
log "Script input client starting"
timeout "20s"

expect "Connected"
expect "Starting port trading script"
log "Script started"

expect "Enter sector number:"
send "2157"

expect "How many times to execute:"
send "5"

expect "Enter markup percentage:"
send ""

expect "Configuration complete:"
expect "Sector: 2157"
expect "Times: 5" 
expect "Percentage: 10"
expect "Script execution complete"

log "Script input test completed successfully"
`

	err := bridge.RunExpectScript(clientScript)
	if err != nil {
		t.Fatalf("Script input client failed: %v", err)
	}

	err = bridge.WaitForServerScript(5 * time.Second)
	if err != nil {
		t.Fatalf("Script input server failed: %v", err)
	}

	t.Log("Dual expect script input test completed!")
}