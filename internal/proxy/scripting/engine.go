package scripting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"twist/internal/ansi"
	"twist/internal/debug"
	"twist/internal/proxy/interfaces"
	"twist/internal/proxy/scripting/include"
	"twist/internal/proxy/scripting/parser"
	"twist/internal/proxy/scripting/triggers"
	"twist/internal/proxy/scripting/types"
	"twist/internal/proxy/scripting/vm"
)

// Script represents a loaded script
type Script struct {
	ID       string
	Filename string
	Name     string
	AST      *parser.ASTNode
	VM       *vm.VirtualMachine
	Running  bool
	System   bool
}

// GetID implements ScriptInterface
func (s *Script) GetID() string {
	return s.ID
}

// GetFilename implements ScriptInterface
func (s *Script) GetFilename() string {
	return s.Filename
}

// IsRunning implements interfaces.ScriptInfo
func (s *Script) IsRunning() bool {
	return s.Running
}

// GetName implements ScriptInterface
func (s *Script) GetName() string {
	return s.Name
}

// IsSystem implements ScriptInterface
func (s *Script) IsSystem() bool {
	return s.System
}

// Stop implements ScriptInterface
func (s *Script) Stop() error {
	s.Running = false
	if s.VM != nil {
		return s.VM.Halt()
	}
	return nil
}

// Engine is the main scripting engine
type Engine struct {
	scriptsRef     atomic.Pointer[map[string]*Script]
	gameInterface  types.GameInterface
	triggerManager *triggers.Manager
	mutex          sync.Mutex // Only needed for writes now
	nextScriptID   atomic.Int32

	// ANSI stripper for streaming text processing
	ansiStripper *ansi.StreamingStripper

	// Event handlers
	outputHandler func(string) error
	echoHandler   func(string) error
	sendHandler   func(string) error
}

// NewEngine creates a new scripting engine
func NewEngine(gameInterface types.GameInterface) *Engine {
	engine := &Engine{
		gameInterface: gameInterface,
		ansiStripper:  ansi.NewStreamingStripper(),
	}
	engine.nextScriptID.Store(1)

	// Initialize empty scripts map
	initialScripts := make(map[string]*Script)
	engine.scriptsRef.Store(&initialScripts)

	// Create a dummy VM for the trigger manager
	dummyVM := vm.NewVirtualMachine(gameInterface)
	engine.triggerManager = triggers.NewManager(dummyVM)

	return engine
}

// getScripts returns a copy of the current scripts map for read operations
func (e *Engine) getScripts() map[string]*Script {
	return *e.scriptsRef.Load()
}

// updateScripts performs a copy-on-write update to the scripts map
func (e *Engine) updateScripts(updateFn func(map[string]*Script) map[string]*Script) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Load current scripts
	currentScripts := *e.scriptsRef.Load()

	// Create a copy and apply update
	newScripts := updateFn(currentScripts)

	// Store the new map
	e.scriptsRef.Store(&newScripts)
}

// SetOutputHandler sets the handler for output messages
func (e *Engine) SetOutputHandler(handler func(string) error) {
	e.outputHandler = handler
}

// SetEchoHandler sets the handler for echo messages
func (e *Engine) SetEchoHandler(handler func(string) error) {
	e.echoHandler = handler
}

// SetSendHandler sets the handler for send messages
func (e *Engine) SetSendHandler(handler func(string) error) {
	e.mutex.Lock()
	e.sendHandler = handler
	e.mutex.Unlock()

	// Update all existing script VMs with the new sendHandler (lockless read)
	scripts := e.getScripts()
	for _, script := range scripts {
		if script.VM != nil {
			script.VM.SetSendHandler(handler)
		}
	}
}

// LoadScript loads a script from a file
func (e *Engine) LoadScript(filename string) (*Script, error) {
	// Read script file
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file %s: %v", filename, err)
	}
	previewLen := 200
	if len(content) < previewLen {
		previewLen = len(content)
	}

	// Parse script with proper base path for includes
	basePath := filepath.Dir(filename)
	ast, err := e.parseScriptWithBasePath(string(content), basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script %s: %v", filename, err)
	}

	var script *Script
	e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
		// Create script object
		scriptID := e.generateScriptID()
		scriptName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

		script = &Script{
			ID:       scriptID,
			Filename: filename,
			Name:     scriptName,
			AST:      ast,
			Running:  false,
			System:   false,
		}

		// Create VM for this script
		scriptVM := vm.NewVirtualMachine(e.gameInterface)
		scriptVM.SetOutputHandler(e.outputHandler)
		scriptVM.SetEchoHandler(e.echoHandler)
		scriptVM.SetSendHandler(e.sendHandler)
		script.VM = scriptVM

		// Copy current scripts and add new one
		newScripts := make(map[string]*Script, len(currentScripts)+1)
		for k, v := range currentScripts {
			newScripts[k] = v
		}
		newScripts[scriptID] = script

		return newScripts
	})

	return script, nil
}

