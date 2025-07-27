//go:build integration

package scripting

import (
	"strings"
	"testing"
	"twist/integration/setup"
	"twist/internal/scripting"
	"twist/internal/scripting/parser"
	"twist/internal/scripting/vm"
)

// IntegrationTestResult captures the output from a script execution  
type IntegrationTestResult struct {
	Output   []string
	Commands []string
	Error    error
}

// IntegrationScriptTester provides real integration testing for TWX scripts
type IntegrationScriptTester struct {
	setupData *setup.IntegrationTestSetup
	t         *testing.T
}

// NewIntegrationScriptTester creates a new integration script tester with real components
func NewIntegrationScriptTester(t *testing.T) *IntegrationScriptTester {
	setupData := setup.SetupRealComponents(t)
	
	return &IntegrationScriptTester{
		setupData: setupData,
		t:         t,
	}
}

// NewIntegrationScriptTesterWithSharedDB creates a tester that shares a database with another tester
func NewIntegrationScriptTesterWithSharedDB(t *testing.T, sharedSetup *setup.IntegrationTestSetup) *IntegrationScriptTester {
	// Create real game adapter using the shared database
	gameAdapter := scripting.NewGameAdapter(sharedSetup.DB)
	
	// Create real VM
	realVM := vm.NewVirtualMachine(gameAdapter)
	
	newSetup := &setup.IntegrationTestSetup{
		DB:          sharedSetup.DB, // Share the same database
		GameAdapter: gameAdapter,
		VM:          realVM,
		DBPath:      sharedSetup.DBPath, // Share the same DB path
	}
	
	// Register cleanup for VM only (don't clean up shared DB)
	t.Cleanup(func() {
		// Only cleanup VM resources, not the shared database
	})
	
	return &IntegrationScriptTester{
		setupData: newSetup,
		t:         t,
	}
}

// ExecuteScript executes a TWX script and returns the results
func (tester *IntegrationScriptTester) ExecuteScript(script string) *IntegrationTestResult {
	// Parse the script
	lexer := parser.NewLexer(strings.NewReader(script))
	tokens, err := lexer.TokenizeAll()
	if err != nil {
		return &IntegrationTestResult{
			Output:   []string{},
			Commands: []string{},
			Error:    err,
		}
	}
	
	parserObj := parser.NewParser(tokens)
	ast, err := parserObj.Parse()
	if err != nil {
		return &IntegrationTestResult{
			Output:   []string{},
			Commands: []string{},
			Error:    err,
		}
	}
	
	// Capture output and commands
	var output []string
	var commands []string
	
	// Set up output handlers
	tester.setupData.VM.SetOutputHandler(func(text string) error {
		output = append(output, text)
		return nil
	})
	
	tester.setupData.VM.SetEchoHandler(func(text string) error {
		output = append(output, text)
		return nil
	})
	
	tester.setupData.VM.SetSendHandler(func(text string) error {
		commands = append(commands, text)
		return nil
	})
	
	// Load and execute the script
	if err := tester.setupData.VM.LoadScript(ast, nil); err != nil {
		return &IntegrationTestResult{
			Output:   output,
			Commands: commands,
			Error:    err,
		}
	}
	
	if err := tester.setupData.VM.Execute(); err != nil {
		return &IntegrationTestResult{
			Output:   output,
			Commands: commands,
			Error:    err,
		}
	}
	
	return &IntegrationTestResult{
		Output:   output,
		Commands: commands,
		Error:    nil,
	}
}

// AssertNoError asserts that the script execution had no error
func (tester *IntegrationScriptTester) AssertNoError(result *IntegrationTestResult) {
	if result.Error != nil {
		tester.t.Errorf("Script execution failed: %v", result.Error)
	}
}

// AssertError asserts that the script execution had an error
func (tester *IntegrationScriptTester) AssertError(result *IntegrationTestResult) {
	if result.Error == nil {
		tester.t.Error("Expected script execution to fail, but it succeeded")
	}
}

// AssertOutput asserts that the output matches the expected output exactly
func (tester *IntegrationScriptTester) AssertOutput(result *IntegrationTestResult, expectedOutput []string) {
	if len(result.Output) != len(expectedOutput) {
		tester.t.Errorf("Output length mismatch: got %d lines, expected %d lines", len(result.Output), len(expectedOutput))
		tester.t.Errorf("Got: %v", result.Output)
		tester.t.Errorf("Expected: %v", expectedOutput)
		return
	}
	
	for i, expected := range expectedOutput {
		if i >= len(result.Output) {
			tester.t.Errorf("Missing output line %d: expected %q", i, expected)
			continue
		}
		if result.Output[i] != expected {
			tester.t.Errorf("Output line %d mismatch: got %q, expected %q", i, result.Output[i], expected)
		}
	}
}

// AssertOutputContains asserts that the output contains the specified text
func (tester *IntegrationScriptTester) AssertOutputContains(result *IntegrationTestResult, expectedText string) {
	for _, line := range result.Output {
		if line == expectedText {
			return // Found it
		}
	}
	tester.t.Errorf("Output does not contain expected text %q. Got: %v", expectedText, result.Output)
}

// AssertCommands asserts that the commands match the expected commands exactly
func (tester *IntegrationScriptTester) AssertCommands(result *IntegrationTestResult, expectedCommands []string) {
	if len(result.Commands) != len(expectedCommands) {
		tester.t.Errorf("Command count mismatch: got %d commands, expected %d commands", len(result.Commands), len(expectedCommands))
		tester.t.Errorf("Got: %v", result.Commands)
		tester.t.Errorf("Expected: %v", expectedCommands)
		return
	}
	
	for i, expected := range expectedCommands {
		if i >= len(result.Commands) {
			tester.t.Errorf("Missing command %d: expected %q", i, expected)
			continue
		}
		if result.Commands[i] != expected {
			tester.t.Errorf("Command %d mismatch: got %q, expected %q", i, result.Commands[i], expected)
		}
	}
}

// SimulateNetworkInput simulates incoming network text for trigger processing
func (tester *IntegrationScriptTester) SimulateNetworkInput(text string) error {
	// Process the text through the VM's trigger system
	return tester.setupData.VM.ProcessIncomingText(text)
}