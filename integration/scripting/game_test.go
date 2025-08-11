

package scripting

import (
	"strings"
	"testing"
	"time"
	"twist/internal/api"
	"twist/internal/proxy/database"
)

// TestSendCommand_RealIntegration tests SEND command with real VM and database
func TestSendCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		send "look"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// SEND commands don't produce echo output, they send to game server
	// We can verify the command executed without error
}

// TestSendCommand_MultipleParameters tests SEND with multiple parameters
func TestSendCommand_MultipleParameters_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $action "look"
		setVar $target "north"
		send $action " " $target
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
}

// TestSendCommand_WithVariables tests SEND using variables with persistence
func TestSendCommand_WithVariables_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $command "examine"
		setVar $object "sword"
		saveVar $command
		saveVar $object
		send $command " " $object
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
}

// TestWaitForCommand_RealIntegration tests WAITFOR command
func TestWaitForCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	// This test simulates a WAITFOR that should complete when matching text arrives
	script := `
		echo "Starting wait"
		waitfor "test pattern"
		echo "Wait completed"
	`
	
	// Start script execution asynchronously since WAITFOR will block
	resultChan, err := tester.ExecuteScriptAsync(script)
	if err != nil {
		t.Fatalf("Failed to start async script execution: %v", err)
	}
	
	// Give script time to start and reach WAITFOR
	time.Sleep(1 * time.Millisecond)
	
	// Verify the script is waiting
	if !tester.IsWaiting() {
		t.Error("Script should be waiting after WAITFOR command")
	}
	
	// Simulate incoming network text that matches the pattern
	err = tester.SimulateNetworkInput("This contains test pattern in the middle")
	if err != nil {
		t.Errorf("Failed to simulate network input: %v", err)
	}
	
	// Wait for script completion
	select {
	case result := <-resultChan:
		if result.Error != nil {
			t.Errorf("Script execution failed: %v", result.Error)
		}
		
		// Should have both echo messages
		if len(result.Output) < 2 {
			t.Errorf("Expected at least 2 output lines, got %d: %v", len(result.Output), result.Output)
		}
		
		if len(result.Output) > 0 && result.Output[0] != "Starting wait" {
			t.Errorf("First echo: got %q, want %q", result.Output[0], "Starting wait")
		}
		
		if len(result.Output) > 1 && result.Output[1] != "Wait completed" {
			t.Errorf("Second echo: got %q, want %q", result.Output[1], "Wait completed")
		}
		
	case <-time.After(5 * time.Second):
		t.Error("WAITFOR test timed out - WAITFOR may not be working correctly")
	}
}


// TestPauseCommand_RealIntegration tests PAUSE command
func TestPauseCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "Before pause"
		pause
		echo "After pause"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// PAUSE should allow script to continue and complete
	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Before pause" {
		t.Errorf("First echo: got %q, want %q", result.Output[0], "Before pause")
	}
	
	if len(result.Output) > 1 && result.Output[1] != "After pause" {
		t.Errorf("Second echo: got %q, want %q", result.Output[1], "After pause")
	}
}

// TestHaltCommand_RealIntegration tests HALT command
func TestHaltCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		echo "Before halt"
		halt
		echo "This should not execute"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// HALT should stop execution, so only first echo should appear
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line after HALT, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Before halt" {
		t.Errorf("Echo before halt: got %q, want %q", result.Output[0], "Before halt")
	}
}

// TestGameCommands_CrossInstancePersistence tests game commands with persistent variables
func TestGameCommands_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First script execution - save command template
	tester1 := NewIntegrationScriptTester(t)
	
	script1 := `
		setVar $base_command "examine"
		setVar $default_target "room"
		saveVar $base_command
		saveVar $default_target
		echo "Saved command template"
	`
	
	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}
	
	// Second script execution - load and use template (simulates VM restart)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)
	
	script2 := `
		loadVar $base_command
		loadVar $default_target
		send $base_command " " $default_target
		echo "Sent: " $base_command " " $default_target
	`
	
	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}
	
	if len(result2.Output) != 1 {
		t.Errorf("Expected 1 output line from second script, got %d", len(result2.Output))
	}
	
	expected := "Sent: examine room"
	if len(result2.Output) > 0 && result2.Output[0] != expected {
		t.Errorf("Cross-instance command: got %q, want %q", result2.Output[0], expected)
	}
}

