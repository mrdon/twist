package commands

import (
	"fmt"
	"twist/internal/debug"
	"twist/internal/proxy/scripting/types"
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
	vm.RegisterCommand("GETINPUT", 2, 3, []types.ParameterType{types.ParamVar, types.ParamValue, types.ParamValue}, cmdGetInput)
	vm.RegisterCommand("GETCONSOLEINPUT", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdGetConsoleInput)

	// Debug command for troubleshooting
	vm.RegisterCommand("DEBUGLOG", 1, -1, []types.ParameterType{types.ParamValue}, cmdDebugLog)

	// Text processing commands
	vm.RegisterCommand("MERGETEXT", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdMergeText)

	// Game data commands - TWX compatibility
	vm.RegisterCommand("GETSECTOR", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdGetSector)
}

func cmdSend(vm types.VMInterface, params []*types.CommandParam) error {
	// Concatenate all parameters like ECHO
	message := ""
	for _, param := range params {
		if param.Type == types.ParamVar {
			// Get variable value
			value := vm.GetVariable(param.VarName)
			debug.Log("SEND command (line %d): variable %s resolves to %q", vm.GetCurrentLine(), param.VarName, value.ToString())
			message += value.ToString()
		} else {
			// Use literal value
			message += param.Value.ToString()
		}
	}

	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	debug.Log("SEND command [%s] (line %d): sending message %q", scriptName, vm.GetCurrentLine(), message)

	// Send message as-is (carriage returns from lexer are preserved)
	return vm.Send(message)
}

func cmdWaitFor(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("WAITFOR requires exactly 1 parameter: pattern")
	}

	pattern := GetParamString(vm, params[0])
	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	debug.Log("WAITFOR command [%s] (line %d): waiting for pattern %q", scriptName, vm.GetCurrentLine(), pattern)
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
	// Extract prompt text (2nd parameter)
	prompt := ""
	if len(params) > 1 {
		prompt = GetParamString(vm, params[1])
	}

	// Extract default value (3rd parameter, optional)
	defaultValue := ""
	if len(params) > 2 {
		defaultValue = GetParamString(vm, params[2])
	}

	// Debug logging
	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	debug.Log("GETINPUT [%s] (line %d): prompt=%q, default=%q", scriptName, vm.GetCurrentLine(), prompt, defaultValue)
	debug.Log("GETINPUT [%s]: pendingResult=%q, waitingForInput=%v, pendingPrompt=%q",
		scriptName, vm.GetPendingInputResult(), vm.IsWaitingForInput(), vm.GetPendingInputPrompt())

	// Check if there's a pending input result (we're resuming from input)
	// We use JustResumedFromInput to handle cases where input is empty string
	if vm.IsWaitingForInput() || vm.JustResumedFromInput() {
		// We're resuming - get the input result and store it
		debug.Log("GETINPUT [%s]: RESUMING from input, processing stored result", scriptName)
		input := vm.GetPendingInputResult()

		// Use default value if input is empty
		if input == "" && defaultValue != "" {
			input = defaultValue
		}

		debug.Log("GETINPUT [%s]: setting variable %s = %q", scriptName, params[0].VarName, input)

		// Store result in the variable
		result := &types.Value{
			Type:   types.StringType,
			String: input,
		}
		vm.SetVariable(params[0].VarName, result)

		// Clear the pending input state since we've processed it
		vm.ClearPendingInput()

		return nil // This will allow the execution to advance to the next command
	}

	// First time executing this command - initiate input collection
	debug.Log("GETINPUT [%s]: FIRST TIME - initiating input collection", scriptName)
	// Format the prompt like TWX does
	fullPrompt := prompt
	if defaultValue != "" {
		fullPrompt = prompt + " [" + defaultValue + "]"
	}

	// Initiate input collection and pause script execution
	// GetInput will handle displaying the prompt
	_, err := vm.GetInput(fullPrompt)
	if err != nil {
		return err
	}

	// Return a pause error to stop execution until input is provided
	return types.ErrScriptPaused
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

