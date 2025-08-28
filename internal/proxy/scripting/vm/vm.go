package vm

import (
	"fmt"
	"strings"
	"time"
	"twist/internal/debug"
	"twist/internal/proxy/database"
	"twist/internal/proxy/scripting/manager"
	"twist/internal/proxy/scripting/parser"
	"twist/internal/proxy/scripting/types"
)

// VirtualMachine executes TWX scripts using a modular architecture
type VirtualMachine struct {
	// Execution components
	state     *VMState
	callStack *CallStack
	variables *VariableManager
	execution *ExecutionEngine

	// Script context
	script        types.ScriptInterface
	gameInterface types.GameInterface
	scriptManager *manager.ScriptManager

	// Commands and triggers
	commands map[string]*types.CommandDef
	triggers map[string]types.TriggerInterface

	// Output handlers
	outputHandler func(string) error
	echoHandler   func(string) error
	sendHandler   func(string) error

	// Timer system
	timerStart time.Time
	timerValue float64

	// Error tracking
	lastError error

	// Input collection state
	waitingForInput    bool
	pendingInputPrompt string
	pendingInputResult string
	justResumed        bool

	// Trigger processing state (for TWX compatibility)
	processingTrigger bool
}

// NewVirtualMachine creates a new virtual machine
func NewVirtualMachine(gameInterface types.GameInterface) *VirtualMachine {
	vm := &VirtualMachine{
		state:         NewVMState(),
		callStack:     NewCallStack(),
		variables:     NewVariableManager(gameInterface),
		commands:      make(map[string]*types.CommandDef),
		triggers:      make(map[string]types.TriggerInterface),
		gameInterface: gameInterface,
	}

	// Initialize execution engine
	vm.execution = NewExecutionEngine(vm)

	// System constants are now provided by GameInterface

	// Initialize script manager
	if db := gameInterface.GetDatabase(); db != nil {
		if dbInterface, ok := db.(database.Database); ok {
			vm.scriptManager = manager.NewScriptManager(dbInterface)
			// Restore scripts from database
			vm.scriptManager.RestoreFromDatabase()
		}
	}

	// Initialize timer system
	vm.timerStart = time.Now()
	vm.timerValue = 0

	// Register built-in commands
	vm.registerCommands()

	return vm
}

// LoadScript loads a parsed script for execution
func (vm *VirtualMachine) LoadScript(ast *parser.ASTNode, script types.ScriptInterface) error {
	vm.execution.LoadAST(ast)
	vm.script = script
	vm.state.SetRunning()
	vm.state.Position = 0

	// Restore call stack from database if script is provided
	if script != nil {
		if err := vm.restoreCallStack(script.GetID()); err != nil {
			// Failed to restore call stack - just start with empty call stack
		}
	}

	return nil
}

// Execute runs the script execution loop
func (vm *VirtualMachine) Execute() error {
	scriptName := "unknown"
	if vm.script != nil {
		scriptName = vm.script.GetName()
	}
	debug.Info("VM.Execute: starting execution loop", "script", scriptName)

	for vm.state.IsRunning() && !vm.state.IsWaiting() {
		debug.Info("VM.Execute: executing step", "script", scriptName, "position", vm.state.Position)

		if err := vm.execution.ExecuteStep(); err != nil {
			debug.Info("VM.Execute: ExecuteStep returned error", "script", scriptName, "error", err)
			vm.lastError = err
			return err
		}

		// Handle pause state
		if vm.state.IsPaused() {
			debug.Info("VM.Execute: detected PAUSED state", "script", scriptName, "waitingForInput", vm.waitingForInput)
			// If paused waiting for input, return control to caller
			// The script manager will handle input collection and resume
			if vm.waitingForInput {
				debug.Info("VM.Execute: RETURNING from Execute() - waiting for input", "script", scriptName)
				return nil
			}

			debug.Info("VM.Execute: continuing execution after pause (not waiting for input)", "script", scriptName)
			// For other pause types (like regular pause command), continue automatically
			// This maintains backwards compatibility with existing test behavior
			vm.state.SetRunning()
			vm.state.Position++ // Advance past the pause command
		}

	}
	debug.Info("VM.Execute: execution loop finished", "script", scriptName)
	return nil
}