// TestGameCommands_ConditionalSending tests sending with comparison results
func TestGameCommands_ConditionalSending_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		# Test comparison-based logic without complex conditionals
		setVar $health 50
		setVar $health_threshold 75
		
		isequal $health $health_threshold $health_result
		echo "Health comparison result (50 == 75): " $health_result
		
		# Test other comparison
		isgreater $health_threshold $health $greater_result
		echo "Threshold comparison result (75 > 50): " $greater_result
		
		# Simple sending based on known values
		send "check health status"
		echo "Health check command sent"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	// Should have 3 echo outputs
	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && result.Output[0] != "Health comparison result (50 == 75): 0" {
		t.Errorf("Equality comparison: got %q, want %q", result.Output[0], "Health comparison result (50 == 75): 0")
	}
	
	if len(result.Output) > 1 && result.Output[1] != "Threshold comparison result (75 > 50): 1" {
		t.Errorf("Greater comparison: got %q, want %q", result.Output[1], "Threshold comparison result (75 > 50): 1")
	}
	
	if len(result.Output) > 2 && result.Output[2] != "Health check command sent" {
		t.Errorf("Send command echo: got %q, want %q", result.Output[2], "Health check command sent")
	}
}

// TestGameCommands_StringConcatenation tests complex string building for commands
func TestGameCommands_StringConcatenation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $verb "tell"
		setVar $target "merchant"
		setVar $message "I want to buy sword"
		setVar $punctuation "."
		
		send $verb " " $target " " $message $punctuation
		echo "Sent message to " $target
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}
	
	expected := "Sent message to merchant"
	if len(result.Output) > 0 && result.Output[0] != expected {
		t.Errorf("String concatenation: got %q, want %q", result.Output[0], expected)
	}
}

// TestGameCommands_EmptyAndSpecialCommands tests edge cases
func TestGameCommands_EmptyAndSpecialCommands_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $empty ""
		setVar $newline "\n"
		setVar $space " "
		
		send $empty
		echo "Sent empty command"
		
		send "look" $newline "inventory"
		echo "Sent multiline command"
		
		send "say" $space "hello world"
		echo "Sent spaced command"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}
	
	expectedOutputs := []string{
		"Sent empty command",
		"Sent multiline command", 
		"Sent spaced command",
	}
	
	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestGameCommands_NumberToStringConversion tests sending numeric values
func TestGameCommands_NumberToStringConversion_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
		setVar $sector_num 1234
		setVar $credits 5000.50
		
		send "warp " $sector_num
		echo "Warped to sector " $sector_num
		
		send "offer " $credits " credits"
		echo "Offered " $credits " credits"
	`
	
	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}
	
	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}
	
	if len(result.Output) > 0 && !strings.Contains(result.Output[0], "1234") {
		t.Errorf("Sector echo should contain '1234': got %q", result.Output[0])
	}
	
	if len(result.Output) > 1 && !strings.Contains(result.Output[1], "5000.5") {
		t.Errorf("Credits echo should contain '5000.5': got %q", result.Output[1])
	}
}

// TestWaitForCommand_WithTriggerInteraction tests WAITFOR working with triggers
func TestWaitForCommand_WithTriggerInteraction_RealIntegration(t *testing.T) {
	// Server sends game-like messages that will trigger the script behavior
	serverScript := `send "Setting up trigger*"
send "test trigger message*"
send "continue with the script*"`

	// Client expects the messages and verifies trigger/waitfor interaction
	clientScript := `expect "Setting up trigger"
expect "test trigger message" 
expect "continue with the script"`

	// Script that sets up trigger and uses waitfor
	twxScript := `
