package types

import (
	"errors"
	"fmt"
)

// VMError represents a virtual machine execution error
type VMError struct {
	Message string
}

// Error implements the error interface
func (e *VMError) Error() string {
	return fmt.Sprintf("VM Error: %s", e.Message)
}

// Special errors for script control flow
var (
	ErrScriptPaused  = errors.New("script execution paused")
	ErrScriptStopped = errors.New("script execution stopped")
)
