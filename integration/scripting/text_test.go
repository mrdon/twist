package scripting

import (
	"strings"
	"testing"
)

// TestEchoCommand_RealIntegration tests ECHO command with real VM and database
func TestEchoCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		echo "Hello, World!"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Hello, World!" {
		t.Errorf("ECHO output: got %q, want %q", result.Output[0], "Hello, World!")
	}
}

// TestEchoCommand_MultipleParameters tests ECHO with multiple parameters
func TestEchoCommand_MultipleParameters_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $test_var "Variable"
		echo "Hello " $test_var " World!"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	expected := "Hello Variable World!"
	if len(result.Output) > 0 && result.Output[0] != expected {
		t.Errorf("ECHO multi-param output: got %q, want %q", result.Output[0], expected)
	}
}

// TestClientMessageCommand_RealIntegration tests CLIENTMESSAGE command
func TestClientMessageCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		clientmessage "Client message test"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Client message test" {
		t.Errorf("CLIENTMESSAGE output: got %q, want %q", result.Output[0], "Client message test")
	}
}

// TestClientMessageCommand_WithVariable tests CLIENTMESSAGE using a variable
func TestClientMessageCommand_WithVariable_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $msg_var "Variable message"
		clientmessage $msg_var
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Variable message" {
		t.Errorf("CLIENTMESSAGE variable output: got %q, want %q", result.Output[0], "Variable message")
	}
}

// TestDisplayTextCommand_RealIntegration tests DISPLAYTEXT command (alias for ECHO)
func TestDisplayTextCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		displaytext "Display text test"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Display text test" {
		t.Errorf("DISPLAYTEXT output: got %q, want %q", result.Output[0], "Display text test")
	}
}

// TestTextCommands_CrossInstancePersistence tests text output with persistent variables
func TestTextCommands_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First script execution - save variable
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $message "Persistent message"
		saveVar $message
		echo "Saved: " $message
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}

	// Second script execution - load and use variable (simulates VM restart with shared DB)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $message
		echo "Loaded: " $message
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}

	if len(result2.Output) != 1 {
		t.Errorf("Expected 1 output line from second script, got %d", len(result2.Output))
	}

	expected := "Loaded: Persistent message"
	if len(result2.Output) > 0 && result2.Output[0] != expected {
		t.Errorf("Cross-instance echo: got %q, want %q", result2.Output[0], expected)
	}
}

// TestTextCommands_NumberToStringConversion tests number to string conversion
func TestTextCommands_NumberToStringConversion_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $num_var 42.5
		echo "Number: " $num_var
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	// Should convert number to string representation (rounded)
	if len(result.Output) > 0 && !strings.Contains(result.Output[0], "43") {
		t.Errorf("ECHO number conversion: got %q, want to contain '43' (42.5 rounded)", result.Output[0])
	}
}

// TestTextCommands_EmptyAndSpecialCharacters tests text commands with empty and special strings
func TestTextCommands_EmptyAndSpecialCharacters_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $empty ""
		setVar $special "Line1\nLine2\tTabbed"
		echo "Empty: [" $empty "]"
		echo "Special: " $special
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Empty: []" {
		t.Errorf("Empty string echo: got %q, want %q", result.Output[0], "Empty: []")
	}

	if len(result.Output) > 1 && !strings.Contains(result.Output[1], "Special:") {
		t.Errorf("Special char echo: got %q, want to contain 'Special:'", result.Output[1])
	}
}

// TestTextCommands_VariableInterpolation tests complex variable interpolation
func TestTextCommands_VariableInterpolation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $name "World"
		setVar $greeting "Hello"
		setVar $punctuation "!"
		echo $greeting " " $name $punctuation
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	expected := "Hello World!"
	if len(result.Output) > 0 && result.Output[0] != expected {
		t.Errorf("Variable interpolation: got %q, want %q", result.Output[0], expected)
	}
}

// TestCutTextCommand_RealIntegration tests CUTTEXT command with real VM and database
func TestCutTextCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $source "Command [TL=00:10:05]:"
		cutText $source $result 1 7
		echo "Cut result: " $result
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	expected := "Cut result: Command"
	if len(result.Output) > 0 && result.Output[0] != expected {
		t.Errorf("CUTTEXT output: got %q, want %q", result.Output[0], expected)
	}
}

