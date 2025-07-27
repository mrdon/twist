package commands

import (
	"fmt"
	"twist/internal/scripting/manager"
	"twist/internal/scripting/types"
)

// RegisterScriptCommands registers script management commands with the VM
func RegisterScriptCommands(vm CommandRegistry) {
	vm.RegisterCommand("LOAD", 1, 1, []types.ParameterType{types.ParamValue}, cmdLoad)
	vm.RegisterCommand("STOP", 1, 1, []types.ParameterType{types.ParamValue}, cmdStop)
	vm.RegisterCommand("STOPALL", 0, 0, []types.ParameterType{}, cmdStopAll)
	vm.RegisterCommand("SYSTEMSCRIPT", 1, 1, []types.ParameterType{types.ParamValue}, cmdSystemScript)
	vm.RegisterCommand("LISTACTIVESCRIPTS", 1, 1, []types.ParameterType{types.ParamVar}, cmdListActiveScripts)
	vm.RegisterCommand("GETSCRIPTVERSION", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdGetScriptVersion)
	vm.RegisterCommand("REQVERSION", 1, 1, []types.ParameterType{types.ParamValue}, cmdReqVersion)
}

// cmdLoad loads a script
func cmdLoad(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("LOAD requires exactly 1 parameter: script_name")
	}

	scriptName := GetParamString(vm, params[0])
	if scriptName == "" {
		return vm.Error("LOAD requires a non-empty script name")
	}

	// Load the script using the VM's LoadAdditionalScript method
	_, err := vm.LoadAdditionalScript(scriptName)
	if err != nil {
		return vm.Error(fmt.Sprintf("Failed to load script %s: %v", scriptName, err))
	}

	return nil
}

// cmdStop stops a script
func cmdStop(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("STOP requires exactly 1 parameter: script_id")
	}

	scriptID := GetParamString(vm, params[0])
	if scriptID == "" {
		return vm.Error("STOP requires a non-empty script ID")
	}

	// Stop the script using the VM's StopScript method
	err := vm.StopScript(scriptID)
	if err != nil {
		return vm.Error(fmt.Sprintf("Failed to stop script %s: %v", scriptID, err))
	}

	return nil
}

// cmdStopAll stops all scripts
func cmdStopAll(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 0 {
		return vm.Error("STOPALL requires no parameters")
	}

	// Kill all triggers which should stop script execution
	vm.KillAllTriggers()
	
	// In a real implementation, this would stop all running scripts
	// For now, we rely on KillAllTriggers to stop script execution
	return nil
}

// cmdSystemScript executes a system script
func cmdSystemScript(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("SYSTEMSCRIPT requires exactly 1 parameter: script_name")
	}

	systemScriptName := GetParamString(vm, params[0])
	
	// Get the script manager from the VM
	scriptManager := vm.GetScriptManager()
	if scriptManager == nil {
		return vm.Error("Script manager not available")
	}

	sm, ok := scriptManager.(*manager.ScriptManager)
	if !ok {
		return vm.Error("Invalid script manager")
	}

	// Load and execute the system script
	err := sm.LoadSystemScript(systemScriptName)
	if err != nil {
		return vm.Error(fmt.Sprintf("Failed to execute system script %s: %v", systemScriptName, err))
	}
	
	return nil
}

// cmdListActiveScripts lists all active scripts
func cmdListActiveScripts(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("LISTACTIVESCRIPTS requires exactly 1 parameter: result_var")
	}

	// Get the script manager from the VM
	scriptManager := vm.GetScriptManager()
	if scriptManager == nil {
		// No script manager - return empty list
		vm.SetVariable(params[0].VarName, &types.Value{
			Type:   types.StringType,
			String: "",
		})
		return nil
	}

	// Cast to the actual ScriptManager type
	sm, ok := scriptManager.(*manager.ScriptManager)
	if !ok {
		// Type assertion failed - return empty list
		vm.SetVariable(params[0].VarName, &types.Value{
			Type:   types.StringType,
			String: "",
		})
		return nil
	}

	// Get active scripts and format as comma-separated list
	activeScripts := sm.GetActiveScripts()
	var scriptNames []string
	for _, script := range activeScripts {
		scriptNames = append(scriptNames, script.GetName())
	}

	result := ""
	if len(scriptNames) > 0 {
		result = fmt.Sprintf("%s", scriptNames[0])
		for i := 1; i < len(scriptNames); i++ {
			result += "," + scriptNames[i]
		}
	}

	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.StringType,
		String: result,
	})

	return nil
}

// cmdGetScriptVersion gets the version of a script
func cmdGetScriptVersion(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 2 {
		return vm.Error("GETSCRIPTVERSION requires exactly 2 parameters: script_name, result_var")
	}

	scriptName := GetParamString(vm, params[0])
	
	// Get the script manager from the VM
	scriptManager := vm.GetScriptManager()
	version := "6" // Default COMPILED_SCRIPT_VERSION from Pascal
	
	if scriptManager != nil {
		if sm, ok := scriptManager.(*manager.ScriptManager); ok {
			// Try to get version from script manager
			if v, err := sm.GetScriptVersion(scriptName); err == nil {
				version = v
			}
		}
	}
	
	vm.SetVariable(params[1].VarName, &types.Value{
		Type:   types.StringType,
		String: version,
	})

	return nil
}

// cmdReqVersion requires a minimum version
func cmdReqVersion(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("REQVERSION requires exactly 1 parameter: required_version")
	}

	requiredVersion := GetParamString(vm, params[0])
	
	// TODO: Implement real version checking against current TWX version
	// Should compare requiredVersion against COMPILED_SCRIPT_VERSION (6)
	// and return error if current version is too old
	// For now, always succeed until proper version checking is implemented
	_ = requiredVersion // Use the parameter to avoid unused variable warning
	return nil
}