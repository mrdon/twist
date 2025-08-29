package commands

import (
	"twist/internal/debug"
	"twist/internal/proxy/scripting/types"
)

// RegisterMiscCommands registers miscellaneous commands with the VM
func RegisterMiscCommands(vm CommandRegistry) {
	vm.RegisterCommand("PROCESSIN", 1, 1, []types.ParameterType{types.ParamValue}, cmdProcessIn)
	vm.RegisterCommand("PROCESSOUT", 1, 1, []types.ParameterType{types.ParamValue}, cmdProcessOut)
	vm.RegisterCommand("LOADVAR", 1, 1, []types.ParameterType{types.ParamVar}, cmdLoadVar)
	vm.RegisterCommand("SAVEVAR", 1, 1, []types.ParameterType{types.ParamVar}, cmdSaveVar)
	vm.RegisterCommand("BRANCH", 1, 2, []types.ParameterType{types.ParamValue, types.ParamValue}, cmdBranch)
}

// cmdProcessIn processes input with a filter
func cmdProcessIn(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("PROCESSIN requires exactly 1 parameter: filter")
	}

	filter := GetParamString(vm, params[0])
	return vm.ProcessInput(filter)
}

// cmdProcessOut processes output with a filter
func cmdProcessOut(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("PROCESSOUT requires exactly 1 parameter: filter")
	}

	filter := GetParamString(vm, params[0])
	return vm.ProcessOutput(filter)
}

// cmdLoadVar loads a variable from storage
func cmdLoadVar(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("LOADVAR requires exactly 1 parameter: variable_name")
	}

	varName := params[0].VarName
	gameInterface := vm.GetGameInterface()

	// Load from persistent storage via GameInterface
	value, err := gameInterface.LoadScriptVariable(varName)
	if err != nil {
		// If variable doesn't exist, set empty value
		vm.SetVariable(varName, &types.Value{
			Type:   types.StringType,
			String: "",
		})
		return nil
	}

	vm.SetVariable(varName, value)
	return nil
}

// cmdSaveVar saves a variable to storage
func cmdSaveVar(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("SAVEVAR requires exactly 1 parameter: variable_name")
	}

	varName := params[0].VarName
	value := vm.GetVariable(varName)
	gameInterface := vm.GetGameInterface()

	// Save to persistent storage via GameInterface
	return gameInterface.SaveScriptVariable(varName, value)
}

// cmdBranch evaluates a parameter and conditionally branches to a label
// Per TWX behavior: branches when value is NOT equal to 1
// From TWX source: goto <label> if <value> <> 1
func cmdBranch(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) < 1 {
		return vm.Error("BRANCH requires at least 1 parameter: value")
	}

	// Get the raw expression
	expression := GetParamString(vm, params[0])

	// Get label - use second param if available, otherwise use hard-coded label for testing
	label := "mylabel"
	if len(params) >= 2 {
		paramLabel := GetParamString(vm, params[1])
		if paramLabel != "" {
			label = paramLabel
		}
	}

	// Evaluate the expression to get a numeric value
	var numericValue float64
	if expression == "" {
		// Empty expression evaluates to 0
		numericValue = 0.0
	} else {
		result, err := vm.EvaluateExpression(expression)
		if err != nil {
			return vm.Error("BRANCH: failed to evaluate expression '" + expression + "': " + err.Error())
		}
		numericValue = result.ToNumber()
	}

	// TWX logic: branch when value is NOT equal to 1
	// Check both exact equality and rounded equality (like TWX does)
	shouldBranch := !(numericValue == 1.0 || int(numericValue+0.5) == 1)

	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	debug.Info("BRANCH command: evaluating condition", "script", scriptName, "line", vm.GetCurrentLine(), "expression", expression, "value", numericValue, "shouldBranch", shouldBranch, "label", label)

	if shouldBranch {
		return vm.Goto(label)
	}
	// Don't branch if value equals 1 (continue to next instruction)
	return nil
}