echo "Setting up trigger"
settexttrigger 1 "echo 'Trigger fired'" "test trigger"
echo "Starting wait"  
waitfor "continue"
echo "Wait completed"
`

	result := Execute(t, serverScript, clientScript, &api.ConnectOptions{ScriptName: twxScript})
	
	if result.Database != nil {
		t.Error("Expected no database instance when DatabasePath not provided")
	}

	// Verify script executed and produced expected output
	if !strings.Contains(result.ClientOutput, "Setting up trigger") {
		t.Errorf("Expected 'Setting up trigger' in client output, got: %q", result.ClientOutput)
	}
	
	if !strings.Contains(result.ClientOutput, "continue with the script") {
		t.Errorf("Expected 'continue with the script' in client output, got: %q", result.ClientOutput)
	}
	
	t.Log("WAITFOR with trigger interaction test completed successfully!")
}

// TestGetSectorCommand_RealIntegration tests getSector with real database and VM
func TestGetSectorCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Create test sector data in database
	err := tester.setupData.DB.SaveSector(createTestSector(), 123)
	if err != nil {
		t.Fatalf("Failed to save test sector: %v", err)
	}

	// Create test port data in database
	testPort := database.TPort{
		Name:           "Trading Post Alpha",
		Dead:           false,
		BuildTime:      0,
		ClassIndex:     1, // Port class 1
		BuyProduct:     [3]bool{true, false, true},
		ProductPercent: [3]int{100, 0, 100},
		ProductAmount:  [3]int{500, 0, 300},
		UpDate:         time.Now(),
	}
	err = tester.setupData.DB.SavePort(testPort, 123)
	if err != nil {
		t.Fatalf("Failed to save test port: %v", err)
	}

	script := `
		getSector 123 $s
		echo "Sector index: " $s.index
		echo "Density: " $s.density
		echo "Explored: " $s.explored
		echo "Port class: " $s.port.class
		echo "Port exists: " $s.port.exists
		echo "Warps: " $s.warps
		echo "Warp[1]: " $s.warp[1]
		echo "Warp[2]: " $s.warp[2]
		echo "Beacon: " $s.beacon
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Verify the expected outputs
	expectedOutputs := []string{
		"Sector index: 123",
		"Density: 45",
		"Explored: DENSITY",
		"Port class: 1",
		"Port exists: 1", 
		"Warps: 3",
		"Warp[1]: 2",
		"Warp[2]: 3", 
		"Beacon: Test Beacon",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestGetSectorCommand_NonExistentSector tests getSector with sector that doesn't exist
func TestGetSectorCommand_NonExistentSector_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		getSector 999 $empty
		echo "Non-existent density: " $empty.density
		echo "Non-existent port class: " $empty.port.class
		echo "Non-existent explored: " $empty.explored
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Non-existent density: -1",
		"Non-existent port class: 0",
		"Non-existent explored: NO",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestGetSectorCommand_ZeroIndex tests getSector with zero index (should be ignored)
func TestGetSectorCommand_ZeroIndex_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		getSector 0 $zero
		echo "Zero test completed"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Zero test completed",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestGetSectorCommand_TradingScriptPattern tests getSector like 1_Trade.ts would use it
func TestGetSectorCommand_TradingScriptPattern_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Create test sectors like a real trading scenario
	sectors := []struct {
		index int
		density int
		hasPort bool
		portClass int
	}{
		{1, 0, false, 0},        // Empty sector
		{2, 100, true, 1},       // Port sector with 100 density (bad for trading)
		{3, 45, false, 0},       // Good density sector without port
		{4, 67, true, 2},        // Good density sector with port
	}

	for _, s := range sectors {
		sector := createTestSectorWithData(s.density, s.hasPort, s.portClass)
		err := tester.setupData.DB.SaveSector(sector, s.index)
		if err != nil {
			t.Fatalf("Failed to save test sector %d: %v", s.index, err)
		}
	}

	// Script that mimics 1_Trade.ts decision logic
	script := `
		# Test sector analysis like in 1_Trade.ts
		setVar $bestWarp 0
		setVar $bestWeight 9999
		
		# Check sector 2 (high density - should be avoided, following real 1_Trade.ts logic)
		getSector 2 $s
		setVar $weight 0
		if ($s.density <> 100) and ($s.density <> 0)
			add $weight 100
			add $weight $s.density
		end
		echo "Sector 2 weight: " $weight
		
		# Check sector 3 (good density, following real 1_Trade.ts logic)
		getSector 3 $s3
		setVar $weight3 0
		if ($s3.density <> 100) and ($s3.density <> 0)
			add $weight3 100
			add $weight3 $s3.density
		end
		echo "Sector 3 weight: " $weight3
		
		# Check sector 4 (good density, following real 1_Trade.ts logic)
		getSector 4 $s4
		setVar $weight4 0
		if ($s4.density <> 100) and ($s4.density <> 0)
			add $weight4 100
			add $weight4 $s4.density
		end
		echo "Sector 4 weight: " $weight4
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	// Verify trading logic calculations (following real 1_Trade.ts logic)
	// Sector 2: weight = 0 (density = 100, so condition fails)
	// Sector 3: weight = 100 + 45 = 145 (good density)
	// Sector 4: weight = 100 + 67 = 167 (good density)
	expectedOutputs := []string{
		"Sector 2 weight: 0",      // High density sector avoided
		"Sector 3 weight: 145",    // Good density sector
		"Sector 4 weight: 167",    // Good density sector
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output line %d: got %q, want %q", i, result.Output[i], expected)
		}
	}
}

// TestGetSectorCommand_CrossInstancePersistence tests getSector data persistence across VM instances
func TestGetSectorCommand_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First instance - save sector data and verify getSector works
	tester1 := NewIntegrationScriptTester(t)
	
	err := tester1.setupData.DB.SaveSector(createTestSector(), 456)
	if err != nil {
		t.Fatalf("Failed to save test sector: %v", err)
	}

	// Create test port data
	testPort := database.TPort{
		Name:           "Trading Post Alpha",
		Dead:           false,
		BuildTime:      0,
		ClassIndex:     1,
		BuyProduct:     [3]bool{true, false, true},
		ProductPercent: [3]int{100, 0, 100},
		ProductAmount:  [3]int{500, 0, 300},
		UpDate:         time.Now(),
	}
	err = tester1.setupData.DB.SavePort(testPort, 456)
	if err != nil {
		t.Fatalf("Failed to save test port: %v", err)
	}

	script1 := `
		getSector 456 $s
		echo "First instance density: " $s.density
		echo "First instance port class: " $s.port.class
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}

	// Second instance with shared database - should see same sector data
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		getSector 456 $s2
		echo "Second instance density: " $s2.density
		echo "Second instance port class: " $s2.port.class
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}

	// Both instances should see the same data
	expectedOutputs1 := []string{
		"First instance density: 45",
		"First instance port class: 1",
	}

	expectedOutputs2 := []string{
		"Second instance density: 45", 
		"Second instance port class: 1",
	}

	// Verify first instance
	if len(result1.Output) != len(expectedOutputs1) {
		t.Errorf("Expected %d output lines from first instance, got %d", len(expectedOutputs1), len(result1.Output))
	}

	for i, expected := range expectedOutputs1 {
		if i < len(result1.Output) && result1.Output[i] != expected {
			t.Errorf("First instance output line %d: got %q, want %q", i, result1.Output[i], expected)
		}
	}

	// Verify second instance
	if len(result2.Output) != len(expectedOutputs2) {
		t.Errorf("Expected %d output lines from second instance, got %d", len(expectedOutputs2), len(result2.Output))
	}

	for i, expected := range expectedOutputs2 {
		if i < len(result2.Output) && result2.Output[i] != expected {
			t.Errorf("Second instance output line %d: got %q, want %q", i, result2.Output[i], expected)
		}
	}
}

// Helper functions for getSector tests

// createTestSector creates a test sector with predefined data for testing
func createTestSector() database.TSector {
	return database.TSector{
		Warp:          [6]int{2, 3, 4, 0, 0, 0}, // 3 warps
		Density:       45,
		NavHaz:        0,
		Beacon:        "Test Beacon",
		Constellation: "Test Constellation",
		Explored:      database.EtDensity, // 2 = DENSITY
	}
}

// createTestSectorWithData creates a test sector with custom data
func createTestSectorWithData(density int, hasPort bool, portClass int) database.TSector {
	sector := database.TSector{
		Warp:          [6]int{2, 3, 4, 0, 0, 0},
		Density:       density,
		NavHaz:        0,
		Beacon:        "Test Beacon",
		Constellation: "Test Constellation", 
		Explored:      database.EtDensity,
	}

	// If hasPort is true, the port will be saved separately using SavePort()

	return sector
}