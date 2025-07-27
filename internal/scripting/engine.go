package scripting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"twist/internal/scripting/include"
	"twist/internal/scripting/parser"
	"twist/internal/scripting/triggers"
	"twist/internal/scripting/types"
	"twist/internal/scripting/vm"
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
	e.sendHandler = handler
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
			script.Running = false
		}()
		
		if err := script.VM.Execute(); err != nil {
			// Handle script error
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
	return e.triggerManager.ProcessText(text)
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
	
	prefix := strings.Repeat("  ", indent)
	fmt.Printf("%sNode: Type=%d, Value=%s, Line=%d\n", prefix, ast.Type, ast.Value, ast.Line)
	
	for _, child := range ast.Children {
		e.PrintAST(child, indent+1)
	}
}