package commands

import (
	"fmt"
	"strconv"
	"time"
	"twist/internal/proxy/scripting/types"
)

// RegisterTriggerCommands registers trigger commands with the VM
func RegisterTriggerCommands(vm CommandRegistry) {
	// Enhanced trigger commands for Pascal TWX compatibility
	vm.RegisterCommand("SETTEXTLINETRIGGER", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue}, cmdSetTextLineTrigger)
	vm.RegisterCommand("SETTEXTTRIGGER", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue}, cmdSetTextTrigger)
	vm.RegisterCommand("KILLTRIGGER", 1, 1, []types.ParameterType{types.ParamValue}, cmdKillTrigger)
	vm.RegisterCommand("KILLALLTRIGGERS", 0, 0, []types.ParameterType{}, cmdKillAllTriggers)

	// Legacy trigger commands (for backwards compatibility)
	vm.RegisterCommand("SETTRIGGER", 2, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue}, cmdSetTrigger)
	vm.RegisterCommand("SETTEXTOUTTRIGGER", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue}, cmdSetTextOutTrigger)
	vm.RegisterCommand("SETDELAYTRIGGER", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue}, cmdSetDelayTrigger)
	vm.RegisterCommand("SETEVENTTRIGGER", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue}, cmdSetEventTrigger)
}

// cmdSetTextLineTrigger implements the setTextLineTrigger command
// Syntax: setTextLineTrigger <id> <label> <pattern>
// Example: setTextLineTrigger 1 :getWarp "Sector "
func cmdSetTextLineTrigger(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("SETTEXTLINETRIGGER requires exactly 3 parameters: id, label, pattern")
	}

	id := GetParamString(vm, params[0])
	label := GetParamString(vm, params[1])
	pattern := GetParamString(vm, params[2])

	if id == "" {
		return vm.Error("SETTEXTLINETRIGGER requires a non-empty trigger ID")
	}
	if label == "" {
		return vm.Error("SETTEXTLINETRIGGER requires a non-empty label")
	}
	if pattern == "" {
		return vm.Error("SETTEXTLINETRIGGER requires a non-empty pattern")
	}

	// Create TextLineTrigger with Pascal-compatible behavior
	trigger := &types.TextLineTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerTextLine,
			Label:     label,
			Value:     pattern,
			Active:    true,
			LifeCycle: -1, // Permanent by default, matches Pascal behavior
		},
	}

	// Use the existing VM trigger interface
	return vm.SetTrigger(trigger)
}

// cmdSetTextTrigger implements the setTextTrigger command
// Syntax: setTextTrigger <id> <label> <pattern>
// Example: setTextTrigger 2 :gotWarps "Command [TL="
func cmdSetTextTrigger(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("SETTEXTTRIGGER requires exactly 3 parameters: id, label, pattern")
	}

	id := GetParamString(vm, params[0])
	label := GetParamString(vm, params[1])
	pattern := GetParamString(vm, params[2])

	// Empty trigger IDs are valid in TWX Pascal implementation
	// Empty trigger labels are valid in TWX Pascal implementation (will cause runtime errors if executed)
	// Empty patterns are valid in TWX - they match any text

	// Create TextTrigger with Pascal-compatible behavior
	trigger := &types.TextTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerText,
			Label:     label,
			Value:     pattern,
			Active:    true,
			LifeCycle: -1, // Permanent by default, matches Pascal behavior
		},
	}

	// Use the existing VM trigger interface
	return vm.SetTrigger(trigger)
}

// cmdKillTrigger implements the killTrigger command
// Syntax: killTrigger <id>
// Example: killTrigger 1
func cmdKillTrigger(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("KILLTRIGGER requires exactly 1 parameter: trigger_id")
	}

	id := GetParamString(vm, params[0])
	if id == "" {
		return vm.Error("KILLTRIGGER requires a non-empty trigger ID")
	}

	// Use the existing VM trigger interface
	return vm.KillTrigger(id)
}

// cmdKillAllTriggers implements the killAllTriggers command
// Syntax: killAllTriggers
func cmdKillAllTriggers(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 0 {
		return vm.Error("KILLALLTRIGGERS requires no parameters")
	}

	// Use the existing VM trigger interface
	vm.KillAllTriggers()
	return nil
}

// Legacy trigger commands for backwards compatibility

// cmdSetTrigger implements the legacy setTrigger command
func cmdSetTrigger(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) < 2 || len(params) > 3 {
		return vm.Error("SETTRIGGER requires 2-3 parameters: pattern, response [, label]")
	}

	pattern := GetParamString(vm, params[0])
	response := GetParamString(vm, params[1])
	label := ""
	if len(params) == 3 {
		label = GetParamString(vm, params[2])
	}

	// Create a basic text trigger with auto-generated ID
	trigger := &types.TextTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        fmt.Sprintf("auto_%d", time.Now().UnixNano()),
			Type:      types.TriggerText,
			Label:     label,
			Value:     pattern,
			Response:  response,
			Active:    true,
			LifeCycle: -1,
		},
	}

	return vm.SetTrigger(trigger)
}

// cmdSetTextOutTrigger implements the setTextOutTrigger command
func cmdSetTextOutTrigger(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("SETTEXTOUTTRIGGER requires exactly 3 parameters: id, label, pattern")
	}

	id := GetParamString(vm, params[0])
	label := GetParamString(vm, params[1])
	pattern := GetParamString(vm, params[2])

	trigger := &types.TextOutTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerTextOut,
			Label:     label,
			Value:     pattern,
			Active:    true,
			LifeCycle: -1,
		},
	}

	return vm.SetTrigger(trigger)
}

// cmdSetDelayTrigger implements the setDelayTrigger command
func cmdSetDelayTrigger(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("SETDELAYTRIGGER requires exactly 3 parameters: id, label, delay_ms")
	}

	id := GetParamString(vm, params[0])
	label := GetParamString(vm, params[1])
	delayStr := GetParamString(vm, params[2])

	delay, err := strconv.ParseFloat(delayStr, 64)
	if err != nil {
		return vm.Error(fmt.Sprintf("Invalid delay value: %s", delayStr))
	}

	trigger := &types.DelayTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        id,
			Type:      types.TriggerDelay,
			Label:     label,
			Value:     delayStr,
			Active:    true,
			LifeCycle: 1, // One-shot by default
		},
		Delay:     time.Duration(delay) * time.Millisecond,
		StartTime: time.Now(),
	}

	// Set up timer
	trigger.Timer = time.AfterFunc(trigger.Delay, func() {
		// Timer expired, trigger can now match
	})

	return vm.SetTrigger(trigger)
}

// cmdSetEventTrigger implements the setEventTrigger command
func cmdSetEventTrigger(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("SETEVENTTRIGGER requires exactly 3 parameters: event_name, response, label")
	}

	eventName := GetParamString(vm, params[0])
	response := GetParamString(vm, params[1])
	label := GetParamString(vm, params[2])

	trigger := &types.EventTrigger{
		BaseTrigger: types.BaseTrigger{
			ID:        fmt.Sprintf("event_%s_%d", eventName, time.Now().UnixNano()),
			Type:      types.TriggerEvent,
			Label:     label,
			Value:     eventName,
			Response:  response,
			Active:    true,
			LifeCycle: -1,
		},
		EventName: eventName,
	}

	return vm.SetTrigger(trigger)
}