// LoadScriptFromString loads a script from a string
func (e *Engine) LoadScriptFromString(content, name string) (*Script, error) {
	// Parse script
	ast, err := e.parseScript(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script %s: %v", name, err)
	}

	var script *Script
	e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
		// Create script object
		scriptID := e.generateScriptID()

		script = &Script{
			ID:       scriptID,
			Filename: "",
			Name:     name,
			AST:      ast,
			Running:  false,
			System:   false,
		}

		// Create VM for this script
		scriptVM := vm.NewVirtualMachine(e.gameInterface)
		scriptVM.SetOutputHandler(e.outputHandler)
		scriptVM.SetEchoHandler(e.echoHandler)
		scriptVM.SetSendHandler(e.sendHandler)
		script.VM = scriptVM

		// Copy current scripts and add new one
		newScripts := make(map[string]*Script, len(currentScripts)+1)
		for k, v := range currentScripts {
			newScripts[k] = v
		}
		newScripts[scriptID] = script

		return newScripts
	})

	return script, nil
}

// RunScript starts executing a script
func (e *Engine) RunScript(scriptID string) error {
	scripts := e.getScripts()
	script, exists := scripts[scriptID]

	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	if script.Running {
		return fmt.Errorf("script %s is already running", scriptID)
	}

	// Load script into VM
	if err := script.VM.LoadScript(script.AST, script); err != nil {
		return fmt.Errorf("failed to load script into VM: %v", err)
	}

	// Update script state in copy-on-write manner
	e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
		// Copy the map
		newScripts := make(map[string]*Script, len(currentScripts))
		for k, v := range currentScripts {
			newScripts[k] = v
		}
		// Update the running state
		newScripts[scriptID].Running = true
		return newScripts
	})

	// Execute script once - TWX style single execution
	// Script will pause on waitfor commands and resume when text matches
	err := script.VM.Execute()
	if err != nil {
		// Mark script as not running on error
		e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
			newScripts := make(map[string]*Script, len(currentScripts))
			for k, v := range currentScripts {
				newScripts[k] = v
			}
			if _, exists := newScripts[scriptID]; exists {
				newScripts[scriptID].Running = false
			}
			return newScripts
		})
		if e.outputHandler != nil {
			e.outputHandler(fmt.Sprintf("Script error in %s: %v", script.Name, err))
		}
		return err
	}

	// Check if script completed (halted)
	state := script.VM.GetState()
	if state.IsHalted() {
		e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
			newScripts := make(map[string]*Script, len(currentScripts))
			for k, v := range currentScripts {
				newScripts[k] = v
			}
			if _, exists := newScripts[scriptID]; exists {
				newScripts[scriptID].Running = false
			}
			return newScripts
		})
	}

	return nil
}

// ResumeScriptWithInput resumes a paused script with input
func (e *Engine) ResumeScriptWithInput(scriptID string, input string) error {
	scripts := e.getScripts()
	script, exists := scripts[scriptID]

	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	if !script.VM.IsWaitingForInput() {
		return fmt.Errorf("script %s is not waiting for input", scriptID)
	}

	// Resume the VM with the input
	if err := script.VM.ResumeWithInput(input); err != nil {
		return err
	}

	// Continue script execution - TWX style single execution
	// Script will execute until next waitfor, getinput, or completion
	err := script.VM.Execute()
	if err != nil {
		// Mark script as not running on error
		e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
			newScripts := make(map[string]*Script, len(currentScripts))
			for k, v := range currentScripts {
				newScripts[k] = v
			}
			if _, exists := newScripts[scriptID]; exists {
				newScripts[scriptID].Running = false
			}
			return newScripts
		})
		if e.outputHandler != nil {
			e.outputHandler(fmt.Sprintf("Script error in %s: %v", script.Name, err))
		}
		return err
	}

	// Check if script completed (halted)
	state := script.VM.GetState()
	if state.IsHalted() {
		e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
			newScripts := make(map[string]*Script, len(currentScripts))
			for k, v := range currentScripts {
				newScripts[k] = v
			}
			if _, exists := newScripts[scriptID]; exists {
				newScripts[scriptID].Running = false
			}
			return newScripts
		})
	}

	return nil
}