// TestCutTextCommand_EdgeCases tests CUTTEXT command with edge cases
func TestCutTextCommand_EdgeCases_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $source "Short"
		cutText $source $result1 1 10
		cutText $source $result3 3 2
		echo "Long cut: [" $result1 "]"
		echo "Mid cut: [" $result3 "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Long cut: [Short]", // Should return full string when length exceeds
		"Mid cut: [or]",     // Should cut from position 3, length 2
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("CUTTEXT edge case %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestCutTextCommand_ErrorHandling tests CUTTEXT command error behavior matching Pascal
func TestCutTextCommand_ErrorHandling_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test case where start position is beyond end of line (should error like Pascal)
	script := `
		setVar $source "Short"
		cutText $source $result 10 5
		echo "Should not reach here"
	`

	result := tester.ExecuteScript(script)

	// Should have an error like Pascal: "CutText: Start position beyond End Of Line"
	if result.Error == nil {
		t.Errorf("Expected error for start position beyond end of line, but got none")
	}

	if result.Error != nil && !strings.Contains(result.Error.Error(), "Start position beyond End Of Line") {
		t.Errorf("Expected Pascal-style error message, got: %v", result.Error)
	}

	// No output should be produced when error occurs
	if len(result.Output) != 0 {
		t.Errorf("Expected no output when error occurs, got %d lines", len(result.Output))
	}
}

