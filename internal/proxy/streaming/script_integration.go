package streaming

import (
)

// ScriptEngine interface for script integration (mirrors Pascal TWXInterpreter)
type ScriptEngine interface {
	// TextEvent processes text events (mirrors Pascal TWXInterpreter.TextEvent)
	ProcessText(text string) error
	
	// TextLineEvent processes text line events (mirrors Pascal TWXInterpreter.TextLineEvent)
	ProcessTextLine(line string) error
	
	// ActivateTriggers activates script triggers (mirrors Pascal TWXInterpreter.ActivateTriggers)
	ActivateTriggers() error
	
	// AutoTextEvent processes auto text events (mirrors Pascal TWXInterpreter.AutoTextEvent)
	ProcessAutoText(text string) error
}

// ScriptEventProcessor implements script event firing functionality
type ScriptEventProcessor struct {
	scriptEngine ScriptEngine
	enabled      bool
}

// NewScriptEventProcessor creates a new script event processor
func NewScriptEventProcessor(scriptEngine ScriptEngine) *ScriptEventProcessor {
	return &ScriptEventProcessor{
		scriptEngine: scriptEngine,
		enabled:      scriptEngine != nil,
	}
}

// SetScriptEngine sets or updates the script engine
func (sep *ScriptEventProcessor) SetScriptEngine(scriptEngine ScriptEngine) {
	sep.scriptEngine = scriptEngine
	sep.enabled = scriptEngine != nil
}

// IsEnabled returns true if script integration is enabled
func (sep *ScriptEventProcessor) IsEnabled() bool {
	return sep.enabled && sep.scriptEngine != nil
}

// FireTextEvent fires a text event (mirrors Pascal TWXInterpreter.TextEvent)
func (sep *ScriptEventProcessor) FireTextEvent(text string, blockExtended bool) error {
	if !sep.IsEnabled() {
		return nil
	}
	
	
	// In Pascal TWX, blockExtended parameter controls whether extended characters are processed
	// For now, we'll process all text events
	if err := sep.scriptEngine.ProcessText(text); err != nil {
		return err
	}
	
	return nil
}

// FireTextLineEvent fires a text line event (mirrors Pascal TWXInterpreter.TextLineEvent)
func (sep *ScriptEventProcessor) FireTextLineEvent(line string, blockExtended bool) error {
	if !sep.IsEnabled() {
		return nil
	}
	
	
	if err := sep.scriptEngine.ProcessTextLine(line); err != nil {
		return err
	}
	
	return nil
}

// FireActivateTriggers activates script triggers (mirrors Pascal TWXInterpreter.ActivateTriggers)
func (sep *ScriptEventProcessor) FireActivateTriggers() error {
	if !sep.IsEnabled() {
		return nil
	}
	
	
	if err := sep.scriptEngine.ActivateTriggers(); err != nil {
		return err
	}
	
	return nil
}

// FireAutoTextEvent fires an auto text event (mirrors Pascal TWXInterpreter.AutoTextEvent)
func (sep *ScriptEventProcessor) FireAutoTextEvent(text string, blockExtended bool) error {
	if !sep.IsEnabled() {
		return nil
	}
	
	
	if err := sep.scriptEngine.ProcessAutoText(text); err != nil {
		return err
	}
	
	return nil
}

// ProcessLineWithScriptEvents processes a complete line with all appropriate script events
// This mirrors the Pascal TWX logic where multiple events are fired for each line
func (sep *ScriptEventProcessor) ProcessLineWithScriptEvents(line string) error {
	if !sep.IsEnabled() {
		return nil
	}
	
	// For complete lines, fire TextLineEvent, TextEvent (in processPrompt), and ActivateTriggers
	// Pascal TWX: ProcessLine -> TextLineEvent -> ProcessPrompt -> TextEvent -> ActivateTriggers
	// AutoTextEvent is NOT fired for complete lines, only for prompts
	
	// Fire TextLineEvent (mirrors Pascal ProcessLine)
	if err := sep.FireTextLineEvent(line, false); err != nil {
		return err
	}
	
	// Fire TextEvent (mirrors Pascal ProcessPrompt)
	if err := sep.FireTextEvent(line, false); err != nil {
		return err
	}
	
	// Activate triggers (mirrors Pascal ProcessLine end)
	if err := sep.FireActivateTriggers(); err != nil {
		return err
	}
	
	return nil
}