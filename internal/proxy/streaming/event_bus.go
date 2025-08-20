package streaming

import (
	"fmt"
	"sync"
	"time"
)

// EventBus implements the IEventBus interface for module communication
type EventBus struct {
	subscribers map[EventType]map[string]EventHandler
	mutex       sync.RWMutex
	nextID      int
}

// NewEventBus creates a new event bus instance
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType]map[string]EventHandler),
		nextID:      1,
	}
}

// Subscribe registers an event handler for a specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) string {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	// Create subscription ID
	subscriptionID := fmt.Sprintf("sub_%d", eb.nextID)
	eb.nextID++

	// Initialize event type map if needed
	if eb.subscribers[eventType] == nil {
		eb.subscribers[eventType] = make(map[string]EventHandler)
	}

	// Add handler
	eb.subscribers[eventType][subscriptionID] = handler

	return subscriptionID
}

// Unsubscribe removes an event handler
func (eb *EventBus) Unsubscribe(eventType EventType, subscriptionID string) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if handlers, exists := eb.subscribers[eventType]; exists {
		delete(handlers, subscriptionID)

		// Clean up empty event type map
		if len(handlers) == 0 {
			delete(eb.subscribers, eventType)
		}
	}
}

// Fire synchronously delivers an event to all subscribers
func (eb *EventBus) Fire(event Event) {
	eb.mutex.RLock()
	handlers, exists := eb.subscribers[event.Type]
	eb.mutex.RUnlock()

	if !exists {
		return
	}

	// Set timestamp if not already set
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixNano()
	}

	// Call all handlers synchronously
	for _, handler := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			handler(event)
		}()
	}
}

// FireAsync asynchronously delivers an event to all subscribers
func (eb *EventBus) FireAsync(event Event) {
	eb.mutex.RLock()
	handlers, exists := eb.subscribers[event.Type]
	eb.mutex.RUnlock()

	if !exists {
		return
	}

	// Set timestamp if not already set
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixNano()
	}

	// Call all handlers asynchronously
	for subscriptionID, handler := range handlers {
		go func(id string, h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			h(event)
		}(subscriptionID, handler)
	}
}

// GetSubscriberCount returns the number of subscribers for an event type
func (eb *EventBus) GetSubscriberCount(eventType EventType) int {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	if handlers, exists := eb.subscribers[eventType]; exists {
		return len(handlers)
	}
	return 0
}

// GetAllEventTypes returns all event types that have subscribers
func (eb *EventBus) GetAllEventTypes() []EventType {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	var eventTypes []EventType
	for eventType := range eb.subscribers {
		eventTypes = append(eventTypes, eventType)
	}
	return eventTypes
}

// Clear removes all subscribers (useful for testing)
func (eb *EventBus) Clear() {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	eb.subscribers = make(map[EventType]map[string]EventHandler)
}
