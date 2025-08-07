package scripting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"twist/internal/ansi"
	"twist/internal/debug"
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

// GetName implements ScriptInterface
func (s *Script) GetName() string {
	return s.Name
}

// IsRunning implements ScriptInterface
func (s *Script) IsRunning() bool {
	return s.Running
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
	scripts        map[string]*Script
	gameInterface  types.GameInterface
	triggerManager *triggers.Manager
	mutex          sync.RWMutex
	nextScriptID   int
	
	// ANSI stripper for streaming text processing
	ansiStripper   *ansi.StreamingStripper
	
	// Event handlers
	outputHandler func(string) error
	echoHandler   func(string) error
	sendHandler   func(string) error
}

// NewEngine creates a new scripting engine
func NewEngine(gameInterface types.GameInterface) *Engine {
	engine := &Engine{
		scripts:        make(map[string]*Script),
		gameInterface:  gameInterface,
		nextScriptID:   1,
		ansiStripper:   ansi.NewStreamingStripper(),
	}
	
	// Create a dummy VM for the trigger manager
	dummyVM := vm.NewVirtualMachine(gameInterface)
	engine.triggerManager = triggers.NewManager(dummyVM)
	
	return engine
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
	defer e.mutex.Unlock()
	
	e.sendHandler = handler
	
	// Update all existing script VMs with the new sendHandler
	for _, script := range e.scripts {
		if script.VM != nil {
			script.VM.SetSendHandler(handler)
		}
	}
}

// LoadScript loads a script from a file
func (e *Engine) LoadScript(filename string) (*Script, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
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
	
	// Create script object
	scriptID := e.generateScriptID()
	scriptName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	
	script := &Script{
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
	
	// Store script
	e.scripts[scriptID] = script
	
	return script, nil
}

// LoadScriptFromString loads a script from a string
func (e *Engine) LoadScriptFromString(content, name string) (*Script, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	// Parse script
	ast, err := e.parseScript(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script %s: %v", name, err)
	}
	
	// Create script object
	scriptID := e.generateScriptID()
	
	script := &Script{
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
	
	// Store script
	e.scripts[scriptID] = script
	
	return script, nil
}

// RunScript starts executing a script
func (e *Engine) RunScript(scriptID string) error {
	e.mutex.RLock()
	script, exists := e.scripts[scriptID]
	e.mutex.RUnlock()
	
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
	
	script.Running = true
	
	// Execute script in a goroutine
	go func() {
		defer func() {
			// Only mark as not running if the script truly finished or errored
			// Don't mark as not running if it's just waiting for input
			if !script.VM.IsWaiting() {
				script.Running = false
			} else {
			}
		}()
		
		if err := script.VM.Execute(); err != nil {
			// Handle script error
			script.Running = false
			if e.outputHandler != nil {
				e.outputHandler(fmt.Sprintf("Script error in %s: %v", script.Name, err))
			}
		}
	}()
	
	return nil
}

// RunScriptSync executes a script synchronously and returns any execution error
func (e *Engine) RunScriptSync(scriptID string) error {
	e.mutex.RLock()
	script, exists := e.scripts[scriptID]
	e.mutex.RUnlock()
	
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
	
	script.Running = true
	defer func() {
		script.Running = false
	}()
	
	// Execute script synchronously
	return script.VM.Execute()
}

// StopScript stops a running script
func (e *Engine) StopScript(scriptID string) error {
	e.mutex.RLock()
	script, exists := e.scripts[scriptID]
	e.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}
	
	return script.Stop()
}

// StopAllScripts stops all running scripts
func (e *Engine) StopAllScripts() error {
	e.mutex.RLock()
	scripts := make([]*Script, 0, len(e.scripts))
	for _, script := range e.scripts {
		if script.Running {
			scripts = append(scripts, script)
		}
	}
	e.mutex.RUnlock()
	
	for _, script := range scripts {
		if err := script.Stop(); err != nil {
			return err
		}
	}
	
	return nil
}

// UnloadScript removes a script from memory
func (e *Engine) UnloadScript(scriptID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	script, exists := e.scripts[scriptID]
	if !exists {
		return fmt.Errorf("script not found: %s", scriptID)
	}
	
	if script.Running {
		if err := script.Stop(); err != nil {
			return err
		}
	}
	
	delete(e.scripts, scriptID)
	return nil
}

// GetScript returns a script by ID
func (e *Engine) GetScript(scriptID string) (*Script, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	script, exists := e.scripts[scriptID]
	if !exists {
		return nil, fmt.Errorf("script not found: %s", scriptID)
	}
	
	return script, nil
}

// ListScripts returns all loaded scripts
func (e *Engine) ListScripts() map[string]*Script {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	result := make(map[string]*Script)
	for k, v := range e.scripts {
		result[k] = v
	}
	return result
}

// GetRunningScripts returns all running scripts
func (e *Engine) GetRunningScripts() []*Script {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	result := make([]*Script, 0)
	for _, script := range e.scripts {
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
	
	// Forward stripped text to all running script VMs for waitfor processing
	e.mutex.RLock()
	scriptCount := 0
	for _, script := range e.scripts {
		if script.Running && script.VM != nil {
			scriptCount++
			if err := script.VM.ProcessIncomingText(strippedText); err != nil {
			} else {
			}
		}
	}
	e.mutex.RUnlock()
	
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
		debug.Log("Token %d: %+v", i, token)
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
	id := fmt.Sprintf("script_%d", e.nextScriptID)
	e.nextScriptID++
	return id
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
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	for _, script := range e.scripts {
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
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	return len(e.scripts)
}

// GetRunningScriptCount returns the number of running scripts
func (e *Engine) GetRunningScriptCount() int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	count := 0
	for _, script := range e.scripts {
		if script.Running {
			count++
		}
	}
	return count
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