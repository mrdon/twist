package triggers

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"twist/internal/scripting/types"
)

// Manager manages script triggers
type Manager struct {
	triggers map[string]types.TriggerInterface
	mutex    sync.RWMutex
	vm       types.VMInterface
	nextID   int
}

// NewManager creates a new trigger manager
func NewManager(vm types.VMInterface) *Manager {
	return &Manager{
		triggers: make(map[string]types.TriggerInterface),
		vm:       vm,
		nextID:   1,
	}
}

// AddTrigger adds a trigger to the manager
func (m *Manager) AddTrigger(trigger types.TriggerInterface) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.triggers[trigger.GetID()] = trigger
	return nil
}

// RemoveTrigger removes a trigger by ID
func (m *Manager) RemoveTrigger(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.triggers, id)
	return nil
}

// RemoveAllTriggers removes all triggers
func (m *Manager) RemoveAllTriggers() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.triggers = make(map[string]types.TriggerInterface)
	return nil
}

// GetTrigger gets a trigger by ID
func (m *Manager) GetTrigger(id string) types.TriggerInterface {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.triggers[id]
}

// GetAllTriggers returns all triggers
func (m *Manager) GetAllTriggers() map[string]types.TriggerInterface {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	result := make(map[string]types.TriggerInterface)
	for k, v := range m.triggers {
		result[k] = v
	}
	return result
}

// ProcessText processes incoming text against text triggers
func (m *Manager) ProcessText(text string) error {
	m.mutex.RLock()
	triggers := make([]types.TriggerInterface, 0)
	for _, trigger := range m.triggers {
		if trigger.GetType() == types.TriggerText && trigger.IsActive() {
			triggers = append(triggers, trigger)
		}
	}
	m.mutex.RUnlock()
	
	for _, trigger := range triggers {
		if trigger.Matches(text) {
			if err := trigger.Execute(m.vm); err != nil {
				return err
			}
			
			// Handle lifecycle
			if trigger.GetLifeCycle() > 0 {
				// Decrement lifecycle and remove if expired
				if lifecycleTrigger, ok := trigger.(*types.TextTrigger); ok {
					lifecycleTrigger.LifeCycle--
					if lifecycleTrigger.LifeCycle <= 0 {
						m.RemoveTrigger(trigger.GetID())
					}
				}
			}
		}
	}
	
	return nil
}

// ProcessTextLine processes incoming text line against text line triggers
func (m *Manager) ProcessTextLine(line string) error {
	m.mutex.RLock()
	triggers := make([]types.TriggerInterface, 0)
	for _, trigger := range m.triggers {
		if trigger.GetType() == types.TriggerTextLine && trigger.IsActive() {
			triggers = append(triggers, trigger)
		}
	}
	m.mutex.RUnlock()
	
	for _, trigger := range triggers {
		if trigger.Matches(line) {
			if err := trigger.Execute(m.vm); err != nil {
				return err
			}
			
			// Handle lifecycle
			if trigger.GetLifeCycle() > 0 {
				if lifecycleTrigger, ok := trigger.(*types.TextLineTrigger); ok {
					lifecycleTrigger.LifeCycle--
					if lifecycleTrigger.LifeCycle <= 0 {
						m.RemoveTrigger(trigger.GetID())
					}
				}
			}
		}
	}
	
	return nil
}

// ProcessTextOut processes outgoing text against text out triggers
func (m *Manager) ProcessTextOut(text string) error {
	m.mutex.RLock()
	triggers := make([]types.TriggerInterface, 0)
	for _, trigger := range m.triggers {
		if trigger.GetType() == types.TriggerTextOut && trigger.IsActive() {
			triggers = append(triggers, trigger)
		}
	}
	m.mutex.RUnlock()
	
	for _, trigger := range triggers {
		if trigger.Matches(text) {
			if err := trigger.Execute(m.vm); err != nil {
				return err
			}
			
			// Handle lifecycle
			if trigger.GetLifeCycle() > 0 {
				if lifecycleTrigger, ok := trigger.(*types.TextOutTrigger); ok {
					lifecycleTrigger.LifeCycle--
					if lifecycleTrigger.LifeCycle <= 0 {
						m.RemoveTrigger(trigger.GetID())
					}
				}
			}
		}
	}
	
	return nil
}

// ProcessEvent processes system events against event triggers
func (m *Manager) ProcessEvent(eventName string) error {
	m.mutex.RLock()
	triggers := make([]types.TriggerInterface, 0)
	for _, trigger := range m.triggers {
		if trigger.GetType() == types.TriggerEvent && trigger.IsActive() {
			triggers = append(triggers, trigger)
		}
	}
	m.mutex.RUnlock()
	
	for _, trigger := range triggers {
		if trigger.Matches(eventName) {
			if err := trigger.Execute(m.vm); err != nil {
				return err
			}
			
			// Handle lifecycle
			if trigger.GetLifeCycle() > 0 {
				if lifecycleTrigger, ok := trigger.(*types.EventTrigger); ok {
					lifecycleTrigger.LifeCycle--
					if lifecycleTrigger.LifeCycle <= 0 {
						m.RemoveTrigger(trigger.GetID())
					}
				}
			}
		}
	}
	
	return nil
}

// ProcessDelayTriggers processes delay triggers
func (m *Manager) ProcessDelayTriggers() error {
	m.mutex.RLock()
	triggers := make([]types.TriggerInterface, 0)
	for _, trigger := range m.triggers {
		if trigger.GetType() == types.TriggerDelay && trigger.IsActive() {
			triggers = append(triggers, trigger)
		}
	}
	m.mutex.RUnlock()
	
	for _, trigger := range triggers {
		if trigger.Matches("") { // Delay triggers don't need input text
			if err := trigger.Execute(m.vm); err != nil {
				return err
			}
			
			// Delay triggers are one-shot by default
			m.RemoveTrigger(trigger.GetID())
		}
	}
	
	return nil
}

