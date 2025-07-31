package debug

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Logger provides centralized debug logging for the entire application
type Logger struct {
	file   *os.File
	logger *log.Logger
}

var globalLogger *Logger

// init creates the global debug logger
func init() {
	var err error
	globalLogger, err = NewLogger("twist_debug.log")
	if err != nil {
		// Fallback to stdout if we can't create the log file
		globalLogger = &Logger{
			file:   os.Stdout,
			logger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		}
	}
}

// NewLogger creates a new debug logger that writes to the specified file
func NewLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	logger := log.New(file, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
	
	return &Logger{
		file:   file,
		logger: logger,
	}, nil
}

// Log writes a debug message with caller information
func Log(format string, args ...interface{}) {
	if globalLogger != nil {
		// Get caller information
		_, file, line, ok := runtime.Caller(1)
		if ok {
			file = filepath.Base(file)
			prefix := fmt.Sprintf("[%s:%d] ", file, line)
			globalLogger.logger.Printf(prefix+format, args...)
		} else {
			globalLogger.logger.Printf(format, args...)
		}
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

// Close closes the debug logger file
func Close() {
	if globalLogger != nil && globalLogger.file != os.Stdout {
		globalLogger.file.Close()
	}
}