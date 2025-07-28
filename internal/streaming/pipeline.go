package streaming

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"twist/internal/telnet"
	"twist/internal/database"
	"twist/internal/streaming/parser"
)

// ScriptManager interface for script processing
type ScriptManager interface {
	ProcessGameLine(line string) error
}

// Pipeline provides high-performance streaming from network to terminal buffer
type Pipeline struct {
	// Input
	rawDataChan chan []byte
	
	// Processing layers
	telnetHandler  *telnet.Handler
	terminalWriter TerminalWriter
	decoder        *encoding.Decoder
	sectorParser   *parser.SectorParser
	scriptManager  ScriptManager
	
	// Batching
	batchBuffer   []byte
	batchMutex    sync.Mutex
	batchTimer    *time.Timer
	batchSize     int
	batchTimeout  time.Duration
	
	// State
	running       bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	
	// Metrics
	bytesProcessed uint64
	batchesProcessed uint64
	
	logger *log.Logger
	pvpLogger *log.Logger
}

// TerminalWriter interface for writing to terminal buffer
type TerminalWriter interface {
	Write([]byte)
}

// NewPipeline creates an optimized streaming pipeline
func NewPipeline(terminalWriter TerminalWriter, writer func([]byte) error, db database.Database) *Pipeline {
	return NewPipelineWithScriptManager(terminalWriter, writer, db, nil, nil)
}

