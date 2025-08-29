package types

import (
	"strings"
	"time"
	"twist/internal/debug"
)

// TriggerType represents the type of trigger
type TriggerType int

const (
	TriggerText TriggerType = iota
	TriggerTextLine
	TriggerTextOut
	TriggerDelay
	TriggerEvent
	TriggerAuto
	TriggerAutoText
)

// TriggerInterface defines the interface for all triggers
type TriggerInterface interface {
	GetID() string
	GetType() TriggerType
	GetLabel() string
	GetValue() string
	GetResponse() string
	GetParam() string
	GetLifeCycle() int
	IsActive() bool
	SetActive(active bool)
	Matches(input string) bool
	Execute(vm VMInterface) error
}

// BaseTrigger provides common trigger functionality
type BaseTrigger struct {
	ID        string
	Type      TriggerType
	Label     string
	Value     string
	Response  string
	Param     string
	LifeCycle int
	Active    bool
	ScriptID  string
}

// GetID returns the trigger ID
func (t *BaseTrigger) GetID() string {
	return t.ID
}

// GetType returns the trigger type
func (t *BaseTrigger) GetType() TriggerType {
	return t.Type
}

// GetLabel returns the trigger label
func (t *BaseTrigger) GetLabel() string {
	return t.Label
}

// GetValue returns the trigger value/pattern
func (t *BaseTrigger) GetValue() string {
	return t.Value
}

// GetResponse returns the trigger response
func (t *BaseTrigger) GetResponse() string {
	return t.Response
}

// GetParam returns the trigger parameter
func (t *BaseTrigger) GetParam() string {
	return t.Param
}

// GetLifeCycle returns the trigger lifecycle count
func (t *BaseTrigger) GetLifeCycle() int {
	return t.LifeCycle
}

// IsActive returns whether the trigger is active
func (t *BaseTrigger) IsActive() bool {
	return t.Active
}

// SetActive sets the trigger active state
func (t *BaseTrigger) SetActive(active bool) {
	t.Active = active
}

// TextTrigger handles text pattern matching
type TextTrigger struct {
	BaseTrigger
}

// Matches checks if the input matches the trigger pattern
func (t *TextTrigger) Matches(input string) bool {
	if !t.Active {
		return false
	}
	// TextTrigger matches if the pattern is contained anywhere in the input
	// This matches Pascal TWX behavior for text triggers
	return input != "" && t.Value != "" &&
		(input == t.Value || strings.Contains(input, t.Value))
}

// Execute executes the trigger action
func (t *TextTrigger) Execute(vm VMInterface) error {
	scriptName := "unknown"
	currentLine := vm.GetCurrentLine()
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	
	// DEBUG: Log the CURRENTLINE at the exact moment the trigger fires
	currentLineVar := vm.GetVariable("CURRENTLINE")
	if currentLineVar != nil {
		debug.Info("TEXT TRIGGER FIRED", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "pattern", t.Value, "CURRENTLINE", currentLineVar.String)
	}
	
	if t.Response != "" {
		if err := vm.Send(t.Response); err != nil {
			return err
		}
	}

	if t.Label != "" {
		debug.Info("TEXT TRIGGER: jumping to label", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "jumpTarget", t.Label)
		// In TWX, setTextTrigger ALWAYS jumps to labels, never executes inline code
		// This is TWX-compatible behavior - label parameter is always a label name
		return vm.GotoAndExecuteSync(t.Label)
	}

	return nil
}

// TextLineTrigger handles line-based text matching
type TextLineTrigger struct {
	BaseTrigger
}

// Matches checks if the input line matches the trigger pattern
func (t *TextLineTrigger) Matches(input string) bool {
	if !t.Active {
		return false
	}
	// TextLineTrigger matches if the line starts with the pattern
	// This matches Pascal TWX behavior for text line triggers
	return input != "" && t.Value != "" &&
		(input == t.Value || strings.HasPrefix(input, t.Value))
}

// Execute executes the trigger action
func (t *TextLineTrigger) Execute(vm VMInterface) error {
	scriptName := "unknown"
	currentLine := vm.GetCurrentLine()
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	
	debug.Info("TEXTLINE TRIGGER FIRED", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "pattern", t.Value, "label", t.Label, "response", t.Response)
	
	if t.Response != "" {
		if err := vm.Send(t.Response); err != nil {
			return err
		}
	}

	if t.Label != "" {
		// Same logic as TextTrigger - handle commands vs labels
		if strings.HasPrefix(t.Label, "echo ") ||
			strings.HasPrefix(t.Label, "send ") ||
			strings.HasPrefix(t.Label, "setvar ") ||
			strings.Contains(t.Label, " ") { // If it contains spaces, likely a command
			if strings.HasPrefix(t.Label, "echo ") {
				message := strings.TrimPrefix(t.Label, "echo ")
				message = strings.Trim(message, "'\"") // Remove quotes
				return vm.Echo(message)
			}
		}
		
		debug.Info("TEXTLINE TRIGGER: jumping to label", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "jumpTarget", t.Label)
		// In TWX, triggers cause permanent jumps in execution flow (not temporary detours)
		// This is different from GOSUB which uses a stack for returns
		return vm.GotoAndExecuteSync(t.Label)
	}

	return nil
}

