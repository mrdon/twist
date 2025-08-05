package streaming

import (
	"testing"
	"time"
	"twist/internal/proxy/database"
)

// TestObserver implements IObserver for testing
type TestObserver struct {
	id             string
	receivedEvents []Event
}

func (o *TestObserver) Update(subject ISubject, event Event) {
	o.receivedEvents = append(o.receivedEvents, event)
}

func (o *TestObserver) GetObserverID() string {
	return o.id
}

func (o *TestObserver) GetReceivedEvents() []Event {
	return o.receivedEvents
}

func (o *TestObserver) ClearEvents() {
	o.receivedEvents = []Event{}
}

func NewTestObserver(id string) *TestObserver {
	return &TestObserver{
		id:             id,
		receivedEvents: make([]Event, 0),
	}
}

func TestObserverPatternBasic(t *testing.T) {
	// Setup
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	observer1 := NewTestObserver("observer1")
	observer2 := NewTestObserver("observer2")
	
	// Test attaching observers
	parser.Attach(observer1)
	parser.Attach(observer2)
	
	// Fire a test event
	testEvent := Event{
		Type: EventStateChange,
		Data: map[string]interface{}{
			"property": "test",
			"value":    "testValue",
		},
		Source: "TestSource",
	}
	
	parser.Notify(testEvent)
	
	// Verify both observers received the event
	events1 := observer1.GetReceivedEvents()
	events2 := observer2.GetReceivedEvents()
	
	if len(events1) != 1 {
		t.Errorf("Observer1 expected 1 event, got %d", len(events1))
	}
	if len(events2) != 1 {
		t.Errorf("Observer2 expected 1 event, got %d", len(events2))
	}
	
	if events1[0].Type != EventStateChange {
		t.Errorf("Observer1 expected EventStateChange, got %d", int(events1[0].Type))
	}
	if events2[0].Type != EventStateChange {
		t.Errorf("Observer2 expected EventStateChange, got %d", int(events2[0].Type))
	}
}

func TestObserverDetach(t *testing.T) {
	// Setup
	db := database.NewDatabase()
	parser := NewTWXParser(db, nil)
	
	observer1 := NewTestObserver("observer1")
	observer2 := NewTestObserver("observer2")
	
	// Attach observers
	parser.Attach(observer1)
	parser.Attach(observer2)
	
	// Detach observer1
	parser.Detach("observer1")
	
	// Fire event
	testEvent := Event{
		Type:   EventStateChange,
		Data:   map[string]interface{}{"test": "value"},
		Source: "TestSource",
	}
	
	parser.Notify(testEvent)
	
	// Verify only observer2 received the event
	events1 := observer1.GetReceivedEvents()
	events2 := observer2.GetReceivedEvents()
	
	if len(events1) != 0 {
		t.Errorf("Observer1 expected 0 events after detach, got %d", len(events1))
	}
	if len(events2) != 1 {
		t.Errorf("Observer2 expected 1 event, got %d", len(events2))
	}
}

func TestEventBusBasic(t *testing.T) {
	eventBus := NewEventBus()
	
	var receivedEvent Event
	handler := func(event Event) {
		receivedEvent = event
	}
	
	// Subscribe to event
	subscriptionID := eventBus.Subscribe(EventText, handler)
	
	// Fire event
	testEvent := Event{
		Type: EventText,
		Data: map[string]interface{}{
			"line":     "test line",
			"outbound": false,
		},
		Source: "TestSource",
	}
	
	eventBus.Fire(testEvent)
	
	// Verify event was received
	if receivedEvent.Type != EventText {
		t.Errorf("Expected EventText, got %d", int(receivedEvent.Type))
	}
	
	data := receivedEvent.Data.(map[string]interface{})
	if data["line"] != "test line" {
		t.Errorf("Expected 'test line', got %v", data["line"])
	}
	
	// Test unsubscribe
	eventBus.Unsubscribe(EventText, subscriptionID)
	
	// Fire another event
	receivedEvent = Event{} // Reset
	eventBus.Fire(testEvent)
	
	// Verify event was not received
	if receivedEvent.Type == EventText {
		t.Error("Event should not have been received after unsubscribe")
	}
}

func TestEventBusAsync(t *testing.T) {
	eventBus := NewEventBus()
	
	eventReceived := make(chan bool, 1)
	var receivedEvent Event
	
	handler := func(event Event) {
		receivedEvent = event
		eventReceived <- true
	}
	
	// Subscribe to event
	eventBus.Subscribe(EventText, handler)
	
	// Fire async event
	testEvent := Event{
		Type: EventText,
		Data: map[string]interface{}{
			"line":     "async test line",
			"outbound": true,
		},
		Source: "TestSource",
	}
	
	eventBus.FireAsync(testEvent)
	
	// Wait for async event
	select {
	case <-eventReceived:
		// Event received
	case <-time.After(100 * time.Millisecond):
		t.Error("Async event not received within timeout")
	}
	
	// Verify event content
	if receivedEvent.Type != EventText {
		t.Errorf("Expected EventText, got %d", int(receivedEvent.Type))
	}
	
	data := receivedEvent.Data.(map[string]interface{})
	if data["line"] != "async test line" {
		t.Errorf("Expected 'async test line', got %v", data["line"])
	}
	if data["outbound"] != true {
		t.Errorf("Expected outbound=true, got %v", data["outbound"])
	}
}

