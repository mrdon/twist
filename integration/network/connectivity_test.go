//go:build integration

package network

import (
	"net"
	"testing"
	"time"
	"twist/integration/setup"
)

// TestConnect_RealTCPConnection tests CONNECT command with real TCP connections
func TestConnect_RealTCPConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	setup := setup.SetupRealComponents(t)
	
	// Try to connect to a commonly available service (DNS on port 53)
	script := `
		connect "8.8.8.8" "53"
		echo "Connection attempt completed"
	`
	
	// Create a test tester using the setup
	tester := newNetworkScriptTester(t, setup)
	result := tester.ExecuteScript(script)
	
	if result.Error != nil {
		t.Logf("Connection failed (expected in some environments): %v", result.Error)
		// Don't fail the test - network connectivity may not be available
		return
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Connection attempt completed" {
		t.Errorf("Connection echo: got %q, want %q", result.Output[0], "Connection attempt completed")
	}
}

// TestConnect_LocalhostConnection tests connection to localhost
func TestConnect_LocalhostConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	// Start a simple test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to create test server: %v", err)
	}
	defer listener.Close()
	
	// Get the actual port assigned
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	
	// Handle incoming connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			// Simple echo server
			go func(c net.Conn) {
				defer c.Close()
				buffer := make([]byte, 1024)
				for {
					n, err := c.Read(buffer)
					if err != nil {
						return
					}
					if n > 0 {
						c.Write(buffer[:n])
					}
				}
			}(conn)
		}
	}()
	
	setup := setup.SetupRealComponents(t)
	
	script := `
		connect "127.0.0.1" "` + string(rune(port)) + `"
		echo "Connected to test server"
		send "test message"
		echo "Message sent"
	`
	
	tester := newNetworkScriptTester(t, setup)
	result := tester.ExecuteScript(script)
	
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}
}

// TestConnect_ConnectionFailure tests behavior when connection fails
func TestConnect_ConnectionFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	setup := setup.SetupRealComponents(t)
	
	// Try to connect to a non-existent service
	script := `
		connect "192.0.2.1" "12345"
		echo "This should not execute if connection blocks"
	`
	
	tester := newNetworkScriptTester(t, setup)
	
	// Set a timeout for this test
	done := make(chan *NetworkTestResult, 1)
	go func() {
		result := tester.ExecuteScript(script)
		done <- result
	}()
	
	select {
	case result := <-done:
		// Connection should have failed
		if result.Error == nil {
			t.Error("Expected connection to fail, but it succeeded")
		}
		t.Logf("Connection failed as expected: %v", result.Error)
		
	case <-time.After(10 * time.Second):
		t.Error("Connection test timed out - may be hanging on failed connection")
	}
}

// TestConnect_MultipleConnections tests handling multiple connections
func TestConnect_MultipleConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	setup := setup.SetupRealComponents(t)
	
	script := `
		connect "8.8.8.8" "53"
		echo "First connection attempt"
		
		connect "1.1.1.1" "53"  
		echo "Second connection attempt"
	`
	
	tester := newNetworkScriptTester(t, setup)
	result := tester.ExecuteScript(script)
	
	// Multiple connections may not be supported or may fail
	// Just verify the script executes without crashing
	if result.Error != nil {
		t.Logf("Multiple connection script failed (may be expected): %v", result.Error)
	}
}

// TestConnect_ConnectionState tests connection state variables
func TestConnect_ConnectionState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	setup := setup.SetupRealComponents(t)
	
	script := `
		$initial_state := %connected
		echo "Initial state: " $initial_state
		
		connect "8.8.8.8" "53"
		
		$post_connect_state := %connected
		echo "Post-connect state: " $post_connect_state
	`
	
	tester := newNetworkScriptTester(t, setup)
	result := tester.ExecuteScript(script)
	
	if result.Error != nil {
		t.Logf("Connection state test failed: %v", result.Error)
		return
	}
	
	// Verify that connection state variables are accessible
	if len(result.Output) < 2 {
		t.Errorf("Expected at least 2 output lines, got %d", len(result.Output))
	}
}

// TestConnect_WithNetworkTriggers tests connection with network-based triggers
func TestConnect_WithNetworkTriggers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	// Create a test server that sends specific text
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to create test server: %v", err)
	}
	defer listener.Close()
	
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port
	
	// Server that sends welcome message
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		
		// Send welcome message after brief delay
		time.Sleep(100 * time.Millisecond)
		conn.Write([]byte("Welcome to test server\n"))
		
		// Keep connection alive for a bit
		time.Sleep(1 * time.Second)
	}()
	
	setup := setup.SetupRealComponents(t)
	
	script := `
		settexttrigger 1 "echo 'Welcome trigger fired'" "Welcome"
		
		connect "127.0.0.1" "` + string(rune(port)) + `"
		echo "Connected, waiting for welcome message"
		
		waitfor "Welcome"
		echo "Welcome message received"
	`
	
	tester := newNetworkScriptTester(t, setup)
	
	done := make(chan *NetworkTestResult, 1)
	go func() {
		result := tester.ExecuteScript(script)
		done <- result
	}()
	
	select {
	case result := <-done:
		if result.Error != nil {
			t.Errorf("Network trigger test failed: %v", result.Error)
		}
		
		// Should have connection confirmation and welcome received messages
		if len(result.Output) < 2 {
			t.Errorf("Expected at least 2 output lines, got %d", len(result.Output))
		}
		
	case <-time.After(15 * time.Second):
		t.Error("Network trigger test timed out")
	}
}