// TestCutTextCommand_BoundaryConditions tests CUTTEXT boundary conditions
func TestCutTextCommand_BoundaryConditions_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $source "Test"
		cutText $source $result1 1 0
		cutText $source $result2 4 1
		cutText $source $result3 1 4
		echo "Zero length: [" $result1 "]"
		echo "Last char: [" $result2 "]"
		echo "Exact length: [" $result3 "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Zero length: []",      // Zero length should return empty
		"Last char: [t]",       // Should get last character
		"Exact length: [Test]", // Should get exact string
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("CUTTEXT boundary %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestGetWordCommand_RealIntegration tests GETWORD command with real VM and database
func TestGetWordCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $line "Sector 123 Density: 45 Warps: 3"
		getWord $line $sector 2
		getWord $line $density 4
		getWord $line $warps 6
		echo "Sector: " $sector
		echo "Density: " $density  
		echo "Warps: " $warps
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Sector: 123",
		"Density: 45",
		"Warps: 3",
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("GETWORD output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestGetWordCommand_EdgeCases tests GETWORD command with edge cases
func TestGetWordCommand_EdgeCases_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $line "One Two Three"
		getWord $line $first 1
		getWord $line $beyond 5
		getWord $line $zero 0
		echo "First: [" $first "]"
		echo "Beyond: [" $beyond "]"
		echo "Zero: [" $zero "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"First: [One]", // First word
		"Beyond: [0]",  // Word number beyond range should return "0" (Pascal default)
		"Zero: [0]",    // Word number 0 should return "0" (Pascal default)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("GETWORD edge case %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestGetWordCommand_DefaultParameter tests GETWORD command with optional default parameter
func TestGetWordCommand_DefaultParameter_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $line "Alpha Beta"
		getWord $line $exists 1
		getWord $line $missing 5 "DefaultValue"
		getWord $line $missing_no_default 6
		getWord $line $custom_default 10 "CUSTOM"
		echo "Exists: [" $exists "]"
		echo "Missing with default: [" $missing "]"
		echo "Missing no default: [" $missing_no_default "]"
		echo "Custom default: [" $custom_default "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Exists: [Alpha]",                      // Normal word extraction
		"Missing with default: [DefaultValue]", // Uses provided default
		"Missing no default: [0]",              // Uses Pascal default "0"
		"Custom default: [CUSTOM]",             // Uses custom default
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("GETWORD default parameter %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestGetWordCommand_EmptyString tests GETWORD command with empty input
func TestGetWordCommand_EmptyString_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $empty ""
		getWord $empty $result1 1
		getWord $empty $result2 1 "EmptyDefault"
		echo "Empty string word 1: [" $result1 "]"
		echo "Empty string with default: [" $result2 "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Empty string word 1: [0]",                  // Pascal default for empty string
		"Empty string with default: [EmptyDefault]", // Provided default for empty string
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("GETWORD empty string %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestStripTextCommand_RealIntegration tests STRIPTEXT command with real VM and database
func TestStripTextCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $line "Sector (123) has density"
		echo "Before: " $line
		striptext $line "("
		echo "After (: " $line
		striptext $line ")"
		echo "After ): " $line
		striptext $line " "
		echo "After spaces: " $line
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Before: Sector (123) has density",
		"After (: Sector 123) has density",
		"After ): Sector 123 has density",
		"After spaces: Sector123hasdensity",
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("STRIPTEXT output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestStripTextCommand_EmptyAndNonExistent tests STRIPTEXT with empty and non-existent strings
func TestStripTextCommand_EmptyAndNonExistent_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $line "Hello World"
		striptext $line ""
		echo "After empty strip: " $line
		striptext $line "xyz"
		echo "After non-existent strip: " $line
		striptext $line "Hello World"
		echo "After full strip: [" $line "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"After empty strip: Hello World",        // Empty string should not change anything
		"After non-existent strip: Hello World", // Non-existent string should not change anything
		"After full strip: []",                  // Stripping entire string should leave empty
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("STRIPTEXT edge case %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXTextProcessing_TradingScriptScenario tests text processing commands like in 1_Trade.ts
func TestTWXTextProcessing_TradingScriptScenario_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $currentline "Command [TL=00:10:05]:"
		cutText $currentline $location 1 7
		echo "Location: " $location
		
		setVar $scanline "LongRange Scan : Holographic Scanner"
		getWord $scanline $scantype 4
		echo "Scanner Type: " $scantype
		
		setVar $densityline "Sector 123 : 45 density, 3 warps"
		striptext $densityline ":"
		striptext $densityline ","
		echo "Cleaned line: " $densityline
		getWord $densityline $sector 2
		getWord $densityline $density 3
		echo "Parsed - Sector: " $sector " Density: " $density
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Location: Command",
		"Scanner Type: Holographic",
		"Cleaned line: Sector 123  45 density 3 warps",
		"Parsed - Sector: 123 Density: 45",
	}

	for i, expected := range expectedOutputs[:4] { // Check first 4 outputs
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Trading script scenario %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXTextProcessing_DatabasePersistence tests that text processing results persist across VM instances
func TestTWXTextProcessing_DatabasePersistence_RealIntegration(t *testing.T) {
	// First script execution - process text and save results
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $gameoutput "Sector 456 : 78 density, 2 warps"
		cutText $gameoutput $sector_part 1 10
		getWord $gameoutput $sector_num 2
		striptext $gameoutput ":"
		getWord $gameoutput $density_val 3
		saveVar $sector_part
		saveVar $sector_num
		saveVar $density_val
		echo "Processed and saved"
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}

	// Second script execution - load processed data from database (simulates VM restart)
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $sector_part
		loadVar $sector_num
		loadVar $density_val
		echo "Loaded sector part: " $sector_part
		echo "Loaded sector number: " $sector_num
		echo "Loaded density: " $density_val
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}

	if len(result2.Output) != 3 {
		t.Errorf("Expected 3 output lines from second script, got %d", len(result2.Output))
	}

	expectedOutputs := []string{
		"Loaded sector part: Sector 456",
		"Loaded sector number: 456",
		"Loaded density: 78",
	}

	for i, expected := range expectedOutputs {
		if i < len(result2.Output) && result2.Output[i] != expected {
			t.Errorf("Database persistence %d: got %q, want %q", i+1, result2.Output[i], expected)
		}
	}
}

// TestTWXStringManipulation_GetWord tests TWX getWord function
func TestTWXStringManipulation_GetWord_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test TWX string manipulation functions
		setVar $text "Hello World 123"
		
		# Get word from string
		getWord $text $word1 1
		getWord $text $word2 2
		getWord $text $word3 3
		
		echo "Word 1: " $word1
		echo "Word 2: " $word2
		echo "Word 3: " $word3
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX getWord script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Word 1: Hello",
		"Word 2: World",
		"Word 3: 123",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("TWX getWord output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXStringManipulation_CutText tests TWX cutText function
func TestTWXStringManipulation_CutText_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $text "Hello World 123"
		
		# Cut text
		cutText $text $substring 1 5
		echo "Substring: " $substring
		
		# Get length
		getLength $text $length
		echo "Length: " $length
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX cutText script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Substring: Hello", // Characters 1-5
		"Length: 15",       // Length of "Hello World 123"
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("TWX cutText output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXStringManipulation_StripText tests TWX stripText function
func TestTWXStringManipulation_StripText_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Strip text
		setVar $textWithSpaces "  Hello  "  
		stripText $textWithSpaces " "
		echo "Stripped: [" $textWithSpaces "]"
		
		# Strip multiple characters
		setVar $textWithPunc "Hello, World!"
		stripText $textWithPunc ","
		stripText $textWithPunc "!"
		echo "No punctuation: " $textWithPunc
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX stripText script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Stripped: [Hello]",           // Spaces stripped
		"No punctuation: Hello World", // Punctuation stripped
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("TWX stripText output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXTextMerging_MergeText tests mergeText function
func TestTWXTextMerging_MergeText_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test TWX mergeText function
		setVar $base "Hello"
		setVar $suffix "World"
		mergeText $base $suffix $result
		
		echo "Merged: " $result
		
		# Test with numbers
		setVar $prefix "Sector "
		setVar $number 1234
		mergeText $prefix $number $sector_text
		echo "Sector text: " $sector_text
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX mergeText script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Merged: HelloWorld",
		"Sector text: Sector 1234",
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("TWX mergeText output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXStringManipulation_CURRENTLINESimulation tests CURRENTLINE simulation
func TestTWXStringManipulation_CURRENTLINESimulation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test mock game state commands
		setVar $currentSector 1234
		echo "Current sector: " $currentSector
		
		# Test CURRENTLINE simulation
		setVar $CURRENTLINE "Sector  : 1234  Density: 100"
		cutText $CURRENTLINE $location 1 6
		echo "Location: " $location
		
		getWord $CURRENTLINE $sector 3
		echo "Parsed sector: " $sector
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX CURRENTLINE script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Current sector: 1234",
		"Location: Sector",
		"Parsed sector: 1234",
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("TWX CURRENTLINE output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}
