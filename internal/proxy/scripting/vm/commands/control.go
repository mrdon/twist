package commands

import (
	"twist/internal/proxy/scripting/types"
)

// RegisterControlCommands registers all control flow commands
func RegisterControlCommands(vm CommandRegistry) {
	vm.RegisterCommand("GOTO", 1, 1, []types.ParameterType{types.ParamValue}, cmdGoto)
	vm.RegisterCommand("GOSUB", 1, 1, []types.ParameterType{types.ParamValue}, cmdGosub)
	vm.RegisterCommand("RETURN", 0, 0, []types.ParameterType{}, cmdReturn)
}

func cmdGoto(vm types.VMInterface, params []*types.CommandParam) error {
	label := GetParamString(vm, params[0])
	return vm.Goto(label)
}

func cmdGosub(vm types.VMInterface, params []*types.CommandParam) error {
	label := GetParamString(vm, params[0])
	return vm.Gosub(label)
}

func cmdReturn(vm types.VMInterface, params []*types.CommandParam) error {
	return vm.Return()
}