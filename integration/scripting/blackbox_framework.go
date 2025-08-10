package scripting

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"twist/internal/api"
	"twist/internal/proxy"
	"twist/internal/proxy/database"
)

// BlackBoxTestFramework provides completely black box testing
// It simulates a real user typing at a terminal - no knowledge of internals
type BlackBoxTestFramework struct {
	t                *testing.T
	proxy            *proxy.Proxy
	database         database.Database
	tempDir          string
	scriptFiles      []string
	cleanupFunctions []func()
	
	// User input/output simulation
	inputChan        chan string    // Where "user" types input
	outputCapture    []string       // What the "user" sees on screen
	outputMutex      sync.Mutex
	
	// Synchronization
	waitingForInput  bool
	inputMutex       sync.Mutex
}

// NewBlackBoxTestFramework creates a new black box testing framework
func NewBlackBoxTestFramework(t *testing.T) *BlackBoxTestFramework {
	framework := &BlackBoxTestFramework{
		t:                t,
		scriptFiles:      make([]string, 0),
		cleanupFunctions: make([]func(), 0),
		tempDir:          t.TempDir(),
		outputCapture:    make([]string, 0),
		inputChan:        make(chan string, 10),
		waitingForInput:  false,
	}

	// Register cleanup
	t.Cleanup(framework.Cleanup)

	return framework
}