// TextOutTrigger handles outgoing text matching
type TextOutTrigger struct {
	BaseTrigger
}

// Matches checks if the outgoing text matches the trigger pattern
func (t *TextOutTrigger) Matches(input string) bool {
	if !t.Active {
		return false
	}
	return input == t.Value || (t.Value != "" && len(input) >= len(t.Value) &&
		input[len(input)-len(t.Value):] == t.Value)
}

// Execute executes the trigger action
func (t *TextOutTrigger) Execute(vm VMInterface) error {
	scriptName := "unknown"
	currentLine := vm.GetCurrentLine()
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	
	debug.Info("TEXTOUT TRIGGER FIRED", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "pattern", t.Value)
	
	if t.Label != "" {
		debug.Info("TEXTOUT TRIGGER: jumping to label", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "jumpTarget", t.Label)
		// In TWX, triggers cause permanent jumps in execution flow (not temporary detours)
		// This is different from GOSUB which uses a stack for returns
		return vm.GotoAndExecuteSync(t.Label)
	}
	return nil
}

// DelayTrigger handles time-based triggers
type DelayTrigger struct {
	BaseTrigger
	Delay     time.Duration
	StartTime time.Time
	Timer     *time.Timer
}

// Matches checks if the delay has elapsed
func (t *DelayTrigger) Matches(input string) bool {
	return t.Active && t.Timer != nil && time.Since(t.StartTime) >= t.Delay
}

// Execute executes the trigger action
func (t *DelayTrigger) Execute(vm VMInterface) error {
	scriptName := "unknown"
	currentLine := vm.GetCurrentLine()
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	
	debug.Info("DELAY TRIGGER FIRED", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "delay", t.Delay)
	
	if t.Label != "" {
		debug.Info("DELAY TRIGGER: jumping to label", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "jumpTarget", t.Label)
		// In TWX, triggers cause permanent jumps in execution flow (not temporary detours)
		// This is different from GOSUB which uses a stack for returns
		return vm.GotoAndExecuteSync(t.Label)
	}
	return nil
}

// EventTrigger handles system events
type EventTrigger struct {
	BaseTrigger
	EventName string
}

// Matches checks if the event matches
func (t *EventTrigger) Matches(input string) bool {
	return t.Active && input == t.EventName
}

// Execute executes the trigger action
func (t *EventTrigger) Execute(vm VMInterface) error {
	scriptName := "unknown"
	currentLine := vm.GetCurrentLine()
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	
	debug.Info("EVENT TRIGGER FIRED", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "eventName", t.EventName)
	
	if t.Response != "" {
		if err := vm.Send(t.Response); err != nil {
			return err
		}
	}

	if t.Label != "" {
		debug.Info("EVENT TRIGGER: jumping to label", "script", scriptName, "currentLine", currentLine, "triggerId", t.ID, "jumpTarget", t.Label)
		// In TWX, triggers cause permanent jumps in execution flow (not temporary detours)
		// This is different from GOSUB which uses a stack for returns
		return vm.GotoAndExecuteSync(t.Label)
	}

	return nil
}

// AutoTrigger handles automatic response triggers
type AutoTrigger struct {
	BaseTrigger
	Persistent bool
}

// Matches checks if the input matches the trigger pattern
func (t *AutoTrigger) Matches(input string) bool {
	if !t.Active {
		return false
	}
	return input == t.Value || (t.Value != "" && len(input) >= len(t.Value) &&
		input[len(input)-len(t.Value):] == t.Value)
}

// Execute executes the trigger action
func (t *AutoTrigger) Execute(vm VMInterface) error {
	if t.Response != "" {
		if err := vm.Send(t.Response); err != nil {
			return err
		}
	}

	// Auto triggers don't jump to labels, they just respond
	return nil
}

// AutoTextTrigger handles automatic text response triggers
type AutoTextTrigger struct {
	BaseTrigger
	Persistent bool
}

// Matches checks if the input matches the trigger pattern
func (t *AutoTextTrigger) Matches(input string) bool {
	if !t.Active {
		return false
	}
	return input == t.Value || (t.Value != "" && len(input) >= len(t.Value) &&
		input[len(input)-len(t.Value):] == t.Value)
}

// Execute executes the trigger action
func (t *AutoTextTrigger) Execute(vm VMInterface) error {
	if t.Response != "" {
		if err := vm.Echo(t.Response); err != nil {
			return err
		}
	}

	return nil
}

// TriggerManagerInterface defines the interface for managing triggers (matches TWX per-script design)
type TriggerManagerInterface interface {
	// Trigger management
	AddTrigger(trigger TriggerInterface) error
	RemoveTrigger(id string) error
	RemoveAllTriggers() error
	GetTrigger(id string) TriggerInterface
	
	// Trigger processing
	ProcessText(text string) error
	ProcessTextLine(line string) (bool, error)
	ProcessTextOut(text string) error
	ProcessDelayTriggers() error
	
	// Trigger queries
	HasTriggers() bool
	GetTriggerCount() int
}