func TestScriptInterpreterEvents(t *testing.T) {
	eventBus := NewEventBus()
	scriptInterpreter := NewScriptInterpreter(eventBus)
	
	var receivedEvents []Event
	handler := func(event Event) {
		receivedEvents = append(receivedEvents, event)
	}
	
	// Subscribe to different event types
	eventBus.Subscribe(EventText, handler)
	eventBus.Subscribe(EventTextLine, handler)
	eventBus.Subscribe(EventTrigger, handler)
	eventBus.Subscribe(EventAutoText, handler)
	
	// Test TextEvent
	scriptInterpreter.TextEvent("test text", false)
	
	// Test TextLineEvent
	scriptInterpreter.TextLineEvent("test line", true)
	
	// Test ActivateTriggers
	scriptInterpreter.ActivateTriggers()
	
	// Test AutoTextEvent
	scriptInterpreter.AutoTextEvent("auto text", false)
	
	// Verify all events were received
	if len(receivedEvents) != 4 {
		t.Errorf("Expected 4 events, got %d", len(receivedEvents))
	}
	
	// Check event types
	expectedTypes := []EventType{EventText, EventTextLine, EventTrigger, EventAutoText}
	for i, expectedType := range expectedTypes {
		if receivedEvents[i].Type != expectedType {
			t.Errorf("Event %d: expected type %d, got %d", i, int(expectedType), int(receivedEvents[i].Type))
		}
	}
}

func TestModExtractorInterface(t *testing.T) {
	// Test that TWXParser implements IModExtractor interface
	db := database.NewDatabase()
	var parser IModExtractor = NewTWXParser(db, nil)
	
	// Test interface methods
	currentSector := parser.GetCurrentSector()
	if currentSector != 0 {
		t.Errorf("Expected initial sector 0, got %d", currentSector)
	}
	
	currentDisplay := parser.GetCurrentDisplay()
	if currentDisplay != DisplayNone {
		t.Errorf("Expected DisplayNone, got %d", int(currentDisplay))
	}
	
	// Test setting display
	parser.SetCurrentDisplay(DisplaySector)
	newDisplay := parser.GetCurrentDisplay()
	if newDisplay != DisplaySector {
		t.Errorf("Expected DisplaySector, got %d", int(newDisplay))
	}
	
	// Test event bus
	eventBus := parser.GetEventBus()
	if eventBus == nil {
		t.Error("Expected event bus to be initialized")
	}
	
	// Test database
	database := parser.GetDatabase()
	if database == nil {
		t.Error("Expected database to be set")
	}
}

func TestIntegrationParsingWithEvents(t *testing.T) {
	// Create an in-memory test database
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.CloseDatabase()
	
	parser := NewTWXParser(db, nil)
	
	// Create test observer
	observer := NewTestObserver("integration_test")
	parser.Attach(observer)
	
	// Test event bus subscription
	var eventBusEvents []Event
	handler := func(event Event) {
		eventBusEvents = append(eventBusEvents, event)
	}
	
	eventBus := parser.GetEventBus()
	eventBus.Subscribe(EventSectorComplete, handler)
	eventBus.Subscribe(EventStateChange, handler)
	
	// Process sector data that should trigger events
	testData := []string{
		"Sector  : 1000 in Test Space",
		"Beacon  : Test Beacon",
		"Warps to Sector(s) : 1001 - 1002",
		"Command [TL=9999]: ",
	}
	
	for _, line := range testData {
		parser.ProcessInBound(line + "\r")
	}
	
	// Verify observer received events
	observerEvents := observer.GetReceivedEvents()
	if len(observerEvents) == 0 {
		t.Error("Observer should have received events")
	}
	
	// Verify event bus received events
	if len(eventBusEvents) == 0 {
		t.Error("Event bus should have received events")
	}
	
	// Check for sector complete event
	foundSectorComplete := false
	for _, event := range eventBusEvents {
		if event.Type == EventSectorComplete {
			foundSectorComplete = true
			break
		}
	}
	
	if !foundSectorComplete {
		t.Error("Expected to find EventSectorComplete in event bus events")
	}
	
	t.Logf("Observer received %d events", len(observerEvents))
	t.Logf("Event bus received %d events", len(eventBusEvents))
}