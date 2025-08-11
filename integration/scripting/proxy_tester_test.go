package scripting

import (
	"strings"
	"testing"
	"twist/internal/api"
)

// TestDualExpectScripts demonstrates server-side and client-side expect scripts working together
func TestDualExpectScripts(t *testing.T) {
	serverScript := `send "Hello*"
expect "Hi"
send "Bye*"`
	clientScript := `expect "Hello"
send "Hi"
expect "Bye"`
	
	result := Execute(t, serverScript, clientScript, nil)
	
	// Basic assertions - database will be nil since no DatabasePath was provided
	if result.Database != nil {
		t.Error("Expected no database instance when DatabasePath not provided")
	}
	
	if !strings.Contains(result.ClientOutput, "Hello") {
		t.Errorf("Expected 'Hello' in client output, got: %q", result.ClientOutput)
	}
	
	t.Log("Dual expect scripts completed successfully!")
}


// TestExecuteScript demonstrates running a TWX script with the API
func TestExecuteScript(t *testing.T) {
	// Simple server responses for script execution
	serverScript := `send "Connected*"
send "Script response*"`

	// Client expects script output, then server responses
	clientScript := `expect "Hello from script"
expect "Connected"
expect "Script response"`

	// Use ConnectOptions with ScriptName as content (will be auto-converted to temp file)
	twxScript := `echo "Hello from script"`
	connectOpts := &api.ConnectOptions{ScriptName: twxScript}
	result := Execute(t, serverScript, clientScript, connectOpts)

	// Verify script execution - database will be nil since no DatabasePath was provided
	if result.Database != nil {
		t.Error("Expected no database instance when DatabasePath not provided")
	}

	if !strings.Contains(result.ClientOutput, "Connected") {
		t.Errorf("Expected 'Connected' in client output, got: %q", result.ClientOutput)
	}

	if !strings.Contains(result.ClientOutput, "Script response") {
		t.Errorf("Expected 'Script response' in client output, got: %q", result.ClientOutput)
	}

	t.Log("ExecuteScript test completed successfully!")
}

// TestExecuteScriptWithGame demonstrates running a script in game context and validating database state
func TestExecuteScriptWithGame(t *testing.T) {
	// Server sends sector display, warps, and command prompt to trigger database save
	serverScript := `send "Sector  : 123 in The Sphere*"
send "Warps to Sector(s) : (124) - 125*"
send "Command [TL=00:00:01]:[123] (?=Help)? : "`

	// Client expects the sector data
	clientScript := `expect "Sector  : 123"
expect "Warps to Sector"
expect "Command"`

	// Use ConnectOptions with both DatabasePath and ScriptName (content will be auto-converted to temp file)
	dbPath := t.TempDir() + "/test.db"
	twxScript := `# Script just needs to be present for parsing to occur`
	connectOpts := &api.ConnectOptions{
		DatabasePath: dbPath,
		ScriptName:   twxScript,
	}
	result := Execute(t, serverScript, clientScript, connectOpts)

	// Verify game context was created
	if result.Database == nil {
		t.Fatal("Expected database instance")
	}

	// Verify sector was parsed and saved to database
	var sectorExists int
	var err error
	err = result.Database.QueryRow("SELECT COUNT(*) FROM sectors WHERE sector_index = 123").Scan(&sectorExists)
	if err != nil {
		t.Fatalf("Failed to query sector 123: %v", err)
	}

	if sectorExists != 1 {
		t.Errorf("Expected sector 123 to be saved, but found %d records", sectorExists)
	}

	t.Log("ExecuteScriptWithGame test completed successfully!")
}

// TestExecuteWithGame demonstrates Execute() in game context without a script
func TestExecuteWithGame(t *testing.T) {
	// Server sends sector display, warps, and command prompt to trigger database save
	serverScript := `send "Sector  : 456 in The Sphere*"
send "Warps to Sector(s) : (457) - 458*"
send "Command [TL=00:00:01]:[456] (?=Help)? : "`

	// Client expects the sector data
	clientScript := `expect "Sector  : 456"
expect "Warps to Sector"
expect "Command"`

	// Use Connect with forced database path
	dbPath := t.TempDir() + "/test.db"
	connectOpts := &api.ConnectOptions{DatabasePath: dbPath}
	
	result := Execute(t, serverScript, clientScript, connectOpts)

	// Verify game context was created
	if result.Database == nil {
		t.Fatal("Expected database instance")
	}

	// Verify sector was parsed and saved to database
	var sectorExists int
	var err error
	err = result.Database.QueryRow("SELECT COUNT(*) FROM sectors WHERE sector_index = 456").Scan(&sectorExists)
	if err != nil {
		t.Fatalf("Failed to query sector 456: %v", err)
	}

	if sectorExists != 1 {
		t.Errorf("Expected sector 456 to be saved, but found %d records", sectorExists)
	}

	t.Log("ExecuteWithGame test completed successfully!")
}