// SetTextTrigger creates a text trigger
func (m *Manager) SetTextTrigger(pattern, response, label string) (string, error) {
	id := m.generateID()
	
	trigger := &types.TextTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerText,
			Label:     label,
			Value:     pattern,
			Response:  response,
			Active:    true,
			LifeCycle: -1, // Permanent by default
		},
	}
	
	return id, m.AddTrigger(trigger)
}

// SetTextLineTrigger creates a text line trigger
func (m *Manager) SetTextLineTrigger(pattern, response, label string) (string, error) {
	id := m.generateID()
	
	trigger := &types.TextLineTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerTextLine,
			Label:     label,
			Value:     pattern,
			Response:  response,
			Active:    true,
			LifeCycle: -1, // Permanent by default
		},
	}
	
	return id, m.AddTrigger(trigger)
}

// SetTextOutTrigger creates a text out trigger
func (m *Manager) SetTextOutTrigger(pattern, response, label string) (string, error) {
	id := m.generateID()
	
	trigger := &types.TextOutTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerTextOut,
			Label:     label,
			Value:     pattern,
			Response:  response,
			Active:    true,
			LifeCycle: -1, // Permanent by default
		},
	}
	
	return id, m.AddTrigger(trigger)
}

// SetDelayTrigger creates a delay trigger
func (m *Manager) SetDelayTrigger(delayMs float64, label string) (string, error) {
	id := m.generateID()
	delay := time.Duration(delayMs) * time.Millisecond
	
	trigger := &types.DelayTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerDelay,
			Label:     label,
			Value:     strconv.FormatFloat(delayMs, 'f', -1, 64),
			Active:    true,
			LifeCycle: 1, // One-shot
		},
		Delay:     delay,
		StartTime: time.Now(),
	}
	
	// Set up timer
	trigger.Timer = time.AfterFunc(delay, func() {
		// Timer expired, trigger can now match
	})
	
	return id, m.AddTrigger(trigger)
}

// SetEventTrigger creates an event trigger
func (m *Manager) SetEventTrigger(eventName, response, label string) (string, error) {
	id := m.generateID()
	
	trigger := &types.EventTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerEvent,
			Label:     label,
			Value:     eventName,
			Response:  response,
			Active:    true,
			LifeCycle: -1, // Permanent by default
		},
		EventName: eventName,
	}
	
	return id, m.AddTrigger(trigger)
}

// SetAutoTrigger creates an auto trigger
func (m *Manager) SetAutoTrigger(pattern, response string, persistent bool) (string, error) {
	id := m.generateID()
	
	lifecycle := 1 // One-shot
	if persistent {
		lifecycle = -1 // Permanent
	}
	
	trigger := &types.AutoTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerAuto,
			Value:     pattern,
			Response:  response,
			Active:    true,
			LifeCycle: lifecycle,
		},
		Persistent: persistent,
	}
	
	return id, m.AddTrigger(trigger)
}

// SetAutoTextTrigger creates an auto text trigger
func (m *Manager) SetAutoTextTrigger(pattern, response string, persistent bool) (string, error) {
	id := m.generateID()
	
	lifecycle := 1 // One-shot
	if persistent {
		lifecycle = -1 // Permanent
	}
	
	trigger := &types.AutoTextTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerAutoText,
			Value:     pattern,
			Response:  response,
			Active:    true,
			LifeCycle: lifecycle,
		},
		Persistent: persistent,
	}
	
	return id, m.AddTrigger(trigger)
}

// generateID generates a unique trigger ID
func (m *Manager) generateID() string {
	id := fmt.Sprintf("trigger_%d", m.nextID)
	m.nextID++
	return id
}

// HasTriggers returns true if there are active triggers
func (m *Manager) HasTriggers() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	for _, trigger := range m.triggers {
		if trigger.IsActive() {
			return true
		}
	}
	return false
}

// GetTriggersByType returns all triggers of a specific type
func (m *Manager) GetTriggersByType(triggerType types.TriggerType) []types.TriggerInterface {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	result := make([]types.TriggerInterface, 0)
	for _, trigger := range m.triggers {
		if trigger.GetType() == triggerType && trigger.IsActive() {
			result = append(result, trigger)
		}
	}
	return result
}

// ActivateTrigger activates a trigger by ID
func (m *Manager) ActivateTrigger(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	trigger, exists := m.triggers[id]
	if !exists {
		return fmt.Errorf("trigger not found: %s", id)
	}
	
	trigger.SetActive(true)
	return nil
}

// DeactivateTrigger deactivates a trigger by ID
func (m *Manager) DeactivateTrigger(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	trigger, exists := m.triggers[id]
	if !exists {
		return fmt.Errorf("trigger not found: %s", id)
	}
	
	trigger.SetActive(false)
	return nil
}

// GetTriggerCount returns the number of active triggers
func (m *Manager) GetTriggerCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	count := 0
	for _, trigger := range m.triggers {
		if trigger.IsActive() {
			count++
		}
	}
	return count
}

// ListTriggers returns a formatted list of all triggers
func (m *Manager) ListTriggers() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var result strings.Builder
	result.WriteString("Active Triggers:\n")
	
	for id, trigger := range m.triggers {
		if trigger.IsActive() {
			result.WriteString(fmt.Sprintf("ID: %s, Type: %d, Pattern: %s, Label: %s\n",
				id, trigger.GetType(), trigger.GetValue(), trigger.GetLabel()))
		}
	}
	
	if result.Len() == len("Active Triggers:\n") {
		result.WriteString("No active triggers.\n")
	}
	
	return result.String()
}