// NewPipelineWithScriptManager creates an optimized streaming pipeline with script support
func NewPipelineWithScriptManager(terminalWriter TerminalWriter, writer func([]byte) error, db database.Database, scriptManager ScriptManager, pvpLogger *log.Logger) *Pipeline {
	// Set up debug logging
	logFile, err := os.OpenFile("twist_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	
	logger := log.New(logFile, "[PIPELINE] ", log.LstdFlags|log.Lshortfile)
	
	p := &Pipeline{
		rawDataChan:    make(chan []byte, 100), // Buffered for burst handling
		terminalWriter: terminalWriter,
		decoder:        charmap.CodePage437.NewDecoder(),
		sectorParser:   parser.NewSectorParser(db),
		scriptManager:  scriptManager,
		batchBuffer:    make([]byte, 0, 4096),
		batchSize:      1,     // Process immediately - no batching
		batchTimeout:   0,     // No timeout needed
		stopChan:       make(chan struct{}),
		logger:         logger,
		pvpLogger:      pvpLogger,
	}
	
	// Initialize telnet handler
	p.telnetHandler = telnet.NewHandler(writer)
	
	return p
}

// Start begins the streaming pipeline
func (p *Pipeline) Start() {
	p.running = true
	p.logger.Println("Starting streaming pipeline")
	
	// Start the processing goroutine
	p.wg.Add(1)
	go p.batchProcessor()
	
	p.logger.Println("Pipeline started")
}

// Stop gracefully shuts down the pipeline
func (p *Pipeline) Stop() {
	if !p.running {
		return
	}
	
	p.logger.Println("Stopping pipeline")
	p.running = false
	close(p.stopChan)
	
	// Stop batch timer if running
	if p.batchTimer != nil {
		p.batchTimer.Stop()
	}
	
	p.wg.Wait()
	p.logger.Printf("Pipeline stopped - processed %d bytes in %d batches", 
		p.bytesProcessed, p.batchesProcessed)
}

// Write feeds raw data into the pipeline
func (p *Pipeline) Write(data []byte) {
	if !p.running {
		return
	}
	
	p.logger.Printf("Pipeline received %d bytes for processing", len(data))
	
	// Make a copy since the caller might reuse the buffer
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	
	select {
	case p.rawDataChan <- dataCopy:
		p.logger.Printf("Data queued in pipeline channel")
	default:
		// Channel full - this shouldn't happen with proper sizing
		p.logger.Printf("Warning: dropping %d bytes due to full channel", len(data))
	}
}

// SendTelnetNegotiation sends initial telnet negotiation
func (p *Pipeline) SendTelnetNegotiation() error {
	return p.telnetHandler.SendInitialNegotiation()
}

// batchProcessor processes raw data immediately (no batching)
func (p *Pipeline) batchProcessor() {
	defer p.wg.Done()
	
	for {
		select {
		case rawData := <-p.rawDataChan:
			p.logger.Printf("Processing %d bytes immediately", len(rawData))
			
			// Process telnet commands immediately
			cleanData := p.telnetHandler.ProcessData(rawData)
			p.logger.Printf("Telnet processed %d bytes -> %d clean bytes", len(rawData), len(cleanData))
			
			if len(cleanData) > 0 {
				// Track NO PVP after telnet processing
				cleanStr := string(cleanData)
				if strings.Contains(cleanStr, "NO") && strings.Contains(cleanStr, "PVP") {
					start := strings.Index(cleanStr, "NO") - 10
					if start < 0 { start = 0 }
					end := strings.Index(cleanStr, "PVP") + 10
					if end > len(cleanStr) { end = len(cleanStr) }
					context := cleanStr[start:end]
					context = strings.ReplaceAll(context, "\x1b", "\\x1b")
					p.pvpLogger.Printf("STAGE 2 - TELNET: %s", context)
				}
				
				// Process through script triggers EARLY with original ANSI codes intact
				if p.scriptManager != nil {
					if err := p.scriptManager.ProcessGameLine(string(cleanData)); err != nil {
						p.logger.Printf("Script processing error: %v", err)
					}
				}
				
				// Use standard CP437 to UTF-8 conversion
				decoded, err := p.decoder.Bytes(cleanData)
				if err != nil {
					p.logger.Printf("CP437 decode error: %v, falling back to raw data", err)
					decoded = cleanData
				}
				
				// Track NO PVP after encoding
				decodedStr := string(decoded)
				if strings.Contains(decodedStr, "NO") && strings.Contains(decodedStr, "PVP") {
					start := strings.Index(decodedStr, "NO") - 10
					if start < 0 { start = 0 }
					end := strings.Index(decodedStr, "PVP") + 10
					if end > len(decodedStr) { end = len(decodedStr) }
					context := decodedStr[start:end]
					context = strings.ReplaceAll(context, "\x1b", "\\x1b")
					p.pvpLogger.Printf("STAGE 3 - ENCODING: %s", context)
				}
				
				// Parse the decoded text for sector information
				p.sectorParser.ProcessData(decoded)
				
				p.logger.Printf("Sending %d converted bytes to terminal: %q", len(decoded), string(decoded))
				p.terminalWriter.Write(decoded)
				p.bytesProcessed += uint64(len(rawData))
				p.batchesProcessed++
			} else {
				p.logger.Printf("No clean data after telnet processing - %d raw bytes filtered out", len(rawData))
			}
			
		case <-p.stopChan:
			return
		}
	}
}

// addToBatch adds data to the current batch and flushes if needed
func (p *Pipeline) addToBatch(data []byte, output chan<- []byte) {
	p.batchMutex.Lock()
	defer p.batchMutex.Unlock()
	
	p.batchBuffer = append(p.batchBuffer, data...)
	p.bytesProcessed += uint64(len(data))
	
	// Check if we should flush the batch
	shouldFlush := false
	
	// Size-based flush
	if len(p.batchBuffer) >= p.batchSize {
		shouldFlush = true
	}
	
	// Time-based flush (start timer on first data)
	if p.batchTimer == nil && len(p.batchBuffer) > 0 {
		p.batchTimer = time.AfterFunc(p.batchTimeout, func() {
			p.batchMutex.Lock()
			defer p.batchMutex.Unlock()
			if len(p.batchBuffer) > 0 {
				p.flushBatchLocked(output)
			}
		})
	}
	
	if shouldFlush {
		if p.batchTimer != nil {
			p.batchTimer.Stop()
			p.batchTimer = nil
		}
		p.flushBatchLocked(output)
	}
}

// flushBatch processes and sends the current batch (with locking)
func (p *Pipeline) flushBatch(output chan<- []byte) {
	p.batchMutex.Lock()
	defer p.batchMutex.Unlock()
	p.flushBatchLocked(output)
}

// flushBatchLocked processes and sends the current batch (assumes lock held)
func (p *Pipeline) flushBatchLocked(output chan<- []byte) {
	if len(p.batchBuffer) == 0 {
		return
	}
	
	// Process telnet commands
	cleanData := p.telnetHandler.ProcessData(p.batchBuffer)
	p.logger.Printf("Telnet processed %d bytes -> %d clean bytes", len(p.batchBuffer), len(cleanData))
	
	if len(cleanData) > 0 {
		// Decode character encoding
		decoded, err := p.decoder.Bytes(cleanData)
		if err != nil {
			p.logger.Printf("Decode error: %v, using raw data", err)
			decoded = cleanData
		}
		
		// Send to terminal processor
		select {
		case output <- decoded:
			p.batchesProcessed++
			p.logger.Printf("Processed batch: %d bytes -> %d decoded bytes", 
				len(p.batchBuffer), len(decoded))
		default:
			p.logger.Printf("Warning: terminal processor channel full")
		}
	}
	
	// Reset batch buffer - reuse underlying array
	p.batchBuffer = p.batchBuffer[:0]
}

// terminalProcessor handles terminal updates (placeholder for now)
func (p *Pipeline) terminalProcessor() {
	defer p.wg.Done()
	
	// This goroutine is now handled inside batchProcessor
	// but we keep this method for future terminal-specific optimizations
	select {
	case <-p.stopChan:
		return
	}
}

// GetTerminalWriter returns the terminal writer for external access
func (p *Pipeline) GetTerminalWriter() TerminalWriter {
	return p.terminalWriter
}


// GetMetrics returns pipeline performance metrics
func (p *Pipeline) GetMetrics() (bytesProcessed, batchesProcessed uint64) {
	return p.bytesProcessed, p.batchesProcessed
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