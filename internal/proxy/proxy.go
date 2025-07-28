package proxy

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"twist/internal/streaming"
	"twist/internal/database"
	"twist/internal/scripting"
)

type Proxy struct {
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	mu       sync.RWMutex
	connected bool
	
	// Channels for communication with TUI
	outputChan chan string
	inputChan  chan string
	errorChan  chan error
	
	// Logger for debugging
	logger     *log.Logger
	rawLogger  *log.Logger // Logger for raw server data
	pvpLogger  *log.Logger // Logger for NO PVP tracking
	
	// Streaming pipeline
	pipeline   *streaming.Pipeline
	
	// Script manager
	scriptManager *scripting.ScriptManager
	db            database.Database
}

func New(terminalWriter streaming.TerminalWriter) *Proxy {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	
	// Set up raw server data logging
	rawLogFile, err := os.OpenFile("raw_server_data.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open raw server data log file: %v", err)
	}
	
	// Set up NO PVP tracking log
	pvpLogFile, err := os.OpenFile("no_pvp_tracking.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open NO PVP tracking log file: %v", err)
	}
	
	logger := log.New(logFile, "[PROXY] ", log.LstdFlags|log.Lshortfile)
	rawLogger := log.New(rawLogFile, "[RAW] ", log.LstdFlags|log.Lshortfile)
	pvpLogger := log.New(pvpLogFile, "[PVP] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Proxy initialized")
	
	// Initialize database
	db := database.NewDatabase()
	// Create or open database (TODO: make configurable)
	if err := db.CreateDatabase("twist.db"); err != nil {
		logger.Printf("Failed to create database, trying to open: %v", err)
		if err := db.OpenDatabase("twist.db"); err != nil {
			logger.Fatalf("Failed to open database: %v", err)
		}
	}
	
	// Initialize script manager
	scriptManager := scripting.NewScriptManager(db)
	logger.Println("Script manager initialized")
	
	p := &Proxy{
		outputChan:    make(chan string, 100),
		inputChan:     make(chan string, 100),
		errorChan:     make(chan error, 10),
		connected:     false,
		logger:        logger,
		rawLogger:     rawLogger,
		pvpLogger:     pvpLogger,
		scriptManager: scriptManager,
		db:            db,
	}
	
	// Initialize streaming pipeline with script manager and shared logger
	p.pipeline = streaming.NewPipelineWithScriptManager(terminalWriter, func(data []byte) error {
		if p.conn != nil {
			_, err := p.conn.Write(data)
			return err
		}
		return fmt.Errorf("not connected")
	}, db, scriptManager, pvpLogger)
	
	// Setup script manager connections to proxy and terminal
	if terminal, ok := terminalWriter.(scripting.TerminalInterface); ok {
		scriptManager.SetupConnections(p, terminal)
		logger.Println("Script manager connections established")
	} else {
		logger.Println("Warning: terminalWriter does not implement TerminalInterface")
	}
	
	return p
}

// escapeANSI converts ANSI escape sequences to readable text
func escapeANSI(data []byte) string {
	str := string(data)
	// Replace escape character with \x1b for readability
	str = strings.ReplaceAll(str, "\x1b", "\\x1b")
	// Replace other common control characters
	str = strings.ReplaceAll(str, "\r", "\\r")
	str = strings.ReplaceAll(str, "\n", "\\n")
	str = strings.ReplaceAll(str, "\t", "\\t")
	return str
}

// extractContext returns 10 chars before and after the target string
func extractContext(data []byte, target string) string {
	str := string(data)
	index := strings.Index(str, target)
	if index == -1 {
		return ""
	}
	
	start := index - 10
	if start < 0 {
		start = 0
	}
	
	end := index + len(target) + 10
	if end > len(str) {
		end = len(str)
	}
	
	context := str[start:end]
	return escapeANSI([]byte(context))
}

func (p *Proxy) Connect(address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Printf("Attempting to connect to %s", address)

	if p.connected {
		return fmt.Errorf("already connected")
	}

	// Parse address (default to telnet port if not specified)
	if !strings.Contains(address, ":") {
		address = address + ":23"
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		p.logger.Printf("Connection failed: %v", err)
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	p.logger.Printf("TCP connection established to %s", address)
	p.conn = conn
	p.reader = bufio.NewReader(conn)
	p.writer = bufio.NewWriter(conn)
	p.connected = true

	// Start the streaming pipeline
	p.logger.Println("Starting streaming pipeline")
	p.pipeline.Start()
	
	// Send initial telnet negotiation through pipeline
	err = p.pipeline.SendTelnetNegotiation()
	if err != nil {
		p.logger.Printf("Telnet negotiation failed: %v", err)
		conn.Close()
		return fmt.Errorf("telnet negotiation failed: %w", err)
	}

	// Start goroutines for handling I/O
	p.logger.Println("Starting I/O goroutines")
	go p.handleInput()
	go p.handleOutput()

	return nil
}

func (p *Proxy) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.connected {
		return nil
	}

	p.connected = false
	
	// Stop all scripts
	if p.scriptManager != nil {
		if err := p.scriptManager.Stop(); err != nil {
			p.logger.Printf("Error stopping scripts: %v", err)
		}
	}
	
	// Stop the streaming pipeline
	if p.pipeline != nil {
		p.pipeline.Stop()
	}
	
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}

	return nil
}