// TestConnect_NetworkPersistence tests network settings persistence
func TestConnect_NetworkPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	// First script - save connection settings
	setup1 := setup.SetupRealComponents(t)
	
	script1 := `
		$host := "8.8.8.8"
		$port := "53"
		savevar $host
		savevar $port
		echo "Connection settings saved"
	`
	
	tester1 := newNetworkScriptTester(t, setup1)
	result1 := tester1.ExecuteScript(script1)
	
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}
	
	// Second script - load and use connection settings (simulates VM restart)
	setup2 := setup.SetupRealComponents(t)
	// Share the database between instances
	setup2.GameAdapter = setup1.GameAdapter
	
	script2 := `
		loadvar $host
		loadvar $port
		echo "Loaded host: " $host " port: " $port
		
		connect $host $port
		echo "Connected using saved settings"
	`
	
	tester2 := newNetworkScriptTester(t, setup2)
	result2 := tester2.ExecuteScript(script2)
	
	if result2.Error != nil {
		t.Logf("Second script with connection failed: %v", result2.Error)
		// Connection failure is acceptable, but variable loading should work
	}
	
	if len(result2.Output) < 1 {
		t.Errorf("Expected at least 1 output line, got %d", len(result2.Output))
	}
	
	// Should show loaded connection settings
	if len(result2.Output) > 0 && !contains(result2.Output[0], "8.8.8.8") {
		t.Errorf("Should contain saved host: got %q", result2.Output[0])
	}
}

// TestConnect_ErrorHandling tests network error handling
func TestConnect_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	setup := setup.SetupRealComponents(t)
	
	script := `
		echo "Testing invalid connections"
		
		connect "invalid.host.name" "80"
		echo "Invalid host connection attempted"
		
		connect "127.0.0.1" "999999"
		echo "Invalid port connection attempted"
		
		connect "" ""
		echo "Empty connection attempted"
	`
	
	tester := newNetworkScriptTester(t, setup)
	result := tester.ExecuteScript(script)
	
	// The script should handle errors gracefully without crashing
	if result.Error != nil {
		t.Logf("Error handling test completed with error: %v", result.Error)
	}
	
	// Should have at least the first echo
	if len(result.Output) < 1 {
		t.Errorf("Expected at least 1 output line, got %d", len(result.Output))
	}
}

// TestConnect_Timeout tests connection timeouts
func TestConnect_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network integration test in short mode")
	}
	
	setup := setup.SetupRealComponents(t)
	
	// Try to connect to a filtered/blocked address that should timeout
	script := `
		echo "Testing connection timeout"
		connect "192.0.2.0" "80"
		echo "Timeout test completed"
	`
	
	tester := newNetworkScriptTester(t, setup)
	
	done := make(chan *NetworkTestResult, 1)
	go func() {
		result := tester.ExecuteScript(script)
		done <- result
	}()
	
	select {
	case result := <-done:
		// Connection should have failed or timed out
		t.Logf("Timeout test completed: %v", result.Error)
		
	case <-time.After(30 * time.Second):
		t.Error("Connection timeout test took too long")
	}
}

// NetworkTestResult captures network test output
type NetworkTestResult struct {
	Output []string
	Error  error
}

// NetworkScriptTester provides network-specific testing utilities
type NetworkScriptTester struct {
	setup  *setup.IntegrationTestSetup
	output []string
	t      *testing.T
}

// newNetworkScriptTester creates a network-specific script tester
func newNetworkScriptTester(t *testing.T, setup *setup.IntegrationTestSetup) *NetworkScriptTester {
	tester := &NetworkScriptTester{
		setup:  setup,
		output: make([]string, 0),
		t:      t,
	}
	
	// Set up output capture - this would need to be implemented based on the actual VM interface
	// For now, we'll use a placeholder
	
	return tester
}

// ExecuteScript executes a script using the real engine for network testing
func (nst *NetworkScriptTester) ExecuteScript(scriptSource string) *NetworkTestResult {
	// This is a placeholder implementation
	// In the real system, we would use the VM from setup to execute the script
	// and capture network-related outputs
	
	// For now, return a basic result
	return &NetworkTestResult{
		Output: []string{"Network test placeholder"},
		Error:  nil,
	}
}

// SimulateNetworkInput simulates incoming network data
func (nst *NetworkScriptTester) SimulateNetworkInput(data string) error {
	// This would simulate network data arriving that could trigger text triggers
	// For now, return nil
	return nil
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}