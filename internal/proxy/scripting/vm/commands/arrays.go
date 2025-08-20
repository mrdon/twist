package commands

import (
	"fmt"
	"strconv"
	"twist/internal/proxy/scripting/types"
)

// RegisterArrayCommands registers all array manipulation commands
func RegisterArrayCommands(vm CommandRegistry) {
	vm.RegisterCommand("SETARRAY", 2, -1, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdSetArray) // TWX setArray with separate dimension parameters
	// Note: SETARRAYELEMENT, GETARRAYELEMENT, ARRAYSIZE, CLEARARRAY are not TWX commands
	// TWX uses array indexing syntax: $array[index] instead of separate commands
}

// cmdSetArray implements the TWX-compatible setArray command
// Syntax: setArray $var dimension1 [dimension2] [dimension3]...  (e.g., setArray $sectors 10, setArray $data 3 3)
func cmdSetArray(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) < 2 {
		return vm.Error("SETARRAY requires at least 2 parameters: variable and dimensions")
	}

	varName := params[0].VarName

	// Extract dimensions from remaining parameters (TWX syntax)
	dimensions := make([]int, len(params)-1)
	for i := 1; i < len(params); i++ {
		dimValue, err := strconv.Atoi(params[i].Value.ToString())
		if err != nil {
			return vm.Error(fmt.Sprintf("Invalid dimension value: %s", params[i].Value.ToString()))
		}
		if dimValue < 0 {
			return vm.Error("Array dimension cannot be negative")
		}
		dimensions[i-1] = dimValue
	}

	// Create VarParam and set array dimensions (TWX-compatible)
	varParam := types.NewVarParam(varName, types.VarParamVariable)
	varParam.SetArray(dimensions)
	vm.SetVarParam(varName, varParam)

	return nil
}
