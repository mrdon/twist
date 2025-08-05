package streaming

import (
	"testing"
	"twist/internal/proxy/database"
)

// MockScriptEngine implements ScriptEngine interface for testing
type MockScriptEngine struct {
	textEvents     []string
	textLineEvents []string
	autoTextEvents []string
	triggersCalled int
}

func NewMockScriptEngine() *MockScriptEngine {
	return &MockScriptEngine{
		textEvents:     make([]string, 0),
		textLineEvents: make([]string, 0),
		autoTextEvents: make([]string, 0),
		triggersCalled: 0,
	}
}

func (m *MockScriptEngine) ProcessText(text string) error {
	m.textEvents = append(m.textEvents, text)
	return nil
}

func (m *MockScriptEngine) ProcessTextLine(line string) error {
	m.textLineEvents = append(m.textLineEvents, line)
	return nil
}

func (m *MockScriptEngine) ActivateTriggers() error {
	m.triggersCalled++
	return nil
}

func (m *MockScriptEngine) ProcessAutoText(text string) error {
	m.autoTextEvents = append(m.autoTextEvents, text)
	return nil
}

func TestScriptEventProcessor_Creation(t *testing.T) {
	mockEngine := NewMockScriptEngine()
	processor := NewScriptEventProcessor(mockEngine)
	
	if !processor.IsEnabled() {
		t.Error("Expected processor to be enabled with valid script engine")
	}
	
	// Test with nil engine
	nilProcessor := NewScriptEventProcessor(nil)
	if nilProcessor.IsEnabled() {
		t.Error("Expected processor to be disabled with nil script engine")
	}
}

func TestScriptEventProcessor_FireTextEvent(t *testing.T) {
	mockEngine := NewMockScriptEngine()
	processor := NewScriptEventProcessor(mockEngine)
	
	testText := "Test text event"
	err := processor.FireTextEvent(testText, false)
	
	if err != nil {
		t.Errorf("Unexpected error firing text event: %v", err)
	}
	
	if len(mockEngine.textEvents) != 1 {
		t.Errorf("Expected 1 text event, got %d", len(mockEngine.textEvents))
	}
	
	if mockEngine.textEvents[0] != testText {
		t.Errorf("Expected text event '%s', got '%s'", testText, mockEngine.textEvents[0])
	}
}

func TestScriptEventProcessor_FireTextLineEvent(t *testing.T) {
	mockEngine := NewMockScriptEngine()
	processor := NewScriptEventProcessor(mockEngine)
	
	testLine := "Test text line event"
	err := processor.FireTextLineEvent(testLine, false)
	
	if err != nil {
		t.Errorf("Unexpected error firing text line event: %v", err)
	}
	
	if len(mockEngine.textLineEvents) != 1 {
		t.Errorf("Expected 1 text line event, got %d", len(mockEngine.textLineEvents))
	}
	
	if mockEngine.textLineEvents[0] != testLine {
		t.Errorf("Expected text line event '%s', got '%s'", testLine, mockEngine.textLineEvents[0])
	}
}

func TestScriptEventProcessor_FireActivateTriggers(t *testing.T) {
	mockEngine := NewMockScriptEngine()
	processor := NewScriptEventProcessor(mockEngine)
	
	err := processor.FireActivateTriggers()
	
	if err != nil {
		t.Errorf("Unexpected error activating triggers: %v", err)
	}
	
	if mockEngine.triggersCalled != 1 {
		t.Errorf("Expected triggers to be called 1 time, got %d", mockEngine.triggersCalled)
	}
}

func TestScriptEventProcessor_FireAutoTextEvent(t *testing.T) {
	mockEngine := NewMockScriptEngine()
	processor := NewScriptEventProcessor(mockEngine)
	
	testText := "Test auto text event"
	err := processor.FireAutoTextEvent(testText, false)
	
	if err != nil {
		t.Errorf("Unexpected error firing auto text event: %v", err)
	}
	
	if len(mockEngine.autoTextEvents) != 1 {
		t.Errorf("Expected 1 auto text event, got %d", len(mockEngine.autoTextEvents))
	}
	
	if mockEngine.autoTextEvents[0] != testText {
		t.Errorf("Expected auto text event '%s', got '%s'", testText, mockEngine.autoTextEvents[0])
	}
}

func TestScriptEventProcessor_ProcessLineWithScriptEvents(t *testing.T) {
	mockEngine := NewMockScriptEngine()
	processor := NewScriptEventProcessor(mockEngine)
	
	testLine := "Complete line with all events"
	err := processor.ProcessLineWithScriptEvents(testLine)
	
	if err != nil {
		t.Errorf("Unexpected error processing line with script events: %v", err)
	}
	
	// Verify all events were fired
	if len(mockEngine.textEvents) != 1 {
		t.Errorf("Expected 1 text event, got %d", len(mockEngine.textEvents))
	}
	
	if len(mockEngine.textLineEvents) != 1 {
		t.Errorf("Expected 1 text line event, got %d", len(mockEngine.textLineEvents))
	}
	
	if len(mockEngine.autoTextEvents) != 1 {
		t.Errorf("Expected 1 auto text event, got %d", len(mockEngine.autoTextEvents))
	}
	
	if mockEngine.triggersCalled != 1 {
		t.Errorf("Expected triggers to be called 1 time, got %d", mockEngine.triggersCalled)
	}
	
	// Verify all events received the same line
	if mockEngine.textEvents[0] != testLine {
		t.Errorf("TextEvent received wrong line: expected '%s', got '%s'", testLine, mockEngine.textEvents[0])
	}
	
	if mockEngine.textLineEvents[0] != testLine {
		t.Errorf("TextLineEvent received wrong line: expected '%s', got '%s'", testLine, mockEngine.textLineEvents[0])
	}
	
	if mockEngine.autoTextEvents[0] != testLine {
		t.Errorf("AutoTextEvent received wrong line: expected '%s', got '%s'", testLine, mockEngine.autoTextEvents[0])
	}
}