func cmdDebugLog(vm types.VMInterface, params []*types.CommandParam) error {
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

	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	debug.Log("SCRIPT DEBUG [%s] (line %d): %s", scriptName, vm.GetCurrentLine(), message)
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

// cmdGetSector implements the TWX getSector command with full Pascal compatibility
// Syntax: getSector <index> <var>
// Example: getSector 123 $s
func cmdGetSector(vm types.VMInterface, params []*types.CommandParam) error {
	// Add panic recovery for debugging
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdGetSector: %v", r)
			panic(r) // Re-panic after logging
		}
	}()
	
	if len(params) != 2 {
		return vm.Error("GETSECTOR requires exactly 2 parameters: sector_index, result_var")
	}

	// Get sector index from first parameter
	indexValue := GetParamValue(vm, params[0])
	debug.Log("cmdGetSector: indexValue=%v", indexValue)
	sectorIndex := int(indexValue.ToNumber())
	debug.Log("cmdGetSector: sectorIndex=%d", sectorIndex)

	// Ignore invalid call with index of zero (Pascal TWX behavior)
	if sectorIndex == 0 {
		return nil
	}

	// Get variable name for result
	varName := params[1].VarName
	debug.Log("cmdGetSector: varName=%s", varName)

	// Get sector data from game interface
	gameInterface := vm.GetGameInterface()
	debug.Log("cmdGetSector: gameInterface=%v", gameInterface)
	if gameInterface == nil {
		debug.Log("cmdGetSector: ERROR - gameInterface is nil!")
		return vm.Error("Game interface not available")
	}
	sector, err := gameInterface.GetSector(sectorIndex)
	debug.Log("cmdGetSector: after GetSector call, err=%v", err)
	if err != nil {
		debug.Log("GETSECTOR: sector %d not found in database, setting default values. Error: %v", sectorIndex, err)
		// If sector not found, set default empty values
		setSectorVariables(vm, varName, sectorIndex, nil)
		return nil
	}

	debug.Log("GETSECTOR: sector %d found - PortName=%q, PortClass=%d, HasPort=%v", sectorIndex, sector.PortName, sector.PortClass, sector.HasPort)

	// Set all sector variables matching Pascal TWX exactly
	setSectorVariables(vm, varName, sectorIndex, &sector)
	return nil
}

