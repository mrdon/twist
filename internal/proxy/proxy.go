package proxy

import (
	"bufio"
	"fmt"
	"net"
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
	
	
	// Streaming pipeline
	pipeline   *streaming.Pipeline
	
	// Script manager
	scriptManager *scripting.ScriptManager
	db            database.Database
}

func New(terminalWriter streaming.TerminalWriter) *Proxy {
	
	// Initialize database
	db := database.NewDatabase()
	// Create or open database (TODO: make configurable)
	if err := db.CreateDatabase("twist.db"); err != nil {
		db.OpenDatabase("twist.db")
	}
	
	// Initialize script manager
	scriptManager := scripting.NewScriptManager(db)
	
	p := &Proxy{
		outputChan:    make(chan string, 100),
		inputChan:     make(chan string, 100),
		errorChan:     make(chan error, 10),
		connected:     false,
		scriptManager: scriptManager,
		db:            db,
	}
	
	// Initialize streaming pipeline with script manager
	p.pipeline = streaming.NewPipelineWithScriptManager(terminalWriter, func(data []byte) error {
		if p.conn != nil {
			_, err := p.conn.Write(data)
			return err
		}
		return fmt.Errorf("not connected")
	}, db, scriptManager, nil)
	
	// Setup script manager connections to proxy and terminal
	if terminal, ok := terminalWriter.(scripting.TerminalInterface); ok {
		scriptManager.SetupConnections(p, terminal)
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
	if p.connected {
		return fmt.Errorf("already connected")
	}

	// Parse address (default to telnet port if not specified)
	if !strings.Contains(address, ":") {
		address = address + ":23"
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	p.conn = conn
	p.reader = bufio.NewReader(conn)
	p.writer = bufio.NewWriter(conn)
	p.connected = true

	// Start the streaming pipeline
	p.pipeline.Start()
	
	// Send initial telnet negotiation through pipeline
	err = p.pipeline.SendTelnetNegotiation()
	if err != nil {
		conn.Close()
		return fmt.Errorf("telnet negotiation failed: %w", err)
	}

	// Start goroutines for handling I/O
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
		p.scriptManager.Stop()
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
			p.scriptManager.ProcessOutgoingText(input)
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
	// Use a buffer for continuous reading
	buffer := make([]byte, 4096)
	
	for {
		p.mu.RLock()
		connected := p.connected
		p.mu.RUnlock()

		if !connected {
			break
		}

		// Read raw bytes from connection
		n, err := p.reader.Read(buffer)
		if err != nil {
			if err.Error() != "EOF" {
				p.errorChan <- fmt.Errorf("read error: %w", err)
			}
			break
		}
		
		if n > 0 {
			rawData := buffer[:n]
			
			// Send raw data directly to the streaming pipeline
			p.pipeline.Write(rawData)
		}
	}
	
}

// GetScriptManager returns the script manager for external access
func (p *Proxy) GetScriptManager() *scripting.ScriptManager {
	return p.scriptManager
}

// LoadScript loads a script from file
func (p *Proxy) LoadScript(filename string) error {
	return p.scriptManager.LoadAndRunScript(filename)
}

// ExecuteScriptCommand executes a single script command
func (p *Proxy) ExecuteScriptCommand(command string) error {
	return p.scriptManager.ExecuteCommand(command)
}

// GetScriptStatus returns script engine status
func (p *Proxy) GetScriptStatus() map[string]interface{} {
	return p.scriptManager.GetStatus()
}

// StopAllScripts stops all running scripts
func (p *Proxy) StopAllScripts() error {
	return p.scriptManager.Stop()
}

