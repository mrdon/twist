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
	
	// Streaming pipeline
	pipeline   *streaming.Pipeline
}

func New(terminalWriter streaming.TerminalWriter) *Proxy {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	
	logger := log.New(logFile, "[PROXY] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Proxy initialized")
	
	p := &Proxy{
		outputChan: make(chan string, 100),
		inputChan:  make(chan string, 100),
		errorChan:  make(chan error, 10),
		connected:  false,
		logger:     logger,
	}
	
	// Initialize streaming pipeline
	p.pipeline = streaming.NewPipeline(terminalWriter, func(data []byte) error {
		if p.conn != nil {
			_, err := p.conn.Write(data)
			return err
		}
		return fmt.Errorf("not connected")
	})
	
	return p
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
			// Send raw data directly to the streaming pipeline
			p.pipeline.Write(buffer[:n])
		}
	}
	
	p.logger.Println("Output handler exiting")
}