func TestScriptEventProcessor_DisabledEngine(t *testing.T) {
	processor := NewScriptEventProcessor(nil)
	
	// All methods should succeed but do nothing when engine is disabled
	err := processor.FireTextEvent("test", false)
	if err != nil {
		t.Errorf("Unexpected error with disabled engine: %v", err)
	}
	
	err = processor.FireTextLineEvent("test", false)
	if err != nil {
		t.Errorf("Unexpected error with disabled engine: %v", err)
	}
	
	err = processor.FireActivateTriggers()
	if err != nil {
		t.Errorf("Unexpected error with disabled engine: %v", err)
	}
	
	err = processor.FireAutoTextEvent("test", false)
	if err != nil {
		t.Errorf("Unexpected error with disabled engine: %v", err)
	}
	
	err = processor.ProcessLineWithScriptEvents("test")
	if err != nil {
		t.Errorf("Unexpected error with disabled engine: %v", err)
	}
}

func TestTWXParser_ScriptIntegration(t *testing.T) {
	// Create a test database
	db := database.NewDatabase()
	
	// Create parser with script integration
	parser := NewTWXParser(db, nil)
	mockEngine := NewMockScriptEngine()
	parser.SetScriptEngine(mockEngine)
	
	// Verify script engine is set
	if !parser.GetScriptEventProcessor().IsEnabled() {
		t.Error("Expected script event processor to be enabled after setting engine")
	}
	
	// Process a test line
	testLine := "Command [TL=150] (1234) ?"
	parser.ProcessString(testLine + "\r")
	
	// Verify script events were fired
	if len(mockEngine.textEvents) == 0 {
		t.Error("Expected text events to be fired during line processing")
	}
	
	if len(mockEngine.textLineEvents) == 0 {
		t.Error("Expected text line events to be fired during line processing")
	}
	
	if mockEngine.triggersCalled == 0 {
		t.Error("Expected triggers to be activated during line processing")
	}
}

func TestTWXParser_VersionDetectionScriptEvent(t *testing.T) {
	// Create a test database
	db := database.NewDatabase()
	
	// Create parser with script integration
	parser := NewTWXParser(db, nil)
	mockEngine := NewMockScriptEngine()
	parser.SetScriptEngine(mockEngine)
	
	// Process TWGS version detection line
	parser.ProcessString("TradeWars Game Server v2.20b\r")
	
	// Verify version was detected
	if parser.GetTWGSType() != 2 {
		t.Errorf("Expected TWGS type 2, got %d", parser.GetTWGSType())
	}
	
	// Verify script event was fired for version detection
	found := false
	for _, event := range mockEngine.textEvents {
		if event == "Selection (? for menu):" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected version detection script event 'Selection (? for menu):' to be fired")
	}
}

func TestScriptEventProcessor_SetScriptEngine(t *testing.T) {
	processor := NewScriptEventProcessor(nil)
	
	// Initially disabled
	if processor.IsEnabled() {
		t.Error("Expected processor to be disabled initially")
	}
	
	// Set engine and verify it's enabled
	mockEngine := NewMockScriptEngine()
	processor.SetScriptEngine(mockEngine)
	
	if !processor.IsEnabled() {
		t.Error("Expected processor to be enabled after setting engine")
	}
	
	// Test functionality works
	err := processor.FireTextEvent("test", false)
	if err != nil {
		t.Errorf("Unexpected error after setting engine: %v", err)
	}
	
	if len(mockEngine.textEvents) != 1 {
		t.Errorf("Expected 1 text event after setting engine, got %d", len(mockEngine.textEvents))
	}
}

// Integration test that mirrors Pascal TWX behavior
func TestTWXParser_PascalIntegrationBehavior(t *testing.T) {
	// Create a test database
	db := database.NewDatabase()
	
	// Create parser with script integration
	parser := NewTWXParser(db, nil)
	mockEngine := NewMockScriptEngine()
	parser.SetScriptEngine(mockEngine)
	
	// Test lines that should trigger script events (mirrors Pascal behavior)
	testLines := []string{
		"Command [TL=150] (1234) ?",
		"Sector  : 1234 in The Sphere",
		"Ships   : TestShip [Owned by Player]",
		"Incoming transmission from Player:",
		"Commerce report for StarPort Alpha:",
	}
	
	// Process each line
	for _, line := range testLines {
		parser.ProcessString(line + "\r")
	}
	
	// Verify script events were fired for each line
	if len(mockEngine.textEvents) != len(testLines) {
		t.Errorf("Expected %d text events, got %d", len(testLines), len(mockEngine.textEvents))
	}
	
	if len(mockEngine.textLineEvents) != len(testLines) {
		t.Errorf("Expected %d text line events, got %d", len(testLines), len(mockEngine.textLineEvents))
	}
	
	if len(mockEngine.autoTextEvents) != len(testLines) {
		t.Errorf("Expected %d auto text events, got %d", len(testLines), len(mockEngine.autoTextEvents))
	}
	
	if mockEngine.triggersCalled != len(testLines) {
		t.Errorf("Expected triggers to be called %d times, got %d", len(testLines), mockEngine.triggersCalled)
	}
}