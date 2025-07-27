package commands

import (
	"fmt"
	"twist/internal/scripting/types"
)

// RegisterArrayCommands registers all array manipulation commands
func RegisterArrayCommands(vm CommandRegistry) {
	vm.RegisterCommand("ARRAY", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdArray)
	vm.RegisterCommand("SETARRAY", 2, -1, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdSetArray) // Pascal-compatible setArray
	vm.RegisterCommand("SETARRAYELEMENT", 3, 3, []types.ParameterType{types.ParamVar, types.ParamValue, types.ParamValue}, cmdSetArrayElement)
	vm.RegisterCommand("GETARRAYELEMENT", 3, 3, []types.ParameterType{types.ParamVar, types.ParamValue, types.ParamVar}, cmdGetArrayElement)
	vm.RegisterCommand("ARRAYSIZE", 2, 2, []types.ParameterType{types.ParamVar, types.ParamVar}, cmdArraySize)
	vm.RegisterCommand("CLEARARRAY", 1, 1, []types.ParameterType{types.ParamVar}, cmdClearArray)
}

func cmdArray(vm types.VMInterface, params []*types.CommandParam) error {
	// Create an array with specified size
	size := int(GetParamNumber(vm, params[1]))
	if size < 0 {
		return vm.Error("Array size cannot be negative")
	}
	
	// Create array as a special array value
	array := &types.Value{
		Type:   types.ArrayType,
		Array:  make(map[string]*types.Value),
		Number: float64(size), // Store size in Number field
	}
	
	// Initialize all elements to empty strings
	for i := 0; i < size; i++ {
		array.Array[fmt.Sprintf("%d", i)] = &types.Value{
			Type:   types.StringType,
			String: "",
		}
	}
	
	vm.SetVariable(params[0].VarName, array)
	return nil
}

func cmdSetArrayElement(vm types.VMInterface, params []*types.CommandParam) error {
	// Get the array variable
	arrayValue := vm.GetVariable(params[0].VarName)
	
	if arrayValue.Type != types.ArrayType {
		return vm.Error(fmt.Sprintf("Variable %s is not an array", params[0].VarName))
	}
	
	index := int(GetParamNumber(vm, params[1]))
	if index < 0 || index >= int(arrayValue.Number) {
		return vm.Error("Array index out of bounds")
	}
	
	// Set the element
	if arrayValue.Array == nil {
		arrayValue.Array = make(map[string]*types.Value)
	}
	
	arrayValue.Array[fmt.Sprintf("%d", index)] = &types.Value{
		Type:   types.StringType,
		String: GetParamString(vm, params[2]),
	}
	
	return nil
}

func cmdGetArrayElement(vm types.VMInterface, params []*types.CommandParam) error {
	// Get the array variable
	arrayValue := vm.GetVariable(params[0].VarName)
	
	if arrayValue.Type != types.ArrayType {
		return vm.Error(fmt.Sprintf("Variable %s is not an array", params[0].VarName))
	}
	
	index := int(GetParamNumber(vm, params[1]))
	if index < 0 || index >= int(arrayValue.Number) {
		return vm.Error("Array index out of bounds")
	}
	
	// Get the element
	var result *types.Value
	if arrayValue.Array != nil && arrayValue.Array[fmt.Sprintf("%d", index)] != nil {
		result = arrayValue.Array[fmt.Sprintf("%d", index)]
	} else {
		result = &types.Value{
			Type:   types.StringType,
			String: "",
		}
	}
	
	vm.SetVariable(params[2].VarName, result)
	return nil
}

func cmdArraySize(vm types.VMInterface, params []*types.CommandParam) error {
	// Get the array variable
	arrayValue := vm.GetVariable(params[0].VarName)
	
	if arrayValue.Type != types.ArrayType {
		return vm.Error(fmt.Sprintf("Variable %s is not an array", params[0].VarName))
	}
	
	// Return the size
	result := &types.Value{
		Type:   types.NumberType,
		Number: arrayValue.Number,
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdClearArray(vm types.VMInterface, params []*types.CommandParam) error {
	// Clear all elements in the array
	arrayValue := vm.GetVariable(params[0].VarName)
	
	if arrayValue.Type != types.ArrayType {
		return vm.Error(fmt.Sprintf("Variable %s is not an array", params[0].VarName))
	}
	
	// Clear all elements
	size := int(arrayValue.Number)
	for i := 0; i < size; i++ {
		arrayValue.Array[fmt.Sprintf("%d", i)] = &types.Value{
			Type:   types.StringType,
			String: "",
		}
	}
	
	return nil
}

// cmdSetArray implements the Pascal-compatible setArray command
// Syntax: setArray var <dimensions...>
// Example: setArray $sectors 3
func cmdSetArray(vm types.VMInterface, params []*types.CommandParam) error {
	// Parse dimensions from parameters (Pascal supports multi-dimensional)
	dimensions := make([]int, len(params)-1)
	for i := 1; i < len(params); i++ {
		dimensions[i-1] = int(GetParamNumber(vm, params[i]))
	}
	
	// Get or create the VarParam for this variable
	varParam := vm.GetVarParam(params[0].VarName)
	if varParam == nil {
		// Create new VarParam if it doesn't exist
		varParam = types.NewVarParam(params[0].VarName, types.VarParamVariable)
		vm.SetVarParam(params[0].VarName, varParam)
	}
	
	// Initialize the array with Pascal behavior
	varParam.SetArray(dimensions)
	
	return nil
}