// Output handlers
func (vm *VirtualMachine) SetOutputHandler(handler func(string) error) {
	vm.outputHandler = handler
}

func (vm *VirtualMachine) SetEchoHandler(handler func(string) error) {
	vm.echoHandler = handler
}

func (vm *VirtualMachine) SetSendHandler(handler func(string) error) {
	vm.sendHandler = handler
}

// Variable operations
func (vm *VirtualMachine) GetVariable(name string) *types.Value {
	return vm.variables.Get(name)
}

func (vm *VirtualMachine) SetVariable(name string, value *types.Value) {
	vm.variables.Set(name, value)
}

// VarParam methods for Pascal-compatible array support
func (vm *VirtualMachine) GetVarParam(name string) *types.VarParam {
	return vm.variables.GetVarParam(name)
}

func (vm *VirtualMachine) SetVarParam(name string, varParam *types.VarParam) {
	vm.variables.SetVarParam(name, varParam)
}

// Control flow operations
func (vm *VirtualMachine) Goto(label string) error {
	vm.state.SetJumpTarget(label)
	return nil
}

func (vm *VirtualMachine) Gosub(label string) error {
	frame := NewStackFrame(label, vm.state.Position, vm.state.Position+1)
	vm.callStack.Push(frame)
	vm.state.SetJumpTarget(label)

	// Save call stack to database for persistence across VM instances
	if vm.script != nil {
		if err := vm.saveCallStack(vm.script.GetID()); err != nil {
			// Failed to save call stack - continue with GOSUB
		}
	}

	return nil
}

func (vm *VirtualMachine) Return() error {
	frame, err := vm.callStack.Pop()
	if err != nil {
		return vm.Error("Return without gosub")
	}
	// Set position to ReturnAddr - 1 because the execution loop will increment it
	vm.state.Position = frame.ReturnAddr - 1

	// Save updated call stack to database for persistence across VM instances
	if vm.script != nil {
		if err := vm.saveCallStack(vm.script.GetID()); err != nil {
			// Failed to save call stack - continue with RETURN
		}
	}

	return nil
}

// GotoAndExecuteSync jumps to a label and executes it synchronously (for triggers)
// This is needed for TWX compatibility where triggers execute immediately
func (vm *VirtualMachine) GotoAndExecuteSync(label string) error {
	scriptName := "unknown"
	if vm.script != nil {
		scriptName = vm.script.GetName()
	}
	debug.Info("VM.GotoAndExecuteSync: STARTING synchronous execution of trigger handler", "script", scriptName, "label", label)

	// Save current execution state
	savedPosition := vm.state.Position
	savedRunning := vm.state.IsRunning()

	// Jump to the label immediately (no delayed jump)
	newPos := vm.execution.FindLabel(label)
	if newPos == -1 {
		return fmt.Errorf("label not found: %s", label)
	}
	vm.state.Position = newPos

	// Execute until we hit a pause, halt, or return
	for vm.state.IsRunning() && !vm.state.IsWaiting() && !vm.state.IsPaused() {
		if err := vm.execution.ExecuteStep(); err != nil {
			debug.Info("VM.GotoAndExecuteSync: ExecuteStep returned error", "script", scriptName, "error", err)
			// Restore state on error
			vm.state.Position = savedPosition
			if savedRunning {
				vm.state.SetRunning()
			}
			return err
		}

		// If we hit a pause or getinput, break and let the caller handle it
		if vm.state.IsPaused() || vm.waitingForInput {
			debug.Info("VM.GotoAndExecuteSync: trigger handler paused or waiting for input", "script", scriptName)
			break
		}
	}

	// If the handler didn't end naturally, restore execution state
	// (In TWX, trigger handlers typically end with a return or implicit return)
	if vm.state.IsRunning() && !vm.state.IsPaused() && !vm.waitingForInput {
		debug.Info("VM.GotoAndExecuteSync: restoring execution position", "script", scriptName, "position", savedPosition)
		vm.state.Position = savedPosition
		if savedRunning {
			vm.state.SetRunning()
		}
	}

	debug.Info("VM.GotoAndExecuteSync: COMPLETED synchronous execution of trigger handler", "script", scriptName, "label", label)
	return nil
}

// State control
func (vm *VirtualMachine) Halt() error {
	vm.state.SetHalted()
	return nil
}

