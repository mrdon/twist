package terminal

import (
	"testing"
)

// MockANSIConverter for testing
type MockANSIConverter struct {
	calls []string
}

func (m *MockANSIConverter) ConvertANSIParams(params string) (fgHex, bgHex string, bold, underline, reverse bool) {
	m.calls = append(m.calls, params)
	// Return some dummy values
	return "#ffffff", "#000000", false, false, false
}

// MockTerminal to track processANSISequence calls
type MockTerminal struct {
	*Terminal
	processedSequences []string
}

func (mt *MockTerminal) processANSISequence(sequence []byte) {
	mt.processedSequences = append(mt.processedSequences, string(sequence))
	// Call the original method
	mt.Terminal.processANSISequence(sequence)
}

func TestANSISequenceParsing(t *testing.T) {
	// Create a terminal with our mock converter
	mockConverter := &MockANSIConverter{}
	terminal := NewTerminalWithConverter(80, 24, mockConverter)
	
	// Test the sequence: "\x1b[31m NO \x1b[1mPVP\x1b[36m"
	testData := []byte("\x1b[31m NO \x1b[1mPVP\x1b[36m")
	
	t.Logf("Processing test data: %q", string(testData))
	t.Logf("Raw bytes: %x", testData)
	
	// Process the data
	terminal.Write(testData)
	
	// Check that we called ConvertANSIParams exactly 3 times
	expectedCalls := []string{"31", "1", "36"}
	
	t.Logf("Converter was called %d times with params: %v", len(mockConverter.calls), mockConverter.calls)
	
	if len(mockConverter.calls) != len(expectedCalls) {
		t.Errorf("Expected %d ANSI sequence calls, got %d", len(expectedCalls), len(mockConverter.calls))
		t.Errorf("Expected calls: %v", expectedCalls)
		t.Errorf("Actual calls: %v", mockConverter.calls)
		return
	}
	
	// Check that the parameters match what we expect
	for i, expected := range expectedCalls {
		if mockConverter.calls[i] != expected {
			t.Errorf("Call %d: expected params %q, got %q", i, expected, mockConverter.calls[i])
		}
	}
	
	t.Logf("✓ Successfully parsed %d ANSI sequences: %v", len(mockConverter.calls), mockConverter.calls)
}