// RunScriptSync executes a script synchronously and returns any execution error
func (e *Engine) RunScriptSync(scriptID string) error {
	scripts := e.getScripts()
	script, exists := scripts[scriptID]

	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	if script.Running {
		return fmt.Errorf("script %s is already running", scriptID)
	}

	// Load script into VM
	if err := script.VM.LoadScript(script.AST, script); err != nil {
		return fmt.Errorf("failed to load script into VM: %v", err)
	}

	// Update script state in copy-on-write manner
	e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
		newScripts := make(map[string]*Script, len(currentScripts))
		for k, v := range currentScripts {
			newScripts[k] = v
		}
		newScripts[scriptID].Running = true
		return newScripts
	})

	defer func() {
		e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
			newScripts := make(map[string]*Script, len(currentScripts))
			for k, v := range currentScripts {
				newScripts[k] = v
			}
			if _, exists := newScripts[scriptID]; exists {
				newScripts[scriptID].Running = false
			}
			return newScripts
		})
	}()

	// Execute script synchronously
	return script.VM.Execute()
}

// StopScript stops a running script
func (e *Engine) StopScript(scriptID string) error {
	scripts := e.getScripts()
	script, exists := scripts[scriptID]

	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	err := script.Stop()
	if err == nil {
		// Notify about script termination
		e.onScriptTerminated(scriptID)
	}

	return err
}

// StopAllScripts stops all running scripts
func (e *Engine) StopAllScripts() error {
	scriptsMap := e.getScripts()
	scripts := make([]*Script, 0, len(scriptsMap))
	for _, script := range scriptsMap {
		if script.Running {
			scripts = append(scripts, script)
		}
	}

	for _, script := range scripts {
		if err := script.Stop(); err != nil {
			return err
		}
		// Notify about script termination
		e.onScriptTerminated(script.ID)
	}

	return nil
}

// UnloadScript removes a script from memory
func (e *Engine) UnloadScript(scriptID string) error {
	var script *Script
	var scriptExists bool

	e.updateScripts(func(currentScripts map[string]*Script) map[string]*Script {
		var exists bool
		script, exists = currentScripts[scriptID]
		scriptExists = exists

		if !exists {
			return currentScripts // No change
		}

		// Copy current scripts excluding the one to remove
		newScripts := make(map[string]*Script, len(currentScripts)-1)
		for k, v := range currentScripts {
			if k != scriptID {
				newScripts[k] = v
			}
		}

		return newScripts
	})

	if !scriptExists {
		return fmt.Errorf("script not found: %s", scriptID)
	}

	if script.Running {
		if err := script.Stop(); err != nil {
			return err
		}
	}

	return nil
}

// GetScript returns a script by ID
func (e *Engine) GetScript(scriptID string) (*Script, error) {
	scripts := e.getScripts()
	script, exists := scripts[scriptID]
	if !exists {
		return nil, fmt.Errorf("script not found: %s", scriptID)
	}

	return script, nil
}

// ListScripts returns all loaded scripts
func (e *Engine) ListScripts() map[string]*Script {
	scripts := e.getScripts()
	result := make(map[string]*Script)
	for k, v := range scripts {
		result[k] = v
	}
	return result
}

// GetRunningScripts returns all running scripts (implements interfaces.ScriptEngine)
func (e *Engine) GetRunningScripts() []interfaces.ScriptInfo {
	scripts := e.getScripts()
	result := make([]interfaces.ScriptInfo, 0)
	for _, script := range scripts {
		if script.Running {
			result = append(result, script)
		}
	}
	return result
}

// GetAllScripts returns all loaded scripts regardless of running state (implements interfaces.ScriptEngine)
func (e *Engine) GetAllScripts() []interfaces.ScriptInfo {
	scripts := e.getScripts()
	result := make([]interfaces.ScriptInfo, 0)
	for _, script := range scripts {
		result = append(result, script)
	}
	return result
}

// GetRunningScriptsInternal returns all running scripts as concrete types for internal use
func (e *Engine) GetRunningScriptsInternal() []*Script {
	scripts := e.getScripts()
	result := make([]*Script, 0)
	for _, script := range scripts {
		if script.Running {
			result = append(result, script)
		}
	}
	return result
}

