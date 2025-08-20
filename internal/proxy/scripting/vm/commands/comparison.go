package commands

import (
	"strconv"
	"strings"
	"twist/internal/proxy/scripting/types"
)

// RegisterComparisonCommands registers comparison commands with the VM
func RegisterComparisonCommands(vm CommandRegistry) {
	// Comparison commands
	vm.RegisterCommand("ISEQUAL", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdIsEqual)
	vm.RegisterCommand("ISNOTEQUAL", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdIsNotEqual)
	vm.RegisterCommand("ISGREATER", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdIsGreater)
	vm.RegisterCommand("ISLESS", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdIsLesser)
	vm.RegisterCommand("ISGREATEREQUAL", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdIsGreaterEqual)
	vm.RegisterCommand("ISLESSEQUAL", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdIsLesserEqual)
}

// cmdIsEqual checks if two values are equal
func cmdIsEqual(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("ISEQUAL requires exactly 3 parameters: value1, value2, result_var")
	}

	val1 := GetParamValue(vm, params[0])
	val2 := GetParamValue(vm, params[1])

	result := 0.0
	if compareValues(val1, val2) == 0 {
		result = 1.0
	}

	vm.SetVariable(params[2].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// cmdIsNotEqual checks if two values are not equal
func cmdIsNotEqual(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("ISNOTEQUAL requires exactly 3 parameters: value1, value2, result_var")
	}

	val1 := GetParamValue(vm, params[0])
	val2 := GetParamValue(vm, params[1])

	result := 0.0
	if compareValues(val1, val2) != 0 {
		result = 1.0
	}

	vm.SetVariable(params[2].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// cmdIsGreater checks if first value is greater than second
func cmdIsGreater(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("ISGREATER requires exactly 3 parameters: value1, value2, result_var")
	}

	val1 := GetParamValue(vm, params[0])
	val2 := GetParamValue(vm, params[1])

	result := 0.0
	if compareValues(val1, val2) > 0 {
		result = 1.0
	}

	vm.SetVariable(params[2].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// cmdIsLesser checks if first value is less than second
func cmdIsLesser(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("ISLESS requires exactly 3 parameters: value1, value2, result_var")
	}

	val1 := GetParamValue(vm, params[0])
	val2 := GetParamValue(vm, params[1])

	result := 0.0
	if compareValues(val1, val2) < 0 {
		result = 1.0
	}

	vm.SetVariable(params[2].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// cmdIsGreaterEqual checks if first value is greater than or equal to second
func cmdIsGreaterEqual(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("ISGREATEREQUAL requires exactly 3 parameters: value1, value2, result_var")
	}

	val1 := GetParamValue(vm, params[0])
	val2 := GetParamValue(vm, params[1])

	result := 0.0
	if compareValues(val1, val2) >= 0 {
		result = 1.0
	}

	vm.SetVariable(params[2].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// cmdIsLesserEqual checks if first value is less than or equal to second
func cmdIsLesserEqual(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 3 {
		return vm.Error("ISLESSEQUAL requires exactly 3 parameters: value1, value2, result_var")
	}

	val1 := GetParamValue(vm, params[0])
	val2 := GetParamValue(vm, params[1])

	result := 0.0
	if compareValues(val1, val2) <= 0 {
		result = 1.0
	}

	vm.SetVariable(params[2].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// compareValues compares two values and returns:
// -1 if val1 < val2
//
//	0 if val1 == val2
//	1 if val1 > val2
func compareValues(val1, val2 *types.Value) int {
	// Handle nil values
	if val1 == nil && val2 == nil {
		return 0
	}
	if val1 == nil {
		return -1
	}
	if val2 == nil {
		return 1
	}

	// If both are numbers, compare numerically
	if val1.Type == types.NumberType && val2.Type == types.NumberType {
		if val1.Number < val2.Number {
			return -1
		} else if val1.Number > val2.Number {
			return 1
		}
		return 0
	}

	// If one is number and one is string, try to convert string to number
	if val1.Type == types.NumberType && val2.Type == types.StringType {
		if num, err := strconv.ParseFloat(val2.String, 64); err == nil {
			if val1.Number < num {
				return -1
			} else if val1.Number > num {
				return 1
			}
			return 0
		}
		// If conversion fails, convert number to string for comparison
		str1 := strconv.FormatFloat(val1.Number, 'g', -1, 64)
		return strings.Compare(str1, val2.String)
	}

	if val1.Type == types.StringType && val2.Type == types.NumberType {
		if num, err := strconv.ParseFloat(val1.String, 64); err == nil {
			if num < val2.Number {
				return -1
			} else if num > val2.Number {
				return 1
			}
			return 0
		}
		// If conversion fails, convert number to string for comparison
		str2 := strconv.FormatFloat(val2.Number, 'g', -1, 64)
		return strings.Compare(val1.String, str2)
	}

	// Both are strings - do string comparison
	return strings.Compare(val1.String, val2.String)
}
