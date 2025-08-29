package scripting

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"twist/integration/setup"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting"
	"twist/internal/proxy/scripting/include"
	"twist/internal/proxy/scripting/parser"
	"twist/internal/proxy/scripting/vm"
)

// IntegrationTestResult captures the output from a script execution
type IntegrationTestResult struct {
	Output   []string
	Commands []string
	Error    error
}

// MockScript implements types.ScriptInterface for testing purposes
type MockScript struct {
	id       string
	filename string
	name     string
	running  bool
	system   bool
}

// NewMockScript creates a new test script with a unique ID and registers it in the database
func NewMockScript(name string, db database.Database) *MockScript {
	script := &MockScript{
		id:       "test_script_" + name, // Remove timestamp to make it deterministic
		filename: name + ".twx",
		name:     name,
		running:  true,
		system:   false,
	}

	// Register the script in the database to satisfy foreign key constraints
	if sqlDB := db.GetDB(); sqlDB != nil {
		query := `
		INSERT OR IGNORE INTO scripts (script_id, name, filename, version, is_running, is_system, loaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);`

		_, err := sqlDB.Exec(query, script.id, script.name, script.filename, 6, script.running, script.system, time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			// Log but don't fail - the script creation can still proceed
			// In a real implementation, we'd use proper logging
		}
	}

	return script
}

func (ts *MockScript) GetID() string {
	return ts.id
}

func (ts *MockScript) GetFilename() string {
	return ts.filename
}

func (ts *MockScript) GetName() string {
	return ts.name
}

func (ts *MockScript) IsRunning() bool {
	return ts.running
}

func (ts *MockScript) IsSystem() bool {
	return ts.system
}

func (ts *MockScript) Stop() error {
	ts.running = false
	return nil
}

