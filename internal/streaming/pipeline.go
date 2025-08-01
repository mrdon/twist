package streaming

import (
	"strings"
	"sync"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"twist/internal/telnet"
	"twist/internal/database"
	"twist/internal/streaming/parser"
	"twist/internal/api"
	// "twist/internal/debug" // Keep for future debugging
)

// ScriptManager interface for script processing
type ScriptManager interface {
	ProcessGameLine(line string) error
}

// StateManager interface for game state updates (avoids circular import with proxy)
type StateManager interface {
	SetCurrentSector(sectorNum int)
	SetPlayerName(name string)
}

// Pipeline provides high-performance streaming from network to terminal buffer
type Pipeline struct {
	// Input
	rawDataChan chan []byte
	
	// Processing layers
	telnetHandler  *telnet.Handler
	tuiAPI         api.TuiAPI  // Direct TuiAPI reference
	decoder        *encoding.Decoder
	sectorParser   *parser.SectorParser
	scriptManager  ScriptManager
	stateManager   StateManager  // Game state updates
	
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
}


// NewPipeline creates an optimized streaming pipeline
func NewPipeline(tuiAPI api.TuiAPI, db database.Database) *Pipeline {
	return &Pipeline{
		rawDataChan:   make(chan []byte, 100), // Buffered for burst handling
		tuiAPI:        tuiAPI,  // Direct TuiAPI reference
		decoder:       charmap.CodePage437.NewDecoder(),
		sectorParser:  parser.NewSectorParser(db),
		batchBuffer:   make([]byte, 0, 4096),
		batchSize:     1,     // Process immediately - no batching
		batchTimeout:  0,     // No timeout needed
		stopChan:      make(chan struct{}),
	}
}

// NewPipelineWithScriptManager creates an optimized streaming pipeline with script support
func NewPipelineWithScriptManager(tuiAPI api.TuiAPI, db database.Database, scriptManager ScriptManager) *Pipeline {
	p := &Pipeline{
		rawDataChan:   make(chan []byte, 100), // Buffered for burst handling
		tuiAPI:        tuiAPI,  // Direct TuiAPI reference
		decoder:       charmap.CodePage437.NewDecoder(),
		sectorParser:  parser.NewSectorParser(db),
		scriptManager: scriptManager,
		batchBuffer:   make([]byte, 0, 4096),
		batchSize:     1,     // Process immediately - no batching
		batchTimeout:  0,     // No timeout needed
		stopChan:      make(chan struct{}),
	}
	
	// Initialize telnet handler with no writer - scripts don't need to write back to connection
	p.telnetHandler = telnet.NewHandler(nil)
	
	return p
}

// NewPipelineWithWriter creates an optimized streaming pipeline with a writer for telnet negotiation
func NewPipelineWithWriter(tuiAPI api.TuiAPI, db database.Database, scriptManager ScriptManager, stateManager StateManager, writer func([]byte) error) *Pipeline {
	p := &Pipeline{
		rawDataChan:   make(chan []byte, 100), // Buffered for burst handling
		tuiAPI:        tuiAPI,  // Direct TuiAPI reference
		decoder:       charmap.CodePage437.NewDecoder(),
		sectorParser:  parser.NewSectorParserWithStateManager(db, stateManager),
		scriptManager: scriptManager,
		stateManager:  stateManager,
		batchBuffer:   make([]byte, 0, 4096),
		batchSize:     1,     // Process immediately - no batching
		batchTimeout:  0,     // No timeout needed
		stopChan:      make(chan struct{}),
	}
	
	// Initialize telnet handler with proper writer for negotiation
	p.telnetHandler = telnet.NewHandler(writer)
	
	return p
}

// Start begins the streaming pipeline
func (p *Pipeline) Start() {
	p.running = true
	
	// Start the processing goroutine
	p.wg.Add(1)
	go p.batchProcessor()
}

// Stop gracefully shuts down the pipeline
func (p *Pipeline) Stop() {
	if !p.running {
		return
	}
	
	p.running = false
	close(p.stopChan)
	
	// Stop batch timer if running
	if p.batchTimer != nil {
		p.batchTimer.Stop()
	}
	
	p.wg.Wait()
}

// Write feeds raw data into the pipeline
func (p *Pipeline) Write(data []byte) {
	if !p.running {
		return
	}
	
	// Make a copy since the caller might reuse the buffer
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	
	select {
	case p.rawDataChan <- dataCopy:
	default:
		// Channel full - this shouldn't happen with proper sizing
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
			// Process telnet commands immediately
			cleanData := p.telnetHandler.ProcessData(rawData)
			
			if len(cleanData) > 0 {
				
				// Process through script triggers EARLY with original ANSI codes intact
				if p.scriptManager != nil {
					if err := p.scriptManager.ProcessGameLine(string(cleanData)); err != nil {
						// Ignore script processing errors
					}
				}
				
				// Use standard CP437 to UTF-8 conversion
				decoded, err := p.decoder.Bytes(cleanData)
				if err != nil {
					decoded = cleanData
				}
				
				
				// Parse the decoded text for sector information
				p.sectorParser.ProcessData(decoded)
				
				if p.tuiAPI != nil {
					p.tuiAPI.OnData(decoded)
				}
				p.bytesProcessed += uint64(len(rawData))
				p.batchesProcessed++
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
	
	if len(cleanData) > 0 {
		// Decode character encoding
		decoded, err := p.decoder.Bytes(cleanData)
		if err != nil {
			decoded = cleanData
		}
		
		// Send to terminal processor
		select {
		case output <- decoded:
			p.batchesProcessed++
		default:
			// Channel full
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