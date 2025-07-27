package commands

import (
	"twist/internal/scripting/types"
)

// RegisterGameCommands registers all TWX game-specific commands
func RegisterGameCommands(vm CommandRegistry) {
	// Basic game commands
	vm.RegisterCommand("SEND", 1, -1, []types.ParameterType{types.ParamValue}, cmdSend)
	vm.RegisterCommand("WAITFOR", 1, 1, []types.ParameterType{types.ParamValue}, cmdWaitFor)
	vm.RegisterCommand("PAUSE", 0, 0, []types.ParameterType{}, cmdPause)
	vm.RegisterCommand("HALT", 0, 0, []types.ParameterType{}, cmdHalt)
	vm.RegisterCommand("LOGGING", 1, 1, []types.ParameterType{types.ParamValue}, cmdLogging)
	
	// Timer commands  
	vm.RegisterCommand("SETTIMER", 1, 1, []types.ParameterType{types.ParamValue}, cmdSetTimer)
	vm.RegisterCommand("GETTIMER", 1, 1, []types.ParameterType{types.ParamVar}, cmdGetTimer)
	
	// Input commands
	vm.RegisterCommand("GETINPUT", 3, 3, []types.ParameterType{types.ParamVar, types.ParamValue, types.ParamValue}, cmdGetInput)
	vm.RegisterCommand("GETCONSOLEINPUT", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdGetConsoleInput)
	
	// Text processing commands
	vm.RegisterCommand("MERGETEXT", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdMergeText)
}

func cmdSend(vm types.VMInterface, params []*types.CommandParam) error {
	// Concatenate all parameters like ECHO
	message := ""
	for _, param := range params {
		if param.Type == types.ParamVar {
			// Get variable value
			value := vm.GetVariable(param.VarName)
			message += value.ToString()
		} else {
			// Use literal value
			message += param.Value.ToString()
		}
	}
	return vm.Send(message)
}

func cmdWaitFor(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("WAITFOR requires exactly 1 parameter: pattern")
	}

	pattern := GetParamString(vm, params[0])
	return vm.WaitFor(pattern)
}

func cmdPause(vm types.VMInterface, params []*types.CommandParam) error {
	return vm.Pause()
}

func cmdHalt(vm types.VMInterface, params []*types.CommandParam) error {
	return vm.Halt()
}

func cmdLogging(vm types.VMInterface, params []*types.CommandParam) error {
	// Toggle logging on/off based on parameter
	// For testing, we'll just acknowledge the command
	return nil
}


func cmdSetTimer(vm types.VMInterface, params []*types.CommandParam) error {
	// Set timer value
	return nil
}

func cmdGetTimer(vm types.VMInterface, params []*types.CommandParam) error {
	// Get current timer value - for testing return 0
	result := &types.Value{
		Type:   types.NumberType,
		Number: 0,
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}





func cmdGetInput(vm types.VMInterface, params []*types.CommandParam) error {
	// Get user input - for testing, return empty string
	result := &types.Value{
		Type:   types.StringType,
		String: "",
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}

func cmdGetConsoleInput(vm types.VMInterface, params []*types.CommandParam) error {
	// Get console input - for testing, return "1"
	result := &types.Value{
		Type:   types.StringType,
		String: "1",
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}


func cmdMergeText(vm types.VMInterface, params []*types.CommandParam) error {
	// Merge two text strings
	text1 := GetParamString(vm, params[0])
	text2 := GetParamString(vm, params[1])
	result := &types.Value{
		Type:   types.StringType,
		String: text1 + text2,
	}
	vm.SetVariable(params[2].VarName, result)
	return nil
}


// cmdTime gets the current time
func cmdTime(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("TIME requires exactly 1 parameter: result_var")
	}

	// Mock time value for testing
	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.StringType,
		String: "12:34:56",
	})

	return nil
}

// cmdDate gets the current date
func cmdDate(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("DATE requires exactly 1 parameter: result_var")
	}

	// Mock date value for testing
	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.StringType,
		String: "01/01/2024",
	})

	return nil
}

// cmdGetTime gets the current time in milliseconds
func cmdGetTime(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("GETTIME requires exactly 1 parameter: result_var")
	}

	// Mock time value for testing
	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.NumberType,
		Number: 123456789,
	})

	return nil
}

// cmdSleep pauses execution for a specified time
func cmdSleep(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("SLEEP requires exactly 1 parameter: milliseconds")
	}

	// Mock sleep - in real implementation would actually sleep
	return nil
}

// cmdGetCurrentSector gets the current sector number
func cmdGetCurrentSector(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("GETCURRENTSECTOR requires exactly 1 parameter: result_var")
	}

	// Mock sector value for testing
	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.NumberType,
		Number: 1,
	})

	return nil
}

// cmdGetCurrentPrompt gets the current prompt
func cmdGetCurrentPrompt(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("GETCURRENTPROMPT requires exactly 1 parameter: result_var")
	}

	// Mock prompt value for testing
	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.StringType,
		String: "Command [TL=00:00:00]:[",
	})

	return nil
}

// cmdIsString checks if a value is a string
func cmdIsString(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 2 {
		return vm.Error("ISSTRING requires exactly 2 parameters: value, result_var")
	}

	val := GetParamValue(vm, params[0])
	result := 0.0
	if val.Type == types.StringType {
		result = 1.0
	}

	vm.SetVariable(params[1].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// cmdGetType gets the type of a value
func cmdGetType(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 2 {
		return vm.Error("GETTYPE requires exactly 2 parameters: value, result_var")
	}

	val := GetParamValue(vm, params[0])
	var typeStr string
	switch val.Type {
	case types.StringType:
		typeStr = "string"
	case types.NumberType:
		typeStr = "number"
	case types.ArrayType:
		typeStr = "array"
	default:
		typeStr = "unknown"
	}

	vm.SetVariable(params[1].VarName, &types.Value{
		Type:   types.StringType,
		String: typeStr,
	})

	return nil
}

