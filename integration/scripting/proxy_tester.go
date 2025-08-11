package scripting

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strings"
	"testing"
	"twist/internal/api"
	"twist/internal/api/factory"
)

// ProxyResult contains the result of running a proxy test
type ProxyResult struct {
	Database     *sql.DB // Database instance for assertions
	ClientOutput string  // All output the client received
}

// Execute runs server and client scripts using the API with ConnectOptions
//
// Flow:
// 1. SERVER: Creates ExpectTelnetServer that runs serverScript (sends data to proxy)
// 2. PROXY: Created via api.Connect(), may load TWX script, processes data bidirectionally  
// 3. CLIENT: Creates SimpleExpectEngine that runs clientScript (expects data from proxy, sends user input)
//
func Execute(t *testing.T, serverScript, clientScript string, connectOpts *api.ConnectOptions) *ProxyResult {
	// 1. SERVER: Create and start telnet server with server script
	server := NewExpectTelnetServer(t)
	server.SetServerScript(serverScript)
	port, err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start telnet server: %v", err)
	}
	defer server.Stop()

	// If connectOpts contains a ScriptName that looks like script content, create a temp file
	if connectOpts != nil && connectOpts.ScriptName != "" && !strings.HasSuffix(connectOpts.ScriptName, ".ts") {
		// This looks like script content, not a filename - create temp file
		scriptPath := t.TempDir() + "/temp_script.ts"
		err := os.WriteFile(scriptPath, []byte(connectOpts.ScriptName), 0644)
		if err != nil {
			t.Fatalf("Failed to create temp script file: %v", err)
		}
		// Update connectOpts to use the file path
		connectOpts.ScriptName = scriptPath
	}

	// 2. PROXY: Create client expect engine and connect proxy
	clientExpectEngine := NewExpectEngine(t, nil, "\r")
	tuiAPI := &TestTuiAPI{expectEngine: clientExpectEngine}
	address := fmt.Sprintf("localhost:%d", port)
	proxyInstance := factory.Connect(address, tuiAPI, connectOpts)
	defer proxyInstance.Disconnect()

	// Set the input sender for client expect engine - this simulates user typing
	clientExpectEngine.inputSender = func(input string) {
		t.Logf("CLIENT EXPECT ENGINE SENDING USER INPUT: %q", input)
		// Send user input to the proxy, which forwards it to the server
		err := proxyInstance.SendData([]byte(input))
		if err != nil {
			t.Logf("Failed to send input: %v", err)
		}
	}

	// 3. CLIENT: Run client script (includes automatic sync token wait)
	err = clientExpectEngine.Run(clientScript)
	if err != nil {
		t.Fatalf("Client script failed: %v", err)
	}

	// Open database for return if path was specified
	var sqlDB *sql.DB
	if connectOpts != nil && connectOpts.DatabasePath != "" {
		sqlDB, err = sql.Open("sqlite3", connectOpts.DatabasePath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		t.Logf("PROXY TESTER: Returning database at %s", connectOpts.DatabasePath)
	}

	return &ProxyResult{
		Database:     sqlDB,
		ClientOutput: clientExpectEngine.GetAllOutput(),
	}
}

// TestTuiAPI implements api.TuiAPI to capture proxy output for client expect engine
type TestTuiAPI struct {
	expectEngine *SimpleExpectEngine // This is the client expect engine
}

func (t *TestTuiAPI) OnConnectionStatusChanged(status api.ConnectionStatus, address string) {}
func (t *TestTuiAPI) OnConnectionError(err error)                                           {}
func (t *TestTuiAPI) OnData(data []byte) {
	if t.expectEngine != nil {
		t.expectEngine.AddOutput(string(data))
	}
}
func (t *TestTuiAPI) OnScriptStatusChanged(status api.ScriptStatusInfo)              {}
func (t *TestTuiAPI) OnScriptError(scriptName string, err error)                     {}
func (t *TestTuiAPI) OnDatabaseStateChanged(info api.DatabaseStateInfo)              {}
func (t *TestTuiAPI) OnCurrentSectorChanged(sectorInfo api.SectorInfo)               {}
func (t *TestTuiAPI) OnTraderDataUpdated(sectorNumber int, traders []api.TraderInfo) {}
func (t *TestTuiAPI) OnPlayerStatsUpdated(stats api.PlayerStatsInfo)                 {}
