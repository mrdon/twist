package streaming

import (
	"fmt"
)

// ScriptInterpreter implements the IScriptInterpreter interface
// This is a stub implementation that mirrors Pascal TWXInterpreter functionality
type ScriptInterpreter struct {
	eventBus    IEventBus
	scripts     map[string]interface{} // In a full implementation, this would hold script objects
	triggersEnabled bool
}

// NewScriptInterpreter creates a new script interpreter instance
func NewScriptInterpreter(eventBus IEventBus) *ScriptInterpreter {
	return &ScriptInterpreter{
		eventBus:        eventBus,
		scripts:         make(map[string]interface{}),
		triggersEnabled: true,
	}
}

// TextEvent fires a text event to the script system (Pascal: TWXInterpreter.TextEvent)
func (si *ScriptInterpreter) TextEvent(line string, outbound bool) {
	
	if si.eventBus != nil {
		event := Event{
			Type:      EventText,
			Data:      map[string]interface{}{
				"line":     line,
				"outbound": outbound,
			},
			Source:    "ScriptInterpreter",
			Timestamp: 0, // Will be set by event bus
		}
		si.eventBus.Fire(event)
	}
}

// TextLineEvent fires a text line event to the script system (Pascal: TWXInterpreter.TextLineEvent)
func (si *ScriptInterpreter) TextLineEvent(line string, outbound bool) {
	
	if si.eventBus != nil {
		event := Event{
			Type:      EventTextLine,
			Data:      map[string]interface{}{
				"line":     line,
				"outbound": outbound,
			},
			Source:    "ScriptInterpreter",
			Timestamp: 0, // Will be set by event bus
		}
		si.eventBus.Fire(event)
	}
}

// ActivateTriggers activates script triggers (Pascal: TWXInterpreter.ActivateTriggers)
func (si *ScriptInterpreter) ActivateTriggers() {
	if !si.triggersEnabled {
		return
	}
	
	
	if si.eventBus != nil {
		event := Event{
			Type:      EventTrigger,
			Data:      map[string]interface{}{
				"action": "activate",
			},
			Source:    "ScriptInterpreter",
			Timestamp: 0, // Will be set by event bus
		}
		si.eventBus.Fire(event)
	}
}

// AutoTextEvent fires an auto text event to the script system (Pascal: TWXInterpreter.AutoTextEvent)
func (si *ScriptInterpreter) AutoTextEvent(line string, outbound bool) {
	
	if si.eventBus != nil {
		event := Event{
			Type:      EventAutoText,
			Data:      map[string]interface{}{
				"line":     line,
				"outbound": outbound,
			},
			Source:    "ScriptInterpreter",
			Timestamp: 0, // Will be set by event bus
		}
		si.eventBus.Fire(event)
	}
}

// LoadScript loads a script file (stub implementation)
func (si *ScriptInterpreter) LoadScript(filename string) error {
	
	// In a full implementation, this would load and compile the script
	si.scripts[filename] = nil // Placeholder
	return nil
}

// UnloadScript unloads a script (stub implementation)
func (si *ScriptInterpreter) UnloadScript(name string) error {
	
	if _, exists := si.scripts[name]; !exists {
		return fmt.Errorf("script '%s' not found", name)
	}
	
	delete(si.scripts, name)
	return nil
}

// ExecuteScript executes a loaded script (stub implementation)
func (si *ScriptInterpreter) ExecuteScript(name string, params map[string]interface{}) error {
	
	if _, exists := si.scripts[name]; !exists {
		return fmt.Errorf("script '%s' not found", name)
	}
	
	// In a full implementation, this would execute the script with the given parameters
	return nil
}

// SetTriggersEnabled enables or disables script triggers
func (si *ScriptInterpreter) SetTriggersEnabled(enabled bool) {
	si.triggersEnabled = enabled
}

// IsTriggersEnabled returns whether triggers are enabled
func (si *ScriptInterpreter) IsTriggersEnabled() bool {
	return si.triggersEnabled
}

// GetLoadedScripts returns a list of loaded script names
func (si *ScriptInterpreter) GetLoadedScripts() []string {
	var scripts []string
	for name := range si.scripts {
		scripts = append(scripts, name)
	}
	return scripts
}