func (p *Proxy) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected
}

func (p *Proxy) SendInput(input string) {
	select {
	case p.inputChan <- input:
	default:
		// Channel full, drop input
	}
}

func (p *Proxy) GetOutputChan() <-chan string {
	return p.outputChan
}

func (p *Proxy) GetErrorChan() <-chan error {
	return p.errorChan
}

// GetTerminal method removed - TUI now owns the terminal buffer

func (p *Proxy) handleInput() {
	for input := range p.inputChan {
		p.mu.RLock()
		connected := p.connected && p.writer != nil
		p.mu.RUnlock()

		if !connected {
			continue
		}

		// Process outgoing text through script manager
		if p.scriptManager != nil {
			if err := p.scriptManager.ProcessOutgoingText(input); err != nil {
				p.logger.Printf("Outgoing script processing error: %v", err)
			}
		}

		_, err := p.writer.WriteString(input)
		if err != nil {
			p.errorChan <- fmt.Errorf("write error: %w", err)
			continue
		}

		err = p.writer.Flush()
		if err != nil {
			p.errorChan <- fmt.Errorf("flush error: %w", err)
		}
	}
}

func (p *Proxy) handleOutput() {
	p.logger.Println("Output handler started")
	
	// Use a buffer for continuous reading
	buffer := make([]byte, 4096)
	
	for {
		p.mu.RLock()
		connected := p.connected
		p.mu.RUnlock()

		if !connected {
			p.logger.Println("Output handler: connection closed, exiting")
			break
		}

		// Read raw bytes from connection
		n, err := p.reader.Read(buffer)
		if err != nil {
			p.logger.Printf("Read error: %v", err)
			if err.Error() != "EOF" {
				p.errorChan <- fmt.Errorf("read error: %w", err)
			}
			break
		}
		
		if n > 0 {
			p.logger.Printf("Read %d bytes from server", n)
			
			// Log raw server data with escaped ANSI codes
			rawData := buffer[:n]
			escapedData := escapeANSI(rawData)
			p.rawLogger.Printf("RAW SERVER DATA (%d bytes): %s", n, escapedData)
			
			// Track NO PVP with color analysis
			rawStr := string(rawData)
			if strings.Contains(rawStr, "NO") && strings.Contains(rawStr, "PVP") {
				// Extract the actual ANSI sequence around NO PVP
				start := strings.Index(rawStr, "NO") - 10
				if start < 0 { start = 0 }
				end := strings.Index(rawStr, "PVP") + 10
				if end > len(rawStr) { end = len(rawStr) }
				context := rawStr[start:end]
				// Escape for readability
				context = strings.ReplaceAll(context, "\x1b", "\\x1b")
				p.pvpLogger.Printf("STAGE 1 - RAW: %s", context)
			}
			
			// Also log hex dump for complete analysis
			p.rawLogger.Printf("HEX DUMP: %x", rawData)
			
			// Send raw data directly to the streaming pipeline
			p.pipeline.Write(rawData)
		}
	}
	
	p.logger.Println("Output handler exiting")
}

// GetScriptManager returns the script manager for external access
func (p *Proxy) GetScriptManager() *scripting.ScriptManager {
	return p.scriptManager
}

// LoadScript loads a script from file
func (p *Proxy) LoadScript(filename string) error {
	p.logger.Printf("Loading script: %s", filename)
	return p.scriptManager.LoadAndRunScript(filename)
}

// ExecuteScriptCommand executes a single script command
func (p *Proxy) ExecuteScriptCommand(command string) error {
	p.logger.Printf("Executing script command: %s", command)
	return p.scriptManager.ExecuteCommand(command)
}

// GetScriptStatus returns script engine status
func (p *Proxy) GetScriptStatus() map[string]interface{} {
	return p.scriptManager.GetStatus()
}

// StopAllScripts stops all running scripts
func (p *Proxy) StopAllScripts() error {
	p.logger.Println("Stopping all scripts")
	return p.scriptManager.Stop()
}

