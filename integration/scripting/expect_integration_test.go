package scripting

import (
	"strings"
	"testing"
)


// TestExpectTimeout demonstrates timeout handling
func TestExpectTimeout(t *testing.T) {
	expectEngine := NewSimpleExpectEngine(t, nil)
	expectEngine.AddOutput("Some initial output")

	expectScript := `
timeout "100ms"
expect "This text will never appear"
log "Should not reach this line"
`

	err := expectEngine.Run(expectScript)
	if err == nil {
		t.Fatal("Expected timeout error but test passed")
	}

	if !strings.Contains(err.Error(), "timeout waiting") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// Helper function to avoid conflicts
func expectSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}