func (vm *VirtualMachine) Pause() error {
	vm.state.SetPaused()
	return types.ErrScriptPaused
}

// Communication
func (vm *VirtualMachine) Echo(message string) error {
	if vm.echoHandler != nil {
		return vm.echoHandler(message)
	}
	return nil
}

func (vm *VirtualMachine) ClientMessage(message string) error {
	if vm.outputHandler != nil {
		return vm.outputHandler(message)
	}
	return nil
}

func (vm *VirtualMachine) Send(data string) error {
	if vm.sendHandler != nil {
		return vm.sendHandler(data)
	} else {
	}
	return nil
}

func (vm *VirtualMachine) WaitFor(text string) error {
	scriptName := "unknown"
	if vm.script != nil {
		scriptName = vm.script.GetName()
	}
	debug.Info("VM.WaitFor: waiting for trigger", "script", scriptName, "trigger", text)
	vm.state.SetWaiting(text)
	return nil
}

// Input handling
func (vm *VirtualMachine) GetInput(prompt string) (string, error) {
	scriptName := "unknown"
	if vm.script != nil {
		scriptName = vm.script.GetName()
	}
	debug.Info("VM.GetInput: initiating input", "script", scriptName, "prompt", prompt)

	// Display the prompt (matching TWX behavior - prompt on its own line, cursor stays at end)
	if err := vm.Echo("\r\n" + prompt + " "); err != nil {
		return "", err
	}

	// Set up script input collection state
	vm.pendingInputPrompt = prompt
	vm.pendingInputResult = ""
	vm.waitingForInput = true

	debug.Info("VM.GetInput: set waitingForInput=true", "script", scriptName, "pendingInputPrompt", prompt)

	// Pause script execution - this will cause the Run loop to exit
	// and return control to the caller (matching TWX caPause behavior)
	vm.state.SetPaused()

	debug.Info("VM.GetInput: set state to PAUSED", "script", scriptName)

	// The script manager should detect this paused state and initiate
	// input collection via the menu system's InputCollector
	// This matches TWX's BeginScriptInput() integration

	// This returns immediately with empty result - the actual input
	// will be provided later via ResumeWithInput()
	return vm.pendingInputResult, nil
}

// IsWaitingForInput returns true if the VM is paused waiting for user input
func (vm *VirtualMachine) IsWaitingForInput() bool {
	return vm.waitingForInput && vm.state.IsPaused()
}

// GetPendingInputPrompt returns the prompt for pending input
func (vm *VirtualMachine) GetPendingInputPrompt() string {
	return vm.pendingInputPrompt
}

// GetPendingInputResult returns the result of pending input
func (vm *VirtualMachine) GetPendingInputResult() string {
	return vm.pendingInputResult
}

// JustResumedFromInput returns true if we just resumed from input and haven't processed it yet
func (vm *VirtualMachine) JustResumedFromInput() bool {
	return vm.justResumed
}

// ResumeWithInput provides the input value and resumes script execution
func (vm *VirtualMachine) ResumeWithInput(input string) error {
	scriptName := "unknown"
	if vm.script != nil {
		scriptName = vm.script.GetName()
	}

	if !vm.waitingForInput {
		debug.Info("VM.ResumeWithInput: ERROR - not waiting for input!", "script", scriptName)
		return fmt.Errorf("VM is not waiting for input")
	}

	debug.Info("VM.ResumeWithInput: resuming with input", "script", scriptName, "input", input)

	vm.pendingInputResult = input
	vm.waitingForInput = false
	vm.pendingInputPrompt = "" // Clear prompt since input has been provided
	vm.justResumed = true      // Flag to indicate we just resumed with input

	// Resume script execution
	vm.state.SetRunning()

	debug.Info("VM.ResumeWithInput: set state to RUNNING", "script", scriptName)

	return nil
}

// ClearPendingInput clears the pending input state after processing
func (vm *VirtualMachine) ClearPendingInput() {
	vm.pendingInputResult = ""
	vm.pendingInputPrompt = ""
	vm.waitingForInput = false
	vm.justResumed = false
}

// GetState returns the VM's execution state
func (vm *VirtualMachine) GetState() *VMState {
	return vm.state
}

