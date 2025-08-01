package types

import "fmt"

// VMError represents a virtual machine execution error
type VMError struct {
	Message string
}

// Error implements the error interface
func (e *VMError) Error() string {
	return fmt.Sprintf("VM Error: %s", e.Message)
}