// IntegrationScriptTester provides real integration testing for TWX scripts
type IntegrationScriptTester struct {
	setupData        *setup.IntegrationTestSetup
	t                *testing.T
	currentScript    *scripting.Script
	capturedOutput   []string
	capturedCommands []string
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
	// Parse the script using the same pipeline as the engine (including preprocessing)
	ast, err := tester.parseScriptWithPreprocessor(script)
	if err != nil {
		return &IntegrationTestResult{
			Output:   []string{},
			Commands: []string{},
			Error:    err,
		}
	}

	// Initialize captured data
	tester.capturedOutput = []string{}
	tester.capturedCommands = []string{}

	// Set up output handlers to use instance fields
	tester.setupData.VM.SetOutputHandler(func(text string) error {
		tester.capturedOutput = append(tester.capturedOutput, text)
		return nil
	})

	tester.setupData.VM.SetEchoHandler(func(text string) error {
		tester.capturedOutput = append(tester.capturedOutput, text)
		return nil
	})

	tester.setupData.VM.SetSendHandler(func(text string) error {
		tester.capturedCommands = append(tester.capturedCommands, text)
		return nil
	})

	// Create a test script instance for call stack persistence
	testScript := NewMockScript("integration_test", tester.setupData.DB)
	tester.currentScript = &scripting.Script{
		ID:       "integration_test",
		Filename: "test.ts",
		AST:      ast,
		VM:       tester.setupData.VM,
	}

	// Load and execute the script
	if err := tester.setupData.VM.LoadScript(ast, testScript); err != nil {
		return &IntegrationTestResult{
			Output:   append([]string{}, tester.capturedOutput...),
			Commands: append([]string{}, tester.capturedCommands...),
			Error:    err,
		}
	}

	if err := tester.setupData.VM.Execute(); err != nil {
		return &IntegrationTestResult{
			Output:   append([]string{}, tester.capturedOutput...),
			Commands: append([]string{}, tester.capturedCommands...),
			Error:    err,
		}
	}

	// For WAITFOR tests, we need to keep the handlers active
	// The result should be returned by the goroutine only when the script truly completes
	return &IntegrationTestResult{
		Output:   append([]string{}, tester.capturedOutput...),
		Commands: append([]string{}, tester.capturedCommands...),
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
		if strings.Contains(line, expectedText) {
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

// IsScriptWaitingForInput checks if the script is currently paused waiting for user input
func (tester *IntegrationScriptTester) IsScriptWaitingForInput() bool {
	if tester.currentScript == nil || tester.currentScript.VM == nil {
		return false
	}
	return tester.currentScript.VM.IsWaitingForInput()
}

// ProvideInput provides input to a script that is waiting for user input
func (tester *IntegrationScriptTester) ProvideInput(input string) error {
	if tester.currentScript == nil || tester.currentScript.VM == nil {
		return fmt.Errorf("no active script")
	}

	if !tester.currentScript.VM.IsWaitingForInput() {
		return fmt.Errorf("script is not waiting for input")
	}

	return tester.currentScript.VM.ResumeWithInput(input)
}

// ContinueExecution continues script execution after providing input or other pause
func (tester *IntegrationScriptTester) ContinueExecution() *IntegrationTestResult {
	if tester.currentScript == nil || tester.currentScript.VM == nil {
		return &IntegrationTestResult{
			Error: fmt.Errorf("no active script"),
		}
	}

	// Clear previous outputs and commands for this continuation
	tester.capturedOutput = []string{}
	tester.capturedCommands = []string{}

	// Continue execution
	err := tester.currentScript.VM.Execute()

	return &IntegrationTestResult{
		Output:   append([]string{}, tester.capturedOutput...),
		Commands: append([]string{}, tester.capturedCommands...),
		Error:    err,
	}
}

// ExecuteScriptAsync executes a script asynchronously and returns a channel for the result
// This is needed for WAITFOR tests that need to continue execution after network input
func (tester *IntegrationScriptTester) ExecuteScriptAsync(script string) (<-chan *IntegrationTestResult, error) {
	// Parse the script using the same pipeline as the engine (including preprocessing)
	ast, err := tester.parseScriptWithPreprocessor(script)
	if err != nil {
		return nil, err
	}

	// Create result channel
	resultChan := make(chan *IntegrationTestResult, 1)

	// Capture output and commands in shared slices
	var output []string
	var commands []string

	// Set up output handlers that will persist across the async execution
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

	// Create a test script instance
	testScript := NewMockScript("integration_test_async", tester.setupData.DB)

	// Start async execution
	go func() {
		defer func() {
			// Always send result when goroutine completes
			resultChan <- &IntegrationTestResult{
				Output:   output,
				Commands: commands,
				Error:    nil,
			}
		}()

		// Load and execute the script
		if err := tester.setupData.VM.LoadScript(ast, testScript); err != nil {
			return
		}

		// Execute until completion or waiting
		if err := tester.setupData.VM.Execute(); err != nil {
			return
		}

		// If we're waiting, the script will continue via ProcessIncomingText calls
		// The goroutine will remain alive until the script completes or times out
		for tester.setupData.VM.IsWaiting() {
			// Keep the goroutine alive while waiting
			// ProcessIncomingText will resume execution and eventually complete
			time.Sleep(1 * time.Millisecond)
		}
	}()

	return resultChan, nil
}

// IsWaiting returns true if the VM is currently waiting for input
func (tester *IntegrationScriptTester) IsWaiting() bool {
	// Access the VM's waiting state - we need to add this method to the VM interface
	return tester.setupData.VM.IsWaiting()
}

// SimulateNetworkInput simulates incoming network text for trigger processing
func (tester *IntegrationScriptTester) SimulateNetworkInput(text string) error {
	// Process the text through the VM's trigger system
	return tester.setupData.VM.ProcessIncomingText(text)
}

// parseScriptWithPreprocessor parses script source code using the same pipeline as the engine
// This mirrors the parseScriptWithBasePath method from the engine but without file path handling
func (tester *IntegrationScriptTester) parseScriptWithPreprocessor(source string) (*parser.ASTNode, error) {
	// Step 1: Preprocess script to expand IF/ELSE/END and WHILE/END macros
	lines := strings.Split(source, "\n")
	preprocessor := parser.NewPreprocessor()
	processedLines, err := preprocessor.ProcessScript(lines)
	if err != nil {
		return nil, fmt.Errorf("preprocessing error: %v", err)
	}

	// Rejoin the processed lines
	processedSource := strings.Join(processedLines, "\n")

	// Step 2: Create lexer (no line mappings for integration tests)
	lexer := parser.NewLexer(strings.NewReader(processedSource), nil)

	// Step 3: Tokenize
	tokens, err := lexer.TokenizeAll()
	if err != nil {
		return nil, fmt.Errorf("lexer error: %v", err)
	}

	// Step 4: Parse
	parserObj := parser.NewParser(tokens)
	ast, err := parserObj.Parse()
	if err != nil {
		return nil, fmt.Errorf("parser error: %v", err)
	}

	// Step 5: Process includes (minimal processing for integration tests)
	includeProcessor := include.NewIncludeProcessor(".")
	processedAST, err := includeProcessor.ProcessIncludes(ast)
	if err != nil {
		return nil, fmt.Errorf("include processing error: %v", err)
	}

	return processedAST, nil
}
