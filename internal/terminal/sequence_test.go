package terminal

import (
	"testing"
)

// Test the specific parsing logic that might be problematic
func TestANSIParsingLogic(t *testing.T) {
	testData := []byte("\x1b[31m NO \x1b[1mPVP\x1b[36m")

	t.Logf("Manual parsing test:")
	i := 0
	sequenceCount := 0

	for i < len(testData) {
		if testData[i] == '\x1b' && i+1 < len(testData) && testData[i+1] == '[' {
			start := i
			i += 2 // Skip \x1b[

			// Find the end of the escape sequence (letter that terminates it)
			for i < len(testData) && !((testData[i] >= 'a' && testData[i] <= 'z') || (testData[i] >= 'A' && testData[i] <= 'Z')) {
				i++
			}

			if i < len(testData) {
				// Include the terminating letter
				i++
				sequence := testData[start:i]
				sequenceCount++
				t.Logf("  Found sequence %d: %q (hex: %x)", sequenceCount, string(sequence), sequence)
			}
		} else {
			// Regular character
			if testData[i] >= 32 && testData[i] <= 126 { // Printable ASCII
				t.Logf("  Character: '%c'", testData[i])
			}
			i++
		}
	}

	if sequenceCount != 3 {
		t.Errorf("Expected to find 3 ANSI sequences, found %d", sequenceCount)
	}
}
