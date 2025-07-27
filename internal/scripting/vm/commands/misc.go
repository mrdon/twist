package commands

import (
	"twist/internal/scripting/types"
)

// RegisterMiscCommands registers miscellaneous commands with the VM
func RegisterMiscCommands(vm CommandRegistry) {
	vm.RegisterCommand("PROCESSIN", 1, 1, []types.ParameterType{types.ParamValue}, cmdProcessIn)
	vm.RegisterCommand("PROCESSOUT", 1, 1, []types.ParameterType{types.ParamValue}, cmdProcessOut)
	vm.RegisterCommand("LOADVAR", 1, 1, []types.ParameterType{types.ParamVar}, cmdLoadVar)
	vm.RegisterCommand("SAVEVAR", 1, 1, []types.ParameterType{types.ParamVar}, cmdSaveVar)
	vm.RegisterCommand("BRANCH", 2, 2, []types.ParameterType{types.ParamValue, types.ParamValue}, cmdBranch)
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

// cmdBranch evaluates an expression and conditionally branches to a label
// Per TWX behavior: branches when condition is FALSE (0, empty string, etc.)
// This is used by IF/WHILE macros to jump over or out of blocks when condition fails
func cmdBranch(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 2 {
		return vm.Error("BRANCH requires exactly 2 parameters: expression, label")
	}

	expression := GetParamString(vm, params[0])
	label := GetParamString(vm, params[1])
	
	// Handle empty expression - branch on empty (false condition)
	if expression == "" {
		return vm.Goto(label)
	}
	
	// Evaluate the expression using the VM's expression evaluator
	result, err := vm.EvaluateExpression(expression)
	if err != nil {
		return vm.Error("BRANCH: failed to evaluate expression '" + expression + "': " + err.Error())
	}
	
	// TWX logic: branch when condition is FALSE (0 or empty string)
	shouldBranch := false
	if result.Type == types.NumberType {
		// Branch if the number equals 0
		shouldBranch = (result.Number == 0.0)
	} else if result.Type == types.StringType {
		// Branch if the string is empty or "0"
		shouldBranch = (result.String == "" || result.String == "0")
	}
	
	if shouldBranch {
		return vm.Goto(label)
	}
	
	// Don't branch if condition is true (continue to next instruction)
	return nil
}