// Interface implementations
func (vm *VirtualMachine) GetGameInterface() types.GameInterface {
	return vm.gameInterface
}

func (vm *VirtualMachine) GetCurrentScript() types.ScriptInterface {
	return vm.script
}

func (vm *VirtualMachine) GetCurrentLine() int {
	// TODO: Implement proper line number tracking
	return 0
}

func (vm *VirtualMachine) GetScriptManager() interface{} {
	return vm.scriptManager
}

func (vm *VirtualMachine) LoadAdditionalScript(filename string) (types.ScriptInterface, error) {
	if vm.scriptManager == nil {
		return nil, fmt.Errorf("script manager not initialized")
	}

	// Load script through the script manager
	scriptInfo, err := vm.scriptManager.LoadScript(filename, false)
	if err != nil {
		return nil, err
	}

	// TODO: Actually parse and compile the script file
	// For now, just return the script info which implements ScriptInterface
	return scriptInfo, nil
}

func (vm *VirtualMachine) StopScript(scriptID string) error {
	if vm.scriptManager == nil {
		return fmt.Errorf("script manager not initialized")
	}

	// Try to stop by ID first, then by name
	err := vm.scriptManager.StopScript(scriptID)
	if err != nil {
		// If ID didn't work, try as a name
		return vm.scriptManager.StopScriptByName(scriptID)
	}

	return nil
}

// Trigger management
func (vm *VirtualMachine) SetTrigger(trigger types.TriggerInterface) error {
	vm.triggers[trigger.GetID()] = trigger
	return nil
}

func (vm *VirtualMachine) KillTrigger(triggerID string) error {
	delete(vm.triggers, triggerID)
	return nil
}

func (vm *VirtualMachine) KillAllTriggers() {
	vm.triggers = make(map[string]types.TriggerInterface)
}

func (vm *VirtualMachine) GetTriggerCount() int {
	return len(vm.triggers)
}

func (vm *VirtualMachine) GetActiveTriggersCount() int {
	count := 0
	for _, trigger := range vm.triggers {
		if trigger.IsActive() {
			count++
		}
	}
	return count
}

// Text processing
func (vm *VirtualMachine) ProcessTriggers(text string) error {
	for _, trigger := range vm.triggers {
		if trigger.IsActive() && trigger.Matches(text) {
			if err := trigger.Execute(vm); err != nil {
				return err
			}
		}
	}
	return nil
}

func (vm *VirtualMachine) ProcessIncomingText(text string) error {

	// Process triggers first
	if err := vm.ProcessTriggers(text); err != nil {
		return err
	}

	// Check if we're waiting for specific text (like TWX WaitFor)
	if vm.state.IsWaiting() && vm.state.WaitText != "" {
		scriptName := "unknown"
		if vm.script != nil {
			scriptName = vm.script.GetName()
		}
		debug.Info("VM.ProcessIncomingText: checking if text contains waitfor trigger", "script", scriptName, "text", text, "waitforTrigger", vm.state.WaitText)
		// Use substring matching like TWX does with Pos(FWaitText, Text) > 0
		if strings.Contains(text, vm.state.WaitText) {
			debug.Info("VM.ProcessIncomingText: TRIGGER MATCHED! Continuing script execution", "script", scriptName)
			vm.state.ClearWait()
			// Resume execution - the position was already advanced by ExecuteStep
			return vm.Execute()
		} else {
			debug.Info("VM.ProcessIncomingText: trigger not found, still waiting", "script", scriptName)
		}
	}

	return nil
}

// Error handling
func (vm *VirtualMachine) Error(message string) error {
	vm.state.SetError(message)
	vm.lastError = &types.VMError{Message: message}
	return vm.lastError
}

// IsWaiting returns true if the VM is currently waiting for input (for testing)
func (vm *VirtualMachine) IsWaiting() bool {
	return vm.state.IsWaiting()
}

// Processing filters (for testing compatibility)
func (vm *VirtualMachine) ProcessInput(filter string) error {
	// In a real implementation, this would set up input processing filters
	// For now, just return success for compatibility with tests
	return nil
}

func (vm *VirtualMachine) ProcessOutput(filter string) error {
	// In a real implementation, this would set up output processing filters
	// For now, just return success for compatibility with tests
	return nil
}

