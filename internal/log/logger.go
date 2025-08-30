package log

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger provides centralized debug logging for the entire application
type Logger struct {
	logger *slog.Logger
	file   *os.File
}

var globalLogger *Logger

// init creates the global debug logger with console output by default
func init() {
	// Default to console logging
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	globalLogger = &Logger{
		logger: slog.New(handler),
		file:   os.Stdout,
	}
}

// SetFileOutput configures the logger to write to the specified file
func SetFileOutput(filename string) error {
	logger, err := NewLogger(filename)
	if err != nil {
		return err
	}

	// Close existing file if it's not stdout
	if globalLogger != nil && globalLogger.file != os.Stdout {
		globalLogger.file.Close()
	}

	globalLogger = logger
	return nil
}

// NewLogger creates a new debug logger that writes to the specified file
func NewLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format to match old format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   slog.TimeKey,
					Value: slog.StringValue(a.Value.Time().Format("2006/01/02 15:04:05.000000")),
				}
			}
			return a
		},
	})

	return &Logger{
		logger: slog.New(handler),
		file:   file,
	}, nil
}

// Standard logging methods
func Debug(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.logger.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.logger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.logger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.logger.Error(msg, args...)
	}
}

// LogDataChunk logs raw data chunks to a separate file for debugging network/terminal issues
func LogDataChunk(direction string, data []byte) {
	if logFile, err := os.OpenFile("raw.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		// Use %q to encode escapes, then strip the outer quotes
		encoded := fmt.Sprintf("%q", string(data))
		// Remove the first and last quote characters
		if len(encoded) >= 2 {
			encoded = encoded[1 : len(encoded)-1]
		}

		fmt.Fprintf(logFile, "%s %s\n", direction, encoded)

		logFile.Close()
	} else {
		// Fallback to regular debug log if we can't open the data chunks file
		Error("Could not open data chunks log", "error", err)
		Debug("Raw data", "direction", direction, "data", string(data))
	}
}

// Close closes the debug logger file
func Close() {
	if globalLogger != nil && globalLogger.file != os.Stdout {
		globalLogger.file.Close()
	}
}