// setSectorVariables sets all sector variables exactly like Pascal TWX CmdGetSector
func setSectorVariables(vm types.VMInterface, varName string, index int, sector *types.SectorData) {
	// Always set the index
	vm.SetVariable(varName+".INDEX", &types.Value{
		Type: types.NumberType, Number: float64(index),
	})

	if sector == nil {
		// Set default values for non-existent sector
		setDefaultSectorValues(vm, varName)
		return
	}

	// Set exploration status
	switch sector.Explored {
	case 0: // etNo
		vm.SetVariable(varName+".EXPLORED", &types.Value{Type: types.StringType, String: "NO"})
	case 1: // etCalc
		vm.SetVariable(varName+".EXPLORED", &types.Value{Type: types.StringType, String: "CALC"})
	case 2: // etDensity
		vm.SetVariable(varName+".EXPLORED", &types.Value{Type: types.StringType, String: "DENSITY"})
	case 3: // etHolo
		vm.SetVariable(varName+".EXPLORED", &types.Value{Type: types.StringType, String: "YES"})
	default:
		vm.SetVariable(varName+".EXPLORED", &types.Value{Type: types.StringType, String: "NO"})
	}

	// Basic sector properties
	vm.SetVariable(varName+".BEACON", &types.Value{Type: types.StringType, String: sector.Beacon})
	vm.SetVariable(varName+".CONSTELLATION", &types.Value{Type: types.StringType, String: sector.Constellation})

	// Mines - for now set to empty (would need full mine implementation)
	vm.SetVariable(varName+".ARMIDMINES.QUANTITY", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable(varName+".LIMPETMINES.QUANTITY", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable(varName+".ARMIDMINES.OWNER", &types.Value{Type: types.StringType, String: ""})
	vm.SetVariable(varName+".LIMPETMINES.OWNER", &types.Value{Type: types.StringType, String: ""})

	// Fighters - for now set to empty
	vm.SetVariable(varName+".FIGS.QUANTITY", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable(varName+".FIGS.OWNER", &types.Value{Type: types.StringType, String: ""})

	// Warp and density information
	vm.SetVariable(varName+".WARPS", &types.Value{Type: types.NumberType, Number: float64(len(sector.Warps))})
	vm.SetVariable(varName+".DENSITY", &types.Value{Type: types.NumberType, Number: float64(sector.Density)})
	vm.SetVariable(varName+".NAVHAZ", &types.Value{Type: types.NumberType, Number: float64(sector.NavHaz)})

	// Set warp array (1-6 like Pascal TWX)
	for i := 1; i <= 6; i++ {
		warpValue := 0
		if i-1 < len(sector.Warps) {
			warpValue = sector.Warps[i-1]
		}
		vm.SetVariable(varName+".WARP["+fmt.Sprintf("%d", i)+"]", &types.Value{
			Type: types.NumberType, Number: float64(warpValue),
		})
	}

	// Port information (key part for 1_Trade.ts compatibility)
	setPortVariables(vm, varName, sector)

	// Trader, ship, planet counts - for basic compatibility set to 0
	vm.SetVariable(varName+".TRADERS", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable(varName+".SHIPS", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable(varName+".PLANETS", &types.Value{Type: types.NumberType, Number: 0})
}

// setPortVariables sets port variables exactly like Pascal TWX
func setPortVariables(vm types.VMInterface, varName string, sector *types.SectorData) {
	// Always set port name
	portName := ""
	if sector != nil {
		portName = sector.PortName
	}
	vm.SetVariable(varName+".PORT.NAME", &types.Value{Type: types.StringType, String: portName})

	if !sector.HasPort {
		// No port exists
		debug.Log("SETPORTVARS: %s - No port (portName=%q, hasPort=%v) - setting PORT.CLASS=0", varName, portName, sector != nil && sector.HasPort)
		vm.SetVariable(varName+".PORT.CLASS", &types.Value{Type: types.NumberType, Number: 0})
		vm.SetVariable(varName+".PORT.EXISTS", &types.Value{Type: types.NumberType, Number: 0})
	} else {
		// Port exists - set all port variables using actual sector data
		debug.Log("SETPORTVARS: %s - Port exists (portName=%q, portClass=%d) - setting PORT.CLASS=%d", varName, portName, sector.PortClass, sector.PortClass)
		vm.SetVariable(varName+".PORT.CLASS", &types.Value{Type: types.NumberType, Number: float64(sector.PortClass)})
		vm.SetVariable(varName+".PORT.EXISTS", &types.Value{Type: types.NumberType, Number: 1})
		vm.SetVariable(varName+".PORT.BUILDTIME", &types.Value{Type: types.NumberType, Number: 0})

		// Product percentages (placeholder values)
		vm.SetVariable(varName+".PORT.PERC_ORE", &types.Value{Type: types.NumberType, Number: 100})
		vm.SetVariable(varName+".PORT.PERC_ORG", &types.Value{Type: types.NumberType, Number: 100})
		vm.SetVariable(varName+".PORT.PERC_EQUIP", &types.Value{Type: types.NumberType, Number: 100})

		// Product amounts (placeholder values)
		vm.SetVariable(varName+".PORT.ORE", &types.Value{Type: types.NumberType, Number: 0})
		vm.SetVariable(varName+".PORT.ORG", &types.Value{Type: types.NumberType, Number: 0})
		vm.SetVariable(varName+".PORT.EQUIP", &types.Value{Type: types.NumberType, Number: 0})

		// Port update timestamp (placeholder)
		vm.SetVariable(varName+".PORT.UPDATED", &types.Value{Type: types.StringType, String: "01/01/2024 00:00:00"})

		// Buy flags (placeholder - assume port buys everything)
		vm.SetVariable(varName+".PORT.BUY_ORE", &types.Value{Type: types.StringType, String: "YES"})
		vm.SetVariable(varName+".PORT.BUY_ORG", &types.Value{Type: types.StringType, String: "YES"})
		vm.SetVariable(varName+".PORT.BUY_EQUIP", &types.Value{Type: types.StringType, String: "YES"})
	}
}

// setDefaultSectorValues sets default values for non-existent sectors
func setDefaultSectorValues(vm types.VMInterface, varName string) {
	// Set minimal default values
	vm.SetVariable(varName+".EXPLORED", &types.Value{Type: types.StringType, String: "NO"})
	vm.SetVariable(varName+".BEACON", &types.Value{Type: types.StringType, String: ""})
	vm.SetVariable(varName+".CONSTELLATION", &types.Value{Type: types.StringType, String: ""})
	vm.SetVariable(varName+".WARPS", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable(varName+".DENSITY", &types.Value{Type: types.NumberType, Number: -1})
	vm.SetVariable(varName+".NAVHAZ", &types.Value{Type: types.NumberType, Number: 0})

	// Default warp array
	for i := 1; i <= 6; i++ {
		vm.SetVariable(varName+".WARP["+fmt.Sprintf("%d", i)+"]", &types.Value{
			Type: types.NumberType, Number: 0,
		})
	}

	// No port
	vm.SetVariable(varName+".PORT.CLASS", &types.Value{Type: types.NumberType, Number: 0})
	vm.SetVariable(varName+".PORT.EXISTS", &types.Value{Type: types.NumberType, Number: 0})
}