// SetupDatabase creates and initializes a fresh database for testing
func (f *BlackBoxTestFramework) SetupDatabase() *BlackBoxTestFramework {
	dbPath := filepath.Join(f.tempDir, "test.db")
	
	// Clean up any existing database file
	if _, err := os.Stat(dbPath); err == nil {
		os.Remove(dbPath)
	}
	
	// Create a completely new database instance
	f.database = database.NewDatabase()
	
	// Create database file
	err := f.database.CreateDatabase(dbPath)
	if err != nil {
		f.t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Close after creation, then reopen - this ensures clean state
	f.database.CloseDatabase()
	
	// Create fresh instance and open
	f.database = database.NewDatabase()
	err = f.database.OpenDatabase(dbPath)
	if err != nil {
		f.t.Fatalf("Failed to open test database: %v", err)
	}
	
	f.cleanupFunctions = append(f.cleanupFunctions, func() {
		if f.database != nil {
			f.database.CloseDatabase()
		}
	})
	
	return f
}

// SetupProxy creates and configures the proxy with black box TUI
func (f *BlackBoxTestFramework) SetupProxy() *BlackBoxTestFramework {
	// Create a TUI API that captures output and provides input
	tuiAPI := &BlackBoxTuiAPI{
		framework: f,
	}
	
	// Create proxy with black box TUI
	f.proxy = proxy.New(tuiAPI)
	
	return f
}

// CreateScript creates a temporary script file and returns its path
func (f *BlackBoxTestFramework) CreateScript(name, content string) string {
	scriptPath := filepath.Join(f.tempDir, name)
	
	err := os.WriteFile(scriptPath, []byte(content), 0644)
	if err != nil {
		f.t.Fatalf("Failed to create script %s: %v", name, err)
	}
	
	f.scriptFiles = append(f.scriptFiles, scriptPath)
	return scriptPath
}

// LoadScript loads and starts execution of a script (like a user would)
func (f *BlackBoxTestFramework) LoadScript(scriptPath string) *BlackBoxTestFramework {
	if f.proxy == nil {
		f.t.Fatal("Must call SetupProxy() before LoadScript()")
	}
	
	// Start a goroutine to handle input requests
	go f.handleInputRequests()
	
	err := f.proxy.LoadScript(scriptPath)
	if err != nil {
		f.t.Fatalf("Failed to load script %s: %v", scriptPath, err)
	}
	
	// Give the script a moment to start executing
	time.Sleep(100 * time.Millisecond)
	
	f.t.Logf("Script loaded: %s", scriptPath)
	return f
}

// handleInputRequests simulates a user sitting at the terminal ready to type
func (f *BlackBoxTestFramework) handleInputRequests() {
	for {
		// Check if we're waiting for input
		f.inputMutex.Lock()
		waiting := f.waitingForInput
		f.inputMutex.Unlock()
		
		if waiting {
			// Wait for user to "type" something
			select {
			case input := <-f.inputChan:
				f.t.Logf("USER TYPES: %q", input)
				
				// Send each character followed by Enter (like real typing)
				for _, char := range input {
					f.proxy.SendInput(string(char))
					time.Sleep(5 * time.Millisecond) // Simulate typing speed
				}
				// Send Enter
				f.proxy.SendInput("\r")
				
				// No longer waiting
				f.inputMutex.Lock()
				f.waitingForInput = false
				f.inputMutex.Unlock()
				
			case <-time.After(5 * time.Second):
				// Timeout - no input provided
				f.t.Logf("No input provided after 5 seconds")
				return
			}
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// TypeInput simulates the user typing input (completely black box)
func (f *BlackBoxTestFramework) TypeInput(input string) *BlackBoxTestFramework {
	f.t.Logf("USER WANTS TO TYPE: %q", input)
	f.inputChan <- input
	return f
}

// WaitForPrompt waits for a specific prompt to appear (what user would see)
func (f *BlackBoxTestFramework) WaitForPrompt(expectedPrompt string, timeout time.Duration) *BlackBoxTestFramework {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		f.outputMutex.Lock()
		allOutput := strings.Join(f.outputCapture, "")
		f.outputMutex.Unlock()
		
		if strings.Contains(allOutput, expectedPrompt) {
			f.t.Logf("USER SEES PROMPT: %q", expectedPrompt)
			
			// Mark that we're waiting for input
			f.inputMutex.Lock()
			f.waitingForInput = true
			f.inputMutex.Unlock()
			
			return f
		}
		
		time.Sleep(50 * time.Millisecond)
	}
	
	// Timeout - show what we captured
	f.outputMutex.Lock()
	allOutput := strings.Join(f.outputCapture, "")
	f.outputMutex.Unlock()
	
	f.t.Errorf("Timeout waiting for prompt %q. User sees: %q", expectedPrompt, allOutput)
	return f
}

// WaitForOutput waits for specific output to appear (what user would see)
func (f *BlackBoxTestFramework) WaitForOutput(expectedOutput string, timeout time.Duration) *BlackBoxTestFramework {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		f.outputMutex.Lock()
		allOutput := strings.Join(f.outputCapture, "")
		f.outputMutex.Unlock()
		
		if strings.Contains(allOutput, expectedOutput) {
			f.t.Logf("USER SEES OUTPUT: %q", expectedOutput)
			return f
		}
		
		time.Sleep(50 * time.Millisecond)
	}
	
	// Timeout - show what we captured
	f.outputMutex.Lock()
	allOutput := strings.Join(f.outputCapture, "")
	f.outputMutex.Unlock()
	
	f.t.Errorf("Timeout waiting for output %q. User sees: %q", expectedOutput, allOutput)
	return f
}

// GetUserOutput returns everything the user would see on their screen
func (f *BlackBoxTestFramework) GetUserOutput() string {
	f.outputMutex.Lock()
	defer f.outputMutex.Unlock()
	return strings.Join(f.outputCapture, "")
}

// AssertUserSees verifies the user sees specific output
func (f *BlackBoxTestFramework) AssertUserSees(expected string) *BlackBoxTestFramework {
	output := f.GetUserOutput()
	if !strings.Contains(output, expected) {
		f.t.Errorf("User should see %q but sees: %q", expected, output)
	}
	return f
}

// AssertUserDoesNotSee verifies the user doesn't see specific output
func (f *BlackBoxTestFramework) AssertUserDoesNotSee(notExpected string) *BlackBoxTestFramework {
	output := f.GetUserOutput()
	if strings.Contains(output, notExpected) {
		f.t.Errorf("User should NOT see %q but sees: %q", notExpected, output)
	}
	return f
}

// Cleanup cleans up all resources
func (f *BlackBoxTestFramework) Cleanup() {
	for i := len(f.cleanupFunctions) - 1; i >= 0; i-- {
		f.cleanupFunctions[i]()
	}
	
	// Clean up script files
	for _, scriptPath := range f.scriptFiles {
		os.Remove(scriptPath)
	}
}

// BlackBoxTuiAPI implements the TuiAPI interface for black box testing
// This simulates what the user sees on their terminal screen
type BlackBoxTuiAPI struct {
	framework *BlackBoxTestFramework
}

func (b *BlackBoxTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {
	b.framework.t.Logf("Connection status: %s (%s)", status, address)
}

func (b *BlackBoxTuiAPI) OnConnectionError(err error) {
	b.framework.t.Logf("Connection error: %v", err)
}

func (b *BlackBoxTuiAPI) OnData(data []byte) {
	// This is what appears on the user's screen
	output := string(data)
	
	b.framework.outputMutex.Lock()
	b.framework.outputCapture = append(b.framework.outputCapture, output)
	b.framework.outputMutex.Unlock()
	
	b.framework.t.Logf("USER SEES: %q", output)
}

func (b *BlackBoxTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo) {
	b.framework.t.Logf("Script status: %+v", status)
}

func (b *BlackBoxTuiAPI) OnScriptError(scriptName string, err error) {
	b.framework.t.Logf("Script error in %s: %v", scriptName, err)
}

func (b *BlackBoxTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo) {
	b.framework.t.Logf("Database state: %+v", info)
}

func (b *BlackBoxTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo) {
	b.framework.t.Logf("Sector changed: %+v", sectorInfo)
}

func (b *BlackBoxTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {
	b.framework.t.Logf("Traders updated in sector %d: %d traders", sectorNumber, len(traders))
}

func (b *BlackBoxTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo) {
	b.framework.t.Logf("Player stats: %+v", stats)
}