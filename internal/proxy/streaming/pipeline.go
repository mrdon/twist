package streaming

import (
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"twist/internal/api"
	"twist/internal/debug"
	"twist/internal/proxy/database"
	"twist/internal/proxy/interfaces"
	"twist/internal/telnet"
)

// External script engine interface from scripting package
type ExternalScriptEngine interface {
	ProcessText(text string) error
	ProcessTextLine(line string) error
	ActivateTriggers() error
	ProcessAutoText(text string) error
	UpdateCurrentLine(text string) error
}

// scriptEngineAdapter adapts between external and internal interfaces
type scriptEngineAdapter struct {
	engine ExternalScriptEngine
}

func (a *scriptEngineAdapter) ProcessText(text string) error {
	return a.engine.ProcessText(text)
}

func (a *scriptEngineAdapter) ProcessTextLine(line string) error {
	return a.engine.ProcessTextLine(line)
}

func (a *scriptEngineAdapter) ActivateTriggers() error {
	return a.engine.ActivateTriggers()
}

func (a *scriptEngineAdapter) ProcessAutoText(text string) error {
	return a.engine.ProcessAutoText(text)
}

func (a *scriptEngineAdapter) UpdateCurrentLine(text string) error {
	return a.engine.UpdateCurrentLine(text)
}

// ScriptManager interface for script processing
type ScriptManager interface {
	ProcessGameLine(line string) error
	GetEngine() interfaces.ScriptEngine // Return properly typed interface
}

// StateManager interface for game state updates (avoids circular import with proxy)
type StateManager interface {
	SetCurrentSector(sectorNum int)
	SetPlayerName(name string)
}

// GameDetector interface for game detection
type GameDetector interface {
	ProcessLine(line string)
}

// Pipeline provides high-performance streaming from network to terminal buffer
type Pipeline struct {
	// Processing layers (no async channels needed - synchronous like TWX)
	telnetHandler *telnet.Handler
	tuiAPI        api.TuiAPI // Direct TuiAPI reference
	decoder       *encoding.Decoder
	twxParser     *TWXParser
	scriptManager ScriptManager
	stateManager  StateManager // Game state updates
	gameDetector  GameDetector // Game detection

	// State
	running bool

	// Metrics
	bytesProcessed   uint64
	batchesProcessed uint64
}

// NewPipeline creates an optimized streaming pipeline
func NewPipeline(tuiAPI api.TuiAPI, db database.Database) *Pipeline {
	return &Pipeline{
		tuiAPI:    tuiAPI, // Direct TuiAPI reference
		decoder:   charmap.CodePage437.NewDecoder(),
		twxParser: NewTWXParser(db, tuiAPI),
	}
}

// NewPipelineWithScriptManager creates an optimized streaming pipeline with script support
func NewPipelineWithScriptManager(tuiAPI api.TuiAPI, db database.Database, scriptManager ScriptManager) *Pipeline {
	p := &Pipeline{
		tuiAPI:        tuiAPI, // Direct TuiAPI reference
		decoder:       charmap.CodePage437.NewDecoder(),
		twxParser:     NewTWXParser(db, tuiAPI),
		scriptManager: scriptManager,
	}

	// Initialize telnet handler with no writer - scripts don't need to write back to connection
	p.telnetHandler = telnet.NewHandler(nil)

	return p
}

// NewPipelineWithWriter creates an optimized streaming pipeline with a writer for telnet negotiation
func NewPipelineWithWriter(tuiAPI api.TuiAPI, db database.Database, scriptManager ScriptManager, stateManager StateManager, gameDetector GameDetector, writer func([]byte) error) *Pipeline {
	p := &Pipeline{
		tuiAPI:        tuiAPI, // Direct TuiAPI reference
		decoder:       charmap.CodePage437.NewDecoder(),
		twxParser:     NewTWXParser(db, tuiAPI),
		scriptManager: scriptManager,
		stateManager:  stateManager,
		gameDetector:  gameDetector,
	}

	// Initialize telnet handler with proper writer for negotiation
	p.telnetHandler = telnet.NewHandler(writer)

	// Connect script engine to TWX parser for script events
	if scriptManager != nil {
		engineInterface := scriptManager.GetEngine()
		if engineInterface != nil {
			// Type assert the engine to our external interface
			if engine, ok := engineInterface.(ExternalScriptEngine); ok {
				// Create adapter to convert between interface types
				adapter := &scriptEngineAdapter{engine: engine}
				p.twxParser.SetScriptEngine(adapter)
			}
		}
	}

	return p
}

// Start begins the streaming pipeline
func (p *Pipeline) Start() {
	p.running = true

	// No goroutine needed - processing is now synchronous like TWX
}

// Stop gracefully shuts down the pipeline
func (p *Pipeline) Stop() {
	p.running = false
}

// Write feeds raw data into the pipeline
func (p *Pipeline) Write(data []byte) {
	if !p.running {
		return
	}

	// Process data synchronously like TWX (no goroutines or channels)
	p.processDataSync(data)
}

// processDataSync processes data synchronously (replaces async batchProcessor)
func (p *Pipeline) processDataSync(rawData []byte) {
	debug.LogDataChunk("<<", rawData)
	// Process telnet commands immediately
	cleanData := p.telnetHandler.ProcessData(rawData)

	if len(cleanData) > 0 {
		// Use standard CP437 to UTF-8 conversion
		decoded, err := p.decoder.Bytes(cleanData)
		if err != nil {
			decoded = cleanData
		}

		// Process through game detector FIRST with full decoded data (like original async pipeline)
		if p.gameDetector != nil {
			p.gameDetector.ProcessLine(string(decoded))
		}

		// Split into lines to process synchronously like TWX for scripts and parsing
		lines := strings.Split(string(decoded), "\n")

		for i, line := range lines {
			// Skip empty lines except the last one (which might be a partial line)
			if line == "" && i < len(lines)-1 {
				continue
			}

			// SINGLE PROCESSING PATH like TWX Pascal: let TWX parser handle everything
			// This will update CURRENTLINE AND fire script triggers in correct sequence
			if p.twxParser != nil {
				p.twxParser.ProcessInBound(line)
			}
		}

		// Send full decoded data to TUI
		if p.tuiAPI != nil {
			p.tuiAPI.OnData(decoded)
		}
		p.bytesProcessed += uint64(len(rawData))
		p.batchesProcessed++
	}
}

// SendTelnetNegotiation sends initial telnet negotiation
func (p *Pipeline) SendTelnetNegotiation() error {
	return p.telnetHandler.SendInitialNegotiation()
}

// GetMetrics returns pipeline performance metrics
func (p *Pipeline) GetMetrics() (bytesProcessed, batchesProcessed uint64) {
	return p.bytesProcessed, p.batchesProcessed
}

// GetParser returns the TWX parser instance
func (p *Pipeline) GetParser() *TWXParser {
	return p.twxParser
}

// InjectTUIData sends data directly to the TUI without processing through the pipeline
// This is used for script echo output that should display in the terminal but not go to the server
func (p *Pipeline) InjectTUIData(data []byte) {
	if p.tuiAPI != nil {
		// Apply the same character encoding conversion as the normal pipeline
		decoded, err := p.decoder.Bytes(data)
		if err != nil {
			decoded = data
		}
		p.tuiAPI.OnData(decoded)
	} else {
		panic("critical error: missing tui api")
	}
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