// GetCurrentPosition returns the current execution position for debugging
func (vm *VirtualMachine) GetCurrentPosition() int {
	return vm.state.Position
}

// EvaluateExpression evaluates a string expression and returns its value
func (vm *VirtualMachine) EvaluateExpression(expression string) (*types.Value, error) {
	// Unescape any escaped quotes in the expression
	unescapedExpression := strings.ReplaceAll(expression, "\\\"", "\"")

	// Parse the expression string into an AST node
	lexer := parser.NewLexer(strings.NewReader(unescapedExpression))
	tokens, err := lexer.TokenizeAll()
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize expression: %v", err)
	}

	parserObj := parser.NewParser(tokens)
	// Parse as a single expression
	expr, err := parserObj.ParseExpression()
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression: %v", err)
	}

	// Use the execution engine to evaluate the expression
	if vm.execution == nil {
		vm.execution = NewExecutionEngine(vm)
	}

	return vm.execution.evaluateExpression(expr)
}

// Command registration - this is used by the command registry
func (vm *VirtualMachine) registerCommand(name string, minParams, maxParams int, paramTypes []types.ParameterType, handler types.CommandHandler) {
	vm.commands[name] = &types.CommandDef{
		Name:       name,
		MinParams:  minParams,
		MaxParams:  maxParams,
		ParamTypes: paramTypes,
		Handler:    handler,
	}
}

// Variable access methods for menu system
func (vm *VirtualMachine) GetAllVariables() map[string]*types.Value {
	if vm.variables == nil {
		return make(map[string]*types.Value)
	}
	return vm.variables.GetAll()
}

func (vm *VirtualMachine) GetVariableNames() []string {
	if vm.variables == nil {
		return []string{}
	}
	return vm.variables.GetNames()
}

func (vm *VirtualMachine) GetVariableCount() int {
	if vm.variables == nil {
		return 0
	}
	return vm.variables.Count()
}

// Call stack persistence methods for TWX compatibility
func (vm *VirtualMachine) saveCallStack(scriptID string) error {
	if vm.gameInterface == nil {
		return nil // No database available
	}

	db := vm.gameInterface.GetDatabase()
	if db == nil {
		return nil // No database available
	}

	dbInterface, ok := db.(database.Database)
	if !ok {
		return nil // Not the right database interface
	}

	// Clear existing call stack for this script
	deleteQuery := `DELETE FROM script_call_stack WHERE script_id = ?;`
	if _, err := dbInterface.GetDB().Exec(deleteQuery, scriptID); err != nil {
		return fmt.Errorf("failed to clear call stack: %w", err)
	}

	// Save current call stack frames
	frames := vm.callStack.GetFrames()
	if len(frames) == 0 {
		return nil // Nothing to save
	}

	insertQuery := `
	INSERT INTO script_call_stack (script_id, frame_index, label, position, return_addr)
	VALUES (?, ?, ?, ?, ?);`

	for i, frame := range frames {
		_, err := dbInterface.GetDB().Exec(insertQuery, scriptID, i, frame.Label, frame.Position, frame.ReturnAddr)
		if err != nil {
			return fmt.Errorf("failed to save call stack frame %d: %w", i, err)
		}
	}

	return nil
}

func (vm *VirtualMachine) restoreCallStack(scriptID string) error {
	if vm.gameInterface == nil {
		return nil // No database available
	}

	db := vm.gameInterface.GetDatabase()
	if db == nil {
		return nil // No database available
	}

	dbInterface, ok := db.(database.Database)
	if !ok {
		return nil // Not the right database interface
	}

	// Clear current call stack
	vm.callStack.Clear()

	// Load call stack frames from database, ordered by frame_index
	query := `
	SELECT label, position, return_addr
	FROM script_call_stack
	WHERE script_id = ?
	ORDER BY frame_index;`

	rows, err := dbInterface.GetDB().Query(query, scriptID)
	if err != nil {
		return fmt.Errorf("failed to query call stack: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var label string
		var position, returnAddr int

		if err := rows.Scan(&label, &position, &returnAddr); err != nil {
			return fmt.Errorf("failed to scan call stack frame: %w", err)
		}

		frame := NewStackFrame(label, position, returnAddr)
		vm.callStack.Push(frame)
	}

	return rows.Err()
}
