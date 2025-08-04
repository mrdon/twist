package debug

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Logger provides centralized debug logging for the entire application
type Logger struct {
	file   *os.File
	logger *log.Logger
}

var globalLogger *Logger

// isTestMode detects if we're running in test mode
func isTestMode() bool {
	// Check if any argument contains "test" (e.g., go test, _test, etc.)
	for _, arg := range os.Args {
		if strings.Contains(arg, "test") || strings.HasSuffix(arg, ".test") {
			return true
		}
	}
	return false
}

// init creates the global debug logger
func init() {
	var err error
	if isTestMode() {
		// In test mode, log to stdout
		globalLogger = &Logger{
			file:   os.Stdout,
			logger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds),
		}
	} else {
		// In normal mode, log to file
		globalLogger, err = NewLogger("twist_debug.log")
		if err != nil {
			// Disable logging if we can't create the log file (don't write to stdout in TUI mode)
			globalLogger = nil
		}
	}
}

// NewLogger creates a new debug logger that writes to the specified file
func NewLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	logger := log.New(file, "", log.LstdFlags|log.Lmicroseconds)
	
	return &Logger{
		file:   file,
		logger: logger,
	}, nil
}

// Log writes a debug message with caller information
func Log(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.logger.Printf(format, args...)
	}
}

// LogError writes an error message
func LogError(err error, context string) {
	Log("ERROR in %s: %v", context, err)
}

// LogFunction logs function entry and exit
func LogFunction(funcName string) func() {
	Log("ENTER %s", funcName)
	start := time.Now()
	return func() {
		duration := time.Since(start)
		Log("EXIT %s (took %v)", funcName, duration)
	}
}

// LogState logs application state changes
func LogState(component, state string, details ...interface{}) {
	if len(details) > 0 {
		Log("STATE %s: %s - %v", component, state, details)
	} else {
		Log("STATE %s: %s", component, state)
	}
}

// LogDataChunk logs raw data chunks to a separate file for debugging network/terminal issues
func LogDataChunk(source string, data []byte) {
	if logFile, err := os.OpenFile("raw.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		// Use %q to encode escapes, then strip the outer quotes
		encoded := fmt.Sprintf("%q", string(data))
		// Remove the first and last quote characters
		if len(encoded) >= 2 {
			encoded = encoded[1 : len(encoded)-1]
		}
		
		fmt.Fprintf(logFile, "%s chunk (%d bytes):\n%s", source, len(data), encoded)
		
		// Insert extra newline if we detect \r\n (to separate chunks visually)
		dataStr := string(data)
		if len(dataStr) > 0 && (dataStr[len(dataStr)-1] == '\n' || dataStr[len(dataStr)-1] == '\r') {
			fmt.Fprintf(logFile, "\n")
		} else {
			fmt.Fprintf(logFile, "\n")
		}
		
		logFile.Close()
	} else {
		// Fallback to regular debug log if we can't open the data chunks file
		Log("ERROR: Could not open data chunks log: %v", err)
		Log("%s chunk (%d bytes): %q", source, len(data), string(data))
	}
}

// Close closes the debug logger file
func Close() {
	if globalLogger != nil && globalLogger.file != os.Stdout {
		globalLogger.file.Close()
	}
}