// ProcessText processes incoming text through triggers
func (e *Engine) ProcessText(text string) error {

	// Process global triggers first
	if err := e.triggerManager.ProcessText(text); err != nil {
		return err
	}

	// Strip ANSI escape sequences using streaming stripper to handle chunks properly
	// This ensures waitfor triggers match properly against clean text
	strippedText := e.ansiStripper.StripChunk(text)

	// Forward stripped text to all running script VMs for waitfor processing (lockless!)
	scripts := e.getScripts()
	scriptCount := 0
	for _, script := range scripts {
		if script.Running && script.VM != nil {
			scriptCount++
			if err := script.VM.ProcessIncomingText(strippedText); err != nil {
			} else {
			}
		}
	}

	return nil
}

// ProcessTextLine processes incoming text line through triggers
func (e *Engine) ProcessTextLine(line string) error {
	return e.triggerManager.ProcessTextLine(line)
}

// ProcessTextOut processes outgoing text through triggers
func (e *Engine) ProcessTextOut(text string) error {
	return e.triggerManager.ProcessTextOut(text)
}

// ProcessEvent processes system events through triggers
func (e *Engine) ProcessEvent(eventName string) error {
	return e.triggerManager.ProcessEvent(eventName)
}

// ActivateTriggers activates script triggers (mirrors Pascal TWXInterpreter.ActivateTriggers)
func (e *Engine) ActivateTriggers() error {
	// In Pascal TWX, ActivateTriggers processes delay triggers and reactivates disabled triggers
	// For now, we'll process delay triggers which is the main functionality
	return e.triggerManager.ProcessDelayTriggers()
}

// ProcessAutoText processes auto text events (mirrors Pascal TWXInterpreter.AutoTextEvent)
func (e *Engine) ProcessAutoText(text string) error {
	// Auto text events are similar to regular text events but for automated text
	// For now, delegate to regular text processing
	return e.ProcessText(text)
}

// UpdateCurrentLine updates the CURRENTLINE system constant (TWX compatibility)
func (e *Engine) UpdateCurrentLine(text string) error {
	// Update CURRENTLINE through the system constants
	if e.gameInterface != nil {
		if systemConstants := e.gameInterface.GetSystemConstants(); systemConstants != nil {
			systemConstants.UpdateCurrentLine(text)
		}
	}
	return nil
}

// GetTriggerManager returns the trigger manager
func (e *Engine) GetTriggerManager() *triggers.Manager {
	return e.triggerManager
}

// parseScript parses script source code into an AST
func (e *Engine) parseScript(source string) (*parser.ASTNode, error) {
	return e.parseScriptWithBasePath(source, ".")
}

// parseScriptWithBasePath parses script source code into an AST with a specific base path for includes
func (e *Engine) parseScriptWithBasePath(source, basePath string) (*parser.ASTNode, error) {
	// Step 1: Preprocess script to expand IF/ELSE/END and WHILE/END macros
	lines := strings.Split(source, "\n")
	preprocessor := parser.NewPreprocessor()
	processedLines, err := preprocessor.ProcessScript(lines)
	if err != nil {
		return nil, fmt.Errorf("preprocessing error: %v", err)
	}

	// Rejoin the processed lines
	processedSource := strings.Join(processedLines, "\n")

	// Step 2: Create lexer
	lexer := parser.NewLexer(strings.NewReader(processedSource))

	// Step 3: Tokenize
	tokens, err := lexer.TokenizeAll()
	if err != nil {
		return nil, fmt.Errorf("lexer error: %v", err)
	}

	// Debug: Show first few tokens
	for i, token := range tokens {
		if i >= 10 { // Show first 10 tokens
			break
		}
		debug.Info("Token", "index", i, "token", token)
	}

	// Step 4: Parse
	parser := parser.NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parser error: %v", err)
	}

	// Step 5: Process includes
	includeProcessor := include.NewIncludeProcessor(basePath)
	processedAST, err := includeProcessor.ProcessIncludes(ast)
	if err != nil {
		return nil, fmt.Errorf("include processing error: %v", err)
	}

	return processedAST, nil
}

// generateScriptID generates a unique script ID
func (e *Engine) generateScriptID() string {
	id := e.nextScriptID.Add(1)
	return fmt.Sprintf("script_%d", id)
}

// ExecuteScriptString executes a script string directly
func (e *Engine) ExecuteScriptString(source, name string) error {
	script, err := e.LoadScriptFromString(source, name)
	if err != nil {
		return err
	}

	defer e.UnloadScript(script.ID)

	return e.RunScript(script.ID)
}

// ExecuteScriptFile executes a script file directly
func (e *Engine) ExecuteScriptFile(filename string) error {
	script, err := e.LoadScript(filename)
	if err != nil {
		return err
	}

	defer e.UnloadScript(script.ID)

	return e.RunScript(script.ID)
}

// GetScriptByName returns a script by name
func (e *Engine) GetScriptByName(name string) (*Script, error) {
	scripts := e.getScripts()
	for _, script := range scripts {
		if script.Name == name {
			return script, nil
		}
	}

	return nil, fmt.Errorf("script not found: %s", name)
}

// IsScriptRunning checks if a script is running by name
func (e *Engine) IsScriptRunning(name string) bool {
	script, err := e.GetScriptByName(name)
	if err != nil {
		return false
	}
	return script.Running
}

// GetScriptCount returns the number of loaded scripts
func (e *Engine) GetScriptCount() int {
	scripts := e.getScripts()
	return len(scripts)
}

// GetRunningScriptCount returns the number of running scripts
func (e *Engine) GetRunningScriptCount() int {
	scripts := e.getScripts()
	count := 0
	for _, script := range scripts {
		if script.Running {
			count++
		}
	}
	return count
}

// GetStatus implements interfaces.ScriptEngine
func (e *Engine) GetStatus() map[string]interface{} {
	scripts := e.getScripts()
	status := make(map[string]interface{})
	status["total_scripts"] = len(scripts)

	runningCount := 0
	for _, script := range scripts {
		if script.Running {
			runningCount++
		}
	}
	status["running_scripts"] = runningCount

	if e.triggerManager != nil {
		status["trigger_count"] = e.triggerManager.GetTriggerCount()
	} else {
		status["trigger_count"] = 0
	}
	return status
}

// GetAllVariables returns all variables from all running scripts
func (e *Engine) GetAllVariables() map[string]*types.Value {
	scripts := e.getScripts()
	allVariables := make(map[string]*types.Value)

	for _, script := range scripts {
		if script.Running && script.VM != nil {
			// Get variables from this script's VM
			scriptVars := script.VM.GetAllVariables()

			// Prefix variable names with script name to avoid conflicts
			for varName, varValue := range scriptVars {
				prefixedName := fmt.Sprintf("%s.%s", script.Name, varName)
				allVariables[prefixedName] = varValue
			}

			// Also add without prefix for convenience (latest script wins in case of conflicts)
			for varName, varValue := range scriptVars {
				allVariables[varName] = varValue
			}
		}
	}

	return allVariables
}

// onScriptTerminated handles cleanup when a script terminates
func (e *Engine) onScriptTerminated(scriptID string) {
	defer func() {
		if r := recover(); r != nil {
			debug.Error("PANIC in onScriptTerminated", "error", r)
		}
	}()

	// Notify menu manager to clean up any menus owned by this script
	if e.gameInterface != nil {
		if gameAdapter, ok := e.gameInterface.(interface {
			GetMenuManager() interface{}
		}); ok {
			if menuManager := gameAdapter.GetMenuManager(); menuManager != nil {
				if tmm, ok := menuManager.(interface {
					RemoveScriptMenusByOwner(string)
				}); ok {
					tmm.RemoveScriptMenusByOwner(scriptID)
				}
			}
		}
	}
}

// ValidateScript validates a script without executing it
func (e *Engine) ValidateScript(source string) error {
	_, err := e.parseScript(source)
	return err
}

// ValidateScriptFile validates a script file without executing it
func (e *Engine) ValidateScriptFile(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read script file %s: %v", filename, err)
	}

	return e.ValidateScript(string(content))
}

// CompileScript compiles a script to check for syntax errors
func (e *Engine) CompileScript(source string) (*parser.ASTNode, error) {
	return e.parseScript(source)
}

// CompileScriptFile compiles a script file to check for syntax errors
func (e *Engine) CompileScriptFile(filename string) (*parser.ASTNode, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file %s: %v", filename, err)
	}

	return e.CompileScript(string(content))
}

// PrintAST prints the AST for debugging
func (e *Engine) PrintAST(ast *parser.ASTNode, indent int) {
	if ast == nil {
		return
	}

	// AST printing disabled - enable debug logging to see tree structure
	for _, child := range ast.Children {
		e.PrintAST(child, indent+1)
	}
}
